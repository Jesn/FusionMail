package cloudflare

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

// init 注册 Cloudflare Temp Email 适配器
func init() {
	// 注册适配器创建函数
	webapi.RegisterAdapter(model.WebAPIServiceTypeCloudflareTempEmail, func(authDataJSON string) (webapi.WebAPIProvider, error) {
		var config model.CloudflareTempEmailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, webapi.WrapError(webapi.ErrCodeConfigError, "解析 Cloudflare Temp Email 配置失败", err)
		}
		return NewCloudflareTempEmailAdapter(&config)
	})

	// 注册服务模板
	webapi.RegisterServiceTemplate(&webapi.ServiceTemplate{
		ServiceType: model.WebAPIServiceTypeCloudflareTempEmail,
		Name:        "Cloudflare Temp Email",
		Description: "Cloudflare Workers 临时邮箱服务，支持 Single 模式（单邮箱）和 Admin 模式（域名管理）",
		AccessModes: []string{model.WebAPIAccessModeSingle, model.WebAPIAccessModeAdmin},
		AuthFields: []webapi.AuthField{
			{Name: "base_url", Label: "API 地址", Type: "text", Required: true, Placeholder: "https://temp-email.example.com"},
			{Name: "access_mode", Label: "访问模式", Type: "select", Required: true, HelpText: "Single: 单邮箱模式；Admin: 域名管理模式"},
			{Name: "jwt_token", Label: "JWT Token", Type: "password", Required: false, HelpText: "Single 模式必填"},
			{Name: "email", Label: "邮箱地址", Type: "text", Required: false, HelpText: "Single 模式必填"},
			{Name: "admin_password", Label: "管理员密码", Type: "password", Required: false, HelpText: "Admin 模式必填"},
			{Name: "domain", Label: "域名", Type: "text", Required: false, HelpText: "Admin 模式可选"},
		},
	})
}

// CloudflareTempEmailAdapter Cloudflare Temp Email 适配器
// 支持 Single 模式（单邮箱 JWT）和 Admin 模式（域名管理）
type CloudflareTempEmailAdapter struct {
	*webapi.BaseWebAPIAdapter

	// 配置
	config *model.CloudflareTempEmailAuthData

	// HTTP 客户端
	httpClient *http.Client

	// 日志
	log *logger.Logger

	// RFC822 解析器
	parser *webapi.RFC822Parser
}

// NewCloudflareTempEmailAdapter 创建 Cloudflare Temp Email 适配器
func NewCloudflareTempEmailAdapter(config *model.CloudflareTempEmailAuthData) (*CloudflareTempEmailAdapter, error) {
	if config == nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置不能为空", nil)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置验证失败", err)
	}

	return &CloudflareTempEmailAdapter{
		BaseWebAPIAdapter: webapi.NewBaseWebAPIAdapter(model.WebAPIServiceTypeCloudflareTempEmail),
		config:            config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log:    logger.NewWithModule("CloudflareTempEmail"),
		parser: webapi.NewRFC822Parser(),
	}, nil
}

// Connect 连接到 Cloudflare Temp Email 服务
func (a *CloudflareTempEmailAdapter) Connect(ctx context.Context) error {
	a.log.Info("连接到 Cloudflare Temp Email: base_url=%s, mode=%s", a.config.BaseURL, a.config.AccessMode)

	// 测试连接
	if err := a.TestConnection(ctx); err != nil {
		return err
	}

	a.SetConnected(true)
	return nil
}

// Disconnect 断开连接
func (a *CloudflareTempEmailAdapter) Disconnect() error {
	a.SetConnected(false)
	return nil
}

