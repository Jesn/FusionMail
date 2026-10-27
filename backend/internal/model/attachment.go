package model

import (
	"time"
)

// EmailAttachment 邮件附件模型
type EmailAttachment struct {
	ID      int64 `gorm:"primaryKey"`
	EmailID int64 `gorm:"not null;index"`

	// 附件信息
	Filename    string `gorm:"size:255;not null"`
	ContentType string `gorm:"size:100"`
	SizeBytes   int64  `gorm:"not null"`

	// 存储信息
	StorageType string `gorm:"size:20;default:'local'"` // local/s3/oss
	StoragePath string `gorm:"type:text;not null"`      // 存储路径或 URL
	URL         string `gorm:"-"`                       // 访问 URL（不存储在数据库）

	// 元数据
	IsInline  bool   `gorm:"default:false"` // 是否内联附件
	ContentID string `gorm:"size:255"`      // Content-ID（用于内联）

	CreatedAt time.Time
}

// TableName 指定表名
func (EmailAttachment) TableName() string {
	return "email_attachments"
}
