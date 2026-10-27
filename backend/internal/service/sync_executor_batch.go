package service

// SyncExecutor 的分批/传统同步路径与进度持久化：
// doSyncWithBatch、doSyncLegacy、persistSyncProgress、clearSyncCursor。
// 自 sync_executor.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// doSyncWithBatch 使用分批处理模式同步
// Requirements: 3.1, 3.2 - 分批处理和进度持久化
func (s *SyncExecutor) doSyncWithBatch(
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
				return s.errorPolicy.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails batch: %w", retryErr))
			}
		}

		// 处理这批邮件
		tracker.SetPhase(model.SyncPhaseProcessing)
		batchNew, batchUpdated, batchFailed := s.processor.processBatchEmails(ctx, account.UID, emails, syncLog)

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
func (s *SyncExecutor) doSyncLegacy(
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
		return s.errorPolicy.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails: %w", err))
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

		if err := s.processor.processEmail(ctx, account.UID, email, syncLog); err != nil {
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
func (s *SyncExecutor) persistSyncProgress(ctx context.Context, accountUID string, cursor string, progress *model.SyncProgress) {
	if progress == nil {
		return
	}

	progressJSON := progress.ToJSON()
	if err := s.accountRepo.UpdateSyncProgress(ctx, accountUID, cursor, progressJSON); err != nil {
		s.logger.Error("持久化同步进度失败: account=%s, err=%v", accountUID, err)
	}
}

// clearSyncCursor 清除同步游标（同步完成时调用）
func (s *SyncExecutor) clearSyncCursor(ctx context.Context, accountUID string) {
	if err := s.accountRepo.UpdateSyncProgress(ctx, accountUID, "", ""); err != nil {
		s.logger.Error("清除同步游标失败: account=%s, err=%v", accountUID, err)
	}
}
