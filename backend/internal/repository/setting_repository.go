package repository

import (
	"context"
	"fmt"

	"fusionmail/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingRepository Setting配置数据仓库
type SettingRepository struct {
	db *gorm.DB
}

// NewSettingRepository 创建Setting数据仓库
func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

// Get 获取单个配置项
// 支持系统级配置（user_id IS NULL）和用户级配置（user_id = X）
// 优先返回用户级配置，如果不存在则返回系统级配置
func (r *SettingRepository) Get(ctx context.Context, userID *int64, category, key string) (*model.Setting, error) {
	var setting model.Setting

	query := r.db.WithContext(ctx)

	// 如果有userID，优先查找用户级配置，否则查找系统级配置
	if userID != nil {
		err := query.
			Where("(user_id = ? OR user_id IS NULL) AND category = ? AND key = ?", *userID, category, key).
			Order("user_id DESC NULLS LAST"). // 用户级配置优先
			First(&setting).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil
			}
			return nil, err
		}
		return &setting, nil
	}

	// 没有userID，查找系统级配置（取 updated_at 最新的一条，避免历史重复数据干扰）
	err := query.
		Where("user_id IS NULL AND category = ? AND key = ?", category, key).
		Order("updated_at DESC").
		First(&setting).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

// GetByCategory 获取分类下的所有配置
// 返回该分类的系统级配置 + 用户级配置（如果有userID）
func (r *SettingRepository) GetByCategory(ctx context.Context, userID *int64, category string) ([]*model.Setting, error) {
	var settings []*model.Setting

	query := r.db.WithContext(ctx).Where("category = ?", category)

	if userID != nil {
		// 有用户ID，返回系统级配置 + 用户级配置
		err := query.
			Where("user_id IS NULL OR user_id = ?", *userID).
			Order("user_id DESC NULLS LAST, key ASC").
			Find(&settings).Error
		return settings, err
	}

	// 没有用户ID，只返回系统级配置
	err := query.
		Where("user_id IS NULL").
		Order("key ASC").
		Find(&settings).Error

	return settings, err
}

// GetSystem 获取系统级配置
func (r *SettingRepository) GetSystem(ctx context.Context, category, key string) (*model.Setting, error) {
	return r.Get(ctx, nil, category, key)
}

// GetUser 获取用户级配置
func (r *SettingRepository) GetUser(ctx context.Context, userID int64, category, key string) (*model.Setting, error) {
	return r.Get(ctx, &userID, category, key)
}

// Set 设置配置项
// 使用Upsert逻辑：如果存在则更新，不存在则创建
// 注意：PostgreSQL 中 NULL != NULL，ON CONFLICT 对 user_id IS NULL 的行不触发冲突检测，
// 因此系统级配置（userID=nil）必须使用手动 find-then-update-or-insert 逻辑。
func (r *SettingRepository) Set(ctx context.Context, userID *int64, category, key, value string, isSensitive bool, valueType string) error {
	if userID == nil {
		// 系统级配置：ON CONFLICT 无法匹配 user_id IS NULL 的行，使用手动 upsert
		return r.setSystem(ctx, category, key, value, isSensitive, valueType)
	}

	// 用户级配置：user_id 非 NULL，ON CONFLICT 可正常工作
	setting := model.Setting{
		UserID:      userID,
		Category:    category,
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		IsSensitive: isSensitive,
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "category"},
				{Name: "key"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&setting).Error
}

