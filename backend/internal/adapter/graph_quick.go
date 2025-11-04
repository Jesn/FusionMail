package adapter

// GraphQuickAdapter - Microsoft Graph API 短效适配器
//
// 重要说明：
// 本适配器的核心实现必须与 backend/micro.py 参考实现保持完全一致。
// 这是确保短效邮箱能够正确接收邮件的关键要求。
//
// 核心流程（必须与 micro.py 一致）：
// 1. Token 刷新：POST https://login.microsoftonline.com/common/oauth2/v2.0/token
//    - 参数：client_id, grant_type=refresh_token, refresh_token, scope=https://graph.microsoft.com/.default
// 2. 邮件获取：GET https://graph.microsoft.com/v1.0/me/mailFolders/inbox/messages
//    - 头部：Authorization: Bearer {access_token}
//
// 扩展功能（在核心流程之上构建）：
// - FetchEmailsWithFilter：使用过滤条件获取邮件
// - FetchEmailsWithPagination：分页获取邮件
// - FetchEmailsByFolder：按文件夹获取邮件
// - 其他辅助方法
//
// 修改核心流程时，必须确保与 micro.py 的行为完全一致。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Logger 日志接口
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// SimpleLogger 简单的日志实现
type SimpleLogger struct {
	prefix string
}

func NewSimpleLogger(prefix string) *SimpleLogger {
	return &SimpleLogger{prefix: prefix}
}

func (l *SimpleLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[INFO] %s: %s %v\n", l.prefix, msg, keysAndValues)
}

func (l *SimpleLogger) Warn(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[WARN] %s: %s %v\n", l.prefix, msg, keysAndValues)
}

func (l *SimpleLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[ERROR] %s: %s %v\n", l.prefix, msg, keysAndValues)
}

// GraphQuickAdapter Microsoft Graph API 短效适配器
// 使用简化的认证流程，直接通过 refresh_token 获取 access_token
// 适用于批量导入、测试验证等场景
type GraphQuickAdapter struct {
	config      *Config
	httpClient  *http.Client
	baseURL     string
	tokenURL    string // OAuth2 token 端点
	accessToken string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex
	logger      Logger
}

// QuickAuthConfig 短效认证配置
type QuickAuthConfig struct {
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	AuthType     string `json:"auth_type"`
}

// TokenResponse Microsoft OAuth2 token 响应
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// TokenError token 相关错误
type TokenError struct {
	Code        string
	Description string
	StatusCode  int
}

func (e *TokenError) Error() string {
	return fmt.Sprintf("token error %s: %s (status: %d)", e.Code, e.Description, e.StatusCode)
}

// NewGraphQuickAdapter 创建短效适配器实例
func NewGraphQuickAdapter(config *Config) (*GraphQuickAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证短效认证所需的参数
	if config.Credentials.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token is required for quick adapter")
	}

	if config.Credentials.ClientID == "" {
		return nil, fmt.Errorf("client ID is required for quick adapter")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// 默认 URL，可以通过配置覆盖（用于测试）
	baseURL := "https://graph.microsoft.com/v1.0"
	tokenURL := "https://login.microsoftonline.com/common/oauth2/v2.0/token"

	// 支持测试时自定义 URL
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}

	return &GraphQuickAdapter{
		config:     config,
		baseURL:    baseURL,
		tokenURL:   tokenURL,
		httpClient: &http.Client{Timeout: config.Timeout},
		logger:     NewSimpleLogger("GraphQuickAdapter"),
	}, nil
}

// Connect 连接到 Microsoft Graph API
func (a *GraphQuickAdapter) Connect(ctx context.Context) error {
	// 获取 access token
	if err := a.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	return nil
}

// Disconnect 断开连接
func (a *GraphQuickAdapter) Disconnect() error {
	// 清理敏感信息
	a.accessToken = ""
	a.tokenExpiry = time.Time{}
	a.httpClient = nil
	return nil
}