// TestConnection 测试连接
func (a *CloudflareTempEmailAdapter) TestConnection(ctx context.Context) error {
	// 根据模式选择测试端点
	var endpoint string
	var headers map[string]string

	if a.config.IsSingleMode() {
		endpoint = "/api/mails"
		headers = map[string]string{
			"Authorization": "Bearer " + a.config.JWTToken,
		}
	} else {
		endpoint = "/admin/mails"
		headers = map[string]string{
			"x-admin-auth": a.config.AdminPassword,
		}
	}

	// 发送测试请求（只获取少量数据）
	url := a.config.BaseURL + endpoint + "?limit=1"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

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

// FetchEmails 拉取邮件列表
func (a *CloudflareTempEmailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	// 根据模式选择拉取方法
	if a.config.IsSingleMode() {
		return a.fetchEmailsSingle(ctx, since, limit)
	}
	return a.fetchEmailsAdmin(ctx, since, limit)
}

// fetchEmailsSingle Single 模式拉取邮件
func (a *CloudflareTempEmailAdapter) fetchEmailsSingle(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	a.log.Debug("Single 模式拉取邮件: since=%v, limit=%d", since, limit)

	// 构建请求 URL
	url := a.config.BaseURL + "/api/mails"
	if limit > 0 {
		url += fmt.Sprintf("?limit=%d", limit)
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
	var response SingleModeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	// 转换邮件
	emails := make([]*adapter.Email, 0, len(response.Results))
	for _, item := range response.Results {
		email, err := a.parseEmailItem(item, a.config.Email)
		if err != nil {
			a.log.Warn("解析邮件失败: id=%s, err=%v", item.ID, err)
			continue
		}

		// 过滤时间
		if !since.IsZero() && email.ReceivedAt.Before(since) {
			continue
		}

		emails = append(emails, email)
	}

	a.log.Info("Single 模式拉取完成: count=%d", len(emails))
	return emails, nil
}

// fetchEmailsAdmin Admin 模式拉取邮件
func (a *CloudflareTempEmailAdapter) fetchEmailsAdmin(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	a.log.Debug("Admin 模式拉取邮件: since=%v, limit=%d", since, limit)

	// 构建请求 URL
	url := a.config.BaseURL + "/admin/mails"
	if limit > 0 {
		url += fmt.Sprintf("?limit=%d", limit)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置认证头
	req.Header.Set("x-admin-auth", a.config.AdminPassword)

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
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
	var response AdminModeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	// 转换邮件
	emails := make([]*adapter.Email, 0, len(response.Results))
	for _, item := range response.Results {
		// Admin 模式从响应中获取目标地址
		targetAddr := item.Address
		if targetAddr == "" && len(item.ToAddresses) > 0 {
			targetAddr = item.ToAddresses[0]
		}

		email, err := a.parseEmailItem(item.EmailItem, targetAddr)
		if err != nil {
			a.log.Warn("解析邮件失败: id=%s, err=%v", item.ID, err)
			continue
		}

		// 过滤时间
		if !since.IsZero() && email.ReceivedAt.Before(since) {
			continue
		}

		emails = append(emails, email)
	}

	a.log.Info("Admin 模式拉取完成: count=%d", len(emails))
	return emails, nil
}

// parseEmailItem 解析单封邮件
func (a *CloudflareTempEmailAdapter) parseEmailItem(item EmailItem, targetAddress string) (*adapter.Email, error) {
	// 如果有 RFC822 原始内容，使用解析器解析
	if item.Raw != "" {
		email, err := a.parser.Parse(item.Raw)
		if err != nil {
			a.log.Warn("RFC822 解析失败，使用字段解析: %v", err)
		} else {
			// 设置 ProviderID
			email.ProviderID = item.ID
			// 确保 ToAddresses 包含目标地址
			if targetAddress != "" && len(email.ToAddresses) == 0 {
				email.ToAddresses = []string{targetAddress}
			}
			return email, nil
		}
	}

	// 使用字段直接构建
	email := &adapter.Email{
		ProviderID:  item.ID,
		MessageID:   item.MessageID,
		Subject:     item.Subject,
		FromAddress: item.From,
		FromName:    item.FromName,
		ToAddresses: item.ToAddresses,
		TextBody:    item.TextBody,
		HTMLBody:    item.HTMLBody,
		Snippet:     item.Snippet,
	}

	// 解析时间
	if item.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, item.CreatedAt); err == nil {
			email.ReceivedAt = t
			email.SentAt = t
		}
	}

	// 确保有目标地址
	if targetAddress != "" {
		if len(email.ToAddresses) == 0 {
			email.ToAddresses = []string{targetAddress}
		}
	}

	// 生成摘要
	if email.Snippet == "" {
		email.Snippet = webapi.GenerateSnippet(email, 200)
	}

	return email, nil
}

// FetchEmailDetail 获取邮件详情
func (a *CloudflareTempEmailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	// 构建请求 URL
	var url string
	var headers map[string]string

	if a.config.IsSingleMode() {
		url = fmt.Sprintf("%s/api/mails/%s", a.config.BaseURL, providerID)
		headers = map[string]string{
			"Authorization": "Bearer " + a.config.JWTToken,
		}
	} else {
		url = fmt.Sprintf("%s/admin/mails/%s", a.config.BaseURL, providerID)
		headers = map[string]string{
			"x-admin-auth": a.config.AdminPassword,
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

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

	var item EmailItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	return a.parseEmailItem(item, a.config.Email)
}

// GetProviderType 获取提供商类型
func (a *CloudflareTempEmailAdapter) GetProviderType() string {
	return model.WebAPIServiceTypeCloudflareTempEmail
}

// GetProtocol 获取协议类型
func (a *CloudflareTempEmailAdapter) GetProtocol() string {
	return "webapi"
}

// GetConfig 获取配置
func (a *CloudflareTempEmailAdapter) GetConfig() *model.CloudflareTempEmailAuthData {
	return a.config
}

// ============================================
// 响应数据结构
// ============================================

// EmailItem 邮件项
type EmailItem struct {
	ID          string   `json:"id"`
	MessageID   string   `json:"message_id,omitempty"`
	Subject     string   `json:"subject"`
	From        string   `json:"from"`
	FromName    string   `json:"from_name,omitempty"`
	ToAddresses []string `json:"to,omitempty"`
	TextBody    string   `json:"text,omitempty"`
	HTMLBody    string   `json:"html,omitempty"`
	Snippet     string   `json:"snippet,omitempty"`
	Raw         string   `json:"raw,omitempty"` // RFC822 原始内容
	CreatedAt   string   `json:"created_at,omitempty"`
}

// SingleModeResponse Single 模式响应
type SingleModeResponse struct {
	Results []EmailItem `json:"results"`
}

// AdminEmailItem Admin 模式邮件项（包含地址信息）
type AdminEmailItem struct {
	EmailItem
	Address string `json:"address"` // 目标邮箱地址
}

// AdminModeResponse Admin 模式响应
type AdminModeResponse struct {
	Results []AdminEmailItem `json:"results"`
}
