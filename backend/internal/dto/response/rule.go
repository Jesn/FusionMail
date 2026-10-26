package response

// TestRuleResponse 测试规则响应
type TestRuleResponse struct {
	Matched bool         `json:"matched"` // 是否匹配
	Rule    *RuleSummary `json:"rule"`    // 规则信息（对外摘要，不含存储层字段）
}
