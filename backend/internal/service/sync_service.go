package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"fusionmail/internal/sse"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/oauth2config"
	"fusionmail/pkg/synclock"

	"github.com/redis/go-redis/v9"
)

// 模块日志记录器
var syncLog = logger.NewWithModule("Sync")

// SyncService 邮件同步服务接口
type SyncService interface {
	// SyncAccount 同步指定账户的邮件
	SyncAccount(ctx context.Context, accountUID string) error

	// SyncAllAccounts 同步所有启用的账户
	SyncAllAccounts(ctx context.Context) error

	// StartScheduler 启动定时同步调度器
	StartScheduler(ctx context.Context) error

	// StopScheduler 停止定时同步调度器
	StopScheduler() error

	// CancelSync 取消指定账户的同步
	// Requirements: 5.1 - 支持同步取消
	CancelSync(accountUID string) error

	// GetSyncProgress 获取指定账户的同步进度
	GetSyncProgress(accountUID string) *model.SyncProgress
}

// syncService 邮件同步服务实现
type syncService struct {
	accountRepo          repository.AccountRepository
	emailRepo            repository.EmailRepository
	syncLogRepo          repository.SyncLogRepository
	deletedKeyRepo       *repository.DeletedEmailKeyRepository // 已删除邮件去重标识仓库
	adapterFactory       *adapter.Factory
	webAPIAdapterFactory *webapi.WebAPIAdapterFactory // WebAPI 适配器工厂
	cryptoService        *crypto.Service
	schedulerStop        chan struct{}
	oauth2ConfigProvider *oauth2config.Provider // 新增：OAuth2配置提供者
	spamDetector         SpamDetectorInterface  // 垃圾邮件检测器
	dedupeKeyGen         *DedupeKeyGenerator    // 去重标识生成器
	logger               *logger.Logger         // 日志记录器

	// 分布式同步锁（基于 Redis，支持自动过期和续期）
	syncLock *synclock.SyncLock

	// 同步取消支持 (Requirements: 5.1, 5.2)
	// 注意：activeSyncs 现在仅用于本地进度追踪，锁由 Redis 管理
	activeSyncs    map[string]*synclock.LockInfo // 活跃同步的锁信息
	activeTrackers map[string]ProgressTracker    // 活跃同步的进度追踪器
	syncMu         sync.RWMutex                  // 保护 activeSyncs 和 activeTrackers
}

// SpamDetectorInterface 垃圾邮件检测器接口
type SpamDetectorInterface interface {
	DetectSpamSimple(ctx context.Context, email *model.Email) (*spam.SpamSimpleResult, error)
}

// NewSyncService 创建邮件同步服务实例
func NewSyncService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	deletedKeyRepo *repository.DeletedEmailKeyRepository, // 已删除邮件去重标识仓库
	adapterFactory *adapter.Factory,
	oauth2ClientRepo repository.OAuth2ClientRepository,
	providerRepo repository.ProviderRepository,
	appLogger *logger.Logger,
	cryptoService *crypto.Service, // 添加加密服务参数（指针类型）
	spamDetector SpamDetectorInterface, // 垃圾邮件检测器（可选）
	redisClient *redis.Client, // Redis 客户端，用于分布式锁
) SyncService {

	// 创建模块日志记录器
	syncLogger := logger.NewWithModule("Sync")

	// 创建OAuth2配置提供者
	oauth2Provider := oauth2config.NewProvider(oauth2ClientRepo, providerRepo, cryptoService, appLogger)

	// 创建分布式同步锁（如果 Redis 可用）
	var sl *synclock.SyncLock
	if redisClient != nil {
		sl = synclock.NewSyncLock(redisClient)
		syncLogger.Info("分布式同步锁已启用 (Redis)")
	} else {
		syncLogger.Warn("Redis 不可用，使用内存锁（不推荐用于生产环境）")
	}

	return &syncService{
		accountRepo:          accountRepo,
		emailRepo:            emailRepo,
		syncLogRepo:          syncLogRepo,
		deletedKeyRepo:       deletedKeyRepo,
		adapterFactory:       adapterFactory,
		webAPIAdapterFactory: webapi.NewWebAPIAdapterFactory(), // 初始化 WebAPI 适配器工厂
		cryptoService:        cryptoService,
		oauth2ConfigProvider: oauth2Provider,
		spamDetector:         spamDetector,
		dedupeKeyGen:         NewDedupeKeyGenerator(),
		logger:               syncLogger,
		syncLock:             sl,
		activeSyncs:          make(map[string]*synclock.LockInfo),
		activeTrackers:       make(map[string]ProgressTracker),
	}
}

