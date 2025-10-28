package model

import (
	"time"
)

// EmailRule 邮件规则模型
type EmailRule struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:255;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// 规则配置
	Enabled  bool `gorm:"default:true;index" json:"enabled"`
	Priority int  `gorm:"default:0;index" json:"priority"` // 优先级（数字越小越高）

	// 触发条件（JSON）
	Conditions string `gorm:"type:text;not null" json:"conditions"`

	// 执行动作（JSON）
	Actions string `gorm:"type:text;not null" json:"actions"`

	// 统计信息
	MatchedCount  int        `gorm:"default:0" json:"matched_count"`
	LastMatchedAt *time.Time `json:"last_matched_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (EmailRule) TableName() string {
	return "email_rules"
}
