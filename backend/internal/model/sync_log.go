package model

import (
	"time"
)

// SyncLog 同步日志模型
type SyncLog struct {
	ID         int64  `gorm:"primaryKey"`
	AccountUID string `gorm:"size:64;not null;index"`

	// 同步信息
	SyncType string `gorm:"size:20;not null"`       // scheduled/manual
	Status   string `gorm:"size:20;not null;index"` // running/success/failed

	// 统计信息
	EmailsFetched int64 `gorm:"default:0"`
	EmailsNew     int64 `gorm:"default:0"`
	EmailsUpdated int64 `gorm:"default:0"`

	// 首次同步优化扩展字段 (Requirements 6.1)
	TotalEstimated int
	CurrentBatch   int
	TotalBatches   int
	SyncCursor     string `gorm:"type:text"`     // 同步游标位置
	IsFirstSync    bool   `gorm:"default:false"` // 是否为首次同步

	// 时间信息
	StartedAt   time.Time `gorm:"not null;index:idx_started_at,sort:desc"`
	CompletedAt *time.Time
	DurationMs  int64

	// 错误信息
	ErrorMessage string `gorm:"type:text"`
}

// TableName 指定表名
func (SyncLog) TableName() string {
	return "sync_logs"
}
