package cloudmail

// CloudMailAdapter 认证与连接管理：
// 通用请求头（模拟浏览器）、邮箱+密码登录换 JWT Token、连接测试、账户列表获取。
// 自 adapter.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"fusionmail/internal/adapter/webapi"
)

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
