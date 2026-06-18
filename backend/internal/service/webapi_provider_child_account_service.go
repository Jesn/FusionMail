package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fusionmail/internal/adapter/webapi/cloudmail"
	"fusionmail/internal/model"
)

// ChildAccountInfo 子邮箱账户信息
type ChildAccountInfo struct {
	UID         string     `json:"uid"`          // 账户 UID
	Email       string     `json:"email"`        // 邮箱地址
	Status      string     `json:"status"`       // 状态
	TotalEmails int64      `json:"total_emails"` // 邮件总数
	UnreadCount int        `json:"unread_count"` // 未读数量
	LastSyncAt  *time.Time `json:"last_sync_at"` // 上次同步时间
	CreatedAt   time.Time  `json:"created_at"`   // 创建时间
}

// GetChildAccounts 获取 WebAPI 账户关联的子邮箱列表
// 子邮箱是指 ParentAccountUID 等于父账户 UID 的账户
func (s *WebAPIProviderService) GetChildAccounts(ctx context.Context, parentUID string) ([]*ChildAccountInfo, error) {
	// 1. 验证父账户存在且为 WebAPI 类型
	parentAccount, err := s.accountRepo.FindByUID(ctx, parentUID)
	if err != nil {
		return nil, fmt.Errorf("父账户未找到: %w", err)
	}
	if parentAccount == nil {
		return nil, errors.New("父账户不存在")
	}

	// 2. 验证是否为 WebAPI 类型
	if !s.isWebAPIAccount(ctx, parentAccount) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 查找所有子账户（通过 ParentAccountUID 关联）
	childAccountList, err := s.accountRepo.FindByParentAccountUID(ctx, parentUID)
	if err != nil {
		return nil, fmt.Errorf("查询子账户失败: %w", err)
	}

	// 4. 构建子账户信息列表
	var childAccounts []*ChildAccountInfo
	for _, acc := range childAccountList {
		// 获取邮件数量
		emailCount, _ := s.emailRepo.CountByAccount(ctx, acc.UID)

		childAccounts = append(childAccounts, &ChildAccountInfo{
			UID:         acc.UID,
			Email:       acc.Email,
			Status:      acc.Status,
			TotalEmails: emailCount,
			UnreadCount: acc.UnreadCount,
			LastSyncAt:  acc.LastSyncAt,
			CreatedAt:   acc.CreatedAt,
		})
	}

	s.log.Info("获取子邮箱列表: parentUID=%s, 子邮箱数量=%d", parentUID, len(childAccounts))
	return childAccounts, nil
}

// SubAccountInfo 服务端子邮箱账户信息（通用结构）
type SubAccountInfo struct {
	AccountID int    `json:"account_id"` // 账户 ID（Cloud Mail 使用）
	Email     string `json:"email"`      // 邮箱地址
	Name      string `json:"name"`       // 账户名称
}

// GetSubAccounts 获取服务端的子邮箱账户列表
// 支持 Cloud Mail 和 Cloudflare Temp Email
func (s *WebAPIProviderService) GetSubAccounts(ctx context.Context, accountUID string) ([]*SubAccountInfo, error) {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("账户未找到: %w", err)
	}
	if account == nil {
		return nil, errors.New("账户不存在")
	}

	// 2. 验证是否为 WebAPI 类型
	if !s.isWebAPIAccount(ctx, account) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 获取服务类型
	serviceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	// 4. 解密认证数据
	authDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("解密认证数据失败: %w", err)
	}

	// 5. 根据服务类型获取子邮箱列表
	switch serviceType {
	case model.WebAPIServiceTypeCloudMail:
		return s.getCloudMailSubAccounts(ctx, string(authDataBytes))

	case model.WebAPIServiceTypeCloudflareTempEmail:
		return s.getCloudflareTempEmailSubAccounts(ctx, string(authDataBytes))

	default:
		return nil, fmt.Errorf("该服务类型不支持获取子邮箱列表: %s", serviceType)
	}
}

