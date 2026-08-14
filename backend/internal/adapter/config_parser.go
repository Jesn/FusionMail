package adapter

import (
	"fmt"
	"strings"
	"time"
)

// AccountStringFormat 账户字符串格式常量
const (
	// QuickAccountSeparator 短效账户字符串分隔符
	QuickAccountSeparator = "----"

	// QuickAccountFieldCount 短效账户字段数量
	QuickAccountFieldCount = 4
)

// 支持的字段类型
const (
	FieldEmail        = "email"
	FieldPassword     = "password"
	FieldRefreshToken = "refresh_token"
	FieldClientID     = "client_id"
)

// DefaultImportFields 默认导入字段顺序
var DefaultImportFields = []string{FieldEmail, FieldPassword, FieldRefreshToken, FieldClientID}

// requiredImportFields 导入时必须包含的字段
var requiredImportFields = map[string]bool{
	FieldEmail:        true,
	FieldRefreshToken: true,
	FieldClientID:     true,
}

// validImportFields 所有合法的字段类型
var validImportFields = map[string]bool{
	FieldEmail:        true,
	FieldPassword:     true,
	FieldRefreshToken: true,
	FieldClientID:     true,
}

// ParseAccountStringWithFormat 按指定分隔符和字段顺序解析账户字符串
// fields 为字段顺序，如 []string{"email", "password", "refresh_token", "client_id"}
func ParseAccountStringWithFormat(accountString, delimiter string, fields []string) (*Config, error) {
	if accountString == "" {
		return nil, fmt.Errorf("account string is empty")
	}
	if delimiter == "" {
		return nil, fmt.Errorf("delimiter is empty")
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields is empty")
	}

	// 验证字段类型合法且无重复
	seen := make(map[string]bool, len(fields))
	for _, f := range fields {
		if !validImportFields[f] {
			return nil, fmt.Errorf("unknown field type: %s", f)
		}
		if seen[f] {
			return nil, fmt.Errorf("duplicate field: %s", f)
		}
		seen[f] = true
	}

	parts := strings.Split(accountString, delimiter)
	if len(parts) != len(fields) {
		return nil, fmt.Errorf("expected %d fields separated by '%s', got %d",
			len(fields), delimiter, len(parts))
	}

	// 按字段顺序映射值
	values := make(map[string]string, len(fields))
	for i, f := range fields {
		values[f] = strings.TrimSpace(parts[i])
	}

	// 验证必需字段
	for field := range requiredImportFields {
		if !seen[field] {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
		if values[field] == "" {
			return nil, fmt.Errorf("%s is required", field)
		}
	}

	email := values[FieldEmail]
	provider := inferProviderFromEmail(email)

	config := &Config{
		Email:    email,
		Provider: provider,
		AuthType: "quick",
		Credentials: &Credentials{
			Email:        email,
			Password:     values[FieldPassword],
			RefreshToken: values[FieldRefreshToken],
			ClientID:     values[FieldClientID],
		},
		Timeout: 30 * time.Second,
	}

	return config, nil
}

// ParseQuickAccountString 解析短效账户字符串
// 格式: email----password----refresh_token----client_id
// 返回解析后的配置对象
func ParseQuickAccountString(accountString string) (*Config, error) {
	return ParseAccountStringWithFormat(accountString, QuickAccountSeparator, DefaultImportFields)
}

// inferProviderFromEmail 从邮箱地址推断提供商
func inferProviderFromEmail(email string) string {
	if email == "" {
		return "generic"
	}

	// 提取域名部分
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "generic"
	}

	domain := strings.ToLower(parts[1])

	// 根据域名映射提供商
	providerMap := map[string]string{
		// Microsoft 系列
		"outlook.com":  "outlook",
		"hotmail.com":  "outlook",
		"live.com":     "outlook",
		"msn.com":      "outlook",
		"passport.com": "outlook",

		// Google 系列
		"gmail.com":      "gmail",
		"googlemail.com": "gmail",

		// Apple 系列
		"icloud.com": "icloud",
		"me.com":     "icloud",
		"mac.com":    "icloud",

		// 中国邮箱
		"qq.com":      "qq",
		"foxmail.com": "qq",
		"163.com":     "163",
		"126.com":     "163",
		"yeah.net":    "163",
	}

	if provider, exists := providerMap[domain]; exists {
		return provider
	}

	return "generic"
}

