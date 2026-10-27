package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/dto/webapiconfig"
	"fusionmail/pkg/logger"
)

// init 注册 Cloudflare Temp Email 适配器
// Register 向 webapi 注册本服务的适配器创建函数与服务模板。
// 必须由组合根（cmd/server/container.go）显式调用 —— 不用 init() 隐式注册：
// blank import 的副作用注册在重构中一旦被当作"未使用的 import"删掉，
// 适配器会静默从注册表消失（无编译错误，仅运行时故障）。
func Register() {
	// 注册适配器创建函数
	webapi.RegisterAdapter(webapiconfig.WebAPIServiceTypeCloudflareTempEmail, func(authDataJSON string) (webapi.WebAPIProvider, error) {
		var config webapiconfig.CloudflareTempEmailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, webapi.WrapError(webapi.ErrCodeConfigError, "解析 Cloudflare Temp Email 配置失败", err)
		}
		return NewCloudflareTempEmailAdapter(&config)
	})

	// 注册服务模板
	webapi.RegisterServiceTemplate(&webapi.ServiceTemplate{
		ServiceType: webapiconfig.WebAPIServiceTypeCloudflareTempEmail,
		Name:        "Cloudflare Temp Email",
		Description: "Cloudflare Workers 临时邮箱服务，支持 Single 模式（单邮箱）和 Admin 模式（域名管理）",
		AccessModes: []string{webapiconfig.WebAPIAccessModeSingle, webapiconfig.WebAPIAccessModeAdmin},
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
	config *webapiconfig.CloudflareTempEmailAuthData

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
func NewCloudflareTempEmailAdapter(config *webapiconfig.CloudflareTempEmailAuthData) (*CloudflareTempEmailAdapter, error) {
	if config == nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置不能为空", nil)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置验证失败", err)
	}

	return &CloudflareTempEmailAdapter{
		BaseWebAPIAdapter: webapi.NewBaseWebAPIAdapter(webapiconfig.WebAPIServiceTypeCloudflareTempEmail),
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

// GetProviderType 获取提供商类型
func (a *CloudflareTempEmailAdapter) GetProviderType() string {
	return webapiconfig.WebAPIServiceTypeCloudflareTempEmail
}

// GetProtocol 获取协议类型
func (a *CloudflareTempEmailAdapter) GetProtocol() string {
	return "webapi"
}

// GetConfig 获取配置
func (a *CloudflareTempEmailAdapter) GetConfig() *webapiconfig.CloudflareTempEmailAuthData {
	return a.config
}

// SetTokenUpdateCallback 设置 Token 更新回调
// 当 user_token 被刷新时，会调用此回调通知服务层保存新 token
func (a *CloudflareTempEmailAdapter) SetTokenUpdateCallback(callback TokenUpdateCallback) {
	a.tokenUpdateMu.Lock()
	defer a.tokenUpdateMu.Unlock()
	a.tokenUpdateCallback = callback
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
