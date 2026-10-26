package response

import (
	"fusionmail/internal/model"
	"time"
)

// PublicEmailItem 是公开邮件 API（/api/v1/public/mail/*）的响应条目。
//
// 为什么需要它：这些接口此前直接返回 []*model.Email，而 model.Email 同时承担
// GORM 存储映射与 JSON 序列化，导致内部字段外泄——其中 DedupeKey（去重标识）
// 与 SyncedAt（内部同步时间戳）正是 EmailDetailResponse 明确声明要隐藏的字段。
// 更糟的是 MailListData.Emails 被声明为 interface{}，类型擦除使编译器无法拦截，
// 未来给 model 新增字段会继续静默泄露。
//
// 因此这里逐字段显式列出对外契约：新增 model 字段默认不���出现在公开响应里。
type PublicEmailItem struct {
	ID         int64  `json:"id"`
	ProviderID string `json:"provider_id"` // 邮箱服务商原生 ID
	AccountUID string `json:"account_uid"` // 所属账户 UID
	MessageID  string `json:"message_id"`  // 邮件 Message-ID

	// 基本信息
	Subject     string `json:"subject"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	ToAddress   string `json:"to_address"`
	ToAddresses string `json:"to_addresses"` // JSON 数组
	CcAddresses string `json:"cc_addresses"` // JSON 数组
	ReplyTo     string `json:"reply_to"`

	// 内容
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body"`
	Snippet  string `json:"snippet"`

	// 本地状态（只读镜像）
	IsRead     bool   `json:"is_read"`
	IsStarred  bool   `json:"is_starred"`
	IsArchived bool   `json:"is_archived"`
	IsDeleted  bool   `json:"is_deleted"`
	Labels     string `json:"labels"` // JSON 数组
	Folder     string `json:"folder"`

	// 附件
	HasAttachment    bool `json:"has_attachment"`
	AttachmentsCount int  `json:"attachments_count"`

	// 时间
	SentAt     time.Time `json:"sent_at"`
	ReceivedAt time.Time `json:"received_at"`

	// 元数据
	SizeBytes  int64  `json:"size_bytes"`
	ThreadID   string `json:"thread_id"`
	InReplyTo  string `json:"in_reply_to"`
	References string `json:"references"`

	// 垃圾邮件判定结果（对外只暴露结论，不暴露评分��检测细节）
	IsSpam bool `json:"is_spam"`
}

// NewPublicEmailItem 把存储模型映射为公开响应条目。
//
// 显式逐字段赋值：这是"新增 model 字段不会自动进入公开契约"的保证点，
// 不要改成结构体嵌入或反射拷贝。
func NewPublicEmailItem(e *model.Email) PublicEmailItem {
	if e == nil {
		return PublicEmailItem{}
	}
	return PublicEmailItem{
		ID:         e.ID,
		ProviderID: e.ProviderID,
		AccountUID: e.AccountUID,
		MessageID:  e.MessageID,

		Subject:     e.Subject,
		FromAddress: e.FromAddress,
		FromName:    e.FromName,
		ToAddress:   e.ToAddress,
		ToAddresses: e.ToAddresses,
		CcAddresses: e.CcAddresses,
		ReplyTo:     e.ReplyTo,

		TextBody: e.TextBody,
		HTMLBody: e.HTMLBody,
		Snippet:  e.Snippet,

		IsRead:     e.IsRead,
		IsStarred:  e.IsStarred,
		IsArchived: e.IsArchived,
		IsDeleted:  e.IsDeleted,
		Labels:     e.Labels,
		Folder:     e.Folder,

		HasAttachment:    e.HasAttachment,
		AttachmentsCount: e.AttachmentsCount,

		SentAt:     e.SentAt,
		ReceivedAt: e.ReceivedAt,

		SizeBytes:  e.SizeBytes,
		ThreadID:   e.ThreadID,
		InReplyTo:  e.InReplyTo,
		References: e.References,

		IsSpam: e.IsSpam,
	}
}

// NewPublicEmailItems 批量映射。
func NewPublicEmailItems(emails []*model.Email) []PublicEmailItem {
	items := make([]PublicEmailItem, 0, len(emails))
	for _, e := range emails {
		items = append(items, NewPublicEmailItem(e))
	}
	return items
}
