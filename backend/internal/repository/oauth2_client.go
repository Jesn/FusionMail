package repository

import (
	"context"
	"fmt"

	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// OAuth2ClientRepository OAuth2 客户端数据访问接口
type OAuth2ClientRepository interface {
	// Create 创建新的 OAuth2 客户端配置
	Create(ctx context.Context, client *model.OAuth2Client) error

	// Update 更新 OAuth2 客户端配置
	Update(ctx context.Context, client *model.OAuth2Client) error

	// Delete 删除 OAuth2 客户端配置
	Delete(ctx context.Context, id int64) error

	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id int64) (*model.OAuth2Client, error)

	// FindByProvider 根据提供商ID查询所有客户端
	FindByProvider(ctx context.Context, providerID int64) ([]model.OAuth2Client, error)

	// FindByProviderType 根据提供商类型查询所有客户端
	FindByProviderType(ctx context.Context, providerType int) ([]model.OAuth2Client, error)

	// FindEnabled 查询启用的客户端
	FindEnabled(ctx context.Context) ([]model.OAuth2Client, error)

	// FindDefaultByProviderType 根据提供商类型查询默认客户端
	FindDefaultByProviderType(ctx context.Context, providerType int) (*model.OAuth2Client, error)

	// FindDefault 根据提供商ID查询默认客户端
	FindDefault(ctx context.Context, providerID int64) (*model.OAuth2Client, error)

	// FindAll 查询所有客户端（分页）
	FindAll(ctx context.Context, page, pageSize int) ([]model.OAuth2Client, int64, error)

	// SetDefault 设置默认客户端
	SetDefault(ctx context.Context, id int64, providerID int64) error

	// SetDefaultByProviderType 根据提供商类型设置默认客户端
	SetDefaultByProviderType(ctx context.Context, id int64, providerType int) error
	IncrementUsage(ctx context.Context, id int64) error
}

// OAuth2ClientGormRepository GORM 实现
type OAuth2ClientGormRepository struct {
	db *gorm.DB
}

// NewOAuth2ClientRepository 创建 OAuth2 客户端仓库
func NewOAuth2ClientRepository(db *gorm.DB) OAuth2ClientRepository {
	return &OAuth2ClientGormRepository{
		db: db,
	}
}

// Create 创建新的 OAuth2 客户端配置
func (r *OAuth2ClientGormRepository) Create(ctx context.Context, client *model.OAuth2Client) error {
	if err := client.Validate(); err != nil {
		return err
	}

	// 加密客户端密钥
	if client.ClientSecretEncrypted == "" && client.ClientID != "" {
		// 这里应该从请求中获取密钥并加密
		// 暂时跳过，假设已在调用方处理
	}

	return r.db.WithContext(ctx).Create(client).Error
}

// Update 更新 OAuth2 客户端配置
func (r *OAuth2ClientGormRepository) Update(ctx context.Context, client *model.OAuth2Client) error {
	if err := client.Validate(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Save(client).Error
}

// Delete 删除 OAuth2 客户端配置
func (r *OAuth2ClientGormRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.OAuth2Client{}, id).Error
}

// FindByID 根据 ID 查询
func (r *OAuth2ClientGormRepository) FindByID(ctx context.Context, id int64) (*model.OAuth2Client, error) {
	var client model.OAuth2Client
	if err := r.db.WithContext(ctx).First(&client, id).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

// FindByProvider 根据提供商ID查询所有客户端
func (r *OAuth2ClientGormRepository) FindByProvider(ctx context.Context, providerID int64) ([]model.OAuth2Client, error) {
	var clients []model.OAuth2Client
	if err := r.db.WithContext(ctx).
		Preload("Provider"). // 预加载提供商信息
		Where("provider_id = ?", providerID).
		Order("is_default DESC, name ASC").
		Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

// FindEnabled 查询启用的客户端
func (r *OAuth2ClientGormRepository) FindEnabled(ctx context.Context) ([]model.OAuth2Client, error) {
	var clients []model.OAuth2Client
	if err := r.db.WithContext(ctx).
		Preload("Provider"). // 预加载提供商信息
		Where("enabled = ?", true).
		Order("provider_id ASC, is_default DESC, name ASC").
		Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

// FindDefault 查询默认客户端
func (r *OAuth2ClientGormRepository) FindDefault(ctx context.Context, providerID int64) (*model.OAuth2Client, error) {
	var client model.OAuth2Client
	if err := r.db.WithContext(ctx).
		Preload("Provider"). // 预加载提供商信息
		Where("provider_id = ? AND is_default = ? AND enabled = ?", providerID, true, true).
		First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

// FindAll 查询所有客户端（分页）
func (r *OAuth2ClientGormRepository) FindAll(ctx context.Context, page, pageSize int) ([]model.OAuth2Client, int64, error) {
	var clients []model.OAuth2Client
	var total int64

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&model.OAuth2Client{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).
		Preload("Provider"). // 预加载提供商信息
		Offset(offset).
		Limit(pageSize).
		Order("provider_id ASC, is_default DESC, name ASC").
		Find(&clients).Error; err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// SetDefaultByProviderType 根据提供商类型设置默认客户端
func (r *OAuth2ClientGormRepository) SetDefaultByProviderType(ctx context.Context, id int64, providerType int) error {
	// 先清除该提供商类型的所有默认设置
	if err := r.db.WithContext(ctx).
		Model(&model.OAuth2Client{}).
		Where("provider_id IN (SELECT id FROM providers WHERE provider_type = ?)", providerType).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("failed to clear existing defaults: %w", err)
	}

	// 设置新的默认客户端
	return r.db.WithContext(ctx).
		Model(&model.OAuth2Client{}).
		Where("id = ? AND provider_id IN (SELECT id FROM providers WHERE provider_type = ?)", id, providerType).
		Update("is_default", true).Error
}

// SetDefault 设置默认客户端
func (r *OAuth2ClientGormRepository) SetDefault(ctx context.Context, id int64, providerID int64) error {
	// 首先清除该提供商的所有默认标记
	if err := r.db.WithContext(ctx).
		Model(&model.OAuth2Client{}).
		Where("provider_id = ?", providerID).
		Update("is_default", false).Error; err != nil {
		return err
	}

	// 设置新的默认客户端
	return r.db.WithContext(ctx).
		Model(&model.OAuth2Client{}).
		Where("id = ?", id).
		Update("is_default", true).Error
}

// FindByProviderType 根据提供商类型查询所有客户端
func (r *OAuth2ClientGormRepository) FindByProviderType(ctx context.Context, providerType int) ([]model.OAuth2Client, error) {
	var clients []model.OAuth2Client
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Joins("JOIN providers ON email_oauth2_tokens.provider_id = providers.id").
		Where("providers.provider_type = ?", providerType).
		Find(&clients).Error
	return clients, err
}

// FindDefaultByProviderType 根据提供商类型查询默认客户端
func (r *OAuth2ClientGormRepository) FindDefaultByProviderType(ctx context.Context, providerType int) (*model.OAuth2Client, error) {
	var client model.OAuth2Client
	if err := r.db.WithContext(ctx).
		Preload("Provider").
		Joins("JOIN providers ON email_oauth2_tokens.provider_id = providers.id").
		Where("providers.provider_type = ? AND email_oauth2_tokens.is_default = ? AND email_oauth2_tokens.enabled = ?", providerType, true, true).
		First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

// IncrementUsage 增加使用计数
func (r *OAuth2ClientGormRepository) IncrementUsage(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.OAuth2Client{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"usage_count":  gorm.Expr("usage_count + 1"),
			"last_used_at": gorm.Expr("NOW()"),
		}).Error
}