// refreshAccessToken 使用 refresh token 获取新的 access token
// 实现线程安全的 token 刷新机制，包含重试逻辑
func (a *GraphQuickAdapter) refreshAccessToken(ctx context.Context) error {
	// 使用写锁确保线程安全
	a.tokenMutex.Lock()
	defer a.tokenMutex.Unlock()

	// 双重检查：可能在等待锁的过程中，其他协程已经刷新了 token
	if a.accessToken != "" && time.Now().Add(5*time.Minute).Before(a.tokenExpiry) {
		return nil
	}

	a.logger.Info("开始刷新 access token", "email", a.config.Email)

	var lastErr error
	maxRetries := 3
	baseDelay := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// 指数退避重试
			delay := time.Duration(attempt-1) * baseDelay
			a.logger.Info("重试刷新 token", "attempt", attempt, "delay", delay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := a.doRefreshToken(ctx)
		if err == nil {
			a.logger.Info("成功刷新 access token",
				"email", a.config.Email,
				"expires_at", a.tokenExpiry.Format(time.RFC3339),
				"attempt", attempt)
			return nil
		}

		lastErr = err
		a.logger.Warn("刷新 token 失败",
			"email", a.config.Email,
			"attempt", attempt,
			"error", err)

		// 检查是否是不可重试的错误
		if a.isNonRetryableError(err) {
			a.logger.Error("遇到不可重试的错误，停止重试", "error", err)
			break
		}
	}

	return fmt.Errorf("刷新 token 失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// doRefreshToken 执行实际的 token 刷新请求
func (a *GraphQuickAdapter) doRefreshToken(ctx context.Context) error {
	// 构建请求参数
	data := url.Values{}
	data.Set("client_id", a.config.Credentials.ClientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", a.config.Credentials.RefreshToken)
	data.Set("scope", "https://graph.microsoft.com/.default")

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST",
		a.tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "FusionMail/1.0")

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误响应
		var errorResp TokenResponse
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
			return &TokenError{
				Code:        errorResp.Error,
				Description: errorResp.ErrorDesc,
				StatusCode:  resp.StatusCode,
			}
		}
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析成功响应
	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResponse.Error != "" {
		return &TokenError{
			Code:        tokenResponse.Error,
			Description: tokenResponse.ErrorDesc,
			StatusCode:  resp.StatusCode,
		}
	}

	if tokenResponse.AccessToken == "" {
		return fmt.Errorf("no access token in response")
	}

	// 保存 token 信息（已在锁内，无需额外加锁）
	a.accessToken = tokenResponse.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)

	return nil
}

// isNonRetryableError 判断是否为不可重试的错误
func (a *GraphQuickAdapter) isNonRetryableError(err error) bool {
	var tokenErr *TokenError
	if errors.As(err, &tokenErr) {
		// 这些错误码表示配置问题，不应该重试
		nonRetryableCodes := []string{
			"invalid_client",
			"invalid_grant",
			"unsupported_grant_type",
			"invalid_scope",
		}

		for _, code := range nonRetryableCodes {
			if tokenErr.Code == code {
				return true
			}
		}
	}
	return false
}

// TestConnection 测试连接
// TestConnection 测试连接
func (a *GraphQuickAdapter) TestConnection(ctx context.Context) error {
	a.logger.Info("开始测试连接", "email", a.config.Email)

	// 创建带超时的上下文
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 执行连接测试
	result, err := a.performConnectionTest(testCtx)
	if err != nil {
		a.logger.Error("连接测试失败", "email", a.config.Email, "error", err)
		return err
	}

	a.logger.Info("连接测试成功",
		"email", a.config.Email,
		"user_id", result.UserID,
		"display_name", result.DisplayName,
		"response_time_ms", result.ResponseTimeMs)

	return nil
}

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	UserID         string `json:"id"`
	DisplayName    string `json:"displayName"`
	Mail           string `json:"mail"`
	UserPrincipal  string `json:"userPrincipalName"`
	ResponseTimeMs int64  `json:"response_time_ms"`
}

// performConnectionTest 执行实际的连接测试
func (a *GraphQuickAdapter) performConnectionTest(ctx context.Context) (*ConnectionTestResult, error) {
	startTime := time.Now()

	// 确保有有效的 access token
	if err := a.ensureValidToken(ctx); err != nil {
		return nil, &ConnectionTestError{
			Type:    "authentication",
			Message: "无法获取有效的访问令牌",
			Details: err.Error(),
		}
	}

	// 测试获取用户信息
	requestURL := fmt.Sprintf("%s/me", a.baseURL)

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, &ConnectionTestError{
			Type:    "network",
			Message: "网络请求失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	// 计算响应时间
	responseTime := time.Since(startTime).Milliseconds()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, a.parseGraphAPIError(resp.StatusCode, body)
	}

	// 解析用户信息
	var userInfo ConnectionTestResult
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, &ConnectionTestError{
			Type:    "parsing",
			Message: "无法解析用户信息响应",
			Details: err.Error(),
		}
	}

	userInfo.ResponseTimeMs = responseTime

	// 验证关键字段
	if userInfo.UserID == "" {
		return nil, &ConnectionTestError{
			Type:    "validation",
			Message: "用户信息不完整",
			Details: "用户 ID 为空",
		}
	}

	return &userInfo, nil
}

