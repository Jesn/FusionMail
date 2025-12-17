package model

import (
	"time"
)

// DeletedEmailKey 已删除邮件的去重标识记录
// 用于防止物理删除的邮件在同步时被重新创建
type DeletedEmailKey struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	AccountUID string    `gorm:"size:64;not null;uniqueIndex:uk_deleted_email_keys,priority:1" json:"account_uid"` // 邮箱账户 UID
	DedupeKey  string    `gorm:"size:64;not null;uniqueIndex:uk_deleted_email_keys,priority:2" json:"dedupe_key"`  // 邮件去重标识
	DeletedAt  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"deleted_at"`                                      // 删除时间，用于 90 天后清理
}

// TableName 指定表名
func (DeletedEmailKey) TableName() string {
	return "deleted_email_keys"
}
