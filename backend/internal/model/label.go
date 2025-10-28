package model

import (
	"time"
)

// EmailLabel 邮件标签模型
type EmailLabel struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Color       string `gorm:"size:20" json:"color"` // 标签颜色
	Description string `gorm:"type:text" json:"description"`

	// 统计信息
	EmailCount int `gorm:"default:0" json:"email_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (EmailLabel) TableName() string {
	return "email_labels"
}

// EmailLabelRelation 邮件-标签关联模型
type EmailLabelRelation struct {
	EmailID   int64     `gorm:"primaryKey" json:"email_id"`
	LabelID   int64     `gorm:"primaryKey" json:"label_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (EmailLabelRelation) TableName() string {
	return "email_label_relations"
}
