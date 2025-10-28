package model

import (
	"time"
)

// EmailAttachment 邮件附件模型
type EmailAttachment struct {
	ID      int64 `gorm:"primaryKey" json:"id"`
	EmailID int64 `gorm:"not null;index" json:"email_id"`

	// 附件信息
	Filename    string `gorm:"size:255;not null" json:"filename"`
	ContentType string `gorm:"size:100" json:"content_type"`
	SizeBytes   int64  `gorm:"not null" json:"size_bytes"`

	// 存储信息
	StorageType string `gorm:"size:20;default:'local'" json:"storage_type"` // local/s3/oss
	StoragePath string `gorm:"type:text;not null" json:"storage_path"`      // 存储路径或 URL

	// 元数据
	IsInline  bool   `gorm:"default:false" json:"is_inline"` // 是否内联附件
	ContentID string `gorm:"size:255" json:"content_id"`     // Content-ID（用于内联）

	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (EmailAttachment) TableName() string {
	return "email_attachments"
}
