package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

// GraphAdapter Microsoft Graph API 适配器
type GraphAdapter struct {
	config     *Config
	httpClient *http.Client
	baseURL    string
}

// GraphMessage Graph API 邮件响应结构
type GraphMessage struct {
	ID                      string           `json:"id"`
	Subject                 string           `json:"subject"`
	BodyPreview             string           `json:"bodyPreview"`
	Body                    GraphItemBody    `json:"body"`
	From                    GraphRecipient   `json:"from"`
	ToRecipients            []GraphRecipient `json:"toRecipients"`
	CcRecipients            []GraphRecipient `json:"ccRecipients"`
	BccRecipients           []GraphRecipient `json:"bccRecipients"`
	ReplyTo                 []GraphRecipient `json:"replyTo"`
	SentDateTime            string           `json:"sentDateTime"`
	ReceivedDateTime        string           `json:"receivedDateTime"`
	HasAttachments          bool             `json:"hasAttachments"`
	InternetMessageID       string           `json:"internetMessageId"`
	ConversationID          string           `json:"conversationId"`
	IsRead                  bool             `json:"isRead"`
	Categories              []string         `json:"categories"`
	InferenceClassification string           `json:"inferenceClassification"`
}

// GraphItemBody 邮件正文
type GraphItemBody struct {
	ContentType string `json:"contentType"` // text 或 html
	Content     string `json:"content"`
}

// GraphRecipient 收件人信息
type GraphRecipient struct {
	EmailAddress GraphEmailAddress `json:"emailAddress"`
}

// GraphEmailAddress 邮件地址
type GraphEmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// GraphMessageList 邮件列表响应
type GraphMessageList struct {
	Value    []GraphMessage `json:"value"`
	NextLink string         `json:"@odata.nextLink"`
}

// GraphAttachment 附件信息
type GraphAttachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	IsInline    bool   `json:"isInline"`
	ContentID   string `json:"contentId"`
}

// GraphAttachmentList 附件列表响应
type GraphAttachmentList struct {
	Value []GraphAttachment `json:"value"`
}

