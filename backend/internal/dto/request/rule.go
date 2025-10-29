package request

import "fusionmail/internal/model"

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name           string                `json:"name" binding:"required"`             // 规则名称
	Description    string                `json:"description"`                         // 规则描述
	AccountUID     string                `json:"account_uid" binding:"required"`      // 账户 UID
	Enabled        bool                  `json:"enabled"`                             // 是否启用
	Priority       int                   `json:"priority"`                            // 优先级（数字越小优先级越高）
	MatchMode      string                `json:"match_mode" binding:"required"`       // 匹配模式：all（所有条件）或 any（任意条件）
	Conditions     []model.RuleCondition `json:"conditions" binding:"required,min=1"` // 条件列表
	Actions        []model.RuleAction    `json:"actions" binding:"required,min=1"`    // 动作列表
	StopProcessing bool                  `json:"stop_processing"`                     // 匹配后是否停止处理后续规则
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name           *string               `json:"name"`            // 规则名称
	Description    *string               `json:"description"`     // 规则描述
	Enabled        *bool                 `json:"enabled"`         // 是否启用
	Priority       *int                  `json:"priority"`        // 优先级
	MatchMode      *string               `json:"match_mode"`      // 匹配模式
	Conditions     []model.RuleCondition `json:"conditions"`      // 条件列表
	Actions        []model.RuleAction    `json:"actions"`         // 动作列表
	StopProcessing *bool                 `json:"stop_processing"` // 是否停止处理后续规则
}

// TestRuleRequest 测试规则请求
type TestRuleRequest struct {
	FromAddress   string `json:"from_address"`   // 发件人地址
	ToAddress     string `json:"to_address"`     // 收件人地址
	Subject       string `json:"subject"`        // 邮件主题
	Body          string `json:"body"`           // 邮件正文
	HasAttachment bool   `json:"has_attachment"` // 是否有附件
}
