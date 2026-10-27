package repository

import (
	"context"

	"gorm.io/gorm"

	"fusionmail/internal/model"
)

// RuleMatchLogRepository 规则命中记录仓库
type RuleMatchLogRepository interface {
	// CreateBatch 批量写入命中记录（一次同步循环可能命中多条规则）
	CreateBatch(ctx context.Context, logs []*model.RuleMatchLog) error
	// FindByEmailID 查询某封邮件的全部命中记录（时间升序 = 规则应用顺序）
	FindByEmailID(ctx context.Context, emailID int64) ([]*model.RuleMatchLog, error)
	// DeleteOrphans 清理邮件已不存在的孤儿记录（永久删除/清空回收站后的残留）
	DeleteOrphans(ctx context.Context) (int64, error)
}

type ruleMatchLogRepository struct {
	db *gorm.DB
}

// NewRuleMatchLogRepository 创建规则命中记录仓库
func NewRuleMatchLogRepository(db *gorm.DB) RuleMatchLogRepository {
	return &ruleMatchLogRepository{db: db}
}

func (r *ruleMatchLogRepository) CreateBatch(ctx context.Context, logs []*model.RuleMatchLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&logs).Error
}

func (r *ruleMatchLogRepository) FindByEmailID(ctx context.Context, emailID int64) ([]*model.RuleMatchLog, error) {
	var logs []*model.RuleMatchLog
	err := r.db.WithContext(ctx).
		Where("email_id = ?", emailID).
		Order("id ASC").
		Find(&logs).Error
	return logs, err
}

func (r *ruleMatchLogRepository) DeleteOrphans(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("email_id NOT IN (?)",
			r.db.Model(&model.Email{}).Select("id")).
		Delete(&model.RuleMatchLog{})
	return res.RowsAffected, res.Error
}
