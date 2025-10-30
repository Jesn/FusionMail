package repository

import (
	"context"
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// SyncLogRepository 同步日志数据仓库接口
type SyncLogRepository interface {
	Create(ctx context.Context, log *model.SyncLog) error
	FindByID(ctx context.Context, id int64) (*model.SyncLog, error)
	List(ctx context.Context, accountUID string, offset, limit int) ([]*model.SyncLog, int64, error)
	ListByStatus(ctx context.Context, status string, offset, limit int) ([]*model.SyncLog, int64, error)
	DeleteOldLogs(ctx context.Context, days int) error

	// 系统管理需要的方法
	Count(ctx context.Context, status string) (int64, error)
	GetLatest(ctx context.Context) (*model.SyncLog, error)
	GetLatestByAccount(ctx context.Context, accountUID string) (*model.SyncLog, error)
	FindWithPagination(ctx context.Context, offset, limit int, accountUID, status string) ([]*model.SyncLog, int64, error)
}

// syncLogRepository 同步日志数据仓库实现
type syncLogRepository struct {
	db *gorm.DB
}

// NewSyncLogRepository 创建同步日志数据仓库实例
func NewSyncLogRepository(db *gorm.DB) SyncLogRepository {
	return &syncLogRepository{db: db}
}

// Create 创建同步日志
func (r *syncLogRepository) Create(ctx context.Context, log *model.SyncLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// FindByID 根据 ID 查找同步日志
func (r *syncLogRepository) FindByID(ctx context.Context, id int64) (*model.SyncLog, error) {
	var log model.SyncLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List 获取同步日志列表
func (r *syncLogRepository) List(ctx context.Context, accountUID string, offset, limit int) ([]*model.SyncLog, int64, error) {
	var logs []*model.SyncLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SyncLog{})

	if accountUID != "" {
		query = query.Where("account_uid = ?", accountUID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&logs).Error

	return logs, total, err
}

// ListByStatus 根据状态获取同步日志列表
func (r *syncLogRepository) ListByStatus(ctx context.Context, status string, offset, limit int) ([]*model.SyncLog, int64, error) {
	var logs []*model.SyncLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SyncLog{}).
		Where("status = ?", status)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&logs).Error

	return logs, total, err
}

// DeleteOldLogs 删除旧日志
func (r *syncLogRepository) DeleteOldLogs(ctx context.Context, days int) error {
	return r.db.WithContext(ctx).
		Where("started_at < NOW() - INTERVAL '? days'", days).
		Delete(&model.SyncLog{}).Error
}

// Count 统计同步日志数量
func (r *syncLogRepository) Count(ctx context.Context, status string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.SyncLog{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Count(&count).Error
	return count, err
}

// GetLatest 获取最新的同步日志
func (r *syncLogRepository) GetLatest(ctx context.Context) (*model.SyncLog, error) {
	var log model.SyncLog
	err := r.db.WithContext(ctx).
		Order("started_at DESC").
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetLatestByAccount 获取指定账户的最新同步日志
func (r *syncLogRepository) GetLatestByAccount(ctx context.Context, accountUID string) (*model.SyncLog, error) {
	var log model.SyncLog
	err := r.db.WithContext(ctx).
		Where("account_uid = ?", accountUID).
		Order("started_at DESC").
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// FindWithPagination 分页查询同步日志
func (r *syncLogRepository) FindWithPagination(ctx context.Context, offset, limit int, accountUID, status string) ([]*model.SyncLog, int64, error) {
	var logs []*model.SyncLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SyncLog{})

	// 添加筛选条件
	if accountUID != "" {
		query = query.Where("account_uid = ?", accountUID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&logs).Error

	return logs, total, err
}
