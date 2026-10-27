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

	"fusionmail/internal/dto/webapiconfig"
)

// CloudMailAccountInfo Cloud Mail 服务端账户信息（保留兼容性）
type CloudMailAccountInfo = SubAccountInfo

// GetCloudMailAccounts 获取 Cloud Mail 服务端的账户列表（兼容旧 API）
// 已废弃，请使用 GetSubAccounts
func (s *WebAPIProviderChildAccountService) GetCloudMailAccounts(ctx context.Context, accountUID string) ([]*CloudMailAccountInfo, error) {
	return s.GetSubAccounts(ctx, accountUID)
}

// CloudflareTempEmailSettings Cloudflare Temp Email 设置信息
type CloudflareTempEmailSettings struct {
	Email   string   `json:"email"`             // 邮箱地址
	Domains []string `json:"domains,omitempty"` // 可用域名列表
}

// FetchCloudflareTempEmailSettings 获取 Cloudflare Temp Email 设置
// 通过 JWT Token 调用 /api/settings 接口获取邮箱地址和域名信息
func (s *WebAPIProviderChildAccountService) FetchCloudflareTempEmailSettings(ctx context.Context, baseURL, jwtToken string) (*CloudflareTempEmailSettings, error) {
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
func (s *WebAPIProviderChildAccountService) fetchCloudflareTempEmailBindAddresses(ctx context.Context, config *webapiconfig.CloudflareTempEmailAuthData) ([]*BindAddressInfo, error) {
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