// SyncAccount 同步指定账户的邮件
// 使用 Redis 分布式锁防止重复同步，支持自动超时和锁续期
func (s *syncService) SyncAccount(ctx context.Context, accountUID string) error {
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

	// 检查是否为 Webhook 模式（Webhook 模式不需要轮询同步）
	if account.IsWebhookMode() {
		return fmt.Errorf("account uses webhook mode, polling sync is not needed: %s", accountUID)
	}

	// 使用分布式锁（如果可用）
	var syncCtx context.Context
	var lockInfo *synclock.LockInfo

	if s.syncLock != nil {
		// 先检查内存中是否有过期的锁（Redis 锁已过期但内存锁未清理的情况）
		s.syncMu.Lock()
		if existingLock, exists := s.activeSyncs[accountUID]; exists {
			// 检查 Redis 锁是否仍然存在
			isLocked, _ := s.syncLock.IsLocked(ctx, accountUID)
			if !isLocked {
				// Redis 锁已过期，清理内存中的过期锁
				s.logger.Warn("发现过期内存锁，正在清理: account=%s", accountUID)
				if existingLock.CancelFunc != nil {
					existingLock.CancelFunc()
				}
				delete(s.activeSyncs, accountUID)
				delete(s.activeTrackers, accountUID)
			}
		}
		s.syncMu.Unlock()

		// 使用 Redis 分布式锁（带自动超时和续期）
		var err error
		lockInfo, syncCtx, err = s.syncLock.AcquireLock(ctx, accountUID)
		if err != nil {
			return err // 锁获取失败，可能是同步已在进行中
		}

		// 注册锁信息到本地 map（用于取消同步）
		s.syncMu.Lock()
		s.activeSyncs[accountUID] = lockInfo
		s.syncMu.Unlock()

		// 确保同步结束时释放锁
		defer func() {
			s.syncMu.Lock()
			delete(s.activeSyncs, accountUID)
			delete(s.activeTrackers, accountUID)
			s.syncMu.Unlock()

			// 释放 Redis 锁
			if releaseErr := s.syncLock.ReleaseLock(context.Background(), lockInfo); releaseErr != nil {
				s.logger.Warn("释放同步锁失败: account=%s, err=%v", accountUID, releaseErr)
			}
		}()
	} else {
		// 降级到内存锁（不推荐用于生产环境）
		s.syncMu.Lock()
		if _, exists := s.activeSyncs[accountUID]; exists {
			s.syncMu.Unlock()
			return fmt.Errorf("sync already in progress for account: %s", accountUID)
		}

		// 创建带超时的 context（默认 10 分钟）
		var cancelFunc context.CancelFunc
		syncCtx, cancelFunc = context.WithTimeout(ctx, synclock.DefaultSyncTimeout)
		lockInfo = &synclock.LockInfo{
			AccountUID: accountUID,
			CancelFunc: cancelFunc,
			AcquiredAt: time.Now(),
		}
		s.activeSyncs[accountUID] = lockInfo
		s.syncMu.Unlock()

		// 确保同步结束时清理
		defer func() {
			s.syncMu.Lock()
			delete(s.activeSyncs, accountUID)
			delete(s.activeTrackers, accountUID)
			s.syncMu.Unlock()
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

// CancelSync 取消指定账户的同步
// Requirements: 5.1 - 支持同步取消
func (s *syncService) CancelSync(accountUID string) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	lockInfo, exists := s.activeSyncs[accountUID]
	if !exists {
		return fmt.Errorf("no active sync for account: %s", accountUID)
	}

	// 调用取消函数
	if lockInfo.CancelFunc != nil {
		lockInfo.CancelFunc()
	}
	s.logger.Info("同步已取消: account=%s", accountUID)

	return nil
}

// GetSyncProgress 获取指定账户的同步进度
func (s *syncService) GetSyncProgress(accountUID string) *model.SyncProgress {
	s.syncMu.RLock()
	defer s.syncMu.RUnlock()

	tracker, exists := s.activeTrackers[accountUID]
	if !exists {
		return nil
	}

	return tracker.GetProgress()
}

// doSync 执行实际的同步逻辑
// Requirements: 1.2, 3.1, 3.2, 7.1 - 集成 BatchProcessor 和 ProgressTracker
func (s *syncService) doSync(ctx context.Context, account *model.EmailAccount, syncLog *model.SyncLog) error {
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
	credentials, err := s.parseCredentials(account)
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
		return s.handleSyncError(ctx, account, fmt.Errorf("failed to connect: %w", err))
	}
	defer provider.Disconnect()

	// 创建进度追踪器 (Requirements: 2.1, 2.2)
	tracker := NewProgressTracker(syncConfig.ProgressInterval)

	// 注册进度追踪器
	s.syncMu.Lock()
	s.activeTrackers[account.UID] = tracker
	s.syncMu.Unlock()

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
func (s *syncService) calculateSyncSince(account *model.EmailAccount, config *model.SyncConfig, isFirstSync bool) time.Time {
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

// doSyncWithUID 使用 UID 增量同步模式（IMAP 专用）
// Requirements: 1.1, 1.2, 1.3 - 基于 UID 的增量同步
func (s *syncService) doSyncWithUID(
	ctx context.Context,
	account *model.EmailAccount,
	syncLog *model.SyncLog,
	imapAdapter *adapter.IMAPAdapter,
	tracker ProgressTracker,
	config *model.SyncConfig,
	isFirstSync bool,
) error {
	// 获取当前邮箱的 UID 同步状态
	uidState, err := imapAdapter.GetUIDSyncState(ctx)
	if err != nil {
		s.logger.Warn("获取 UID 同步状态失败，降级到传统模式: account=%s, err=%v", account.UID, err)
		since := s.calculateSyncSince(account, config, isFirstSync)
		return s.doSyncWithBatch(ctx, account, syncLog, imapAdapter, tracker, config, since, isFirstSync)
	}

	s.logger.Info("UID 同步状态: account=%s, currentValidity=%d, storedValidity=%d, storedLastUID=%d, maxUID=%d",
		account.UID, uidState.UIDValidity, account.UIDValidity, account.LastUID, uidState.MaxUID)

	// 检查是否需要全量同步 (Requirements: 1.3)
	needFullSync := imapAdapter.ShouldFullSync(uint32(account.UIDValidity), uidState.UIDValidity)
	if needFullSync {
		if account.UIDValidity > 0 {
			// UIDVALIDITY 变化，记录警告 (Requirements: 5.3)
			s.logger.Warn("UIDVALIDITY 变化，执行全量同步: account=%s, old=%d, new=%d",
				account.UID, account.UIDValidity, uidState.UIDValidity)
		}
		// 首次同步或 UIDVALIDITY 变化，使用传统模式
		since := s.calculateSyncSince(account, config, isFirstSync)
		err := s.doSyncWithBatch(ctx, account, syncLog, imapAdapter, tracker, config, since, isFirstSync)
		if err != nil {
			return err
		}
		// 更新 UID 同步状态
		return s.updateUIDSyncState(ctx, account.UID, uidState.UIDValidity, uidState.MaxUID)
	}

	// 增量同步：只拉取 UID > LastUID 的邮件 (Requirements: 1.1)
	sinceUID := uint32(account.LastUID)
	s.logger.Info("开始 UID 增量同步: account=%s, sinceUID=%d", account.UID, sinceUID)

	// 开始进度追踪
	tracker.Start(account.UID, 0, isFirstSync)
	tracker.SetPhase(model.SyncPhaseFetching)

	// 拉取新邮件
	emails, maxUID, err := imapAdapter.FetchEmailsSinceUID(ctx, sinceUID, config.MaxEmailsPerSync)
	if err != nil {
		tracker.Fail(err)
		return s.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails since UID: %w", err))
	}

	syncLog.EmailsFetched = int64(len(emails))
	syncLog.TotalEstimated = len(emails)

	// 如果没有新邮件 (Requirements: 1.4)
	if len(emails) == 0 {
		s.logger.Info("无新邮件: account=%s, sinceUID=%d", account.UID, sinceUID)
		tracker.Complete()
		// 更新同步时间但不更新 LastUID
		return nil
	}

	// 处理邮件
	tracker.SetPhase(model.SyncPhaseProcessing)
	var totalNew, totalUpdated, totalSkipped, totalFailed int

	for i, email := range emails {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			tracker.Cancel()
			return ctx.Err()
		default:
		}

		// 使用新的处理方法（支持 dedupe_key）
		result, err := s.processEmailWithDedupe(ctx, account.UID, email, syncLog)
		if err != nil {
			totalFailed++
			continue
		}

		switch result {
		case "new":
			totalNew++
		case "updated":
			totalUpdated++
		case "skipped":
			totalSkipped++
		}

		// 更新进度
		tracker.Update(i+1, totalNew, totalUpdated, totalFailed)
	}

	// 完成同步
	tracker.SetPhase(model.SyncPhaseFinalizing)
	syncLog.EmailsNew = int64(totalNew)
	syncLog.EmailsUpdated = int64(totalUpdated)

	// 更新 LastUID (Requirements: 1.2)
	if maxUID > sinceUID {
		if err := s.updateUIDSyncState(ctx, account.UID, uidState.UIDValidity, maxUID); err != nil {
			s.logger.Error("更新 UID 同步状态失败: account=%s, err=%v", account.UID, err)
		}
	}

	// 如果本次同步有新增或更新的邮件，通过 SSE 通知前端刷新
	if totalNew > 0 || totalUpdated > 0 {
		sse.Broadcast("email_counts_maybe_changed", "{}")
	}

	// 同步成功，重置失败计数（仅对 quick 账号）
	if account.GetAuthType() == "quick" && account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			s.logger.Error("重置失败计数失败: account=%s, err=%v", account.UID, resetErr)
		}
	}

	s.logger.Info("UID 增量同步完成: account=%s, new=%d, updated=%d, skipped=%d, maxUID=%d",
		account.UID, totalNew, totalUpdated, totalSkipped, maxUID)

	tracker.Complete()
	return nil
}

// updateUIDSyncState 更新 UID 同步状态
// Requirements: 6.1 - 持久化同步状态
func (s *syncService) updateUIDSyncState(ctx context.Context, accountUID string, uidValidity uint32, lastUID uint32) error {
	return s.accountRepo.UpdateUIDSyncState(ctx, accountUID, int64(uidValidity), int64(lastUID))
}

