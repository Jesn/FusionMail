package request

// MarkAllAsReadRequest 全部标记为已读请求
type MarkAllAsReadRequest struct {
	AccountUID *string `json:"account_uid"` // 账户 UID，为空则应用于所有账号
}
