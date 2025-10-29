package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// RuleCondition 规则条件
type RuleCondition struct {
	Field    string `json:"field"`    // 字段：from, to, subject, body, has_attachment
	Operator string `json:"operator"` // 操作符：contains, not_contains, equals, not_equals, starts_with, ends_with, regex
	Value    string `json:"value"`    // 值
}

// RuleAction 规则动作
type RuleAction struct {
	Type  string `json:"type"`  // 动作类型：add_label, remove_label, mark_read, mark_unread, star, unstar, archive, delete, move_folder, webhook
	Value string `json:"value"` // 动作参数
}

// RuleConditions 条件数组类型
type RuleConditions []RuleCondition

// Scan 实现 sql.Scanner 接口
func (rc *RuleConditions) Scan(value interface{}) error {
	if value == nil {
		*rc = []RuleCondition{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, rc)
}

// Value 实现 driver.Valuer 接口
func (rc RuleConditions) Value() (driver.Value, error) {
	if len(rc) == 0 {
		return "[]", nil
	}
	return json.Marshal(rc)
}

// RuleActions 动作数组类型
type RuleActions []RuleAction

// Scan 实现 sql.Scanner 接口
func (ra *RuleActions) Scan(value interface{}) error {
	if value == nil {
		*ra = []RuleAction{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, ra)
}

// Value 实现 driver.Valuer 接口
func (ra RuleActions) Value() (driver.Value, error) {
	if len(ra) == 0 {
		return "[]", nil
	}
	return json.Marshal(ra)
}

// EmailRule 邮件规则模型
type EmailRule struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:255;not null" json:"name"`
	AccountUID  string `gorm:"size:64;not null;index" json:"account_uid"` // 所属账户
	Description string `gorm:"type:text" json:"description"`

	// 规则配置
	Enabled        bool   `gorm:"default:true;index" json:"enabled"`
	Priority       int    `gorm:"default:0;index" json:"priority"`         // 优先级（数字越小越高）
	MatchMode      string `gorm:"size:10;default:'all'" json:"match_mode"` // 匹配模式：all（所有条件）或 any（任意条件）
	StopProcessing bool   `gorm:"default:false" json:"stop_processing"`    // 匹配后是否停止处理后续规则

	// 触发条件（JSON）
	Conditions RuleConditions `gorm:"type:text;not null" json:"conditions"`

	// 执行动作（JSON）
	Actions RuleActions `gorm:"type:text;not null" json:"actions"`

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
