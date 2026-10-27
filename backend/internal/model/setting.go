package model

import "time"

// Setting 系统配置模型
// 用于存储系统级和用户级的配置信息，支持敏感数据加密存储
type Setting struct {
	ID          int64  `gorm:"primaryKey"`
	UserID      *int64 `gorm:"index;null"`                        // NULL表示系统级配置，非NULL表示用户级配置
	Category    string `gorm:"size:50;not null;index"`            // 配置分类：ui, sync, security, oauth等
	Key         string `gorm:"size:100;not null"`                 // 配置键名
	Value       string `gorm:"type:text;not null"`                // 配置值（敏感数据会加密存储）
	ValueType   string `gorm:"size:20;not null;default:'string'"` // 值类型：string, number, boolean, json
	IsSensitive bool   `gorm:"default:false;index"`               // 是否敏感配置（需加密）
	IsPublic    bool   `gorm:"default:false"`                     // 是否公开（前端可获取）
	Description string `gorm:"type:text"`                         // 配置描述

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (Setting) TableName() string {
	return "settings"
}

// UniqueKey 返回用户ID、分类、键的组合（用于唯一性检查）
func (s Setting) UniqueKey() (userID *int64, category, key string) {
	return s.UserID, s.Category, s.Key
}

// IsSystemLevel 判断是否为系统级配置
func (s Setting) IsSystemLevel() bool {
	return s.UserID == nil
}

// IsUserLevel 判断是否为用户级配置
func (s Setting) IsUserLevel() bool {
	return s.UserID != nil
}
