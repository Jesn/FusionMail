package receiver

import (
	"strings"
	"time"

	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
)

// CloudflareAdapter Cloudflare Temp Email Webhook 适配器
// 处理 Cloudflare Temp Email 服务推送的邮件数据
type CloudflareAdapter struct {
	logger *logger.Logger
}

// 模块日志记录器
var cloudflareLog = logger.NewWithModule("CloudflareWebhook")

// NewCloudflareAdapter 创建 Cloudflare Temp Email 适配器实例
func NewCloudflareAdapter(log *logger.Logger) *CloudflareAdapter {
	if log == nil {
		log = cloudflareLog
	}
	return &CloudflareAdapter{
		logger: log,
	}
}

// GetProviderType 返回适配器对应的 provider 类型
func (a *CloudflareAdapter) GetProviderType() string {
	return "cloudflare_temp_email"
}

// GetSignatureHeader 返回签名 Header 名称
func (a *CloudflareAdapter) GetSignatureHeader() string {
	return "X-Webhook-Secret"
}

// ValidateRequest 验证请求的合法性
// 通过比对 X-Webhook-Secret header 与配置的 secret
func (a *CloudflareAdapter) ValidateRequest(c *gin.Context, secret string) error {
	// 如果没有配置 secret，跳过验证
	if secret == "" {
		a.logger.Debug("webhook secret 未配置，跳过验证")
		return nil
	}

	// 获取请求中的 secret
	headerSecret := c.GetHeader("X-Webhook-Secret")
	if headerSecret == "" {
		a.logger.Warn("缺少 X-Webhook-Secret header")
		return NewInvalidSecretError()
	}

	// 比对 secret（使用常量时间比较防止时序攻击）
	if !secureCompare(headerSecret, secret) {
		a.logger.Warn("webhook secret 验证失败: received=%s, expected=%s",
			MaskSecret(headerSecret), MaskSecret(secret))
		return NewInvalidSecretError()
	}

	return nil
}

// ParsePayload 解析 Cloudflare Temp Email 的 webhook payload
// 将原始格式转换为统一的 NormalizedEmail 格式
func (a *CloudflareAdapter) ParsePayload(c *gin.Context) (*NormalizedEmail, error) {
	var payload CloudflarePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		a.logger.Error("解析 Cloudflare webhook payload 失败: %v", err)
		return nil, NewInvalidPayloadError(err.Error())
	}

	// 验证必填字段
	if err := payload.Validate(); err != nil {
		return nil, err
	}

	// 转换为统一格式
	email := a.convertToNormalizedEmail(&payload)

	a.logger.Debug("解析 Cloudflare webhook payload 成功: provider_id=%s, from=%s, to=%s, subject=%s",
		email.ProviderID, email.FromAddress, email.To, TruncateString(email.Subject, 50))

	return email, nil
}

// convertToNormalizedEmail 将 Cloudflare payload 转换为统一格式
func (a *CloudflareAdapter) convertToNormalizedEmail(payload *CloudflarePayload) *NormalizedEmail {
	// 解析发件人信息
	fromName, fromAddress := ParseEmailAddress(payload.From)

	// 解析收件人信息
	toAddresses := ParseEmailAddressList(payload.To)
	if len(toAddresses) == 0 && payload.To != "" {
		toAddresses = []string{payload.To}
	}

	// 获取主收件人
	mainTo := payload.To
	if len(toAddresses) > 0 {
		mainTo = toAddresses[0]
	}

	// 解析时间
	var sentAt *time.Time
	if payload.Date != "" {
		sentAt = ParseRFC822Date(payload.Date)
	}
	// 如果没有发送时间，使用当前时间
	if sentAt == nil {
		now := time.Now()
		sentAt = &now
	}

	receivedAt := time.Now()

	// 构建统一格式
	email := &NormalizedEmail{
		ProviderID:  payload.ID,
		From:        payload.From,
		FromName:    fromName,
		FromAddress: fromAddress,
		To:          mainTo,
		ToAddresses: toAddresses,
		Subject:     SanitizeSubject(payload.Subject),
		TextBody:    payload.ParsedText,
		HtmlBody:    payload.ParsedHtml,
		RawContent:  payload.Raw,
		SentAt:      sentAt,
		ReceivedAt:  &receivedAt,
		MessageID:   payload.MessageID,
	}

	// 解析 CC 地址
	if payload.Cc != "" {
		email.CcAddresses = ParseEmailAddressList(payload.Cc)
	}

	// 解析 Reply-To
	if payload.ReplyTo != "" {
		_, email.ReplyTo = ParseEmailAddress(payload.ReplyTo)
	}

	// 解析附件信息
	if len(payload.Attachments) > 0 {
		email.Attachments = make([]NormalizedAttachment, 0, len(payload.Attachments))
		for _, att := range payload.Attachments {
			email.Attachments = append(email.Attachments, NormalizedAttachment{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Size:        att.Size,
				ContentID:   att.ContentID,
				URL:         att.URL,
			})
		}
	}

	return email
}

