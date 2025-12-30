package custom

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"
)

// init 注册自定义 WebAPI 适配器
func init() {
	// 注册适配器创建函数
	webapi.RegisterAdapter(model.WebAPIServiceTypeCustom, func(authDataJSON string) (webapi.WebAPIProvider, error) {
		var config model.CustomWebAPIAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, webapi.WrapError(webapi.ErrCodeConfigError, "解析自定义 WebAPI 配置失败", err)
		}
		return NewCustomWebAPIAdapter(&config)
	})

	// 注册服务模板
	webapi.RegisterServiceTemplate(&webapi.ServiceTemplate{
		ServiceType: model.WebAPIServiceTypeCustom,
		Name:        "自定义 Web API",
		Description: "通过配置接入任意支持 RESTful API 的邮箱服务",
		AccessModes: []string{"single", "admin"},
		AuthFields: []webapi.AuthField{
			{Name: "base_url", Label: "API 地址", Type: "text", Required: true, Placeholder: "https://api.example.com", HelpText: "必须使用 HTTPS"},
			{Name: "service_name", Label: "服务名称", Type: "text", Required: true, Placeholder: "My Mail Service"},
			{Name: "auth.type", Label: "认证类型", Type: "select", Required: true, HelpText: "jwt/bearer/apikey/basic/custom"},
			{Name: "auth.token", Label: "Token", Type: "password", Required: false, HelpText: "JWT/Bearer Token"},
			{Name: "auth.api_key", Label: "API Key", Type: "password", Required: false},
			{Name: "auth.api_key_name", Label: "API Key Header", Type: "text", Required: false, Placeholder: "X-API-Key"},
			{Name: "auth.username", Label: "用户名", Type: "text", Required: false, HelpText: "Basic Auth"},
			{Name: "auth.password", Label: "密码", Type: "password", Required: false, HelpText: "Basic Auth"},
			{Name: "list_endpoint", Label: "邮件列表端点", Type: "text", Required: true, Placeholder: "/api/mails"},
			{Name: "data_path", Label: "数据路径", Type: "text", Required: false, Placeholder: "data.list", HelpText: "JSON 响应中邮件数组的路径"},
			{Name: "target_email", Label: "目标邮箱", Type: "text", Required: false, HelpText: "Single 模式下的目标邮箱地址"},
		},
	})
}

// CustomWebAPIAdapter 自定义 Web API 适配器
// 通过配置接入任意 RESTful API 邮箱服务
type CustomWebAPIAdapter struct {
	*webapi.BaseWebAPIAdapter

	// 配置
	config *model.CustomWebAPIAuthData

	// HTTP 客户端
	httpClient *http.Client

	// 日志
	log *logger.Logger

	// 响应解析器
	parser *ResponseParser

	// 分页器
	paginator Paginator
}

// NewCustomWebAPIAdapter 创建自定义 WebAPI 适配器
func NewCustomWebAPIAdapter(config *model.CustomWebAPIAuthData) (*CustomWebAPIAdapter, error) {
	if config == nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置不能为空", nil)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置验证失败", err)
	}

	// 创建分页器
	paginatorFactory := NewPaginatorFactory()
	paginator := paginatorFactory.CreatePaginator(config.Pagination)

	return &CustomWebAPIAdapter{
		BaseWebAPIAdapter: webapi.NewBaseWebAPIAdapter(config.ServiceName),
		config:            config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log:       logger.NewWithModule("CustomWebAPI:" + config.ServiceName),
		parser:    NewResponseParser(config),
		paginator: paginator,
	}, nil
}

// Connect 连接到服务
func (a *CustomWebAPIAdapter) Connect(ctx context.Context) error {
	a.log.Info("连接到自定义 WebAPI: base_url=%s, service=%s", a.config.BaseURL, a.config.ServiceName)

	// 测试连接
	if err := a.TestConnection(ctx); err != nil {
		return err
	}

	a.SetConnected(true)
	return nil
}

// Disconnect 断开连接
func (a *CustomWebAPIAdapter) Disconnect() error {
	a.SetConnected(false)
	return nil
}

// TestConnection 测试连接
func (a *CustomWebAPIAdapter) TestConnection(ctx context.Context) error {
	// 构建测试请求（只获取少量数据）
	params := url.Values{}
	params.Set("limit", "1")

	reqURL := BuildURL(a.config.BaseURL, a.config.ListEndpoint, params)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 应用认证
	a.applyAuth(req)

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "认证失败", resp.StatusCode, false, nil)
	}

	if resp.StatusCode != http.StatusOK {
		return webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
	}

	return nil
}

