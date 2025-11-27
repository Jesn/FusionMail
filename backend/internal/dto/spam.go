package dto

// MarkSpamRequest 标记垃圾邮件请求
type MarkSpamRequest struct {
	EmailIDs []int64 `json:"email_ids" binding:"required,min=1"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	EmailIDs []int64 `json:"email_ids" binding:"required,min=1"`
}

// SpamStatsResponse 垃圾邮件统计响应
type SpamStatsResponse struct {
	TotalCount   int64 `json:"total_count"`   // 总垃圾邮件数
	UnreadCount  int64 `json:"unread_count"`  // 未读垃圾邮件数
	TodayCount   int64 `json:"today_count"`   // 今日垃圾邮件数
	WeekCount    int64 `json:"week_count"`    // 本周垃圾邮件数
	MonthCount   int64 `json:"month_count"`   // 本月垃圾邮件数
	BlockedCount int64 `json:"blocked_count"` // 拦截的垃圾邮件数
}

// UpdateReputationRequest 更新发件人信誉请求
type UpdateReputationRequest struct {
	Email  string `json:"email" binding:"required,email"` // 发件人邮箱
	IsSpam bool   `json:"is_spam"`                        // 是否为垃圾邮件
	Delta  int    `json:"delta"`                          // 评分变化量（可选，优先使用）
}

// ReputationResponse 发件人信誉响应
type ReputationResponse struct {
	Email        string  `json:"email"`          // 发件人邮箱
	Domain       string  `json:"domain"`         // 发件人域名
	Score        float64 `json:"score"`          // 信誉评分
	TrustLevel   string  `json:"trust_level"`    // 信任级别
	TotalEmails  int64   `json:"total_emails"`   // 总邮件数
	SpamCount    int64   `json:"spam_count"`     // 垃圾邮件数
	HamCount     int64   `json:"ham_count"`      // 正常邮件数
	SpamRate     float64 `json:"spam_rate"`      // 垃圾邮件率
	RBLStatus    string  `json:"rbl_status"`     // RBL 状态
	RBLCheckedAt *string `json:"rbl_checked_at"` // RBL 检查时间
	CreatedAt    string  `json:"created_at"`     // 创建时间
	UpdatedAt    string  `json:"updated_at"`     // 更新时间
}

// ReputationStatsResponse 发件人信誉统计响应
type ReputationStatsResponse struct {
	TotalSenders    int64   `json:"total_senders"`    // 总发件人数
	TrustedCount    int64   `json:"trusted_count"`    // 可信发件人数
	NeutralCount    int64   `json:"neutral_count"`    // 中立发件人数
	SuspiciousCount int64   `json:"suspicious_count"` // 可疑发件人数
	BlockedCount    int64   `json:"blocked_count"`    // 已拦截发件人数
	AverageScore    float64 `json:"average_score"`    // 平均信誉评分
	RBLListedCount  int64   `json:"rbl_listed_count"` // RBL 黑名单数
}
