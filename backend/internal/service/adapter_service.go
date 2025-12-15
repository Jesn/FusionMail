package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// AdapterService 适配器业务逻辑服务
type AdapterService struct {
	repo repository.AdapterRepository
}

// NewAdapterService 创建 AdapterService 实例
func NewAdapterService(repo repository.AdapterRepository) *AdapterService {
	return &AdapterService{
		repo: repo,
	}
}

// List 获取所有适配器列表
func (s *AdapterService) List(ctx context.Context) ([]model.Adapter, error) {
	adapters, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapters: %w", err)
	}
	return adapters, nil
}

// ListEnabled 获取启用的适配器列表
func (s *AdapterService) ListEnabled(ctx context.Context) ([]model.Adapter, error) {
	adapters, err := s.repo.FindEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find enabled adapters: %w", err)
	}
	return adapters, nil
}

// GetByID 通过 ID 获取适配器
func (s *AdapterService) GetByID(ctx context.Context, id int64) (*model.Adapter, error) {
	adapter, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapter %d: %w", id, err)
	}
	if adapter == nil {
		return nil, fmt.Errorf("adapter with ID %d not found", id)
	}
	return adapter, nil
}

// GetByName 通过名称获取适配器
func (s *AdapterService) GetByName(ctx context.Context, name string) (*model.Adapter, error) {
	adapter, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapter %s: %w", name, err)
	}
	if adapter == nil {
		return nil, fmt.Errorf("adapter with name '%s' not found", name)
	}
	return adapter, nil
}

// GetByIDs 批量获取适配器
func (s *AdapterService) GetByIDs(ctx context.Context, ids []int64) ([]model.Adapter, error) {
	if len(ids) == 0 {
		return []model.Adapter{}, nil
	}
	adapters, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapters by IDs: %w", err)
	}
	return adapters, nil
}

// Create 创建新的适配器
func (s *AdapterService) Create(ctx context.Context, adapter *model.Adapter) (*model.Adapter, error) {
	// 验证配置
	if err := adapter.Validate(); err != nil {
		return nil, fmt.Errorf("adapter validation failed: %w", err)
	}

	// 检查名称是否已存在
	existing, err := s.repo.FindByName(ctx, adapter.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check adapter name: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("adapter with name '%s' already exists", adapter.Name)
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, adapter); err != nil {
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	return adapter, nil
}

// Update 更新适配器
func (s *AdapterService) Update(ctx context.Context, id int64, adapter *model.Adapter) (*model.Adapter, error) {
	// 获取现有适配器
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapter: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("adapter with ID %d not found", id)
	}

	// 更新字段
	existing.Name = adapter.Name
	existing.DisplayName = adapter.DisplayName
	existing.AuthType = adapter.AuthType
	existing.Description = adapter.Description
	existing.IsEnabled = adapter.IsEnabled

	// 验证配置
	if err := existing.Validate(); err != nil {
		return nil, fmt.Errorf("adapter validation failed: %w", err)
	}

	// 保存更新
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update adapter: %w", err)
	}

	return existing, nil
}

// Delete 删除适配器
func (s *AdapterService) Delete(ctx context.Context, id int64) error {
	// 检查适配器是否存在
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find adapter: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("adapter with ID %d not found", id)
	}

	// 删除适配器
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete adapter: %w", err)
	}

	return nil
}

// Enable 启用适配器
func (s *AdapterService) Enable(ctx context.Context, id int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find adapter: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("adapter with ID %d not found", id)
	}

	existing.IsEnabled = true
	if err := s.repo.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to enable adapter: %w", err)
	}

	return nil
}

// Disable 禁用适配器
func (s *AdapterService) Disable(ctx context.Context, id int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find adapter: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("adapter with ID %d not found", id)
	}

	existing.IsEnabled = false
	if err := s.repo.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to disable adapter: %w", err)
	}

	return nil
}

// IsOAuth2Adapter 检查适配器是否为 OAuth2 类型
func (s *AdapterService) IsOAuth2Adapter(ctx context.Context, id int64) (bool, error) {
	adapter, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to find adapter: %w", err)
	}
	if adapter == nil {
		return false, fmt.Errorf("adapter with ID %d not found", id)
	}
	return adapter.IsOAuth2(), nil
}
