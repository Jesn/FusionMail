package response

import (
	"fusionmail/internal/model"
	"time"
)

// RuleSummary 是规则对外暴露的摘要视图。
//
// 为什么需要它：response.TestRuleResponse 此前直接内嵌 *model.EmailRule
// （带 gorm tag 的持久化模型），使 created_at / updated_at 等存储层字段
// 进入 API 契约。这个"DTO"只是包了一层，没有解耦。
//
// 逐字段显式列出对外契约：新增 model 字段默认不会出现在响应里。
type RuleSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	AccountUID  string `json:"account_uid"`
	Description string `json:"description"`

	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority"`
	MatchMode      string `json:"match_mode"`
	StopProcessing bool   `json:"stop_processing"`

	Conditions model.RuleConditions `json:"conditions"`
	Actions    model.RuleActions    `json:"actions"`

	// 统计信息（业务可见，非存储细节）
	MatchedCount  int        `json:"matched_count"`
	LastMatchedAt *time.Time `json:"last_matched_at,omitempty"`
}

// NewRuleSummary 把存储模型映射为对外摘要。
func NewRuleSummary(r *model.EmailRule) *RuleSummary {
	if r == nil {
		return nil
	}
	return &RuleSummary{
		ID:          r.ID,
		Name:        r.Name,
		AccountUID:  r.AccountUID,
		Description: r.Description,

		Enabled:        r.Enabled,
		Priority:       r.Priority,
		MatchMode:      r.MatchMode,
		StopProcessing: r.StopProcessing,

		Conditions: r.Conditions,
		Actions:    r.Actions,

		MatchedCount:  r.MatchedCount,
		LastMatchedAt: r.LastMatchedAt,
	}
}
