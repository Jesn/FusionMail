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
