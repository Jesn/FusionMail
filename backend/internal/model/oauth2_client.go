package model

import (
	"context"
	"encoding/json"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// OAuth2Client OAuth2 客户端配置
type OAuth2Client struct {
	ID                      int64      `gorm:"primaryKey" json:"id"`
	ProviderID              int64      `gorm:"not null;index" json:"provider_id"` // 提供商ID
	Name                    string     `gorm:"size:100;not null" json:"name"`
	ClientID                string     `gorm:"size:255;not null" json:"client_id"`
	ClientSecretEncrypted   string     `gorm:"type:text;not null" json:"-"`
	RedirectURI             string     `gorm:"size:500;not null" json:"redirect_uri"`
	Enabled                 bool       `gorm:"default:true;index" json:"enabled"`
	IsDefault               bool       `gorm:"default:false;index" json:"is_default"`
	UsageCount              int        `gorm:"default:0" json:"usage_count"`
	QuotaDaily              int        `gorm:"default:-1" json:"quota_daily"`
	QuotaMonthly            int        `gorm:"default:-1" json:"quota_monthly"`
	LastUsedAt              *time.Time `gorm:"index" json:"last_used_at"`
	Metadata                string     `gorm:"type:text" json:"metadata"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`

	// 关联关系
	Provider *Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

// OAuth2ClientCreateRequest 创建请求
type OAuth2ClientCreateRequest struct {
	ProviderID    int64  `json:"provider_id" binding:"required"` // 提供商ID
	Name          string `json:"name" binding:"required,max=100"`
	ClientID      string `json:"client_id" binding:"required,max=255"`
	ClientSecret  string `json:"client_secret" binding:"required"`
	RedirectURI   string `json:"redirect_uri" binding:"required,url"`
	QuotaDaily    int    `json:"quota_daily" binding:"omitempty,min=-1"`
	QuotaMonthly  int    `json:"quota_monthly" binding:"omitempty,min=-1"`
	Metadata      string `json:"metadata"`
}

// OAuth2ClientUpdateRequest 更新请求
type OAuth2ClientUpdateRequest struct {
	ProviderID    *int64 `json:"provider_id"` // 提供商ID（可选）
	Name          string `json:"name" binding:"omitempty,max=100"`
	ClientID      string `json:"client_id" binding:"omitempty,max=255"`
	ClientSecret  string `json:"client_secret" binding:"omitempty"`
	RedirectURI   string `json:"redirect_uri" binding:"omitempty,url"`
	Enabled       *bool  `json:"enabled"`
	QuotaDaily    *int   `json:"quota_daily" binding:"omitempty,min=-1"`
	QuotaMonthly  *int   `json:"quota_monthly" binding:"omitempty,min=-1"`
	Metadata      string `json:"metadata"`
}

// OAuth2ClientResponse 响应
type OAuth2ClientResponse struct {
	ID            int64      `json:"id"`
	ProviderID    int64      `json:"provider_id"`
	ProviderName  string     `json:"provider_name,omitempty"` // 提供商名称（通过关联获取）
	Name          string     `json:"name"`
	ClientID      string     `json:"client_id"`
	RedirectURI   string     `json:"redirect_uri"`
	Enabled       bool       `json:"enabled"`
	IsDefault     bool       `json:"is_default"`
	UsageCount    int        `json:"usage_count"`
	QuotaDaily    int        `json:"quota_daily"`
	QuotaMonthly  int        `json:"quota_monthly"`
	LastUsedAt    *time.Time `json:"last_used_at"`
	Metadata      string     `json:"metadata"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ToResponse 转换为响应
func (c *OAuth2Client) ToResponse() *OAuth2ClientResponse {
	providerName := ""
	if c.Provider != nil {
		providerName = c.Provider.Name
	}

	return &OAuth2ClientResponse{
		ID:            c.ID,
		ProviderID:    c.ProviderID,
		ProviderName:  providerName,
		Name:          c.Name,
		ClientID:      c.ClientID,
		RedirectURI:   c.RedirectURI,
		Enabled:       c.Enabled,
		IsDefault:     c.IsDefault,
		UsageCount:    c.UsageCount,
		QuotaDaily:    c.QuotaDaily,
		QuotaMonthly:  c.QuotaMonthly,
		LastUsedAt:    c.LastUsedAt,
		Metadata:      c.Metadata,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// Validate 验证模型
func (c *OAuth2Client) Validate() error {
	// 验证必要字段
	if c.ProviderID == 0 {
		return ErrValidation("provider_id", "provider_id is required")
	}
	if c.Name == "" {
		return ErrValidation("name", "name is required")
	}
	if c.ClientID == "" {
		return ErrValidation("client_id", "client_id is required")
	}
	if c.RedirectURI == "" {
		return ErrValidation("redirect_uri", "redirect_uri is required")
	}
	return nil
}

// SetClientSecret 加密设置客户端密钥
func (c *OAuth2Client) SetClientSecret(secret string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	c.ClientSecretEncrypted = string(hashed)
	return nil
}

// VerifyClientSecret 验证客户端密钥
func (c *OAuth2Client) VerifyClientSecret(secret string) error {
	return bcrypt.CompareHashAndPassword([]byte(c.ClientSecretEncrypted), []byte(secret))
}

// GetMetadata 获取元数据
func (c *OAuth2Client) GetMetadata() (map[string]interface{}, error) {
	if c.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(c.Metadata), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetMetadata 设置元数据
func (c *OAuth2Client) SetMetadata(data map[string]interface{}) error {
	if data == nil {
		c.Metadata = ""
		return nil
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.Metadata = string(jsonData)
	return nil
}

// CanUse 检查是否可以继续使用（配额检查）
func (c *OAuth2Client) CanUse() bool {
	if !c.Enabled {
		return false
	}
	// 检查日配额
	if c.QuotaDaily >= 0 && c.UsageCount >= c.QuotaDaily {
		return false
	}
	// 检查月配额
	if c.QuotaMonthly >= 0 && c.UsageCount >= c.QuotaMonthly {
		return false
	}
	return true
}

// IncrementUsage 增加使用计数
func (c *OAuth2Client) IncrementUsage(ctx context.Context) {
	c.UsageCount++
	now := time.Now()
	c.LastUsedAt = &now
}
