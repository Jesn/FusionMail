package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
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
	RestoreEmail(ctx context.Context, id int64) error

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
	emailRepo      repository.EmailRepository
	accountRepo    repository.AccountRepository
	adapterFactory *adapter.Factory
	encryptor      crypto.Encryptor
}

// NewEmailService 创建邮件服务实例
func NewEmailService(
	emailRepo repository.EmailRepository,
	accountRepo repository.AccountRepository,
	adapterFactory *adapter.Factory,
	encryptor crypto.Encryptor,
) EmailService {
	return &emailService{
		emailRepo:      emailRepo,
		accountRepo:    accountRepo,
		adapterFactory: adapterFactory,
		encryptor:      encryptor,
	}
}

// GetEmailByID 根据 ID 获取邮件详情
func (s *emailService) GetEmailByID(ctx context.Context, id int64) (*model.Email, error) {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("database error when finding email: id=%d, error=%v", id, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return nil, dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 已删除邮件也允许查看详情（便于在详情中执行恢复操作）
	// if email.IsDeleted {
	// 	return nil, dto.NewAPIError(dto.ErrEmailNotFound)
	// }

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
		log.Printf("failed to mark emails as read: ids=%v, error=%v", ids, err)
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
		log.Printf("failed to mark emails as unread: ids=%v, error=%v", ids, err)
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
			log.Printf("database error when finding account: uid=%s, error=%v", *accountUID, err)
			return 0, fmt.Errorf("database error: %w", err)
		}
		if account == nil {
			return 0, dto.NewAPIError(dto.ErrAccountNotFound)
		}
	}

	// 批量更新
	count, err := s.emailRepo.MarkAllAsRead(ctx, accountUID)
	if err != nil {
		log.Printf("failed to mark all as read: error=%v", err)
		return 0, fmt.Errorf("database error: %w", err)
	}

	return count, nil
}

// ToggleStar 切换星标状态
func (s *emailService) ToggleStar(ctx context.Context, id int64) error {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("database error when finding email: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 切换星标状态
	newStarred := !email.IsStarred
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, &newStarred, nil, nil); err != nil {
		log.Printf("failed to toggle star: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// ArchiveEmail 归档邮件
func (s *emailService) ArchiveEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("database error when finding email: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	archived := true
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, &archived, nil); err != nil {
		log.Printf("failed to archive email: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// DeleteEmail 删除邮件（软删除）
func (s *emailService) DeleteEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("database error when finding email: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 本地删除
	deleted := true
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, nil, &deleted); err != nil {
		log.Printf("failed to delete email: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}

	// 后台执行服务器软删除
	go s.tryServerSoftDelete(context.Background(), email)

	return nil
}

// RestoreEmail 恢复已删除邮件（从垃圾箱恢复到收件箱）
func (s *emailService) RestoreEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("database error when finding email: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 恢复：取消删除，同时取消归档，确保回到收件箱
	deleted := false
	archived := false
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, &archived, &deleted); err != nil {
		log.Printf("failed to restore email: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// tryServerSoftDelete 尝试在服务器上软删除邮件
func (s *emailService) tryServerSoftDelete(ctx context.Context, email *model.Email) {
	// 获取账号信息
	account, err := s.accountRepo.FindByUID(ctx, email.AccountUID)
	if err != nil || account == nil {
		log.Printf("failed to get account for email %d: %v", email.ID, err)
		return
	}

	// 检查是否启用服务器软删除
	if account.ServerDeletePolicy != "soft" {
		return
	}

	// 解密凭证
	decryptedData, err := s.encryptor.Decrypt(account.EncryptedCredentials)
	if err != nil {
		log.Printf("failed to decrypt credentials for account %s: %v", account.UID, err)
		return
	}

	// 创建凭证对象
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: account.AuthType,
	}

	// 根据认证类型解析凭证
	if account.AuthType == "oauth2" {
		// OAuth2 凭证是 JSON 格式
		var oauthCreds struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenExpiry  string `json:"token_expiry"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}

		if err := json.Unmarshal([]byte(decryptedData), &oauthCreds); err != nil {
			log.Printf("failed to parse oauth2 credentials: %v", err)
			return
		}

		credentials.AccessToken = oauthCreds.AccessToken
		credentials.RefreshToken = oauthCreds.RefreshToken
		credentials.ClientID = oauthCreds.ClientID
		credentials.ClientSecret = oauthCreds.ClientSecret

		if oauthCreds.TokenExpiry != "" {
			if expiry, err := time.Parse(time.RFC3339, oauthCreds.TokenExpiry); err == nil {
				credentials.TokenExpiry = expiry
			}
		}
	} else if account.AuthType == "quick" {
		// 短效认证凭证是 JSON 格式
		var quickCreds struct {
			RefreshToken string `json:"refresh_token"`
			ClientID     string `json:"client_id"`
		}

		if err := json.Unmarshal([]byte(decryptedData), &quickCreds); err != nil {
			log.Printf("failed to parse quick credentials: %v", err)
			return
		}

		credentials.RefreshToken = quickCreds.RefreshToken
		credentials.ClientID = quickCreds.ClientID
	} else {
		// 密码认证
		credentials.Password = decryptedData
	}

	// 创建适配器
	mailAdapter, err := s.adapterFactory.CreateProviderFromAccount(
		account.Provider,
		account.Protocol,
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		log.Printf("failed to create adapter for account %s: %v", account.UID, err)
		return
	}

	// 连接到邮箱服务器
	if err := mailAdapter.Connect(ctx); err != nil {
		log.Printf("failed to connect to mail server for account %s: %v", account.UID, err)
		return
	}
	defer mailAdapter.Disconnect()

	// 检查是否支持软删除
	softDeleter, ok := mailAdapter.(adapter.SoftDeleter)
	if !ok {
		log.Printf("adapter for account %s does not support soft delete", account.UID)
		return
	}

	// 执行软删除（带重试）
	if err := s.softDeleteWithRetry(ctx, softDeleter, email.ProviderID); err != nil {
		log.Printf("failed to soft delete email %d on server: %v", email.ID, err)
		// 不中断流程，仅记录日志
	}
}

// softDeleteWithRetry 带重试的软删除
func (s *emailService) softDeleteWithRetry(ctx context.Context, deleter adapter.SoftDeleter, providerID string) error {
	maxRetries := 3
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := deleter.MoveToTrash(ctx, providerID)
		if err == nil {
			return nil
		}

		// 404 视为成功
		if strings.Contains(err.Error(), "404") {
			return nil
		}

		if attempt < maxRetries {
			select {
			case <-time.After(backoff):
				backoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("max retries exceeded")
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
