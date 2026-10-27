package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/synclock"
)

// SyncExecutor 负责单账户同步的执行用例：锁管理、适配器交互、
// 分批/UID 增量/传统三条同步路径，以及 WebAPI 协议同步。
//
// 为什么从 syncService 里拆出来：原类型跨 4 个文件、24 个方法、19 个字段。
// 邮件落库委托给 SyncProcessor（叶子），失败分类落库委托给 SyncErrorPolicy（叶子），
// 活跃同步的共享状态（锁信息/进度追踪器）读写 SyncProgressRegistry，
// 本类型不再持有任何 map + 锁的可变状态。
type SyncExecutor struct {
	accountRepo          repository.AccountRepository
	emailRepo            repository.EmailRepository // WebAPI 同步路径直接落库用
	syncLogRepo          repository.SyncLogRepository
	adapterFactory       *adapter.Factory
	webAPIAdapterFactory *webapi.WebAPIAdapterFactory // WebAPI 适配器工厂
	cryptoService        *crypto.Service
	credentialResolver   *CredentialResolver
	notifier             SyncNotifier
	dedupeKeyGen         *DedupeKeyGenerator // 去重标识生成器（WebAPI 同步路径用）
	processor            *SyncProcessor      // 邮件落库（依赖图上的叶子）
	errorPolicy          *SyncErrorPolicy    // 同步失败分类与落库（依赖图上的叶子）
	registry             *SyncProgressRegistry
	syncLock             *synclock.SyncLock // 分布式同步锁（基于 Redis，支持自动过期和续期）
	logger               *logger.Logger
}

// NewSyncExecutor 创建同步执行器。
func NewSyncExecutor(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	adapterFactory *adapter.Factory,
	webAPIAdapterFactory *webapi.WebAPIAdapterFactory,
	cryptoService *crypto.Service,
	credentialResolver *CredentialResolver,
	notifier SyncNotifier,
	dedupeKeyGen *DedupeKeyGenerator,
	processor *SyncProcessor,
	errorPolicy *SyncErrorPolicy,
	registry *SyncProgressRegistry,
	syncLock *synclock.SyncLock,
) *SyncExecutor {
	return &SyncExecutor{
		accountRepo:          accountRepo,
		emailRepo:            emailRepo,
		syncLogRepo:          syncLogRepo,
		adapterFactory:       adapterFactory,
		webAPIAdapterFactory: webAPIAdapterFactory,
		cryptoService:        cryptoService,
		credentialResolver:   credentialResolver,
		notifier:             notifier,
		dedupeKeyGen:         dedupeKeyGen,
		processor:            processor,
		errorPolicy:          errorPolicy,
		registry:             registry,
		syncLock:             syncLock,
		logger:               logger.NewWithModule("Sync"),
	}
}

