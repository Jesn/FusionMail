package repository

import (
	"context"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

// SpamDetectionLogRepository 垃圾邮件检测日志仓库接口
type SpamDetectionLogRepository interface {
	Create(ctx context.Context, log *model.SpamDetectionLog) error
	FindByEmailID(ctx context.Context, emailID int64) ([]*model.SpamDetectionLog, error)
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*model.SpamDetectionLog, error)
	DeleteOldLogs(ctx context.Context, before time.Time) error
}

// spamDetectionLogRepository 垃圾邮件检测日志仓库实现
type spamDetectionLogRepository struct {
	db *gorm.DB
}

// NewSpamDetectionLogRepository 创建垃圾邮件检测日志仓库
func NewSpamDetectionLogRepository(db *gorm.DB) SpamDetectionLogRepository {
	return &spamDetectionLogRepository{db: db}
}

// Create 创建检测日志
func (r *spamDetectionLogRepository) Create(ctx context.Context, log *model.SpamDetectionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// FindByEmailID 根据邮件 ID 查询检测日志
func (r *spamDetectionLogRepository) FindByEmailID(ctx context.Context, emailID int64) ([]*model.SpamDetectionLog, error) {
	var logs []*model.SpamDetectionLog
	err := r.db.WithContext(ctx).
		Where("email_id = ?", emailID).
		Order("checked_at DESC").
		Find(&logs).Error
	return logs, err
}

// FindByTimeRange 根据时间范围查询检测日志
func (r *spamDetectionLogRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*model.SpamDetectionLog, error) {
	var logs []*model.SpamDetectionLog
	err := r.db.WithContext(ctx).
		Where("checked_at BETWEEN ? AND ?", startTime, endTime).
		Order("checked_at DESC").
		Find(&logs).Error
	return logs, err
}

// DeleteOldLogs 删除旧的检测日志
func (r *spamDetectionLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("checked_at < ?", before).
		Delete(&model.SpamDetectionLog{}).Error
}