// getCloudMailSubAccounts 获取 Cloud Mail 服务端的子邮箱列表
func (s *WebAPIProviderService) getCloudMailSubAccounts(ctx context.Context, authDataJSON string) ([]*SubAccountInfo, error) {
	// 创建适配器并连接
	adapter, err := s.factory.CreateAdapter(model.WebAPIServiceTypeCloudMail, authDataJSON)
	if err != nil {
		return nil, fmt.Errorf("创建适配器失败: %w", err)
	}

	// 连接到服务
	if err := adapter.Connect(ctx); err != nil {
		return nil, fmt.Errorf("连接服务失败: %w", err)
	}
	defer adapter.Disconnect()

	// 获取账户列表（通过类型断言获取 Cloud Mail 特有方法）
	cloudMailAdapter, ok := adapter.(*cloudmail.CloudMailAdapter)
	if !ok {
		return nil, errors.New("适配器不支持获取账户列表")
	}

	accounts := cloudMailAdapter.GetAccounts()

	// 转换为通用格式
	var result []*SubAccountInfo
	for _, acc := range accounts {
		result = append(result, &SubAccountInfo{
			AccountID: acc.AccountID,
			Email:     acc.Email,
			Name:      acc.Name,
		})
	}

	s.log.Info("获取 Cloud Mail 子邮箱列表: 账户数量=%d", len(result))
	return result, nil
}

// getCloudflareTempEmailSubAccounts 获取 Cloudflare Temp Email 的子邮箱列表
func (s *WebAPIProviderService) getCloudflareTempEmailSubAccounts(ctx context.Context, authDataJSON string) ([]*SubAccountInfo, error) {
	// 解析配置
	var config model.CloudflareTempEmailAuthData
	if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	var result []*SubAccountInfo

	// 根据访问模式返回不同的邮箱列表
	if config.AccessMode == model.WebAPIAccessModeSingle {
		// Single 模式：返回当前配置的邮箱或从 API 获取
		email := config.Email
		if email == "" {
			// 根据认证方式选择不同的 API 获取邮箱地址
			if config.HasUserToken() {
				// user_token 模式：调用 /user_api/bind_address 获取绑定的邮箱列表
				bindAddresses, err := s.fetchCloudflareTempEmailBindAddresses(ctx, &config)
				if err == nil && len(bindAddresses) > 0 {
					// 返回所有绑定的邮箱
					for _, addr := range bindAddresses {
						result = append(result, &SubAccountInfo{
							AccountID: addr.ID,
							Email:     addr.Name,
							Name:      extractNameFromEmail(addr.Name),
						})
					}
					s.log.Info("获取 Cloudflare Temp Email 子邮箱列表 (user_token): 模式=%s, 账户数量=%d", config.AccessMode, len(result))
					return result, nil
				} else if err != nil {
					s.log.Warn("获取绑定邮箱列表失败: %v", err)
				}
			} else if config.JWTToken != "" {
				// jwt_token 模式：调用 /api/settings 获取邮箱地址
				settings, err := s.FetchCloudflareTempEmailSettings(ctx, config.BaseURL, config.JWTToken)
				if err == nil && settings.Email != "" {
					email = settings.Email
				}
			}
		}
		if email != "" {
			result = append(result, &SubAccountInfo{
				AccountID: 0,
				Email:     email,
				Name:      extractNameFromEmail(email),
			})
		}
	} else if config.AccessMode == model.WebAPIAccessModeAdmin {
		// Admin 模式：从邮件列表中提取唯一的收件地址
		subAccounts, err := s.fetchCloudflareTempEmailAddresses(ctx, &config)
		if err != nil {
			s.log.Warn("获取 Cloudflare Temp Email 地址列表失败: %v", err)
			// 如果获取失败，返回配置的域名信息
			domains := config.GetDomainList()
			for i, domain := range domains {
				result = append(result, &SubAccountInfo{
					AccountID: i + 1,
					Email:     fmt.Sprintf("*@%s", domain),
					Name:      fmt.Sprintf("域名: %s", domain),
				})
			}
		} else {
			result = subAccounts
		}
	}

	s.log.Info("获取 Cloudflare Temp Email 子邮箱列表: 模式=%s, 账户数量=%d", config.AccessMode, len(result))
	return result, nil
}

