package adapter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SMTPSender SMTP 邮件发送器
// Requirements: 2.3, 4.4
type SMTPSender struct {
	config *SMTPConfig
	email  string // 发件人邮箱地址
}

// NewSMTPSender 创建 SMTP 发送器
func NewSMTPSender(config *SMTPConfig, email string) *SMTPSender {
	return &SMTPSender{
		config: config,
		email:  email,
	}
}

// Send 发送邮件
// Requirements: 2.3
func (s *SMTPSender) Send(ctx context.Context, email *OutgoingEmail) (*SendResult, error) {
	// 生成 Message-ID
	messageID := generateMessageID(s.email)

	// 构建邮件内容
	msg, err := s.buildMessage(email, messageID)
	if err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("构建邮件失败: %v", err),
			SenderType: SenderTypeSMTP,
		}, err
	}

	// 获取所有收件人
	recipients := make([]string, 0, len(email.To)+len(email.Cc)+len(email.Bcc))
	recipients = append(recipients, email.To...)
	recipients = append(recipients, email.Cc...)
	recipients = append(recipients, email.Bcc...)

	// 发送邮件
	if err := s.sendMail(ctx, recipients, msg); err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("发送邮件失败: %v", err),
			SenderType: SenderTypeSMTP,
		}, err
	}

	return &SendResult{
		MessageID:  messageID,
		Success:    true,
		SenderType: SenderTypeSMTP,
	}, nil
}

// TestConnection 测试 SMTP 连接
// Requirements: 3.2
func (s *SMTPSender) TestConnection(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 创建带超时的连接
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	var conn net.Conn
	var err error

	// 根据加密方式建立连接
	switch strings.ToLower(s.config.Encryption) {
	case "tls", "ssl":
		// 直接 TLS 连接
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	default:
		// 普通 TCP 连接
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	defer conn.Close()

	// 创建 SMTP 客户端
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	// 如果是 STARTTLS，升级连接
	if strings.ToLower(s.config.Encryption) == "starttls" {
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS 升级失败: %w", err)
		}
	}

	// 认证
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}

	return nil
}

// GetSenderType 获取发送器类型
func (s *SMTPSender) GetSenderType() string {
	return SenderTypeSMTP
}

// sendMail 发送邮件
func (s *SMTPSender) sendMail(ctx context.Context, recipients []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 创建带超时的连接
	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	var conn net.Conn
	var client *smtp.Client
	var err error

	// 根据加密方式建立连接
	encryption := strings.ToLower(s.config.Encryption)

	if encryption == "tls" || encryption == "ssl" {
		// 隐式 TLS 连接（端口 465）
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS 连接 SMTP 服务器失败: %w", err)
		}
	} else {
		// 普通 TCP 连接（用于 STARTTLS 或无加密）
		conn, err = dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
		}
	}
	defer conn.Close()

	// 创建 SMTP 客户端
	client, err = smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	// 如果是 STARTTLS，升级连接
	if encryption == "starttls" {
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS 升级失败: %w", err)
		}
	}

	// 认证
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}

	// 设置发件人
	if err := client.Mail(s.email); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	// 设置收件人
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("设置收件人 %s 失败: %w", rcpt, err)
		}
	}

	// 发送邮件内容
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("关闭数据写入器失败: %w", err)
	}

	// 发送 QUIT 命令
	// 注意：某些 SMTP 服务器（如 QQ 邮箱）在 TLS 连接上可能返回异常响应
	// 但此时邮件已经成功发送（Data 阶段已完成），所以 Quit 错误不影响结果
	if err := client.Quit(); err != nil {
		// 记录错误但不返回，因为邮件已发送成功
		// 常见原因：服务器在 QUIT 后立即关闭连接，导致读取响应失败
		return nil
	}
	return nil
}

// buildMessage 构建邮件消息
func (s *SMTPSender) buildMessage(email *OutgoingEmail, messageID string) ([]byte, error) {
	var buf bytes.Buffer

	// 写入邮件头
	s.writeHeader(&buf, "From", s.formatFrom(email))
	s.writeHeader(&buf, "To", strings.Join(email.To, ", "))
	if len(email.Cc) > 0 {
		s.writeHeader(&buf, "Cc", strings.Join(email.Cc, ", "))
	}
	s.writeHeader(&buf, "Subject", encodeSubject(email.Subject))
	s.writeHeader(&buf, "Message-ID", fmt.Sprintf("<%s>", messageID))
	s.writeHeader(&buf, "Date", time.Now().Format(time.RFC1123Z))
	s.writeHeader(&buf, "MIME-Version", "1.0")

	// 回复/转发相关头
	if email.InReplyTo != "" {
		s.writeHeader(&buf, "In-Reply-To", fmt.Sprintf("<%s>", email.InReplyTo))
	}
	if email.References != "" {
		s.writeHeader(&buf, "References", email.References)
	}
	if email.ReplyTo != "" {
		s.writeHeader(&buf, "Reply-To", email.ReplyTo)
	}

	// 根据内容类型构建邮件体
	if len(email.Attachments) > 0 {
		// 有附件，使用 multipart/mixed
		return s.buildMultipartMessage(&buf, email)
	} else if email.HTMLBody != "" && email.TextBody != "" {
		// 同时有 HTML 和纯文本，使用 multipart/alternative
		return s.buildAlternativeMessage(&buf, email)
	} else if email.HTMLBody != "" {
		// 只有 HTML
		s.writeHeader(&buf, "Content-Type", "text/html; charset=utf-8")
		s.writeHeader(&buf, "Content-Transfer-Encoding", "base64")
		buf.WriteString("\r\n")
		buf.WriteString(base64.StdEncoding.EncodeToString([]byte(email.HTMLBody)))
	} else {
		// 只有纯文本
		s.writeHeader(&buf, "Content-Type", "text/plain; charset=utf-8")
		s.writeHeader(&buf, "Content-Transfer-Encoding", "base64")
		buf.WriteString("\r\n")
		buf.WriteString(base64.StdEncoding.EncodeToString([]byte(email.TextBody)))
	}

	return buf.Bytes(), nil
}

