package service

// SyncExecutor 的 UID 增量同步路径（IMAP 专用）：
// doSyncWithUID、updateUIDSyncState。
// 自 sync_executor.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"fmt"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// doSyncWithUID 使用 UID 增量同步模式（IMAP 专用）
// Requirements: 1.1, 1.2, 1.3 - 基于 UID 的增量同步
func (s *SyncExecutor) doSyncWithUID(
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
		return s.errorPolicy.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails since UID: %w", err))
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
		result, err := s.processor.processEmailWithDedupe(ctx, account.UID, email, syncLog)
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
func (s *SyncExecutor) updateUIDSyncState(ctx context.Context, accountUID string, uidValidity uint32, lastUID uint32) error {
	return s.accountRepo.UpdateUIDSyncState(ctx, accountUID, int64(uidValidity), int64(lastUID))
}
