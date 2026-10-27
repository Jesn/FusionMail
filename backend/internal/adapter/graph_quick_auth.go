package adapter

// GraphQuickAdapter 认证与 token 管理：
// refresh token 换取 access token（含重试）、token 状态查询与清除、
// 401 错误处理、带认证的 HTTP 请求发送。
// 自 graph_quick.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
