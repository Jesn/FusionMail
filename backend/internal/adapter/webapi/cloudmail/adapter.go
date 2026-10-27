package cloudmail

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/dto/webapiconfig"
	"fusionmail/pkg/logger"
)

// init 注册 Cloud Mail 适配器
// Register 向 webapi 注册本服务的适配器创建函数与服务模板。
// 必须由组合根（cmd/server/container.go）显式调用 —— 不用 init() 隐式注册：
// blank import 的副作用注册在重构中一旦被当作"未使用的 import"删掉，
// 适配器会静默从注册表消失（无编译错误，仅运行时故障）。
func Register() {
	// 注册适配器创建函数
	webapi.RegisterAdapter(webapiconfig.WebAPIServiceTypeCloudMail, func(authDataJSON string) (webapi.WebAPIProvider, error) {
		var config webapiconfig.CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, webapi.WrapError(webapi.ErrCodeConfigError, "解析 Cloud Mail 配置失败", err)
		}
		return NewCloudMailAdapter(&config)
	})

	// 注册服务模板
	webapi.RegisterServiceTemplate(&webapi.ServiceTemplate{
		ServiceType: webapiconfig.WebAPIServiceTypeCloudMail,
		Name:        "Cloud Mail",
		Description: "Cloud Mail 邮箱服务（如 mail.hema.edu.kg），支持多账户管理，通过 JWT Token 认证",
		AccessModes: []string{"multi_account"},
		AuthFields: []webapi.AuthField{
			{Name: "base_url", Label: "API 地址", Type: "text", Required: true, Placeholder: "https://mail.hema.edu.kg"},
			{Name: "jwt_token", Label: "JWT Token", Type: "password", Required: true, HelpText: "登录后从浏览器获取的 authorization token"},
			{Name: "accounts", Label: "账户列表", Type: "textarea", Required: false, HelpText: "可选，留空则自动从 API 获取账户列表"},
		},
	})
}

// CloudMailAdapter Cloud Mail 适配器
// 支持多账户管理，通过 JWT Token 认证
// 适配 mail.hema.edu.kg 等 Cloud Mail 服务
type CloudMailAdapter struct {
	*webapi.BaseWebAPIAdapter

	// 配置
	config *webapiconfig.CloudMailAuthData

	// 运行时账户列表（从 API 获取或配置指定）
	accounts []CloudMailAccount

	// HTTP 客户端
	httpClient *http.Client

	// 日志
	log *logger.Logger
}