// NewGraphAdapter 创建 Graph 适配器实例
func NewGraphAdapter(config *Config) (*GraphAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证 OAuth2 凭证
	if config.Credentials.AccessToken == "" {
		return nil, fmt.Errorf("access token is required for Microsoft Graph API")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &GraphAdapter{
		config:  config,
		baseURL: "https://graph.microsoft.com/v1.0",
	}, nil
}

// Connect 连接到 Microsoft Graph API
func (a *GraphAdapter) Connect(ctx context.Context) error {
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
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
		Scopes: []string{
			"https://graph.microsoft.com/Mail.Read",
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

	a.httpClient = httpClient
	return nil
}

// getProxyURL 获取代理 URL
func (a *GraphAdapter) getProxyURL() *url.URL {
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
func (a *GraphAdapter) Disconnect() error {
	// Graph API 是无状态的，不需要断开连接
	a.httpClient = nil
	return nil
}

// FetchEmails 拉取邮件列表
func (a *GraphAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	if a.httpClient == nil {
		return nil, fmt.Errorf("not connected to Microsoft Graph API")
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("$top", "100") // 每次最多获取 100 封
	params.Set("$orderby", "receivedDateTime DESC")

	// 添加时间过滤
	if !since.IsZero() {
		filter := fmt.Sprintf("receivedDateTime ge %s", since.Format(time.RFC3339))
		params.Set("$filter", filter)
	}

	// 限制结果数量
	if limit > 0 && limit < 100 {
		params.Set("$top", fmt.Sprintf("%d", limit))
	}

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/me/messages?%s", a.baseURL, params.Encode())

	// 发送请求
	resp, err := a.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		emails = append(emails, email)
	}

	return emails, nil
}

// FetchEmailDetail 获取邮件详情
func (a *GraphAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	if a.httpClient == nil {
		return nil, fmt.Errorf("not connected to Microsoft Graph API")
	}

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/me/messages/%s", a.baseURL, providerID)

	// 发送请求
	resp, err := a.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var msg GraphMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为 Email 对象
	email := a.convertGraphMessageToEmail(&msg)

	// 获取附件信息
	if msg.HasAttachments {
		attachments, err := a.fetchAttachments(ctx, providerID)
		if err == nil {
			email.Attachments = attachments
			email.AttachmentsCount = len(attachments)
		}
	}

	return email, nil
}

// fetchAttachments 获取附件列表
func (a *GraphAdapter) fetchAttachments(ctx context.Context, messageID string) ([]Attachment, error) {
	requestURL := fmt.Sprintf("%s/me/messages/%s/attachments", a.baseURL, messageID)

	resp, err := a.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attachments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Graph API returned status %d", resp.StatusCode)
	}

	var attachmentList GraphAttachmentList
	if err := json.NewDecoder(resp.Body).Decode(&attachmentList); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	// 转换为 Attachment 对象
	attachments := make([]Attachment, 0, len(attachmentList.Value))
	for _, att := range attachmentList.Value {
		attachments = append(attachments, Attachment{
			Filename:    att.Name,
			ContentType: att.ContentType,
			SizeBytes:   att.Size,
			IsInline:    att.IsInline,
			ContentID:   att.ContentID,
		})
	}

	return attachments, nil
}

// convertGraphMessageToEmail 转换 Graph 消息为 Email 对象
func (a *GraphAdapter) convertGraphMessageToEmail(msg *GraphMessage) *Email {
	email := &Email{
		ProviderID:     msg.ID,
		MessageID:      msg.InternetMessageID,
		Subject:        msg.Subject,
		Snippet:        msg.BodyPreview,
		ThreadID:       msg.ConversationID,
		HasAttachments: msg.HasAttachments,
		SourceLabels:   msg.Categories,
		SourceIsRead:   &msg.IsRead,
	}

	// 解析发件人
	email.FromAddress = msg.From.EmailAddress.Address
	email.FromName = msg.From.EmailAddress.Name

	// 解析收件人
	email.ToAddresses = make([]string, len(msg.ToRecipients))
	for i, recipient := range msg.ToRecipients {
		email.ToAddresses[i] = recipient.EmailAddress.Address
	}

	// 解析抄送
	email.CcAddresses = make([]string, len(msg.CcRecipients))
	for i, recipient := range msg.CcRecipients {
		email.CcAddresses[i] = recipient.EmailAddress.Address
	}

	// 解析密送
	email.BccAddresses = make([]string, len(msg.BccRecipients))
	for i, recipient := range msg.BccRecipients {
		email.BccAddresses[i] = recipient.EmailAddress.Address
	}

	// 解析回复地址
	if len(msg.ReplyTo) > 0 {
		email.ReplyTo = msg.ReplyTo[0].EmailAddress.Address
	}

	// 解析时间
	if sentTime, err := time.Parse(time.RFC3339, msg.SentDateTime); err == nil {
		email.SentAt = sentTime
	}
	if receivedTime, err := time.Parse(time.RFC3339, msg.ReceivedDateTime); err == nil {
		email.ReceivedAt = receivedTime
	}

	// 解析邮件正文
	if msg.Body.ContentType == "html" {
		email.HTMLBody = msg.Body.Content
	} else {
		email.TextBody = msg.Body.Content
	}

	return email
}

// GetProviderType 获取提供商类型
func (a *GraphAdapter) GetProviderType() string {
	return "outlook"
}

// GetProtocol 获取协议类型
func (a *GraphAdapter) GetProtocol() string {
	return "graph"
}

// TestConnection 测试连接
func (a *GraphAdapter) TestConnection(ctx context.Context) error {
	if a.httpClient == nil {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}

	// 测试获取用户信息
	requestURL := fmt.Sprintf("%s/me", a.baseURL)
	resp, err := a.httpClient.Get(requestURL)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
