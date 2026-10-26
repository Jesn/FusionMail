package response

import (
	"fusionmail/internal/model"
	"time"
)

// SentEmailResponse 是已发送邮件的对外响应视图。
//
// 为什么需要它：service.ListSentEmailsResponse.Emails 此前直接是
// []*model.SentEmail（带 gorm tag 的持久化模型），使 created_at / updated_at /
// deleted_at 等存储层字段进入 API 契约。
//
// 逐字段显式列出对外契约：新增 model 字段默认不会出现在响应里。
type SentEmailResponse struct {
	ID            int64  `json:"id"`
	AccountUID    string `json:"account_uid"`
	ProviderMsgID string `json:"provider_msg_id,omitempty"`
	MessageID     string `json:"message_id,omitempty"`

	// 基本信息
	Subject      string `json:"subject"`
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	ToAddresses  string `json:"to_addresses"`  // JSON 数组
	CcAddresses  string `json:"cc_addresses"`  // JSON 数组
	BccAddresses string `json:"bcc_addresses"` // JSON 数组

	// 内容
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body"`

	// 附件
	HasAttachments  bool   `json:"has_attachments"`
	AttachmentCount int    `json:"attachment_count"`
	AttachmentInfo  string `json:"attachment_info,omitempty"` // JSON 数组

	// 关联
	ReplyToEmailID *int64 `json:"reply_to_email_id,omitempty"`
	ForwardFromID  *int64 `json:"forward_from_id,omitempty"`
	InReplyTo      string `json:"in_reply_to,omitempty"`
	References     string `json:"references,omitempty"`

	// 发送状态
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	SentAt       time.Time  `json:"sent_at"`
	RetryCount   int        `json:"retry_count"`
	LastRetryAt  *time.Time `json:"last_retry_at,omitempty"`

	SenderType string `json:"sender_type"`
	SizeBytes  int64  `json:"size_bytes"`
}

// NewSentEmailResponse 把存储模型映射为对外响应。
func NewSentEmailResponse(e *model.SentEmail) *SentEmailResponse {
	if e == nil {
		return nil
	}
	return &SentEmailResponse{
		ID:            e.ID,
		AccountUID:    e.AccountUID,
		ProviderMsgID: e.ProviderMsgID,
		MessageID:     e.MessageID,

		Subject:      e.Subject,
		FromAddress:  e.FromAddress,
		FromName:     e.FromName,
		ToAddresses:  e.ToAddresses,
		CcAddresses:  e.CcAddresses,
		BccAddresses: e.BccAddresses,

		TextBody: e.TextBody,
		HTMLBody: e.HTMLBody,

		HasAttachments:  e.HasAttachments,
		AttachmentCount: e.AttachmentCount,
		AttachmentInfo:  e.AttachmentInfo,

		ReplyToEmailID: e.ReplyToEmailID,
		ForwardFromID:  e.ForwardFromID,
		InReplyTo:      e.InReplyTo,
		References:     e.References,

		Status:       e.Status,
		ErrorMessage: e.ErrorMessage,
		SentAt:       e.SentAt,
		RetryCount:   e.RetryCount,
		LastRetryAt:  e.LastRetryAt,

		SenderType: e.SenderType,
		SizeBytes:  e.SizeBytes,
	}
}

// NewSentEmailResponses 批量映射。
func NewSentEmailResponses(emails []*model.SentEmail) []*SentEmailResponse {
	out := make([]*SentEmailResponse, 0, len(emails))
	for _, e := range emails {
		out = append(out, NewSentEmailResponse(e))
	}
	return out
}
