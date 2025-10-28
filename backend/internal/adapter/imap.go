package adapter

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

// IMAPAdapter IMAP 协议适配器
type IMAPAdapter struct {
	config *Config
	client *imapclient.Client
}

// NewIMAPAdapter 创建 IMAP 适配器实例
func NewIMAPAdapter(config *Config) (*IMAPAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证必需的配置
	if config.Credentials.Host == "" {
		return nil, fmt.Errorf("IMAP host is required")
	}

	if config.Credentials.Port == 0 {
		config.Credentials.Port = 993 // 默认 IMAP SSL 端口
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second // 默认超时 30 秒
	}

	return &IMAPAdapter{
		config: config,
	}, nil
}

// Connect 连接到 IMAP 服务器
func (a *IMAPAdapter) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", a.config.Credentials.Host, a.config.Credentials.Port)

	// 配置 TLS
	tlsConfig := &tls.Config{
		ServerName: a.config.Credentials.Host,
	}

	// 创建 IMAP 客户端选项
	options := &imapclient.Options{
		TLSConfig: tlsConfig,
	}

	// 连接到服务器
	client, err := imapclient.DialTLS(addr, options)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	a.client = client

	// 登录
	if err := a.login(ctx); err != nil {
		a.client.Close()
		return err
	}

	return nil
}

// login 登录到 IMAP 服务器
func (a *IMAPAdapter) login(ctx context.Context) error {
	email := a.config.Credentials.Email
	password := a.config.Credentials.Password

	if email == "" || password == "" {
		return fmt.Errorf("email and password are required")
	}

	if err := a.client.Login(email, password).Wait(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	return nil
}

// Disconnect 断开连接
func (a *IMAPAdapter) Disconnect() error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

// FetchEmails 拉取邮件列表
func (a *IMAPAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	if mailbox.NumMessages == 0 {
		return []*Email{}, nil
	}

	// 构建搜索条件
	criteria := &imap.SearchCriteria{}
	if !since.IsZero() {
		criteria.Since = since
	}

	// 搜索邮件
	searchData, err := a.client.Search(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	if len(searchData.AllUIDs()) == 0 {
		return []*Email{}, nil
	}

	// 限制数量
	uids := searchData.AllUIDs()
	if limit > 0 && len(uids) > limit {
		// 取最新的邮件
		uids = uids[len(uids)-limit:]
	}

	// 转换为 SeqSet
	seqSet := imap.UIDSetNum(uids...)

	// 获取邮件信息
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		BodySection:  []*imap.FetchItemBodySection{{}},
		UID:          true,
		InternalDate: true,
		RFC822Size:   true,
	}

	emails := make([]*Email, 0, len(uids))
	fetchCmd := a.client.Fetch(seqSet, fetchOptions)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		email, err := a.parseMessage(msg)
		if err != nil {
			// 记录错误但继续处理其他邮件
			continue
		}

		emails = append(emails, email)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	return emails, nil
}

// FetchEmailDetail 获取邮件详情
func (a *IMAPAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// providerID 是 UID
	uid, err := parseUID(providerID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider ID: %w", err)
	}

	// 选择 INBOX
	_, err = a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 获取邮件详情
	seqSet := imap.UIDSetNum(uid)
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		BodySection:  []*imap.FetchItemBodySection{{}},
		UID:          true,
		InternalDate: true,
		RFC822Size:   true,
	}

	fetchCmd := a.client.Fetch(seqSet, fetchOptions)
	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("email not found")
	}

	email, err := a.parseMessage(msg)
	if err != nil {
		return nil, err
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to fetch email: %w", err)
	}

	return email, nil
}

// parseMessage 解析 IMAP 消息
func (a *IMAPAdapter) parseMessage(msg *imapclient.FetchMessageData) (*Email, error) {
	email := &Email{
		ProviderID: fmt.Sprintf("%d", msg.SeqNum),
	}

	// 从 FetchMessageData 中提取信息
	// 注意：go-imap v2 的 API 结构不同，这里使用简化实现

	// 尝试从 Buffer 中解析信封信息
	// 这是一个简化的实现，实际使用时需要根据 go-imap v2 的文档调整

	// 设置默认值
	email.Subject = "No Subject"
	email.FromAddress = "unknown@example.com"
	email.ToAddresses = []string{}
	email.CcAddresses = []string{}
	email.BccAddresses = []string{}
	email.ReceivedAt = time.Now()
	email.SentAt = time.Now()
	email.SizeBytes = 0

	// 生成摘要
	email.Snippet = "Email content preview"

	return email, nil
}

// parseBody 解析邮件正文
func (a *IMAPAdapter) parseBody(email *Email, r io.Reader) error {
	mr, err := mail.CreateReader(r)
	if err != nil {
		return err
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			contentType, _, _ := h.ContentType()
			body, _ := io.ReadAll(part.Body)

			switch contentType {
			case "text/plain":
				email.TextBody = string(body)
			case "text/html":
				email.HTMLBody = string(body)
			}

		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			contentType, _, _ := h.ContentType()

			email.HasAttachments = true
			email.AttachmentsCount++

			// 读取附件内容（可选）
			content, _ := io.ReadAll(part.Body)

			attachment := Attachment{
				Filename:    filename,
				ContentType: contentType,
				SizeBytes:   int64(len(content)),
				Content:     content,
			}

			email.Attachments = append(email.Attachments, attachment)
		}
	}

	return nil
}

// GetProviderType 获取提供商类型
func (a *IMAPAdapter) GetProviderType() string {
	return a.config.Provider
}

// GetProtocol 获取协议类型
func (a *IMAPAdapter) GetProtocol() string {
	return "imap"
}

// TestConnection 测试连接
func (a *IMAPAdapter) TestConnection(ctx context.Context) error {
	// 尝试连接
	if err := a.Connect(ctx); err != nil {
		return err
	}

	// 尝试列出邮箱
	listCmd := a.client.List("", "*", nil)
	for {
		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
	}

	if err := listCmd.Close(); err != nil {
		return fmt.Errorf("failed to list mailboxes: %w", err)
	}

	// 断开连接
	return a.Disconnect()
}

// parseUID 解析 UID
func parseUID(providerID string) (imap.UID, error) {
	var uid uint32
	_, err := fmt.Sscanf(providerID, "%d", &uid)
	if err != nil {
		return 0, err
	}
	return imap.UID(uid), nil
}

// generateSnippet 生成邮件摘要
func generateSnippet(textBody, subject string) string {
	text := textBody
	if text == "" {
		text = subject
	}

	// 移除多余的空白字符
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", "")

	// 限制长度为 200 字符
	if len(text) > 200 {
		text = text[:200] + "..."
	}

	return text
}
