package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GetSyncStatus 获取同步状态
func (s *SystemService) GetSyncStatus(ctx context.Context) ([]SyncStatusResponse, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取账户列表失败: %w", err)
	}

	var statusList []SyncStatusResponse
	for _, account := range accounts {
		status := SyncStatusResponse{
			AccountUID:   account.UID,
			AccountName:  account.Email, // 使用邮箱地址作为账户名称
			Provider:     account.GetProviderName(),
			SyncInterval: account.SyncInterval,
		}

		// 获取最后同步记录
		lastSync, err := s.syncLogRepo.GetLatestByAccount(ctx, account.UID)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Error("获取账户同步记录失败", "account_uid", account.UID, "error", err)
			status.Status = "unknown"
			status.ErrorMessage = "获取同步状态失败"
		} else if lastSync != nil {
			status.LastSyncTime = &lastSync.StartedAt
			if lastSync.Status == "success" {
				status.Status = "idle"
			} else {
				status.Status = "failed"
				status.ErrorMessage = lastSync.ErrorMessage
			}
		} else {
			status.Status = "idle"
		}

		// 计算下次同步时间
		if status.LastSyncTime != nil && account.SyncEnabled {
			nextSync := status.LastSyncTime.Add(time.Duration(account.SyncInterval) * time.Minute)
			status.NextSyncTime = &nextSync
		}

		// 获取邮件统计
		status.EmailCount, _ = s.emailRepo.CountByAccount(ctx, account.UID)
		status.UnreadCount, _ = s.emailRepo.CountUnread(ctx, account.UID)

		statusList = append(statusList, status)
	}

	return statusList, nil
}

// GetSyncLogs 获取同步日志
func (s *SystemService) GetSyncLogs(ctx context.Context, page, pageSize int, accountUID, status string) ([]SyncLogItem, int64, error) {
	offset := (page - 1) * pageSize

	logs, total, err := s.syncLogRepo.FindWithPagination(ctx, offset, pageSize, accountUID, status)
	if err != nil {
		return nil, 0, fmt.Errorf("获取同步日志失败: %w", err)
	}

	var logItems []SyncLogItem
	for _, log := range logs {
		item := SyncLogItem{
			ID:           log.ID,
			AccountUID:   log.AccountUID,
			Status:       log.Status,
			StartTime:    log.StartedAt,
			EndTime:      log.CompletedAt,
			EmailsAdded:  log.EmailsNew,
			EmailsTotal:  log.EmailsFetched,
			ErrorMessage: log.ErrorMessage,
		}

		// 计算持续时间
		if log.CompletedAt != nil {
			item.Duration = log.CompletedAt.Sub(log.StartedAt).Milliseconds()
		} else {
			item.Duration = int64(log.DurationMs)
		}

		// 获取账户信息
		if account, err := s.accountRepo.FindByUID(ctx, log.AccountUID); err == nil && account != nil {
			item.AccountName = account.Email // 使用邮箱地址作为账户名称
			item.Provider = account.GetProviderName()
		} else {
			// 账户已删除，使用占位符
			item.AccountName = "已删除的账户"
			item.Provider = "unknown"
		}

		logItems = append(logItems, item)
	}

	return logItems, total, nil
}
