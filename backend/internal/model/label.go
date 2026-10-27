package model

import (
	"time"
)

// EmailLabel 邮件标签模型
type EmailLabel struct {
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex;size:100;not null"`
	Color       string `gorm:"size:20"` // 标签颜色
	Description string `gorm:"type:text"`

	// 统计信息
	EmailCount int `gorm:"default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (EmailLabel) TableName() string {
	return "email_labels"
}

// EmailLabelRelation 邮件-标签关联模型
type EmailLabelRelation struct {
	EmailID   int64 `gorm:"primaryKey"`
	LabelID   int64 `gorm:"primaryKey"`
	CreatedAt time.Time
}

// TableName 指定表名
func (EmailLabelRelation) TableName() string {
	return "email_label_relations"
}