// processEmailWithDedupe 处理单封邮件（支持 dedupe_key）
// Requirements: 2.2, 3.1, 3.2, 3.3, 3.4
// 返回值: "new", "updated", "skipped"
func (s *syncService) processEmailWithDedupe(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) (string, error) {
	// 生成 dedupe_key (Requirements: 3.1, 3.2)
	dedupeKey := s.dedupeKeyGen.GenerateFromRaw(
		adapterEmail.MessageID,
		adapterEmail.FromAddress,
		adapterEmail.Subject,
		adapterEmail.SentAt,
	)

	// 检查是否在已删除列表中 (Requirements: 2.2)
	if s.deletedKeyRepo != nil {
		isDeleted, err := s.deletedKeyRepo.IsDeleted(ctx, accountUID, dedupeKey)
		if err != nil {
			s.logger.Warn("检查已删除标识失败: account=%s, key=%s, err=%v", accountUID, dedupeKey, err)
		} else if isDeleted {
			// 邮件已被删除，跳过
			return "skipped", nil
		}
	}

	// 先通过 dedupe_key 查找（优先）
	existingEmail, err := s.emailRepo.FindByDedupeKey(ctx, accountUID, dedupeKey)
	if err != nil {
		return "", err
	}

	// 如果 dedupe_key 没找到，再通过 provider_id 查找（兼容旧数据）
	if existingEmail == nil {
		existingEmail, err = s.emailRepo.FindByProviderID(ctx, adapterEmail.ProviderID, accountUID)
		if err != nil {
			return "", err
		}
	}

	if existingEmail != nil {
		// 如果邮件已被软删除，跳过更新 (Requirements: 2.4)
		if existingEmail.DeletedAt.Valid {
			return "skipped", nil
		}

		// 邮件已存在且未删除，更新
		s.updateEmailFromAdapter(existingEmail, adapterEmail, accountUID)
		// 更新 dedupe_key（如果之前没有）
		if existingEmail.DedupeKey == "" {
			existingEmail.DedupeKey = dedupeKey
		}
		if err := s.emailRepo.Update(ctx, existingEmail); err != nil {
			return "", err
		}
		// 应用规则到已存在邮件（更新后）
		if err := s.applyRulesForEmail(ctx, existingEmail); err != nil {
			s.logger.Warn("应用规则失败(更新): email=%d, err=%v", existingEmail.ID, err)
		}
		syncLog.EmailsUpdated++
		return "updated", nil
	}

	// 新邮件，创建 (Requirements: 3.3)
	newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)
	newEmail.DedupeKey = dedupeKey

	// 先保存邮件到数据库，获取正确的 ID
	if err := s.emailRepo.Create(ctx, newEmail); err != nil {
		return "", err
	}

	// 垃圾邮件检测（在邮件保存后执行，确保 email.ID 正确）
	if s.spamDetector != nil {
		spamResult, spamErr := s.spamDetector.DetectSpamSimple(ctx, newEmail)
		if spamErr != nil {
			s.logger.Warn("垃圾邮件检测失败: emailId=%d, msgId=%s, err=%v", newEmail.ID, newEmail.MessageID, spamErr)
		} else if spamResult != nil {
			// 更新邮件的垃圾检测结果
			newEmail.IsSpam = spamResult.IsSpam
			newEmail.SpamScore = float64(spamResult.Score)
			newEmail.SpamConfidence = spamResult.Confidence
			newEmail.SpamReason = spamResult.Reason
			newEmail.SpamDetectedBy = spamResult.DetectedBy
			if spamResult.IsSpam {
				now := time.Now()
				newEmail.SpamDetectedAt = &now
				s.logger.Info("检测到垃圾邮件: emailId=%d, subject=%s, score=%d", newEmail.ID, newEmail.Subject, spamResult.Score)
			}
			// 更新数据库中的垃圾检测结果
			if err := s.emailRepo.Update(ctx, newEmail); err != nil {
				s.logger.Warn("更新垃圾检测结果失败: emailId=%d, err=%v", newEmail.ID, err)
			}
		}
	}

	// 应用规则到新邮件
	if err := s.applyRulesForEmail(ctx, newEmail); err != nil {
		s.logger.Warn("应用规则失败(新建): email=%d, err=%v", newEmail.ID, err)
	}
	syncLog.EmailsNew++
	return "new", nil
}

// doSyncWithBatch 使用分批处理模式同步
// Requirements: 3.1, 3.2 - 分批处理和进度持久化
func (s *syncService) doSyncWithBatch(
	ctx context.Context,
	account *model.EmailAccount,
	syncLog *model.SyncLog,
	batchFetcher adapter.BatchFetcher,
	tracker ProgressTracker,
	config *model.SyncConfig,
	since time.Time,
	isFirstSync bool,
) error {
	// 获取预估邮件数量
	estimatedCount, err := batchFetcher.GetEstimatedCount(ctx, since)
	if err != nil {
		s.logger.Warn("获取预估邮件数失败: account=%s, err=%v", account.UID, err)
		estimatedCount = 0
	}

	// 大邮箱警告 (Requirements: 3.3)
	if estimatedCount > 5000 {
		s.logger.Warn("检测到大邮箱: account=%s, estimated=%d", account.UID, estimatedCount)
	}

	// 计算总批次数
	totalBatches := CalculateTotalBatches(estimatedCount, config.BatchSize)

	// 开始进度追踪 (Requirements: 2.1)
	tracker.Start(account.UID, estimatedCount, isFirstSync)
	tracker.SetPhase(model.SyncPhaseFetching)

	// 更新同步日志
	syncLog.TotalEstimated = estimatedCount
	syncLog.TotalBatches = totalBatches
	syncLog.IsFirstSync = isFirstSync

	// 从游标恢复 (Requirements: 5.3, 7.5)
	cursor := account.SyncCursor

	var totalProcessed, totalNew, totalUpdated, totalFailed int
	currentBatch := 0
	hasMore := true

	// 分批拉取和处理
	for hasMore {
		// 检查 context 是否已取消 (Requirements: 5.1, 7.3)
		select {
		case <-ctx.Done():
			// 保存当前进度
			s.persistSyncProgress(context.Background(), account.UID, cursor, tracker.GetProgress())
			tracker.Cancel()
			return ctx.Err()
		default:
		}

		// 检查是否超过最大邮件数限制
		if config.MaxEmailsPerSync > 0 && totalProcessed >= config.MaxEmailsPerSync {
			s.logger.Info("达到单次同步上限: account=%s, limit=%d", account.UID, config.MaxEmailsPerSync)
			break
		}

		currentBatch++
		tracker.SetPhase(model.SyncPhaseFetching)

		// 拉取一批邮件 (Requirements: 3.1)
		emails, nextCursor, more, fetchErr := batchFetcher.FetchEmailsBatch(ctx, since, config.BatchSize, cursor)
		if fetchErr != nil {
			// 使用重试机制 (Requirements: 3.4)
			retryErr := RetryWithBackoff(ctx, func() error {
				var retryEmails []*adapter.Email
				var retryMore bool
				retryEmails, nextCursor, retryMore, fetchErr = batchFetcher.FetchEmailsBatch(ctx, since, config.BatchSize, cursor)
				if fetchErr == nil {
					emails = retryEmails
					more = retryMore
				}
				return fetchErr
			}, config.RetryCount, config.RetryBackoffMs)

			if retryErr != nil {
				// 保存当前进度后失败
				s.persistSyncProgress(context.Background(), account.UID, cursor, tracker.GetProgress())
				tracker.Fail(retryErr)
				return s.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails batch: %w", retryErr))
			}
		}

		// 处理这批邮件
		tracker.SetPhase(model.SyncPhaseProcessing)
		batchNew, batchUpdated, batchFailed := s.processBatchEmails(ctx, account.UID, emails, syncLog)

		totalProcessed += len(emails)
		totalNew += batchNew
		totalUpdated += batchUpdated
		totalFailed += batchFailed

		// 更新进度 (Requirements: 2.2)
		tracker.Update(totalProcessed, totalNew, totalUpdated, totalFailed)
		tracker.IncrementBatch()

		// 更新游标
		cursor = nextCursor
		hasMore = more

		// 持久化进度 (Requirements: 3.2)
		syncLog.CurrentBatch = currentBatch
		syncLog.SyncCursor = cursor
		s.persistSyncProgress(ctx, account.UID, cursor, tracker.GetProgress())

		// 每 10 批或最后一批输出进度日志
		if currentBatch%10 == 0 || !hasMore {
			s.logger.Debug("同步进度: account=%s, batch=%d/%d, new=%d, updated=%d",
				account.UID, currentBatch, totalBatches, totalNew, totalUpdated)
		}
	}

	// 完成同步
	tracker.SetPhase(model.SyncPhaseFinalizing)
	syncLog.EmailsFetched = int64(totalProcessed)
	syncLog.EmailsNew = int64(totalNew)
	syncLog.EmailsUpdated = int64(totalUpdated)

	// 提供商保留期限制检测 (Requirements: 1.4)
	if isFirstSync && config.FirstSyncDays > 0 && estimatedCount > 0 && totalProcessed < estimatedCount/2 {
		s.logger.Warn("可能存在提供商保留期限制: account=%s, expected=%d, got=%d", account.UID, estimatedCount, totalProcessed)
	}

	// 清除游标（同步完成）
	s.clearSyncCursor(ctx, account.UID)

	// 如果本次同步有新增或更新的邮件，通过 SSE 通知前端刷新统计/列表缓存
	if totalNew > 0 || totalUpdated > 0 {
		sse.Broadcast("email_counts_maybe_changed", "{}")
	}

	// 同步成功，重置失败计数（仅对 quick 账号）
	if account.GetAuthType() == "quick" && account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			s.logger.Error("重置失败计数失败: account=%s, err=%v", account.UID, resetErr)
		}
	}

	s.logger.Info("同步完成: account=%s, new=%d, updated=%d, duration=%v",
		account.UID, totalNew, totalUpdated, time.Since(syncLog.StartedAt).Round(time.Second))

	tracker.Complete()
	return nil
}

