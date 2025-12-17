package repository

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

// ProviderRepository 提供商数据访问接口
// 负责数据库中 providers 表的所有操作
type ProviderRepository interface {
	// 基础 CRUD 操作
	Create(ctx context.Context, provider *model.Provider) error
	Update(ctx context.Context, provider *model.Provider) error
	UpdateByID(ctx context.Context, provider *model.Provider) error
	Delete(ctx context.Context, name string) error
	DeleteByID(ctx context.Context, id int64) error

	// 查询操作
	FindAll(ctx context.Context) ([]model.Provider, error)
	FindByName(ctx context.Context, name string) (*model.Provider, error)
	FindByID(ctx context.Context, id int64) (*model.Provider, error)
	FindEnabled(ctx context.Context) ([]model.Provider, error)
	FindWithPagination(ctx context.Context, page, pageSize int) ([]model.Provider, int64, error)

	// 新增：域名匹配和适配器关联查询
	FindByDomain(ctx context.Context, domain string) (*model.Provider, error)
	FindWithAdapters(ctx context.Context, id int64) (*model.Provider, error)
	FindAllWithAdapters(ctx context.Context) ([]model.Provider, error)
	FindEnabledWithAdapters(ctx context.Context) ([]model.Provider, error)
}

// providerRepository ProviderRepository 的具体实现
type providerRepository struct {
	db *gorm.DB
}

// NewProviderRepository 创建 ProviderRepository 实例
// 依赖数据库连接实例
func NewProviderRepository(db *gorm.DB) ProviderRepository {
	return &providerRepository{db: db}
}

// Create 创建新的提供商配置
// 在创建前会自动验证配置的有效性
func (r *providerRepository) Create(ctx context.Context, provider *model.Provider) error {
	// 验证配置
	if err := provider.Validate(); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	// 设置时间戳
	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	// 创建记录
	err := r.db.WithContext(ctx).Create(provider).Error
	if err != nil {
		// 检查是否是唯一约束冲突
		if isUniqueConstraintError(err, "name") {
			return fmt.Errorf("provider with name '%s' already exists", provider.Name)
		}
		return fmt.Errorf("create provider failed: %w", err)
	}

	return nil
}

// Update 更新现有提供商配置
// 支持部分更新（只更新非空字段）
func (r *providerRepository) Update(ctx context.Context, provider *model.Provider) error {
	// 验证配置
	if err := provider.Validate(); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	// 更新更新时间
	provider.UpdatedAt = time.Now()

	// 更新记录（使用 Update 的条件更新模式）
	result := r.db.WithContext(ctx).
		Model(&model.Provider{}).
		Where("name = ?", provider.Name).
		Updates(map[string]interface{}{
			"display_name":         provider.DisplayName,
			"supported_protocols":  provider.SupportedProtocols,
			"recommended_protocol": provider.RecommendedProtocol,
			"imap_host":            provider.IMAPHost,
			"imap_port":            provider.IMAPPort,
			"pop3_host":            provider.POP3Host,
			"pop3_port":            provider.POP3Port,
			"smtp_host":            provider.SMTPHost,
			"smtp_port":            provider.SMTPPort,
			"enabled":              provider.Enabled,
			"sort_order":           provider.SortOrder,
			"description":          provider.Description,
			"metadata":             provider.Metadata,
			"updated_at":           provider.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("update provider failed: %w", result.Error)
	}

	// 检查是否有记录被更新
	if result.RowsAffected == 0 {
		return fmt.Errorf("provider '%s' not found", provider.Name)
	}

	return nil
}

// UpdateByID 通过ID更新提供商配置
// 使用ID作为查询条件,允许更新包括name在内的所有字段
func (r *providerRepository) UpdateByID(ctx context.Context, provider *model.Provider) error {
	// 验证配置
	if err := provider.Validate(); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	// 更新更新时间
	provider.UpdatedAt = time.Now()

	// 更新记录（使用 ID 作为条件）
	result := r.db.WithContext(ctx).
		Model(&model.Provider{}).
		Where("id = ?", provider.ID).
		Updates(map[string]interface{}{
			"name":                 provider.Name,
			"display_name":         provider.DisplayName,
			"supported_protocols":  provider.SupportedProtocols,
			"recommended_protocol": provider.RecommendedProtocol,
			"imap_host":            provider.IMAPHost,
			"imap_port":            provider.IMAPPort,
			"pop3_host":            provider.POP3Host,
			"pop3_port":            provider.POP3Port,
			"smtp_host":            provider.SMTPHost,
			"smtp_port":            provider.SMTPPort,
			"enabled":              provider.Enabled,
			"sort_order":           provider.SortOrder,
			"description":          provider.Description,
			"metadata":             provider.Metadata,
			"updated_at":           provider.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("update provider failed: %w", result.Error)
	}

	// 检查是否有记录被更新
	if result.RowsAffected == 0 {
		return fmt.Errorf("provider with ID %d not found", provider.ID)
	}

	return nil
}

// Delete 删除提供商配置
// 通过提供商名称删除记录
func (r *providerRepository) Delete(ctx context.Context, name string) error {
	// 检查记录是否存在
	existing, err := r.FindByName(ctx, name)
	if err != nil {
		return fmt.Errorf("provider '%s' not found: %w", name, err)
	}

	// 删除记录
	return r.db.WithContext(ctx).Delete(existing).Error
}

// DeleteByID 通过ID删除 Provider 配置
func (r *providerRepository) DeleteByID(ctx context.Context, id int64) error {
	// 检查记录是否存在
	existing, err := r.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("provider %d not found: %w", id, err)
	}

	// 删除记录
	return r.db.WithContext(ctx).Delete(existing).Error
}

// FindAll 获取所有提供商配置
// 返回所有记录，按 sort_order 升序排序
func (r *providerRepository) FindAll(ctx context.Context) ([]model.Provider, error) {
	var providers []model.Provider
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, name ASC").
		Find(&providers).Error
	return providers, err
}

// FindByName 根据提供商名称查找配置
// 名称是唯一的，所以返回单个记录
func (r *providerRepository) FindByName(ctx context.Context, name string) (*model.Provider, error) {
	var provider model.Provider
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&provider).Error

	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}
	return &provider, err
}

