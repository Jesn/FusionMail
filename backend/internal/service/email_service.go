package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	emailRepo      repository.EmailRepository
	accountRepo    repository.AccountRepository
	adapterFactory *adapter.Factory
	encryptor      crypto.Encryptor
	logger         *logger.Logger
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
		logger:         logger.NewWithModule("Email"),
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
			accounts, err := s.accountRepo.FindByGroupID(ctx, groupID)
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

// DeleteEmail 删除邮件（软删除）
func (s *emailService) DeleteEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 本地删除
	deleted := true
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, nil, &deleted); err != nil {
		s.logger.Error("删除邮件失败: id=%d, error=%v", id, err)
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
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 恢复：取消删除，同时取消归档，确保回到收件箱
	deleted := false
	archived := false
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, &archived, &deleted); err != nil {
		s.logger.Error("恢复邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// tryServerSoftDelete 尝试在服务器上软删除邮件
func (s *emailService) tryServerSoftDelete(ctx context.Context, email *model.Email) {
	// 获取账号信息
	account, err := s.accountRepo.FindByUID(ctx, email.AccountUID)
	if err != nil || account == nil {
		s.logger.Debug("获取账户失败: email_id=%d, error=%v", email.ID, err)
		return
	}

	// 检查是否启用服务器软删除
	if account.ServerDeletePolicy != "soft" {
		return
	}

	// 解密凭证
	decryptedData, err := s.encryptor.Decrypt(account.EncryptedCredentials)
	if err != nil {
		s.logger.Debug("解密凭证失败: account=%s, error=%v", account.UID, err)
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
			s.logger.Debug("解析OAuth2凭证失败: %v", err)
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
			s.logger.Debug("解析快速认证凭证失败: %v", err)
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
		s.logger.Debug("创建适配器失败: account=%s, error=%v", account.UID, err)
		return
	}

	// 连接到邮箱服务器
	if err := mailAdapter.Connect(ctx); err != nil {
		s.logger.Debug("连接邮箱服务器失败: account=%s, error=%v", account.UID, err)
		return
	}
	defer mailAdapter.Disconnect()

	// 检查是否支持软删除
	softDeleter, ok := mailAdapter.(adapter.SoftDeleter)
	if !ok {
		s.logger.Debug("适配器不支持软删除: account=%s", account.UID)
		return
	}

	// 执行软删除（带重试）
	if err := s.softDeleteWithRetry(ctx, softDeleter, email.ProviderID); err != nil {
		s.logger.Debug("服务器软删除失败: email_id=%d, error=%v", email.ID, err)
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

// GetGlobalStats 获取全局邮件统计信息
func (s *emailService) GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error) {
	stats := &GlobalEmailStats{}

	falseVal := false
	trueVal := true

	// 总数（不含已删除）
	total, err := s.emailRepo.Count(ctx, &repository.EmailFilter{IsDeleted: &falseVal})
	if err != nil {
		return nil, fmt.Errorf("failed to count total emails: %w", err)
	}
	stats.TotalCount = total

	// 未读数（全账户）
	unreadCount, err := s.emailRepo.CountUnread(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to count unread emails: %w", err)
	}
	stats.UnreadCount = unreadCount

	// 星标数（不含已删除）
	starred, err := s.emailRepo.Count(ctx, &repository.EmailFilter{IsStarred: &trueVal, IsDeleted: &falseVal})
	if err != nil {
		return nil, fmt.Errorf("failed to count starred emails: %w", err)
	}
	stats.StarredCount = starred

	// 归档数（不含已删除）
	archived, err := s.emailRepo.Count(ctx, &repository.EmailFilter{IsArchived: &trueVal, IsDeleted: &falseVal})
	if err != nil {
		return nil, fmt.Errorf("failed to count archived emails: %w", err)
	}
	stats.ArchivedCount = archived

	// 已删除数
	deletedTrue := true
	deleted, err := s.emailRepo.Count(ctx, &repository.EmailFilter{IsDeleted: &deletedTrue})
	if err != nil {
		return nil, fmt.Errorf("failed to count deleted emails: %w", err)
	}
	stats.DeletedCount = deleted

	// 垃圾邮件数（不含已删除）
	spam, err := s.emailRepo.Count(ctx, &repository.EmailFilter{IsSpam: &trueVal, IsDeleted: &falseVal})
	if err != nil {
		return nil, fmt.Errorf("failed to count spam emails: %w", err)
	}
	stats.SpamCount = spam

	return stats, nil
}

// looksLikeRawMIME 粗略判断一段文本是否像原始 MIME 源文
func looksLikeRawMIME(s string) bool {
	if s == "" {
		return false
	}
	ls := strings.ToLower(s)
	if strings.Contains(ls, "content-transfer-encoding:") && strings.Contains(ls, "content-type:") {
		return true
	}
	if strings.Contains(ls, "mime-version:") && strings.Contains(ls, "content-type:") {
		return true
	}
	return false
}

// tryRepairEmailBody 即时从远端拉取并解析邮件详情，更新本地存储
func (s *emailService) tryRepairEmailBody(ctx context.Context, email *model.Email) (bool, error) {
	// 获取账号
	account, err := s.accountRepo.FindByUID(ctx, email.AccountUID)
	if err != nil || account == nil {
		return false, err
	}

	// 解密凭证
	decryptedData, err := s.encryptor.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return false, err
	}

	// 组装凭证
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: account.AuthType,
	}

	if account.AuthType == "oauth2" {
		var oauthCreds struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenExpiry  string `json:"token_expiry"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		if err := json.Unmarshal([]byte(decryptedData), &oauthCreds); err != nil {
			return false, err
		}
		credentials.AccessToken = oauthCreds.AccessToken
		credentials.RefreshToken = oauthCreds.RefreshToken
		credentials.ClientID = oauthCreds.ClientID
		credentials.ClientSecret = oauthCreds.ClientSecret
		if oauthCreds.TokenExpiry != "" {
			if expiry, e := time.Parse(time.RFC3339, oauthCreds.TokenExpiry); e == nil {
				credentials.TokenExpiry = expiry
			}
		}
	} else if account.AuthType == "quick" {
		var quickCreds struct {
			RefreshToken string `json:"refresh_token"`
			ClientID     string `json:"client_id"`
		}
		if err := json.Unmarshal([]byte(decryptedData), &quickCreds); err != nil {
			return false, err
		}
		credentials.RefreshToken = quickCreds.RefreshToken
		credentials.ClientID = quickCreds.ClientID
	} else {
		credentials.Password = decryptedData
	}

	// 创建适配器并连接
	mailAdapter, err := s.adapterFactory.CreateProviderFromAccount(
		account.Provider,
		account.Protocol,
		credentials,
		nil,
	)
	if err != nil {
		return false, err
	}
	if err := mailAdapter.Connect(ctx); err != nil {
		return false, err
	}
	defer mailAdapter.Disconnect()

	// 拉取详情
	detail, err := mailAdapter.FetchEmailDetail(ctx, email.ProviderID)
	if err != nil || detail == nil {
		return false, err
	}

	changed := false
	if detail.HTMLBody != "" && detail.HTMLBody != email.HTMLBody {
		email.HTMLBody = detail.HTMLBody
		changed = true
	}
	if detail.TextBody != "" && detail.TextBody != email.TextBody {
		email.TextBody = detail.TextBody
		changed = true
	}
	if detail.Snippet != "" && detail.Snippet != email.Snippet {
		email.Snippet = detail.Snippet
		changed = true
	}
	if detail.HasAttachments != email.HasAttachments || detail.AttachmentsCount != email.AttachmentsCount {
		email.HasAttachments = detail.HasAttachments
		email.AttachmentsCount = detail.AttachmentsCount
		changed = true
	}

	if !changed {
		return false, nil
	}
	if err := s.emailRepo.Update(ctx, email); err != nil {
		return false, err
	}
	return true, nil
}

// PermanentDeleteEmail 永久删除单封邮件（物理删除）
func (s *emailService) PermanentDeleteEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 只允许删除已在回收站中的邮件
	if !email.IsDeleted {
		return dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "只能永久删除回收站中的邮件")
	}

	// 物理删除邮件
	if err := s.emailRepo.Delete(ctx, id); err != nil {
		s.logger.Error("永久删除邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}

	return nil
}

// BatchPermanentDeleteEmails 批量永久删除邮件（物理删除）
func (s *emailService) BatchPermanentDeleteEmails(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	deletedCount := int64(0)
	for _, id := range ids {
		// 验证邮件是否存在且在回收站中
		email, err := s.emailRepo.FindByID(ctx, id)
		if err != nil {
			s.logger.Debug("查询邮件失败: id=%d, error=%v", id, err)
			continue
		}
		if email == nil {
			continue
		}
		if !email.IsDeleted {
			continue // 跳过不在回收站中的邮件
		}

		// 物理删除
		if err := s.emailRepo.Delete(ctx, id); err != nil {
			s.logger.Debug("永久删除邮件失败: id=%d, error=%v", id, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// EmptyTrash 清空回收站（永久删除所有已删除邮件）
func (s *emailService) EmptyTrash(ctx context.Context) (int64, error) {
	// 获取所有已删除的邮件
	trueVal := true
	filter := &repository.EmailFilter{
		IsDeleted: &trueVal,
	}

	// 分批获取并删除
	deletedCount := int64(0)
	batchSize := 100
	offset := 0

	for {
		emails, _, err := s.emailRepo.List(ctx, filter, offset, batchSize)
		if err != nil {
			return deletedCount, fmt.Errorf("failed to list deleted emails: %w", err)
		}

		if len(emails) == 0 {
			break
		}

		for _, email := range emails {
			if err := s.emailRepo.Delete(ctx, email.ID); err != nil {
				s.logger.Debug("清空回收站时删除邮件失败: id=%d, error=%v", email.ID, err)
				continue
			}
			deletedCount++
		}

		// 由于删除后数据减少，不需要增加 offset
		if len(emails) < batchSize {
			break
		}
	}

	return deletedCount, nil
}
