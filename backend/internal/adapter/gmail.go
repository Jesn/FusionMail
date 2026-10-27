package adapter

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailAdapter Gmail API 适配器
type GmailAdapter struct {
	config       *Config
	service      *gmail.Service
	oauth2Config *oauth2.Config // OAuth2 配置，用于刷新 token
	httpClient   *http.Client   // HTTP 客户端
}

// NewGmailAdapter 创建 Gmail 适配器实例
func NewGmailAdapter(config *Config) (*GmailAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证 OAuth2 凭证
	if config.Credentials.AccessToken == "" {
		return nil, fmt.Errorf("access token is required for Gmail API")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &GmailAdapter{
		config: config,
	}, nil
}

// Connect 连接到 Gmail API
func (a *GmailAdapter) Connect(ctx context.Context) error {
	// 创建 OAuth2 token
	token := &oauth2.Token{
		AccessToken:  a.config.Credentials.AccessToken,
		RefreshToken: a.config.Credentials.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       a.config.Credentials.TokenExpiry,
	}

	// 创建 OAuth2 配置
	oauth2Config := &oauth2.Config{
		ClientID:     a.config.Credentials.ClientID,
		ClientSecret: a.config.Credentials.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			gmail.GmailReadonlyScope,
		},
	}

	// 保存 OAuth2 配置，用于后续刷新 token
	a.oauth2Config = oauth2Config

	// 创建 HTTP 客户端
	httpClient := oauth2Config.Client(ctx, token)

	// 如果配置了代理，设置代理
	if a.config.Proxy != nil && a.config.Proxy.Enabled {
		transport := httpClient.Transport.(*oauth2.Transport)
		transport.Base = &http.Transport{
			Proxy: http.ProxyURL(a.getProxyURL()),
		}
	}

	// 保存 HTTP 客户端
	a.httpClient = httpClient

	// 创建 Gmail 服务
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	a.service = service
	return nil
}

// getProxyURL 获取代理 URL
func (a *GmailAdapter) getProxyURL() *url.URL {
	if a.config.Proxy == nil || !a.config.Proxy.Enabled {
		return nil
	}

	proxyURL := fmt.Sprintf("%s://%s:%d",
		a.config.Proxy.Type,
		a.config.Proxy.Host,
		a.config.Proxy.Port,
	)

	if a.config.Proxy.Username != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%d",
			a.config.Proxy.Type,
			a.config.Proxy.Username,
			a.config.Proxy.Password,
			a.config.Proxy.Host,
			a.config.Proxy.Port,
		)
	}

	parsedURL, _ := url.Parse(proxyURL)
	return parsedURL
}

// Disconnect 断开连接
func (a *GmailAdapter) Disconnect() error {
	// Gmail API 是无状态的，不需要断开连接
	a.service = nil
	return nil
}

// GetProviderType 获取提供商类型
func (a *GmailAdapter) GetProviderType() string {
	return "gmail"
}

// GetProtocol 获取协议类型
func (a *GmailAdapter) GetProtocol() string {
	return "gmail_api"
}

// TestConnection 测试连接
func (a *GmailAdapter) TestConnection(ctx context.Context) error {
	if a.service == nil {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}

	// 测试获取用户配置文件
	_, err := a.service.Users.GetProfile("me").Do()
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	return nil
}

// MoveToTrash 将邮件移至垃圾箱
func (a *GmailAdapter) MoveToTrash(ctx context.Context, providerID string) error {
	if a.service == nil {
		return fmt.Errorf("Gmail service not initialized, call Connect first")
	}

	// 自动刷新 token（如果需要）
	if err := a.RefreshTokenIfNeeded(ctx); err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}

	// 调用 Gmail API 的 trash 方法
	_, err := a.service.Users.Messages.Trash("me", providerID).Do()
	if err != nil {
		// 处理 404：邮件不存在，视为幂等成功
		if strings.Contains(err.Error(), "404") {
			return nil
		}
		return fmt.Errorf("failed to trash message: %w", err)
	}

	return nil
}
