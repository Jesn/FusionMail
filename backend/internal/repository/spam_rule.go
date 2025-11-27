package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// SpamRuleRepository 垃圾邮件规则数据仓库接口
type SpamRuleRepository interface {
	Create(ctx context.Context, rule *model.SpamRule) error
	FindByID(ctx context.Context, id int64) (*model.SpamRule, error)
	Update(ctx context.Context, rule *model.SpamRule) error
	Delete(ctx context.Context, id int64) error
	FindAll(ctx context.Context) ([]*model.SpamRule, error)
	FindEnabled(ctx context.Context) ([]*model.SpamRule, error)
	FindByCategory(ctx context.Context, category string) ([]*model.SpamRule, error)
	ToggleEnabled(ctx context.Context, id int64) error
	IncrementHitCount(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*model.SpamRule, int64, error)
	ListByCategory(ctx context.Context, category string, offset, limit int) ([]*model.SpamRule, int64, error)
	CountBuiltin(ctx context.Context) (int64, error)
	CountCustom(ctx context.Context) (int64, error)
}

// spamRuleRepository 垃圾邮件规则数据仓库实现
type spamRuleRepository struct {
	db *gorm.DB
}

// NewSpamRuleRepository 创建垃圾邮件规则数据仓库实例
func NewSpamRuleRepository(db *gorm.DB) SpamRuleRepository {
	return &spamRuleRepository{db: db}
}

// Create 创建规则
func (r *spamRuleRepository) Create(ctx context.Context, rule *model.SpamRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// FindByID 根据 ID 查找规则
func (r *spamRuleRepository) FindByID(ctx context.Context, id int64) (*model.SpamRule, error) {
	var rule model.SpamRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// Update 更新规则
func (r *spamRuleRepository) Update(ctx context.Context, rule *model.SpamRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete 删除规则（仅允许删除自定义规则）
func (r *spamRuleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND is_builtin = ?", id, false).
		Delete(&model.SpamRule{}).Error
}

// FindAll 获取所有规则
func (r *spamRuleRepository) FindAll(ctx context.Context) ([]*model.SpamRule, error) {
	var rules []*model.SpamRule
	err := r.db.WithContext(ctx).
		Order("category ASC, score DESC").
		Find(&rules).Error
	return rules, err
}

// FindEnabled 获取所有启用的规则
func (r *spamRuleRepository) FindEnabled(ctx context.Context) ([]*model.SpamRule, error) {
	var rules []*model.SpamRule
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("category ASC, score DESC").
		Find(&rules).Error
	return rules, err
}

// FindByCategory 根据类别获取规则
func (r *spamRuleRepository) FindByCategory(ctx context.Context, category string) ([]*model.SpamRule, error) {
	var rules []*model.SpamRule
	err := r.db.WithContext(ctx).
		Where("category = ?", category).
		Order("score DESC").
		Find(&rules).Error
	return rules, err
}

// ToggleEnabled 切换规则启用状态
func (r *spamRuleRepository) ToggleEnabled(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.SpamRule{}).
		Where("id = ?", id).
		Update("enabled", gorm.Expr("NOT enabled")).Error
}

// IncrementHitCount 增加规则命中次数
func (r *spamRuleRepository) IncrementHitCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.SpamRule{}).
		Where("id = ?", id).
		UpdateColumn("hit_count", gorm.Expr("hit_count + 1")).Error
}

// List 获取规则列表（支持分页）
func (r *spamRuleRepository) List(ctx context.Context, offset, limit int) ([]*model.SpamRule, int64, error) {
	var rules []*model.SpamRule
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.SpamRule{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("category ASC, score DESC").
		Find(&rules).Error

	return rules, total, err
}

// ListByCategory 根据类别获取规则列表（支持分页）
func (r *spamRuleRepository) ListByCategory(ctx context.Context, category string, offset, limit int) ([]*model.SpamRule, int64, error) {
	var rules []*model.SpamRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SpamRule{}).
		Where("category = ?", category)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("score DESC").
		Find(&rules).Error

	return rules, total, err
}

// CountBuiltin 统计内置规则数量
func (r *spamRuleRepository) CountBuiltin(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.SpamRule{}).
		Where("is_builtin = ?", true).
		Count(&count).Error
	return count, err
}

// CountCustom 统计自定义规则数量
func (r *spamRuleRepository) CountCustom(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.SpamRule{}).
		Where("is_builtin = ?", false).
		Count(&count).Error
	return count, err
}
