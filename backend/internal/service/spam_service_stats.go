package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// GetSpamEmails 获取垃圾邮件列表
func (s *spamService) GetSpamEmails(ctx context.Context, accountUID string, page, pageSize int) ([]*model.Email, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建过滤条件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 查询垃圾邮件列表
	emails, total, err := s.emailRepo.List(ctx, filter, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get spam emails: %w", err)
	}

	return emails, total, nil
}

// GetSpamStats 获取垃圾邮件统计
func (s *spamService) GetSpamStats(ctx context.Context, accountUID string) (*SpamStats, error) {
	stats := &SpamStats{}

	// 构建基础过滤条件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 统计总数
	total, err := s.emailRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count spam emails: %w", err)
	}
	stats.TotalCount = total

	// 统计未读数
	isRead := false
	unreadFilter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
		IsRead:    &isRead,
	}
	if accountUID != "" {
		unreadFilter.AccountUID = accountUID
	}
	unreadCount, err := s.emailRepo.Count(ctx, unreadFilter)
	if err == nil {
		stats.UnreadCount = unreadCount
	}

	// 其他统计暂时设为 0，后续实现
	stats.TodayCount = 0
	stats.WeekCount = 0
	stats.MonthCount = 0
	stats.BlockedCount = 0

	return stats, nil
}
