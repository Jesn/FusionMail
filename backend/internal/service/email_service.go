package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"time"
)

// EmailService 邮件服务接口
type EmailService interface {
	// 邮件查询
	GetEmailByID(ctx context.Context, id int64) (*model.Email, error)
	GetEmailList(ctx context.Context, filter *repository.EmailFilter, page, pageSize int) (*EmailListResponse, error)
	SearchEmails(ctx context.Context, query string, accountUID string, page, pageSize int) (*EmailListResponse, error)
	
	// 邮件状态管理（本地）
	MarkAsRead(ctx context.Context, ids []int64) error
	MarkAsUnread(ctx context.Context, ids []int64) error
	ToggleStar(ctx context.Context, id int64) error
	ArchiveEmail(ctx context.Context, id int64) error
	DeleteEmail(ctx context.Context, id int64) error
	
	// 统计信息
	GetUnreadCount(ctx context.Context, accountUID string) (int64, error)
	GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error)
}

// EmailListResponse 邮件列表响应
type EmailListResponse struct {
	Emails     []*model.Email `json:"emails"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// AccountEmailStats 账户邮件统计
type AccountEmailStats struct {
	TotalCount    int64 `json:"total_count"`
	UnreadCount   int64 `json:"unread_count"`
	StarredCount  int64 `json:"starred_count"`
	ArchivedCount int64 `json:"archived_count"`
}

// emailService 邮件服务实现
type emailService struct {
	emailRepo   repository.EmailRepository
	accountRepo repository.AccountRepository
}

// NewEmailService 创建邮件服务实例
func NewEmailService(emailRepo repository.EmailRepository, accountRepo repository.AccountRepository) EmailService {
	return &emailService{
		emailRepo:   emailRepo,
		accountRepo: accountRepo,
	}
}

// GetEmailByID 根据 ID 获取邮件详情
func (s *emailService) GetEmailByID(ctx context.Context, id int64) (*model.Email, error) {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}
	if email == nil {
		return nil, fmt.Errorf("email not found")
	}
	return email, nil
}

// GetEmailList 获取邮件列表（支持分页和筛选）
func (s *emailService) GetEmailList(ctx context.Context, filter *repository.EmailFilter, page, pageSize int) (*EmailListResponse, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询邮件列表
	emails, total, err := s.emailRepo.List(ctx, filter, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get email list: %w", err)
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &EmailListResponse{
		Emails:     emails,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// SearchEmails 全文搜索邮件
func (s *emailService) SearchEmails(ctx context.Context, query string, accountUID string, page, pageSize int) (*EmailListResponse, error) {
	// 参数验证
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 搜索邮件
	emails, total, err := s.emailRepo.Search(ctx, query, accountUID, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &EmailListResponse{
		Emails:     emails,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// MarkAsRead 标记邮件为已读
func (s *emailService) MarkAsRead(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return s.emailRepo.MarkAsRead(ctx, ids)
}

// MarkAsUnread 标记邮件为未读
func (s *emailService) MarkAsUnread(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return s.emailRepo.MarkAsUnread(ctx, ids)
}

// ToggleStar 切换星标状态
func (s *emailService) ToggleStar(ctx context.Context, id int64) error {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}
	if email == nil {
		return fmt.Errorf("email not found")
	}

	// 切换星标状态
	newStarred := !email.IsStarred
	return s.emailRepo.UpdateLocalStatus(ctx, id, nil, &newStarred, nil, nil)
}

// ArchiveEmail 归档邮件
func (s *emailService) ArchiveEmail(ctx context.Context, id int64) error {
	archived := true
	return s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, &archived, nil)
}

// DeleteEmail 删除邮件（软删除）
func (s *emailService) DeleteEmail(ctx context.Context, id int64) error {
	deleted := true
	return s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, nil, &deleted)
}

// GetUnreadCount 获取未读邮件数
func (s *emailService) GetUnreadCount(ctx context.Context, accountUID string) (int64, error) {
	return s.emailRepo.CountUnread(ctx, accountUID)
}

// GetAccountStats 获取账户邮件统计信息
func (s *emailService) GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error) {
	stats := &AccountEmailStats{}

	// 统计总数
	filter := &repository.EmailFilter{
		AccountUID: accountUID,
	}
	falseVal := false
	filter.IsDeleted = &falseVal
	
	_, total, err := s.emailRepo.List(ctx, filter, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to count total emails: %w", err)
	}
	stats.TotalCount = total

	// 统计未读数
	unreadCount, err := s.emailRepo.CountUnread(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("failed to count unread emails: %w", err)
	}
	stats.UnreadCount = unreadCount

	// 统计星标数
	trueVal := true
	starredFilter := &repository.EmailFilter{
		AccountUID: accountUID,
		IsStarred:  &trueVal,
		IsDeleted:  &falseVal,
	}
	_, starredCount, err := s.emailRepo.List(ctx, starredFilter, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to count starred emails: %w", err)
	}
	stats.StarredCount = starredCount

	// 统计归档数
	archivedFilter := &repository.EmailFilter{
		AccountUID: accountUID,
		IsArchived: &trueVal,
		IsDeleted:  &falseVal,
	}
	_, archivedCount, err := s.emailRepo.List(ctx, archivedFilter, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to count archived emails: %w", err)
	}
	stats.ArchivedCount = archivedCount

	return stats, nil
}