// ConnectionTestError 连接测试错误
type ConnectionTestError struct {
	Type    string `json:"type"`             // authentication, network, parsing, validation, api
	Message string `json:"message"`          // 用户友好的错误消息
	Details string `json:"details"`          // 详细的技术错误信息
	Code    string `json:"code,omitempty"`   // API 错误码
	Status  int    `json:"status,omitempty"` // HTTP 状态码
}

func (e *ConnectionTestError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s (code: %s): %s", e.Type, e.Message, e.Code, e.Details)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Message, e.Details)
}

// parseGraphAPIError 解析 Graph API 错误响应
func (a *GraphQuickAdapter) parseGraphAPIError(statusCode int, body []byte) *ConnectionTestError {
	// 尝试解析 Graph API 错误格式
	var graphError struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if json.Unmarshal(body, &graphError) == nil && graphError.Error.Code != "" {
		return &ConnectionTestError{
			Type:    "api",
			Message: a.getErrorMessage(graphError.Error.Code, statusCode),
			Details: graphError.Error.Message,
			Code:    graphError.Error.Code,
			Status:  statusCode,
		}
	}

	// 如果无法解析，返回通用错误
	return &ConnectionTestError{
		Type:    "api",
		Message: a.getErrorMessage("", statusCode),
		Details: string(body),
		Status:  statusCode,
	}
}

// getErrorMessage 根据错误码和状态码返回用户友好的错误消息
func (a *GraphQuickAdapter) getErrorMessage(errorCode string, statusCode int) string {
	switch errorCode {
	case "InvalidAuthenticationToken":
		return "访问令牌无效或已过期"
	case "Forbidden":
		return "没有权限访问此资源"
	case "ThrottledRequest":
		return "请求过于频繁，请稍后重试"
	case "ServiceNotAvailable":
		return "Microsoft Graph 服务暂时不可用"
	case "TooManyRequests":
		return "请求次数超过限制"
	default:
		switch statusCode {
		case 401:
			return "身份验证失败"
		case 403:
			return "访问被拒绝"
		case 404:
			return "资源不存在"
		case 429:
			return "请求过于频繁"
		case 500:
			return "服务器内部错误"
		case 502:
			return "网关错误"
		case 503:
			return "服务不可用"
		case 504:
			return "网关超时"
		default:
			return fmt.Sprintf("请求失败 (HTTP %d)", statusCode)
		}
	}
}

// TestConnectionWithDetails 测试连接并返回详细信息
func (a *GraphQuickAdapter) TestConnectionWithDetails(ctx context.Context) (*ConnectionTestResult, error) {
	a.logger.Info("开始详细连接测试", "email", a.config.Email)

	// 创建带超时的上下文
	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 执行连接测试
	result, err := a.performConnectionTest(testCtx)
	if err != nil {
		a.logger.Error("详细连接测试失败", "email", a.config.Email, "error", err)
		return nil, err
	}

	a.logger.Info("详细连接测试成功",
		"email", a.config.Email,
		"user_id", result.UserID,
		"display_name", result.DisplayName,
		"mail", result.Mail,
		"response_time_ms", result.ResponseTimeMs)

	return result, nil
}

