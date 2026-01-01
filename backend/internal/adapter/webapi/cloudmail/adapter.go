package cloudmail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	config *model.CloudMailAuthData

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
func NewCloudMailAdapter(config *model.CloudMailAuthData) (*CloudMailAdapter, error) {
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
		BaseWebAPIAdapter: webapi.NewBaseWebAPIAdapter(model.WebAPIServiceTypeCloudMail),
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

// setCommonHeaders 设置通用请求头（模拟浏览器请求）
func (a *CloudMailAdapter) setCommonHeaders(req *http.Request) {
	req.Header.Set("Authorization", a.config.JWTToken)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")
	req.Header.Set("Origin", a.config.BaseURL)
	req.Header.Set("Referer", a.config.BaseURL+"/email")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
}

// login 通过邮箱+密码登录获取 JWT Token
func (a *CloudMailAdapter) login(ctx context.Context) error {
	a.log.Info("通过邮箱+密码登录 Cloud Mail: email=%s", a.config.Email)

	// 构建登录请求
	loginURL := a.config.BaseURL + "/api/login"
	loginData := map[string]string{
		"email":    a.config.Email,
		"password": a.config.Password,
	}
	loginBody, err := json.Marshal(loginData)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConfigError, "构建登录请求失败", err)
	}

	a.log.Info("登录请求 URL: %s", loginURL)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(string(loginBody)))
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建登录请求失败", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "FusionMail/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "登录请求失败", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeParseError, "读取登录响应失败", err)
	}

	a.log.Info("登录响应: status=%d, body=%s", resp.StatusCode, string(body))

	// 解析登录响应
	var loginResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return webapi.WrapError(webapi.ErrCodeParseError, "解析登录响应失败", err)
	}

	if loginResp.Code != 200 {
		return webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, fmt.Sprintf("登录失败: %s", loginResp.Message), loginResp.Code, false, nil)
	}

	if loginResp.Data.Token == "" {
		return webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "登录成功但未返回 Token", 0, false, nil)
	}

	// 保存 Token（添加 Bearer 前缀，如果 API 需要的话）
	a.config.JWTToken = loginResp.Data.Token
	a.log.Info("Cloud Mail 登录成功，Token 长度: %d", len(a.config.JWTToken))
	return nil
}

// TestConnection 测试连接
func (a *CloudMailAdapter) TestConnection(ctx context.Context) error {
	// 如果没有 JWT Token 但有邮箱+密码，先登录获取 Token
	if a.config.JWTToken == "" && a.config.Email != "" && a.config.Password != "" {
		if err := a.login(ctx); err != nil {
			return err
		}
	}

	a.log.Info("测试连接: base_url=%s, token_len=%d", a.config.BaseURL, len(a.config.JWTToken))

	// 测试账户列表 API（带查询参数）
	url := a.config.BaseURL + "/api/account/list?accountId=0&size=20"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置请求头（模拟浏览器请求）
	a.setCommonHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "认证失败，请检查 JWT Token", resp.StatusCode, false, nil)
	}

	if resp.StatusCode != http.StatusOK {
		return webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
	}

	// 解析响应验证格式
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
	}

	var response AccountListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return webapi.WrapError(webapi.ErrCodeParseError, "解析响应失败", err)
	}

	if response.Code != 200 {
		return webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("API 返回错误: %s", response.Message), response.Code, true, nil)
	}

	return nil
}

// fetchAccountList 获取账户列表（从 API 自动获取所有账户）
func (a *CloudMailAdapter) fetchAccountList(ctx context.Context) error {
	apiAccounts, err := a.fetchAccountsFromAPI(ctx)
	if err != nil {
		return err
	}

	a.accounts = apiAccounts
	a.log.Info("从 API 获取账户列表: count=%d", len(a.accounts))
	return nil
}

// fetchAccountsFromAPI 从 API 获取账户列表
func (a *CloudMailAdapter) fetchAccountsFromAPI(ctx context.Context) ([]CloudMailAccount, error) {
	url := a.config.BaseURL + "/api/account/list?accountId=0&size=100"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置请求头（模拟浏览器请求）
	a.setCommonHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
	}

	var response AccountListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析账户列表失败", err)
	}

	if response.Code != 200 {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("获取账户列表失败: %s", response.Message), response.Code, true, nil)
	}

	// 转换账户列表
	accounts := make([]CloudMailAccount, 0, len(response.Data))
	for _, acc := range response.Data {
		if acc.IsDel == 0 { // 只添加未删除的账户
			accounts = append(accounts, CloudMailAccount{
				AccountID: acc.AccountID,
				Email:     acc.Email,
				Name:      acc.Name,
			})
		}
	}

	return accounts, nil
}

