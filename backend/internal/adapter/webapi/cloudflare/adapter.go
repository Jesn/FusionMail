package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
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
			{Name: "jwt_token", Label: "JWT Token", Type: "password", Required: false, HelpText: "Single 模式：直接登录方式必填（永不过期）"},
			{Name: "user_token", Label: "User Token", Type: "password", Required: false, HelpText: "Single 模式：第三方授权登录方式必填（30天过期，支持自动刷新）"},
			{Name: "email", Label: "邮箱地址", Type: "text", Required: false, HelpText: "Single 模式可选，用于显示"},
			{Name: "admin_password", Label: "管理员密码", Type: "password", Required: false, HelpText: "Admin 模式必填"},
			{Name: "domain", Label: "域名", Type: "text", Required: false, HelpText: "Admin 模式可选"},
		},
	})
}

// TokenUpdateCallback Token 更新回调函数类型
// 当 user_token 被刷新时调用，用于通知服务层保存新 token
type TokenUpdateCallback func(newUserToken string) error

// CloudflareTempEmailAdapter Cloudflare Temp Email 适配器
// 支持 Single 模式（单邮箱 JWT）和 Admin 模式（域名管理）
// 支持两种认证方式：
// 1. jwt_token - 直接 JWT Token 登录（永不过期）
// 2. user_token - 第三方授权登录（30 天过期，支持自动刷新）
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

	// Token 更新回调
	tokenUpdateCallback TokenUpdateCallback
	tokenUpdateMu       sync.Mutex
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

	// 检查并刷新 user_token（如果需要）
	if err := a.checkAndRefreshToken(ctx); err != nil {
		a.log.Warn("检查/刷新 user_token 失败: %v", err)
		// 不阻止连接，继续尝试
	}

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
		headers = make(map[string]string)

		// 根据认证方式选择不同的 API 端点和请求头
		// user_token 模式：使用 /user_api/mails 端点，只需要 x-user-token 头
		// jwt_token 模式：使用 /api/mails 端点，需要 Authorization 头
		if a.config.HasUserToken() {
			endpoint = "/user_api/mails"
			// user_api 只需要 x-user-token 头，不需要 Authorization
			headers["x-user-token"] = a.config.UserToken
		} else {
			endpoint = "/api/mails"
			// api 需要 Authorization 头
			if a.config.JWTToken != "" {
				headers["Authorization"] = "Bearer " + a.config.JWTToken
			}
		}
	} else {
		endpoint = "/admin/mails"
		headers = map[string]string{
			"x-admin-auth": a.config.AdminPassword,
		}
		// Admin 模式如果配置了 JWT Token，也需要发送
		if a.config.JWTToken != "" {
			headers["Authorization"] = "Bearer " + a.config.JWTToken
		}
	}

	// 发送测试请求（只获取少量数据，必须包含 offset 参数）
	url := a.config.BaseURL + endpoint + "?limit=1&offset=0"

	a.log.Info("测试连接: url=%s, 认证方式: user_token=%v, jwt_token=%v",
		url, a.config.HasUserToken(), a.config.JWTToken != "")

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

	// 读取响应体用于诊断
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "认证失败", resp.StatusCode, false, nil)
	}

	if resp.StatusCode != http.StatusOK {
		// 检查是否返回 HTML
		if len(body) > 0 && body[0] == '<' {
			preview := string(body)
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			a.log.Error("服务器返回 HTML: %s", preview)
			return webapi.NewWebAPIError(webapi.ErrCodeServerError,
				fmt.Sprintf("服务器返回 HTML 而不是 JSON，请检查 base_url 是否正确（当前: %s）", a.config.BaseURL),
				resp.StatusCode, false, nil)
		}
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

	// 根据认证方式选择不同的 API 端点
	// user_token 模式使用 /user_api/mails，jwt_token 模式使用 /api/mails
	var endpoint string
	if a.config.HasUserToken() {
		endpoint = "/user_api/mails"
	} else {
		endpoint = "/api/mails"
	}

	// 构建请求 URL（必须包含 offset 参数）
	url := a.config.BaseURL + endpoint + "?offset=0"
	if limit > 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}

	a.log.Info("请求 URL: %s, 认证方式: user_token=%v, jwt_token=%v",
		url, a.config.HasUserToken(), a.config.JWTToken != "")

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置认证头
	// user_token 模式：只需要 x-user-token 头
	// jwt_token 模式：需要 Authorization 头
	if a.config.HasUserToken() {
		// user_api 只需要 x-user-token 头，不需要 Authorization
		req.Header.Set("x-user-token", a.config.UserToken)
	} else if a.config.JWTToken != "" {
		// api 需要 Authorization 头
		req.Header.Set("Authorization", "Bearer "+a.config.JWTToken)
	}

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

	// 检查响应内容类型，帮助诊断问题
	contentType := resp.Header.Get("Content-Type")
	a.log.Debug("响应 Content-Type: %s, 响应长度: %d", contentType, len(body))

	// 如果响应以 < 开头，说明返回的是 HTML 而不是 JSON
	if len(body) > 0 && body[0] == '<' {
		// 截取前 200 字符用于日志
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		a.log.Error("服务器返回 HTML 而不是 JSON: %s", preview)
		return nil, webapi.WrapError(webapi.ErrCodeParseError,
			fmt.Sprintf("服务器返回 HTML 而不是 JSON，请检查 base_url 是否正确（当前: %s%s）", a.config.BaseURL, endpoint), nil)
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
	a.log.Debug("Admin 模式拉取邮件: since=%v, limit=%d, domains=%v", since, limit, a.config.GetDomainList())

	// 构建请求 URL（必须包含 offset 参数）
	url := a.config.BaseURL + "/admin/mails?offset=0"
	if limit > 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置认证头（Admin 模式需要同时发送 x-admin-auth 和 Authorization）
	req.Header.Set("x-admin-auth", a.config.AdminPassword)
	// 如果配置了 JWT Token，也需要发送（某些 API 实现需要双重认证）
	if a.config.JWTToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.config.JWTToken)
	}

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
	filteredCount := 0
	for _, item := range response.Results {
		// Admin 模式从响应中获取目标地址
		targetAddr := item.Address
		if targetAddr == "" && len(item.ToAddresses) > 0 {
			targetAddr = item.ToAddresses[0]
		}

		// 域名过滤：检查目标地址是否匹配配置的域名
		if a.config.HasDomainFilter() && !a.config.MatchesDomain(targetAddr) {
			filteredCount++
			continue
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

	if filteredCount > 0 {
		a.log.Info("Admin 模式域名过滤: 过滤掉 %d 封不匹配域名的邮件", filteredCount)
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
			email.ProviderID = item.ID.String()
			// 确保 ToAddresses 包含目标地址
			if targetAddress != "" && len(email.ToAddresses) == 0 {
				email.ToAddresses = []string{targetAddress}
			}
			return email, nil
		}
	}

	// 使用字段直接构建
	email := &adapter.Email{
		ProviderID:  item.ID.String(),
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

// SetTokenUpdateCallback 设置 Token 更新回调
// 当 user_token 被刷新时，会调用此回调通知服务层保存新 token
func (a *CloudflareTempEmailAdapter) SetTokenUpdateCallback(callback TokenUpdateCallback) {
	a.tokenUpdateMu.Lock()
	defer a.tokenUpdateMu.Unlock()
	a.tokenUpdateCallback = callback
}

// checkAndRefreshToken 检查并刷新 user_token
// 仅在配置了 user_token 且距离过期 ≤7 天时才刷新
func (a *CloudflareTempEmailAdapter) checkAndRefreshToken(ctx context.Context) error {
	// 只有配置了 user_token 才需要检查刷新
	if !a.config.HasUserToken() {
		return nil
	}

	// 检查是否需要刷新（距离过期 ≤7 天）
	if !a.config.NeedsTokenRefresh() {
		exp := a.config.GetUserTokenExpiry()
		if exp > 0 {
			remaining := time.Until(time.Unix(exp, 0))
			a.log.Debug("user_token 无需刷新: 剩余有效期 %v", remaining)
		}
		return nil
	}

	a.log.Info("user_token 即将过期，尝试刷新...")

	// 调用 /user_api/settings 获取新 token
	newToken, err := a.refreshUserToken(ctx)
	if err != nil {
		return fmt.Errorf("刷新 user_token 失败: %w", err)
	}

	if newToken == "" {
		a.log.Debug("服务端未返回新 token，当前 token 仍有效")
		return nil
	}

	// 更新本地配置
	oldToken := a.config.UserToken
	a.config.UserToken = newToken

	// 通知服务层保存新 token
	a.tokenUpdateMu.Lock()
	callback := a.tokenUpdateCallback
	a.tokenUpdateMu.Unlock()

	if callback != nil {
		if err := callback(newToken); err != nil {
			// 回滚本地配置
			a.config.UserToken = oldToken
			return fmt.Errorf("保存新 token 失败: %w", err)
		}
	}

	// 解析新 token 的过期时间用于日志
	newExp := model.ParseJWTExpiry(newToken)
	if newExp > 0 {
		a.log.Info("user_token 刷新成功: 新过期时间 %v", time.Unix(newExp, 0))
	} else {
		a.log.Info("user_token 刷新成功")
	}

	return nil
}

// refreshUserToken 调用 /user_api/settings 刷新 user_token
// 返回新的 token（如果服务端返回了 new_user_token）
func (a *CloudflareTempEmailAdapter) refreshUserToken(ctx context.Context) (string, error) {
	// 构建请求 URL
	url := a.config.BaseURL + "/user_api/settings"

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	// user_api 只需要 x-user-token 头，不需要 Authorization
	req.Header.Set("x-user-token", a.config.UserToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("认证失败: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("服务器返回错误: HTTP %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 JSON
	// 当 token 距离过期 ≤7 天时，响应中会包含 new_user_token 字段
	var settingsResp struct {
		NewUserToken string `json:"new_user_token"` // 新的 user_token（如果需要刷新）
	}

	if err := json.Unmarshal(body, &settingsResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	return settingsResp.NewUserToken, nil
}

// ============================================
// 响应数据结构
// ============================================

// FlexibleID 灵活的 ID 类型，支持数字和字符串
type FlexibleID string

// UnmarshalJSON 自定义 JSON 反序列化，支持数字和字符串
func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	// 尝试作为字符串解析
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleID(s)
		return nil
	}

	// 尝试作为数字解析
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexibleID(n.String())
		return nil
	}

	return fmt.Errorf("无法解析 ID: %s", string(data))
}

// String 返回字符串值
func (f FlexibleID) String() string {
	return string(f)
}

// EmailItem 邮件项
type EmailItem struct {
	ID          FlexibleID `json:"id"`
	MessageID   string     `json:"message_id,omitempty"`
	Subject     string     `json:"subject"`
	From        string     `json:"from"`
	FromName    string     `json:"from_name,omitempty"`
	ToAddresses []string   `json:"to,omitempty"`
	TextBody    string     `json:"text,omitempty"`
	HTMLBody    string     `json:"html,omitempty"`
	Snippet     string     `json:"snippet,omitempty"`
	Raw         string     `json:"raw,omitempty"` // RFC822 原始内容
	CreatedAt   string     `json:"created_at,omitempty"`
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
