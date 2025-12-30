package cloudmail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"
)

// init 注册 Cloud Mail 适配器
func init() {
	// 注册适配器创建函数
	webapi.RegisterAdapter(model.WebAPIServiceTypeCloudMail, func(authDataJSON string) (webapi.WebAPIProvider, error) {
		var config model.CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, webapi.WrapError(webapi.ErrCodeConfigError, "解析 Cloud Mail 配置失败", err)
		}
		return NewCloudMailAdapter(&config)
	})

	// 注册服务模板
	webapi.RegisterServiceTemplate(&webapi.ServiceTemplate{
		ServiceType: model.WebAPIServiceTypeCloudMail,
		Name:        "Cloud Mail",
		Description: "Cloud Mail 邮箱服务，支持多账户管理，通过 JWT Token 认证",
		AccessModes: []string{"multi_account"},
		AuthFields: []webapi.AuthField{
			{Name: "base_url", Label: "API 地址", Type: "text", Required: true, Placeholder: "https://cloudmail.example.com"},
			{Name: "jwt_token", Label: "JWT Token", Type: "password", Required: true},
			{Name: "accounts", Label: "账户列表", Type: "textarea", Required: true, HelpText: "JSON 格式的账户列表"},
		},
	})
}

// CloudMailAdapter Cloud Mail 适配器
// 支持多账户管理，通过 JWT Token 认证
type CloudMailAdapter struct {
	*webapi.BaseWebAPIAdapter

	// 配置
	config *model.CloudMailAuthData

	// HTTP 客户端
	httpClient *http.Client

	// 日志
	log *logger.Logger
}

// NewCloudMailAdapter 创建 Cloud Mail 适配器
func NewCloudMailAdapter(config *model.CloudMailAuthData) (*CloudMailAdapter, error) {
	if config == nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置不能为空", nil)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置验证失败", err)
	}

	return &CloudMailAdapter{
		BaseWebAPIAdapter: webapi.NewBaseWebAPIAdapter(model.WebAPIServiceTypeCloudMail),
		config:            config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: logger.NewWithModule("CloudMail"),
	}, nil
}

// Connect 连接到 Cloud Mail 服务
func (a *CloudMailAdapter) Connect(ctx context.Context) error {
	a.log.Info("连接到 Cloud Mail: base_url=%s, accounts=%d", a.config.BaseURL, len(a.config.Accounts))

	// 测试连接
	if err := a.TestConnection(ctx); err != nil {
		return err
	}

	a.SetConnected(true)
	return nil
}

// Disconnect 断开连接
func (a *CloudMailAdapter) Disconnect() error {
	a.SetConnected(false)
	return nil
}

// TestConnection 测试连接
func (a *CloudMailAdapter) TestConnection(ctx context.Context) error {
	// 测试 API 连接
	url := a.config.BaseURL + "/api/mails?limit=1"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.config.JWTToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "认证失败", resp.StatusCode, false, nil)
	}

	if resp.StatusCode != http.StatusOK {
		return webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
	}

	return nil
}

// FetchEmails 拉取所有账户的邮件
func (a *CloudMailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	allEmails := make([]*adapter.Email, 0)

	// 遍历所有账户拉取邮件
	for _, account := range a.config.Accounts {
		emails, err := a.fetchEmailsForAccount(ctx, account.Email, since, limit)
		if err != nil {
			a.log.Warn("拉取账户邮件失败: account=%s, err=%v", account.Email, err)
			continue
		}
		allEmails = append(allEmails, emails...)
	}

	a.log.Info("Cloud Mail 拉取完成: total=%d, accounts=%d", len(allEmails), len(a.config.Accounts))
	return allEmails, nil
}