// FormatQuickAccountString 格式化短效账户字符串
// 将配置对象转换为账户字符串格式
func FormatQuickAccountString(config *Config) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config is nil")
	}

	if config.Credentials == nil {
		return "", fmt.Errorf("credentials is nil")
	}

	email := config.Email
	if email == "" && config.Credentials.Email != "" {
		email = config.Credentials.Email
	}

	if email == "" {
		return "", fmt.Errorf("email is required")
	}

	if config.Credentials.RefreshToken == "" {
		return "", fmt.Errorf("refresh token is required")
	}

	if config.Credentials.ClientID == "" {
		return "", fmt.Errorf("client ID is required")
	}

	// 构建账户字符串
	parts := []string{
		email,
		config.Credentials.Password, // 可能为空
		config.Credentials.RefreshToken,
		config.Credentials.ClientID,
	}

	return strings.Join(parts, QuickAccountSeparator), nil
}

// ValidateQuickAccountConfig 验证短效账户配置
func ValidateQuickAccountConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if config.Credentials == nil {
		return fmt.Errorf("credentials is nil")
	}

	// 验证邮箱地址格式
	if config.Email == "" {
		return fmt.Errorf("email is required")
	}

	if !strings.Contains(config.Email, "@") {
		return fmt.Errorf("invalid email format: %s", config.Email)
	}

	// 验证必需的认证信息
	if config.Credentials.RefreshToken == "" {
		return fmt.Errorf("refresh token is required for quick authentication")
	}

	if config.Credentials.ClientID == "" {
		return fmt.Errorf("client ID is required for quick authentication")
	}

	// 验证认证类型
	if config.AuthType != "quick" {
		return fmt.Errorf("auth type must be 'quick' for quick account config")
	}

	// 验证提供商支持
	supportedProviders := []string{"outlook", "gmail", "icloud", "qq", "163", "generic"}
	providerSupported := false
	for _, supported := range supportedProviders {
		if config.Provider == supported {
			providerSupported = true
			break
		}
	}

	if !providerSupported {
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	return nil
}

// ParseBatchQuickAccounts 批量解析短效账户字符串
// 输入: 账户字符串数组
// 输出: 配置对象数组和错误信息数组
func ParseBatchQuickAccounts(accountStrings []string) ([]*Config, []error) {
	configs := make([]*Config, 0, len(accountStrings))
	errors := make([]error, 0)

	for i, accountString := range accountStrings {
		config, err := ParseQuickAccountString(accountString)
		if err != nil {
			errors = append(errors, fmt.Errorf("account %d: %w", i+1, err))
			continue
		}

		// 验证配置
		if err := ValidateQuickAccountConfig(config); err != nil {
			errors = append(errors, fmt.Errorf("account %d validation: %w", i+1, err))
			continue
		}

		configs = append(configs, config)
	}

	return configs, errors
}

// QuickAccountInfo 短效账户信息结构
type QuickAccountInfo struct {
	Email        string `json:"email"`
	Provider     string `json:"provider"`
	ClientID     string `json:"client_id"`
	HasPassword  bool   `json:"has_password"`
	IsValid      bool   `json:"is_valid"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ExtractQuickAccountInfo 提取短效账户信息（不包含敏感数据）
func ExtractQuickAccountInfo(accountString string) *QuickAccountInfo {
	info := &QuickAccountInfo{
		IsValid: false,
	}

	config, err := ParseQuickAccountString(accountString)
	if err != nil {
		info.ErrorMessage = err.Error()
		return info
	}

	info.Email = config.Email
	info.Provider = config.Provider
	info.ClientID = config.Credentials.ClientID
	info.HasPassword = config.Credentials.Password != ""

	// 验证配置
	if err := ValidateQuickAccountConfig(config); err != nil {
		info.ErrorMessage = err.Error()
		return info
	}

	info.IsValid = true
	return info
}

// GetSupportedQuickProviders 获取支持短效认证的提供商列表
func GetSupportedQuickProviders() []string {
	return []string{
		"outlook", // Microsoft Graph API
		// 注意：目前只有 Outlook 支持短效认证
		// 其他提供商可能在未来版本中添加支持
	}
}

// IsQuickAuthSupported 检查提供商是否支持短效认证
func IsQuickAuthSupported(provider string) bool {
	supported := GetSupportedQuickProviders()
	for _, p := range supported {
		if p == provider {
			return true
		}
	}
	return false
}
