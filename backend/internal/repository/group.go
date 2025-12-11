package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"

	"gorm.io/gorm"
)

// 模块日志记录器
var groupRepoLog = logger.NewWithModule("GroupRepo")

// GroupRepository 账号分组数据仓库接口
type GroupRepository interface {
	// 基础 CRUD 操作
	Create(ctx context.Context, group *model.AccountGroup) error
	Update(ctx context.Context, group *model.AccountGroup) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*model.AccountGroup, error)
	FindAll(ctx context.Context) ([]*model.AccountGroup, error)

	// 唯一性检查
	FindByName(ctx context.Context, name string) (*model.AccountGroup, error)
	ExistsByName(ctx context.Context, name string, excludeID int64) (bool, error)

	// 账号计数
	CountAccountsByGroupID(ctx context.Context, groupID int64) (int64, error)

	// 排序
	UpdateDisplayOrders(ctx context.Context, groupIDs []int64) error
	GetMaxDisplayOrder(ctx context.Context) (int, error)

	// 级联处理
	ClearGroupIDForAccounts(ctx context.Context, groupID int64) error
}

// groupRepository 账号分组数据仓库实现
type groupRepository struct {
	db *gorm.DB
}

// NewGroupRepository 创建账号分组数据仓库实例
func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

// Create 创建分组
func (r *groupRepository) Create(ctx context.Context, group *model.AccountGroup) error {
	return r.db.WithContext(ctx).Create(group).Error
}

// Update 更新分组
func (r *groupRepository) Update(ctx context.Context, group *model.AccountGroup) error {
	return r.db.WithContext(ctx).Save(group).Error
}

// Delete 删除分组（软删除）
func (r *groupRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.AccountGroup{}, id).Error
}

// FindByID 根据 ID 查找分组
func (r *groupRepository) FindByID(ctx context.Context, id int64) (*model.AccountGroup, error) {
	var group model.AccountGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// FindAll 获取所有分组（按显示顺序排序）
func (r *groupRepository) FindAll(ctx context.Context) ([]*model.AccountGroup, error) {
	var groups []*model.AccountGroup
	err := r.db.WithContext(ctx).
		Order("display_order ASC, id ASC").
		Find(&groups).Error
	return groups, err
}

// FindByName 根据名称查找分组
func (r *groupRepository) FindByName(ctx context.Context, name string) (*model.AccountGroup, error) {
	var group model.AccountGroup
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// ExistsByName 检查名称是否已存在（排除指定 ID）
func (r *groupRepository) ExistsByName(ctx context.Context, name string, excludeID int64) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.AccountGroup{}).Where("name = ?", name)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// CountAccountsByGroupID 统计分组中的账号数量
func (r *groupRepository) CountAccountsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("group_id = ?", groupID).
		Count(&count).Error
	return count, err
}

// UpdateDisplayOrders 批量更新分组显示顺序
// groupIDs 数组的索引即为新的显示顺序
func (r *groupRepository) UpdateDisplayOrders(ctx context.Context, groupIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for order, id := range groupIDs {
			// 使用 UpdateColumn 避免触发 BeforeUpdate 钩子
			if err := tx.Model(&model.AccountGroup{}).
				Where("id = ?", id).
				UpdateColumn("display_order", order).Error; err != nil {
				groupRepoLog.Error("更新分组顺序失败: id=%d, order=%d, err=%v", id, order, err)
				return err
			}
		}
		return nil
	})
}

// GetMaxDisplayOrder 获取当前最大的显示顺序值
func (r *groupRepository) GetMaxDisplayOrder(ctx context.Context) (int, error) {
	var maxOrder *int
	err := r.db.WithContext(ctx).
		Model(&model.AccountGroup{}).
		Select("MAX(display_order)").
		Scan(&maxOrder).Error
	if err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return -1, nil // 没有分组时返回 -1，新分组将从 0 开始
	}
	return *maxOrder, nil
}

// ClearGroupIDForAccounts 清除指定分组的所有账号关联
// 用于删除分组时的级联处理
func (r *groupRepository) ClearGroupIDForAccounts(ctx context.Context, groupID int64) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("group_id = ?", groupID).
		Update("group_id", nil).Error
}