// doSyncLegacy 传统同步模式（不支持分批的适配器）
func (s *syncService) doSyncLegacy(
	ctx context.Context,
	account *model.EmailAccount,
	syncLog *model.SyncLog,
	provider adapter.MailProvider,
	tracker ProgressTracker,
	config *model.SyncConfig,
	since time.Time,
	isFirstSync bool,
) error {
	// 开始进度追踪
	tracker.Start(account.UID, 0, isFirstSync)
	tracker.SetPhase(model.SyncPhaseFetching)

	// 拉取邮件列表
	limit := config.MaxEmailsPerSync
	if limit <= 0 {
		limit = 1000
	}

	emails, err := provider.FetchEmails(ctx, since, limit)
	if err != nil {
		tracker.Fail(err)
		return s.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails: %w", err))
	}

	syncLog.EmailsFetched = int64(len(emails))
	syncLog.TotalEstimated = len(emails)

	// 处理邮件
	tracker.SetPhase(model.SyncPhaseProcessing)
	var totalNew, totalUpdated, totalFailed int

	for i, email := range emails {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			tracker.Cancel()
			return ctx.Err()
		default:
		}

		if err := s.processEmail(ctx, account.UID, email, syncLog); err != nil {
			totalFailed++
			continue
		}

		// 更新进度
		tracker.Update(i+1, int(syncLog.EmailsNew), int(syncLog.EmailsUpdated), totalFailed)
	}

	totalNew = int(syncLog.EmailsNew)
	totalUpdated = int(syncLog.EmailsUpdated)

	// 完成同步
	tracker.SetPhase(model.SyncPhaseFinalizing)

	// 如果本次同步有新增或更新的邮件，通过 SSE 通知前端刷新统计/列表缓存
	if totalNew > 0 || totalUpdated > 0 {
		sse.Broadcast("email_counts_maybe_changed", "{}")
	}

	// 同步成功，重置失败计数（仅对 quick 账号）
	if account.GetAuthType() == "quick" && account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			s.logger.Error("重置失败计数失败: account=%s, err=%v", account.UID, resetErr)
		}
	}

	s.logger.Info("同步完成(Legacy): account=%s, new=%d, updated=%d",
		account.UID, totalNew, totalUpdated)

	tracker.Complete()
	return nil
}

// processBatchEmails 处理一批邮件
func (s *syncService) processBatchEmails(ctx context.Context, accountUID string, emails []*adapter.Email, syncLog *model.SyncLog) (newCount, updatedCount, failedCount int) {
	for _, email := range emails {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.processEmail(ctx, accountUID, email, syncLog); err != nil {
			failedCount++
			continue
		}
	}

	newCount = int(syncLog.EmailsNew)
	updatedCount = int(syncLog.EmailsUpdated)
	return
}

// persistSyncProgress 持久化同步进度
// Requirements: 3.2 - 每批处理后保存进度
func (s *syncService) persistSyncProgress(ctx context.Context, accountUID string, cursor string, progress *model.SyncProgress) {
	if progress == nil {
		return
	}

	progressJSON := progress.ToJSON()
	if err := s.accountRepo.UpdateSyncProgress(ctx, accountUID, cursor, progressJSON); err != nil {
		s.logger.Error("持久化同步进度失败: account=%s, err=%v", accountUID, err)
	}
}

// clearSyncCursor 清除同步游标（同步完成时调用）
func (s *syncService) clearSyncCursor(ctx context.Context, accountUID string) {
	if err := s.accountRepo.UpdateSyncProgress(ctx, accountUID, "", ""); err != nil {
		s.logger.Error("清除同步游标失败: account=%s, err=%v", accountUID, err)
	}
}

