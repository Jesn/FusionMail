package response

import "fusionmail/internal/model"

// TestRuleResponse 测试规则响应
type TestRuleResponse struct {
	Matched bool             `json:"matched"` // 是否匹配
	Rule    *model.EmailRule `json:"rule"`    // 规则信息
}
