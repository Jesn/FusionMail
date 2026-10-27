package model

import "time"

// SpamRule 垃圾邮件规则模型
type SpamRule struct {
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"size:255;not null"`                                                                                // 规则名称
	Description string `gorm:"type:text"`                                                                                        // 规则描述
	Category    string `gorm:"size:50;index;check:category IN ('keyword', 'pattern', 'header', 'content', 'url', 'attachment')"` // 规则类别
	Pattern     string `gorm:"type:text;not null"`                                                                               // 匹配模式（关键词或正则表达式）
	Score       int    `gorm:"default:10"`                                                                                       // 评分权重
	Enabled     bool   `gorm:"index"`                                                                                            // 是否启用
	IsBuiltin   bool   `gorm:"default:false;index"`                                                                              // 是否为内置规则
	HitCount    int64  `gorm:"default:0"`                                                                                        // 命中次数
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName 指定表名
func (SpamRule) TableName() string {
	return "spam_rules"
}
