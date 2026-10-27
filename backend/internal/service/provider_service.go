package service

import (
	"context"
	"fmt"
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"strings"
)

// ProviderService Provider 业务逻辑服务
type ProviderService struct {
	repo        repository.ProviderRepository
	adapterRepo repository.AdapterRepository
}

// NewProviderService 创建 ProviderService 实例
func NewProviderService(repo repository.ProviderRepository) *ProviderService {
	return &ProviderService{
		repo: repo,
	}
}

// NewProviderServiceWithAdapterRepo 创建带 AdapterRepository 的 ProviderService 实例
func NewProviderServiceWithAdapterRepo(repo repository.ProviderRepository, adapterRepo repository.AdapterRepository) *ProviderService {
	return &ProviderService{
		repo:        repo,
		adapterRepo: adapterRepo,
	}
}

// Create 创建新的 Provider
// adapterIDs 为要关联的适配器 ID 列表（nil/空 表示无关联），在事务内写入 provider_adapters 表
func (s *ProviderService) Create(ctx context.Context, provider *model.Provider, adapterIDs []int64) (*model.Provider, error) {
	// 设置默认协议支持
	if len(provider.SupportedProtocols) == 0 {
		// 根据 requires_oauth 或 default_adapter 判断协议类型
		if provider.RequiresOAuth {
			// OAuth2 提供商默认支持 OAuth2 和 IMAP
			if err := provider.SetSupportedProtocols([]string{"oauth2", "imap"}); err != nil {
				return nil, fmt.Errorf("failed to set supported protocols: %w", err)
			}
			provider.RecommendedProtocol = "oauth2"
		} else {
			// 其他类型默认只支持 IMAP
			if err := provider.SetSupportedProtocols([]string{"imap"}); err != nil {
				return nil, fmt.Errorf("failed to set supported protocols: %w", err)
			}
			provider.RecommendedProtocol = "imap"
		}
	}

	// 验证配置
	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	// 校验并去重适配器 ID（防外键悬挂），须在落库前失败
	dedupedAdapterIDs, err := s.validateAdapterIDs(ctx, adapterIDs)
	if err != nil {
		return nil, err
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// 事务内写入适配器关联（创建时 nil 与 [] 语义相同：无关联）
	if len(dedupedAdapterIDs) > 0 {
		if err := s.repo.ReplaceAdapters(ctx, provider.ID, dedupedAdapterIDs); err != nil {
			return nil, fmt.Errorf("failed to sync provider adapters: %w", err)
		}
	}

	// 重新加载关联，保证响应里 supported_adapters 正确返回
	created, err := s.repo.FindWithAdapters(ctx, provider.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider with adapters: %w", err)
	}

	return created, nil
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
// adapterIDs 为要关联的适配器 ID 列表：先删旧关联再插新（事务内原子）；
// 空/nil 表示清空全部关联（用户取消全部勾选是合法操作）
func (s *ProviderService) UpdateByID(ctx context.Context, id int64, provider *model.Provider, adapterIDs []int64) (*model.Provider, error) {
	// 首先获取现有的 Provider
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// 校验并去重适配器 ID（防外键悬挂），须在落库前失败
	dedupedAdapterIDs, err := s.validateAdapterIDs(ctx, adapterIDs)
	if err != nil {
		return nil, err
	}

	// 更新字段
	existing.Name = provider.Name
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

	// 保存更新 (使用基于ID的更新方法)
	if err := s.repo.UpdateByID(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// 事务内同步适配器关联：先删旧关联再插新（空/nil 表示清空全部关联）
	if err := s.repo.ReplaceAdapters(ctx, existing.ID, dedupedAdapterIDs); err != nil {
		return nil, fmt.Errorf("failed to sync provider adapters: %w", err)
	}

	// 重新加载关联，保证响应里 supported_adapters 正确返回
	updated, err := s.repo.FindWithAdapters(ctx, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider with adapters: %w", err)
	}

	return updated, nil
}

// validateAdapterIDs 校验请求的适配器 ID 均存在并去重（防外键悬挂）。
// 去重保持请求顺序，priority 按顺序编号；空/nil 列表直接返回 nil。
func (s *ProviderService) validateAdapterIDs(ctx context.Context, adapterIDs []int64) ([]int64, error) {
	if len(adapterIDs) == 0 {
		return nil, nil
	}

	if s.adapterRepo == nil {
		return nil, fmt.Errorf("adapter repository is not configured")
	}

	// 去重，保持请求顺序（重复 ID 会导致 provider_adapters 复合主键冲突）
	seen := make(map[int64]struct{}, len(adapterIDs))
	deduped := make([]int64, 0, len(adapterIDs))
	for _, id := range adapterIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}

	// 批量查询校验存在性
	adapters, err := s.adapterRepo.FindByIDs(ctx, deduped)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapters: %w", err)
	}
	found := make(map[int64]struct{}, len(adapters))
	for _, a := range adapters {
		found[a.ID] = struct{}{}
	}
	for _, id := range deduped {
		if _, ok := found[id]; !ok {
			return nil, dto.NewAPIErrorWithMessage(dto.ErrValidationFailed, fmt.Sprintf("适配器 %d 不存在", id))
		}
	}

	return deduped, nil
}

// DeleteByID 通过ID删除 Provider 配置
func (s *ProviderService) DeleteByID(ctx context.Context, id int64) error {
	err := s.repo.DeleteByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider %d: %w", id, err)
	}

	return nil
}

