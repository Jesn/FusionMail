package adapter

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html/charset"
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

	// 使用搜索来根据时间过滤邮件
	var seqSet imap.SeqSet

	if !since.IsZero() {
		// 暂时跳过时间过滤，直接获取所有邮件
		fmt.Printf("[IMAP] Time filtering requested but not implemented, fetching all emails\n")
		seqSet.AddRange(1, mailbox.NumMessages)
	} else {
		// 没有时间限制，获取所有邮件或最新的 limit 封
		start := uint32(1)
		end := mailbox.NumMessages

		if limit > 0 && int(mailbox.NumMessages) > limit {
			// 只获取最新的 limit 封邮件
			start = mailbox.NumMessages - uint32(limit) + 1
			fmt.Printf("[IMAP] Limiting to last %d emails (from %d to %d)\n", limit, start, end)
		} else {
			fmt.Printf("[IMAP] Fetching all %d emails\n", mailbox.NumMessages)
		}

		seqSet.AddRange(start, end)
	}

	fmt.Printf("[IMAP] Created SeqSet\n")

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
		// 使用 mail.CreateReader 正确解析 MIME 结构
		reader := bytes.NewReader(section.Bytes)
		if err := a.parseBody(email, reader); err != nil {
			// 如果解析失败，回退到简单处理
			fmt.Printf("[IMAP] Failed to parse body with mail.CreateReader: %v\n", err)
			bodyStr := string(section.Bytes)
			if email.TextBody == "" {
				email.TextBody = bodyStr
			}
		} else {
			fmt.Printf("[IMAP] Successfully parsed body: HTML=%d bytes, Text=%d bytes\n",
				len(email.HTMLBody), len(email.TextBody))
		}
	}

	// 生成摘要
	if email.Snippet == "" {
		if email.TextBody != "" {
			email.Snippet = generateSnippet(email.TextBody, email.Subject)
		} else if email.HTMLBody != "" {
			// 从 HTML 中提取纯文本作为摘要
			email.Snippet = generateSnippet(stripHTML(email.HTMLBody), email.Subject)
		}
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

// parseBody 解析邮件正文（支持传输编码与字符集转码）
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
			contentType, params, _ := h.ContentType()
			cs := strings.ToLower(strings.TrimSpace(params["charset"]))
			raw, _ := io.ReadAll(part.Body)

			decoded := string(raw)
			if d, derr := decodeToUTF8(raw, cs); derr == nil {
				decoded = d
			}

			switch contentType {
			case "text/plain":
				email.TextBody = decoded
			case "text/html":
				// 清理 HTML 内容，移除邮件服务器添加的包装标签
				email.HTMLBody = cleanHTMLBody(decoded)
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

// stripHTML 从 HTML 中提取纯文本
func stripHTML(html string) string {
	// 简单的 HTML 标签移除
	// 移除 script 和 style 标签及其内容
	html = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`).ReplaceAllString(html, "")

	// 移除所有 HTML 标签
	html = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(html, "")

	// 解码 HTML 实体
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&quot;", "\"")

	// 移除多余的空白
	html = strings.TrimSpace(html)
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")

	return html
}

// cleanHTMLBody 清理 HTML 正文，移除邮件服务器添加的包装标签
func cleanHTMLBody(html string) string {
	// 移除 <DATA><MSG> 包装标签（某些邮件服务器会添加）
	html = regexp.MustCompile(`(?i)<DATA><MSG>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?i)</MSG></DATA>`).ReplaceAllString(html, "")

	return strings.TrimSpace(html)
}

// MoveToTrash 将邮件移至垃圾箱
func (a *IMAPAdapter) MoveToTrash(ctx context.Context, providerID string) error {
	if a.client == nil {
		return fmt.Errorf("not connected to IMAP server")
	}

	// 解析 UID
	uid, err := parseUID(providerID)
	if err != nil {
		return fmt.Errorf("invalid provider ID: %w", err)
	}

	// 发现 Trash 文件夹
	trashMailbox, err := a.findTrashMailbox(ctx)
	if err != nil {
		return fmt.Errorf("trash mailbox not found: %w", err)
	}

	// 尝试使用 MOVE 命令（RFC 6851）
	if err := a.moveToTrash(ctx, uid, trashMailbox); err == nil {
		return nil
	}

	// 降级：COPY + STORE +FLAGS \Deleted
	return a.copyAndMarkDeleted(ctx, uid, trashMailbox)
}

// findTrashMailbox 发现 Trash 文件夹
func (a *IMAPAdapter) findTrashMailbox(ctx context.Context) (string, error) {
	// 获取所有邮箱列表
	listCmd := a.client.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return "", fmt.Errorf("failed to list mailboxes: %w", err)
	}

	// 其次查找常见的 Trash 文件夹名称
	trashNames := []string{"Trash", "Deleted Items", "[Gmail]/Trash", "[Gmail]/Deleted Mail", "Deleted", "Bin"}
	for _, name := range trashNames {
		for _, mbox := range mailboxes {
			if strings.EqualFold(mbox.Mailbox, name) {
				return mbox.Mailbox, nil
			}
		}
	}

	return "", fmt.Errorf("trash mailbox not found")
}

// moveToTrash 使用 MOVE 命令移动邮件到 Trash
func (a *IMAPAdapter) moveToTrash(ctx context.Context, uid imap.UID, trashMailbox string) error {
	// 选择 INBOX
	_, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 使用 MOVE 命令（UID 模式）
	seqSet := imap.UIDSetNum(uid)
	_, err = a.client.Move(seqSet, trashMailbox).Wait()
	if err != nil {
		return fmt.Errorf("failed to move message: %w", err)
	}

	return nil
}

// copyAndMarkDeleted COPY 降级方案（仅复制到 Trash，不标记删除）
func (a *IMAPAdapter) copyAndMarkDeleted(ctx context.Context, uid imap.UID, trashMailbox string) error {
	// 选择 INBOX
	_, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	// COPY 到 Trash
	seqSet := imap.UIDSetNum(uid)
	_, err = a.client.Copy(seqSet, trashMailbox).Wait()
	if err != nil {

		return fmt.Errorf("failed to copy message to trash: %w", err)
	}

	return nil
}

// decodeToUTF8 将按声明的 charset 的字节流转换为 UTF-8 字符串
func decodeToUTF8(b []byte, charsetName string) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	name := strings.ToLower(strings.TrimSpace(charsetName))
	if name == "" || name == "utf-8" || name == "utf8" || name == "us-ascii" {
		return string(b), nil
	}
	// 常见别名归一化
	if name == "gb2312" || name == "gbk" {
		name = "gb18030"
	}
	r, err := charset.NewReaderLabel(name, bytes.NewReader(b))
	if err != nil {
		// 回退：直接返回原文（避免阻塞）
		return string(b), err
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return string(b), err
	}
	return string(out), nil
}