// FetchEmails 拉取邮件列表
func (a *CustomWebAPIAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	a.log.Debug("拉取邮件: since=%v, limit=%d", since, limit)

	allEmails := make([]*adapter.Email, 0)
	params := a.paginator.GetFirstPageParams()

	// 分页拉取
	for {
		// 检查上下文
		select {
		case <-ctx.Done():
			return allEmails, ctx.Err()
		default:
		}

		// 构建请求
		reqURL := BuildURL(a.config.BaseURL, a.config.ListEndpoint, params)

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return allEmails, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
		}

		// 应用认证
		a.applyAuth(req)

		// 发送请求
		resp, err := a.httpClient.Do(req)
		if err != nil {
			return allEmails, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
		}

		// 读取响应
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return allEmails, webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
		}

		// 检查响应状态
		if resp.StatusCode == http.StatusUnauthorized {
			return allEmails, webapi.ErrAuthFailed
		}
		if resp.StatusCode != http.StatusOK {
			return allEmails, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
		}

		// 解析邮件
		emails, err := a.parser.ParseResponse(body)
		if err != nil {
			return allEmails, err
		}

		// 过滤时间
		for _, email := range emails {
			if !since.IsZero() && email.ReceivedAt.Before(since) {
				continue
			}
			allEmails = append(allEmails, email)

			// 检查数量限制
			if limit > 0 && len(allEmails) >= limit {
				a.log.Info("拉取完成（达到限制）: count=%d", len(allEmails))
				return allEmails, nil
			}
		}

		// 检查是否有下一页
		pageInfo, _ := a.parser.ParsePaginationInfo(body)
		nextParams := a.paginator.GetNextPageParams(pageInfo, params)
		if nextParams == nil {
			break
		}
		params = nextParams
	}

	a.log.Info("拉取完成: count=%d", len(allEmails))
	return allEmails, nil
}

// FetchEmailDetail 获取邮件详情
func (a *CustomWebAPIAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	// 如果没有配置详情端点，返回错误
	if a.config.DetailEndpoint == "" {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "未配置邮件详情端点", nil)
	}

	// 构建请求 URL（替换 {id} 占位符）
	endpoint := a.config.DetailEndpoint
	endpoint = replacePathParam(endpoint, "id", providerID)

	reqURL := a.config.BaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 应用认证
	a.applyAuth(req)

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

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
	}

	// 解析单封邮件
	emails, err := a.parser.ParseResponse(body)
	if err != nil {
		// 尝试直接解析为单个对象
		var rawData map[string]interface{}
		if jsonErr := json.Unmarshal(body, &rawData); jsonErr == nil {
			// 包装成数组再解析
			wrapped, _ := json.Marshal([]interface{}{rawData})
			emails, err = a.parser.ParseResponse(wrapped)
		}
	}

	if err != nil {
		return nil, err
	}

	if len(emails) == 0 {
		return nil, webapi.ErrNotFound
	}

	return emails[0], nil
}

// applyAuth 应用认证
func (a *CustomWebAPIAdapter) applyAuth(req *http.Request) {
	auth := a.config.Auth

	switch auth.Type {
	case model.WebAPIAuthTypeJWT, model.WebAPIAuthTypeBearer:
		req.Header.Set("Authorization", "Bearer "+auth.Token)

	case model.WebAPIAuthTypeAPIKey:
		headerName := auth.APIKeyName
		if headerName == "" {
			headerName = "X-API-Key"
		}
		req.Header.Set(headerName, auth.APIKey)

	case model.WebAPIAuthTypeBasic:
		credentials := base64.StdEncoding.EncodeToString([]byte(auth.Username + ":" + auth.Password))
		req.Header.Set("Authorization", "Basic "+credentials)

	case model.WebAPIAuthTypeCustom:
		req.Header.Set(auth.HeaderName, auth.HeaderValue)
	}

	// 设置通用头
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "FusionMail/1.0")
}

// GetProviderType 获取提供商类型
func (a *CustomWebAPIAdapter) GetProviderType() string {
	return model.WebAPIServiceTypeCustom
}

// GetProtocol 获取协议类型
func (a *CustomWebAPIAdapter) GetProtocol() string {
	return "webapi"
}

// GetConfig 获取配置
func (a *CustomWebAPIAdapter) GetConfig() *model.CustomWebAPIAuthData {
	return a.config
}

// replacePathParam 替换路径参数
func replacePathParam(path, param, value string) string {
	// 支持 {id} 和 :id 两种格式
	path = replaceAll(path, "{"+param+"}", value)
	path = replaceAll(path, ":"+param, value)
	return path
}

// replaceAll 替换所有匹配
func replaceAll(s, old, new string) string {
	for {
		idx := indexOf(s, old)
		if idx < 0 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

// indexOf 查找子串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
