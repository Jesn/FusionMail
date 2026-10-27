package response

// BatchEmailDeleteResponse 批量/清空删除邮件的响应。
//
// 替换 handler 里的 gin.H（键稳定：deleted_count/message），
// 字段按字母序排列，与 map 序列化的输出逐键一致。
// 适用于：批量删除、批量永久删除、清空回收站。
type BatchEmailDeleteResponse struct {
	DeletedCount int64  `json:"deleted_count"`
	Message      string `json:"message"`
}

// MarkAllAsReadResponse 全部标记已读的响应。
//
// 替换 handler 里的 gin.H（键稳定：count/message），
// 字段按字母序排列以保持输出一致。
type MarkAllAsReadResponse struct {
	Count   int64  `json:"count"`
	Message string `json:"message"`
}

// UnreadCountResponse 未读邮件数响应。
//
// 替换 handler 里的 gin.H（键稳定：unread_count）。
type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}
