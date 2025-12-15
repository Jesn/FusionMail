package repository

import (
	"context"
	"errors"
	"fmt"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

// AdapterRepository 适配器数据访问接口
// 负责数据库中 adapters 表的所有操作
type AdapterRepository interface {
	// 基础 CRUD 操作
	Create(ctx context.Context, adapter *model.Adapter) error
	Update(ctx context.Context, adapter *model.Adapter) error
	Delete(ctx context.Context, id int64) error

	// 查询操作
	FindByID(ctx context.Context, id int64) (*model.Adapter, error)
	FindByName(ctx context.Context, name string) (*model.Adapter, error)
	FindAll(ctx context.Context) ([]model.Adapter, error)
	FindEnabled(ctx context.Context) ([]model.Adapter, error)

	// 批量查询
	FindByIDs(ctx context.Context, ids []int64) ([]model.Adapter, error)
}

// adapterRepository AdapterRepository 的具体实现
type adapterRepository struct {
	db *gorm.DB
}

// NewAdapterRepository 创建 AdapterRepository 实例
func NewAdapterRepository(db *gorm.DB) AdapterRepository {
	return &adapterRepository{db: db}
}

// Create 创建新的适配器
func (r *adapterRepository) Create(ctx context.Context, adapter *model.Adapter) error {
	// 验证配置
	if err := adapter.Validate(); err != nil {
		return fmt.Errorf("adapter validation failed: %w", err)
	}

	// 设置时间戳
	now := time.Now()
	adapter.CreatedAt = now
	adapter.UpdatedAt = now

	// 创建记录
	err := r.db.WithContext(ctx).Create(adapter).Error
	if err != nil {
		return fmt.Errorf("create adapter failed: %w", err)
	}

	return nil
}

// Update 更新适配器
func (r *adapterRepository) Update(ctx context.Context, adapter *model.Adapter) error {
	// 验证配置
	if err := adapter.Validate(); err != nil {
		return fmt.Errorf("adapter validation failed: %w", err)
	}

	// 更新时间戳
	adapter.UpdatedAt = time.Now()

	// 更新记录
	result := r.db.WithContext(ctx).
		Model(&model.Adapter{}).
		Where("id = ?", adapter.ID).
		Updates(map[string]interface{}{
			"name":         adapter.Name,
			"display_name": adapter.DisplayName,
			"auth_type":    adapter.AuthType,
			"description":  adapter.Description,
			"is_enabled":   adapter.IsEnabled,
			"updated_at":   adapter.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("update adapter failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("adapter with ID %d not found", adapter.ID)
	}

	return nil
}

// Delete 删除适配器
func (r *adapterRepository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&model.Adapter{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete adapter failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("adapter with ID %d not found", id)
	}

	return nil
}

// FindByID 根据 ID 查找适配器
func (r *adapterRepository) FindByID(ctx context.Context, id int64) (*model.Adapter, error) {
	var adapter model.Adapter
	err := r.db.WithContext(ctx).First(&adapter, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find adapter by ID failed: %w", err)
	}

	return &adapter, nil
}

// FindByName 根据名称查找适配器
func (r *adapterRepository) FindByName(ctx context.Context, name string) (*model.Adapter, error) {
	var adapter model.Adapter
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&adapter).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find adapter by name failed: %w", err)
	}

	return &adapter, nil
}

// FindAll 获取所有适配器
func (r *adapterRepository) FindAll(ctx context.Context) ([]model.Adapter, error) {
	var adapters []model.Adapter
	err := r.db.WithContext(ctx).
		Order("id ASC").
		Find(&adapters).Error

	if err != nil {
		return nil, fmt.Errorf("find all adapters failed: %w", err)
	}

	return adapters, nil
}

// FindEnabled 获取启用的适配器
func (r *adapterRepository) FindEnabled(ctx context.Context) ([]model.Adapter, error) {
	var adapters []model.Adapter
	err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("id ASC").
		Find(&adapters).Error

	if err != nil {
		return nil, fmt.Errorf("find enabled adapters failed: %w", err)
	}

	return adapters, nil
}

// FindByIDs 根据 ID 列表批量查找适配器
func (r *adapterRepository) FindByIDs(ctx context.Context, ids []int64) ([]model.Adapter, error) {
	if len(ids) == 0 {
		return []model.Adapter{}, nil
	}

	var adapters []model.Adapter
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Order("id ASC").
		Find(&adapters).Error

	if err != nil {
		return nil, fmt.Errorf("find adapters by IDs failed: %w", err)
	}

	return adapters, nil
}
