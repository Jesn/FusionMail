package model

import "time"

// RuleMatchLog 规则命中记录：单封邮件维度回答"命中了哪条规则、执行了什么动作"。
// 历史缺陷（AI 分析文档 §1.1.2）：命中只累加 EmailRule.MatchedCount 计数器，
// 用户无法调试"这封邮件为什么被自动归档"。
type RuleMatchLog struct {
	ID int64 `gorm:"primaryKey"`

	EmailID int64 `gorm:"not null;index"` // 被处理的邮件
	RuleID  int64 `gorm:"not null;index"` // 命中的规则（冗余 RuleName：规则删除后记录仍可读）
	RuleName string `gorm:"size:255;not null"`

	// 执行的动作摘要（规则配置的 Actions JSON，executeActions 全部成功后记录）
	Actions string `gorm:"type:text"`

	CreatedAt time.Time
}

// TableName 指定表名
func (RuleMatchLog) TableName() string {
	return "rule_match_logs"
}