// FindByID 根据ID查找提供商配置
// ID是主键，所以返回单个记录
func (r *providerRepository) FindByID(ctx context.Context, id int64) (*model.Provider, error) {
	var provider model.Provider
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&provider).Error

	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("provider with ID %d not found", id)
	}
	return &provider, err
}

// FindEnabled 获取启用的提供商配置
// 只返回 enabled=true 的记录
func (r *providerRepository) FindEnabled(ctx context.Context) ([]model.Provider, error) {
	var providers []model.Provider
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("sort_order ASC, name ASC").
		Find(&providers).Error
	return providers, err
}

// FindWithPagination 分页查询提供商配置
// 返回指定页的数据和总记录数
func (r *providerRepository) FindWithPagination(ctx context.Context, page, pageSize int) ([]model.Provider, int64, error) {
	var providers []model.Provider
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.Provider{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count providers failed: %w", err)
	}

	// 查询分页数据
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, name ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&providers).Error

	if err != nil {
		return nil, 0, fmt.Errorf("find providers failed: %w", err)
	}

	return providers, total, nil
}

// isUniqueConstraintError 检查错误是否是唯一约束冲突
func isUniqueConstraintError(err error, columnName string) bool {
	if err == nil {
		return false
	}

	// SQLite 错误信息格式：UNIQUE constraint failed: table.column
	errStr := err.Error()

	if columnName != "" {
		// 检查特定列的唯一约束
		prefix := "UNIQUE constraint failed: providers." + columnName
		return errStr == prefix || len(errStr) >= len(prefix) && errStr[:len(prefix)] == prefix
	}

	// 检查是否是任何 UNIQUE constraint failed 错误
	return len(errStr) >= 28 && errStr[:28] == "UNIQUE constraint failed: providers"
}

// FindByDomain 根据邮箱域名查找提供商
// 使用 PostgreSQL 的 ANY 操作符匹配 email_domains 数组
// 返回第一个匹配的启用的提供商
func (r *providerRepository) FindByDomain(ctx context.Context, domain string) (*model.Provider, error) {
	var provider model.Provider

	// 使用 PostgreSQL 的 ANY 操作符查询数组字段
	// 同时检查 enabled = true 确保只返回启用的提供商
	err := r.db.WithContext(ctx).
		Where("? = ANY(email_domains) AND enabled = ?", domain, true).
		Order("sort_order ASC, name ASC").
		First(&provider).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find provider by domain failed: %w", err)
	}

	return &provider, nil
}

// FindWithAdapters 根据 ID 查找提供商并预加载适配器关联
// 包括默认适配器和所有支持的适配器
func (r *providerRepository) FindWithAdapters(ctx context.Context, id int64) (*model.Provider, error) {
	var provider model.Provider

	err := r.db.WithContext(ctx).
		Preload("DefaultAdapter").
		Preload("SupportedAdapters", func(db *gorm.DB) *gorm.DB {
			return db.Order("priority ASC")
		}).
		Preload("SupportedAdapters.Adapter").
		Where("id = ?", id).
		First(&provider).Error

	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("provider with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("find provider with adapters failed: %w", err)
	}

	return &provider, nil
}

// FindAllWithAdapters 获取所有提供商并预加载适配器关联
func (r *providerRepository) FindAllWithAdapters(ctx context.Context) ([]model.Provider, error) {
	var providers []model.Provider

	err := r.db.WithContext(ctx).
		Preload("DefaultAdapter").
		Preload("SupportedAdapters", func(db *gorm.DB) *gorm.DB {
			return db.Order("priority ASC")
		}).
		Preload("SupportedAdapters.Adapter").
		Order("sort_order ASC, name ASC").
		Find(&providers).Error

	if err != nil {
		return nil, fmt.Errorf("find all providers with adapters failed: %w", err)
	}

	return providers, nil
}

// FindEnabledWithAdapters 获取启用的提供商并预加载适配器关联
func (r *providerRepository) FindEnabledWithAdapters(ctx context.Context) ([]model.Provider, error) {
	var providers []model.Provider

	err := r.db.WithContext(ctx).
		Preload("DefaultAdapter").
		Preload("SupportedAdapters", func(db *gorm.DB) *gorm.DB {
			return db.Order("priority ASC")
		}).
		Preload("SupportedAdapters.Adapter").
		Where("enabled = ?", true).
		Order("sort_order ASC, name ASC").
		Find(&providers).Error

	if err != nil {
		return nil, fmt.Errorf("find enabled providers with adapters failed: %w", err)
	}

	return providers, nil
}