// fetchEmailsForAccount 拉取单个账户的邮件
func (a *CloudMailAdapter) fetchEmailsForAccount(ctx context.Context, accountEmail string, since time.Time, limit int) ([]*adapter.Email, error) {
	a.log.Debug("拉取账户邮件: account=%s, since=%v, limit=%d", accountEmail, since, limit)

	// 构建请求 URL
	url := fmt.Sprintf("%s/api/mails?account=%s", a.config.BaseURL, accountEmail)
	if limit > 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置认证头
	req.Header.Set("Authorization", "Bearer "+a.config.JWTToken)

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, webapi.ErrAuthFailed
	}
	if resp.StatusCode != http.StatusOK {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
	}

	// 解析 JSON
	var response MailListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	// 转换邮件
	emails := make([]*adapter.Email, 0, len(response.Data))
	for _, item := range response.Data {
		email := a.convertToEmail(item, accountEmail)

		// 过滤时间
		if !since.IsZero() && email.ReceivedAt.Before(since) {
			continue
		}

		emails = append(emails, email)
	}

	a.log.Debug("账户邮件拉取完成: account=%s, count=%d", accountEmail, len(emails))
	return emails, nil
}

// convertToEmail 转换为标准邮件格式
func (a *CloudMailAdapter) convertToEmail(item MailItem, targetAddress string) *adapter.Email {
	email := &adapter.Email{
		ProviderID:  item.ID,
		MessageID:   item.MessageID,
		Subject:     item.Subject,
		FromAddress: item.From,
		FromName:    item.FromName,
		ToAddresses: []string{targetAddress},
		TextBody:    item.TextBody,
		HTMLBody:    item.HTMLBody,
		Snippet:     item.Snippet,
	}

	// 解析时间
	if item.ReceivedAt != "" {
		if t, err := time.Parse(time.RFC3339, item.ReceivedAt); err == nil {
			email.ReceivedAt = t
		}
	}
	if item.SentAt != "" {
		if t, err := time.Parse(time.RFC3339, item.SentAt); err == nil {
			email.SentAt = t
		}
	}

	// 如果没有发送时间，使用接收时间
	if email.SentAt.IsZero() {
		email.SentAt = email.ReceivedAt
	}

	// 生成摘要
	if email.Snippet == "" {
		email.Snippet = webapi.GenerateSnippet(email, 200)
	}

	// 附件信息
	email.HasAttachments = item.HasAttachments
	email.AttachmentsCount = item.AttachmentsCount

	return email
}

// FetchEmailDetail 获取邮件详情
func (a *CloudMailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	// 构建请求 URL
	url := fmt.Sprintf("%s/api/mails/%s", a.config.BaseURL, providerID)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.config.JWTToken)

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, webapi.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
	}

	var item MailItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	// 从 item 中获取目标地址
	targetAddr := ""
	if len(item.ToAddresses) > 0 {
		targetAddr = item.ToAddresses[0]
	}

	return a.convertToEmail(item, targetAddr), nil
}

// GetProviderType 获取提供商类型
func (a *CloudMailAdapter) GetProviderType() string {
	return model.WebAPIServiceTypeCloudMail
}

// GetProtocol 获取协议类型
func (a *CloudMailAdapter) GetProtocol() string {
	return "webapi"
}

// GetConfig 获取配置
func (a *CloudMailAdapter) GetConfig() *model.CloudMailAuthData {
	return a.config
}

// ============================================
// 响应数据结构
// ============================================

// MailItem 邮件项
type MailItem struct {
	ID               string   `json:"id"`
	MessageID        string   `json:"message_id,omitempty"`
	Subject          string   `json:"subject"`
	From             string   `json:"from"`
	FromName         string   `json:"from_name,omitempty"`
	ToAddresses      []string `json:"to,omitempty"`
	TextBody         string   `json:"text,omitempty"`
	HTMLBody         string   `json:"html,omitempty"`
	Snippet          string   `json:"snippet,omitempty"`
	SentAt           string   `json:"sent_at,omitempty"`
	ReceivedAt       string   `json:"received_at,omitempty"`
	HasAttachments   bool     `json:"has_attachments"`
	AttachmentsCount int      `json:"attachments_count"`
}

// MailListResponse 邮件列表响应
type MailListResponse struct {
	Data       []MailItem `json:"data"`
	Total      int        `json:"total,omitempty"`
	Page       int        `json:"page,omitempty"`
	PageSize   int        `json:"page_size,omitempty"`
	HasMore    bool       `json:"has_more,omitempty"`
	NextCursor string     `json:"next_cursor,omitempty"`
}
