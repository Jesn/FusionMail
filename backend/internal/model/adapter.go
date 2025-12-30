package model

import (
	"time"
)

// Adapter 邮箱协议适配器模型
// 管理不同的邮箱协议实现（Gmail API、Microsoft Graph、IMAP 等）
type Adapter struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:50;not null" json:"name"` // 适配器唯一标识 (gmail/graph/imap)
	DisplayName string    `gorm:"size:100;not null" json:"display_name"`    // 显示名称
	AuthType    string    `gorm:"size:20;not null" json:"auth_type"`        // 认证类型 (oauth2/password)
	Description string    `gorm:"type:text" json:"description"`             // 描述信息
	IsEnabled   bool      `gorm:"default:true" json:"is_enabled"`           // 是否启用
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Adapter) TableName() string {
	return "adapters"
}

// AdapterAuthType 认证类型常量
const (
	AdapterAuthTypeOAuth2   = "oauth2"   // OAuth2 授权
	AdapterAuthTypePassword = "password" // 密码/授权码
	AdapterAuthTypeToken    = "token"    // Token 认证（用于 WebAPI 适配器）
)

// AdapterName 适配器名称常量
const (
	AdapterNameGmail  = "gmail"  // Gmail API 适配器
	AdapterNameGraph  = "graph"  // Microsoft Graph 适配器
	AdapterNameIMAP   = "imap"   // 通用 IMAP 适配器
	AdapterNameWebAPI = "webapi" // 通用 Web API 适配器
)

// IsOAuth2 检查是否为 OAuth2 认证类型
func (a *Adapter) IsOAuth2() bool {
	return a.AuthType == AdapterAuthTypeOAuth2
}

// IsPassword 检查是否为密码认证类型
func (a *Adapter) IsPassword() bool {
	return a.AuthType == AdapterAuthTypePassword
}

// IsToken 检查是否为 Token 认证类型
func (a *Adapter) IsToken() bool {
	return a.AuthType == AdapterAuthTypeToken
}

// Validate 验证适配器配置的有效性
func (a *Adapter) Validate() error {
	if a.Name == "" {
		return ErrValidation("name", "适配器名称不能为空")
	}

	if a.DisplayName == "" {
		return ErrValidation("display_name", "显示名称不能为空")
	}

	if a.AuthType == "" {
		return ErrValidation("auth_type", "认证类型不能为空")
	}

	// 验证认证类型
	validAuthTypes := map[string]bool{
		AdapterAuthTypeOAuth2:   true,
		AdapterAuthTypePassword: true,
		AdapterAuthTypeToken:    true,
	}
	if !validAuthTypes[a.AuthType] {
		return ErrValidation("auth_type", "认证类型必须是 oauth2、password 或 token")
	}

	return nil
}

// AdapterResponse 用于 API 响应的 Adapter 结构
type AdapterResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	AuthType    string    `json:"auth_type"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse 将 Adapter 转换为 AdapterResponse
func (a *Adapter) ToResponse() *AdapterResponse {
	return &AdapterResponse{
		ID:          a.ID,
		Name:        a.Name,
		DisplayName: a.DisplayName,
		AuthType:    a.AuthType,
		Description: a.Description,
		IsEnabled:   a.IsEnabled,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
