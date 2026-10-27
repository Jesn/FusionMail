package response

import (
	"fusionmail/internal/model"
	"time"
)

// EmailRuleResponse 是邮件规则的响应 DTO。
//
// 与 model.EmailRule 逐字段 JSON 等价（model 未声明 json tag，
// 序列化键为 Go 字段名，这里用显式 tag 保持一致，字段声明顺序也与 model 一致）。
// 仅用于 Swagger 契约声明，避免带 gorm tag 的持久化模型进入 API 文档。
//
// 注意：新端点对外暴露规则时优先考虑 RuleSummary（不含存储层时间戳）；
// 本类型的存在只是为了给既有端点（原样返回 model.EmailRule）提供等价契约。
type EmailRuleResponse struct {
	ID          int64  `json:"ID"`
	Name        string `json:"Name"`
	AccountUID  string `json:"AccountUID"`
	Description string `json:"Description"`

	// 规则配置
	Enabled        bool   `json:"Enabled"`
	Priority       int    `json:"Priority"`
	MatchMode      string `json:"MatchMode"`
	StopProcessing bool   `json:"StopProcessing"`

	// 触发条件（JSON）
	Conditions model.RuleConditions `json:"Conditions"`

	// 执行动作（JSON）
	Actions model.RuleActions `json:"Actions"`

	// 统计信息
	MatchedCount  int        `json:"MatchedCount"`
	LastMatchedAt *time.Time `json:"LastMatchedAt"`

	CreatedAt time.Time `json:"CreatedAt"`
	UpdatedAt time.Time `json:"UpdatedAt"`
}

// NewEmailRuleResponse 把存储模型映射为响应 DTO。
func NewEmailRuleResponse(r *model.EmailRule) *EmailRuleResponse {
	if r == nil {
		return nil
	}
	return &EmailRuleResponse{
		ID:             r.ID,
		Name:           r.Name,
		AccountUID:     r.AccountUID,
		Description:    r.Description,
		Enabled:        r.Enabled,
		Priority:       r.Priority,
		MatchMode:      r.MatchMode,
		StopProcessing: r.StopProcessing,
		Conditions:     r.Conditions,
		Actions:        r.Actions,
		MatchedCount:   r.MatchedCount,
		LastMatchedAt:  r.LastMatchedAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// NewEmailRuleResponses 批量映射（保持 nil/空切片语义不变）。
func NewEmailRuleResponses(rules []*model.EmailRule) []*EmailRuleResponse {
	if rules == nil {
		return nil
	}
	out := make([]*EmailRuleResponse, 0, len(rules))
	for _, r := range rules {
		out = append(out, NewEmailRuleResponse(r))
	}
	return out
}
