package webapi

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"fusionmail/internal/adapter"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// RFC822Parser RFC822 邮件解析器
// 用于解析 Cloudflare Temp Email 等服务返回的原始邮件内容
type RFC822Parser struct {
	// 配置选项
	MaxBodySize     int  // 最大正文大小（字节），默认 10MB
	ParseAttachment bool // 是否解析附件内容
}

// NewRFC822Parser 创建 RFC822 解析器
func NewRFC822Parser() *RFC822Parser {
	return &RFC822Parser{
		MaxBodySize:     10 * 1024 * 1024, // 10MB
		ParseAttachment: true,
	}
}

// Parse 解析 RFC822 格式的邮件
func (p *RFC822Parser) Parse(rawContent string) (*adapter.Email, error) {
	if rawContent == "" {
		return nil, WrapError(ErrCodeParseError, "邮件内容为空", nil)
	}

	// 解析邮件
	msg, err := mail.ReadMessage(strings.NewReader(rawContent))
	if err != nil {
		return nil, WrapError(ErrCodeParseError, "解析邮件头失败", err)
	}

	email := &adapter.Email{}

	// 解析邮件头
	if err := p.parseHeaders(msg.Header, email); err != nil {
		return nil, err
	}

	// 解析邮件正文
	if err := p.parseBody(msg, email); err != nil {
		return nil, err
	}

	return email, nil
}

// parseHeaders 解析邮件头
func (p *RFC822Parser) parseHeaders(header mail.Header, email *adapter.Email) error {
	// Message-ID
	email.MessageID = header.Get("Message-ID")
	if email.MessageID != "" {
		// 移除尖括号
		email.MessageID = strings.Trim(email.MessageID, "<>")
	}

	// Subject
	subject := header.Get("Subject")
	if subject != "" {
		decoded, err := decodeRFC2047(subject)
		if err == nil {
			email.Subject = decoded
		} else {
			email.Subject = subject
		}
	}

	// From
	from := header.Get("From")
	if from != "" {
		addr, err := mail.ParseAddress(from)
		if err == nil {
			email.FromAddress = addr.Address
			email.FromName = addr.Name
		} else {
			// 尝试直接使用
			email.FromAddress = from
		}
	}

	// To
	to := header.Get("To")
	if to != "" {
		addrs, err := mail.ParseAddressList(to)
		if err == nil {
			email.ToAddresses = make([]string, len(addrs))
			for i, addr := range addrs {
				email.ToAddresses[i] = addr.Address
			}
		} else {
			// 简单分割
			email.ToAddresses = splitAddresses(to)
		}
	}

	// Cc
	cc := header.Get("Cc")
	if cc != "" {
		addrs, err := mail.ParseAddressList(cc)
		if err == nil {
			email.CcAddresses = make([]string, len(addrs))
			for i, addr := range addrs {
				email.CcAddresses[i] = addr.Address
			}
		} else {
			email.CcAddresses = splitAddresses(cc)
		}
	}

	// Reply-To
	replyTo := header.Get("Reply-To")
	if replyTo != "" {
		addr, err := mail.ParseAddress(replyTo)
		if err == nil {
			email.ReplyTo = addr.Address
		} else {
			email.ReplyTo = replyTo
		}
	}

	// Date
	dateStr := header.Get("Date")
	if dateStr != "" {
		t, err := mail.ParseDate(dateStr)
		if err == nil {
			email.SentAt = t
			email.ReceivedAt = t // 默认使用发送时间
		}
	}

	// In-Reply-To
	email.InReplyTo = strings.Trim(header.Get("In-Reply-To"), "<>")

	// References
	email.References = header.Get("References")

	return nil
}

// parseBody 解析邮件正文
func (p *RFC822Parser) parseBody(msg *mail.Message, email *adapter.Email) error {
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// 如果解析失败，尝试作为纯文本处理
		body, err := io.ReadAll(io.LimitReader(msg.Body, int64(p.MaxBodySize)))
		if err != nil {
			return WrapError(ErrCodeParseError, "读取邮件正文失败", err)
		}
		email.TextBody = string(body)
		return nil
	}

	// 处理不同的内容类型
	if strings.HasPrefix(mediaType, "multipart/") {
		return p.parseMultipart(msg.Body, params["boundary"], email)
	}

	// 单一内容类型
	body, err := p.readBody(msg.Body, msg.Header)
	if err != nil {
		return err
	}

	if strings.HasPrefix(mediaType, "text/html") {
		email.HTMLBody = body
	} else {
		email.TextBody = body
	}

	return nil
}

