package adapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// GraphSender Microsoft Graph API 邮件发送器
// Requirements: 2.2
type GraphSender struct {
	config     *SenderConfig
	httpClient *http.Client
	baseURL    string
}

// NewGraphSender 创建 Graph 发送器
func NewGraphSender(config *SenderConfig) (*GraphSender, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.AccessToken == "" {
		return nil, fmt.Errorf("access token is required for Microsoft Graph API")
	}

	return &GraphSender{
		config:  config,
		baseURL: "https://graph.microsoft.com/v1.0",
	}, nil
}

// connect 连接到 Microsoft Graph API
func (s *GraphSender) connect(ctx context.Context) error {
	if s.httpClient != nil {
		return nil
	}

	// 检查是否有 RefreshToken 和 ClientID/ClientSecret（用于刷新 Token）
	canRefresh := s.config.RefreshToken != "" && s.config.ClientID != "" && s.config.ClientSecret != ""

	// 创建 OAuth2 token
	// 如果 TokenExpiry 是零值或已过期，且有 RefreshToken，则设置为过去的时间以触发刷新
	tokenExpiry := s.config.TokenExpiry
	if canRefresh && (tokenExpiry.IsZero() || tokenExpiry.Before(time.Now())) {
		// 设置为过去的时间，强制 OAuth2 库刷新 Token
		tokenExpiry = time.Now().Add(-time.Hour)
	}

	token := &oauth2.Token{
		AccessToken:  s.config.AccessToken,
		RefreshToken: s.config.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       tokenExpiry,
	}

	// 创建 OAuth2 配置
	oauth2Config := &oauth2.Config{
		ClientID:     s.config.ClientID,
		ClientSecret: s.config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
		Scopes: []string{
			"https://graph.microsoft.com/Mail.Send",
			"offline_access",
		},
	}

	// 如果 Token 已过期且有 RefreshToken，先尝试刷新 Token
	if canRefresh && token.Expiry.Before(time.Now()) {
		tokenSource := oauth2Config.TokenSource(ctx, token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return fmt.Errorf("failed to refresh access token: %w", err)
		}
		token = newToken
	}

	// 创建 HTTP 客户端（OAuth2 库会自动刷新过期的 Token）
	httpClient := oauth2Config.Client(ctx, token)

	// 如果配置了代理，设置代理
	if s.config.Proxy != nil && s.config.Proxy.Enabled {
		transport := httpClient.Transport.(*oauth2.Transport)
		transport.Base = &http.Transport{
			Proxy: http.ProxyURL(s.getProxyURL()),
		}
	}

	s.httpClient = httpClient
	return nil
}

// getProxyURL 获取代理 URL
func (s *GraphSender) getProxyURL() *url.URL {
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
// Requirements: 2.2
func (s *GraphSender) Send(ctx context.Context, email *OutgoingEmail) (*SendResult, error) {
	// 确保已连接
	if err := s.connect(ctx); err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("连接 Microsoft Graph API 失败: %v", err),
			SenderType: SenderTypeGraphAPI,
		}, err
	}

	// 生成 Message-ID
	messageID := s.generateMessageID()

	// 构建 Graph API 请求体
	requestBody, err := s.buildSendMailRequest(email, messageID)
	if err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("构建邮件请求失败: %v", err),
			SenderType: SenderTypeGraphAPI,
		}, err
	}

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/me/sendMail", bytes.NewReader(requestBody))
	if err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("创建请求失败: %v", err),
			SenderType: SenderTypeGraphAPI,
		}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("发送请求失败: %v", err),
			SenderType: SenderTypeGraphAPI,
		}, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &SendResult{
			Success:    false,
			Error:      fmt.Sprintf("发送邮件失败: %s - %s", resp.Status, string(body)),
			SenderType: SenderTypeGraphAPI,
		}, fmt.Errorf("send mail failed: %s", resp.Status)
	}

	return &SendResult{
		MessageID:  messageID,
		Success:    true,
		SenderType: SenderTypeGraphAPI,
	}, nil
}

// TestConnection 测试连接
func (s *GraphSender) TestConnection(ctx context.Context) error {
	if err := s.connect(ctx); err != nil {
		return err
	}

	// 测试获取用户配置文件
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/me", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get user profile: %s", resp.Status)
	}

	return nil
}

// GetSenderType 获取发送器类型
func (s *GraphSender) GetSenderType() string {
	return SenderTypeGraphAPI
}

// GraphSendMailRequest Graph API sendMail 请求结构
type GraphSendMailRequest struct {
	Message         GraphSendMessage `json:"message"`
	SaveToSentItems bool             `json:"saveToSentItems"`
}