// buildMultipartMessage 构建带附件的多部分消息
// Requirements: 4.4
func (s *SMTPSender) buildMultipartMessage(buf *bytes.Buffer, email *OutgoingEmail) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	s.writeHeader(buf, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	buf.WriteString("\r\n")

	// 写入邮件正文部分
	if email.HTMLBody != "" || email.TextBody != "" {
		bodyPart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type": []string{"multipart/alternative"},
		})
		if err != nil {
			return nil, err
		}

		altWriter := multipart.NewWriter(bodyPart)

		// 纯文本部分
		if email.TextBody != "" {
			textPart, err := altWriter.CreatePart(textproto.MIMEHeader{
				"Content-Type":              []string{"text/plain; charset=utf-8"},
				"Content-Transfer-Encoding": []string{"base64"},
			})
			if err != nil {
				return nil, err
			}
			textPart.Write([]byte(base64.StdEncoding.EncodeToString([]byte(email.TextBody))))
		}

		// HTML 部分
		if email.HTMLBody != "" {
			htmlPart, err := altWriter.CreatePart(textproto.MIMEHeader{
				"Content-Type":              []string{"text/html; charset=utf-8"},
				"Content-Transfer-Encoding": []string{"base64"},
			})
			if err != nil {
				return nil, err
			}
			htmlPart.Write([]byte(base64.StdEncoding.EncodeToString([]byte(email.HTMLBody))))
		}

		altWriter.Close()
	}

	// 写入附件部分
	for _, att := range email.Attachments {
		if err := s.writeAttachment(writer, &att); err != nil {
			return nil, err
		}
	}

	writer.Close()
	return buf.Bytes(), nil
}

// buildAlternativeMessage 构建 multipart/alternative 消息
func (s *SMTPSender) buildAlternativeMessage(buf *bytes.Buffer, email *OutgoingEmail) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	s.writeHeader(buf, "Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", boundary))
	buf.WriteString("\r\n")

	// 纯文本部分
	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              []string{"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": []string{"base64"},
	})
	if err != nil {
		return nil, err
	}
	textPart.Write([]byte(base64.StdEncoding.EncodeToString([]byte(email.TextBody))))

	// HTML 部分
	htmlPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              []string{"text/html; charset=utf-8"},
		"Content-Transfer-Encoding": []string{"base64"},
	})
	if err != nil {
		return nil, err
	}
	htmlPart.Write([]byte(base64.StdEncoding.EncodeToString([]byte(email.HTMLBody))))

	writer.Close()
	return buf.Bytes(), nil
}

// writeAttachment 写入附件
// Requirements: 4.4
func (s *SMTPSender) writeAttachment(writer *multipart.Writer, att *OutgoingAttachment) error {
	contentType := att.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := textproto.MIMEHeader{
		"Content-Type":              []string{fmt.Sprintf("%s; name=\"%s\"", contentType, encodeFilename(att.Filename))},
		"Content-Transfer-Encoding": []string{"base64"},
	}

	if att.IsInline && att.ContentID != "" {
		header.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", encodeFilename(att.Filename)))
		header.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
	} else {
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", encodeFilename(att.Filename)))
	}

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	// Base64 编码附件内容
	encoded := base64.StdEncoding.EncodeToString(att.Content)
	_, err = part.Write([]byte(encoded))
	return err
}

// writeHeader 写入邮件头
func (s *SMTPSender) writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

// formatFrom 格式化发件人
func (s *SMTPSender) formatFrom(email *OutgoingEmail) string {
	if email.FromName != "" {
		return fmt.Sprintf("%s <%s>", encodeWord(email.FromName), s.email)
	}
	return s.email
}

// generateMessageID 生成 Message-ID
func generateMessageID(email string) string {
	parts := strings.Split(email, "@")
	domain := "localhost"
	if len(parts) > 1 {
		domain = parts[1]
	}
	return fmt.Sprintf("%s@%s", uuid.New().String(), domain)
}

// encodeSubject 编码邮件主题（支持中文）
func encodeSubject(subject string) string {
	return mime.QEncoding.Encode("utf-8", subject)
}

// encodeWord 编码单词（用于发件人名称等）
func encodeWord(word string) string {
	return mime.QEncoding.Encode("utf-8", word)
}

// encodeFilename 编码文件名
func encodeFilename(filename string) string {
	return mime.QEncoding.Encode("utf-8", filename)
}