// ValidateCredentials 验证凭据有效性
func (a *GraphQuickAdapter) ValidateCredentials(ctx context.Context) error {
	a.logger.Info("开始验证凭据", "email", a.config.Email)

	// 检查必要的凭据字段
	if a.config.Credentials == nil {
		return &ConnectionTestError{
			Type:    "validation",
			Message: "凭据配置为空",
			Details: "Credentials 字段不能为 nil",
		}
	}

	if a.config.Credentials.ClientID == "" {
		return &ConnectionTestError{
			Type:    "validation",
			Message: "客户端 ID 不能为空",
			Details: "ClientID 字段是必需的",
		}
	}

	if a.config.Credentials.RefreshToken == "" {
		return &ConnectionTestError{
			Type:    "validation",
			Message: "刷新令牌不能为空",
			Details: "RefreshToken 字段是必需的",
		}
	}

	// 尝试刷新 token 来验证凭据
	if err := a.refreshAccessToken(ctx); err != nil {
		var tokenErr *TokenError
		if errors.As(err, &tokenErr) {
			return &ConnectionTestError{
				Type:    "authentication",
				Message: "凭据验证失败",
				Details: tokenErr.Description,
				Code:    tokenErr.Code,
				Status:  tokenErr.StatusCode,
			}
		}
		return &ConnectionTestError{
			Type:    "authentication",
			Message: "凭据验证失败",
			Details: err.Error(),
		}
	}

	a.logger.Info("凭据验证成功", "email", a.config.Email)
	return nil
}

// ensureValidToken 确保有有效的 access token
func (a *GraphQuickAdapter) ensureValidToken(ctx context.Context) error {
	// 使用读锁检查 token 状态
	a.tokenMutex.RLock()
	hasToken := a.accessToken != ""
	isExpired := time.Now().Add(5 * time.Minute).After(a.tokenExpiry)
	a.tokenMutex.RUnlock()

	// 如果 token 不存在或即将过期，则刷新
	if !hasToken || isExpired {
		return a.refreshAccessToken(ctx)
	}

	return nil
}

// IsTokenValid 检查当前 token 是否有效
func (a *GraphQuickAdapter) IsTokenValid() bool {
	a.tokenMutex.RLock()
	defer a.tokenMutex.RUnlock()

	return a.accessToken != "" && time.Now().Before(a.tokenExpiry)
}

// GetTokenExpiry 获取 token 过期时间
func (a *GraphQuickAdapter) GetTokenExpiry() time.Time {
	a.tokenMutex.RLock()
	defer a.tokenMutex.RUnlock()

	return a.tokenExpiry
}

// GetTokenInfo 获取 token 信息（用于监控和调试）
func (a *GraphQuickAdapter) GetTokenInfo() map[string]interface{} {
	a.tokenMutex.RLock()
	defer a.tokenMutex.RUnlock()

	info := map[string]interface{}{
		"has_token":  a.accessToken != "",
		"expires_at": a.tokenExpiry.Format(time.RFC3339),
		"is_valid":   a.accessToken != "" && time.Now().Before(a.tokenExpiry),
		"expires_in": int(time.Until(a.tokenExpiry).Seconds()),
	}

	if a.accessToken != "" {
		// 只显示 token 的前几个字符用于调试
		tokenPreview := a.accessToken
		if len(tokenPreview) > 10 {
			tokenPreview = tokenPreview[:10] + "..."
		}
		info["token_preview"] = tokenPreview
	}

	return info
}

// FetchEmails 拉取邮件列表
// 核心实现：必须与 backend/micro.py 的 print_inbox 函数保持一致
// 使用 /me/mailFolders/inbox/messages 端点获取收件箱邮件
func (a *GraphQuickAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	a.logger.Info("开始获取邮件列表",
		"email", a.config.Email,
		"since", since.Format(time.RFC3339),
		"limit", limit)

	// 构建查询参数
	params := url.Values{}
	params.Set("$orderby", "receivedDateTime DESC")

	// 设置获取数量限制
	fetchLimit := 100 // 默认每次最多获取 100 封
	if limit > 0 && limit < 100 {
		fetchLimit = limit
	}
	params.Set("$top", fmt.Sprintf("%d", fetchLimit))

	// 添加时间过滤
	var filters []string
	if !since.IsZero() {
		filters = append(filters, fmt.Sprintf("receivedDateTime ge %s", since.Format(time.RFC3339)))
	}

	// 可以添加其他过滤条件
	if len(filters) > 0 {
		params.Set("$filter", strings.Join(filters, " and "))
	}

	// 选择需要的字段以优化性能
	params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

	// 构建请求 URL
	// 重要：必须使用 /me/mailFolders/inbox/messages 端点（与 micro.py 一致）
	// 不能使用 /me/messages 端点
	requestURL := fmt.Sprintf("%s/me/mailFolders/inbox/messages?%s", a.baseURL, params.Encode())

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("获取邮件列表失败",
			"email", a.config.Email,
			"status", resp.StatusCode,
			"response", string(body))
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

	a.logger.Info("成功获取邮件列表",
		"email", a.config.Email,
		"count", len(emails),
		"has_next", messageList.NextLink != "")

	return emails, nil
}

