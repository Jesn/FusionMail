package adapter

import (
	"context"
	"fmt"
	"time"
)

// GraphAdapter Microsoft Graph API 适配器
type GraphAdapter struct {
	config *Config
	// TODO: 添加 Graph API 客户端
}

// NewGraphAdapter 创建 Graph 适配器实例
func NewGraphAdapter(config *Config) (*GraphAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证 OAuth2 凭证
	if config.Credentials.AccessToken == "" {
		return nil, fmt.Errorf("access token is required for Microsoft Graph API")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &GraphAdapter{
		config: config,
	}, nil
}

// Connect 连接到 Microsoft Graph API
func (a *GraphAdapter) Connect(ctx context.Context) error {
	// TODO: 实现 Graph API 连接逻辑
	return fmt.Errorf("Graph adapter not implemented yet")
}

// Disconnect 断开连接
func (a *GraphAdapter) Disconnect() error {
	// Graph API 是无状态的，不需要断开连接
	return nil
}

// FetchEmails 拉取邮件列表
func (a *GraphAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	// TODO: 实现 Graph API 邮件拉取逻辑
	return nil, fmt.Errorf("Graph adapter not implemented yet")
}

// FetchEmailDetail 获取邮件详情
func (a *GraphAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	// TODO: 实现 Graph API 邮件详情获取逻辑
	return nil, fmt.Errorf("Graph adapter not implemented yet")
}

// GetProviderType 获取提供商类型
func (a *GraphAdapter) GetProviderType() string {
	return "outlook"
}

// GetProtocol 获取协议类型
func (a *GraphAdapter) GetProtocol() string {
	return "graph"
}

// TestConnection 测试连接
func (a *GraphAdapter) TestConnection(ctx context.Context) error {
	// TODO: 实现连接测试逻辑
	return fmt.Errorf("Graph adapter not implemented yet")
}
