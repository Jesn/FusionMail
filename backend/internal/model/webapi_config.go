package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// WebAPI 服务类型常量
const (
	WebAPIServiceTypeCloudflareTempEmail = "cloudflare_temp_email" // Cloudflare Temp Email 服务
	WebAPIServiceTypeCloudMail           = "cloud_mail"            // Cloud Mail 服务
	WebAPIServiceTypeCustom              = "custom"                // 自定义 Web API 服务
)

// WebAPI 访问模式常量
const (
	WebAPIAccessModeSingle = "single" // 单邮箱模式（JWT Token 对应单个邮箱）
	WebAPIAccessModeAdmin  = "admin"  // 管理员模式（可访问域名下所有邮箱）
)

// WebAPI 认证类型常量（用于自定义服务）
const (
	WebAPIAuthTypeJWT    = "jwt"    // JWT Token 认证
	WebAPIAuthTypeBearer = "bearer" // Bearer Token 认证
	WebAPIAuthTypeAPIKey = "apikey" // API Key 认证
	WebAPIAuthTypeBasic  = "basic"  // Basic Auth 认证
	WebAPIAuthTypeCustom = "custom" // 自定义 Header 认证
)

// WebAPIProviderConfig 存储在 Provider.Metadata 中的 WebAPI 配置
type WebAPIProviderConfig struct {
	ServiceType string   `json:"service_type"`           // 服务类型：cloudflare_temp_email, cloud_mail, custom
	AccessModes []string `json:"access_modes,omitempty"` // 支持的访问模式
	APIDocs     string   `json:"api_docs,omitempty"`     // API 文档链接
}

// ParseWebAPIProviderConfig 从 Provider.Metadata 解析 WebAPI 配置
func ParseWebAPIProviderConfig(metadata string) (*WebAPIProviderConfig, error) {
	if metadata == "" {
		return nil, errors.New("metadata 为空")
	}

	var config WebAPIProviderConfig
	if err := json.Unmarshal([]byte(metadata), &config); err != nil {
		return nil, fmt.Errorf("解析 WebAPI 配置失败: %w", err)
	}

	return &config, nil
}

// ============================================
// Cloudflare Temp Email 认证数据
// ============================================

// CloudflareTempEmailAuthData Cloudflare Temp Email 的认证数据
// 存储在 EmailAccount.AuthData 中
type CloudflareTempEmailAuthData struct {
	// 基础配置
	BaseURL    string `json:"base_url"`    // API 基础 URL，如 https://temp-email.example.com
	AccessMode string `json:"access_mode"` // 访问模式：single 或 admin

	// Single 模式认证
	JWTToken string `json:"jwt_token,omitempty"` // JWT Token（Single 模式）
	Email    string `json:"email,omitempty"`     // 对应的邮箱地址（Single 模式）

	// Admin 模式认证
	AdminPassword string `json:"admin_password,omitempty"` // 管理员密码（Admin 模式）
	Domain        string `json:"domain,omitempty"`         // 管理的域名（Admin 模式）
}

// Validate 验证 Cloudflare Temp Email 认证数据
func (c *CloudflareTempEmailAuthData) Validate() error {
	if c.BaseURL == "" {
		return errors.New("base_url 不能为空")
	}

	// 验证 URL 格式
	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("base_url 格式无效: %w", err)
	}

	// 验证访问模式
	if c.AccessMode != WebAPIAccessModeSingle && c.AccessMode != WebAPIAccessModeAdmin {
		return fmt.Errorf("access_mode 必须是 %s 或 %s", WebAPIAccessModeSingle, WebAPIAccessModeAdmin)
	}

	// 根据访问模式验证必填字段
	if c.AccessMode == WebAPIAccessModeSingle {
		if c.JWTToken == "" {
			return errors.New("single 模式下 jwt_token 不能为空")
		}
	} else if c.AccessMode == WebAPIAccessModeAdmin {
		if c.AdminPassword == "" {
			return errors.New("admin 模式下 admin_password 不能为空")
		}
	}

	return nil
}

// IsSingleMode 检查是否为 Single 模式
func (c *CloudflareTempEmailAuthData) IsSingleMode() bool {
	return c.AccessMode == WebAPIAccessModeSingle
}

// IsAdminMode 检查是否为 Admin 模式
func (c *CloudflareTempEmailAuthData) IsAdminMode() bool {
	return c.AccessMode == WebAPIAccessModeAdmin
}

// ============================================
// Cloud Mail 认证数据
// ============================================

// CloudMailAccount Cloud Mail 中的单个账户配置
type CloudMailAccount struct {
	Email    string `json:"email"`    // 邮箱地址
	Password string `json:"password"` // 账户密码（用于获取 JWT）
}

