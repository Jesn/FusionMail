package repository

import (
	"context"
	"time"

	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// DeletedEmailKeyRepository 已删除邮件去重标识仓库
type DeletedEmailKeyRepository struct {
	db *gorm.DB
}

// NewDeletedEmailKeyRepository 创建已删除邮件去重标识仓库
func NewDeletedEmailKeyRepository(db *gorm.DB) *DeletedEmailKeyRepository {
	return &DeletedEmailKeyRepository{db: db}
}

// Create 创建已删除邮件记录
func (r *DeletedEmailKeyRepository) Create(ctx context.Context, accountUID, dedupeKey string) error {
	record := &model.DeletedEmailKey{
		AccountUID: accountUID,
		DedupeKey:  dedupeKey,
		DeletedAt:  time.Now(),
	}

	// 使用 ON CONFLICT DO NOTHING 避免重复插入
	return r.db.WithContext(ctx).
		Clauses().
		Create(record).Error
}

// CreateIfNotExists 如果不存在则创建记录
func (r *DeletedEmailKeyRepository) CreateIfNotExists(ctx context.Context, accountUID, dedupeKey string) error {
	record := &model.DeletedEmailKey{
		AccountUID: accountUID,
		DedupeKey:  dedupeKey,
		DeletedAt:  time.Now(),
	}

	// 先检查是否存在
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.DeletedEmailKey{}).
		Where("account_uid = ? AND dedupe_key = ?", accountUID, dedupeKey).
		Count(&count).Error
	if err != nil {
		return err
	}

	// 如果不存在则创建
	if count == 0 {
		return r.db.WithContext(ctx).Create(record).Error
	}

	return nil
}

// IsDeleted 检查 dedupe_key 是否在已删除列表中
func (r *DeletedEmailKeyRepository) IsDeleted(ctx context.Context, accountUID, dedupeKey string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.DeletedEmailKey{}).
		Where("account_uid = ? AND dedupe_key = ?", accountUID, dedupeKey).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// BatchIsDeleted 批量检查 dedupe_key 是否在已删除列表中
// 返回已删除的 dedupe_key 集合
func (r *DeletedEmailKeyRepository) BatchIsDeleted(ctx context.Context, accountUID string, dedupeKeys []string) (map[string]bool, error) {
	if len(dedupeKeys) == 0 {
		return make(map[string]bool), nil
	}

	var deletedKeys []string
	err := r.db.WithContext(ctx).
		Model(&model.DeletedEmailKey{}).
		Where("account_uid = ? AND dedupe_key IN ?", accountUID, dedupeKeys).
		Pluck("dedupe_key", &deletedKeys).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, key := range deletedKeys {
		result[key] = true
	}
	return result, nil
}

// Delete 删除记录
func (r *DeletedEmailKeyRepository) Delete(ctx context.Context, accountUID, dedupeKey string) error {
	return r.db.WithContext(ctx).
		Where("account_uid = ? AND dedupe_key = ?", accountUID, dedupeKey).
		Delete(&model.DeletedEmailKey{}).Error
}

// CleanupOldKeys 清理过期的删除记录
// 默认清理 90 天前的记录
func (r *DeletedEmailKeyRepository) CleanupOldKeys(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	result := r.db.WithContext(ctx).
		Where("deleted_at < ?", cutoffTime).
		Delete(&model.DeletedEmailKey{})

	return result.RowsAffected, result.Error
}

// CountByAccount 统计账户的已删除记录数
func (r *DeletedEmailKeyRepository) CountByAccount(ctx context.Context, accountUID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.DeletedEmailKey{}).
		Where("account_uid = ?", accountUID).
		Count(&count).Error
	return count, err
}

// DeleteByAccount 删除账户的所有已删除记录
func (r *DeletedEmailKeyRepository) DeleteByAccount(ctx context.Context, accountUID string) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("account_uid = ?", accountUID).
		Delete(&model.DeletedEmailKey{})
	return result.RowsAffected, result.Error
}
