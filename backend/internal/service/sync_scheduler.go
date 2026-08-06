package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fusionmail/internal/model"
)

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
		// 启动时自愈历史脏数据：webhook 子账户被误标为可轮询
		if n, err := s.accountRepo.HealWebhookChildPollingFlags(ctx); err != nil {
			syncLog.Warn("自愈 webhook 子账户同步标志失败: %v", err)
		} else if n > 0 {
			syncLog.Info("已自愈 webhook/子账户轮询标志: affected=%d", n)
		}

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
// Webhook 模式 / webhook 子账户 / 父账户下的子邮箱 均跳过轮询
func (s *syncService) shouldSync(account *model.EmailAccount, now time.Time) bool {
	if account.ShouldSkipPollingSync() {
		s.logger.Debug("跳过轮询同步: account=%s, mode=%s, child=%v",
			account.UID, account.GetSyncMode(), account.IsChildAccount())
		return false
	}

	// 首次同步（从未同步过）
	if account.LastSyncAt == nil {
		return true
	}

	// 计算下次同步时间
	syncInterval := time.Duration(account.SyncInterval) * time.Minute
	if account.SyncInterval <= 0 {
		syncInterval = 2 * time.Minute
	}
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