// FindByEmail 根据邮箱地址查找 Provider
// 解析邮箱域名并匹配 Provider 的 email_domains 字段
func (s *ProviderService) FindByEmail(ctx context.Context, email string) (*model.Provider, error) {
	// 解析邮箱域名
	domain := extractDomain(email)
	if domain == "" {
		return nil, fmt.Errorf("invalid email address: %s", email)
	}

	// 根据域名查找 Provider
	provider, err := s.repo.FindByDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider by domain: %w", err)
	}

	return provider, nil
}

// FindByDomain 根据邮箱域名查找 Provider
func (s *ProviderService) FindByDomain(ctx context.Context, domain string) (*model.Provider, error) {
	provider, err := s.repo.FindByDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider by domain: %w", err)
	}
	return provider, nil
}

// GetWithAdapters 获取 Provider 并预加载适配器关联
func (s *ProviderService) GetWithAdapters(ctx context.Context, id int64) (*model.Provider, error) {
	provider, err := s.repo.FindWithAdapters(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider with adapters: %w", err)
	}
	return provider, nil
}

// ListWithAdapters 获取所有 Provider 并预加载适配器关联
func (s *ProviderService) ListWithAdapters(ctx context.Context) ([]model.Provider, error) {
	providers, err := s.repo.FindAllWithAdapters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find providers with adapters: %w", err)
	}
	return providers, nil
}

// ListEnabledWithAdapters 获取启用的 Provider 并预加载适配器关联
func (s *ProviderService) ListEnabledWithAdapters(ctx context.Context) ([]model.Provider, error) {
	providers, err := s.repo.FindEnabledWithAdapters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find enabled providers with adapters: %w", err)
	}
	return providers, nil
}

// GetSupportedAdapters 获取 Provider 支持的适配器列表
func (s *ProviderService) GetSupportedAdapters(ctx context.Context, providerID int64) ([]model.Adapter, error) {
	// 获取 Provider 并预加载适配器
	provider, err := s.repo.FindWithAdapters(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}

	// 提取适配器列表
	adapters := make([]model.Adapter, 0, len(provider.SupportedAdapters))
	for _, pa := range provider.SupportedAdapters {
		if pa.Adapter != nil {
			adapters = append(adapters, *pa.Adapter)
		}
	}

	return adapters, nil
}

// GetDefaultAdapter 获取 Provider 的默认适配器
func (s *ProviderService) GetDefaultAdapter(ctx context.Context, providerID int64) (*model.Adapter, error) {
	provider, err := s.repo.FindWithAdapters(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}

	// 优先返回 DefaultAdapter
	if provider.DefaultAdapter != nil {
		return provider.DefaultAdapter, nil
	}

	// 如果没有设置默认适配器，返回优先级最高的（priority=0）
	for _, pa := range provider.SupportedAdapters {
		if pa.Priority == 0 && pa.Adapter != nil {
			return pa.Adapter, nil
		}
	}

	// 返回第一个可用的适配器
	if len(provider.SupportedAdapters) > 0 && provider.SupportedAdapters[0].Adapter != nil {
		return provider.SupportedAdapters[0].Adapter, nil
	}

	return nil, fmt.Errorf("no adapter found for provider %d", providerID)
}

// extractDomain 从邮箱地址中提取域名
func extractDomain(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}

	// 查找 @ 符号
	atIndex := strings.LastIndex(email, "@")
	if atIndex == -1 || atIndex == len(email)-1 {
		return ""
	}

	domain := email[atIndex+1:]
	return strings.ToLower(domain)
}