// setSystem 系统级配置的 upsert（user_id IS NULL）
func (r *SettingRepository) setSystem(ctx context.Context, category, key, value string, isSensitive bool, valueType string) error {
	result := r.db.WithContext(ctx).
		Model(&model.Setting{}).
		Where("user_id IS NULL AND category = ? AND key = ?", category, key).
		Updates(map[string]interface{}{
			"value":       value,
			"updated_at":  gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return result.Error
	}

	// 如果没有匹配到行，执行 INSERT
	if result.RowsAffected == 0 {
		setting := model.Setting{
			UserID:      nil,
			Category:    category,
			Key:         key,
			Value:       value,
			ValueType:   valueType,
			IsSensitive: isSensitive,
		}
		return r.db.WithContext(ctx).Create(&setting).Error
	}

	return nil
}

// BatchSet 批量设置配置项
func (r *SettingRepository) BatchSet(ctx context.Context, userID *int64, category string, settings map[string]string, isSensitiveMap map[string]bool, valueTypeMap map[string]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range settings {
			if userID == nil {
				// 系统级配置：ON CONFLICT 无法匹配 user_id IS NULL
				result := tx.WithContext(ctx).
					Model(&model.Setting{}).
					Where("user_id IS NULL AND category = ? AND key = ?", category, key).
					Updates(map[string]interface{}{
						"value":      value,
						"updated_at": gorm.Expr("NOW()"),
					})
				if result.Error != nil {
					return fmt.Errorf("failed to update setting %s: %w", key, result.Error)
				}
				if result.RowsAffected == 0 {
					setting := model.Setting{
						UserID:      nil,
						Category:    category,
						Key:         key,
						Value:       value,
						ValueType:   valueTypeMap[key],
						IsSensitive: isSensitiveMap[key],
					}
					if err := tx.Create(&setting).Error; err != nil {
						return fmt.Errorf("failed to create setting %s: %w", key, err)
					}
				}
			} else {
				// 用户级配置：user_id 非 NULL，ON CONFLICT 可正常工作
				setting := model.Setting{
					UserID:      userID,
					Category:    category,
					Key:         key,
					Value:       value,
					ValueType:   valueTypeMap[key],
					IsSensitive: isSensitiveMap[key],
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "user_id"},
						{Name: "category"},
						{Name: "key"},
					},
					DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
				}).Create(&setting).Error; err != nil {
					return fmt.Errorf("failed to set setting %s: %w", key, err)
				}
			}
		}
		return nil
	})
}

// Delete 删除配置项
func (r *SettingRepository) Delete(ctx context.Context, userID *int64, category, key string) error {
	query := r.db.WithContext(ctx).Where("category = ? AND key = ?", category, key)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	return query.Delete(&model.Setting{}).Error
}

// DeleteSystem 删除系统级配置
func (r *SettingRepository) DeleteSystem(ctx context.Context, category, key string) error {
	return r.Delete(ctx, nil, category, key)
}

// DeleteUser 删除用户级配置
func (r *SettingRepository) DeleteUser(ctx context.Context, userID int64, category, key string) error {
	return r.Delete(ctx, &userID, category, key)
}

// DeleteByCategory 删除整个分类的配置
func (r *SettingRepository) DeleteByCategory(ctx context.Context, userID *int64, category string) error {
	query := r.db.WithContext(ctx).Where("category = ?", category)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	return query.Delete(&model.Setting{}).Error
}

// DeleteSystemByCategory 删除系统级分类的所有配置
func (r *SettingRepository) DeleteSystemByCategory(ctx context.Context, category string) error {
	return r.DeleteByCategory(ctx, nil, category)
}

// DeleteUserByCategory 删除用户级分类的所有配置
func (r *SettingRepository) DeleteUserByCategory(ctx context.Context, userID int64, category string) error {
	return r.DeleteByCategory(ctx, &userID, category)
}

// GetAll 获取所有配置（慎用，可能返回大量数据）
func (r *SettingRepository) GetAll(ctx context.Context, userID *int64) ([]*model.Setting, error) {
	var settings []*model.Setting

	query := r.db.WithContext(ctx)

	if userID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	err := query.Order("category ASC, key ASC").Find(&settings).Error
	return settings, err
}

// GetPublic 获取公开配置
func (r *SettingRepository) GetPublic(ctx context.Context, userID *int64, category string) ([]*model.Setting, error) {
	var settings []*model.Setting

	query := r.db.WithContext(ctx).
		Where("is_public = ?", true).
		Where("category = ?", category)

	if userID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	err := query.Order("key ASC").Find(&settings).Error
	return settings, err
}

