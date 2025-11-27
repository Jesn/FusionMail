package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// BayesianTrainingRepository 贝叶斯训练数据仓库接口
type BayesianTrainingRepository interface {
	Create(ctx context.Context, training *model.BayesianTraining) error
	FindByID(ctx context.Context, id int64) (*model.BayesianTraining, error)
	FindByUser(ctx context.Context, userUID string) ([]*model.BayesianTraining, error)
	FindByUserAndType(ctx context.Context, userUID string, isSpam bool) ([]*model.BayesianTraining, error)
	CountByUser(ctx context.Context, userUID string) (int64, error)
	CountByUserAndType(ctx context.Context, userUID string, isSpam bool) (int64, error)
	DeleteByUser(ctx context.Context, userUID string) error
	DeleteByEmail(ctx context.Context, emailID string) error
	List(ctx context.Context, userUID string, offset, limit int) ([]*model.BayesianTraining, int64, error)
}

// bayesianTrainingRepository 贝叶斯训练数据仓库实现
type bayesianTrainingRepository struct {
	db *gorm.DB
}

// NewBayesianTrainingRepository 创建贝叶斯训练数据仓库实例
func NewBayesianTrainingRepository(db *gorm.DB) BayesianTrainingRepository {
	return &bayesianTrainingRepository{db: db}
}

// Create 创建训练数据
func (r *bayesianTrainingRepository) Create(ctx context.Context, training *model.BayesianTraining) error {
	return r.db.WithContext(ctx).Create(training).Error
}

// FindByID 根据 ID 查找训练数据
func (r *bayesianTrainingRepository) FindByID(ctx context.Context, id int64) (*model.BayesianTraining, error) {
	var training model.BayesianTraining
	err := r.db.WithContext(ctx).First(&training, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &training, nil
}

// FindByUser 根据用户查找所有训练数据
func (r *bayesianTrainingRepository) FindByUser(ctx context.Context, userUID string) ([]*model.BayesianTraining, error) {
	var trainings []*model.BayesianTraining
	err := r.db.WithContext(ctx).
		Where("user_uid = ?", userUID).
		Order("created_at DESC").
		Find(&trainings).Error
	return trainings, err
}

// FindByUserAndType 根据用户和类型查找训练数据
func (r *bayesianTrainingRepository) FindByUserAndType(ctx context.Context, userUID string, isSpam bool) ([]*model.BayesianTraining, error) {
	var trainings []*model.BayesianTraining
	err := r.db.WithContext(ctx).
		Where("user_uid = ? AND is_spam = ?", userUID, isSpam).
		Order("created_at DESC").
		Find(&trainings).Error
	return trainings, err
}

// CountByUser 统计用户的训练数据数量
func (r *bayesianTrainingRepository) CountByUser(ctx context.Context, userUID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.BayesianTraining{}).
		Where("user_uid = ?", userUID).
		Count(&count).Error
	return count, err
}

// CountByUserAndType 统计用户特定类型的训练数据数量
func (r *bayesianTrainingRepository) CountByUserAndType(ctx context.Context, userUID string, isSpam bool) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.BayesianTraining{}).
		Where("user_uid = ? AND is_spam = ?", userUID, isSpam).
		Count(&count).Error
	return count, err
}

// DeleteByUser 删除用户的所有训练数据
func (r *bayesianTrainingRepository) DeleteByUser(ctx context.Context, userUID string) error {
	return r.db.WithContext(ctx).
		Where("user_uid = ?", userUID).
		Delete(&model.BayesianTraining{}).Error
}

// DeleteByEmail 删除特定邮件的训练数据
func (r *bayesianTrainingRepository) DeleteByEmail(ctx context.Context, emailID string) error {
	return r.db.WithContext(ctx).
		Where("email_id = ?", emailID).
		Delete(&model.BayesianTraining{}).Error
}

// List 获取训练数据列表（支持分页）
func (r *bayesianTrainingRepository) List(ctx context.Context, userUID string, offset, limit int) ([]*model.BayesianTraining, int64, error) {
	var trainings []*model.BayesianTraining
	var total int64

	query := r.db.WithContext(ctx).Model(&model.BayesianTraining{}).
		Where("user_uid = ?", userUID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&trainings).Error

	return trainings, total, err
}
