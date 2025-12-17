package model

import (
	"time"

	"gorm.io/gorm"
)

// EmailAccount 邮箱账户模型
type EmailAccount struct {
	ID       int64  `gorm:"primaryKey" json:"id"`
	UID      string `gorm:"uniqueIndex;size:64;not null" json:"uid"` // 账户唯一标识
	Email    string `gorm:"size:255;not null" json:"email"`          // 邮箱地址
	Provider string `gorm:"size:50;not null" json:"provider"`        // 服务商类型 (gmail/outlook/imap/pop3) - 保留用于向后兼容
	Protocol string `gorm:"size:20;not null" json:"protocol"`        // 协议类型 (gmail_api/graph/imap/pop3) - 保留用于向后兼容

	// 外键关联（新增）
	ProviderID  int64     `gorm:"index" json:"provider_id"`                            // 关联的提供商 ID
	ProviderRef *Provider `gorm:"foreignKey:ProviderID" json:"provider_ref,omitempty"` // 关联的提供商
	AdapterID   int64     `gorm:"index" json:"adapter_id"`                             // 用户选择的适配器 ID
	AdapterRef  *Adapter  `gorm:"foreignKey:AdapterID" json:"adapter_ref,omitempty"`   // 关联的适配器

	// 认证信息（加密存储）
	AuthType             string `gorm:"size:20;not null" json:"auth_type"` // 认证类型 (oauth2/password/app_password) - 保留用于向后兼容
	EncryptedCredentials string `gorm:"type:text;not null" json:"-"`       // 加密后的凭证 (JSON)

	// 通用邮箱服务器配置（仅用于 generic 提供商）- 保留用于向后兼容
	IMAPHost   string `gorm:"size:255" json:"imap_host"` // IMAP 服务器地址
	IMAPPort   int    `json:"imap_port"`                 // IMAP 端口
	POP3Host   string `gorm:"size:255" json:"pop3_host"` // POP3 服务器地址
	POP3Port   int    `json:"pop3_port"`                 // POP3 端口
	Encryption string `gorm:"size:20" json:"encryption"` // 加密方式 (ssl/starttls/none)

	// SMTP 发送配置（Requirements: 3.1）- 保留用于向后兼容
	SMTPHost              string `gorm:"size:255" json:"smtp_host"`         // SMTP 服务器地址
	SMTPPort              int    `json:"smtp_port"`                         // SMTP 端口
	SMTPEncryption        string `gorm:"size:20" json:"smtp_encryption"`    // SMTP 加密方式 (none/tls/starttls)
	SMTPUsername          string `gorm:"size:255" json:"smtp_username"`     // SMTP 用户名（通常是邮箱地址）
	EncryptedSMTPPassword string `gorm:"type:text" json:"-"`                // SMTP 密码（AES-256 加密存储）
	SMTPEnabled           bool   `gorm:"default:false" json:"smtp_enabled"` // 是否启用 SMTP 发送

	// 代理配置
	ProxyEnabled           bool   `gorm:"default:false" json:"proxy_enabled"`
	ProxyType              string `gorm:"size:20" json:"proxy_type"` // http/socks5
	ProxyHost              string `gorm:"size:255" json:"proxy_host"`
	ProxyPort              int    `json:"proxy_port"`
	ProxyUsername          string `gorm:"size:255" json:"proxy_username"`
	EncryptedProxyPassword string `gorm:"type:text" json:"-"`

	// 账户状态
	Status string `gorm:"size:20;default:'active'" json:"status"` // 账户状态 (active/disabled/error)

	// 自动禁用相关字段（用于短期邮箱过期处理）
	ConsecutiveAuthFailures int        `gorm:"default:0;not null" json:"consecutive_auth_failures"` // 连续认证失败次数
	AutoDisabledAt          *time.Time `json:"auto_disabled_at,omitempty"`                          // 自动禁用时间
	DisableReason           string     `gorm:"size:100" json:"disable_reason,omitempty"`            // 禁用原因

	// 同步配置
	SyncEnabled    bool       `gorm:"default:true" json:"sync_enabled"`
	SyncInterval   int        `gorm:"default:2" json:"sync_interval"` // 同步间隔（分钟）
	LastSyncAt     *time.Time `json:"last_sync_at"`
	LastSyncStatus string     `gorm:"size:20" json:"last_sync_status"` // success/failed/running
	LastSyncError  string     `gorm:"type:text" json:"last_sync_error"`

	// UID 增量同步状态（用于 IMAP 协议）
	UIDValidity int64 `gorm:"default:0" json:"uid_validity"` // IMAP UIDVALIDITY 值，变化时需要全量同步
	LastUID     int64 `gorm:"default:0" json:"last_uid"`     // 上次同步的最大 UID，用于增量同步

	// 首次同步优化配置 (Requirements 6.1)
	FirstSyncDays    int    `gorm:"default:7" json:"first_sync_days"`                  // 首次同步天数，0 表示全量，默认 7
	BatchSize        int    `gorm:"default:100" json:"batch_size"`                     // 每批处理数量，默认 100
	MaxEmailsPerSync int    `gorm:"default:5000" json:"max_emails_per_sync"`           // 单次同步最大邮件数，默认 5000
	SyncCursor       string `gorm:"type:text" json:"sync_cursor"`                      // 同步游标，用于断点续传
	SyncProgressJSON string `gorm:"type:jsonb;default:'{}'" json:"sync_progress_json"` // 同步进度 JSON，存储详细进度信息

	// 删除策略配置
	ServerDeletePolicy string `gorm:"size:10;default:'off'" json:"server_delete_policy"` // 'off' 或 'soft'

	// 分组配置
	GroupID *int64 `gorm:"index" json:"group_id,omitempty"` // 所属分组ID，NULL 表示未分组

	// 统计信息
	TotalEmails int `gorm:"default:0" json:"total_emails"`
	UnreadCount int `gorm:"default:0" json:"unread_count"`

	// 元数据
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (EmailAccount) TableName() string {
	return "email_accounts"
}

// GetSyncConfig 获取账户的同步配置
// 如果账户未设置配置，则返回默认配置
func (a *EmailAccount) GetSyncConfig() *SyncConfig {
	config := &SyncConfig{
		FirstSyncDays:    a.FirstSyncDays,
		BatchSize:        a.BatchSize,
		MaxEmailsPerSync: a.MaxEmailsPerSync,
		ProgressInterval: DefaultProgressInterval,
		RetryCount:       DefaultRetryCount,
		RetryBackoffMs:   DefaultRetryBackoffMs,
	}

	// 使用默认值填充零值字段
	return config.MergeWithDefaults()
}

// SetSyncConfig 设置账户的同步配置
func (a *EmailAccount) SetSyncConfig(config *SyncConfig) {
	if config == nil {
		return
	}
	a.FirstSyncDays = config.FirstSyncDays
	a.BatchSize = config.BatchSize
	a.MaxEmailsPerSync = config.MaxEmailsPerSync
}

// GetAdapterName 获取账户使用的适配器名称
func (a *EmailAccount) GetAdapterName() string {
	if a.AdapterRef != nil {
		return a.AdapterRef.Name
	}
	return ""
}

// GetAuthType 获取认证类型（优先从适配器获取）
func (a *EmailAccount) GetAuthType() string {
	if a.AdapterRef != nil {
		return a.AdapterRef.AuthType
	}
	return a.AuthType
}

// IsOAuth2 检查是否使用 OAuth2 认证
func (a *EmailAccount) IsOAuth2() bool {
	return a.GetAuthType() == AdapterAuthTypeOAuth2
}

// GetIMAPConfig 获取 IMAP 配置（优先从 Provider 获取）
func (a *EmailAccount) GetIMAPConfig() (host string, port int, encryption string) {
	// 优先从关联的 Provider 获取
	if a.ProviderRef != nil && a.ProviderRef.IMAPHost != "" {
		return a.ProviderRef.IMAPHost, a.ProviderRef.IMAPPort, a.ProviderRef.IMAPEncryption
	}
	// 回退到账户自身的配置（向后兼容）
	return a.IMAPHost, a.IMAPPort, a.Encryption
}

// GetSMTPConfig 获取 SMTP 配置（优先从 Provider 获取）
func (a *EmailAccount) GetSMTPConfig() (host string, port int, encryption string) {
	// 优先从关联的 Provider 获取
	if a.ProviderRef != nil && a.ProviderRef.SMTPHost != "" {
		return a.ProviderRef.SMTPHost, a.ProviderRef.SMTPPort, a.ProviderRef.SMTPEncryption
	}
	// 回退到账户自身的配置（向后兼容）
	return a.SMTPHost, a.SMTPPort, a.SMTPEncryption
}

// GetProviderName 获取提供商名称（优先从 Provider 获取）
func (a *EmailAccount) GetProviderName() string {
	if a.ProviderRef != nil {
		return a.ProviderRef.Name
	}
	return a.Provider
}
