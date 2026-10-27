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
	"fusionmail/internal/dto/webapiconfig"
)

// SubAccountInfo 服务端子邮箱账户信息（通用结构）
type SubAccountInfo struct {
	AccountID int    `json:"account_id"` // 账户 ID（Cloud Mail 使用）
	Email     string `json:"email"`      // 邮箱地址
	Name      string `json:"name"`       // 账户名称
}

// GetSubAccounts 获取服务端的子邮箱账户列表
// 支持 Cloud Mail 和 Cloudflare Temp Email
func (s *WebAPIProviderChildAccountService) GetSubAccounts(ctx context.Context, accountUID string) ([]*SubAccountInfo, error) {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("账户未找到: %w", err)
	}
	if account == nil {
		return nil, errors.New("账户不存在")
	}

	// 2. 验证是否为 WebAPI 类型
	if !isWebAPIAccount(ctx, s.providerRepo, account) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 获取服务类型
	serviceType, err := getServiceTypeFromAccount(ctx, s.providerRepo, account)
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
	case webapiconfig.WebAPIServiceTypeCloudMail:
		return s.getCloudMailSubAccounts(ctx, string(authDataBytes))

	case webapiconfig.WebAPIServiceTypeCloudflareTempEmail:
		return s.getCloudflareTempEmailSubAccounts(ctx, string(authDataBytes))

	default:
		return nil, fmt.Errorf("该服务类型不支持获取子邮箱列表: %s", serviceType)
	}
}

// getCloudMailSubAccounts 获取 Cloud Mail 服务端的子邮箱列表
func (s *WebAPIProviderChildAccountService) getCloudMailSubAccounts(ctx context.Context, authDataJSON string) ([]*SubAccountInfo, error) {
	// 创建适配器并连接
	adapter, err := s.factory.CreateAdapter(webapiconfig.WebAPIServiceTypeCloudMail, authDataJSON)
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
func (s *WebAPIProviderChildAccountService) getCloudflareTempEmailSubAccounts(ctx context.Context, authDataJSON string) ([]*SubAccountInfo, error) {
	// 解析配置
	var config webapiconfig.CloudflareTempEmailAuthData
	if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	var result []*SubAccountInfo

	// 根据访问模式返回不同的邮箱列表
	if config.AccessMode == webapiconfig.WebAPIAccessModeSingle {
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
	} else if config.AccessMode == webapiconfig.WebAPIAccessModeAdmin {
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
func (s *WebAPIProviderChildAccountService) fetchCloudflareTempEmailAddresses(ctx context.Context, config *webapiconfig.CloudflareTempEmailAuthData) ([]*SubAccountInfo, error) {
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