// fetchCloudflareTempEmailAddresses 从 Cloudflare Temp Email API 获取地址列表
func (s *WebAPIProviderService) fetchCloudflareTempEmailAddresses(ctx context.Context, config *model.CloudflareTempEmailAuthData) ([]*SubAccountInfo, error) {
	// 构建请求 URL - 获取邮件列表
	url := config.BaseURL + "/admin/mails?offset=0&limit=100"

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置认证头
	req.Header.Set("x-admin-auth", config.AdminPassword)
	if config.JWTToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.JWTToken)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 - Cloudflare Temp Email Admin API 返回格式
	var response struct {
		Results []struct {
			Address     string   `json:"address"`
			ToAddresses []string `json:"to"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取唯一的邮箱地址
	addressMap := make(map[string]bool)
	for _, item := range response.Results {
		// 优先使用 address 字段
		if item.Address != "" {
			addressMap[item.Address] = true
		}
		// 也检查 to 字段
		for _, addr := range item.ToAddresses {
			if addr != "" {
				addressMap[addr] = true
			}
		}
	}

	// 如果配置了域名过滤，只返回匹配的地址
	var result []*SubAccountInfo
	idx := 1
	for addr := range addressMap {
		// 域名过滤
		if config.HasDomainFilter() && !config.MatchesDomain(addr) {
			continue
		}
		result = append(result, &SubAccountInfo{
			AccountID: idx,
			Email:     addr,
			Name:      extractNameFromEmail(addr),
		})
		idx++
	}

	return result, nil
}

// extractNameFromEmail 从邮箱地址提取名称
func extractNameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}

// CloudMailAccountInfo Cloud Mail 服务端账户信息（保留兼容性）
type CloudMailAccountInfo = SubAccountInfo

// GetCloudMailAccounts 获取 Cloud Mail 服务端的账户列表（兼容旧 API）
// 已废弃，请使用 GetSubAccounts
func (s *WebAPIProviderService) GetCloudMailAccounts(ctx context.Context, accountUID string) ([]*CloudMailAccountInfo, error) {
	return s.GetSubAccounts(ctx, accountUID)
}

// CloudflareTempEmailSettings Cloudflare Temp Email 设置信息
type CloudflareTempEmailSettings struct {
	Email   string   `json:"email"`             // 邮箱地址
	Domains []string `json:"domains,omitempty"` // 可用域名列表
}

// FetchCloudflareTempEmailSettings 获取 Cloudflare Temp Email 设置
// 通过 JWT Token 调用 /api/settings 接口获取邮箱地址和域名信息
func (s *WebAPIProviderService) FetchCloudflareTempEmailSettings(ctx context.Context, baseURL, jwtToken string) (*CloudflareTempEmailSettings, error) {
	if baseURL == "" {
		return nil, errors.New("base_url 不能为空")
	}
	if jwtToken == "" {
		return nil, errors.New("jwt_token 不能为空")
	}

	// 构建请求 URL
	url := baseURL + "/api/settings"

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("认证失败，请检查 JWT Token 是否有效")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 JSON
	// Cloudflare Temp Email 的 /api/settings 响应格式可能是：
	// { "address": "user@domain.com", "domains": ["domain1.com", "domain2.com"], ... }
	var settingsResp struct {
		Address string   `json:"address"` // 邮箱地址
		Domains []string `json:"domains"` // 可用域名列表
	}

	if err := json.Unmarshal(body, &settingsResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 构建返回结果
	settings := &CloudflareTempEmailSettings{
		Email:   settingsResp.Address,
		Domains: settingsResp.Domains,
	}

	s.log.Info("获取 Cloudflare Temp Email 设置成功: email=%s, domains=%v", settings.Email, settings.Domains)
	return settings, nil
}

// BindAddressInfo 绑定邮箱地址信息
type BindAddressInfo struct {
	ID   int    `json:"id"`   // 邮箱 ID
	Name string `json:"name"` // 邮箱地址
}

// fetchCloudflareTempEmailBindAddresses 获取 user_token 模式下绑定的邮箱列表
// 调用 /user_api/bind_address 端点
func (s *WebAPIProviderService) fetchCloudflareTempEmailBindAddresses(ctx context.Context, config *model.CloudflareTempEmailAuthData) ([]*BindAddressInfo, error) {
	// 规范化 URL：去除前后空格和末尾斜杠
	baseURL := strings.TrimSpace(config.BaseURL)
	baseURL = strings.TrimRight(baseURL, "/")

	if baseURL == "" {
		return nil, errors.New("base_url 不能为空")
	}
	if config.UserToken == "" {
		return nil, errors.New("user_token 不能为空")
	}

	// 构建请求 URL
	url := baseURL + "/user_api/bind_address"

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头 - user_token 模式只需要 x-user-token 头
	req.Header.Set("x-user-token", config.UserToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("认证失败，请检查 user_token 是否有效")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 JSON
	// /user_api/bind_address 响应格式：
	// { "results": [{"id": 623, "name": "ui_jesn89@ui.edu.kg", ...}] }
	var bindResp struct {
		Results []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &bindResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 转换为返回格式
	var result []*BindAddressInfo
	for _, item := range bindResp.Results {
		result = append(result, &BindAddressInfo{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	s.log.Info("获取 Cloudflare Temp Email 绑定邮箱列表成功: 数量=%d", len(result))
	return result, nil
}
