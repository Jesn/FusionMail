package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// EmailListRepository 白名单/黑名单数据仓库接口
type EmailListRepository interface {
	Create(ctx context.Context, list *model.EmailList) error
	FindByID(ctx context.Context, id int64) (*model.EmailList, error)
	FindByUserAndType(ctx context.Context, userUID string, listType string) ([]*model.EmailList, error)
	Delete(ctx context.Context, id int64) error
	IsInList(ctx context.Context, userUID string, target string, listType string) (bool, error)
	FindByTarget(ctx context.Context, userUID string, target string, listType string) (*model.EmailList, error)
	List(ctx context.Context, userUID string, listType string, offset, limit int) ([]*model.EmailList, int64, error)
}

// emailListRepository 白名单/黑名单数据仓库实现
type emailListRepository struct {
	db *gorm.DB
}

// NewEmailListRepository 创建白名单/黑名单数据仓库实例
func NewEmailListRepository(db *gorm.DB) EmailListRepository {
	return &emailListRepository{db: db}
}

// Create 创建白名单/黑名单条目
func (r *emailListRepository) Create(ctx context.Context, list *model.EmailList) error {
	return r.db.WithContext(ctx).Create(list).Error
}

// FindByID 根据 ID 查找条目
func (r *emailListRepository) FindByID(ctx context.Context, id int64) (*model.EmailList, error) {
	var list model.EmailList
	err := r.db.WithContext(ctx).First(&list, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &list, nil
}

// FindByUserAndType 根据用户和类型查找所有条目
func (r *emailListRepository) FindByUserAndType(ctx context.Context, userUID string, listType string) ([]*model.EmailList, error) {
	var lists []*model.EmailList
	err := r.db.WithContext(ctx).
		Where("user_uid = ? AND type = ?", userUID, listType).
		Order("created_at DESC").
		Find(&lists).Error
	return lists, err
}

// Delete 删除条目
func (r *emailListRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.EmailList{}, id).Error
}

// IsInList 检查目标是否在列表中
func (r *emailListRepository) IsInList(ctx context.Context, userUID string, target string, listType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.EmailList{}).
		Where("user_uid = ? AND type = ? AND target = ?", userUID, listType, target).
		Count(&count).Error
	return count > 0, err
}

// FindByTarget 根据目标查找条目
func (r *emailListRepository) FindByTarget(ctx context.Context, userUID string, target string, listType string) (*model.EmailList, error) {
	var list model.EmailList
	err := r.db.WithContext(ctx).
		Where("user_uid = ? AND type = ? AND target = ?", userUID, listType, target).
		First(&list).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &list, nil
}

// List 获取列表（支持分页）
func (r *emailListRepository) List(ctx context.Context, userUID string, listType string, offset, limit int) ([]*model.EmailList, int64, error) {
	var lists []*model.EmailList
	var total int64

	query := r.db.WithContext(ctx).Model(&model.EmailList{}).
		Where("user_uid = ? AND type = ?", userUID, listType)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&lists).Error

	return lists, total, err
}