// processEmail 处理单封邮件（兼容旧模式，同时支持 dedupe_key）
func (s *syncService) processEmail(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) error {
	// 生成 dedupe_key (Requirements: 3.1, 3.2)
	dedupeKey := s.dedupeKeyGen.GenerateFromRaw(
		adapterEmail.MessageID,
		adapterEmail.FromAddress,
		adapterEmail.Subject,
		adapterEmail.SentAt,
	)

	// 检查是否在已删除列表中 (Requirements: 2.2)
	if s.deletedKeyRepo != nil {
		isDeleted, err := s.deletedKeyRepo.IsDeleted(ctx, accountUID, dedupeKey)
		if err != nil {
			s.logger.Warn("检查已删除标识失败: account=%s, key=%s, err=%v", accountUID, dedupeKey, err)
		} else if isDeleted {
			// 邮件已被删除，跳过
			return nil
		}
	}

	// 先通过 dedupe_key 查找（优先）
	existingEmail, err := s.emailRepo.FindByDedupeKey(ctx, accountUID, dedupeKey)
	if err != nil {
		return err
	}

	// 如果 dedupe_key 没找到，再通过 provider_id 查找（兼容旧数据）
	if existingEmail == nil {
		existingEmail, err = s.emailRepo.FindByProviderID(ctx, adapterEmail.ProviderID, accountUID)
		if err != nil {
			return err
		}
	}

	if existingEmail != nil {
		// 如果邮件已被软删除，跳过更新（不恢复已删除的邮件）
		if existingEmail.DeletedAt.Valid {
			// 邮件已被用户删除，跳过同步
			return nil
		}

		// 邮件已存在且未删除，更新
		s.updateEmailFromAdapter(existingEmail, adapterEmail, accountUID)
		// 更新 dedupe_key（如果之前没有）
		if existingEmail.DedupeKey == "" {
			existingEmail.DedupeKey = dedupeKey
		}
		if err := s.emailRepo.Update(ctx, existingEmail); err != nil {
			return err
		}
		// 应用规则到已存在邮件（更新后）
		if err := s.applyRulesForEmail(ctx, existingEmail); err != nil {
			s.logger.Warn("应用规则失败(更新): email=%d, err=%v", existingEmail.ID, err)
		}
		syncLog.EmailsUpdated++
	} else {
		// 新邮件，创建
		newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)
		newEmail.DedupeKey = dedupeKey // 设置 dedupe_key

		// 先保存邮件到数据库，获取正确的 ID
		if err := s.emailRepo.Create(ctx, newEmail); err != nil {
			return err
		}

		// 垃圾邮件检测（在邮件保存后执行，确保 email.ID 正确）
		if s.spamDetector != nil {
			spamResult, spamErr := s.spamDetector.DetectSpamSimple(ctx, newEmail)
			if spamErr != nil {
				s.logger.Warn("垃圾邮件检测失败: emailId=%d, msgId=%s, err=%v", newEmail.ID, newEmail.MessageID, spamErr)
			} else if spamResult != nil {
				// 更新邮件的垃圾检测结果
				newEmail.IsSpam = spamResult.IsSpam
				newEmail.SpamScore = float64(spamResult.Score)
				newEmail.SpamConfidence = spamResult.Confidence
				newEmail.SpamReason = spamResult.Reason
				newEmail.SpamDetectedBy = spamResult.DetectedBy
				if spamResult.IsSpam {
					now := time.Now()
					newEmail.SpamDetectedAt = &now
					s.logger.Info("检测到垃圾邮件: emailId=%d, subject=%s, score=%d", newEmail.ID, newEmail.Subject, spamResult.Score)
				}
				// 更新数据库中的垃圾检测结果
				if err := s.emailRepo.Update(ctx, newEmail); err != nil {
					s.logger.Warn("更新垃圾检测结果失败: emailId=%d, err=%v", newEmail.ID, err)
				}
			}
		}

		// 应用规则到新邮件
		if err := s.applyRulesForEmail(ctx, newEmail); err != nil {
			s.logger.Warn("应用规则失败(新建): email=%d, err=%v", newEmail.ID, err)
		}
		syncLog.EmailsNew++
	}

	return nil
}

// createEmailFromAdapter 从适配器邮件创建数据库邮件模型
func (s *syncService) createEmailFromAdapter(adapterEmail *adapter.Email, accountUID string) *model.Email {
	return &model.Email{
		ProviderID:       adapterEmail.ProviderID,
		AccountUID:       accountUID,
		MessageID:        adapterEmail.MessageID,
		Subject:          adapterEmail.Subject,
		FromAddress:      adapterEmail.FromAddress,
		FromName:         adapterEmail.FromName,
		ToAddresses:      s.joinAddresses(adapterEmail.ToAddresses),
		CcAddresses:      s.joinAddresses(adapterEmail.CcAddresses),
		BccAddresses:     s.joinAddresses(adapterEmail.BccAddresses),
		ReplyTo:          adapterEmail.ReplyTo,
		TextBody:         adapterEmail.TextBody,
		HTMLBody:         adapterEmail.HTMLBody,
		Snippet:          adapterEmail.Snippet,
		SourceIsRead:     adapterEmail.SourceIsRead,
		SourceLabels:     s.joinLabels(adapterEmail.SourceLabels),
		SourceFolder:     adapterEmail.SourceFolder,
		HasAttachments:   adapterEmail.HasAttachments,
		AttachmentsCount: adapterEmail.AttachmentsCount,
		SentAt:           adapterEmail.SentAt,
		ReceivedAt:       adapterEmail.ReceivedAt,
		SizeBytes:        adapterEmail.SizeBytes,
		ThreadID:         adapterEmail.ThreadID,
		InReplyTo:        adapterEmail.InReplyTo,
		References:       adapterEmail.References,
		SyncedAt:         time.Now(),
	}
}

// updateEmailFromAdapter 从适配器邮件更新数据库邮件模型
func (s *syncService) updateEmailFromAdapter(dbEmail *model.Email, adapterEmail *adapter.Email, accountUID string) {
	// 更新可能变化的字段
	dbEmail.Subject = adapterEmail.Subject
	dbEmail.TextBody = adapterEmail.TextBody
	dbEmail.HTMLBody = adapterEmail.HTMLBody
	dbEmail.Snippet = adapterEmail.Snippet
	dbEmail.SourceIsRead = adapterEmail.SourceIsRead
	dbEmail.SourceLabels = s.joinLabels(adapterEmail.SourceLabels)
	dbEmail.SourceFolder = adapterEmail.SourceFolder
	dbEmail.HasAttachments = adapterEmail.HasAttachments
	dbEmail.AttachmentsCount = adapterEmail.AttachmentsCount
	dbEmail.SizeBytes = adapterEmail.SizeBytes
	dbEmail.SyncedAt = time.Now()
}

// SyncAllAccounts 同步所有启用的账户（立即同步，不考虑同步间隔）
// 主要用于手动触发全量同步
// 使用信号量限制并发数，避免 Goroutine 泄露
func (s *syncService) SyncAllAccounts(ctx context.Context) error {
	// 获取所有启用同步的账户
	accounts, err := s.accountRepo.ListSyncEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sync enabled accounts: %w", err)
	}

	s.logger.Info("开始手动全量同步: accounts=%d", len(accounts))

	// 使用信号量限制并发数（最多 5 个并发同步）
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	// 并发同步账户（带并发控制）
	for _, account := range accounts {
		wg.Add(1)
		go func(accountUID string) {
			defer wg.Done()

			// 获取信号量
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				s.logger.Warn("同步取消: account=%s", accountUID)
				return
			}

			// 执行同步
			if err := s.SyncAccount(ctx, accountUID); err != nil {
				s.logger.Error("手动同步失败: account=%s, err=%v", accountUID, err)
			}
		}(account.UID)
	}

	// 等待所有同步完成（非阻塞返回，后台继续执行）
	go func() {
		wg.Wait()
		s.logger.Info("手动全量同步完成: accounts=%d", len(accounts))
	}()

	return nil
}

// StartScheduler 启动定时同步调度器
// 使用统一调度器 + 时间判断方案，支持每个账户的个性化同步间隔
func (s *syncService) StartScheduler(ctx context.Context) error {
	s.schedulerStop = make(chan struct{})

	go func() {
		// 每分钟检查一次，判断哪些账户需要同步
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		syncLog.Info("同步调度器已启动 (每分钟检查一次)")

		for {
			select {
			case <-ticker.C:
				s.checkAndSyncAccounts(ctx)
			case <-s.schedulerStop:
				syncLog.Info("同步调度器已停止")
				return
			case <-ctx.Done():
				syncLog.Info("同步调度器已取消")
				return
			}
		}
	}()

	return nil
}

