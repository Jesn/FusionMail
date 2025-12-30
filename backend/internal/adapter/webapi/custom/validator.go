package custom

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"fusionmail/internal/model"
)

// ConfigValidator 配置验证器
type ConfigValidator struct{}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate 验证自定义 WebAPI 配置
func (v *ConfigValidator) Validate(config *model.CustomWebAPIAuthData) error {
	if config == nil {
		return errors.New("配置不能为空")
	}

	// 验证基础配置
	if err := v.validateBaseConfig(config); err != nil {
		return err
	}

	// 验证认证配置
	if err := v.validateAuthConfig(config); err != nil {
		return err
	}

	// 验证端点配置
	if err := v.validateEndpointConfig(config); err != nil {
		return err
	}

	// 验证字段映射
	if err := v.validateFieldMapping(config); err != nil {
		return err
	}

	// 验证分页配置
	if err := v.validatePaginationConfig(config); err != nil {
		return err
	}

	return nil
}

// validateBaseConfig 验证基础配置
func (v *ConfigValidator) validateBaseConfig(config *model.CustomWebAPIAuthData) error {
	// 验证 BaseURL
	if config.BaseURL == "" {
		return errors.New("base_url 不能为空")
	}

	parsedURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return fmt.Errorf("base_url 格式无效: %w", err)
	}

	// 必须使用 HTTPS
	if parsedURL.Scheme != "https" {
		return errors.New("base_url 必须使用 HTTPS 协议")
	}

	// 验证主机名
	if parsedURL.Host == "" {
		return errors.New("base_url 缺少主机名")
	}

	// 不允许 localhost（生产环境）
	host := strings.ToLower(parsedURL.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		// 开发环境可以允许，这里只是警告
		// return errors.New("base_url 不允许使用 localhost")
	}

	// 验证服务名称
	if config.ServiceName == "" {
		return errors.New("service_name 不能为空")
	}

	if len(config.ServiceName) > 100 {
		return errors.New("service_name 长度不能超过 100 字符")
	}

	return nil
}

// validateAuthConfig 验证认证配置
func (v *ConfigValidator) validateAuthConfig(config *model.CustomWebAPIAuthData) error {
	auth := config.Auth

	// 验证认证类型
	validAuthTypes := map[string]bool{
		model.WebAPIAuthTypeJWT:    true,
		model.WebAPIAuthTypeBearer: true,
		model.WebAPIAuthTypeAPIKey: true,
		model.WebAPIAuthTypeBasic:  true,
		model.WebAPIAuthTypeCustom: true,
	}

	if !validAuthTypes[auth.Type] {
		return fmt.Errorf("不支持的认证类型: %s，支持的类型: jwt, bearer, apikey, basic, custom", auth.Type)
	}

	// 根据认证类型验证必填字段
	switch auth.Type {
	case model.WebAPIAuthTypeJWT, model.WebAPIAuthTypeBearer:
		if auth.Token == "" {
			return errors.New("jwt/bearer 认证需要提供 token")
		}
		if len(auth.Token) < 10 {
			return errors.New("token 长度过短，请检查是否正确")
		}

	case model.WebAPIAuthTypeAPIKey:
		if auth.APIKey == "" {
			return errors.New("apikey 认证需要提供 api_key")
		}
		// API Key Header 名称验证
		if auth.APIKeyName != "" {
			if !isValidHeaderName(auth.APIKeyName) {
				return fmt.Errorf("api_key_name 格式无效: %s", auth.APIKeyName)
			}
		}

	case model.WebAPIAuthTypeBasic:
		if auth.Username == "" {
			return errors.New("basic 认证需要提供 username")
		}
		if auth.Password == "" {
			return errors.New("basic 认证需要提供 password")
		}

	case model.WebAPIAuthTypeCustom:
		if auth.HeaderName == "" {
			return errors.New("custom 认证需要提供 header_name")
		}
		if auth.HeaderValue == "" {
			return errors.New("custom 认证需要提供 header_value")
		}
		if !isValidHeaderName(auth.HeaderName) {
			return fmt.Errorf("header_name 格式无效: %s", auth.HeaderName)
		}
	}

	return nil
}

