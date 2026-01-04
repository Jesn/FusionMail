package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// EmailAccount 邮箱账户模型
type EmailAccount struct {
	ID    int64  `gorm:"primaryKey" json:"id"`
	UID   string `gorm:"uniqueIndex;size:64;not null" json:"uid"` // 账户唯一标识
	Email string `gorm:"size:255;not null" json:"email"`          // 邮箱地址

	// 外键关联
	ProviderID  int64     `gorm:"index;not null" json:"provider_id"`                                                                // 关联的提供商 ID
	ProviderRef *Provider `gorm:"foreignKey:ProviderID;references:ID;constraint:OnUpdate:CASCADE;->" json:"provider_ref,omitempty"` // 关联的提供商（只读）
	AdapterID   int64     `gorm:"index;not null" json:"adapter_id"`                                                                 // 用户选择的适配器 ID
	AdapterRef  *Adapter  `gorm:"foreignKey:AdapterID;references:ID;constraint:OnUpdate:CASCADE;->" json:"adapter_ref,omitempty"`   // 关联的适配器（只读）

	// 认证信息（加密存储）
	EncryptedCredentials string `gorm:"type:text;not null" json:"-"` // 加密后的凭证 (JSON)

	// SMTP 发送配置
	SMTPEnabled bool `gorm:"default:false" json:"smtp_enabled"` // 是否启用 SMTP 发送

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
	SyncInterval   int        `gorm:"default:2" json:"sync_interval"`                      // 同步间隔（分钟）
	SyncModeField  string     `gorm:"column:sync_mode;size:20;default:'polling'" json:"-"` // 同步模式：polling（轮询）或 webhook（推送）
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

	// WebAPI 父子账户关系
	ParentAccountUID *string `gorm:"size:64;index" json:"parent_account_uid,omitempty"` // 父账户 UID，用于 WebAPI 子邮箱关联

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

// MarshalJSON 自定义 JSON 序列化
// 添加 provider 字段（从 ProviderRef.Name 获取）以兼容前端
func (a EmailAccount) MarshalJSON() ([]byte, error) {
	// 定义一个别名类型，避免递归调用
	type Alias EmailAccount

	// 获取 provider 名称
	provider := ""
	if a.ProviderRef != nil {
		provider = a.ProviderRef.Name
	}

	// 获取 adapter 名称（auth_type）和协议
	authType := ""
	protocol := ""
	if a.AdapterRef != nil {
		authType = a.AdapterRef.AuthType
		protocol = a.AdapterRef.Name // 使用适配器名称作为协议标识
	}

	// 获取同步模式（polling 或 webhook）
	syncMode := a.GetSyncMode()

	// 创建包含额外字段的结构
	return json.Marshal(&struct {
		Provider string `json:"provider"`
		AuthType string `json:"auth_type"`
		Protocol string `json:"protocol"`
		SyncMode string `json:"sync_mode"` // 同步模式：polling（轮询）或 webhook（推送）
		*Alias
	}{
		Provider: provider,
		AuthType: authType,
		Protocol: protocol,
		SyncMode: syncMode,
		Alias:    (*Alias)(&a),
	})
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

// GetAuthType 获取认证类型（从适配器获取）
func (a *EmailAccount) GetAuthType() string {
	if a.AdapterRef != nil {
		return a.AdapterRef.AuthType
	}
	return ""
}

// IsOAuth2 检查是否使用 OAuth2 认证
func (a *EmailAccount) IsOAuth2() bool {
	return a.GetAuthType() == AdapterAuthTypeOAuth2
}

// GetProtocol 获取协议类型（从适配器推导）
func (a *EmailAccount) GetProtocol() string {
	if a.AdapterRef != nil {
		// 根据适配器名称推导协议
		switch a.AdapterRef.Name {
		case "gmail":
			return "oauth2"
		case "graph":
			return "oauth2"
		case "imap":
			return "imap"
		case "pop3":
			return "pop3"
		default:
			return a.AdapterRef.Name
		}
	}
	return ""
}

// GetIMAPConfig 获取 IMAP 配置（从 Provider 获取）
func (a *EmailAccount) GetIMAPConfig() (host string, port int, encryption string) {
	if a.ProviderRef != nil {
		return a.ProviderRef.IMAPHost, a.ProviderRef.IMAPPort, a.ProviderRef.IMAPEncryption
	}
	return "", 0, ""
}

// GetPOP3Config 获取 POP3 配置（从 Provider 获取）
func (a *EmailAccount) GetPOP3Config() (host string, port int, encryption string) {
	if a.ProviderRef != nil {
		return a.ProviderRef.POP3Host, a.ProviderRef.POP3Port, a.ProviderRef.POP3Encryption
	}
	return "", 0, ""
}

// GetSMTPConfig 获取 SMTP 配置（从 Provider 获取）
func (a *EmailAccount) GetSMTPConfig() (host string, port int, encryption string) {
	if a.ProviderRef != nil {
		return a.ProviderRef.SMTPHost, a.ProviderRef.SMTPPort, a.ProviderRef.SMTPEncryption
	}
	return "", 0, ""
}

// GetProviderName 获取提供商名称（从 Provider 获取）
func (a *EmailAccount) GetProviderName() string {
	if a.ProviderRef != nil {
		return a.ProviderRef.Name
	}
	return ""
}

// SyncMode 同步模式常量
const (
	SyncModePolling = "polling" // 轮询模式（默认）
	SyncModeWebhook = "webhook" // Webhook 推送模式
)

// GetSyncMode 获取账户的同步模式
// 优先使用数据库字段 SyncModeField，如果为空则返回默认值 "polling"
func (a *EmailAccount) GetSyncMode() string {
	if a.SyncModeField != "" {
		return a.SyncModeField
	}
	return SyncModePolling
}

// SetSyncMode 设置账户的同步模式
func (a *EmailAccount) SetSyncMode(mode string) {
	if mode == SyncModeWebhook || mode == SyncModePolling {
		a.SyncModeField = mode
	} else {
		a.SyncModeField = SyncModePolling
	}
}

// IsWebhookMode 检查账户是否使用 Webhook 模式
func (a *EmailAccount) IsWebhookMode() bool {
	return a.GetSyncMode() == SyncModeWebhook
}

// GetWebhookSecret 获取账户的 Webhook Secret
// 从 EncryptedCredentials 中解析 webhook_secret 字段
func (a *EmailAccount) GetWebhookSecret() string {
	if a.EncryptedCredentials == "" {
		return ""
	}

	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(a.EncryptedCredentials), &credentials); err != nil {
		return ""
	}

	if secret, ok := credentials["webhook_secret"].(string); ok {
		return secret
	}

	return ""
}