// checkAndSyncAccounts 检查并同步需要同步的账户
// 根据每个账户的 sync_interval 和 last_sync_at 判断是否需要同步
func (s *syncService) checkAndSyncAccounts(ctx context.Context) {
	// 首先清理卡住的同步任务（超过 5 分钟仍在运行的任务）
	cleanedCount, cleanErr := s.CleanupStaleSyncLogs(ctx, 5*time.Minute)
	if cleanErr != nil {
		s.logger.Error("清理过期同步任务失败: %v", cleanErr)
	} else if cleanedCount > 0 {
		s.logger.Info("已清理过期同步任务: count=%d", cleanedCount)
	}

	// 获取所有启用同步的账户
	accounts, err := s.accountRepo.ListSyncEnabled(ctx)
	if err != nil {
		s.logger.Error("获取同步账户列表失败: %v", err)
		return
	}

	if len(accounts) == 0 {
		return
	}

	now := time.Now()
	syncCount := 0

	// 收集需要同步的账户
	var accountsToSync []*model.EmailAccount
	for _, account := range accounts {
		if s.shouldSync(account, now) {
			accountsToSync = append(accountsToSync, account)
		}
	}

	syncCount = len(accountsToSync)
	if syncCount == 0 {
		return
	}

	s.logger.Info("触发定时同步: triggered=%d, total=%d", syncCount, len(accounts))

	// 使用信号量限制并发数（最多 3 个并发定时同步）
	const maxConcurrent = 3
	sem := make(chan struct{}, maxConcurrent)

	// 异步同步账户（带并发控制）
	for _, account := range accountsToSync {
		go func(acc *model.EmailAccount) {
			// 获取信号量
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			// 执行同步
			if err := s.SyncAccount(ctx, acc.UID); err != nil {
				s.logger.Error("定时同步失败: account=%s, err=%v", acc.UID, err)
			}
		}(account)
	}
}

// shouldSync 判断账户是否需要同步
// 根据账户的 last_sync_at 和 sync_interval 计算是否到达下次同步时间
// 如果账户使用 Webhook 模式，则跳过轮询同步
func (s *syncService) shouldSync(account *model.EmailAccount, now time.Time) bool {
	// 检查是否使用 Webhook 模式（Webhook 模式不需要轮询同步）
	if account.IsWebhookMode() {
		s.logger.Debug("账户使用 Webhook 模式，跳过轮询同步: account=%s", account.UID)
		return false
	}

	// 首次同步（从未同步过）
	if account.LastSyncAt == nil {
		return true
	}

	// 计算下次同步时间
	syncInterval := time.Duration(account.SyncInterval) * time.Minute
	nextSyncTime := account.LastSyncAt.Add(syncInterval)

	// 判断是否到达或超过下次同步时间
	return now.After(nextSyncTime) || now.Equal(nextSyncTime)
}

// StopScheduler 停止定时同步调度器
func (s *syncService) StopScheduler() error {
	if s.schedulerStop != nil {
		close(s.schedulerStop)
		s.schedulerStop = nil
		syncLog.Info("同步调度器已停止")
	}
	return nil
}

// 辅助方法

// parseCredentials 解析认证凭证
func (s *syncService) parseCredentials(account *model.EmailAccount) (*adapter.Credentials, error) {
	// 解密凭证数据
	decryptedData, err := s.cryptoService.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 尝试从凭证 JSON 中读取 auth_type（优先级最高）
	// 这是因为批量导入的短效账户存储的是 JSON 格式凭证，其中包含 auth_type: "quick"
	// 但 AdapterRef.AuthType 可能是 "oauth2"（因为使用的是 graph 适配器）
	var credAuthType struct {
		AuthType string `json:"auth_type"`
	}
	authType := account.GetAuthType() // 默认从 AdapterRef 获取
	if json.Unmarshal(decryptedData, &credAuthType) == nil && credAuthType.AuthType != "" {
		// 凭证 JSON 中明确指定了 auth_type，使用它
		authType = credAuthType.AuthType
	}

	// 初始化凭证结构
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: authType,
	}

	// 根据认证类型处理凭证
	if authType == "quick" {
		// 短效认证凭证是 JSON 格式（必须在 oauth2 之前检查，因为 quick 也使用 graph 适配器）
		var quickCreds struct {
			Email        string `json:"email"`
			AuthType     string `json:"auth_type"`
			RefreshToken string `json:"refresh_token"`
			ClientID     string `json:"client_id"`
		}

		if err := json.Unmarshal(decryptedData, &quickCreds); err != nil {
			return nil, fmt.Errorf("failed to parse quick credentials: %w", err)
		}

		credentials.RefreshToken = quickCreds.RefreshToken
		credentials.ClientID = quickCreds.ClientID
		// 短效适配器不需要 ClientSecret，确保为空以触发 GraphQuickAdapter
		credentials.ClientSecret = ""
	} else if authType == "oauth2" {
		// OAuth2 凭证是 JSON 格式
		var oauthCreds struct {
			Email        string    `json:"email"`
			AuthType     string    `json:"auth_type"`
			AccessToken  string    `json:"access_token"`
			RefreshToken string    `json:"refresh_token"`
			TokenExpiry  time.Time `json:"token_expiry"`
		}

		if err := json.Unmarshal(decryptedData, &oauthCreds); err != nil {
			return nil, fmt.Errorf("failed to parse OAuth2 credentials: %w", err)
		}

		credentials.AccessToken = oauthCreds.AccessToken
		credentials.RefreshToken = oauthCreds.RefreshToken
		credentials.TokenExpiry = oauthCreds.TokenExpiry

		// 为 OAuth2 提供商设置 ClientID 和 ClientSecret
		// 这些凭证用于刷新 access_token
		// 优先使用 ProviderID 获取配置，避免硬编码 provider 名称
		if account.ProviderID > 0 {
			// 使用 ProviderID 获取 OAuth2 配置（推荐方式）
			oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2ConfigByProviderID(context.Background(), account.ProviderID)
			if err != nil {
				return nil, fmt.Errorf("failed to get OAuth2 config for provider_id %d: %w", account.ProviderID, err)
			}
			credentials.ClientID = oauth2Config.ClientID
			credentials.ClientSecret = oauth2Config.ClientSecret
		} else {
			// 回退：使用 provider 名称获取配置（兼容旧数据）
			providerName := account.GetProviderName()
			oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2ConfigByName(context.Background(), providerName)
			if err != nil {
				return nil, fmt.Errorf("failed to get OAuth2 config for provider %s: %w", providerName, err)
			}
			credentials.ClientID = oauth2Config.ClientID
			credentials.ClientSecret = oauth2Config.ClientSecret
		}
	} else {
		// 密码认证，直接使用解密后的数据作为密码
		credentials.Password = string(decryptedData)
	}

	// 设置服务器配置（优先从 Provider 获取，回退到账户废弃字段）
	protocol := account.GetProtocol()
	if protocol == "imap" {
		host, port, encryption := account.GetIMAPConfig()
		credentials.Host = host
		credentials.Port = port

		// 设置加密方式
		switch encryption {
		case "ssl", "":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true // 默认使用 SSL
		}
	} else if protocol == "pop3" {
		host, port, encryption := account.GetPOP3Config()
		credentials.Host = host
		credentials.Port = port

		// 设置加密方式
		switch encryption {
		case "ssl", "":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true // 默认使用 SSL
		}
	}

	// 智能修复常见的配置错误
	if credentials.Host == "mail.linuxdo.org" {
		credentials.Host = "mail.linux.do"
	}

	// 验证必要的配置（仅对 IMAP/POP3 协议需要 Host/Port）
	// OAuth2 协议使用 API 访问，不需要 Host/Port
	if protocol == "imap" || protocol == "pop3" {
		if credentials.Host == "" || credentials.Port == 0 {
			return nil, fmt.Errorf("server configuration missing: host=%s, port=%d (provider=%s, protocol=%s)",
				credentials.Host, credentials.Port, account.GetProviderName(), protocol)
		}
	}

	return credentials, nil
}

