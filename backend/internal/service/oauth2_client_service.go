package service

import (
	"context"
	"fmt"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// OAuth2ClientService OAuth2 客户端业务逻辑
type OAuth2ClientService struct {
	repo repository.OAuth2ClientRepository
}

// NewOAuth2ClientService 创建 OAuth2 客户端服务
func NewOAuth2ClientService(repo repository.OAuth2ClientRepository) *OAuth2ClientService {
	return &OAuth2ClientService{
		repo: repo,
	}
}

// Create 创建新的 OAuth2 客户端配置
func (s *OAuth2ClientService) Create(ctx context.Context, req *model.OAuth2ClientCreateRequest) (*model.OAuth2Client, error) {
	// 创建模型实例 - 使用ProviderID
	client := &model.OAuth2Client{
		ProviderID:   req.ProviderID,  // 使用ProviderID字段
		Name:         req.Name,
		ClientID:     req.ClientID,
		RedirectURI:  req.RedirectURI,
		QuotaDaily:   req.QuotaDaily,
		QuotaMonthly: req.QuotaMonthly,
		Enabled:      true,
	}

	// 加密客户端密钥
	if err := client.SetClientSecret(req.ClientSecret); err != nil {
		return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
	}

	// 设置元数据
	if req.Metadata != "" {
		if err := client.SetMetadata(map[string]interface{}{
			"raw": req.Metadata,
		}); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	// 验证模型（使用修复后的验证逻辑）
	if err := client.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, client); err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 client: %w", err)
	}

	return client, nil
}

// Update 更新 OAuth2 客户端配置
func (s *OAuth2ClientService) Update(ctx context.Context, id int64, req *model.OAuth2ClientUpdateRequest) (*model.OAuth2Client, error) {
	// 查询现有配置
	client, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth2 client: %w", err)
	}

	// 更新字段
	updated := false
	if req.Name != "" {
		client.Name = req.Name
		updated = true
	}
	if req.ClientID != "" {
		client.ClientID = req.ClientID
		updated = true
	}
	if req.RedirectURI != "" {
		client.RedirectURI = req.RedirectURI
		updated = true
	}
	if req.ClientSecret != "" {
		if err := client.SetClientSecret(req.ClientSecret); err != nil {
			return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
		}
		updated = true
	}
	if req.Enabled != nil {
		client.Enabled = *req.Enabled
		updated = true
	}
	if req.QuotaDaily != nil {
		client.QuotaDaily = *req.QuotaDaily
		updated = true
	}
	if req.QuotaMonthly != nil {
		client.QuotaMonthly = *req.QuotaMonthly
		updated = true
	}
	if req.ProviderID != nil {
		client.ProviderID = *req.ProviderID
		updated = true
	}
	if req.Metadata != "" {
		if err := client.SetMetadata(map[string]interface{}{
			"raw": req.Metadata,
		}); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %w", err)
		}
		updated = true
	}

	if !updated {
		return client, nil
	}

	// 验证模型（使用修复后的验证逻辑）
	if err := client.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 保存更新
	if err := s.repo.Update(ctx, client); err != nil {
		return nil, fmt.Errorf("failed to update OAuth2 client: %w", err)
	}

	return client, nil
}

// Delete 删除 OAuth2 客户端配置
func (s *OAuth2ClientService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// GetByID 根据 ID 获取配置
func (s *OAuth2ClientService) GetByID(ctx context.Context, id int64) (*model.OAuth2Client, error) {
	client, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth2 client: %w", err)
	}
	return client, nil
}

// GetByProvider 获取提供商的所有客户端
func (s *OAuth2ClientService) GetByProvider(ctx context.Context, providerID int64) ([]model.OAuth2Client, error) {
	clients, err := s.repo.FindByProvider(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth2 clients: %w", err)
	}
	return clients, nil
}

// GetEnabled 获取所有启用的客户端
func (s *OAuth2ClientService) GetEnabled(ctx context.Context) ([]model.OAuth2Client, error) {
	clients, err := s.repo.FindEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find enabled OAuth2 clients: %w", err)
	}
	return clients, nil
}

// GetDefault 获取默认客户端
func (s *OAuth2ClientService) GetDefault(ctx context.Context, providerID int64) (*model.OAuth2Client, error) {
	client, err := s.repo.FindDefault(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find default OAuth2 client: %w", err)
	}
	return client, nil
}

// List 分页查询所有客户端
func (s *OAuth2ClientService) List(ctx context.Context, page, pageSize int) ([]model.OAuth2Client, int64, error) {
	clients, total, err := s.repo.FindAll(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find OAuth2 clients: %w", err)
	}
	return clients, total, nil
}

// SetDefault 设置默认客户端
func (s *OAuth2ClientService) SetDefault(ctx context.Context, id int64, providerID int64) error {
	// 检查客户端是否存在
	client, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find OAuth2 client: %w", err)
	}

	// 检查提供商是否匹配
	if client.ProviderID != providerID {
		return fmt.Errorf("OAuth2 client provider mismatch")
	}

	// 设置默认
	return s.repo.SetDefault(ctx, id, providerID)
}

// UseClient 使用客户端（检查配额并增加计数）
func (s *OAuth2ClientService) UseClient(ctx context.Context, id int64) (*model.OAuth2Client, error) {
	// 获取客户端配置
	client, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth2 client: %w", err)
	}

	// 检查是否可用
	if !client.CanUse() {
		return nil, fmt.Errorf("OAuth2 client quota exceeded or disabled")
	}

	// 增加使用计数
	if err := s.repo.IncrementUsage(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to increment usage: %w", err)
	}

	// 重新获取更新后的配置
	return s.repo.FindByID(ctx, id)
}

// SmartSelect 智能选择客户端
// 1. 优先选择用户指定的客户端
// 2. 其次选择该提供商的默认客户端
// 3. 最后选择第一个可用的客户端
func (s *OAuth2ClientService) SmartSelect(ctx context.Context, providerID int64, clientID *int64) (*model.OAuth2Client, error) {
	// 如果指定了客户端ID
	if clientID != nil {
		client, err := s.UseClient(ctx, *clientID)
		if err != nil {
			return nil, fmt.Errorf("failed to use specified client: %w", err)
		}
		// 检查客户端是否属于正确的提供商
		if client.ProviderID != providerID {
			return nil, fmt.Errorf("OAuth2 client provider mismatch")
		}
		return client, nil
	}

	// 尝试使用默认客户端
	defaultClient, err := s.GetDefault(ctx, providerID)
	if err == nil && defaultClient.CanUse() {
		return s.UseClient(ctx, defaultClient.ID)
	}

	// 查找第一个可用的客户端
	clients, err := s.GetByProvider(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth2 clients: %w", err)
	}

	for _, client := range clients {
		if client.CanUse() {
			return s.UseClient(ctx, client.ID)
		}
	}

	return nil, fmt.Errorf("no available OAuth2 clients for provider: %d", providerID)
}
