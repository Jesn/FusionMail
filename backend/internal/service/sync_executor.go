package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/pkg/synclock"
)

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
		return s.handleSyncError(ctx, account, fmt.Errorf("failed to connect: %w", err))
	}
	defer provider.Disconnect()

	// 创建进度追踪器 (Requirements: 2.1, 2.2)
	tracker := NewProgressTrackerWithNotifier(syncConfig.ProgressInterval, s.notifier)

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

	// 如果本次同步有新增或更新的邮件，通知前端刷新统计/列表缓存
	if totalNew > 0 || totalUpdated > 0 {
		NotifyEmailCountsMaybeChanged(s.notifier, nil)
	}

	// 同步成功，重置失败计数（所有账号类型）
	if account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			s.logger.Error("重置失败计数失败: account=%s, err=%v", account.UID, resetErr)
		} else {
			s.logger.Info("已重置失败计数: account=%s, 原失败次数=%d", account.UID, account.ConsecutiveAuthFailures)
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

	// 如果本次同步有新增或更新的邮件，通知前端刷新统计/列表缓存
	if totalNew > 0 || totalUpdated > 0 {
		NotifyEmailCountsMaybeChanged(s.notifier, nil)
	}

	// 同步成功，重置失败计数（所有账号类型）
	if account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			s.logger.Error("重置失败计数失败: account=%s, err=%v", account.UID, resetErr)
		} else {
			s.logger.Info("已重置失败计数: account=%s, 原失败次数=%d", account.UID, account.ConsecutiveAuthFailures)
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

	// 如果本次同步有新增或更新的邮件，通知前端刷新统计/列表缓存
	if totalNew > 0 || totalUpdated > 0 {
		NotifyEmailCountsMaybeChanged(s.notifier, nil)
	}

	// 同步成功，重置失败计数（所有账号类型）
	if account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			s.logger.Error("重置失败计数失败: account=%s, err=%v", account.UID, resetErr)
		} else {
			s.logger.Info("已重置失败计数: account=%s, 原失败次数=%d", account.UID, account.ConsecutiveAuthFailures)
		}
	}

	s.logger.Info("同步完成(Legacy): account=%s, new=%d, updated=%d",
		account.UID, totalNew, totalUpdated)

	tracker.Complete()
	return nil
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
