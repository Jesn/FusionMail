package adapter

import (
	"context"
	"time"
)

// MailProvider 邮箱服务提供商接口
// 定义了所有邮箱协议适配器必须实现的方法
type MailProvider interface {
	// Connect 连接到邮箱服务器
	Connect(ctx context.Context) error

	// Disconnect 断开连接
	Disconnect() error

	// FetchEmails 拉取邮件列表
	// since: 从指定时间开始拉取（增量同步）
	// limit: 最大拉取数量（0 表示不限制）
	FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error)

	// FetchEmailDetail 获取邮件详情
	// providerID: 邮箱服务商的邮件 ID
	FetchEmailDetail(ctx context.Context, providerID string) (*Email, error)

	// GetProviderType 获取提供商类型
	GetProviderType() string

	// GetProtocol 获取协议类型
	GetProtocol() string

	// TestConnection 测试连接
	TestConnection(ctx context.Context) error
}

// Email 邮件数据结构
type Email struct {
	// 基本信息
	ProviderID   string   // 邮箱服务商原生 ID
	MessageID    string   // 邮件 Message-ID
	Subject      string   // 主题
	FromAddress  string   // 发件人地址
	FromName     string   // 发件人名称
	ToAddresses  []string // 收件人地址列表
	CcAddresses  []string // 抄送地址列表
	BccAddresses []string // 密送地址列表
	ReplyTo      string   // 回复地址

	// 邮件内容
	TextBody string // 纯文本正文
	HTMLBody string // HTML 正文
	Snippet  string // 摘要

	// 源邮箱状态
	SourceIsRead *bool    // 源邮箱已读状态
	SourceLabels []string // 源邮箱标签
	SourceFolder string   // 源邮箱文件夹

	// 附件信息
	HasAttachments   bool         // 是否有附件
	AttachmentsCount int          // 附件数量
	Attachments      []Attachment // 附件列表

	// 时间信息
	SentAt     time.Time // 发送时间
	ReceivedAt time.Time // 接收时间

	// 元数据
	SizeBytes  int64  // 邮件大小（字节）
	ThreadID   string // 会话 ID
	InReplyTo  string // 回复的邮件 ID
	References string // 引用的邮件 ID 列表
}

// Attachment 附件数据结构
type Attachment struct {
	Filename    string // 文件名
	ContentType string // 内容类型
	SizeBytes   int64  // 大小（字节）
	Content     []byte // 内容（可选，用于下载）
	IsInline    bool   // 是否内联附件
	ContentID   string // Content-ID（用于内联）
}

// Credentials 认证凭证
type Credentials struct {
	// 通用字段
	Email    string // 邮箱地址
	AuthType string // 认证类型：oauth2/password/app_password

	// 密码认证
	Password string // 密码或应用专用密码

	// OAuth2 认证
	AccessToken  string    // 访问令牌
	RefreshToken string    // 刷新令牌
	TokenExpiry  time.Time // 令牌过期时间
	ClientID     string    // 客户端 ID
	ClientSecret string    // 客户端密钥

	// IMAP/POP3 配置
	Host     string // 服务器地址
	Port     int    // 端口
	TLS      bool   // 是否使用 TLS
	StartTLS bool   // 是否使用 STARTTLS
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Enabled  bool   // 是否启用代理
	Type     string // 代理类型：http/socks5
	Host     string // 代理服务器地址
	Port     int    // 代理端口
	Username string // 代理用户名（可选）
	Password string // 代理密码（可选）
}

// Config 适配器配置
type Config struct {
	Provider    string        // 提供商类型：gmail/outlook/imap/pop3
	Protocol    string        // 协议类型：gmail_api/graph/imap/pop3
	Credentials *Credentials  // 认证凭证
	Proxy       *ProxyConfig  // 代理配置（可选）
	Timeout     time.Duration // 超时时间
}
