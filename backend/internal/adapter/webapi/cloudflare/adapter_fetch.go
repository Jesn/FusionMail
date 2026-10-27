package cloudflare

// CloudflareTempEmailAdapter 邮件拉取与解析：
// 邮件列表拉取（Single/Admin 两种模式）、单封邮件解析、邮件详情获取。
// 自 adapter.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
)

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
