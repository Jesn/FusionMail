package model

import (
	"time"
)

// Email 邮件主表模型
type Email struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	// 唯一标识（Provider ID + Account UID）
	ProviderID string `gorm:"size:255;not null;uniqueIndex:idx_provider_account" json:"provider_id"` // 邮箱服务商原生 ID
	AccountUID string `gorm:"size:64;not null;uniqueIndex:idx_provider_account" json:"account_uid"`  // 所属账户 UID
	MessageID  string `gorm:"size:255;index" json:"message_id"`                                      // 邮件 Message-ID（辅助）

	// 基本信息
	Subject      string `gorm:"type:text;not null" json:"subject"`
	FromAddress  string `gorm:"size:255;not null;index" json:"from_address"`
	FromName     string `gorm:"size:255" json:"from_name"`
	ToAddresses  string `gorm:"type:text" json:"to_addresses"`  // JSON 数组
	CcAddresses  string `gorm:"type:text" json:"cc_addresses"`  // JSON 数组
	BccAddresses string `gorm:"type:text" json:"bcc_addresses"` // JSON 数组
	ReplyTo      string `gorm:"size:255" json:"reply_to"`

	// 邮件内容
	TextBody string `gorm:"type:text" json:"text_body"` // 纯文本正文
	HTMLBody string `gorm:"type:text" json:"html_body"` // HTML 正文
	Snippet  string `gorm:"type:text" json:"snippet"`   // 摘要（前 200 字符）

	// 本地状态（只读镜像模式）
	IsRead      bool   `gorm:"default:false;index" json:"is_read"`     // 本地已读状态
	IsStarred   bool   `gorm:"default:false;index" json:"is_starred"`  // 本地星标状态
	IsArchived  bool   `gorm:"default:false;index" json:"is_archived"` // 本地归档状态
	IsDeleted   bool   `gorm:"default:false;index" json:"is_deleted"`  // 本地删除状态（软删除）
	LocalLabels string `gorm:"type:text" json:"local_labels"`          // 本地标签（JSON 数组）

	// 源邮箱状态（只读，不修改）
	SourceIsRead *bool  `json:"source_is_read"`                 // 源邮箱已读状态
	SourceLabels string `gorm:"type:text" json:"source_labels"` // 源邮箱标签（JSON 数组）
	SourceFolder string `gorm:"size:255" json:"source_folder"`  // 源邮箱文件夹

	// 附件信息
	HasAttachments   bool `gorm:"default:false" json:"has_attachments"`
	AttachmentsCount int  `gorm:"default:0" json:"attachments_count"`

	// 时间信息
	SentAt     time.Time `gorm:"not null;index:idx_sent_at,sort:desc" json:"sent_at"` // 发送时间
	ReceivedAt time.Time `gorm:"not null" json:"received_at"`                         // 接收时间
	SyncedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"synced_at"`          // 同步时间

	// 元数据
	SizeBytes  int64  `json:"size_bytes"`                  // 邮件大小
	ThreadID   string `gorm:"size:255" json:"thread_id"`   // 会话 ID
	InReplyTo  string `gorm:"size:255" json:"in_reply_to"` // 回复的邮件 ID
	References string `gorm:"type:text" json:"references"` // 引用的邮件 ID 列表

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Attachments []EmailAttachment `gorm:"foreignKey:EmailID" json:"attachments,omitempty"`
}

// TableName 指定表名
func (Email) TableName() string {
	return "emails"
}