// FetchEmails 拉取所有账户的邮件
func (a *CloudMailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	if len(a.accounts) == 0 {
		a.log.Warn("没有可用的账户")
		return []*adapter.Email{}, nil
	}

	allEmails := make([]*adapter.Email, 0)

	// 遍历所有账户拉取邮件
	for _, account := range a.accounts {
		emails, err := a.fetchEmailsForAccount(ctx, account, since, limit)
		if err != nil {
			a.log.Warn("拉取账户邮件失败: account=%s, err=%v", account.Email, err)
			continue
		}
		allEmails = append(allEmails, emails...)
	}

	a.log.Info("Cloud Mail 拉取完成: total=%d, accounts=%d", len(allEmails), len(a.accounts))
	return allEmails, nil
}

// fetchEmailsForAccount 拉取单个账户的邮件
func (a *CloudMailAdapter) fetchEmailsForAccount(ctx context.Context, account CloudMailAccount, since time.Time, limit int) ([]*adapter.Email, error) {
	a.log.Info("拉取账户邮件: account=%s (id=%d), since=%v, limit=%d", account.Email, account.AccountID, since, limit)

	// 构建请求 URL
	// API: /api/email/list?accountId=xxx&emailId=0&timeSort=0&size=30&type=0
	size := limit
	if size <= 0 {
		size = 50
	}
	url := fmt.Sprintf("%s/api/email/list?accountId=%d&emailId=0&timeSort=0&size=%d&type=0",
		a.config.BaseURL, account.AccountID, size)

	a.log.Info("请求 URL: %s", url)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置请求头（模拟浏览器请求）
	a.setCommonHeaders(req)

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	a.log.Info("响应状态码: %d", resp.StatusCode)

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

	a.log.Info("响应体长度: %d bytes", len(body))

	// 解析 JSON
	var response EmailListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		a.log.Error("解析 JSON 失败: %v, body=%s", err, string(body[:min(500, len(body))]))
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	a.log.Info("API 响应: code=%d, message=%s, list_count=%d, total=%d",
		response.Code, response.Message, len(response.Data.List), response.Data.Total)

	if response.Code != 200 {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("API 返回错误: %s", response.Message), response.Code, true, nil)
	}

	// 转换邮件
	emails := make([]*adapter.Email, 0, len(response.Data.List))
	for _, item := range response.Data.List {
		email := a.convertToEmail(item, account.Email)

		// 过滤时间
		if !since.IsZero() && email.ReceivedAt.Before(since) {
			a.log.Debug("跳过旧邮件: id=%d, time=%v, since=%v", item.EmailID, email.ReceivedAt, since)
			continue
		}

		emails = append(emails, email)
	}

	a.log.Info("账户邮件拉取完成: account=%s, count=%d (原始=%d)", account.Email, len(emails), len(response.Data.List))
	return emails, nil
}

// convertToEmail 转换为标准邮件格式
func (a *CloudMailAdapter) convertToEmail(item EmailItem, targetAddress string) *adapter.Email {
	email := &adapter.Email{
		ProviderID:  strconv.Itoa(item.EmailID),
		MessageID:   item.MessageID,
		Subject:     item.Subject,
		FromAddress: item.SendEmail,
		FromName:    item.Name,
		ToAddresses: []string{targetAddress},
		TextBody:    item.Text,
		HTMLBody:    item.Content,
	}

	// 解析收件人（如果有）
	if item.ToEmail != "" {
		email.ToAddresses = []string{item.ToEmail}
	}

	// 解析时间 (格式: "2025-11-17 05:28:25")
	if item.CreateTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", item.CreateTime); err == nil {
			email.ReceivedAt = t
			email.SentAt = t
		}
	}

	// 生成摘要
	email.Snippet = webapi.GenerateSnippet(email, 200)

	// 附件信息
	email.HasAttachments = len(item.AttList) > 0
	email.AttachmentsCount = len(item.AttList)

	return email
}

// FetchEmailDetail 获取邮件详情
func (a *CloudMailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	// 构建请求 URL
	url := fmt.Sprintf("%s/api/email/detail?emailId=%s", a.config.BaseURL, providerID)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置请求头（模拟浏览器请求）
	a.setCommonHeaders(req)

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

	var response EmailDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	if response.Code != 200 {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("API 返回错误: %s", response.Message), response.Code, true, nil)
	}

	return a.convertToEmail(response.Data, response.Data.ToEmail), nil
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
