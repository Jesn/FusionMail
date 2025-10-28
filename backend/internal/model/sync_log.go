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
	EmailsFetched int `gorm:"default:0" json:"emails_fetched"`
	EmailsNew     int `gorm:"default:0" json:"emails_new"`
	EmailsUpdated int `gorm:"default:0" json:"emails_updated"`

	// 时间信息
	StartedAt   time.Time  `gorm:"not null;index:idx_started_at,sort:desc" json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	DurationMs  int        `json:"duration_ms"`

	// 错误信息
	ErrorMessage string `gorm:"type:text" json:"error_message"`
	ErrorStack   string `gorm:"type:text" json:"error_stack"`
}

// TableName 指定表名
func (SyncLog) TableName() string {
	return "sync_logs"
}
