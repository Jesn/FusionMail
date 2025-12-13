package model

import (
	"time"

	"gorm.io/gorm"
)

// SentEmail 已发送邮件模型
// Requirements: 1.4, 7.1, 7.2
type SentEmail struct {
	ID            int64  `gorm:"primaryKey" json:"id"`
	AccountUID    string `gorm:"size:64;not null;index" json:"account_uid"` // 发送账户 UID
	ProviderMsgID string `gorm:"size:255" json:"provider_msg_id"`           // 服务商返回的消息 ID
	MessageID     string `gorm:"size:255;index" json:"message_id"`          // 邮件 Message-ID（RFC 2822）

	// 邮件内容
	Subject      string `gorm:"type:text" json:"subject"`       // 主题
	FromAddress  string `gorm:"size:255" json:"from_address"`   // 发件人地址
	FromName     string `gorm:"size:255" json:"from_name"`      // 发件人名称
	ToAddresses  string `gorm:"type:text" json:"to_addresses"`  // 收件人列表（JSON 数组）
	CcAddresses  string `gorm:"type:text" json:"cc_addresses"`  // 抄送列表（JSON 数组）
	BccAddresses string `gorm:"type:text" json:"bcc_addresses"` // 密送列表（JSON 数组）
	TextBody     string `gorm:"type:text" json:"text_body"`     // 纯文本正文
	HTMLBody     string `gorm:"type:text" json:"html_body"`     // HTML 正文

	// 附件信息
	HasAttachments  bool   `gorm:"default:false" json:"has_attachments"` // 是否有附件
	AttachmentCount int    `gorm:"default:0" json:"attachment_count"`    // 附件数量
	AttachmentInfo  string `gorm:"type:text" json:"attachment_info"`     // 附件信息（JSON 数组，包含文件名、大小等）

	// 关联信息（回复/转发）
	ReplyToEmailID *int64 `gorm:"index" json:"reply_to_email_id"` // 回复的邮件 ID（本地 Email 表 ID）
	ForwardFromID  *int64 `gorm:"index" json:"forward_from_id"`   // 转发的原邮件 ID（本地 Email 表 ID）
	InReplyTo      string `gorm:"size:255" json:"in_reply_to"`    // In-Reply-To 邮件头
	References     string `gorm:"type:text" json:"references"`    // References 邮件头

	// 发送状态
	// Requirements: 7.2, 7.3
	Status       string     `gorm:"size:20;default:'sent';index" json:"status"` // 发送状态：sent/failed
	ErrorMessage string     `gorm:"type:text" json:"error_message"`             // 错误信息（如果失败）
	SentAt       time.Time  `gorm:"not null;index" json:"sent_at"`              // 发送时间
	RetryCount   int        `gorm:"default:0" json:"retry_count"`               // 重试次数
	LastRetryAt  *time.Time `json:"last_retry_at"`                              // 最后重试时间

	// 发送器信息
	SenderType string `gorm:"size:20" json:"sender_type"` // 发送器类型：gmail_api/graph_api/smtp

	// 元数据
	SizeBytes int64 `json:"size_bytes"` // 邮件大小（字节）

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (SentEmail) TableName() string {
	return "sent_emails"
}

// 发送状态常量
const (
	SentEmailStatusSent   = "sent"   // 发送成功
	SentEmailStatusFailed = "failed" // 发送失败
)

// SentEmailAttachmentInfo 已发送邮件附件信息（用于 JSON 序列化）
type SentEmailAttachmentInfo struct {
	Filename    string `json:"filename"`     // 文件名
	ContentType string `json:"content_type"` // 内容类型
	SizeBytes   int64  `json:"size_bytes"`   // 大小（字节）
}
