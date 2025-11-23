package model

import (
	"encoding/json"
	"time"
)

// Provider 邮箱提供商模型
// 将邮箱提供商配置信息存储在数据库中，支持动态管理
type Provider struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	// 基础信息
	Name string `gorm:"uniqueIndex;size:50;not null" json:"name"` // 提供商标识
	DisplayName string `gorm:"size:100;not null" json:"display_name"` // 显示名称
	ProviderType int `gorm:"index;not null;default:1" json:"provider_type"` // 提供商类型（枚举值）

	// 协议配置
	SupportedProtocols string `gorm:"type:text;not null" json:"-"` // 支持的协议（JSON数组）
	RecommendedProtocol string `gorm:"size:20;not null" json:"recommended_protocol"` // 推荐协议
	RequiresOAuth bool `gorm:"default:false" json:"requires_oauth"` // 是否强制OAuth

	// 服务器配置
	IMAPHost string `gorm:"size:255" json:"imap_host"` // IMAP服务器地址
	IMAPPort int `json:"imap_port"` // IMAP端口
	POP3Host string `gorm:"size:255" json:"pop3_host"` // POP3服务器地址
	POP3Port int `json:"pop3_port"` // POP3端口
	SMTPHost string `gorm:"size:255" json:"smtp_host"` // SMTP服务器地址（预留）
	SMTPPort int `json:"smmtp_port"` // SMTP端口（预留）

	// 管理字段
	Enabled bool `gorm:"default:true" json:"enabled"` // 是否启用
	SortOrder int `gorm:"default:0" json:"sort_order"` // 排序顺序
	Description string `gorm:"type:text" json:"description"` // 描述信息
	Metadata string `gorm:"type:text" json:"metadata,omitempty"` // JSON格式的额外配置

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderResponse 用于 API 响应的 Provider 结构
// 将 SupportedProtocols 从 JSON 字符串转换为数组
type ProviderResponse struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	DisplayName         string    `json:"display_name"`
	ProviderType        int       `json:"provider_type"` // 提供商类型
	SupportedProtocols  []string  `json:"supported_protocols"`
	RecommendedProtocol string    `json:"recommended_protocol"`
	RequiresOAuth       bool      `json:"requires_oauth"`
	IMAPHost            string    `json:"imap_host"`
	IMAPPort            int       `json:"imap_port"`
	POP3Host            string    `json:"pop3_host"`
	POP3Port            int       `json:"pop3_port"`
	SMTPHost            string    `json:"smtp_host"`
	SMTPPort            int       `json:"smtp_port"`
	Enabled             bool      `json:"enabled"`
	SortOrder           int       `json:"sort_order"`
	Description         string    `json:"description"`
	Metadata            string    `json:"metadata,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ToResponse 将 Provider 转换为 ProviderResponse
func (p *Provider) ToResponse() (*ProviderResponse, error) {
	protocols, err := p.GetSupportedProtocols()
	if err != nil {
		return nil, err
	}

	return &ProviderResponse{
		ID:                  p.ID,
		Name:                p.Name,
		DisplayName:         p.DisplayName,
		ProviderType:        p.ProviderType,
		SupportedProtocols:  protocols,
		RecommendedProtocol: p.RecommendedProtocol,
		RequiresOAuth:       p.RequiresOAuth,
		IMAPHost:            p.IMAPHost,
		IMAPPort:            p.IMAPPort,
		POP3Host:            p.POP3Host,
		POP3Port:            p.POP3Port,
		SMTPHost:            p.SMTPHost,
		SMTPPort:            p.SMTPPort,
		Enabled:             p.Enabled,
		SortOrder:           p.SortOrder,
		Description:         p.Description,
		Metadata:            p.Metadata,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}, nil
}

// TableName 指定表名
func (Provider) TableName() string {
	return "providers"
}

// UnmarshalJSON 自定义 JSON 反序列化
// 处理 supported_protocols 字段从数组到字符串的转换
func (p *Provider) UnmarshalJSON(data []byte) error {
	// 定义一个辅助结构体来接收 JSON 数据
	type Alias Provider
	aux := &struct {
		SupportedProtocols []string `json:"supported_protocols"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	// 解析 JSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 将协议数组转换为 JSON 字符串存储
	if len(aux.SupportedProtocols) > 0 {
		if err := p.SetSupportedProtocols(aux.SupportedProtocols); err != nil {
			return err
		}
	}

	return nil
}

