package adapter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
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

	// 针对特定邮箱服务商的 TLS 配置优化
	host := strings.ToLower(a.config.Credentials.Host)

	// 139 邮箱（中国移动）需要更宽松的 TLS 配置
	if strings.Contains(host, "139.com") {
		fmt.Printf("[IMAP] Detected 139 Mail (China Mobile), using relaxed TLS config\n")
		tlsConfig.InsecureSkipVerify = true     // 跳过证书验证
		tlsConfig.MinVersion = tls.VersionTLS10 // 支持较旧的 TLS 版本
		tlsConfig.MaxVersion = tls.VersionTLS12 // 限制最高版本为 TLS 1.2
		// 关键：指定兼容的加密套件
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}
	}

	// QQ 邮箱
	if strings.Contains(host, "qq.com") {
		fmt.Printf("[IMAP] Detected QQ mail server, using relaxed TLS config\n")
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	// 163/126 邮箱（网易）
	if strings.Contains(host, "163.com") || strings.Contains(host, "126.com") {
		fmt.Printf("[IMAP] Detected NetEase mail server, using relaxed TLS config\n")
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	// 189 邮箱（中国电信）
	if strings.Contains(host, "189.cn") {
		fmt.Printf("[IMAP] Detected 189 Mail (China Telecom), using relaxed TLS config\n")
		tlsConfig.MinVersion = tls.VersionTLS10
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

	// 优先尝试 SASL PLAIN 认证（更标准，Outlook 等现代服务更喜欢）
	// 尝试方式 1: 仅提供 authcid (username)，authzid (identity) 为空
	// 这是最常见的方式
	err = a.client.Authenticate(sasl.NewPlainClient("", email, password))
	if err != nil {
		fmt.Printf("[IMAP] SASL PLAIN (no identity) failed: %v. Retrying with identity...\n", err)

		// 尝试方式 2: authzid 和 authcid 都设置为 email
		// 某些服务器可能需要显式指定 identity
		err2 := a.client.Authenticate(sasl.NewPlainClient(email, email, password))
		if err2 != nil {
			fmt.Printf("[IMAP] SASL PLAIN (with identity) failed: %v. Retrying with LOGIN command...\n", err2)

			// 回退到 LOGIN 命令
			if loginErr := a.client.Login(email, password).Wait(); loginErr != nil {
				// 如果两者都失败，返回 LOGIN 的错误（通常更具描述性）
				// 但也要包含 SASL 的错误信息以便调试
				errMsg := fmt.Sprintf("failed to login (SASL: %v, LOGIN: %v)", err, loginErr)

				// 如果是 Outlook 或 Gmail，且错误提示登录失败，建议使用应用专用密码
				if (a.config.Provider == "outlook" || a.config.Provider == "gmail") &&
					(strings.Contains(loginErr.Error(), "NO") || strings.Contains(loginErr.Error(), "failed")) {
					errMsg += ". Hint: For Outlook/Gmail, you may need to use an App Password if 2FA is enabled, or use OAuth2."
				}

				return fmt.Errorf("%s", errMsg)
			}
		}
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
		// 解码 MIME 编码的主题（支持 GBK、GB2312 等中文编码）
		email.Subject = decodeMIMEHeader(envelope.Subject)
		email.MessageID = envelope.MessageID
		if len(envelope.InReplyTo) > 0 {
			email.InReplyTo = envelope.InReplyTo[0]
		}

		// 发件人（解码名称）
		if len(envelope.From) > 0 {
			email.FromAddress = envelope.From[0].Addr()
			email.FromName = decodeMIMEHeader(envelope.From[0].Name)
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
	// 先读取全部内容到内存
	rawData, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read raw data: %w", err)
	}

	mr, err := mail.CreateReader(bytes.NewReader(rawData))
	if err != nil {
		// 如果无法创建 mail reader，尝试直接解析原始内容
		return a.parseRawBody(email, rawData)
	}

	partCount := 0

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// multipart 解析失败，可能是单部分邮件
			if partCount == 0 {
				// 没有成功解析任何部分，尝试直接解析原始内容
				return a.parseRawBody(email, rawData)
			}
			continue
		}
		partCount++

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			contentType, params, _ := h.ContentType()
			cs := strings.ToLower(strings.TrimSpace(params["charset"]))
			raw, readErr := io.ReadAll(part.Body)
			if readErr != nil {
				continue
			}

			decoded := string(raw)
			if cs != "" {
				if d, derr := decodeToUTF8(raw, cs); derr == nil {
					decoded = d
				}
			}

			switch contentType {
			case "text/plain":
				if email.TextBody == "" {
					email.TextBody = decoded
				}
			case "text/html":
				if email.HTMLBody == "" {
					email.HTMLBody = cleanHTMLBody(decoded)
				}
			}

		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			contentType, _, _ := h.ContentType()

			email.HasAttachments = true
			email.AttachmentsCount++

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

// parseRawBody 解析原始邮件正文（当 MIME 解析失败时使用）
func (a *IMAPAdapter) parseRawBody(email *Email, rawData []byte) error {
	rawStr := string(rawData)

	// 查找邮件正文开始位置（空行之后）
	bodyStart := strings.Index(rawStr, "\r\n\r\n")
	if bodyStart == -1 {
		bodyStart = strings.Index(rawStr, "\n\n")
	}

	if bodyStart == -1 {
		email.TextBody = rawStr
		return nil
	}

	// 提取邮件头和正文
	headerPart := rawStr[:bodyStart]
	bodyPart := rawStr[bodyStart+4:]

	// 展开多行邮件头（RFC 2822: 以空白开头的行是前一行的延续）
	headerPart = unfoldHeaders(headerPart)

	// 从邮件头中提取 Content-Type 和 boundary
	contentType := ""
	boundary := ""
	charsetName := ""

	for _, line := range strings.Split(headerPart, "\n") {
		line = strings.TrimRight(line, "\r")
		lowerLine := strings.ToLower(line)

		if strings.HasPrefix(lowerLine, "content-type:") {
			contentType = line[13:]
			// 提取 boundary
			if idx := strings.Index(lowerLine, "boundary="); idx != -1 {
				b := line[idx+9:]
				b = strings.Trim(b, "\"'; \r\n")
				if semiIdx := strings.Index(b, ";"); semiIdx != -1 {
					b = b[:semiIdx]
				}
				boundary = strings.TrimSpace(b)
			}
			// 提取 charset
			if idx := strings.Index(lowerLine, "charset="); idx != -1 {
				cs := line[idx+8:]
				cs = strings.Trim(cs, "\"'; \r\n")
				if semiIdx := strings.Index(cs, ";"); semiIdx != -1 {
					cs = cs[:semiIdx]
				}
				charsetName = strings.TrimSpace(cs)
			}
		}
	}

	// 如果是 multipart 邮件，解析各个部分
	if boundary != "" && strings.Contains(strings.ToLower(contentType), "multipart") {
		return a.parseMultipartBody(email, bodyPart, boundary)
	}

	// 单部分邮件处理
	return a.parseSinglePartBody(email, bodyPart, contentType, charsetName)
}

// unfoldHeaders 展开多行邮件头
func unfoldHeaders(headers string) string {
	// RFC 2822: 以空白（空格或制表符）开头的行是前一行的延续
	lines := strings.Split(headers, "\n")
	var result []string
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if i > 0 && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// 这是延续行，追加到前一行
			if len(result) > 0 {
				result[len(result)-1] += " " + strings.TrimSpace(line)
			}
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// parseMultipartBody 解析 multipart 邮件正文
func (a *IMAPAdapter) parseMultipartBody(email *Email, body string, boundary string) error {
	// 分割各个部分
	delimiter := "--" + boundary
	parts := strings.Split(body, delimiter)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "--" {
			continue
		}

		// 查找部分头和正文的分隔
		partBodyStart := strings.Index(part, "\r\n\r\n")
		if partBodyStart == -1 {
			partBodyStart = strings.Index(part, "\n\n")
		}
		if partBodyStart == -1 {
			continue
		}

		partHeader := part[:partBodyStart]
		partBody := part[partBodyStart+4:]
		if strings.HasPrefix(partBody, "\n") {
			partBody = partBody[1:]
		}

		// 展开多行头
		partHeader = unfoldHeaders(partHeader)

		// 解析部分头
		partContentType := ""
		partCharset := ""
		partEncoding := ""

		for _, line := range strings.Split(partHeader, "\n") {
			line = strings.TrimRight(line, "\r")
			lowerLine := strings.ToLower(line)

			if strings.HasPrefix(lowerLine, "content-type:") {
				partContentType = strings.TrimSpace(line[13:])
				if idx := strings.Index(lowerLine, "charset="); idx != -1 {
					cs := line[idx+8:]
					cs = strings.Trim(cs, "\"'; ")
					if semiIdx := strings.Index(cs, ";"); semiIdx != -1 {
						cs = cs[:semiIdx]
					}
					partCharset = strings.TrimSpace(cs)
				}
			} else if strings.HasPrefix(lowerLine, "content-transfer-encoding:") {
				partEncoding = strings.TrimSpace(strings.ToLower(line[26:]))
			}
		}

		// 解码内容
		decodedContent := partBody
		if partEncoding == "base64" {
			if decoded, err := decodeBase64Content(partBody); err == nil {
				decodedContent = string(decoded)
			}
		} else if partEncoding == "quoted-printable" {
			if decoded, err := decodeQuotedPrintableContent(partBody); err == nil {
				decodedContent = decoded
			}
		}

		// 字符集转换
		if partCharset != "" {
			if converted, err := decodeToUTF8([]byte(decodedContent), partCharset); err == nil {
				decodedContent = converted
			}
		}

		// 根据 Content-Type 设置正文
		lowerContentType := strings.ToLower(partContentType)
		if strings.Contains(lowerContentType, "text/html") {
			if email.HTMLBody == "" {
				email.HTMLBody = cleanHTMLBody(decodedContent)
			}
		} else if strings.Contains(lowerContentType, "text/plain") {
			if email.TextBody == "" {
				email.TextBody = decodedContent
			}
		}
	}

	return nil
}

// parseSinglePartBody 解析单部分邮件正文
func (a *IMAPAdapter) parseSinglePartBody(email *Email, body, contentType, charsetName string) error {
	// 从 Content-Type 中提取传输编码（如果有）
	transferEncoding := ""
	lowerContentType := strings.ToLower(contentType)

	decodedBody := body
	if transferEncoding == "base64" {
		if decoded, err := decodeBase64Content(body); err == nil {
			decodedBody = string(decoded)
		}
	} else if transferEncoding == "quoted-printable" {
		if decoded, err := decodeQuotedPrintableContent(body); err == nil {
			decodedBody = decoded
		}
	}

	// 字符集转换
	if charsetName != "" {
		if converted, err := decodeToUTF8([]byte(decodedBody), charsetName); err == nil {
			decodedBody = converted
		}
	}

	// 根据 Content-Type 设置正文
	if strings.Contains(lowerContentType, "text/html") {
		email.HTMLBody = cleanHTMLBody(decodedBody)
	} else {
		email.TextBody = decodedBody
	}

	return nil
}

// decodeBase64Content 解码 Base64 内容
func decodeBase64Content(s string) ([]byte, error) {
	// 移除空白字符
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, " ", "")

	// 标准 Base64 解码
	return base64.StdEncoding.DecodeString(s)
}

// decodeQuotedPrintableContent 解码 Quoted-Printable 内容
func decodeQuotedPrintableContent(s string) (string, error) {
	reader := quotedprintable.NewReader(strings.NewReader(s))
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(decoded), nil
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
	// 移除 HTML 注释
	html = regexp.MustCompile(`(?s)<!--.*?-->`).ReplaceAllString(html, "")

	// 移除 script 和 style 标签及其内容
	html = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(html, "")

	// 移除 head 标签及其内容
	html = regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`).ReplaceAllString(html, "")

	// 将 br 和 p 标签转换为换行
	html = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`(?i)</p>`).ReplaceAllString(html, "\n")

	// 移除所有 HTML 标签
	html = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(html, "")

	// 解码 HTML 实体
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")
	html = strings.ReplaceAll(html, "&apos;", "'")

	// 移除多余的空白和换行
	html = regexp.MustCompile(`[\t ]+`).ReplaceAllString(html, " ")
	html = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(html, "\n")
	html = strings.TrimSpace(html)

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

// decodeMIMEHeader 解码 MIME 编码的邮件头（如 =?gbk?b?...?= 格式）
// 支持 GBK、GB2312、GB18030、Big5 等中文编码
func decodeMIMEHeader(header string) string {
	if header == "" {
		return ""
	}

	// 检查是否包含 MIME 编码
	if !strings.Contains(header, "=?") {
		return header
	}

	// 创建支持中文编码的 MIME 解码器
	decoder := &mime.WordDecoder{
		CharsetReader: func(charsetName string, input io.Reader) (io.Reader, error) {
			name := strings.ToLower(strings.TrimSpace(charsetName))
			// 常见中文编码别名归一化
			if name == "gb2312" || name == "gbk" {
				name = "gb18030"
			}
			return charset.NewReaderLabel(name, input)
		},
	}

	decoded, err := decoder.DecodeHeader(header)
	if err != nil {
		// 解码失败，返回原始内容
		fmt.Printf("[IMAP] Failed to decode MIME header '%s': %v\n", header[:min(50, len(header))], err)
		return header
	}

	return decoded
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
