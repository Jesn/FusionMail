package adapter

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

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
