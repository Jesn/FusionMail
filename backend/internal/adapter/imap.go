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

	// 发送 IMAP ID 信息（某些服务器如 163 需要这个来识别客户端）
	clientID := &imap.IDData{
		Name:       "FusionMail",
		Version:    "1.0.0",
		Vendor:     "FusionMail",
		SupportURL: "https://fusionmail.com",
	}

	fmt.Printf("[IMAP] Sending ID command with client info...\n")
	_, err := a.client.ID(clientID).Wait()
	if err != nil {
		// ID 命令失败不应该阻止登录，只记录警告
		fmt.Printf("[IMAP] Warning: ID command failed: %v\n", err)
	} else {
		fmt.Printf("[IMAP] ID command sent successfully\n")
	}

	// 登录
	fmt.Printf("[IMAP] Logging in as %s...\n", email)
	if err := a.client.Login(email, password).Wait(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	fmt.Printf("[IMAP] Login successful\n")

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
	fmt.Printf("[IMAP] FetchEmails called, since=%v, limit=%d\n", since, limit)

	if a.client == nil {
		fmt.Printf("[IMAP] Error: client is nil\n")
		return nil, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	fmt.Printf("[IMAP] Selecting INBOX...\n")
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		fmt.Printf("[IMAP] Error selecting INBOX: %v\n", err)
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	fmt.Printf("[IMAP] Mailbox INBOX: %d messages, %d recent\n",
		mailbox.NumMessages, mailbox.NumRecent)

	if mailbox.NumMessages == 0 {
		fmt.Printf("[IMAP] No messages in INBOX\n")
		return []*Email{}, nil
	}

	// 不使用搜索，直接使用序列号范围获取邮件
	// 计算要获取的邮件范围
	start := uint32(1)
	end := mailbox.NumMessages

	if limit > 0 && int(mailbox.NumMessages) > limit {
		// 只获取最新的 limit 封邮件
		start = mailbox.NumMessages - uint32(limit) + 1
		fmt.Printf("[IMAP] Limiting to last %d emails (from %d to %d)\n", limit, start, end)
	} else {
		fmt.Printf("[IMAP] Fetching all %d emails\n", mailbox.NumMessages)
	}

	// 创建序列号集合
	seqSet := imap.SeqSet{}
	seqSet.AddRange(start, end)
	fmt.Printf("[IMAP] Created SeqSet for range %d:%d\n", start, end)

	// 获取邮件信息
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		BodySection:  []*imap.FetchItemBodySection{{}},
		UID:          true,
		InternalDate: true,
		RFC822Size:   true,
	}

	emails := make([]*Email, 0)
	fmt.Printf("[IMAP] Starting to fetch messages...\n")
	fetchCmd := a.client.Fetch(seqSet, fetchOptions)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		// 使用 Collect() 获取完整的消息数据
		buf, err := msg.Collect()
		if err != nil {
			fmt.Printf("[IMAP] Failed to collect message: %v\n", err)
			continue
		}

		email, err := a.parseMessageBuffer(buf)
		if err != nil {
			fmt.Printf("[IMAP] Failed to parse message: %v\n", err)
			continue
		}

		emails = append(emails, email)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	fmt.Printf("[IMAP] Successfully fetched %d emails\n", len(emails))
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

	buf, err := msg.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect message: %w", err)
	}

	email, err := a.parseMessageBuffer(buf)
	if err != nil {
		return nil, err
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to fetch email: %w", err)
	}

	return email, nil
}

// parseMessageBuffer 解析 IMAP 消息缓冲区
func (a *IMAPAdapter) parseMessageBuffer(buf *imapclient.FetchMessageBuffer) (*Email, error) {
	email := &Email{
		ProviderID: fmt.Sprintf("%d", buf.UID),
	}

	// 解析信封信息
	if buf.Envelope != nil {
		envelope := buf.Envelope
		email.Subject = envelope.Subject
		email.MessageID = envelope.MessageID
		if len(envelope.InReplyTo) > 0 {
			email.InReplyTo = envelope.InReplyTo[0]
		}

		// 发件人
		if len(envelope.From) > 0 {
			email.FromAddress = envelope.From[0].Addr()
			email.FromName = envelope.From[0].Name
		}

		// 收件人
		for _, addr := range envelope.To {
			email.ToAddresses = append(email.ToAddresses, addr.Addr())
		}

		// 抄送
		for _, addr := range envelope.Cc {
			email.CcAddresses = append(email.CcAddresses, addr.Addr())
		}

		// 密送
		for _, addr := range envelope.Bcc {
			email.BccAddresses = append(email.BccAddresses, addr.Addr())
		}

		// 回复地址
		if len(envelope.ReplyTo) > 0 {
			email.ReplyTo = envelope.ReplyTo[0].Addr()
		}

		// 发送时间
		if !envelope.Date.IsZero() {
			email.SentAt = envelope.Date
		}
	}

	// 接收时间
	if !buf.InternalDate.IsZero() {
		email.ReceivedAt = buf.InternalDate
	}

	// 邮件大小
	email.SizeBytes = buf.RFC822Size

	// 解析邮件正文
	for _, section := range buf.BodySection {
		// 简单处理：将正文内容转换为字符串
		bodyStr := string(section.Bytes)
		if email.TextBody == "" {
			email.TextBody = bodyStr
		}
	}

	// 生成摘要
	if email.Snippet == "" {
		email.Snippet = generateSnippet(email.TextBody, email.Subject)
	}

	// 设置默认值
	if email.Subject == "" {
		email.Subject = "No Subject"
	}
	if email.FromAddress == "" {
		email.FromAddress = "unknown@example.com"
	}
	if email.SentAt.IsZero() {
		email.SentAt = time.Now()
	}
	if email.ReceivedAt.IsZero() {
		email.ReceivedAt = time.Now()
	}

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