// validateEndpointConfig 验证端点配置
func (v *ConfigValidator) validateEndpointConfig(config *model.CustomWebAPIAuthData) error {
	// 验证列表端点
	if config.ListEndpoint == "" {
		return errors.New("list_endpoint 不能为空")
	}

	// 端点必须以 / 开头
	if !strings.HasPrefix(config.ListEndpoint, "/") {
		return errors.New("list_endpoint 必须以 / 开头")
	}

	// 验证详情端点（可选）
	if config.DetailEndpoint != "" {
		if !strings.HasPrefix(config.DetailEndpoint, "/") {
			return errors.New("detail_endpoint 必须以 / 开头")
		}
	}

	return nil
}

// validateFieldMapping 验证字段映射
func (v *ConfigValidator) validateFieldMapping(config *model.CustomWebAPIAuthData) error {
	mapping := config.FieldMapping

	// 必填字段
	if mapping.ID == "" {
		return errors.New("field_mapping.id 不能为空")
	}

	if mapping.Subject == "" {
		return errors.New("field_mapping.subject 不能为空")
	}

	if mapping.From == "" {
		return errors.New("field_mapping.from 不能为空")
	}

	if mapping.Date == "" {
		return errors.New("field_mapping.date 不能为空")
	}

	// 验证路径格式
	paths := []string{
		mapping.ID,
		mapping.Subject,
		mapping.From,
		mapping.To,
		mapping.Date,
		mapping.Body,
		mapping.HTMLBody,
		mapping.RawContent,
		mapping.Attachments,
		mapping.IsRead,
		mapping.TargetEmail,
	}

	for _, path := range paths {
		if path != "" && !isValidJSONPath(path) {
			return fmt.Errorf("字段路径格式无效: %s", path)
		}
	}

	return nil
}

// validatePaginationConfig 验证分页配置
func (v *ConfigValidator) validatePaginationConfig(config *model.CustomWebAPIAuthData) error {
	if config.Pagination == nil {
		return nil // 分页配置可选
	}

	pagination := config.Pagination

	// 验证分页类型
	validTypes := map[string]bool{
		"offset":   true,
		"cursor":   true,
		"id_based": true,
		"page":     true,
	}

	if pagination.Type != "" && !validTypes[pagination.Type] {
		return fmt.Errorf("不支持的分页类型: %s，支持的类型: offset, cursor, id_based, page", pagination.Type)
	}

	// 验证页大小
	if pagination.PageSize < 0 {
		return errors.New("page_size 不能为负数")
	}

	if pagination.PageSize > 1000 {
		return errors.New("page_size 不能超过 1000")
	}

	return nil
}

// ValidateDataPath 验证数据路径
func (v *ConfigValidator) ValidateDataPath(dataPath string) error {
	if dataPath == "" {
		return nil // 空路径表示响应本身就是数组
	}

	if !isValidJSONPath(dataPath) {
		return fmt.Errorf("data_path 格式无效: %s", dataPath)
	}

	return nil
}

// ============================================
// 辅助函数
// ============================================

// isValidHeaderName 检查 HTTP Header 名称是否有效
func isValidHeaderName(name string) bool {
	if name == "" {
		return false
	}

	// Header 名称只能包含字母、数字和连字符
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}

	return true
}

// isValidJSONPath 检查 JSON 路径是否有效
func isValidJSONPath(path string) bool {
	if path == "" {
		return true
	}

	// 简单验证：只允许字母、数字、下划线、点和方括号
	for _, c := range path {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '[' || c == ']') {
			return false
		}
	}

	// 不能以点开头或结尾
	if strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return false
	}

	// 不能有连续的点
	if strings.Contains(path, "..") {
		return false
	}

	return true
}

// SanitizeConfig 清理配置（移除敏感信息用于日志）
func SanitizeConfig(config *model.CustomWebAPIAuthData) map[string]interface{} {
	if config == nil {
		return nil
	}

	return map[string]interface{}{
		"base_url":        config.BaseURL,
		"service_name":    config.ServiceName,
		"auth_type":       config.Auth.Type,
		"list_endpoint":   config.ListEndpoint,
		"detail_endpoint": config.DetailEndpoint,
		"data_path":       config.DataPath,
		"target_email":    maskEmail(config.TargetEmail),
		"has_pagination":  config.Pagination != nil,
	}
}

// maskEmail 遮蔽邮箱地址
func maskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return local + "***@" + domain
	}

	return local[:2] + "***@" + domain
}

// ValidateAndSanitize 验证并清理配置
func ValidateAndSanitize(config *model.CustomWebAPIAuthData) (map[string]interface{}, error) {
	validator := NewConfigValidator()
	if err := validator.Validate(config); err != nil {
		return nil, err
	}
	return SanitizeConfig(config), nil
}
