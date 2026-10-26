package service

import (
	"context"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
)

// EmailQueryParams 邮件查询业务参数（handler 层使用，不直接依赖 repository.EmailFilter）
type EmailQueryParams struct {
	AccountUID  string
	AccountUIDs []string // 多账号筛选（用于分组筛选）
	GroupID     *int64   // 分组 ID：nil 表示不过滤，0 表示未分组，>0 表示具体分组
	IsRead      *bool
	IsStarred   *bool
	IsArchived  *bool
	IsDeleted   *bool
	IsSpam      *bool
	FromAddress string
	Subject     string
	StartDate   string
	EndDate     string
	SearchQuery string
}

// toFilter 将业务查询参数转换为 repository 层过滤条件
func (p *EmailQueryParams) toFilter() *repository.EmailFilter {
	if p == nil {
		return nil
	}
	return &repository.EmailFilter{
		AccountUID:  p.AccountUID,
		AccountUIDs: p.AccountUIDs,
		GroupID:     p.GroupID,
		IsRead:      p.IsRead,
		IsStarred:   p.IsStarred,
		IsArchived:  p.IsArchived,
		IsDeleted:   p.IsDeleted,
		IsSpam:      p.IsSpam,
		FromAddress: p.FromAddress,
		Subject:     p.Subject,
		StartDate:   p.StartDate,
		EndDate:     p.EndDate,
		SearchQuery: p.SearchQuery,
	}
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

func toEmailDetailResponse(e *model.Email) *response.EmailDetailResponse {
	return &response.EmailDetailResponse{
		ID:               e.ID,
		ProviderID:       e.ProviderID,
		AccountUID:       e.AccountUID,
		MessageID:        e.MessageID,
		Subject:          e.Subject,
		FromAddress:      e.FromAddress,
		FromName:         e.FromName,
		ToAddress:        e.ToAddress,
		ToAddresses:      e.ToAddresses,
		CcAddresses:      e.CcAddresses,
		BccAddresses:     e.BccAddresses,
		ReplyTo:          e.ReplyTo,
		TextBody:         e.TextBody,
		HTMLBody:         e.HTMLBody,
		Snippet:          e.Snippet,
		IsRead:           e.IsRead,
		IsStarred:        e.IsStarred,
		IsArchived:       e.IsArchived,
		IsDeleted:        e.IsDeleted,
		Labels:           e.Labels,
		LocalLabels:      e.LocalLabels,
		Folder:           e.Folder,
		SourceIsRead:     e.SourceIsRead,
		SourceLabels:     e.SourceLabels,
		SourceFolder:     e.SourceFolder,
		HasAttachment:    e.HasAttachment,
		HasAttachments:   e.HasAttachments,
		AttachmentsCount: e.AttachmentsCount,
		SentAt:           e.SentAt,
		ReceivedAt:       e.ReceivedAt,
		SizeBytes:        e.SizeBytes,
		ThreadID:         e.ThreadID,
		InReplyTo:        e.InReplyTo,
		References:       e.References,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
		IsSpam:           e.IsSpam,
		SpamScore:        e.SpamScore,
		SpamConfidence:   e.SpamConfidence,
		SpamReason:       e.SpamReason,
		SpamDetectedAt:   e.SpamDetectedAt,
		SpamDetectedBy:   e.SpamDetectedBy,
		UserMarkedSpam:   e.UserMarkedSpam,
		UserMarkedAt:     e.UserMarkedAt,
	}
}

// EmailQueryServiceInterface 邮件读取与统计能力。
//
// 为什么拆成三个接口而不是一个大接口：消费方只应依赖自己真正用到的能力。
// 例如 public_handler 只需要查询与删除，不需要归档/星标；
// 接口越窄，mock 越小，越容易看出某个 handler 是不是管得太宽。
type EmailQueryServiceInterface interface {
	GetEmailByID(ctx context.Context, id int64) (*response.EmailDetailResponse, error)
	GetEmailList(ctx context.Context, params *EmailQueryParams, page, pageSize int) (*EmailListResponse, error)
	SearchEmails(ctx context.Context, query string, accountUID string, page, pageSize int) (*EmailListResponse, error)
	GetEmailListWithFilter(ctx context.Context, params *EmailQueryParams, offset, limit int) ([]*model.Email, int64, error)
	SearchEmailsWithFilter(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error)
	GetUnreadCount(ctx context.Context, accountUID string) (int64, error)
	GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error)
	GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error)
}

// EmailStatusServiceInterface 邮件本地状态变更能力。
type EmailStatusServiceInterface interface {
	MarkAsRead(ctx context.Context, ids []int64) error
	MarkAsUnread(ctx context.Context, ids []int64) error
	MarkAllAsRead(ctx context.Context, accountUID *string) (int64, error)
	ToggleStar(ctx context.Context, id int64) error
	ArchiveEmail(ctx context.Context, id int64) error
}