// CloudMailAccount 运行时账户信息
type CloudMailAccount struct {
	AccountID int    `json:"accountId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
}

// NewCloudMailAdapter 创建 Cloud Mail 适配器
func NewCloudMailAdapter(config *webapiconfig.CloudMailAuthData) (*CloudMailAdapter, error) {
	if config == nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置不能为空", nil)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConfigError, "配置验证失败", err)
	}

	// 创建 HTTP 客户端，配置更好的连接处理
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 5,
	}

	return &CloudMailAdapter{
		BaseWebAPIAdapter: webapi.NewBaseWebAPIAdapter(webapiconfig.WebAPIServiceTypeCloudMail),
		config:            config,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		log: logger.NewWithModule("CloudMail"),
	}, nil
}

// Connect 连接到 Cloud Mail 服务
func (a *CloudMailAdapter) Connect(ctx context.Context) error {
	a.log.Info("连接到 Cloud Mail: base_url=%s", a.config.BaseURL)

	// 如果没有 JWT Token 但有邮箱+密码，先登录获取 Token
	if a.config.JWTToken == "" && a.config.Email != "" && a.config.Password != "" {
		if err := a.login(ctx); err != nil {
			return err
		}
	}

	// 测试连接并获取账户列表
	if err := a.TestConnection(ctx); err != nil {
		return err
	}

	// 获取账户列表
	if err := a.fetchAccountList(ctx); err != nil {
		return err
	}

	a.SetConnected(true)
	a.log.Info("Cloud Mail 连接成功: accounts=%d", len(a.accounts))
	return nil
}

// Disconnect 断开连接
func (a *CloudMailAdapter) Disconnect() error {
	a.SetConnected(false)
	a.accounts = nil
	return nil
}

// GetProviderType 获取提供商类型
func (a *CloudMailAdapter) GetProviderType() string {
	return webapiconfig.WebAPIServiceTypeCloudMail
}

// GetProtocol 获取协议类型
func (a *CloudMailAdapter) GetProtocol() string {
	return "webapi"
}

// GetConfig 获取配置
func (a *CloudMailAdapter) GetConfig() *webapiconfig.CloudMailAuthData {
	return a.config
}

// GetAccounts 获取账户列表
func (a *CloudMailAdapter) GetAccounts() []CloudMailAccount {
	return a.accounts
}

// ============================================
// 响应数据结构（适配 mail.hema.edu.kg API）
// ============================================

// AccountListResponse 账户列表响应
type AccountListResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    []AccountItem `json:"data"`
}

// AccountItem 账户项
type AccountItem struct {
	AccountID       int     `json:"accountId"`
	Email           string  `json:"email"`
	Name            string  `json:"name"`
	Status          int     `json:"status"`
	LatestEmailTime *string `json:"latestEmailTime"`
	CreateTime      string  `json:"createTime"`
	UserID          int     `json:"userId"`
	IsDel           int     `json:"isDel"`
}

// EmailListResponse 邮件列表响应
type EmailListResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    EmailListData `json:"data"`
}

// EmailListData 邮件列表数据
type EmailListData struct {
	List        []EmailItem `json:"list"`
	Total       int         `json:"total"`
	LatestEmail *EmailItem  `json:"latestEmail,omitempty"`
}

// EmailDetailResponse 邮件详情响应
type EmailDetailResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    EmailItem `json:"data"`
}

// EmailItem 邮件项（适配 mail.hema.edu.kg 格式）
type EmailItem struct {
	EmailID       int              `json:"emailId"`
	SendEmail     string           `json:"sendEmail"`     // 发件人邮箱
	Name          string           `json:"name"`          // 发件人名称
	AccountID     int              `json:"accountId"`     // 账户 ID
	UserID        int              `json:"userId"`        // 用户 ID
	Subject       string           `json:"subject"`       // 主题
	Text          string           `json:"text"`          // 纯文本内容
	Content       string           `json:"content"`       // HTML 内容
	CC            string           `json:"cc"`            // 抄送（JSON 数组字符串）
	BCC           string           `json:"bcc"`           // 密送（JSON 数组字符串）
	Recipient     string           `json:"recipient"`     // 收件人（JSON 数组字符串）
	ToEmail       string           `json:"toEmail"`       // 收件人邮箱
	ToName        string           `json:"toName"`        // 收件人名称
	InReplyTo     string           `json:"inReplyTo"`     // 回复的邮件 ID
	Relation      string           `json:"relation"`      // 关联
	MessageID     string           `json:"messageId"`     // Message-ID
	Type          int              `json:"type"`          // 类型
	Status        int              `json:"status"`        // 状态
	ResendEmailID *int             `json:"resendEmailId"` // 重发邮件 ID
	Message       *string          `json:"message"`       // 消息
	CreateTime    string           `json:"createTime"`    // 创建时间
	IsDel         int              `json:"isDel"`         // 是否删除
	StarID        *int             `json:"starId"`        // 星标 ID
	IsStar        int              `json:"isStar"`        // 是否星标
	AttList       []AttachmentItem `json:"attList"`       // 附件列表
}

// AttachmentItem 附件项
type AttachmentItem struct {
	AttID      int    `json:"attId"`
	FileName   string `json:"fileName"`
	FileSize   int64  `json:"fileSize"`
	FilePath   string `json:"filePath"`
	CreateTime string `json:"createTime"`
}
