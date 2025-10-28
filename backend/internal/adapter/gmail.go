package adapter

import (
	"context"
	"fmt"
	"time"
)

// GmailAdapter Gmail API 适配器
type GmailAdapter struct {
	config *Config
	// TODO: 添加 Gmail API 客户端
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
	// TODO: 实现 Gmail API 连接逻辑
	return fmt.Errorf("Gmail adapter not implemented yet")
}

// Disconnect 断开连接
func (a *GmailAdapter) Disconnect() error {
	// Gmail API 是无状态的，不需要断开连接
	return nil
}

// FetchEmails 拉取邮件列表
func (a *GmailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	// TODO: 实现 Gmail API 邮件拉取逻辑
	return nil, fmt.Errorf("Gmail adapter not implemented yet")
}

// FetchEmailDetail 获取邮件详情
func (a *GmailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	// TODO: 实现 Gmail API 邮件详情获取逻辑
	return nil, fmt.Errorf("Gmail adapter not implemented yet")
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
	// TODO: 实现连接测试逻辑
	return fmt.Errorf("Gmail adapter not implemented yet")
}