// parseMultipart 解析 multipart 邮件
func (p *RFC822Parser) parseMultipart(body io.Reader, boundary string, email *adapter.Email) error {
	if boundary == "" {
		return WrapError(ErrCodeParseError, "multipart 边界为空", nil)
	}

	reader := multipart.NewReader(body, boundary)
	attachments := make([]adapter.Attachment, 0)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return WrapError(ErrCodeParseError, "解析 multipart 部分失败", err)
		}

		contentType := part.Header.Get("Content-Type")
		contentDisposition := part.Header.Get("Content-Disposition")

		mediaType, params, _ := mime.ParseMediaType(contentType)

		// 检查是否为附件
		if strings.HasPrefix(contentDisposition, "attachment") ||
			(contentDisposition != "" && strings.Contains(contentDisposition, "filename")) {
			att, err := p.parseAttachment(part, contentDisposition, mediaType)
			if err == nil && att != nil {
				attachments = append(attachments, *att)
			}
			continue
		}

		// 处理嵌套的 multipart
		if strings.HasPrefix(mediaType, "multipart/") {
			nestedBoundary := params["boundary"]
			if nestedBoundary != "" {
				if err := p.parseMultipart(part, nestedBoundary, email); err != nil {
					// 忽略嵌套解析错误，继续处理
					continue
				}
			}
			continue
		}

		// 读取内容
		content, err := p.readPartBody(part)
		if err != nil {
			continue
		}

		// 根据内容类型设置正文
		if strings.HasPrefix(mediaType, "text/html") {
			if email.HTMLBody == "" {
				email.HTMLBody = content
			}
		} else if strings.HasPrefix(mediaType, "text/plain") {
			if email.TextBody == "" {
				email.TextBody = content
			}
		}
	}

	// 设置附件信息
	if len(attachments) > 0 {
		email.HasAttachments = true
		email.AttachmentsCount = len(attachments)
		email.Attachments = attachments
	}

	return nil
}

// parseAttachment 解析附件
func (p *RFC822Parser) parseAttachment(part *multipart.Part, disposition, mediaType string) (*adapter.Attachment, error) {
	// 获取文件名
	filename := part.FileName()
	if filename == "" {
		// 尝试从 Content-Disposition 解析
		_, params, _ := mime.ParseMediaType(disposition)
		filename = params["filename"]
	}
	if filename != "" {
		// 解码文件名
		decoded, err := decodeRFC2047(filename)
		if err == nil {
			filename = decoded
		}
	}

	att := &adapter.Attachment{
		Filename:    filename,
		ContentType: mediaType,
	}

	// 检查是否为内联附件
	if strings.HasPrefix(disposition, "inline") {
		att.IsInline = true
		att.ContentID = strings.Trim(part.Header.Get("Content-ID"), "<>")
	}

	// 读取附件内容
	if p.ParseAttachment {
		content, err := p.readPartBody(part)
		if err == nil {
			att.Content = []byte(content)
			att.SizeBytes = int64(len(att.Content))
		}
	} else {
		// 只计算大小，不保存内容
		data, err := io.ReadAll(io.LimitReader(part, int64(p.MaxBodySize)))
		if err == nil {
			att.SizeBytes = int64(len(data))
		}
	}

	return att, nil
}

// readBody 读取邮件正文（处理编码）
func (p *RFC822Parser) readBody(body io.Reader, header mail.Header) (string, error) {
	// 获取传输编码
	encoding := header.Get("Content-Transfer-Encoding")

	// 获取字符集
	contentType := header.Get("Content-Type")
	charset := "utf-8"
	if contentType != "" {
		_, params, _ := mime.ParseMediaType(contentType)
		if cs := params["charset"]; cs != "" {
			charset = strings.ToLower(cs)
		}
	}

	// 读取原始内容
	rawData, err := io.ReadAll(io.LimitReader(body, int64(p.MaxBodySize)))
	if err != nil {
		return "", WrapError(ErrCodeParseError, "读取正文失败", err)
	}

	// 解码传输编码
	var decoded []byte
	switch strings.ToLower(encoding) {
	case "base64":
		decoded, err = base64.StdEncoding.DecodeString(string(rawData))
		if err != nil {
			// 尝试宽松解码
			decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(rawData)))
			if err != nil {
				decoded = rawData
			}
		}
	case "quoted-printable":
		reader := quotedprintable.NewReader(bytes.NewReader(rawData))
		decoded, err = io.ReadAll(reader)
		if err != nil {
			decoded = rawData
		}
	default:
		decoded = rawData
	}

	// 转换字符集
	result := decodeCharset(decoded, charset)
	return result, nil
}

