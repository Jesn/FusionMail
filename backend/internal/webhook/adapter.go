// Package webhook 提供通用的邮件 Webhook 接收器框架
// 支持多种邮件服务商（Cloudflare Temp Email、Mailgun、SendGrid、Postmark 等）
package webhook

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// WebhookAdapter 通用 Webhook 适配器接口
// 每个邮件服务商需要实现此接口以支持 webhook 推送
type WebhookAdapter interface {
	// GetProviderType 返回适配器对应的 provider 类型
	// 例如: "cloudflare_temp_email", "mailgun", "sendgrid"
	GetProviderType() string

	// GetSignatureHeader 返回签名/Secret 所在的 Header 名称
	// 例如: "X-Webhook-Secret", "X-Mailgun-Signature"
	GetSignatureHeader() string

	// ValidateRequest 验证请求的合法性（签名、Secret 等）
	// secret 从 Provider 或 EmailAccount 配置中获取
	// 返回 nil 表示验证通过，否则返回具体错误
	ValidateRequest(c *gin.Context, secret string) error

	// ParsePayload 解析原始 payload 并转换为统一格式
	// 返回标准化的邮件数据结构
	ParsePayload(c *gin.Context) (*NormalizedEmail, error)
}

// NormalizedEmail 统一的邮件数据格式
// 所有 Webhook 适配器都将原始 payload 转换为此格式
type NormalizedEmail struct {
	// ProviderID 邮件服务商的原始 ID（用于去重）
	ProviderID string `json:"provider_id"`

	// 发件人信息
	From        string `json:"from"`         // 发件人原始格式，如 "Name <email>" 或 "email"
	FromName    string `json:"from_name"`    // 发件人名称（解析后）
	FromAddress string `json:"from_address"` // 发件人地址（解析后）

	// 收件人信息
	To           string   `json:"to"`            // 主收件人地址
	ToAddresses  []string `json:"to_addresses"`  // 所有收件人地址
	CcAddresses  []string `json:"cc_addresses"`  // 抄送地址
	BccAddresses []string `json:"bcc_addresses"` // 密送地址
	ReplyTo      string   `json:"reply_to"`      // 回复地址

	// 邮件内容
	Subject    string `json:"subject"`     // 邮件主题
	TextBody   string `json:"text_body"`   // 纯文本正文
	HtmlBody   string `json:"html_body"`   // HTML 正文
	RawContent string `json:"raw_content"` // RFC822 原始内容（可选）

	// 时间信息
	SentAt     *time.Time `json:"sent_at"`     // 发送时间
	ReceivedAt *time.Time `json:"received_at"` // 接收时间

	// 邮件头信息
	MessageID  string            `json:"message_id"`  // Message-ID 头
	InReplyTo  string            `json:"in_reply_to"` // In-Reply-To 头
	References string            `json:"references"`  // References 头
	Headers    map[string]string `json:"headers"`     // 其他邮件头（可选）

	// 附件信息
	Attachments []NormalizedAttachment `json:"attachments"` // 附件列表
}

// NormalizedAttachment 统一的附件格式
type NormalizedAttachment struct {
	// Filename 附件文件名
	Filename string `json:"filename"`

	// ContentType MIME 类型
	ContentType string `json:"content_type"`

	// Size 附件大小（字节）
	Size int64 `json:"size"`

	// Content 附件内容（Base64 解码后的原始字节）
	// 注意：大附件可能为空，需要通过 URL 下载
	Content []byte `json:"content,omitempty"`

	// ContentID 内嵌图片的 Content-ID（用于 HTML 邮件中的内嵌图片）
	ContentID string `json:"content_id,omitempty"`

	// URL 附件下载地址（某些服务商提供）
	URL string `json:"url,omitempty"`
}

// HasAttachments 检查是否有附件
func (e *NormalizedEmail) HasAttachments() bool {
	return len(e.Attachments) > 0
}

// AttachmentsCount 返回附件数量
func (e *NormalizedEmail) AttachmentsCount() int {
	return len(e.Attachments)
}

// GetToAddresses 获取所有收件人地址
// 如果 ToAddresses 为空，则返回包含 To 的切片
func (e *NormalizedEmail) GetToAddresses() []string {
	if len(e.ToAddresses) > 0 {
		return e.ToAddresses
	}
	if e.To != "" {
		return []string{e.To}
	}
	return nil
}

// WebhookResult Webhook 处理结果
type WebhookResult struct {
	// Success 是否处理成功
	Success bool `json:"success"`

	// Message 处理结果消息
	Message string `json:"message"`

	// EmailID 存储后的邮件 ID（成功时返回）
	EmailID int64 `json:"email_id,omitempty"`

	// Duplicate 是否为重复邮件
	Duplicate bool `json:"duplicate,omitempty"`

	// AccountUID 关联的账户 UID
	AccountUID string `json:"account_uid,omitempty"`
}

// WebhookContext Webhook 处理上下文
// 用于在处理链中传递信息
type WebhookContext struct {
	// Context 标准上下文
	Context context.Context

	// ProviderType 服务商类型
	ProviderType string

	// AccountUID 目标账户 UID（根据收件人地址查找）
	AccountUID string

	// Secret Webhook 密钥
	Secret string

	// RequestID 请求追踪 ID
	RequestID string
}
