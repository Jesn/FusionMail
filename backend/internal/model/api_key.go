package model

import (
	"time"
)

// APIKey API 密钥模型
type APIKey struct {
	ID          int64  `gorm:"primaryKey"`
	KeyHash     string `gorm:"uniqueIndex;size:64;not null"` // API Key 的 SHA-256 哈希
	Name        string `gorm:"size:255;not null"`
	Description string `gorm:"type:text"`

	// 权限配置
	Permissions string `gorm:"type:text"` // 权限列表（JSON 数组）

	// 速率限制
	RateLimit int `gorm:"default:100"` // 每分钟请求数

	// 状态
	Enabled   bool `gorm:"default:true;index"`
	ExpiresAt *time.Time

	// 统计信息
	TotalRequests int `gorm:"default:0"`
	LastUsedAt    *time.Time
	LastIP        string `gorm:"size:45"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}
