package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"fusionmail/internal/adapter"
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
	adapterFactory       *adapter.Factory
	cryptoService        *crypto.Service
	schedulerStop        chan struct{}
	oauth2ConfigProvider *oauth2config.Provider // 新增：OAuth2配置提供者
	spamDetector         SpamDetectorInterface  // 垃圾邮件检测器

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
	adapterFactory *adapter.Factory,
	oauth2ClientRepo repository.OAuth2ClientRepository,
	providerRepo repository.ProviderRepository,
	logger *logger.Logger,
	cryptoService *crypto.Service, // 添加加密服务参数（指针类型）
	spamDetector SpamDetectorInterface, // 垃圾邮件检测器（可选）
	redisClient *redis.Client, // Redis 客户端，用于分布式锁
) SyncService {

	// 创建OAuth2配置提供者
	oauth2Provider := oauth2config.NewProvider(oauth2ClientRepo, providerRepo, cryptoService, logger)

	// 创建分布式同步锁（如果 Redis 可用）
	var sl *synclock.SyncLock
	if redisClient != nil {
		sl = synclock.NewSyncLock(redisClient)
		log.Printf("[INFO] Distributed sync lock enabled (Redis-based)")
	} else {
		log.Printf("[WARN] Redis client not available, using in-memory sync lock (not recommended for production)")
	}

	return &syncService{
		accountRepo:          accountRepo,
		emailRepo:            emailRepo,
		syncLogRepo:          syncLogRepo,
		adapterFactory:       adapterFactory,
		cryptoService:        cryptoService,
		oauth2ConfigProvider: oauth2Provider,
		spamDetector:         spamDetector,
		syncLock:             sl,
		activeSyncs:          make(map[string]*synclock.LockInfo),
		activeTrackers:       make(map[string]ProgressTracker),
	}
}

