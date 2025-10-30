package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

// WebhookRepository Webhook 数据仓库接口
type WebhookRepository interface {
	Create(ctx context.Context, webhook *model.Webhook) error
	FindByID(ctx context.Context, id int64) (*model.Webhook, error)
	Update(ctx context.Context, webhook *model.Webhook) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*model.Webhook, int64, error)
	ListEnabled(ctx context.Context) ([]*model.Webhook, error)
	UpdateCallStats(ctx context.Context, id int64, success bool) error
	Enable(ctx context.Context, id int64) error
	Disable(ctx context.Context, id int64) error

	// 系统管理需要的方法
	Count(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)
}

// webhookRepository Webhook 数据仓库实现
type webhookRepository struct {
	db *gorm.DB
}

// NewWebhookRepository 创建 Webhook 数据仓库实例
func NewWebhookRepository(db *gorm.DB) WebhookRepository {
	return &webhookRepository{db: db}
}

// Create 创建 Webhook
func (r *webhookRepository) Create(ctx context.Context, webhook *model.Webhook) error {
	return r.db.WithContext(ctx).Create(webhook).Error
}

// FindByID 根据 ID 查找 Webhook
func (r *webhookRepository) FindByID(ctx context.Context, id int64) (*model.Webhook, error) {
	var webhook model.Webhook
	err := r.db.WithContext(ctx).First(&webhook, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &webhook, nil
}

// Update 更新 Webhook
func (r *webhookRepository) Update(ctx context.Context, webhook *model.Webhook) error {
	return r.db.WithContext(ctx).Save(webhook).Error
}

// Delete 删除 Webhook
func (r *webhookRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Webhook{}, id).Error
}

// List 获取 Webhook 列表
func (r *webhookRepository) List(ctx context.Context, offset, limit int) ([]*model.Webhook, int64, error) {
	var webhooks []*model.Webhook
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.Webhook{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&webhooks).Error

	return webhooks, total, err
}

// ListEnabled 获取启用的 Webhook 列表
func (r *webhookRepository) ListEnabled(ctx context.Context) ([]*model.Webhook, error) {
	var webhooks []*model.Webhook
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&webhooks).Error
	return webhooks, err
}

// UpdateCallStats 更新调用统计
func (r *webhookRepository) UpdateCallStats(ctx context.Context, id int64, success bool) error {
	updates := map[string]interface{}{
		"total_calls":    gorm.Expr("total_calls + 1"),
		"last_called_at": gorm.Expr("NOW()"),
	}

	if success {
		updates["success_calls"] = gorm.Expr("success_calls + 1")
		updates["last_status"] = "success"
	} else {
		updates["failed_calls"] = gorm.Expr("failed_calls + 1")
		updates["last_status"] = "failed"
	}

	return r.db.WithContext(ctx).
		Model(&model.Webhook{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Enable 启用 Webhook
func (r *webhookRepository) Enable(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.Webhook{}).
		Where("id = ?", id).
		Update("enabled", true).Error
}

// Disable 禁用 Webhook
func (r *webhookRepository) Disable(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.Webhook{}).
		Where("id = ?", id).
		Update("enabled", false).Error
}

// Count 统计 Webhook 总数
func (r *webhookRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Webhook{}).Count(&count).Error
	return count, err
}

// CountActive 统计活跃 Webhook 数（启用的 Webhook）
func (r *webhookRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Webhook{}).
		Where("enabled = ?", true).
		Count(&count).Error
	return count, err
}

// WebhookLogRepository Webhook 日志数据仓库接口
type WebhookLogRepository interface {
	Create(ctx context.Context, log *model.WebhookLog) error
	FindByWebhookID(ctx context.Context, webhookID int64, offset, limit int) ([]*model.WebhookLog, int64, error)
	DeleteOldLogs(ctx context.Context, beforeDate time.Time) error
}

// webhookLogRepository Webhook 日志数据仓库实现
type webhookLogRepository struct {
	db *gorm.DB
}

// NewWebhookLogRepository 创建 Webhook 日志数据仓库实例
func NewWebhookLogRepository(db *gorm.DB) WebhookLogRepository {
	return &webhookLogRepository{db: db}
}

// Create 创建 Webhook 日志
func (r *webhookLogRepository) Create(ctx context.Context, log *model.WebhookLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// FindByWebhookID 根据 Webhook ID 查找日志
func (r *webhookLogRepository) FindByWebhookID(ctx context.Context, webhookID int64, offset, limit int) ([]*model.WebhookLog, int64, error) {
	var logs []*model.WebhookLog
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.WebhookLog{}).Where("webhook_id = ?", webhookID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := r.db.WithContext(ctx).
		Where("webhook_id = ?", webhookID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

// DeleteOldLogs 删除旧日志
func (r *webhookLogRepository) DeleteOldLogs(ctx context.Context, beforeDate time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", beforeDate).
		Delete(&model.WebhookLog{}).Error
}
