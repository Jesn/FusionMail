package response

// RuleMatchLogItem 单封邮件的规则命中记录项
type RuleMatchLogItem struct {
	ID        int64  `json:"id"`
	RuleID    int64  `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	Actions   string `json:"actions"` // 规则配置的动作 JSON 数组
	MatchedAt string `json:"matched_at"`
}

// RuleMatchLogListResponse 邮件规则命中记录列表响应
type RuleMatchLogListResponse struct {
	Matches []RuleMatchLogItem `json:"matches"`
}