// parseProxyConfig 解析代理配置
func (s *syncService) parseProxyConfig(account *model.EmailAccount) (*adapter.ProxyConfig, error) {
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

// joinAddresses 将地址列表转换为 JSON 字符串
func (s *syncService) joinAddresses(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	data, _ := json.Marshal(addresses)
	return string(data)
}

// joinLabels 将标签列表转换为 JSON 字符串
func (s *syncService) joinLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	data, _ := json.Marshal(labels)
	return string(data)
}

// isAuthError 判断错误是否为认证错误
// 认证错误包括：HTTP 401、token 过期、invalid_grant、WebAPI 连接失败等
func (s *syncService) isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// 检查 HTTP 401 状态码
	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
		return true
	}

	// 检查 WebAPI 连接失败（包括解析错误、网络错误等）
	if strings.Contains(errMsg, "webapi 连接失败") || strings.Contains(errMsg, "webapi connection failed") {
		return true
	}

	// 检查 token 过期相关错误
	authErrorPatterns := []string{
		"token expired",
		"token has been expired",
		"invalid_grant",
		"authentication failed",
		"authenticate failed",
		"invalid credentials",
		"access denied",
		"auth",
	}

	for _, pattern := range authErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// handleSyncError 处理同步错误
// handleSyncError 处理同步错误
// 对于认证错误，进行失败计数和自动处理：
// - quick 类型账号：连续 3 次失败后自动禁用
// - oauth2 类型的 Outlook 账号：连续 10 次失败后自动软删除（放入回收站）
// - 其他类型账号（SMTP、WebAPI、IMAP、POP3 等）：连续 10 次失败后自动禁用
func (s *syncService) handleSyncError(ctx context.Context, account *model.EmailAccount, err error) error {
	// 判断是否为认证错误
	if !s.isAuthError(err) {
		return err
	}

	// 判断账号类型，决定处理策略
	authType := account.GetAuthType()
	providerName := account.GetProviderName()
	protocol := account.GetProtocol()

	isQuickAccount := authType == "quick"
	isOAuth2Outlook := authType == "oauth2" && providerName == "outlook"
	isOtherAccount := !isQuickAccount && !isOAuth2Outlook // SMTP、WebAPI、IMAP、POP3 等其他类型

	// 增加失败计数（所有账号类型都记录）
	failureCount, incErr := s.accountRepo.IncrementConsecutiveFailures(ctx, account.UID)
	if incErr != nil {
		s.logger.Error("增加失败计数失败: account=%s, err=%v", account.UID, incErr)
		return err
	}

	// 根据账号类型设置阈值和处理方式
	if isQuickAccount {
		// Quick 账号：3 次失败后自动禁用
		threshold := 3
		s.logger.Warn("Quick账号认证失败: account=%s, count=%d/%d", account.UID, failureCount, threshold)

		if failureCount >= threshold {
			disableErr := s.accountRepo.AutoDisableAccount(ctx, account.UID, "auto_disabled_auth_failure")
			if disableErr != nil {
				s.logger.Error("自动禁用账号失败: account=%s, err=%v", account.UID, disableErr)
			} else {
				s.logger.Info("已自动禁用Quick账号: account=%s, failures=%d", account.UID, failureCount)
			}
		}
	} else if isOAuth2Outlook {
		// OAuth2 Outlook 账号：10 次失败后自动软删除
		threshold := 10
		s.logger.Warn("OAuth2 Outlook账号认证失败: account=%s, count=%d/%d", account.UID, failureCount, threshold)

		if failureCount >= threshold {
			softDeleteErr := s.accountRepo.AutoSoftDeleteAccount(ctx, account.UID, "auto_recycled_token_invalid")
			if softDeleteErr != nil {
				s.logger.Error("自动回收账号失败: account=%s, err=%v", account.UID, softDeleteErr)
			} else {
				s.logger.Info("已自动回收OAuth2 Outlook账号: account=%s, failures=%d", account.UID, failureCount)
			}
		}
	} else if isOtherAccount {
		// 其他类型账号（SMTP、WebAPI、IMAP、POP3 等）：10 次失败后自动禁用
		threshold := 10
		s.logger.Warn("账号认证失败: account=%s, type=%s, protocol=%s, count=%d/%d",
			account.UID, authType, protocol, failureCount, threshold)

		if failureCount >= threshold {
			disableErr := s.accountRepo.AutoDisableAccount(ctx, account.UID, "auto_disabled_auth_failure")
			if disableErr != nil {
				s.logger.Error("自动禁用账号失败: account=%s, err=%v", account.UID, disableErr)
			} else {
				s.logger.Info("已自动禁用账号: account=%s, type=%s, protocol=%s, failures=%d",
					account.UID, authType, protocol, failureCount)
			}
		}
	}

	return err
}

// applyRulesForEmail 在同步阶段对单封邮件应用规则
func (s *syncService) applyRulesForEmail(ctx context.Context, email *model.Email) error {
	// 临时构建 ruleService（避免改动更大范围的依赖注入）
	ruleRepo := repository.NewRuleRepository(database.GetDB())
	rs := NewRuleService(ruleRepo, s.emailRepo)
	return rs.ApplyRules(ctx, email)
}

// CleanupStaleSyncLogs 清理卡住的同步日志
// 将超过指定时间仍处于 running 状态的同步日志标记为失败
// 同时更新对应账户的同步状态
func (s *syncService) CleanupStaleSyncLogs(ctx context.Context, maxAge time.Duration) (int, error) {
	// 查找所有卡住的同步日志
	staleLogs, err := s.syncLogRepo.FindStaleRunning(ctx, maxAge)
	if err != nil {
		return 0, fmt.Errorf("failed to find stale sync logs: %w", err)
	}

	if len(staleLogs) == 0 {
		return 0, nil
	}

	cleanedCount := 0
	errorMsg := fmt.Sprintf("同步超时 - 任务运行超过 %v 未完成，已自动清理", maxAge)

	for _, syncLog := range staleLogs {
		// 更新同步日志状态
		syncLog.Status = "failed"
		syncLog.ErrorMessage = errorMsg
		now := time.Now()
		syncLog.CompletedAt = &now
		syncLog.DurationMs = time.Since(syncLog.StartedAt).Milliseconds()

		if updateErr := s.syncLogRepo.Update(ctx, syncLog); updateErr != nil {
			s.logger.Error("更新过期同步日志失败: logId=%d, err=%v", syncLog.ID, updateErr)
			continue
		}

		// 更新账户的同步状态（保存错误信息）
		if updateErr := s.accountRepo.UpdateSyncStatus(ctx, syncLog.AccountUID, "failed", errorMsg); updateErr != nil {
			s.logger.Error("更新账户同步状态失败: account=%s, err=%v", syncLog.AccountUID, updateErr)
		}

		// 清理内存中的锁（如果存在）
		s.syncMu.Lock()
		if lockInfo, exists := s.activeSyncs[syncLog.AccountUID]; exists {
			if lockInfo.CancelFunc != nil {
				lockInfo.CancelFunc()
			}
			delete(s.activeSyncs, syncLog.AccountUID)
			delete(s.activeTrackers, syncLog.AccountUID)
		}
		s.syncMu.Unlock()

		// 清理 Redis 锁（如果存在）
		if s.syncLock != nil {
			_ = s.syncLock.ForceReleaseLock(ctx, syncLog.AccountUID)
		}

		s.logger.Info("已清理过期同步日志: logId=%d, account=%s", syncLog.ID, syncLog.AccountUID)
		cleanedCount++
	}

	return cleanedCount, nil
}

