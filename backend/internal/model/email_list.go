package model

import "time"

// EmailList 白名单/黑名单模型
type EmailList struct {
	ID         int64  `gorm:"primaryKey"`
	UserUID    string `gorm:"size:64;not null;index"`                                          // 用户 UID
	Type       string `gorm:"size:20;not null;index;check:type IN ('whitelist', 'blacklist')"` // 列表类型：whitelist/blacklist
	Target     string `gorm:"size:255;not null;index"`                                         // 目标（邮箱地址或域名）
	TargetType string `gorm:"size:20;not null;check:target_type IN ('email', 'domain')"`       // 目标类型：email/domain
	Reason     string `gorm:"type:text"`                                                       // 添加原因
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName 指定表名
func (EmailList) TableName() string {
	return "email_lists"
}