// SyncAccount 同步指定账户的邮件
// 使用 Redis 分布式锁防止重复同步，支持自动超时和锁续期
func (s *syncService) SyncAccount(ctx context.Context, accountUID string) error {
	log.Printf("[DEBUG] SyncAccount called for UID: %s", accountUID)

	// 获取账户信息
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
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
				log.Printf("[WARN] Found stale memory lock for account %s (Redis lock expired), cleaning up", accountUID)
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
				log.Printf("[WARN] Failed to release sync lock for account %s: %v", accountUID, releaseErr)
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
		log.Printf("Failed to create sync log: %v", err)
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

	// 保存同步日志到数据库
	if updateErr := s.syncLogRepo.Update(ctx, syncLog); updateErr != nil {
		log.Printf("Failed to update sync log: %v", updateErr)
	}

	// 更新账户同步状态（只更新同步相关字段，避免覆盖其他字段如 consecutive_auth_failures）
	if updateErr := s.accountRepo.UpdateSyncStatus(ctx, accountUID, syncLog.Status, syncLog.ErrorMessage); updateErr != nil {
		log.Printf("Failed to update account sync status: %v", updateErr)
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
	log.Printf("[INFO] Sync cancelled for account: %s", accountUID)

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
	log.Printf("[DEBUG] Starting sync for account %s (email: %s, auth_type: %s)", account.UID, account.Email, account.AuthType)

	// 获取同步配置
	syncConfig := account.GetSyncConfig()
	isFirstSync := account.LastSyncAt == nil

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
		Provider:    account.Provider,
		Protocol:    account.Protocol,
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

	// 计算同步起始时间 (Requirements: 1.2, 5.3, 7.5)
	since := s.calculateSyncSince(account, syncConfig, isFirstSync)

	// 创建进度追踪器 (Requirements: 2.1, 2.2)
	tracker := NewProgressTracker(syncConfig.ProgressInterval)

	// 注册进度追踪器
	s.syncMu.Lock()
	s.activeTrackers[account.UID] = tracker
	s.syncMu.Unlock()

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
	// 记录当前服务器时间（UTC），便于调试
	now := time.Now().UTC()
	log.Printf("[DEBUG] Server time (UTC): %s", now.Format(time.RFC3339))

	// 如果有同步游标且不是首次同步，尝试从游标恢复
	if account.SyncCursor != "" && !isFirstSync {
		// 游标格式可能包含时间戳，这里简化处理
		// 实际实现中可以解析游标获取更精确的位置
		log.Printf("[DEBUG] Resuming sync from cursor for account %s", account.UID)
	}

	if account.LastSyncAt != nil {
		// 增量同步：从上次同步时间开始，减去缓冲时间
		//
		// 为什么需要缓冲时间：
		// 1. IMAP SEARCH 的 SINCE 条件只精确到日期（不含时间）
		// 2. 时区差异可能导致边界邮件被遗漏
		// 3. 邮件服务器的时间可能与我们的服务器有偏差
		//
		// 为什么 1 小时足够：
		// - 适配器层已移除时间过滤，完全依赖 ProviderID 去重
		// - 即使拉取了重复邮件，也只会更新而不会重复创建
		// - 1 小时缓冲足以覆盖大多数时区和时间偏差问题
		since := account.LastSyncAt.Add(-1 * time.Hour)
		log.Printf("[DEBUG] Incremental sync for account %s, LastSyncAt: %s, since: %s",
			account.UID, account.LastSyncAt.Format(time.RFC3339), since.Format(time.RFC3339))
		return since
	}

	// 首次同步：根据配置计算起始时间 (Requirements: 1.2)
	if config.IsFullSync() {
		// 全量同步：不限制时间
		log.Printf("[DEBUG] Full sync for account %s (first_sync_days=0)", account.UID)
		return time.Time{}
	}

	// 从 N 天前开始（使用 UTC）
	since := time.Now().UTC().AddDate(0, 0, -config.FirstSyncDays)
	log.Printf("[DEBUG] First sync for account %s, since: %s (%d days)", account.UID, since.Format(time.RFC3339), config.FirstSyncDays)
	return since
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
		log.Printf("[WARN] Failed to get estimated count for account %s: %v", account.UID, err)
		estimatedCount = 0
	}

	// 大邮箱警告 (Requirements: 3.3)
	if estimatedCount > 5000 {
		log.Printf("[WARN] Large mailbox detected for account %s: estimated %d emails", account.UID, estimatedCount)
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
			log.Printf("[INFO] Reached max emails per sync limit (%d) for account %s", config.MaxEmailsPerSync, account.UID)
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

		log.Printf("[DEBUG] Batch %d/%d completed for account %s: processed=%d, new=%d, updated=%d",
			currentBatch, totalBatches, account.UID, len(emails), batchNew, batchUpdated)
	}

	// 完成同步
	tracker.SetPhase(model.SyncPhaseFinalizing)
	syncLog.EmailsFetched = int64(totalProcessed)
	syncLog.EmailsNew = int64(totalNew)
	syncLog.EmailsUpdated = int64(totalUpdated)

	// 提供商保留期限制检测 (Requirements: 1.4)
	// 如果配置了同步天数但实际获取的邮件数远少于预期，可能是提供商有保留期限制
	if isFirstSync && config.FirstSyncDays > 0 && estimatedCount > 0 {
		// 如果实际获取的邮件数少于预估的 50%，记录警告
		if totalProcessed < estimatedCount/2 {
			log.Printf("[WARN] Provider retention limit may apply for account %s: expected ~%d emails for %d days, got %d. Provider may have shorter retention period.",
				account.UID, estimatedCount, config.FirstSyncDays, totalProcessed)
		}
	}

	// 清除游标（同步完成）
	s.clearSyncCursor(ctx, account.UID)

	// 如果本次同步有新增或更新的邮件，通过 SSE 通知前端刷新统计/列表缓存
	if totalNew > 0 || totalUpdated > 0 {
		sse.Broadcast("email_counts_maybe_changed", "{}")
	}

	// 同步成功，重置失败计数（仅对 quick 账号）
	if account.AuthType == "quick" && account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			log.Printf("[ERROR] Failed to reset failure counter for account %s: %v", account.UID, resetErr)
		} else {
			log.Printf("[DEBUG] Reset failure counter for quick account %s after successful sync", account.UID)
		}
	}

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
	if account.AuthType == "quick" && account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			log.Printf("[ERROR] Failed to reset failure counter for account %s: %v", account.UID, resetErr)
		} else {
			log.Printf("[DEBUG] Reset failure counter for quick account %s after successful sync", account.UID)
		}
	}

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
		log.Printf("[ERROR] Failed to persist sync progress for account %s: %v", accountUID, err)
	}
}

// clearSyncCursor 清除同步游标（同步完成时调用）
func (s *syncService) clearSyncCursor(ctx context.Context, accountUID string) {
	if err := s.accountRepo.UpdateSyncProgress(ctx, accountUID, "", ""); err != nil {
		log.Printf("[ERROR] Failed to clear sync cursor for account %s: %v", accountUID, err)
	}
}

