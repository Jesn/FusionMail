package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
)

// EmailService 邮件服务接口
type EmailService interface {
	// 邮件查询
	GetEmailByID(ctx context.Context, id int64) (*model.Email, error)
	GetEmailList(ctx context.Context, filter *repository.EmailFilter, page, pageSize int) (*EmailListResponse, error)
	SearchEmails(ctx context.Context, query string, accountUID string, page, pageSize int) (*EmailListResponse, error)
	GetEmailListWithFilter(ctx context.Context, filter *repository.EmailFilter, offset, limit int) ([]*model.Email, int64, error)
	SearchEmailsWithFilter(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error)

	// 邮件状态管理（本地）
	MarkAsRead(ctx context.Context, ids []int64) error
	MarkAsUnread(ctx context.Context, ids []int64) error
	MarkAllAsRead(ctx context.Context, accountUID *string) (int64, error)
	ToggleStar(ctx context.Context, id int64) error
	ArchiveEmail(ctx context.Context, id int64) error
	DeleteEmail(ctx context.Context, id int64) error
	BatchDeleteEmails(ctx context.Context, ids []int64) (int64, error)
	RestoreEmail(ctx context.Context, id int64) error

	// 物理删除（永久删除）
	PermanentDeleteEmail(ctx context.Context, id int64) error
	BatchPermanentDeleteEmails(ctx context.Context, ids []int64) (int64, error)
	EmptyTrash(ctx context.Context) (int64, error)

	// 统计信息
	GetUnreadCount(ctx context.Context, accountUID string) (int64, error)
	GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error)
	GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error)
}