// SyncAccount 同步指定账户的邮件
// 使用 Redis 分布式锁防止重复同步，支持自动超时和锁续期
func (s *SyncExecutor) SyncAccount(ctx context.Context, accountUID string) error {
	// 获取账户信息（预加载 Provider 和 Adapter 关联，以便获取服务器配置）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountUID)
	}

	// 检查账户状态
	if account.Status != "active" {
		return fmt.Errorf("account is not active (status: %s): %s", account.Status, accountUID)
	}

	// 检查是否启用同步
	if !account.SyncEnabled {
		return fmt.Errorf("sync is disabled for account: %s", accountUID)
	}

	// Webhook 模式 / webhook 子账户 / 父子账户中的子邮箱：不走轮询同步
	if account.ShouldSkipPollingSync() {
		s.logger.Debug("跳过轮询同步: account=%s, mode=%s", accountUID, account.GetSyncMode())
		return nil
	}

	// 使用分布式锁（如果可用）
	var syncCtx context.Context
	var lockInfo *synclock.LockInfo

	if s.syncLock != nil {
		// 先检查内存中是否有过期的锁（Redis 锁已过期但内存锁未清理的情况）
		s.registry.EvictStaleLock(accountUID, func() bool {
			isLocked, _ := s.syncLock.IsLocked(ctx, accountUID)
			return isLocked
		})

		// 使用 Redis 分布式锁（带自动超时和续期）
		var err error
		lockInfo, syncCtx, err = s.syncLock.AcquireLock(ctx, accountUID)
		if err != nil {
			return err // 锁获取失败，可能是同步已在进行中
		}

		// 注册锁信息到注册表（用于取消同步）
		s.registry.Register(accountUID, lockInfo)

		// 确保同步结束时释放锁
		defer func() {
			s.registry.Remove(accountUID)

			// 释放 Redis 锁
			if releaseErr := s.syncLock.ReleaseLock(context.Background(), lockInfo); releaseErr != nil {
				s.logger.Warn("释放同步锁失败: account=%s, err=%v", accountUID, releaseErr)
			}
		}()
	} else {
		// 降级到内存锁（不推荐用于生产环境）
		// 创建带超时的 context（默认 10 分钟）
		var cancelFunc context.CancelFunc
		syncCtx, cancelFunc = context.WithTimeout(ctx, synclock.DefaultSyncTimeout)
		lockInfo = &synclock.LockInfo{
			AccountUID: accountUID,
			CancelFunc: cancelFunc,
			AcquiredAt: time.Now(),
		}
		if !s.registry.TryRegister(accountUID, lockInfo) {
			cancelFunc() // 释放刚创建的 context，避免泄漏
			return fmt.Errorf("sync already in progress for account: %s", accountUID)
		}

		// 确保同步结束时清理
		defer func() {
			s.registry.Remove(accountUID)
			cancelFunc()
		}()
	}

	// 判断是否为首次同步
	isFirstSync := account.LastSyncAt == nil

	// 创建同步日志
	syncLog := &model.SyncLog{
		AccountUID:  accountUID,
		SyncType:    "manual",
		Status:      "running",
		StartedAt:   time.Now(),
		IsFirstSync: isFirstSync,
	}

	if err := s.syncLogRepo.Create(ctx, syncLog); err != nil {
		s.logger.Error("创建同步日志失败: %v", err)
	}

	// 执行同步
	err = s.doSync(syncCtx, account, syncLog)

	// 更新同步日志
	if err != nil {
		if syncCtx.Err() == context.Canceled {
			syncLog.Status = "cancelled"
			syncLog.ErrorMessage = "sync cancelled by user"
		} else {
			syncLog.Status = "failed"
			syncLog.ErrorMessage = err.Error()
		}
	} else {
		syncLog.Status = "success"
	}

	completedAt := time.Now()
	syncLog.CompletedAt = &completedAt
	syncLog.DurationMs = time.Since(syncLog.StartedAt).Milliseconds()

	// 优化：只记录有价值的同步日志
	// - 失败/取消的记录：始终保存（用于问题排查）
	// - 成功且有变化的记录：保存（有实际同步内容）
	// - 成功但无变化的记录：删除（减少存储占用）
	shouldKeepLog := syncLog.Status != "success" ||
		syncLog.EmailsNew > 0 ||
		syncLog.EmailsUpdated > 0 ||
		syncLog.IsFirstSync

	if shouldKeepLog {
		// 保存同步日志到数据库
		if updateErr := s.syncLogRepo.Update(ctx, syncLog); updateErr != nil {
			s.logger.Error("更新同步日志失败: %v", updateErr)
		}
	} else {
		// 删除无变化的成功日志，减少存储占用
		if deleteErr := s.syncLogRepo.Delete(ctx, syncLog.ID); deleteErr != nil {
			s.logger.Error("删除无变化同步日志失败: %v", deleteErr)
		}
	}

	// 更新账户同步状态（只更新同步相关字段，避免覆盖其他字段如 consecutive_auth_failures）
	if updateErr := s.accountRepo.UpdateSyncStatus(ctx, accountUID, syncLog.Status, syncLog.ErrorMessage); updateErr != nil {
		s.logger.Error("更新账户同步状态失败: %v", updateErr)
	}

	return err
}