// CloudMailAuthData Cloud Mail 的认证数据
// 存储在 EmailAccount.AuthData 中
type CloudMailAuthData struct {
	BaseURL  string             `json:"base_url"`  // API 基础 URL
	JWTToken string             `json:"jwt_token"` // 主 JWT Token（用于 API 认证）
	Accounts []CloudMailAccount `json:"accounts"`  // 账户列表
}

// Validate 验证 Cloud Mail 认证数据
func (c *CloudMailAuthData) Validate() error {
	if c.BaseURL == "" {
		return errors.New("base_url 不能为空")
	}

	// 验证 URL 格式
	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("base_url 格式无效: %w", err)
	}

	if c.JWTToken == "" {
		return errors.New("jwt_token 不能为空")
	}

	if len(c.Accounts) == 0 {
		return errors.New("至少需要配置一个账户")
	}

	// 验证每个账户
	for i, acc := range c.Accounts {
		if acc.Email == "" {
			return fmt.Errorf("账户 %d 的 email 不能为空", i+1)
		}
	}

	return nil
}

// GetAccountEmails 获取所有账户的邮箱地址列表
func (c *CloudMailAuthData) GetAccountEmails() []string {
	emails := make([]string, len(c.Accounts))
	for i, acc := range c.Accounts {
		emails[i] = acc.Email
	}
	return emails
}

// ============================================
// 自定义 Web API 认证数据
// ============================================

// CustomWebAPIFieldMapping 自定义 API 的字段映射配置
type CustomWebAPIFieldMapping struct {
	ID          string `json:"id"`                     // 邮件 ID 字段路径
	Subject     string `json:"subject"`                // 主题字段路径
	From        string `json:"from"`                   // 发件人字段路径
	To          string `json:"to"`                     // 收件人字段路径
	Date        string `json:"date"`                   // 日期字段路径
	Body        string `json:"body,omitempty"`         // 正文字段路径（可选）
	HTMLBody    string `json:"html_body,omitempty"`    // HTML 正文字段路径（可选）
	RawContent  string `json:"raw_content,omitempty"`  // RFC822 原始内容字段路径（可选）
	Attachments string `json:"attachments,omitempty"`  // 附件字段路径（可选）
	IsRead      string `json:"is_read,omitempty"`      // 已读状态字段路径（可选）
	TargetEmail string `json:"target_email,omitempty"` // 目标邮箱字段路径（用于 Admin 模式）
}

// CustomWebAPIPagination 自定义 API 的分页配置
type CustomWebAPIPagination struct {
	Type       string `json:"type"`                  // 分页类型：offset, cursor, id_based
	PageParam  string `json:"page_param,omitempty"`  // 页码参数名
	LimitParam string `json:"limit_param,omitempty"` // 每页数量参数名
	PageSize   int    `json:"page_size,omitempty"`   // 每页数量，默认 50
}

// CustomWebAPIAuthConfig 自定义 API 的认证配置
type CustomWebAPIAuthConfig struct {
	Type        string `json:"type"`                   // 认证类型：jwt, bearer, apikey, basic, custom
	Token       string `json:"token,omitempty"`        // Token 值（jwt/bearer）
	APIKey      string `json:"api_key,omitempty"`      // API Key 值
	APIKeyName  string `json:"api_key_name,omitempty"` // API Key 的 Header 名称，默认 X-API-Key
	Username    string `json:"username,omitempty"`     // Basic Auth 用户名
	Password    string `json:"password,omitempty"`     // Basic Auth 密码
	HeaderName  string `json:"header_name,omitempty"`  // 自定义 Header 名称
	HeaderValue string `json:"header_value,omitempty"` // 自定义 Header 值
}

// CustomWebAPIAuthData 自定义 Web API 的完整配置
// 存储在 EmailAccount.AuthData 中
type CustomWebAPIAuthData struct {
	// 基础配置
	BaseURL     string `json:"base_url"`     // API 基础 URL
	ServiceName string `json:"service_name"` // 服务名称（用于显示）

	// 认证配置
	Auth CustomWebAPIAuthConfig `json:"auth"`

	// API 端点配置
	ListEndpoint   string `json:"list_endpoint"`             // 邮件列表端点，如 /api/mails
	DetailEndpoint string `json:"detail_endpoint,omitempty"` // 邮件详情端点（可选）

	// 响应解析配置
	DataPath     string                   `json:"data_path"`     // 数据路径，如 data.list
	FieldMapping CustomWebAPIFieldMapping `json:"field_mapping"` // 字段映射

	// 分页配置
	Pagination *CustomWebAPIPagination `json:"pagination,omitempty"`

	// 目标邮箱（用于 Single 模式）
	TargetEmail string `json:"target_email,omitempty"`
}

