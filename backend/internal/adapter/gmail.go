package adapter

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailAdapter Gmail API 适配器
type GmailAdapter struct {
	config  *Config
	service *gmail.Service
}

// NewGmailAdapter 创建 Gmail 适配器实例
func NewGmailAdapter(config *Config) (*GmailAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证 OAuth2 凭证
	if config.Credentials.AccessToken == "" {
		return nil, fmt.Errorf("access token is required for Gmail API")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &GmailAdapter{
		config: config,
	}, nil
}

// Connect 连接到 Gmail API
func (a *GmailAdapter) Connect(ctx context.Context) error {
	// 创建 OAuth2 token
	token := &oauth2.Token{
		AccessToken:  a.config.Credentials.AccessToken,
		RefreshToken: a.config.Credentials.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       a.config.Credentials.TokenExpiry,
	}

	// 创建 OAuth2 配置
	oauth2Config := &oauth2.Config{
		ClientID:     a.config.Credentials.ClientID,
		ClientSecret: a.config.Credentials.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			gmail.GmailReadonlyScope,
		},
	}

	// 创建 HTTP 客户端
	httpClient := oauth2Config.Client(ctx, token)

	// 如果配置了代理，设置代理
	if a.config.Proxy != nil && a.config.Proxy.Enabled {
		transport := httpClient.Transport.(*oauth2.Transport)
		transport.Base = &http.Transport{
			Proxy: http.ProxyURL(a.getProxyURL()),
		}
	}

	// 创建 Gmail 服务
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	a.service = service
	return nil
}

// getProxyURL 获取代理 URL
func (a *GmailAdapter) getProxyURL() *url.URL {
	if a.config.Proxy == nil || !a.config.Proxy.Enabled {
		return nil
	}

	proxyURL := fmt.Sprintf("%s://%s:%d",
		a.config.Proxy.Type,
		a.config.Proxy.Host,
		a.config.Proxy.Port,
	)

	if a.config.Proxy.Username != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%d",
			a.config.Proxy.Type,
			a.config.Proxy.Username,
			a.config.Proxy.Password,
			a.config.Proxy.Host,
			a.config.Proxy.Port,
		)
	}

	parsedURL, _ := url.Parse(proxyURL)
	return parsedURL
}

// Disconnect 断开连接
func (a *GmailAdapter) Disconnect() error {
	// Gmail API 是无状态的，不需要断开连接
	a.service = nil
	return nil
}

// FetchEmails 拉取邮件列表
func (a *GmailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	if a.service == nil {
		return nil, fmt.Errorf("not connected to Gmail API")
	}

	// 构建查询条件
	query := "in:inbox"
	if !since.IsZero() {
		// Gmail 使用 after: 语法进行时间过滤
		query += fmt.Sprintf(" after:%d", since.Unix())
	}

	// 设置最大结果数
	maxResults := int64(100)
	if limit > 0 && limit < 100 {
		maxResults = int64(limit)
	}

	// 调用 Gmail API 列出邮件
	listCall := a.service.Users.Messages.List("me").
		Q(query).
		MaxResults(maxResults)

	response, err := listCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	if len(response.Messages) == 0 {
		return []*Email{}, nil
	}

	// 获取邮件详情
	emails := make([]*Email, 0, len(response.Messages))
	for _, msg := range response.Messages {
		select {
		case <-ctx.Done():
			return emails, ctx.Err()
		default:
		}

		email, err := a.FetchEmailDetail(ctx, msg.Id)
		if err != nil {
			// 记录错误但继续处理其他邮件
			continue
		}

		emails = append(emails, email)
	}

	return emails, nil
}

