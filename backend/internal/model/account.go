package model

import (
	"time"
)

// Account 邮箱账户模型
type Account struct {
	ID       int64  `gorm:"primaryKey" json:"id"`
	UID      string `gorm:"uniqueIndex;size:64;not null" json:"uid"` // 账户唯一标识
	Email    string `gorm:"size:255;not null" json:"email"`          // 邮箱地址
	Provider string `gorm:"size:50;not null" json:"provider"`        // 服务商类型 (gmail/outlook/imap/pop3)
	Protocol string `gorm:"size:20;not null" json:"protocol"`        // 协议类型 (gmail_api/graph/imap/pop3)

	// 认证信息（加密存储）
	AuthType             string `gorm:"size:20;not null" json:"auth_type"` // 认证类型 (oauth2/password/app_password)
	EncryptedCredentials string `gorm:"type:text;not null" json:"-"`       // 加密后的凭证 (JSON)

	// 代理配置
	ProxyEnabled           bool   `gorm:"default:false" json:"proxy_enabled"`
	ProxyType              string `gorm:"size:20" json:"proxy_type"` // http/socks5
	ProxyHost              string `gorm:"size:255" json:"proxy_host"`
	ProxyPort              int    `json:"proxy_port"`
	ProxyUsername          string `gorm:"size:255" json:"proxy_username"`
	EncryptedProxyPassword string `gorm:"type:text" json:"-"`

	// 账户状态
	Status string `gorm:"size:20;default:'active'" json:"status"` // 账户状态 (active/disabled/error)

	// 同步配置
	SyncEnabled    bool       `gorm:"default:true" json:"sync_enabled"`
	SyncInterval   int        `gorm:"default:5" json:"sync_interval"` // 同步间隔（分钟）
	LastSyncAt     *time.Time `json:"last_sync_at"`
	LastSyncStatus string     `gorm:"size:20" json:"last_sync_status"` // success/failed/running
	LastSyncError  string     `gorm:"type:text" json:"last_sync_error"`

	// 统计信息
	TotalEmails int `gorm:"default:0" json:"total_emails"`
	UnreadCount int `gorm:"default:0" json:"unread_count"`

	// 元数据
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (Account) TableName() string {
	return "accounts"
}
