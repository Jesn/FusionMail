package adapter

// GraphQuickAdapter - Microsoft Graph API 短效适配器
//
// 重要说明：
// 本适配器的核心实现必须与 backend/micro.py 参考实现保持完全一致。
// 这是确保短效邮箱能够正确接收邮件的关键要求。
//
// 核心流程（必须与 micro.py 一致）：
// 1. Token 刷新：POST https://login.microsoftonline.com/common/oauth2/v2.0/token
//    - 参数：client_id, grant_type=refresh_token, refresh_token, scope=https://graph.microsoft.com/.default
// 2. 邮件获取：GET https://graph.microsoft.com/v1.0/me/mailFolders/inbox/messages
//    - 头部：Authorization: Bearer {access_token}
//
// 扩展功能（在核心流程之上构建）：
// - FetchEmailsWithFilter：使用过滤条件获取邮件
// - FetchEmailsWithPagination：分页获取邮件
// - FetchEmailsByFolder：按文件夹获取邮件
// - 其他辅助方法
//
// 修改核心流程时，必须确保与 micro.py 的行为完全一致。

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Logger 日志接口
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// SimpleLogger 简单的日志实现
type SimpleLogger struct {
	prefix string
}

func NewSimpleLogger(prefix string) *SimpleLogger {
	return &SimpleLogger{prefix: prefix}
}

func (l *SimpleLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[INFO] %s: %s %v\n", l.prefix, msg, keysAndValues)
}

func (l *SimpleLogger) Warn(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[WARN] %s: %s %v\n", l.prefix, msg, keysAndValues)
}

func (l *SimpleLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[ERROR] %s: %s %v\n", l.prefix, msg, keysAndValues)
}

// GraphQuickAdapter Microsoft Graph API 短效适配器
// 使用简化的认证流程，直接通过 refresh_token 获取 access_token
// 适用于批量导入、测试验证等场景
type GraphQuickAdapter struct {
	config      *Config
	httpClient  *http.Client
	baseURL     string
	tokenURL    string // OAuth2 token 端点
	accessToken string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex
	logger      Logger
}

// QuickAuthConfig 短效认证配置
type QuickAuthConfig struct {
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	AuthType     string `json:"auth_type"`
}

// TokenResponse Microsoft OAuth2 token 响应
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// TokenError token 相关错误
type TokenError struct {
	Code        string
	Description string
	StatusCode  int
}

func (e *TokenError) Error() string {
	return fmt.Sprintf("token error %s: %s (status: %d)", e.Code, e.Description, e.StatusCode)
}

// NewGraphQuickAdapter 创建短效适配器实例
func NewGraphQuickAdapter(config *Config) (*GraphQuickAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证短效认证所需的参数
	if config.Credentials.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token is required for quick adapter")
	}

	if config.Credentials.ClientID == "" {
		return nil, fmt.Errorf("client ID is required for quick adapter")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// 默认 URL，可以通过配置覆盖（用于测试）
	baseURL := "https://graph.microsoft.com/v1.0"
	tokenURL := "https://login.microsoftonline.com/common/oauth2/v2.0/token"

	// 支持测试时自定义 URL
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}

	return &GraphQuickAdapter{
		config:     config,
		baseURL:    baseURL,
		tokenURL:   tokenURL,
		httpClient: &http.Client{Timeout: config.Timeout},
		logger:     NewSimpleLogger("GraphQuickAdapter"),
	}, nil
}

// Connect 连接到 Microsoft Graph API
func (a *GraphQuickAdapter) Connect(ctx context.Context) error {
	// 获取 access token
	if err := a.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	return nil
}

// Disconnect 断开连接
func (a *GraphQuickAdapter) Disconnect() error {
	// 清理敏感信息
	a.accessToken = ""
	a.tokenExpiry = time.Time{}
	a.httpClient = nil
	return nil
}

// GetProviderType 获取提供商类型
func (a *GraphQuickAdapter) GetProviderType() string {
	return "outlook"
}

// GetProtocol 获取协议类型
func (a *GraphQuickAdapter) GetProtocol() string {
	return "graph_quick"
}