// doSyncWebAPI 执行 WebAPI 协议的同步逻辑
// 专门处理 WebAPI 类型的邮箱账户同步
func (s *syncService) doSyncWebAPI(ctx context.Context, account *model.EmailAccount, syncLog *model.SyncLog, syncConfig *model.SyncConfig, isFirstSync bool) error {
	s.logger.Info("WebAPI 同步开始: account=%s, provider=%s, isFirstSync=%v", account.UID, account.GetProviderName(), isFirstSync)

	// 获取 Provider 信息（包含 WebAPI 服务类型）
	if account.ProviderRef == nil {
		return fmt.Errorf("WebAPI 账户缺少 Provider 信息: %s", account.UID)
	}

	// 从 Provider 获取 WebAPI 服务类型
	serviceType, err := model.GetWebAPIServiceType(account.ProviderRef)
	if err != nil {
		return fmt.Errorf("获取 WebAPI 服务类型失败: %w", err)
	}

	s.logger.Debug("WebAPI 服务类型: %s", serviceType)

	// 解密认证数据
	authDataJSON := account.EncryptedCredentials
	if s.cryptoService != nil && authDataJSON != "" {
		decrypted, err := s.cryptoService.Decrypt(authDataJSON)
		if err != nil {
			s.logger.Warn("解密 AuthData 失败，尝试使用原始数据: %v", err)
		} else {
			authDataJSON = string(decrypted)
		}
	}

	// 创建 WebAPI 适配器
	webAPIProvider, err := s.webAPIAdapterFactory.CreateAdapter(serviceType, authDataJSON)
	if err != nil {
		return fmt.Errorf("创建 WebAPI 适配器失败: %w", err)
	}

	// 连接到 WebAPI 服务
	if err := webAPIProvider.Connect(ctx); err != nil {
		return s.handleSyncError(ctx, account, fmt.Errorf("WebAPI 连接失败: %w", err))
	}
	defer webAPIProvider.Disconnect()

	// 创建进度追踪器
	tracker := NewProgressTracker(syncConfig.ProgressInterval)

	// 注册进度追踪器
	s.syncMu.Lock()
	s.activeTrackers[account.UID] = tracker
	s.syncMu.Unlock()

	// 计算同步起始时间
	// WebAPI 首次同步使用全量模式（since 为零值），因为 WebAPI 服务通常数据量较小
	// 且邮件时间可能较早，使用 FirstSyncDays 可能会过滤掉所有邮件
	var since time.Time
	if isFirstSync || account.LastSyncAt == nil {
		// 首次同步：使用零值时间，拉取所有邮件
		s.logger.Info("WebAPI 首次同步，使用全量模式: account=%s", account.UID)
		since = time.Time{}
	} else {
		// 增量同步：从上次同步时间开始，减去 1 小时缓冲
		since = account.LastSyncAt.Add(-1 * time.Hour)
		s.logger.Info("WebAPI 增量同步: account=%s, since=%s", account.UID, since.Format(time.RFC3339))
	}

	// 拉取邮件
	limit := 100 // 默认拉取数量
	if syncConfig.BatchSize > 0 {
		limit = syncConfig.BatchSize
	}

	emails, err := webAPIProvider.FetchEmails(ctx, since, limit)
	if err != nil {
		return fmt.Errorf("WebAPI 拉取邮件失败: %w", err)
	}

	s.logger.Info("WebAPI 拉取邮件完成: account=%s, count=%d", account.UID, len(emails))

	// 处理邮件
	newCount := 0
	updatedCount := 0
	skippedCount := 0

	for _, email := range emails {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 转换为数据库模型
		dbEmail := s.convertWebAPIEmailToModel(email, account)

		// 生成去重标识
		dedupeKey := s.dedupeKeyGen.Generate(dbEmail)
		dbEmail.DedupeKey = dedupeKey

		// 检查是否已存在
		existing, err := s.emailRepo.FindByDedupeKey(ctx, account.UID, dedupeKey)
		if err != nil {
			s.logger.Warn("检查邮件是否存在失败: %v", err)
		}

		if existing != nil {
			// 更新现有邮件
			dbEmail.ID = existing.ID
			if err := s.emailRepo.Update(ctx, dbEmail); err != nil {
				s.logger.Warn("更新邮件失败: %v", err)
				skippedCount++
				continue
			}
			updatedCount++
		} else {
			// 创建新邮件
			if err := s.emailRepo.Create(ctx, dbEmail); err != nil {
				s.logger.Warn("创建邮件失败: %v", err)
				skippedCount++
				continue
			}
			newCount++
		}

		// 更新进度
		tracker.Update(newCount+updatedCount+skippedCount, newCount, updatedCount, skippedCount)
	}

	// 更新同步日志
	syncLog.EmailsNew = int64(newCount)
	syncLog.EmailsUpdated = int64(updatedCount)
	syncLog.EmailsFetched = int64(len(emails))

	// 更新账户的最后同步时间
	now := time.Now()
	account.LastSyncAt = &now
	if err := s.accountRepo.Update(ctx, account); err != nil {
		s.logger.Warn("更新账户最后同步时间失败: %v", err)
	}

	s.logger.Info("WebAPI 同步完成: account=%s, new=%d, updated=%d, skipped=%d",
		account.UID, newCount, updatedCount, skippedCount)

	// 发送 SSE 事件通知前端
	eventData, _ := json.Marshal(map[string]interface{}{
		"account_uid": account.UID,
		"new_count":   newCount,
	})
	sse.Broadcast("email_counts_maybe_changed", string(eventData))

	return nil
}

// convertWebAPIEmailToModel 将 WebAPI 邮件转换为数据库模型
func (s *syncService) convertWebAPIEmailToModel(email *adapter.Email, account *model.EmailAccount) *model.Email {
	dbEmail := &model.Email{
		AccountUID:       account.UID,
		ProviderID:       email.ProviderID,
		MessageID:        email.MessageID,
		Subject:          email.Subject,
		FromAddress:      email.FromAddress,
		FromName:         email.FromName,
		Snippet:          email.Snippet,
		TextBody:         email.TextBody,
		HTMLBody:         email.HTMLBody,
		SentAt:           email.SentAt,
		ReceivedAt:       email.ReceivedAt,
		HasAttachments:   email.HasAttachments,
		AttachmentsCount: email.AttachmentsCount,
	}

	// 处理收件人地址
	if len(email.ToAddresses) > 0 {
		toJSON, _ := json.Marshal(email.ToAddresses)
		dbEmail.ToAddresses = string(toJSON)
	}

	// 处理抄送地址
	if len(email.CcAddresses) > 0 {
		ccJSON, _ := json.Marshal(email.CcAddresses)
		dbEmail.CcAddresses = string(ccJSON)
	}

	// 处理密送地址
	if len(email.BccAddresses) > 0 {
		bccJSON, _ := json.Marshal(email.BccAddresses)
		dbEmail.BccAddresses = string(bccJSON)
	}

	// 处理标签（从源邮箱标签）
	if len(email.SourceLabels) > 0 {
		labelsJSON, _ := json.Marshal(email.SourceLabels)
		dbEmail.Labels = string(labelsJSON)
	}

	return dbEmail
}