// FetchEmailDetail 获取邮件详情
// FetchEmailDetail 获取邮件详情
func (a *GraphQuickAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	a.logger.Info("开始获取邮件详情",
		"email", a.config.Email,
		"provider_id", providerID)

	// 验证 providerID
	if providerID == "" {
		return nil, fmt.Errorf("provider ID cannot be empty")
	}

	// 构建请求 URL，包含完整的字段选择
	params := url.Values{}
	params.Set("$select", "id,subject,body,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification,inReplyTo,references")

	requestURL := fmt.Sprintf("%s/me/messages/%s?%s", a.baseURL, providerID, params.Encode())

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("message not found: %s", providerID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("获取邮件详情失败",
			"email", a.config.Email,
			"provider_id", providerID,
			"status", resp.StatusCode,
			"response", string(body))
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
		a.logger.Info("获取邮件附件",
			"email", a.config.Email,
			"provider_id", providerID)

		attachments, err := a.fetchAttachments(ctx, providerID)
		if err != nil {
			a.logger.Warn("获取附件失败",
				"email", a.config.Email,
				"provider_id", providerID,
				"error", err)
			// 不因为附件获取失败而失败整个请求
		} else {
			email.Attachments = attachments
			email.AttachmentsCount = len(attachments)
		}
	}

	a.logger.Info("成功获取邮件详情",
		"email", a.config.Email,
		"provider_id", providerID,
		"subject", email.Subject,
		"has_attachments", email.HasAttachments,
		"attachments_count", email.AttachmentsCount)

	return email, nil
}

