package model

import (
	"time"

	"gorm.io/gorm"
)

// Email 邮件主表模型
type Email struct {
	ID int64 `gorm:"primaryKey"`

	// 唯一标识（Provider ID + Account UID）
	ProviderID string `gorm:"size:255;not null;uniqueIndex:idx_provider_account"` // 邮箱服务商原生 ID
	AccountUID string `gorm:"size:64;not null;uniqueIndex:idx_provider_account"`  // 所属账户 UID
	MessageID  string `gorm:"size:255;index"`                                     // 邮件 Message-ID（辅助）

	// 去重标识（用于稳定的邮件去重）
	// 格式：有 Message-ID 时为 "mid:{message_id}"，无 Message-ID 时为 "hash:{sha256[:32]}"
	DedupeKey string `gorm:"size:128;index"` // 去重标识

	// 基本信息
	Subject      string `gorm:"type:text;not null"`
	FromAddress  string `gorm:"size:255;not null;index"`
	FromName     string `gorm:"size:255"`
	ToAddress    string `gorm:"size:255;index"` // 主要收件人（用于规则匹配）
	ToAddresses  string `gorm:"type:text"`      // JSON 数组
	CcAddresses  string `gorm:"type:text"`      // JSON 数组
	BccAddresses string `gorm:"type:text"`      // JSON 数组
	ReplyTo      string `gorm:"size:255"`

	// 邮件内容
	TextBody string `gorm:"type:text"` // 纯文本正文
	HTMLBody string `gorm:"type:text"` // HTML 正文
	Snippet  string `gorm:"type:text"` // 摘要（前 200 字符）

	// 本地状态（只读镜像模式）
	IsRead      bool   `gorm:"default:false;index"` // 本地已读状态
	IsStarred   bool   `gorm:"default:false;index"` // 本地星标状态
	IsArchived  bool   `gorm:"default:false;index"` // 本地归档状态
	IsDeleted   bool   `gorm:"default:false;index"` // 本地删除状态（软删除）
	Labels      string `gorm:"type:text"`           // 本地标签（JSON 数组）
	LocalLabels string `gorm:"type:text"`           // 本地标签（JSON 数组，兼容字段）
	Folder      string `gorm:"size:255"`            // 本地文件夹

	// 源邮箱状态（只读，不修改）
	SourceIsRead *bool
	SourceLabels string `gorm:"type:text"` // 源邮箱标签（JSON 数组）
	SourceFolder string `gorm:"size:255"`  // 源邮箱文件夹

	// 附件信息
	HasAttachment    bool `gorm:"default:false;index"` // 是否有附件（用于规则匹配）
	HasAttachments   bool `gorm:"default:false"`
	AttachmentsCount int  `gorm:"default:0"`

	// 时间信息
	SentAt     time.Time `gorm:"not null;index:idx_sent_at,sort:desc"` // 发送时间
	ReceivedAt time.Time `gorm:"not null"`                             // 接收时间
	SyncedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP"`            // 同步时间

	// 元数据
	SizeBytes  int64
	ThreadID   string `gorm:"size:255"`  // 会话 ID
	InReplyTo  string `gorm:"size:255"`  // 回复的邮件 ID
	References string `gorm:"type:text"` // 引用的邮件 ID 列表

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" swaggerignore:"true"` // 软删除时间

	// 垃圾邮件检测相关
	IsSpam         bool    `gorm:"default:false;index"` // 是否为垃圾邮件
	SpamScore      float64 `gorm:"default:0;index"`     // 垃圾邮件评分（0-100）
	SpamConfidence float64 `gorm:"default:0"`           // 置信度（0-1.0）
	SpamReason     string  `gorm:"type:text"`           // 检测原因（JSON 数组）
	SpamDetectedAt *time.Time
	SpamDetectedBy string `gorm:"size:50"`       // 检测层级
	UserMarkedSpam bool   `gorm:"default:false"` // 用户是否手动标记为垃圾
	UserMarkedAt   *time.Time

	// 关联
	Attachments []EmailAttachment `gorm:"foreignKey:EmailID"`
}

// TableName 指定表名
func (Email) TableName() string {
	return "emails"
}