// EmailDeleteServiceInterface 邮件删除与恢复能力。
type EmailDeleteServiceInterface interface {
	DeleteEmail(ctx context.Context, id int64) error
	BatchDeleteEmails(ctx context.Context, ids []int64) (int64, error)
	RestoreEmail(ctx context.Context, id int64) error
	PermanentDeleteEmail(ctx context.Context, id int64) error
	BatchPermanentDeleteEmails(ctx context.Context, ids []int64) (int64, error)
	EmptyTrash(ctx context.Context) (int64, error)
}

// EmailService 是三个能力的组合接口。
//
// 保留它是为了兼容仍需要全部能力的消费方（如 email_handler，它注册了 17 个端点）。
// 新代码应优先依赖上面三个窄接口之一。
type EmailService interface {
	EmailQueryServiceInterface
	EmailStatusServiceInterface
	EmailDeleteServiceInterface
}

// NewEmailService 构造一个同时实现三个能力的组合服务。
//
// 内部由三个聚焦的类型组合而成，而不是一个 985 行的大类型。
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

// NewEmailServiceWithCredentialResolver 同上，但接受显式的凭证解析器。
func NewEmailServiceWithCredentialResolver(
	emailRepo repository.EmailRepository,
	accountRepo repository.AccountRepository,
	adapterFactory *adapter.Factory,
	credentialResolver *CredentialResolver,
) EmailService {
	return &compositeEmailService{
		query:  NewEmailQueryService(emailRepo, accountRepo, adapterFactory, credentialResolver),
		status: NewEmailStatusService(emailRepo, accountRepo),
		delete: NewEmailDeleteService(emailRepo, accountRepo, adapterFactory, credentialResolver),
	}
}

// compositeEmailService 把三个聚焦服务组合成 EmailService。
//
// 它只做转发，不含任何业务逻辑——这样"上帝接口"不会重新长出上帝实现。
type compositeEmailService struct {
	query  *EmailQueryService
	status *EmailStatusService
	delete *EmailDeleteService
}

func (c *compositeEmailService) GetEmailByID(ctx context.Context, id int64) (*response.EmailDetailResponse, error) {
	return c.query.GetEmailByID(ctx, id)
}
func (c *compositeEmailService) GetEmailList(ctx context.Context, params *EmailQueryParams, page, pageSize int) (*EmailListResponse, error) {
	return c.query.GetEmailList(ctx, params, page, pageSize)
}
func (c *compositeEmailService) SearchEmails(ctx context.Context, query string, accountUID string, page, pageSize int) (*EmailListResponse, error) {
	return c.query.SearchEmails(ctx, query, accountUID, page, pageSize)
}
func (c *compositeEmailService) GetEmailListWithFilter(ctx context.Context, params *EmailQueryParams, offset, limit int) ([]*model.Email, int64, error) {
	return c.query.GetEmailListWithFilter(ctx, params, offset, limit)
}
func (c *compositeEmailService) SearchEmailsWithFilter(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error) {
	return c.query.SearchEmailsWithFilter(ctx, query, accountUID, offset, limit)
}
func (c *compositeEmailService) GetUnreadCount(ctx context.Context, accountUID string) (int64, error) {
	return c.query.GetUnreadCount(ctx, accountUID)
}
func (c *compositeEmailService) GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error) {
	return c.query.GetAccountStats(ctx, accountUID)
}
func (c *compositeEmailService) GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error) {
	return c.query.GetGlobalStats(ctx)
}

func (c *compositeEmailService) MarkAsRead(ctx context.Context, ids []int64) error {
	return c.status.MarkAsRead(ctx, ids)
}
func (c *compositeEmailService) MarkAsUnread(ctx context.Context, ids []int64) error {
	return c.status.MarkAsUnread(ctx, ids)
}
func (c *compositeEmailService) MarkAllAsRead(ctx context.Context, accountUID *string) (int64, error) {
	return c.status.MarkAllAsRead(ctx, accountUID)
}
func (c *compositeEmailService) ToggleStar(ctx context.Context, id int64) error {
	return c.status.ToggleStar(ctx, id)
}
func (c *compositeEmailService) ArchiveEmail(ctx context.Context, id int64) error {
	return c.status.ArchiveEmail(ctx, id)
}

func (c *compositeEmailService) DeleteEmail(ctx context.Context, id int64) error {
	return c.delete.DeleteEmail(ctx, id)
}
func (c *compositeEmailService) BatchDeleteEmails(ctx context.Context, ids []int64) (int64, error) {
	return c.delete.BatchDeleteEmails(ctx, ids)
}
func (c *compositeEmailService) RestoreEmail(ctx context.Context, id int64) error {
	return c.delete.RestoreEmail(ctx, id)
}
func (c *compositeEmailService) PermanentDeleteEmail(ctx context.Context, id int64) error {
	return c.delete.PermanentDeleteEmail(ctx, id)
}
func (c *compositeEmailService) BatchPermanentDeleteEmails(ctx context.Context, ids []int64) (int64, error) {
	return c.delete.BatchPermanentDeleteEmails(ctx, ids)
}
func (c *compositeEmailService) EmptyTrash(ctx context.Context) (int64, error) {
	return c.delete.EmptyTrash(ctx)
}