// readPartBody 读取 multipart 部分的正文
func (p *RFC822Parser) readPartBody(part *multipart.Part) (string, error) {
	// 获取传输编码
	encoding := part.Header.Get("Content-Transfer-Encoding")

	// 获取字符集
	contentType := part.Header.Get("Content-Type")
	charset := "utf-8"
	if contentType != "" {
		_, params, _ := mime.ParseMediaType(contentType)
		if cs := params["charset"]; cs != "" {
			charset = strings.ToLower(cs)
		}
	}

	// 读取原始内容
	rawData, err := io.ReadAll(io.LimitReader(part, int64(p.MaxBodySize)))
	if err != nil {
		return "", err
	}

	// 解码传输编码
	var decoded []byte
	switch strings.ToLower(encoding) {
	case "base64":
		decoded, err = base64.StdEncoding.DecodeString(string(rawData))
		if err != nil {
			decoded = rawData
		}
	case "quoted-printable":
		reader := quotedprintable.NewReader(bytes.NewReader(rawData))
		decoded, err = io.ReadAll(reader)
		if err != nil {
			decoded = rawData
		}
	default:
		decoded = rawData
	}

	// 转换字符集
	result := decodeCharset(decoded, charset)
	return result, nil
}

// ============================================
// 辅助函数
// ============================================

// decodeRFC2047 解码 RFC 2047 编码的字符串（如邮件主题）
func decodeRFC2047(s string) (string, error) {
	dec := new(mime.WordDecoder)
	dec.CharsetReader = charsetReader
	return dec.DecodeHeader(s)
}

// charsetReader 字符集读取器
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	charset = strings.ToLower(charset)

	switch charset {
	case "gb2312", "gbk", "gb18030":
		return simplifiedchinese.GBK.NewDecoder().Reader(input), nil
	case "iso-8859-1", "latin1":
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	case "iso-8859-15", "latin9":
		return charmap.ISO8859_15.NewDecoder().Reader(input), nil
	case "windows-1252", "cp1252":
		return charmap.Windows1252.NewDecoder().Reader(input), nil
	case "utf-8", "utf8", "":
		return input, nil
	default:
		// 尝试作为 UTF-8 处理
		return input, nil
	}
}

// decodeCharset 解码字符集
func decodeCharset(data []byte, charset string) string {
	charset = strings.ToLower(charset)

	switch charset {
	case "gb2312", "gbk", "gb18030":
		decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(data)
		if err == nil {
			return string(decoded)
		}
	case "iso-8859-1", "latin1":
		decoded, err := charmap.ISO8859_1.NewDecoder().Bytes(data)
		if err == nil {
			return string(decoded)
		}
	case "iso-8859-15", "latin9":
		decoded, err := charmap.ISO8859_15.NewDecoder().Bytes(data)
		if err == nil {
			return string(decoded)
		}
	case "windows-1252", "cp1252":
		decoded, err := charmap.Windows1252.NewDecoder().Bytes(data)
		if err == nil {
			return string(decoded)
		}
	}

	// 默认作为 UTF-8
	return string(data)
}

// splitAddresses 简单分割邮箱地址
func splitAddresses(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			// 尝试提取邮箱地址
			if idx := strings.Index(p, "<"); idx >= 0 {
				end := strings.Index(p, ">")
				if end > idx {
					p = p[idx+1 : end]
				}
			}
			result = append(result, p)
		}
	}
	return result
}

// ExtractTargetAddress 从邮件中提取目标地址
// 优先使用 To 地址，如果有多个则返回第一个
func ExtractTargetAddress(email *adapter.Email) string {
	if len(email.ToAddresses) > 0 {
		return email.ToAddresses[0]
	}
	return ""
}

// GenerateSnippet 生成邮件摘要
func GenerateSnippet(email *adapter.Email, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 200
	}

	// 优先使用纯文本
	text := email.TextBody
	if text == "" {
		// 从 HTML 提取文本（简单处理）
		text = stripHTML(email.HTMLBody)
	}

	// 清理空白字符
	text = strings.Join(strings.Fields(text), " ")

	// 截断
	if len(text) > maxLength {
		text = text[:maxLength] + "..."
	}

	return text
}

// stripHTML 简单移除 HTML 标签
func stripHTML(html string) string {
	var result strings.Builder
	inTag := false

	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ParseEmailAddress 解析邮箱地址
func ParseEmailAddress(s string) (name, address string) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", s
	}
	return addr.Name, addr.Address
}

// FormatEmailAddress 格式化邮箱地址
func FormatEmailAddress(name, address string) string {
	if name == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}
