package model

import (
	"time"
)

// APIKey API 密钥模型
type APIKey struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	KeyHash     string `gorm:"uniqueIndex;size:64;not null" json:"-"` // API Key 的 SHA-256 哈希
	Name        string `gorm:"size:255;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// 权限配置
	Permissions string `gorm:"type:text" json:"permissions"` // 权限列表（JSON 数组）

	// 速率限制
	RateLimit int `gorm:"default:100" json:"rate_limit"` // 每分钟请求数

	// 状态
	Enabled   bool       `gorm:"default:true;index" json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at"`

	// 统计信息
	TotalRequests int        `gorm:"default:0" json:"total_requests"`
	LastUsedAt    *time.Time `json:"last_used_at"`
	LastIP        string     `gorm:"size:45" json:"last_ip"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}