// FetchEmailDetail 获取邮件详情
func (a *GmailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	if a.service == nil {
		return nil, fmt.Errorf("not connected to Gmail API")
	}

	// 获取邮件详情
	msg, err := a.service.Users.Messages.Get("me", providerID).
		Format("full").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// 解析邮件
	email := &Email{
		ProviderID:     msg.Id,
		ThreadID:       msg.ThreadId,
		SizeBytes:      msg.SizeEstimate,
		SourceLabels:   msg.LabelIds,
		HasAttachments: false,
	}

	// 解析邮件头
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "Message-ID":
			email.MessageID = header.Value
		case "Subject":
			email.Subject = header.Value
		case "From":
			email.FromAddress, email.FromName = parseEmailAddress(header.Value)
		case "To":
			email.ToAddresses = parseEmailAddresses(header.Value)
		case "Cc":
			email.CcAddresses = parseEmailAddresses(header.Value)
		case "Bcc":
			email.BccAddresses = parseEmailAddresses(header.Value)
		case "Reply-To":
			email.ReplyTo = header.Value
		case "In-Reply-To":
			email.InReplyTo = header.Value
		case "References":
			email.References = header.Value
		case "Date":
			if t, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
				email.SentAt = t
			}
		}
	}

	// 设置接收时间
	email.ReceivedAt = time.Unix(msg.InternalDate/1000, 0)
	if email.SentAt.IsZero() {
		email.SentAt = email.ReceivedAt
	}

	// 解析邮件正文和附件
	a.parseMessagePart(msg.Payload, email)

	// 生成摘要
	if email.Snippet == "" {
		email.Snippet = msg.Snippet
	}

	// 判断是否已读
	isRead := !contains(msg.LabelIds, "UNREAD")
	email.SourceIsRead = &isRead

	return email, nil
}

// parseMessagePart 解析邮件部分（递归处理多部分邮件）
func (a *GmailAdapter) parseMessagePart(part *gmail.MessagePart, email *Email) {
	// 处理邮件正文
	if part.MimeType == "text/plain" && part.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
		email.TextBody = string(data)
	} else if part.MimeType == "text/html" && part.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
		email.HTMLBody = string(data)
	}

	// 处理附件
	if part.Filename != "" {
		email.HasAttachments = true
		email.AttachmentsCount++

		attachment := Attachment{
			Filename:    part.Filename,
			ContentType: part.MimeType,
			SizeBytes:   part.Body.Size,
		}

		// 检查是否是内联附件
		for _, header := range part.Headers {
			if header.Name == "Content-ID" {
				attachment.IsInline = true
				attachment.ContentID = header.Value
				break
			}
		}

		email.Attachments = append(email.Attachments, attachment)
	}

	// 递归处理子部分
	for _, subPart := range part.Parts {
		a.parseMessagePart(subPart, email)
	}
}

// GetProviderType 获取提供商类型
func (a *GmailAdapter) GetProviderType() string {
	return "gmail"
}

// GetProtocol 获取协议类型
func (a *GmailAdapter) GetProtocol() string {
	return "gmail_api"
}

// TestConnection 测试连接
func (a *GmailAdapter) TestConnection(ctx context.Context) error {
	if a.service == nil {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}

	// 测试获取用户配置文件
	_, err := a.service.Users.GetProfile("me").Do()
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	return nil
}

// 辅助函数

// parseEmailAddress 解析邮件地址
func parseEmailAddress(addr string) (email, name string) {
	// 格式：Name <email@example.com> 或 email@example.com
	if strings.Contains(addr, "<") && strings.Contains(addr, ">") {
		parts := strings.Split(addr, "<")
		name = strings.TrimSpace(parts[0])
		email = strings.TrimSpace(strings.Trim(parts[1], ">"))
	} else {
		email = strings.TrimSpace(addr)
	}
	return
}

// parseEmailAddresses 解析多个邮件地址
func parseEmailAddresses(addrs string) []string {
	if addrs == "" {
		return nil
	}

	parts := strings.Split(addrs, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		email, _ := parseEmailAddress(part)
		if email != "" {
			result = append(result, email)
		}
	}

	return result
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
