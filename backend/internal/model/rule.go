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
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*rc = []RuleCondition{}
			return nil
		}
		return json.Unmarshal(v, rc)
	case string:
		if v == "" {
			*rc = []RuleCondition{}
			return nil
		}
		return json.Unmarshal([]byte(v), rc)
	default:
		// 无法识别的类型，回退为空数组，避免返回 nil 造成 JSON null
		*rc = []RuleCondition{}
		return nil
	}
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
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*ra = []RuleAction{}
			return nil
		}
		return json.Unmarshal(v, ra)
	case string:
		if v == "" {
			*ra = []RuleAction{}
			return nil
		}
		return json.Unmarshal([]byte(v), ra)
	default:
		*ra = []RuleAction{}
		return nil
	}
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
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"size:255;not null"`
	AccountUID  string `gorm:"size:64;not null;index"` // 所属账户
	Description string `gorm:"type:text"`

	// 规则配置
	Enabled        bool   `gorm:"default:true;index"`
	Priority       int    `gorm:"default:0;index"`       // 优先级（数字越小越高）
	MatchMode      string `gorm:"size:10;default:'all'"` // 匹配模式：all（所有条件）或 any（任意条件）
	StopProcessing bool   `gorm:"default:false"`         // 匹配后是否停止处理后续规则

	// 触发条件（JSON）
	Conditions RuleConditions `gorm:"type:text;not null"`

	// 执行动作（JSON）
	Actions RuleActions `gorm:"type:text;not null"`

	// 统计信息
	MatchedCount  int `gorm:"default:0"`
	LastMatchedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (EmailRule) TableName() string {
	return "email_rules"
}
