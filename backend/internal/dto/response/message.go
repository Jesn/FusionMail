package response

// MessageResponse 通用消息响应（data 仅含一个 message 键）。
//
// 替换各 handler 里的 gin.H{"message": "..."}（键稳定），
// JSON 输出逐键一致。仅用于 data 形状确为 {"message": string} 的端点。
type MessageResponse struct {
	Message string `json:"message"`
}