// fetchAttachments 获取附件列表
// fetchAttachments 获取附件列表
func (a *GraphQuickAdapter) fetchAttachments(ctx context.Context, messageID string) ([]Attachment, error) {
	requestURL := fmt.Sprintf("%s/me/messages/%s/attachments", a.baseURL, messageID)

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attachments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
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
func (a *GraphQuickAdapter) convertGraphMessageToEmail(msg *GraphMessage) *Email {
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
func (a *GraphQuickAdapter) GetProviderType() string {
	return "outlook"
}

// GetProtocol 获取协议类型
func (a *GraphQuickAdapter) GetProtocol() string {
	return "graph_quick"
}

// RefreshTokenIfNeeded 如果需要则刷新 token（主动刷新）
func (a *GraphQuickAdapter) RefreshTokenIfNeeded(ctx context.Context) error {
	return a.ensureValidToken(ctx)
}

// ForceRefreshToken 强制刷新 token
func (a *GraphQuickAdapter) ForceRefreshToken(ctx context.Context) error {
	a.logger.Info("强制刷新 access token", "email", a.config.Email)
	return a.refreshAccessToken(ctx)
}

// ClearToken 清除当前 token（用于登出或错误恢复）
func (a *GraphQuickAdapter) ClearToken() {
	a.tokenMutex.Lock()
	defer a.tokenMutex.Unlock()

	a.accessToken = ""
	a.tokenExpiry = time.Time{}
	a.logger.Info("已清除 access token", "email", a.config.Email)
}

// handleTokenError 处理 token 相关错误
func (a *GraphQuickAdapter) handleTokenError(ctx context.Context, err error) error {
	// 如果是 401 未授权错误，尝试刷新 token
	if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "Unauthorized") {
		a.logger.Warn("检测到 401 错误，尝试刷新 token", "email", a.config.Email)

		if refreshErr := a.refreshAccessToken(ctx); refreshErr != nil {
			return fmt.Errorf("token refresh failed after 401 error: %w", refreshErr)
		}

		a.logger.Info("token 刷新成功，可以重试请求", "email", a.config.Email)
		return &TokenRefreshedError{OriginalError: err}
	}

	return err
}

// TokenRefreshedError 表示 token 已刷新，可以重试请求
type TokenRefreshedError struct {
	OriginalError error
}

func (e *TokenRefreshedError) Error() string {
	return fmt.Sprintf("token refreshed, retry needed: %v", e.OriginalError)
}

func (e *TokenRefreshedError) Unwrap() error {
	return e.OriginalError
}

// makeAuthenticatedRequest 发送带认证的请求，自动处理 token 刷新
func (a *GraphQuickAdapter) makeAuthenticatedRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	maxRetries := 2

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 确保有有效的 token
		if err := a.ensureValidToken(ctx); err != nil {
			return nil, fmt.Errorf("failed to ensure valid token: %w", err)
		}

		// 创建请求
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// 添加认证头
		a.tokenMutex.RLock()
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.accessToken))
		a.tokenMutex.RUnlock()

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "FusionMail/1.0")

		// 发送请求
		resp, err := a.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		// 检查是否需要刷新 token
		if resp.StatusCode == http.StatusUnauthorized && attempt < maxRetries {
			resp.Body.Close()
			a.logger.Warn("收到 401 响应，尝试刷新 token", "attempt", attempt, "email", a.config.Email)

			// 强制刷新 token
			if refreshErr := a.ForceRefreshToken(ctx); refreshErr != nil {
				return nil, fmt.Errorf("failed to refresh token after 401: %w", refreshErr)
			}

			continue // 重试请求
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// EmailFilter 邮件过滤条件
type EmailFilter struct {
	Since      time.Time // 开始时间
	Until      time.Time // 结束时间
	IsRead     *bool     // 已读状态
	HasAttach  *bool     // 是否有附件
	FromEmail  string    // 发件人邮箱
	Subject    string    // 主题关键词
	Folder     string    // 文件夹名称
	Categories []string  // 分类标签
}

// FetchEmailsWithFilter 使用过滤条件获取邮件
func (a *GraphQuickAdapter) FetchEmailsWithFilter(ctx context.Context, filter *EmailFilter, limit int) ([]*Email, error) {
	a.logger.Info("开始使用过滤条件获取邮件",
		"email", a.config.Email,
		"filter", fmt.Sprintf("%+v", filter),
		"limit", limit)

	// 构建查询参数
	params := url.Values{}
	params.Set("$orderby", "receivedDateTime DESC")

	// 设置获取数量限制
	fetchLimit := 100
	if limit > 0 && limit < 100 {
		fetchLimit = limit
	}
	params.Set("$top", fmt.Sprintf("%d", fetchLimit))

	// 构建过滤条件
	var filters []string

	if filter != nil {
		// 时间范围过滤
		if !filter.Since.IsZero() {
			filters = append(filters, fmt.Sprintf("receivedDateTime ge %s", filter.Since.Format(time.RFC3339)))
		}
		if !filter.Until.IsZero() {
			filters = append(filters, fmt.Sprintf("receivedDateTime le %s", filter.Until.Format(time.RFC3339)))
		}

		// 已读状态过滤
		if filter.IsRead != nil {
			filters = append(filters, fmt.Sprintf("isRead eq %t", *filter.IsRead))
		}

		// 附件过滤
		if filter.HasAttach != nil {
			filters = append(filters, fmt.Sprintf("hasAttachments eq %t", *filter.HasAttach))
		}

		// 发件人过滤
		if filter.FromEmail != "" {
			filters = append(filters, fmt.Sprintf("from/emailAddress/address eq '%s'", filter.FromEmail))
		}

		// 主题关键词过滤
		if filter.Subject != "" {
			filters = append(filters, fmt.Sprintf("contains(subject,'%s')", filter.Subject))
		}

		// 分类过滤
		if len(filter.Categories) > 0 {
			categoryFilters := make([]string, len(filter.Categories))
			for i, cat := range filter.Categories {
				categoryFilters[i] = fmt.Sprintf("categories/any(c:c eq '%s')", cat)
			}
			filters = append(filters, "("+strings.Join(categoryFilters, " or ")+")")
		}
	}

	if len(filters) > 0 {
		params.Set("$filter", strings.Join(filters, " and "))
	}

	// 选择需要的字段
	params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/me/messages?%s", a.baseURL, params.Encode())

	// 发送请求
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch filtered messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("获取过滤邮件失败",
			"email", a.config.Email,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode filtered response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		emails = append(emails, email)
	}

	a.logger.Info("成功获取过滤邮件",
		"email", a.config.Email,
		"count", len(emails))

	return emails, nil
}

// FetchEmailsWithPagination 分页获取邮件
func (a *GraphQuickAdapter) FetchEmailsWithPagination(ctx context.Context, pageSize int, nextPageToken string) (*EmailPage, error) {
	a.logger.Info("开始分页获取邮件",
		"email", a.config.Email,
		"page_size", pageSize,
		"next_token", nextPageToken != "")

	var requestURL string

	if nextPageToken != "" {
		// 使用下一页链接
		requestURL = nextPageToken
	} else {
		// 构建首页请求
		params := url.Values{}
		params.Set("$orderby", "receivedDateTime DESC")

		if pageSize <= 0 || pageSize > 100 {
			pageSize = 50 // 默认页大小
		}
		params.Set("$top", fmt.Sprintf("%d", pageSize))

		// 选择需要的字段
		params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

		requestURL = fmt.Sprintf("%s/me/messages?%s", a.baseURL, params.Encode())
	}

	// 发送请求
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch paginated messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("分页获取邮件失败",
			"email", a.config.Email,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode paginated response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		emails = append(emails, email)
	}

	// 构建分页结果
	page := &EmailPage{
		Emails:        emails,
		NextPageToken: messageList.NextLink,
		HasNextPage:   messageList.NextLink != "",
		PageSize:      len(emails),
	}

	a.logger.Info("成功分页获取邮件",
		"email", a.config.Email,
		"count", len(emails),
		"has_next", page.HasNextPage)

	return page, nil
}

// EmailPage 分页邮件结果
type EmailPage struct {
	Emails        []*Email `json:"emails"`
	NextPageToken string   `json:"next_page_token"`
	HasNextPage   bool     `json:"has_next_page"`
	PageSize      int      `json:"page_size"`
}

// FetchEmailsByFolder 按文件夹获取邮件
func (a *GraphQuickAdapter) FetchEmailsByFolder(ctx context.Context, folderName string, limit int) ([]*Email, error) {
	a.logger.Info("开始按文件夹获取邮件",
		"email", a.config.Email,
		"folder", folderName,
		"limit", limit)

	// 首先获取文件夹 ID
	folderID, err := a.getFolderID(ctx, folderName)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder ID: %w", err)
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("$orderby", "receivedDateTime DESC")

	fetchLimit := 100
	if limit > 0 && limit < 100 {
		fetchLimit = limit
	}
	params.Set("$top", fmt.Sprintf("%d", fetchLimit))

	// 选择需要的字段
	params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/me/mailFolders/%s/messages?%s", a.baseURL, folderID, params.Encode())

	// 发送请求
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch folder messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("按文件夹获取邮件失败",
			"email", a.config.Email,
			"folder", folderName,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode folder response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		email.SourceFolder = folderName // 设置源文件夹
		emails = append(emails, email)
	}

	a.logger.Info("成功按文件夹获取邮件",
		"email", a.config.Email,
		"folder", folderName,
		"count", len(emails))

	return emails, nil
}

// getFolderID 获取文件夹 ID
func (a *GraphQuickAdapter) getFolderID(ctx context.Context, folderName string) (string, error) {
	// 常见文件夹的映射
	wellKnownFolders := map[string]string{
		"inbox":   "inbox",
		"sent":    "sentitems",
		"drafts":  "drafts",
		"deleted": "deleteditems",
		"junk":    "junkemail",
		"outbox":  "outbox",
		"archive": "archive",
	}

	// 检查是否是已知文件夹
	if folderID, exists := wellKnownFolders[strings.ToLower(folderName)]; exists {
		return folderID, nil
	}

	// 搜索自定义文件夹
	requestURL := fmt.Sprintf("%s/me/mailFolders?$filter=displayName eq '%s'", a.baseURL, folderName)

	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to search folder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to find folder: %s", folderName)
	}

	var folderList struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&folderList); err != nil {
		return "", fmt.Errorf("failed to decode folder response: %w", err)
	}

	if len(folderList.Value) == 0 {
		return "", fmt.Errorf("folder not found: %s", folderName)
	}

	return folderList.Value[0].ID, nil
}

// GetEmailCount 获取邮件总数
func (a *GraphQuickAdapter) GetEmailCount(ctx context.Context) (int, error) {
	a.logger.Info("开始获取邮件总数", "email", a.config.Email)

	requestURL := fmt.Sprintf("%s/me/messages/$count", a.baseURL)

	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get email count: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read count response: %w", err)
	}

	count := 0
	if _, err := fmt.Sscanf(string(body), "%d", &count); err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	a.logger.Info("成功获取邮件总数", "email", a.config.Email, "count", count)
	return count, nil
}