// doSync 执行实际的同步逻辑
// Requirements: 1.2, 3.1, 3.2, 7.1 - 集成 BatchProcessor 和 ProgressTracker
func (s *SyncExecutor) doSync(ctx context.Context, account *model.EmailAccount, syncLog *model.SyncLog) error {
	s.logger.Info("开始同步: account=%s, email=%s, auth=%s", account.UID, account.Email, account.GetAuthType())

	// 获取同步配置
	syncConfig := account.GetSyncConfig()
	isFirstSync := account.LastSyncAt == nil

	// 检查是否是 WebAPI 协议
	protocol := account.GetProtocol()
	if protocol == "webapi" {
		return s.doSyncWebAPI(ctx, account, syncLog, syncConfig, isFirstSync)
	}

	// 解析认证凭证
	credentials, err := s.credentialResolver.Resolve(account)
	if err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	// 解析代理配置
	proxy, err := s.parseProxyConfig(account)
	if err != nil {
		return fmt.Errorf("failed to parse proxy config: %w", err)
	}

	// 创建适配器配置
	config := &adapter.Config{
		Provider:    account.GetProviderName(),
		Protocol:    protocol,
		Credentials: credentials,
		Proxy:       proxy,
		Timeout:     0, // 使用默认超时
	}

	// 使用自动选择方法创建适配器（会智能判断是否使用短效适配器）
	provider, err := s.adapterFactory.CreateProviderAuto(config)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// 连接到邮箱服务器
	if err := provider.Connect(ctx); err != nil {
		// 处理连接错误（包括认证错误的特殊处理）
		return s.errorPolicy.handleSyncError(ctx, account, fmt.Errorf("failed to connect: %w", err))
	}
	defer provider.Disconnect()

	// 创建进度追踪器 (Requirements: 2.1, 2.2)
	tracker := NewProgressTrackerWithNotifier(syncConfig.ProgressInterval, s.notifier)

	// 注册进度追踪器
	s.registry.RegisterTracker(account.UID, tracker)

	// 检查适配器是否支持 UID 增量同步（IMAP 适配器）
	imapAdapter, supportsUID := provider.(*adapter.IMAPAdapter)
	if supportsUID && protocol == "imap" {
		// 使用 UID 增量同步模式（优先）
		return s.doSyncWithUID(ctx, account, syncLog, imapAdapter, tracker, syncConfig, isFirstSync)
	}

	// 计算同步起始时间 (Requirements: 1.2, 5.3, 7.5)
	since := s.calculateSyncSince(account, syncConfig, isFirstSync)

	// 检查适配器是否支持分批拉取
	batchFetcher, supportsBatch := provider.(adapter.BatchFetcher)

	if supportsBatch {
		// 使用分批处理模式 (Requirements: 3.1, 3.2)
		return s.doSyncWithBatch(ctx, account, syncLog, batchFetcher, tracker, syncConfig, since, isFirstSync)
	}

	// 降级到传统模式
	return s.doSyncLegacy(ctx, account, syncLog, provider, tracker, syncConfig, since, isFirstSync)
}

// calculateSyncSince 计算同步起始时间
// Requirements: 1.2 - 首次同步时间范围计算
// Requirements: 5.3, 7.5 - 从上次同步位置恢复
//
// 时区策略：
// - 内部统一使用 UTC，与数据库保持一致
// - 支持跨国/分布式部署，避免时区混乱
// - 邮件服务器返回的时间会被转换为 UTC 存储
func (s *SyncExecutor) calculateSyncSince(account *model.EmailAccount, config *model.SyncConfig, isFirstSync bool) time.Time {
	if account.LastSyncAt != nil {
		// 增量同步：从上次同步时间开始，减去 1 小时缓冲
		since := account.LastSyncAt.Add(-1 * time.Hour)
		s.logger.Debug("增量同步: account=%s, since=%s", account.UID, since.Format(time.RFC3339))
		return since
	}

	// 首次同步：根据配置计算起始时间 (Requirements: 1.2)
	if config.IsFullSync() {
		s.logger.Debug("全量同步: account=%s", account.UID)
		return time.Time{}
	}

	// 从 N 天前开始（使用 UTC）
	since := time.Now().UTC().AddDate(0, 0, -config.FirstSyncDays)
	s.logger.Debug("首次同步: account=%s, days=%d", account.UID, config.FirstSyncDays)
	return since
}

// parseProxyConfig 解析代理配置
func (s *SyncExecutor) parseProxyConfig(account *model.EmailAccount) (*adapter.ProxyConfig, error) {
	if !account.ProxyEnabled {
		return nil, nil
	}

	return &adapter.ProxyConfig{
		Enabled:  account.ProxyEnabled,
		Type:     account.ProxyType,
		Host:     account.ProxyHost,
		Port:     account.ProxyPort,
		Username: account.ProxyUsername,
		// Password: decrypt(account.EncryptedProxyPassword),
	}, nil
}
