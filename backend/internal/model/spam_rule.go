package model

import "time"

// SpamRule 垃圾邮件规则模型
type SpamRule struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`                                                                                    // 规则名称
	Description string    `gorm:"type:text" json:"description"`                                                                                     // 规则描述
	Category    string    `gorm:"size:50;index;check:category IN ('keyword', 'pattern', 'header', 'content', 'url', 'attachment')" json:"category"` // 规则类别
	Pattern     string    `gorm:"type:text;not null" json:"pattern"`                                                                                // 匹配模式（关键词或正则表达式）
	Score       int       `gorm:"default:10" json:"score"`                                                                                          // 评分权重
	Enabled     bool      `gorm:"index" json:"enabled"`                                                                                             // 是否启用
	IsBuiltin   bool      `gorm:"default:false;index" json:"is_builtin"`                                                                            // 是否为内置规则
	HitCount    int64     `gorm:"default:0" json:"hit_count"`                                                                                       // 命中次数
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SpamRule) TableName() string {
	return "spam_rules"
}
