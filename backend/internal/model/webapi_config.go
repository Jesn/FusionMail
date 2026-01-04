package model

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
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

	// 同步模式配置
	SyncMode      string `json:"sync_mode,omitempty"`      // 同步模式：polling（轮询，默认）或 webhook（推送）
	WebhookSecret string `json:"webhook_secret,omitempty"` // Webhook Secret（webhook 模式必填）

	// Single 模式认证（两种方式二选一）
	// 方式 1：直接 JWT Token 登录（永不过期，推荐）
	JWTToken string `json:"jwt_token,omitempty"` // JWT Token（包含 address + address_id）
	Email    string `json:"email,omitempty"`     // 对应的邮箱地址

	// 方式 2：第三方授权登录（需要定期刷新）
	UserToken string `json:"user_token,omitempty"` // 用户 Token（包含 user_email + user_id + exp，有过期时间）

	// Admin 模式认证
	AdminPassword string `json:"admin_password,omitempty"` // 管理员密码（Admin 模式）
	Domains       string `json:"domains,omitempty"`        // 过滤域名列表（Admin 模式，逗号分隔，如 "example.com, test.org"）
}

// GetDomainList 解析并返回域名列表（去重、去空、转小写）
func (c *CloudflareTempEmailAuthData) GetDomainList() []string {
	if c.Domains == "" {
		return nil
	}

	parts := strings.Split(c.Domains, ",")
	seen := make(map[string]bool)
	var result []string

	for _, part := range parts {
		domain := strings.TrimSpace(strings.ToLower(part))
		if domain != "" && !seen[domain] {
			seen[domain] = true
			result = append(result, domain)
		}
	}

	return result
}

// HasDomainFilter 检查是否配置了域名过滤
func (c *CloudflareTempEmailAuthData) HasDomainFilter() bool {
	return len(c.GetDomainList()) > 0
}

// MatchesDomain 检查邮箱地址是否匹配配置的域名过滤
// 如果没有配置域名过滤，返回 true（不过滤）
// 如果配置了域名过滤，检查邮箱域名是否在列表中
func (c *CloudflareTempEmailAuthData) MatchesDomain(email string) bool {
	domains := c.GetDomainList()
	if len(domains) == 0 {
		return true // 没有配置过滤，全部通过
	}

	// 提取邮箱域名
	atIndex := strings.LastIndex(email, "@")
	if atIndex == -1 || atIndex == len(email)-1 {
		return false // 无效邮箱格式
	}
	emailDomain := strings.ToLower(email[atIndex+1:])

	// 检查是否匹配任一配置的域名
	for _, domain := range domains {
		if emailDomain == domain {
			return true
		}
	}

	return false
}

// Validate 验证 Cloudflare Temp Email 认证数据
func (c *CloudflareTempEmailAuthData) Validate() error {
	// Webhook 模式：只需要 webhook_secret
	if c.SyncMode == "webhook" {
		if c.WebhookSecret == "" {
			return errors.New("webhook 模式下 webhook_secret 不能为空")
		}
		return nil
	}

	// 轮询模式（默认）：需要 base_url 和认证信息
	// 规范化 URL：去除前后空格和末尾斜杠
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")

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
		// Single 模式：jwt_token 或 user_token 至少需要一个
		if c.JWTToken == "" && c.UserToken == "" {
			return errors.New("single 模式下 jwt_token 或 user_token 至少需要一个")
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

// IsWebhookMode 检查是否为 Webhook 模式
func (c *CloudflareTempEmailAuthData) IsWebhookMode() bool {
	return c.SyncMode == "webhook"
}

// IsPollingMode 检查是否为轮询模式（默认模式）
func (c *CloudflareTempEmailAuthData) IsPollingMode() bool {
	return c.SyncMode == "" || c.SyncMode == "polling"
}

// HasUserToken 检查是否配置了 user_token（第三方授权登录）
func (c *CloudflareTempEmailAuthData) HasUserToken() bool {
	return c.UserToken != ""
}

// GetUserTokenExpiry 解析 user_token 的过期时间
// 返回 Unix 时间戳，如果解析失败返回 0
func (c *CloudflareTempEmailAuthData) GetUserTokenExpiry() int64 {
	if c.UserToken == "" {
		return 0
	}
	return ParseJWTExpiry(c.UserToken)
}

// NeedsTokenRefresh 检查 user_token 是否需要刷新
// 当距离过期时间 ≤7 天时返回 true
func (c *CloudflareTempEmailAuthData) NeedsTokenRefresh() bool {
	if !c.HasUserToken() {
		return false
	}
	exp := c.GetUserTokenExpiry()
	if exp == 0 {
		return false // 无法解析过期时间，不刷新
	}
	// 距离过期 7 天内需要刷新
	return time.Until(time.Unix(exp, 0)) <= 7*24*time.Hour
}

// ParseJWTExpiry 解析 JWT Token 的 exp 字段
// 不验证签名，只读取 payload 中的 exp 字段
func ParseJWTExpiry(token string) int64 {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0
	}

	// Base64 URL 解码 payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// 尝试标准 Base64 解码（某些 JWT 可能使用标准编码）
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return 0
		}
	}

	// 解析 JSON 获取 exp 字段
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return 0
	}

	return claims.Exp
}

// ============================================
// Cloud Mail 认证数据
// ============================================

// CloudMailAuthData Cloud Mail 的认证数据
// 存储在 EmailAccount.AuthData 中
// 适配 mail.hema.edu.kg 等 Cloud Mail 服务
type CloudMailAuthData struct {
	BaseURL  string `json:"base_url"`           // API 基础 URL，如 https://mail.hema.edu.kg
	JWTToken string `json:"jwt_token"`          // JWT Token（从登录获取或手动输入）
	Email    string `json:"email,omitempty"`    // 登录邮箱（用于自动获取 Token）
	Password string `json:"password,omitempty"` // 登录密码（用于自动获取 Token）
}

// Validate 验证 Cloud Mail 认证数据
func (c *CloudMailAuthData) Validate() error {
	// 规范化 URL：去除前后空格和末尾斜杠
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")

	if c.BaseURL == "" {
		return errors.New("base_url 不能为空")
	}

	// 验证 URL 格式
	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("base_url 格式无效: %w", err)
	}

	// 必须提供 JWT Token 或者 邮箱+密码
	if c.JWTToken == "" && (c.Email == "" || c.Password == "") {
		return errors.New("请提供 JWT Token 或者 邮箱+密码")
	}

	return nil
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
	// 规范化 URL：去除前后空格和末尾斜杠
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")

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
