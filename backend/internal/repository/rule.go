package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// RuleRepository 邮件规则数据仓库接口
type RuleRepository interface {
	Create(ctx context.Context, rule *model.EmailRule) error
	GetByID(ctx context.Context, id int64) (*model.EmailRule, error)
	FindByID(ctx context.Context, id int64) (*model.EmailRule, error)
	Update(ctx context.Context, rule *model.EmailRule) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*model.EmailRule, int64, error)
	ListByAccount(ctx context.Context, accountUID string) ([]*model.EmailRule, error)
	ListEnabled(ctx context.Context) ([]*model.EmailRule, error)
	UpdateMatchCount(ctx context.Context, id int64) error
	Enable(ctx context.Context, id int64) error
	Disable(ctx context.Context, id int64) error
}

// ruleRepository 邮件规则数据仓库实现
type ruleRepository struct {
	db *gorm.DB
}

// NewRuleRepository 创建邮件规则数据仓库实例
func NewRuleRepository(db *gorm.DB) RuleRepository {
	return &ruleRepository{db: db}
}

// Create 创建规则
func (r *ruleRepository) Create(ctx context.Context, rule *model.EmailRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID 根据 ID 查找规则
func (r *ruleRepository) GetByID(ctx context.Context, id int64) (*model.EmailRule, error) {
	var rule model.EmailRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// FindByID 根据 ID 查找规则（兼容方法）
func (r *ruleRepository) FindByID(ctx context.Context, id int64) (*model.EmailRule, error) {
	return r.GetByID(ctx, id)
}

// Update 更新规则
func (r *ruleRepository) Update(ctx context.Context, rule *model.EmailRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete 删除规则
func (r *ruleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.EmailRule{}, id).Error
}

// List 获取规则列表
func (r *ruleRepository) List(ctx context.Context, offset, limit int) ([]*model.EmailRule, int64, error) {
	var rules []*model.EmailRule
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.EmailRule{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表（按优先级排序）
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("priority ASC, created_at DESC").
		Find(&rules).Error

	return rules, total, err
}

// ListByAccount 根据账户 UID 获取规则列表
func (r *ruleRepository) ListByAccount(ctx context.Context, accountUID string) ([]*model.EmailRule, error) {
	var rules []*model.EmailRule
	query := r.db.WithContext(ctx).Order("priority ASC, created_at DESC")

	if accountUID != "" {
		query = query.Where("account_uid = ?", accountUID)
	}

	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}

	return rules, nil
}

// ListEnabled 获取启用的规则列表
func (r *ruleRepository) ListEnabled(ctx context.Context) ([]*model.EmailRule, error) {
	var rules []*model.EmailRule
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("priority ASC").
		Find(&rules).Error
	return rules, err
}

// UpdateMatchCount 更新匹配次数
func (r *ruleRepository) UpdateMatchCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailRule{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"matched_count":   gorm.Expr("matched_count + 1"),
			"last_matched_at": gorm.Expr("NOW()"),
		}).Error
}

// Enable 启用规则
func (r *ruleRepository) Enable(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailRule{}).
		Where("id = ?", id).
		Update("enabled", true).Error
}

// Disable 禁用规则
func (r *ruleRepository) Disable(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailRule{}).
		Where("id = ?", id).
		Update("enabled", false).Error
}
