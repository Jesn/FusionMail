package adapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailSender Gmail API 邮件发送器
// Requirements: 2.1
type GmailSender struct {
	config       *SenderConfig
	service      *gmail.Service
	oauth2Config *oauth2.Config
	httpClient   *http.Client
}

// NewGmailSender 创建 Gmail 发送器
func NewGmailSender(config *SenderConfig) (*GmailSender, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.AccessToken == "" {
		return nil, fmt.Errorf("access token is required for Gmail API")
	}

	return &GmailSender{
		config: config,
	}, nil
}

// connect 连接到 Gmail API
func (s *GmailSender) connect(ctx context.Context) error {
	if s.service != nil {
		return nil
	}

	// 创建 OAuth2 token
	token := &oauth2.Token{
		AccessToken:  s.config.AccessToken,
		RefreshToken: s.config.RefreshToken,
		TokenType:    "Bearer",
	}

	// 创建 OAuth2 配置
	oauth2Config := &oauth2.Config{
		ClientID:     s.config.ClientID,
		ClientSecret: s.config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			gmail.GmailSendScope,
		},
	}

	s.oauth2Config = oauth2Config

	// 创建 HTTP 客户端
	httpClient := oauth2Config.Client(ctx, token)

	// 如果配置了代理，设置代理
	if s.config.Proxy != nil && s.config.Proxy.Enabled {
		transport := httpClient.Transport.(*oauth2.Transport)
		transport.Base = &http.Transport{
			Proxy: http.ProxyURL(s.getProxyURL()),
		}
	}

	s.httpClient = httpClient

	// 创建 Gmail 服务
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	s.service = service
	return nil
}

// getProxyURL 获取代理 URL
func (s *GmailSender) getProxyURL() *url.URL {
	if s.config.Proxy == nil || !s.config.Proxy.Enabled {
		return nil
	}

	proxyURL := fmt.Sprintf("%s://%s:%d",
		s.config.Proxy.Type,
		s.config.Proxy.Host,
		s.config.Proxy.Port,
	)

	if s.config.Proxy.Username != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%d",
			s.config.Proxy.Type,
			s.config.Proxy.Username,
			s.config.Proxy.Password,
			s.config.Proxy.Host,
			s.config.Proxy.Port,
		)
	}

	parsedURL, _ := url.Parse(proxyURL)
	return parsedURL
}

// Send 发送邮件
// Requirements: 2.1
func (s *GmailSender) Send(ctx context.Context, email *OutgoingEmail) (*SendResult, error) {
	// 确保已连接
	if err := s.connect(ctx); err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("连接 Gmail API 失败: %v", err),
			SenderType: SenderTypeGmailAPI,
		}, err
	}

	// 生成 Message-ID
	messageID := s.generateMessageID()

	// 构建 RFC 2822 格式的邮件消息
	rawMessage, err := s.buildRawMessage(email, messageID)
	if err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("构建邮件失败: %v", err),
			SenderType: SenderTypeGmailAPI,
		}, err
	}

	// Base64url 编码
	encodedMessage := base64.URLEncoding.EncodeToString(rawMessage)

	// 创建 Gmail 消息
	gmailMessage := &gmail.Message{
		Raw: encodedMessage,
	}

	// 发送邮件
	sentMessage, err := s.service.Users.Messages.Send("me", gmailMessage).Do()
	if err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("发送邮件失败: %v", err),
			SenderType: SenderTypeGmailAPI,
		}, err
	}

	return &SendResult{
		ProviderMsgID: sentMessage.Id,
		MessageID:     messageID,
		Success:       true,
		SenderType:    SenderTypeGmailAPI,
	}, nil
}

// TestConnection 测试连接
func (s *GmailSender) TestConnection(ctx context.Context) error {
	if err := s.connect(ctx); err != nil {
		return err
	}

	// 测试获取用户配置文件
	_, err := s.service.Users.GetProfile("me").Do()
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	return nil
}

// GetSenderType 获取发送器类型
func (s *GmailSender) GetSenderType() string {
	return SenderTypeGmailAPI
}

// buildRawMessage 构建 RFC 2822 格式的邮件消息
// Requirements: 2.1, 4.4
func (s *GmailSender) buildRawMessage(email *OutgoingEmail, messageID string) ([]byte, error) {
	var buf bytes.Buffer

	// 写入邮件头
	s.writeHeader(&buf, "From", s.formatFrom(email))
	s.writeHeader(&buf, "To", strings.Join(email.To, ", "))
	if len(email.Cc) > 0 {
		s.writeHeader(&buf, "Cc", strings.Join(email.Cc, ", "))
	}
	s.writeHeader(&buf, "Subject", s.encodeSubject(email.Subject))
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
		return s.buildMultipartMessage(&buf, email)
	} else if email.HTMLBody != "" && email.TextBody != "" {
		return s.buildAlternativeMessage(&buf, email)
	} else if email.HTMLBody != "" {
		s.writeHeader(&buf, "Content-Type", "text/html; charset=utf-8")
		s.writeHeader(&buf, "Content-Transfer-Encoding", "base64")
		buf.WriteString("\r\n")
		buf.WriteString(base64.StdEncoding.EncodeToString([]byte(email.HTMLBody)))
	} else {
		s.writeHeader(&buf, "Content-Type", "text/plain; charset=utf-8")
		s.writeHeader(&buf, "Content-Transfer-Encoding", "base64")
		buf.WriteString("\r\n")
		buf.WriteString(base64.StdEncoding.EncodeToString([]byte(email.TextBody)))
	}

	return buf.Bytes(), nil
}

// buildMultipartMessage 构建带附件的多部分消息
func (s *GmailSender) buildMultipartMessage(buf *bytes.Buffer, email *OutgoingEmail) ([]byte, error) {
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
func (s *GmailSender) buildAlternativeMessage(buf *bytes.Buffer, email *OutgoingEmail) ([]byte, error) {
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
func (s *GmailSender) writeAttachment(writer *multipart.Writer, att *OutgoingAttachment) error {
	contentType := att.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := textproto.MIMEHeader{
		"Content-Type":              []string{fmt.Sprintf("%s; name=\"%s\"", contentType, s.encodeFilename(att.Filename))},
		"Content-Transfer-Encoding": []string{"base64"},
	}

	if att.IsInline && att.ContentID != "" {
		header.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", s.encodeFilename(att.Filename)))
		header.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
	} else {
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", s.encodeFilename(att.Filename)))
	}

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(att.Content)
	_, err = part.Write([]byte(encoded))
	return err
}

// writeHeader 写入邮件头
func (s *GmailSender) writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

// formatFrom 格式化发件人
func (s *GmailSender) formatFrom(email *OutgoingEmail) string {
	if email.FromName != "" {
		return fmt.Sprintf("%s <%s>", s.encodeWord(email.FromName), s.config.Email)
	}
	return s.config.Email
}

// generateMessageID 生成 Message-ID
func (s *GmailSender) generateMessageID() string {
	parts := strings.Split(s.config.Email, "@")
	domain := "gmail.com"
	if len(parts) > 1 {
		domain = parts[1]
	}
	return fmt.Sprintf("%s@%s", uuid.New().String(), domain)
}

// encodeSubject 编码邮件主题
func (s *GmailSender) encodeSubject(subject string) string {
	return mime.QEncoding.Encode("utf-8", subject)
}

// encodeWord 编码单词
func (s *GmailSender) encodeWord(word string) string {
	return mime.QEncoding.Encode("utf-8", word)
}

// encodeFilename 编码文件名
func (s *GmailSender) encodeFilename(filename string) string {
	return mime.QEncoding.Encode("utf-8", filename)
}