// GetSensitive 获取敏感配置（仅管理员使用）
func (r *SettingRepository) GetSensitive(ctx context.Context, userID *int64, category string) ([]*model.Setting, error) {
	var settings []*model.Setting

	query := r.db.WithContext(ctx).
		Where("is_sensitive = ?", true).
		Where("category = ?", category)

	if userID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	err := query.Order("key ASC").Find(&settings).Error
	return settings, err
}

// Search 搜索配置项
func (r *SettingRepository) Search(ctx context.Context, userID *int64, keyword, category string, onlyPublic bool) ([]*model.Setting, error) {
	var settings []*model.Setting

	query := r.db.WithContext(ctx)

	// 关键词搜索（搜索key和description）
	if keyword != "" {
		query = query.Where("key ILIKE ? OR description ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 分类筛选
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// 公开配置筛选
	if onlyPublic {
		query = query.Where("is_public = ?", true)
	}

	if userID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	err := query.Order("category ASC, key ASC").Find(&settings).Error
	return settings, err
}

// CountByCategory 统计分类下的配置数量
func (r *SettingRepository) CountByCategory(ctx context.Context, userID *int64, category string) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Setting{}).Where("category = ?", category)

	if userID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	err := query.Count(&count).Error
	return count, err
}

// Exists 检查配置是否存在
func (r *SettingRepository) Exists(ctx context.Context, userID *int64, category, key string) (bool, error) {
	var count int64

	query := r.db.WithContext(ctx).
		Model(&model.Setting{}).
		Where("category = ? AND key = ?", category, key)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetStats 获取数据库统计信息
func (r *SettingRepository) GetStats() (map[string]interface{}, error) {
	var stats struct {
		TotalCount       int64 `json:"total_count"`
		SystemCount      int64 `json:"system_count"`
		UserCount        int64 `json:"user_count"`
		SensitiveCount   int64 `json:"sensitive_count"`
		PublicCount      int64 `json:"public_count"`
		UniqueCategories int64 `json:"unique_categories"`
	}

	ctx := context.Background()

	// 总配置数量
	if err := r.db.WithContext(ctx).Model(&model.Setting{}).Count(&stats.TotalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// 系统级配置数量
	if err := r.db.WithContext(ctx).Model(&model.Setting{}).Where("user_id IS NULL").Count(&stats.SystemCount).Error; err != nil {
		return nil, fmt.Errorf("failed to get system count: %w", err)
	}

	// 用户级配置数量
	if err := r.db.WithContext(ctx).Model(&model.Setting{}).Where("user_id IS NOT NULL").Count(&stats.UserCount).Error; err != nil {
		return nil, fmt.Errorf("failed to get user count: %w", err)
	}

	// 敏感配置数量
	if err := r.db.WithContext(ctx).Model(&model.Setting{}).Where("is_sensitive = ?", true).Count(&stats.SensitiveCount).Error; err != nil {
		return nil, fmt.Errorf("failed to get sensitive count: %w", err)
	}

	// 公开配置数量
	if err := r.db.WithContext(ctx).Model(&model.Setting{}).Where("is_public = ?", true).Count(&stats.PublicCount).Error; err != nil {
		return nil, fmt.Errorf("failed to get public count: %w", err)
	}

	// 唯一分类数量
	if err := r.db.WithContext(ctx).Model(&model.Setting{}).Distinct("category").Count(&stats.UniqueCategories).Error; err != nil {
		return nil, fmt.Errorf("failed to get unique categories: %w", err)
	}

	return map[string]interface{}{
		"total_count":        stats.TotalCount,
		"system_count":       stats.SystemCount,
		"user_count":         stats.UserCount,
		"sensitive_count":    stats.SensitiveCount,
		"public_count":       stats.PublicCount,
		"unique_categories":  stats.UniqueCategories,
	}, nil
}