// EmailListItem 列表项（刻意去掉 text_body 字段）
type EmailListItem struct {
	ID               int64     `json:"id"`
	ProviderID       string    `json:"provider_id"`
	AccountUID       string    `json:"account_uid"`
	MessageID        string    `json:"message_id"`
	ThreadID         string    `json:"thread_id"`
	FromAddress      string    `json:"from_address"`
	FromName         string    `json:"from_name"`
	ToAddresses      string    `json:"to_addresses"`
	CcAddresses      string    `json:"cc_addresses"`
	BccAddresses     string    `json:"bcc_addresses"`
	Subject          string    `json:"subject"`
	Snippet          string    `json:"snippet"`
	IsRead           bool      `json:"is_read"`
	IsStarred        bool      `json:"is_starred"`
	IsArchived       bool      `json:"is_archived"`
	IsDeleted        bool      `json:"is_deleted"`
	HasAttachments   bool      `json:"has_attachments"`
	AttachmentsCount int       `json:"attachments_count"`
	Labels           string    `json:"labels"`
	SentAt           time.Time `json:"sent_at"`
	ReceivedAt       time.Time `json:"received_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EmailListResponse 邮件列表响应
type EmailListResponse struct {
	Emails     []EmailListItem `json:"emails"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// BatchDeleteResult 批量删除结果
type BatchDeleteResult struct {
	DeletedCount int64 `json:"deleted_count"`
}

// AccountEmailStats 账户邮件统计
type AccountEmailStats struct {
	TotalCount    int64 `json:"total_count"`
	UnreadCount   int64 `json:"unread_count"`
	StarredCount  int64 `json:"starred_count"`
	ArchivedCount int64 `json:"archived_count"`
}

// GlobalEmailStats 全局邮件统计
type GlobalEmailStats struct {
	TotalCount    int64 `json:"total_count"`
	UnreadCount   int64 `json:"unread_count"`
	StarredCount  int64 `json:"starred_count"`
	ArchivedCount int64 `json:"archived_count"`
	DeletedCount  int64 `json:"deleted_count"`
	SpamCount     int64 `json:"spam_count"`
}

// emailService 邮件服务实现
// toEmailListItems 将模型列表转换为列表项（不包含 text_body）
func toEmailListItems(emails []*model.Email) []EmailListItem {
	items := make([]EmailListItem, 0, len(emails))
	for _, e := range emails {
		if e == nil {
			continue
		}
		item := EmailListItem{
			ID:               e.ID,
			ProviderID:       e.ProviderID,
			AccountUID:       e.AccountUID,
			MessageID:        e.MessageID,
			ThreadID:         e.ThreadID,
			FromAddress:      e.FromAddress,
			FromName:         e.FromName,
			ToAddresses:      e.ToAddresses,
			CcAddresses:      e.CcAddresses,
			BccAddresses:     e.BccAddresses,
			Subject:          e.Subject,
			Snippet:          e.Snippet,
			IsRead:           e.IsRead,
			IsStarred:        e.IsStarred,
			IsArchived:       e.IsArchived,
			IsDeleted:        e.IsDeleted,
			HasAttachments:   e.HasAttachments,
			AttachmentsCount: e.AttachmentsCount,
			Labels:           e.Labels,
			SentAt:           e.SentAt,
			ReceivedAt:       e.ReceivedAt,
			CreatedAt:        e.CreatedAt,
			UpdatedAt:        e.UpdatedAt,
		}
		items = append(items, item)
	}
	return items
}

type emailService struct {
	emailRepo          repository.EmailRepository
	accountRepo        repository.AccountRepository
	adapterFactory     *adapter.Factory
	credentialResolver *CredentialResolver
	logger             *logger.Logger
}

// NewEmailService 创建邮件服务实例
func NewEmailService(
	emailRepo repository.EmailRepository,
	accountRepo repository.AccountRepository,
	adapterFactory *adapter.Factory,
	encryptor crypto.Encryptor,
) EmailService {
	return NewEmailServiceWithCredentialResolver(
		emailRepo,
		accountRepo,
		adapterFactory,
		NewCredentialResolverWithEncryptor(encryptor, nil),
	)
}

func NewEmailServiceWithCredentialResolver(
	emailRepo repository.EmailRepository,
	accountRepo repository.AccountRepository,
	adapterFactory *adapter.Factory,
	credentialResolver *CredentialResolver,
) EmailService {
	return &emailService{
		emailRepo:          emailRepo,
		accountRepo:        accountRepo,
		adapterFactory:     adapterFactory,
		credentialResolver: credentialResolver,
		logger:             logger.NewWithModule("Email"),
	}
}

// GetEmailByID 根据 ID 获取邮件详情
func (s *emailService) GetEmailByID(ctx context.Context, id int64) (*model.Email, error) {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return nil, dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 已删除邮件也允许查看详情（便于在详情中执行恢复操作）

	// 若检测到正文未解析或疑似原始 MIME 文本，尝试即时修复
	if email.HTMLBody == "" || looksLikeRawMIME(email.TextBody) {
		if updated, rerr := s.tryRepairEmailBody(ctx, email); rerr != nil {
			s.logger.Debug("修复邮件内容失败: id=%d, err=%v", id, rerr)
		} else if updated {
			s.logger.Debug("邮件内容已修复: id=%d", id)
		}
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

	// 处理分组筛选：将 GroupID 转换为 AccountUIDs
	if filter.GroupID != nil {
		groupID := *filter.GroupID
		if groupID == -1 {
			// -1: 所有账号，不需要额外筛选
			filter.GroupID = nil
		} else if groupID == 0 {
			// 0: 未分组账号
			accounts, err := s.accountRepo.FindUngrouped(ctx)
			if err != nil {
				s.logger.Error("获取未分组账号失败: %v", err)
				return nil, fmt.Errorf("failed to get ungrouped accounts: %w", err)
			}
			if len(accounts) == 0 {
				// 没有未分组账号，返回空列表
				return &EmailListResponse{
					Emails:     []EmailListItem{},
					Total:      0,
					Page:       page,
					PageSize:   pageSize,
					TotalPages: 0,
				}, nil
			}
			// 提取账号 UID 列表
			accountUIDs := make([]string, len(accounts))
			for i, acc := range accounts {
				accountUIDs[i] = acc.UID
			}
			filter.AccountUIDs = accountUIDs
			filter.GroupID = nil
		} else {
			// >0: 具体分组 ID
			// 使用 FindAllByGroupID 获取分组下的所有账号（包括子账户）
			accounts, err := s.accountRepo.FindAllByGroupID(ctx, groupID)
			if err != nil {
				s.logger.Error("获取分组账号失败: groupID=%d, error=%v", groupID, err)
				return nil, fmt.Errorf("failed to get group accounts: %w", err)
			}
			if len(accounts) == 0 {
				// 分组中没有账号，返回空列表
				return &EmailListResponse{
					Emails:     []EmailListItem{},
					Total:      0,
					Page:       page,
					PageSize:   pageSize,
					TotalPages: 0,
				}, nil
			}
			// 提取账号 UID 列表
			accountUIDs := make([]string, len(accounts))
			for i, acc := range accounts {
				accountUIDs[i] = acc.UID
			}
			filter.AccountUIDs = accountUIDs
			filter.GroupID = nil
		}
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
		Emails:     toEmailListItems(emails),
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
		Emails:     toEmailListItems(emails),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetEmailListWithFilter 获取邮件列表（使用 offset/limit，用于 API）
func (s *emailService) GetEmailListWithFilter(ctx context.Context, filter *repository.EmailFilter, offset, limit int) ([]*model.Email, int64, error) {
	// 参数验证
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 查询邮件列表
	emails, total, err := s.emailRepo.List(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get email list: %w", err)
	}

	return emails, total, nil
}

// SearchEmailsWithFilter 搜索邮件（使用 offset/limit，用于 API）
func (s *emailService) SearchEmailsWithFilter(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error) {
	// 参数验证
	if query == "" {
		return nil, 0, fmt.Errorf("search query is required")
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 搜索邮件
	emails, total, err := s.emailRepo.Search(ctx, query, accountUID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search emails: %w", err)
	}

	return emails, total, nil
}

// MarkAsRead 标记邮件为已读
func (s *emailService) MarkAsRead(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "邮件 ID 列表不能为空")
	}
	if err := s.emailRepo.MarkAsRead(ctx, ids); err != nil {
		s.logger.Error("标记已读失败: count=%d, error=%v", len(ids), err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// MarkAsUnread 标记邮件为未读
func (s *emailService) MarkAsUnread(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "邮件 ID 列表不能为空")
	}
	if err := s.emailRepo.MarkAsUnread(ctx, ids); err != nil {
		s.logger.Error("标记未读失败: count=%d, error=%v", len(ids), err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// MarkAllAsRead 批量标记所有未读邮件为已读
func (s *emailService) MarkAllAsRead(ctx context.Context, accountUID *string) (int64, error) {
	// 如果指定了账号，验证账号是否存在
	if accountUID != nil && *accountUID != "" {
		account, err := s.accountRepo.FindByUID(ctx, *accountUID)
		if err != nil {
			s.logger.Error("查询账户失败: uid=%s, error=%v", *accountUID, err)
			return 0, fmt.Errorf("database error: %w", err)
		}
		if account == nil {
			return 0, dto.NewAPIError(dto.ErrAccountNotFound)
		}
	}

	// 批量更新
	count, err := s.emailRepo.MarkAllAsRead(ctx, accountUID)
	if err != nil {
		s.logger.Error("批量标记已读失败: error=%v", err)
		return 0, fmt.Errorf("database error: %w", err)
	}

	return count, nil
}

// ToggleStar 切换星标状态
func (s *emailService) ToggleStar(ctx context.Context, id int64) error {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 切换星标状态
	newStarred := !email.IsStarred
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, &newStarred, nil, nil); err != nil {
		s.logger.Error("切换星标失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// ArchiveEmail 归档邮件
func (s *emailService) ArchiveEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	archived := true
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, &archived, nil); err != nil {
		s.logger.Error("归档邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// GetUnreadCount 获取未读邮件数
func (s *emailService) GetUnreadCount(ctx context.Context, accountUID string) (int64, error) {
	return s.emailRepo.CountUnread(ctx, accountUID)
}

// GetAccountStats 获取账户邮件统计信息
// 优化：使用单条 SQL 聚合查询，减少数据库往返次数
func (s *emailService) GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error) {
	repoStats, err := s.emailRepo.GetAccountStats(ctx, accountUID)
	if err != nil {
		s.logger.Error("获取账户统计失败: uid=%s, error=%v", accountUID, err)
		return nil, fmt.Errorf("failed to get account stats: %w", err)
	}

	// 转换为 service 层的类型
	return &AccountEmailStats{
		TotalCount:    repoStats.TotalCount,
		UnreadCount:   repoStats.UnreadCount,
		StarredCount:  repoStats.StarredCount,
		ArchivedCount: repoStats.ArchivedCount,
	}, nil
}

// GetGlobalStats 获取全局邮件统计信息
// 优化：使用单条 SQL 聚合查询，减少数据库往返次数
func (s *emailService) GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error) {
	repoStats, err := s.emailRepo.GetGlobalStats(ctx)
	if err != nil {
		s.logger.Error("获取全局统计失败: %v", err)
		return nil, fmt.Errorf("failed to get global stats: %w", err)
	}

	// 转换为 service 层的类型
	return &GlobalEmailStats{
		TotalCount:    repoStats.TotalCount,
		UnreadCount:   repoStats.UnreadCount,
		StarredCount:  repoStats.StarredCount,
		ArchivedCount: repoStats.ArchivedCount,
		DeletedCount:  repoStats.DeletedCount,
		SpamCount:     repoStats.SpamCount,
	}, nil
}