// processEmail 处理单封邮件
func (s *syncService) processEmail(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) error {
	// 检查邮件是否已存在
	existingEmail, err := s.emailRepo.FindByProviderID(ctx, adapterEmail.ProviderID, accountUID)
	if err != nil {
		return err
	}

	if existingEmail != nil {
		// 邮件已存在，更新
		s.updateEmailFromAdapter(existingEmail, adapterEmail, accountUID)
		if err := s.emailRepo.Update(ctx, existingEmail); err != nil {
			return err
		}
		// 应用规则到已存在邮件（更新后）
		if err := s.applyRulesForEmail(ctx, existingEmail); err != nil {
			log.Printf("[WARN] Failed to apply rules to existing email %d: %v", existingEmail.ID, err)
		}
		syncLog.EmailsUpdated++
	} else {
		// 新邮件，创建
		newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)

		// 垃圾邮件检测（仅对新邮件）
		if s.spamDetector != nil {
			spamResult, spamErr := s.spamDetector.DetectSpamSimple(ctx, newEmail)
			if spamErr != nil {
				log.Printf("[WARN] Spam detection failed for email %s: %v", newEmail.MessageID, spamErr)
			} else if spamResult != nil {
				newEmail.IsSpam = spamResult.IsSpam
				newEmail.SpamScore = float64(spamResult.Score)
				newEmail.SpamConfidence = spamResult.Confidence
				newEmail.SpamReason = spamResult.Reason
				newEmail.SpamDetectedBy = spamResult.DetectedBy
				if spamResult.IsSpam {
					now := time.Now()
					newEmail.SpamDetectedAt = &now
					log.Printf("[INFO] 检测到垃圾邮件: %s (评分: %d, 置信度: %.2f, 原因: %s)",
						newEmail.Subject, spamResult.Score, spamResult.Confidence, spamResult.Reason)
				}
			}
		}

		if err := s.emailRepo.Create(ctx, newEmail); err != nil {
			return err
		}
		// 应用规则到新邮件
		if err := s.applyRulesForEmail(ctx, newEmail); err != nil {
			log.Printf("[WARN] Failed to apply rules to new email %d: %v", newEmail.ID, err)
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
func (s *syncService) SyncAllAccounts(ctx context.Context) error {
	// 获取所有启用同步的账户
	accounts, err := s.accountRepo.ListSyncEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sync enabled accounts: %w", err)
	}

	log.Printf("Starting manual sync for %d accounts", len(accounts))

	// 并发同步账户
	for _, account := range accounts {
		go func(accountUID string) {
			if err := s.SyncAccount(ctx, accountUID); err != nil {
				log.Printf("Manual sync failed for account %s: %v", accountUID, err)
			}
		}(account.UID)
	}

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

		log.Println("[Scheduler] Sync scheduler started (checking every 1 minute)")

		for {
			select {
			case <-ticker.C:
				s.checkAndSyncAccounts(ctx)
			case <-s.schedulerStop:
				log.Println("[Scheduler] Sync scheduler stopped")
				return
			case <-ctx.Done():
				log.Println("[Scheduler] Sync scheduler cancelled")
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
		log.Printf("[Scheduler] Failed to cleanup stale sync logs: %v", cleanErr)
	} else if cleanedCount > 0 {
		log.Printf("[Scheduler] Cleaned up %d stale sync tasks", cleanedCount)
	}

	// 获取所有启用同步的账户
	accounts, err := s.accountRepo.ListSyncEnabled(ctx)
	if err != nil {
		log.Printf("[Scheduler] Failed to list sync enabled accounts: %v", err)
		return
	}

	if len(accounts) == 0 {
		return
	}

	now := time.Now()
	syncCount := 0

	// 检查每个账户是否需要同步
	for _, account := range accounts {
		if s.shouldSync(account, now) {
			syncCount++
			log.Printf("[Scheduler] Triggering sync for account %s (email: %s, interval: %d min)",
				account.UID, account.Email, account.SyncInterval)

			// 异步同步账户
			go func(acc *model.EmailAccount) {
				if err := s.SyncAccount(ctx, acc.UID); err != nil {
					log.Printf("[Scheduler] Sync failed for account %s: %v", acc.UID, err)
				}
			}(account)
		}
	}

	if syncCount > 0 {
		log.Printf("[Scheduler] Triggered sync for %d/%d accounts", syncCount, len(accounts))
	}
}

// shouldSync 判断账户是否需要同步
// 根据账户的 last_sync_at 和 sync_interval 计算是否到达下次同步时间
func (s *syncService) shouldSync(account *model.EmailAccount, now time.Time) bool {
	// 首次同步（从未同步过）
	if account.LastSyncAt == nil {
		log.Printf("[Scheduler] Account %s needs first sync", account.UID)
		return true
	}

	// 计算下次同步时间
	syncInterval := time.Duration(account.SyncInterval) * time.Minute
	nextSyncTime := account.LastSyncAt.Add(syncInterval)

	// 判断是否到达或超过下次同步时间
	shouldSync := now.After(nextSyncTime) || now.Equal(nextSyncTime)

	if shouldSync {
		timeSinceLastSync := now.Sub(*account.LastSyncAt)
		log.Printf("[Scheduler] Account %s ready for sync (last: %s ago, interval: %d min)",
			account.UID,
			timeSinceLastSync.Round(time.Minute),
			account.SyncInterval)
	}

	return shouldSync
}

// StopScheduler 停止定时同步调度器
func (s *syncService) StopScheduler() error {
	if s.schedulerStop != nil {
		close(s.schedulerStop)
		s.schedulerStop = nil
		log.Println("Sync scheduler stopped")
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

	// 初始化凭证结构
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: account.AuthType,
	}

	// 根据认证类型处理凭证
	if account.AuthType == "oauth2" {
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
		if account.Provider == "gmail" && account.Protocol == "gmail_api" {
			// Gmail API OAuth2 配置 - 从数据库获取（使用provider_type）
			oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2Config(context.Background(), int(model.ProviderTypeGmail))
			if err != nil {
				return nil, fmt.Errorf("failed to get Gmail OAuth2 config from database: %w", err)
			}
			credentials.ClientID = oauth2Config.ClientID
			credentials.ClientSecret = oauth2Config.ClientSecret
		} else if account.Provider == "outlook" && account.Protocol == "graph" {
			// Microsoft Graph API OAuth2 配置 - 从数据库获取（使用provider_type）
			oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2Config(context.Background(), int(model.ProviderTypeOutlook))
			if err != nil {
				return nil, fmt.Errorf("failed to get Outlook OAuth2 config from database: %w", err)
			}
			credentials.ClientID = oauth2Config.ClientID
			credentials.ClientSecret = oauth2Config.ClientSecret
		}
	} else if account.AuthType == "quick" {
		// 短效认证凭证是 JSON 格式
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
		// 短效适配器不需要 ClientSecret
	} else {
		// 密码认证，直接使用解密后的数据作为密码
		credentials.Password = string(decryptedData)
	}

	// 设置 IMAP 服务器配置
	// 如果用户手动配置了服务器地址，优先使用用户配置
	if account.IMAPHost != "" && account.IMAPPort != 0 {
		credentials.Host = account.IMAPHost
		credentials.Port = account.IMAPPort
		credentials.TLS = true // 默认开启 TLS，后续可根据 Encryption 字段调整
	} else {
		// 使用预设的服务器配置
		switch account.Provider {
		case "icloud":
			credentials.Host = "imap.mail.me.com"
			credentials.Port = 993
			credentials.TLS = true
		case "qq":
			credentials.Host = "imap.qq.com"
			credentials.Port = 993
			credentials.TLS = true
		case "163":
			credentials.Host = "imap.163.com"
			credentials.Port = 993
			credentials.TLS = true
		case "gmail":
			credentials.Host = "imap.gmail.com"
			credentials.Port = 993
			credentials.TLS = true
		case "outlook":
			credentials.Host = "outlook.office365.com"
			credentials.Port = 993
			credentials.TLS = true
		case "generic":
			// generic 必须配置服务器信息，如果上面没有配置（即 IMAPHost 为空），这里会报错
		default:
			return nil, fmt.Errorf("unsupported provider: %s", account.Provider)
		}
	}

	// 对于 generic 或手动配置的情况，进行额外检查和设置
	if account.Provider == "generic" || (account.IMAPHost != "" && account.IMAPPort != 0) {
		if account.Protocol == "imap" {
			// 已经在上面设置了，这里再次确认（如果是 generic 且没有手动配置，会在下面报错）
			if credentials.Host == "" {
				credentials.Host = account.IMAPHost
				credentials.Port = account.IMAPPort
			}
		} else if account.Protocol == "pop3" {
			credentials.Host = account.POP3Host
			credentials.Port = account.POP3Port
		}

		// 智能修复常见的配置错误
		if credentials.Host == "mail.linuxdo.org" {
			// Auto-fixing incorrect host configuration
			credentials.Host = "mail.linux.do"
		}

		// 设置加密方式
		switch account.Encryption {
		case "ssl":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true // 默认使用 SSL
		}

		// 验证必要的配置
		if credentials.Host == "" || credentials.Port == 0 {
			return nil, fmt.Errorf("provider requires host and port configuration")
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
// 认证错误包括：HTTP 401、token 过期、invalid_grant 等
func (s *syncService) isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// 检查 HTTP 401 状态码
	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
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
// 对于认证错误，进行失败计数和自动处理：
// - quick 类型账号：连续 3 次失败后自动禁用
// - oauth2 类型的 Outlook 账号：连续 10 次失败后自动软删除（放入回收站）
func (s *syncService) handleSyncError(ctx context.Context, account *model.EmailAccount, err error) error {
	// 判断是否为认证错误
	if !s.isAuthError(err) {
		log.Printf("[DEBUG] Account %s sync failed with non-auth error: %v", account.UID, err)
		return err
	}

	// 判断账号类型，决定处理策略
	// quick 类型：短效邮箱，阈值 3 次，禁用处理
	// oauth2 + outlook：批量导入邮箱，阈值 10 次，软删除处理
	isQuickAccount := account.AuthType == "quick"
	isOAuth2Outlook := account.AuthType == "oauth2" && account.Provider == "outlook"

	// 仅对 quick 和 oauth2+outlook 类型账号进行特殊处理
	if !isQuickAccount && !isOAuth2Outlook {
		log.Printf("[DEBUG] Account %s (type: %s, provider: %s) auth error, no auto-action configured: %v",
			account.UID, account.AuthType, account.Provider, err)
		return err
	}

	// 增加失败计数
	failureCount, incErr := s.accountRepo.IncrementConsecutiveFailures(ctx, account.UID)
	if incErr != nil {
		log.Printf("[ERROR] Failed to increment failure counter for account %s: %v", account.UID, incErr)
		return err
	}

	// 根据账号类型设置阈值和处理方式
	if isQuickAccount {
		// quick 类型：阈值 3 次，禁用处理
		threshold := 3
		log.Printf("[WARN] Quick account %s (email: %s) auth failure count: %d/%d - Error: %v",
			account.UID, account.Email, failureCount, threshold, err)

		if failureCount >= threshold {
			disableErr := s.accountRepo.AutoDisableAccount(
				ctx,
				account.UID,
				"auto_disabled_auth_failure",
			)

			if disableErr != nil {
				log.Printf("[ERROR] Failed to auto-disable account %s: %v", account.UID, disableErr)
			} else {
				log.Printf("[INFO] Auto-disabled quick account %s (email: %s) after %d consecutive auth failures",
					account.UID, account.Email, failureCount)
			}
		}
	} else if isOAuth2Outlook {
		// oauth2 + outlook 类型：阈值 10 次，软删除处理（放入回收站）
		threshold := 10
		log.Printf("[WARN] OAuth2 Outlook account %s (email: %s) auth failure count: %d/%d - Error: %v",
			account.UID, account.Email, failureCount, threshold, err)

		if failureCount >= threshold {
			softDeleteErr := s.accountRepo.AutoSoftDeleteAccount(
				ctx,
				account.UID,
				"auto_recycled_token_invalid",
			)

			if softDeleteErr != nil {
				log.Printf("[ERROR] Failed to auto-recycle account %s: %v", account.UID, softDeleteErr)
			} else {
				log.Printf("[INFO] Auto-recycled OAuth2 Outlook account %s (email: %s) after %d consecutive auth failures (token expired/invalid)",
					account.UID, account.Email, failureCount)
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
			log.Printf("[ERROR] Failed to update stale sync log %d: %v", syncLog.ID, updateErr)
			continue
		}

		// 更新账户的同步状态（保存错误信息）
		if updateErr := s.accountRepo.UpdateSyncStatus(ctx, syncLog.AccountUID, "failed", errorMsg); updateErr != nil {
			log.Printf("[ERROR] Failed to update account sync status for %s: %v", syncLog.AccountUID, updateErr)
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

		log.Printf("[INFO] Cleaned up stale sync log %d for account %s (started: %s)",
			syncLog.ID, syncLog.AccountUID, syncLog.StartedAt.Format(time.RFC3339))
		cleanedCount++
	}

	return cleanedCount, nil
}
