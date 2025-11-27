package request

// MarkSpamRequest 标记垃圾邮件请求
type MarkSpamRequest struct {
	EmailIDs []int64 `json:"email_ids" binding:"required"` // 邮件 ID 列表
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	EmailIDs []int64 `json:"email_ids" binding:"required"` // 邮件 ID 列表
}
