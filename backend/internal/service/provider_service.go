package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// ProviderService Provider 业务逻辑服务
type ProviderService struct {
	repo repository.ProviderRepository
}

// NewProviderService 创建 ProviderService 实例
func NewProviderService(repo repository.ProviderRepository) *ProviderService {
	return &ProviderService{
		repo: repo,
	}
}

// Create 创建新的 Provider
func (s *ProviderService) Create(ctx context.Context, provider *model.Provider) (*model.Provider, error) {
	// 设置默认提供商类型
	if provider.ProviderType == 0 {
		provider.ProviderType = int(model.ProviderTypeGeneric)
	}

	// 设置默认协议支持
	if len(provider.SupportedProtocols) == 0 {
		if provider.ProviderType == int(model.ProviderTypeGmail) || provider.ProviderType == int(model.ProviderTypeOutlook) {
			// Gmail和Outlook默认支持OAuth2
			if err := provider.SetSupportedProtocols([]string{"oauth2", "imap"}); err != nil {
				return nil, fmt.Errorf("failed to set supported protocols: %w", err)
			}
			provider.RecommendedProtocol = "oauth2"
			provider.RequiresOAuth = true
		} else {
			// 其他类型默认只支持IMAP
			if err := provider.SetSupportedProtocols([]string{"imap"}); err != nil {
				return nil, fmt.Errorf("failed to set supported protocols: %w", err)
			}
			provider.RecommendedProtocol = "imap"
			provider.RequiresOAuth = false
		}
	}

	// 验证配置
	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return provider, nil
}

// List 获取所有 Provider 列表
func (s *ProviderService) List(ctx context.Context) ([]model.Provider, error) {
	providers, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find providers: %w", err)
	}

	return providers, nil
}

// ListWithPagination 分页获取 Provider 列表
func (s *ProviderService) ListWithPagination(ctx context.Context, page, pageSize int) ([]model.Provider, int64, error) {
	providers, total, err := s.repo.FindWithPagination(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find providers with pagination: %w", err)
	}

	return providers, total, nil
}

// GetByName 通过名称获取 Provider（内部使用）
func (s *ProviderService) GetByName(ctx context.Context, name string) (*model.Provider, error) {
	provider, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider %s: %w", name, err)
	}

	return provider, nil
}

// GetByID 通过ID获取 Provider
func (s *ProviderService) GetByID(ctx context.Context, id int64) (*model.Provider, error) {
	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider %d: %w", id, err)
	}

	return provider, nil
}

// GetByProviderType 通过提供商类型获取 Provider
func (s *ProviderService) GetByProviderType(ctx context.Context, providerType int) (*model.Provider, error) {
	provider, err := s.repo.FindByProviderType(ctx, providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider with type %d: %w", providerType, err)
	}

	return provider, nil
}

// UpdateByName 通过名称更新 Provider 配置（内部使用，不对外暴露）
func (s *ProviderService) UpdateByName(ctx context.Context, name string, provider *model.Provider) (*model.Provider, error) {
	// 首先获取现有的 Provider
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// 更新字段
	existing.DisplayName = provider.DisplayName
	existing.SupportedProtocols = provider.SupportedProtocols
	existing.RecommendedProtocol = provider.RecommendedProtocol
	existing.IMAPHost = provider.IMAPHost
	existing.IMAPPort = provider.IMAPPort
	existing.POP3Host = provider.POP3Host
	existing.POP3Port = provider.POP3Port
	existing.SMTPHost = provider.SMTPHost
	existing.SMTPPort = provider.SMTPPort
	existing.Enabled = provider.Enabled
	existing.SortOrder = provider.SortOrder
	existing.Description = provider.Description
	existing.Metadata = provider.Metadata

	// 验证配置
	if err := existing.Validate(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	// 保存更新
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return existing, nil
}

// UpdateByID 通过ID更新 Provider 配置
func (s *ProviderService) UpdateByID(ctx context.Context, id int64, provider *model.Provider) (*model.Provider, error) {
	// 首先获取现有的 Provider
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// 更新字段
	existing.Name = provider.Name
	existing.DisplayName = provider.DisplayName
	existing.ProviderType = provider.ProviderType
	existing.SupportedProtocols = provider.SupportedProtocols
	existing.RecommendedProtocol = provider.RecommendedProtocol
	existing.IMAPHost = provider.IMAPHost
	existing.IMAPPort = provider.IMAPPort
	existing.POP3Host = provider.POP3Host
	existing.POP3Port = provider.POP3Port
	existing.SMTPHost = provider.SMTPHost
	existing.SMTPPort = provider.SMTPPort
	existing.Enabled = provider.Enabled
	existing.SortOrder = provider.SortOrder
	existing.Description = provider.Description
	existing.Metadata = provider.Metadata

	// 验证配置
	if err := existing.Validate(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	// 保存更新 (使用基于ID的更新方法)
	if err := s.repo.UpdateByID(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return existing, nil
}

// DeleteByID 通过ID删除 Provider 配置
func (s *ProviderService) DeleteByID(ctx context.Context, id int64) error {
	err := s.repo.DeleteByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider %d: %w", id, err)
	}

	return nil
}