// Validate 验证自定义 Web API 配置
func (c *CustomWebAPIAuthData) Validate() error {
	if c.BaseURL == "" {
		return errors.New("base_url 不能为空")
	}

	// 验证 URL 格式，必须是 HTTPS
	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("base_url 格式无效: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return errors.New("base_url 必须使用 HTTPS 协议")
	}

	if c.ServiceName == "" {
		return errors.New("service_name 不能为空")
	}

	if c.ListEndpoint == "" {
		return errors.New("list_endpoint 不能为空")
	}

	// 验证认证配置
	if err := c.validateAuth(); err != nil {
		return err
	}

	// 验证字段映射
	if err := c.validateFieldMapping(); err != nil {
		return err
	}

	return nil
}

// validateAuth 验证认证配置
func (c *CustomWebAPIAuthData) validateAuth() error {
	validAuthTypes := map[string]bool{
		WebAPIAuthTypeJWT:    true,
		WebAPIAuthTypeBearer: true,
		WebAPIAuthTypeAPIKey: true,
		WebAPIAuthTypeBasic:  true,
		WebAPIAuthTypeCustom: true,
	}

	if !validAuthTypes[c.Auth.Type] {
		return fmt.Errorf("不支持的认证类型: %s", c.Auth.Type)
	}

	switch c.Auth.Type {
	case WebAPIAuthTypeJWT, WebAPIAuthTypeBearer:
		if c.Auth.Token == "" {
			return errors.New("jwt/bearer 认证需要提供 token")
		}
	case WebAPIAuthTypeAPIKey:
		if c.Auth.APIKey == "" {
			return errors.New("apikey 认证需要提供 api_key")
		}
	case WebAPIAuthTypeBasic:
		if c.Auth.Username == "" || c.Auth.Password == "" {
			return errors.New("basic 认证需要提供 username 和 password")
		}
	case WebAPIAuthTypeCustom:
		if c.Auth.HeaderName == "" || c.Auth.HeaderValue == "" {
			return errors.New("custom 认证需要提供 header_name 和 header_value")
		}
	}

	return nil
}

// validateFieldMapping 验证字段映射配置
func (c *CustomWebAPIAuthData) validateFieldMapping() error {
	// 必填字段
	if c.FieldMapping.ID == "" {
		return errors.New("field_mapping.id 不能为空")
	}
	if c.FieldMapping.Subject == "" {
		return errors.New("field_mapping.subject 不能为空")
	}
	if c.FieldMapping.From == "" {
		return errors.New("field_mapping.from 不能为空")
	}
	if c.FieldMapping.Date == "" {
		return errors.New("field_mapping.date 不能为空")
	}

	return nil
}

// ============================================
// 通用解析函数
// ============================================

// ParseWebAPIAuthData 根据服务类型解析认证数据
func ParseWebAPIAuthData(serviceType string, authDataJSON string) (interface{}, error) {
	if authDataJSON == "" {
		return nil, errors.New("auth_data 为空")
	}

	switch serviceType {
	case WebAPIServiceTypeCloudflareTempEmail:
		var data CloudflareTempEmailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &data); err != nil {
			return nil, fmt.Errorf("解析 Cloudflare Temp Email 认证数据失败: %w", err)
		}
		return &data, nil

	case WebAPIServiceTypeCloudMail:
		var data CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &data); err != nil {
			return nil, fmt.Errorf("解析 Cloud Mail 认证数据失败: %w", err)
		}
		return &data, nil

	case WebAPIServiceTypeCustom:
		var data CustomWebAPIAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &data); err != nil {
			return nil, fmt.Errorf("解析自定义 Web API 认证数据失败: %w", err)
		}
		return &data, nil

	default:
		return nil, fmt.Errorf("不支持的服务类型: %s", serviceType)
	}
}

// IsWebAPIProvider 检查 Provider 是否为 WebAPI 类型
func IsWebAPIProvider(provider *Provider) bool {
	if provider == nil {
		return false
	}

	protocols, err := provider.GetSupportedProtocols()
	if err != nil {
		return false
	}

	for _, p := range protocols {
		if strings.ToLower(p) == "webapi" {
			return true
		}
	}

	return false
}

// GetWebAPIServiceType 从 Provider 获取 WebAPI 服务类型
func GetWebAPIServiceType(provider *Provider) (string, error) {
	if provider == nil {
		return "", errors.New("provider 为空")
	}

	config, err := ParseWebAPIProviderConfig(provider.Metadata)
	if err != nil {
		return "", err
	}

	return config.ServiceType, nil
}