// CloudflarePayload Cloudflare Temp Email 的原始 webhook payload 格式
type CloudflarePayload struct {
	// ID 邮件唯一标识（用于去重）
	ID string `json:"id"`

	// URL 前端查看链接（可选）
	URL string `json:"url,omitempty"`

	// From 发件人，格式可能是 "Name <email>" 或 "email"
	From string `json:"from"`

	// To 收件人地址
	To string `json:"to"`

	// Cc 抄送地址（可选）
	Cc string `json:"cc,omitempty"`

	// ReplyTo 回复地址（可选）
	ReplyTo string `json:"replyTo,omitempty"`

	// Subject 邮件主题
	Subject string `json:"subject"`

	// Date 发送日期（RFC822 格式，可选）
	Date string `json:"date,omitempty"`

	// MessageID Message-ID 头（可选）
	MessageID string `json:"messageId,omitempty"`

	// Raw RFC822 原始内容（可选）
	Raw string `json:"raw,omitempty"`

	// ParsedText 解析后的纯文本正文
	ParsedText string `json:"parsedText,omitempty"`

	// ParsedHtml 解析后的 HTML 正文
	ParsedHtml string `json:"parsedHtml,omitempty"`

	// Attachments 附件列表（可选）
	Attachments []CloudflareAttachment `json:"attachments,omitempty"`
}

// CloudflareAttachment Cloudflare Temp Email 的附件格式
type CloudflareAttachment struct {
	// Filename 文件名
	Filename string `json:"filename"`

	// ContentType MIME 类型
	ContentType string `json:"contentType"`

	// Size 文件大小（字节）
	Size int64 `json:"size"`

	// ContentID 内嵌图片的 Content-ID（可选）
	ContentID string `json:"contentId,omitempty"`

	// URL 附件下载地址（可选）
	URL string `json:"url,omitempty"`
}

// Validate 验证 payload 的必填字段
func (p *CloudflarePayload) Validate() error {
	// ID 是必填的（用于去重）
	if strings.TrimSpace(p.ID) == "" {
		return NewMissingFieldError("id")
	}

	// To 是必填的（用于查找账户）
	if strings.TrimSpace(p.To) == "" {
		return NewMissingFieldError("to")
	}

	// 必须有内容（raw 或 parsedText/parsedHtml 至少有一个）
	if strings.TrimSpace(p.Raw) == "" &&
		strings.TrimSpace(p.ParsedText) == "" &&
		strings.TrimSpace(p.ParsedHtml) == "" {
		return NewMissingFieldError("raw or parsedText or parsedHtml")
	}

	return nil
}

// secureCompare 安全比较两个字符串
// 使用常量时间比较，防止时序攻击
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// 确保 CloudflareAdapter 实现了 WebhookAdapter 接口
var _ WebhookAdapter = (*CloudflareAdapter)(nil)