// GetSupportedProtocols 获取支持的协议列表
// 将 JSON 格式的协议列表解析为字符串切片
func (p *Provider) GetSupportedProtocols() ([]string, error) {
	var protocols []string
	if p.SupportedProtocols == "" {
		return protocols, nil
	}

	err := json.Unmarshal([]byte(p.SupportedProtocols), &protocols)
	return protocols, err
}

// SetSupportedProtocols 设置支持的协议列表
// 将字符串切片序列化为 JSON 格式
func (p *Provider) SetSupportedProtocols(protocols []string) error {
	if protocols == nil {
		p.SupportedProtocols = "[]"
		return nil
	}

	data, err := json.Marshal(protocols)
	if err != nil {
		return err
	}
	p.SupportedProtocols = string(data)
	return nil
}

// GetMetadata 获取元数据
// 将 JSON 格式的元数据解析为 map
func (p *Provider) GetMetadata() (map[string]interface{}, error) {
	var metadata map[string]interface{}
	if p.Metadata == "" {
		return metadata, nil
	}

	err := json.Unmarshal([]byte(p.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
// 将 map 序列化为 JSON 格式
func (p *Provider) SetMetadata(metadata map[string]interface{}) error {
	if metadata == nil {
		p.Metadata = "{}"
		return nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	p.Metadata = string(data)
	return nil
}

// IsOAuth2Supported 检查是否支持 OAuth2 协议
func (p *Provider) IsOAuth2Supported() (bool, error) {
	protocols, err := p.GetSupportedProtocols()
	if err != nil {
		return false, err
	}

	for _, protocol := range protocols {
		if protocol == "oauth2" {
			return true, nil
		}
	}
	return false, nil
}

// IsIMAPSupported 检查是否支持 IMAP 协议
func (p *Provider) IsIMAPSupported() (bool, error) {
	protocols, err := p.GetSupportedProtocols()
	if err != nil {
		return false, err
	}

	for _, protocol := range protocols {
		if protocol == "imap" {
			return true, nil
		}
	}
	return false, nil
}

// IsPOP3Supported 检查是否支持 POP3 协议
func (p *Provider) IsPOP3Supported() (bool, error) {
	protocols, err := p.GetSupportedProtocols()
	if err != nil {
		return false, err
	}

	for _, protocol := range protocols {
		if protocol == "pop3" {
			return true, nil
		}
	}
	return false, nil
}

// Validate 验证配置的有效性
func (p *Provider) Validate() error {
	// 检查基础字段
	if p.Name == "" {
		return ErrValidation("name", "提供商名称不能为空")
	}

	if p.DisplayName == "" {
		return ErrValidation("display_name", "显示名称不能为空")
	}

	if p.RecommendedProtocol == "" {
		return ErrValidation("recommended_protocol", "推荐协议不能为空")
	}

	// 检查协议配置
	protocols, err := p.GetSupportedProtocols()
	if err != nil {
		return ErrValidation("supported_protocols", "协议列表格式错误: "+err.Error())
	}

	if len(protocols) == 0 {
		return ErrValidation("supported_protocols", "至少需要支持一种协议")
	}

	// 检查推荐协议是否在支持的协议中
	hasRecommended := false
	for _, protocol := range protocols {
		if protocol == p.RecommendedProtocol {
			hasRecommended = true
			break
		}
	}

	if !hasRecommended {
		return ErrValidation("recommended_protocol", "推荐协议必须在支持的协议列表中")
	}

	// 如果要求 OAuth2，检查是否支持
	if p.RequiresOAuth {
		supportsOAuth, err := p.IsOAuth2Supported()
		if err != nil {
			return err
		}
		if !supportsOAuth {
			return ErrValidation("requires_oauth", "设置了 requires_oauth=true，但不支持 oauth2 协议")
		}
	}

	// 检查服务器配置
	if p.IMAPPort < 0 || p.IMAPPort > 65535 {
		return ErrValidation("imap_port", "IMAP端口必须在 0-65535 范围内")
	}

	if p.POP3Port < 0 || p.POP3Port > 65535 {
		return ErrValidation("pop3_port", "POP3端口必须在 0-65535 范围内")
	}

	if p.SMTPPort < 0 || p.SMTPPort > 65535 {
		return ErrValidation("smtp_port", "SMTP端口必须在 0-65535 范围内")
	}

	return nil
}

// ErrValidation 创建验证错误
func ErrValidation(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ValidationError 验证错误类型
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error on field '" + e.Field + "': " + e.Message
}
