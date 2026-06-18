package response

import (
	"time"
)

// EmailDetailResponse 邮件详情响应 DTO
// 不包含 DedupeKey、SyncedAt、DeletedAt 等内部字段
type EmailDetailResponse struct {
	ID int64 `json:"id"`

	// 唯一标识
	ProviderID string `json:"provider_id"`
	AccountUID string `json:"account_uid"`
	MessageID  string `json:"message_id"`

	// 基本信息
	Subject      string `json:"subject"`
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	ToAddress    string `json:"to_address"`
	ToAddresses  string `json:"to_addresses"`
	CcAddresses  string `json:"cc_addresses"`
	BccAddresses string `json:"bcc_addresses"`
	ReplyTo      string `json:"reply_to"`

	// 邮件内容
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body"`
	Snippet  string `json:"snippet"`

	// 本地状态
	IsRead      bool   `json:"is_read"`
	IsStarred   bool   `json:"is_starred"`
	IsArchived  bool   `json:"is_archived"`
	IsDeleted   bool   `json:"is_deleted"`
	Labels      string `json:"labels"`
	LocalLabels string `json:"local_labels"`
	Folder      string `json:"folder"`

	// 源邮箱状态
	SourceIsRead *bool  `json:"source_is_read"`
	SourceLabels string `json:"source_labels"`
	SourceFolder string `json:"source_folder"`

	// 附件信息
	HasAttachment    bool `json:"has_attachment"`
	HasAttachments   bool `json:"has_attachments"`
	AttachmentsCount int  `json:"attachments_count"`

	// 时间信息
	SentAt     time.Time `json:"sent_at"`
	ReceivedAt time.Time `json:"received_at"`

	// 元数据
	SizeBytes  int64  `json:"size_bytes"`
	ThreadID   string `json:"thread_id"`
	InReplyTo  string `json:"in_reply_to"`
	References string `json:"references"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 垃圾邮件检测
	IsSpam         bool       `json:"is_spam"`
	SpamScore      float64    `json:"spam_score"`
	SpamConfidence float64    `json:"spam_confidence"`
	SpamReason     string     `json:"spam_reason"`
	SpamDetectedAt *time.Time `json:"spam_detected_at"`
	SpamDetectedBy string     `json:"spam_detected_by"`
	UserMarkedSpam bool       `json:"user_marked_spam"`
	UserMarkedAt   *time.Time `json:"user_marked_at"`
}

// AccountResponse 账户响应 DTO
// 不包含 EncryptedCredentials、SyncProgressJSON、SyncCursor 等内部字段
type AccountResponse struct {
	ID    int64  `json:"id"`
	UID   string `json:"uid"`
	Email string `json:"email"`

	// 外键关联
	ProviderID  int64     `json:"provider_id"`
	ProviderRef *Provider `json:"provider_ref,omitempty"`
	AdapterID   int64     `json:"adapter_id"`
	AdapterRef  *Adapter  `json:"adapter_ref,omitempty"`

	// SMTP 发送配置
	SMTPEnabled bool `json:"smtp_enabled"`

	// 代理配置
	ProxyEnabled  bool   `json:"proxy_enabled"`
	ProxyType     string `json:"proxy_type"`
	ProxyHost     string `json:"proxy_host"`
	ProxyPort     int    `json:"proxy_port"`
	ProxyUsername string `json:"proxy_username"`

	// 账户状态
	Status string `json:"status"`

	// 自动禁用
	AutoDisabledAt *time.Time `json:"auto_disabled_at,omitempty"`
	DisableReason  string     `json:"disable_reason,omitempty"`

	// 同步配置
	SyncEnabled    bool       `json:"sync_enabled"`
	SyncInterval   int        `json:"sync_interval"`
	LastSyncAt     *time.Time `json:"last_sync_at"`
	LastSyncStatus string     `json:"last_sync_status"`
	LastSyncError  string     `json:"last_sync_error"`

	// 首次同步配置
	FirstSyncDays    int `json:"first_sync_days"`
	BatchSize        int `json:"batch_size"`
	MaxEmailsPerSync int `json:"max_emails_per_sync"`

	// 删除策略
	ServerDeletePolicy string `json:"server_delete_policy"`

	// 分组
	GroupID *int64 `json:"group_id,omitempty"`

	// 父子账户
	ParentAccountUID *string `json:"parent_account_uid,omitempty"`

	// 统计
	TotalEmails int `json:"total_emails"`
	UnreadCount int `json:"unread_count"`

	// 元数据
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Provider 响应中的 Provider 简要信息
type Provider struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// Adapter 响应中的 Adapter 简要信息
type Adapter struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
