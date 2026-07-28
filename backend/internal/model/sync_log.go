package model

import (
	"time"
)

// SyncLog 同步日志模型
type SyncLog struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	AccountUID string `gorm:"size:64;not null;index" json:"account_uid"`

	// 同步信息
	SyncType string `gorm:"size:20;not null" json:"sync_type"`    // scheduled/manual
	Status   string `gorm:"size:20;not null;index" json:"status"` // running/success/failed

	// 统计信息
	EmailsFetched int64 `gorm:"default:0" json:"emails_fetched"`
	EmailsNew     int64 `gorm:"default:0" json:"emails_new"`
	EmailsUpdated int64 `gorm:"default:0" json:"emails_updated"`

	// 首次同步优化扩展字段 (Requirements 6.1)
	TotalEstimated int    `json:"total_estimated"`                    // 预估邮件总数
	CurrentBatch   int    `json:"current_batch"`                      // 当前处理批次
	TotalBatches   int    `json:"total_batches"`                      // 总批次数
	SyncCursor     string `gorm:"type:text" json:"sync_cursor"`       // 同步游标位置
	IsFirstSync    bool   `gorm:"default:false" json:"is_first_sync"` // 是否为首次同步

	// 时间信息
	StartedAt   time.Time  `gorm:"not null;index:idx_started_at,sort:desc" json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	DurationMs  int64      `json:"duration_ms"`

	// 错误信息
	ErrorMessage string `gorm:"type:text" json:"error_message"`
}

// TableName 指定表名
func (SyncLog) TableName() string {
	return "sync_logs"
}
