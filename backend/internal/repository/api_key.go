package repository

import (
	"context"
	"time"

	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// APIKeyRepository API Key 数据仓库
type APIKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository 创建 API Key 数据仓库
func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create 创建 API Key
func (r *APIKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	return r.db.WithContext(ctx).Create(apiKey).Error
}

// FindByID 根据 ID 查找 API Key
func (r *APIKeyRepository) FindByID(ctx context.Context, id int64) (*model.APIKey, error) {
	var apiKey model.APIKey
	err := r.db.WithContext(ctx).First(&apiKey, id).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// FindByHash 根据哈希值查找 API Key
func (r *APIKeyRepository) FindByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var apiKey model.APIKey
	err := r.db.WithContext(ctx).
		Where("key_hash = ? AND enabled = ?", keyHash, true).
		First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// FindAll 查找所有 API Key
func (r *APIKeyRepository) FindAll(ctx context.Context) ([]*model.APIKey, error) {
	var apiKeys []*model.APIKey
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&apiKeys).Error
	return apiKeys, err
}

// Update 更新 API Key
func (r *APIKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	return r.db.WithContext(ctx).Save(apiKey).Error
}

// Delete 删除 API Key
func (r *APIKeyRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.APIKey{}, id).Error
}

// UpdateUsage 更新使用统计
func (r *APIKeyRepository) UpdateUsage(ctx context.Context, id int64, ip string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"total_requests": gorm.Expr("total_requests + 1"),
			"last_used_at":   now,
			"last_ip":        ip,
		}).Error
}

// Enable 启用 API Key
func (r *APIKeyRepository) Enable(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.APIKey{}).
		Where("id = ?", id).
		Update("enabled", true).Error
}

// Disable 禁用 API Key
func (r *APIKeyRepository) Disable(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.APIKey{}).
		Where("id = ?", id).
		Update("enabled", false).Error
}

// DeleteExpired 删除过期的 API Key
func (r *APIKeyRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&model.APIKey{}).Error
}
