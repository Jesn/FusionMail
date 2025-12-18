package adapter

import (
	"context"
	"time"
)

// MailSender 邮件发送接口
// 定义了所有邮件发送器必须实现的方法
// Requirements: 1.1, 2.1, 2.2, 2.3
type MailSender interface {
	// Send 发送邮件
	// 返回: 消息ID（服务商返回的唯一标识）, 错误
	Send(ctx context.Context, email *OutgoingEmail) (*SendResult, error)

	// TestConnection 测试发送连接
	// 用于验证 SMTP 配置是否正确
	TestConnection(ctx context.Context) error

	// GetSenderType 获取发送器类型
	// 返回: gmail_api/graph_api/smtp
	GetSenderType() string
}

// OutgoingEmail 待发送邮件结构
// Requirements: 1.1, 4.4, 5.1, 5.2, 5.3, 5.4, 5.5
type OutgoingEmail struct {
	// 发件人信息
	From     string // 发件人地址
	FromName string // 发件人名称（可选）

	// 收件人信息
	To  []string // 收件人列表
	Cc  []string // 抄送列表
	Bcc []string // 密送列表

	// 邮件内容
	Subject  string // 主题
	TextBody string // 纯文本正文
	HTMLBody string // HTML 正文

	// 附件列表
	Attachments []OutgoingAttachment

	// 回复/转发相关（用于设置邮件头）
	InReplyTo  string // 回复的邮件 Message-ID（用于 In-Reply-To 头）
	References string // 引用的邮件 ID 列表（用于 References 头）
	ReplyTo    string // 回复地址（Reply-To 头）

	// 元数据
	AccountUID string // 发送账户 UID
}

// OutgoingAttachment 待发送附件结构
// Requirements: 4.1, 4.2, 4.3, 4.4
type OutgoingAttachment struct {
	Filename    string // 文件名
	ContentType string // 内容类型（MIME 类型）
	Content     []byte // 附件内容
	IsInline    bool   // 是否内联附件
	ContentID   string // Content-ID（用于内联附件）
}

// SendResult 发送结果
// Requirements: 1.4, 1.5
type SendResult struct {
	// 消息标识
	ProviderMsgID string `json:"provider_msg_id,omitempty"` // 服务商返回的消息 ID
	MessageID     string `json:"message_id,omitempty"`      // 邮件 Message-ID（RFC 2822）

	// 发送状态
	Success bool   `json:"success"`         // 是否发送成功
	Error   string `json:"error,omitempty"` // 错误信息（如果失败）

	// 发送器信息
	SenderType string `json:"sender_type,omitempty"` // 使用的发送器类型（gmail_api/graph_api/smtp）

	// 已发送邮件记录 ID
	SentEmailID int64 `json:"sent_email_id,omitempty"` // 已发送邮件记录 ID
}

// SMTPConfig SMTP 发送配置
// Requirements: 3.1, 3.4, 3.5
type SMTPConfig struct {
	Host       string // SMTP 服务器地址
	Port       int    // SMTP 端口
	Encryption string // 加密方式：none/tls/starttls
	Username   string // 用户名（通常是邮箱地址）
	Password   string // 密码或应用专用密码（加密存储）
}

// SenderConfig 发送器配置
// 用于创建发送器实例
type SenderConfig struct {
	// 账户信息
	AccountUID string // 账户 UID
	Email      string // 邮箱地址
	Provider   string // 提供商类型：gmail/outlook/imap/pop3/generic

	// 认证信息
	AuthType     string    // 认证类型：oauth2/password/app_password
	AccessToken  string    // OAuth2 访问令牌
	RefreshToken string    // OAuth2 刷新令牌
	TokenExpiry  time.Time // OAuth2 令牌过期时间（用于自动刷新）
	ClientID     string    // OAuth2 客户端 ID
	ClientSecret string    // OAuth2 客户端密钥

	// SMTP 配置（用于 SMTP 发送）
	SMTP *SMTPConfig

	// 代理配置（可选）
	Proxy *ProxyConfig
}

// 发送器类型常量
const (
	SenderTypeGmailAPI = "gmail_api" // Gmail API 发送
	SenderTypeGraphAPI = "graph_api" // Microsoft Graph API 发送
	SenderTypeSMTP     = "smtp"      // SMTP 协议发送
)

// 附件大小限制常量
// Requirements: 4.2, 4.3
const (
	MaxSingleAttachmentSize = 25 * 1024 * 1024 // 单个附件最大 25MB
	MaxTotalAttachmentSize  = 50 * 1024 * 1024 // 附件总大小最大 50MB
)