// GraphSendMessage Graph API 邮件消息结构
type GraphSendMessage struct {
	Subject                string                       `json:"subject"`
	Body                   GraphItemBody                `json:"body"`
	ToRecipients           []GraphRecipient             `json:"toRecipients"`
	CcRecipients           []GraphRecipient             `json:"ccRecipients,omitempty"`
	BccRecipients          []GraphRecipient             `json:"bccRecipients,omitempty"`
	ReplyTo                []GraphRecipient             `json:"replyTo,omitempty"`
	Attachments            []GraphSendAttachment        `json:"attachments,omitempty"`
	InternetMessageHeaders []GraphInternetMessageHeader `json:"internetMessageHeaders,omitempty"`
}

// GraphSendAttachment Graph API 附件结构
type GraphSendAttachment struct {
	ODataType    string `json:"@odata.type"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	ContentBytes string `json:"contentBytes"` // Base64 编码
	IsInline     bool   `json:"isInline,omitempty"`
	ContentID    string `json:"contentId,omitempty"`
}

// GraphInternetMessageHeader 邮件头
type GraphInternetMessageHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// buildSendMailRequest 构建 sendMail 请求体
// Requirements: 2.2, 4.4
func (s *GraphSender) buildSendMailRequest(email *OutgoingEmail, messageID string) ([]byte, error) {
	// 构建收件人列表
	toRecipients := make([]GraphRecipient, len(email.To))
	for i, addr := range email.To {
		toRecipients[i] = GraphRecipient{
			EmailAddress: GraphEmailAddress{Address: addr},
		}
	}

	ccRecipients := make([]GraphRecipient, len(email.Cc))
	for i, addr := range email.Cc {
		ccRecipients[i] = GraphRecipient{
			EmailAddress: GraphEmailAddress{Address: addr},
		}
	}

	bccRecipients := make([]GraphRecipient, len(email.Bcc))
	for i, addr := range email.Bcc {
		bccRecipients[i] = GraphRecipient{
			EmailAddress: GraphEmailAddress{Address: addr},
		}
	}

	// 确定正文类型
	body := GraphItemBody{
		ContentType: "text",
		Content:     email.TextBody,
	}
	if email.HTMLBody != "" {
		body.ContentType = "html"
		body.Content = email.HTMLBody
	}

	// 构建消息
	message := GraphSendMessage{
		Subject:       email.Subject,
		Body:          body,
		ToRecipients:  toRecipients,
		CcRecipients:  ccRecipients,
		BccRecipients: bccRecipients,
	}

	// 添加回复地址
	if email.ReplyTo != "" {
		message.ReplyTo = []GraphRecipient{
			{EmailAddress: GraphEmailAddress{Address: email.ReplyTo}},
		}
	}

	// 添加邮件头（In-Reply-To, References）
	// 注意：Microsoft Graph API 不允许设置 Message-ID header，它会自动生成
	// internetMessageHeaders 只能包含以 x- 开头的自定义 header 或特定标准 header
	var headers []GraphInternetMessageHeader
	if email.InReplyTo != "" {
		headers = append(headers, GraphInternetMessageHeader{
			Name:  "In-Reply-To",
			Value: fmt.Sprintf("<%s>", email.InReplyTo),
		})
	}
	if email.References != "" {
		headers = append(headers, GraphInternetMessageHeader{
			Name:  "References",
			Value: email.References,
		})
	}
	// 使用自定义 header 存储我们生成的 Message-ID（用于追踪）
	headers = append(headers, GraphInternetMessageHeader{
		Name:  "x-fusionmail-message-id",
		Value: messageID,
	})
	if len(headers) > 0 {
		message.InternetMessageHeaders = headers
	}

	// 添加附件
	if len(email.Attachments) > 0 {
		attachments := make([]GraphSendAttachment, len(email.Attachments))
		for i, att := range email.Attachments {
			contentType := att.ContentType
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			attachments[i] = GraphSendAttachment{
				ODataType:    "#microsoft.graph.fileAttachment",
				Name:         att.Filename,
				ContentType:  contentType,
				ContentBytes: base64.StdEncoding.EncodeToString(att.Content),
				IsInline:     att.IsInline,
				ContentID:    att.ContentID,
			}
		}
		message.Attachments = attachments
	}

	// 构建请求
	request := GraphSendMailRequest{
		Message:         message,
		SaveToSentItems: true,
	}

	return json.Marshal(request)
}

// generateMessageID 生成 Message-ID
func (s *GraphSender) generateMessageID() string {
	parts := strings.Split(s.config.Email, "@")
	domain := "outlook.com"
	if len(parts) > 1 {
		domain = parts[1]
	}
	return fmt.Sprintf("%s@%s", uuid.New().String(), domain)
}
