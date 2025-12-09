package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// TokenManager token 管理器接口
type TokenManager interface {
	// GetToken 获取当前有效的 token
	GetToken(ctx context.Context) (string, error)

	// RefreshToken 刷新 token
	RefreshToken(ctx context.Context) error

	// IsValid 检查 token 是否有效
	IsValid() bool

	// GetExpiry 获取过期时间
	GetExpiry() time.Time

	// Clear 清除 token
	Clear()
}

// QuickTokenManager 短效 token 管理器
type QuickTokenManager struct {
	config     *Config
	httpClient HTTPClient
	logger     Logger

	// Token 状态
	accessToken string
	tokenExpiry time.Time
	mutex       sync.RWMutex

	// 配置
	refreshBuffer time.Duration // 提前刷新时间
	maxRetries    int
	retryDelay    time.Duration
}

// NewQuickTokenManager 创建短效 token 管理器
func NewQuickTokenManager(config *Config, httpClient HTTPClient, logger Logger) *QuickTokenManager {
	return &QuickTokenManager{
		config:        config,
		httpClient:    httpClient,
		logger:        logger,
		refreshBuffer: 5 * time.Minute, // 提前5分钟刷新
		maxRetries:    3,
		retryDelay:    time.Second,
	}
}

// GetToken 获取当前有效的 token
func (tm *QuickTokenManager) GetToken(ctx context.Context) (string, error) {
	tm.mutex.RLock()

	// 检查是否需要刷新
	needRefresh := tm.accessToken == "" || time.Now().Add(tm.refreshBuffer).After(tm.tokenExpiry)

	if !needRefresh {
		token := tm.accessToken
		tm.mutex.RUnlock()
		return token, nil
	}

	tm.mutex.RUnlock()

	// 需要刷新 token
	if err := tm.RefreshToken(ctx); err != nil {
		return "", err
	}

	tm.mutex.RLock()
	token := tm.accessToken
	tm.mutex.RUnlock()

	return token, nil
}

// RefreshToken 刷新 token
func (tm *QuickTokenManager) RefreshToken(ctx context.Context) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 双重检查
	if tm.accessToken != "" && time.Now().Add(tm.refreshBuffer).Before(tm.tokenExpiry) {
		return nil
	}

	// Debug 级别：频繁调用的操作
	// tm.logger.Info("开始刷新 token", ...)

	var lastErr error
	for attempt := 1; attempt <= tm.maxRetries; attempt++ {
		if attempt > 1 {
			delay := time.Duration(attempt-1) * tm.retryDelay
			// Debug 级别：频繁调用的操作
			// tm.logger.Info("重试刷新 token", ...)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		token, expiry, err := tm.doRefresh(ctx)
		if err == nil {
			tm.accessToken = token
			tm.tokenExpiry = expiry
			// Debug 级别：频繁调用的操作
			// tm.logger.Info("token 刷新成功", ...)
			return nil
		}

		lastErr = err
		tm.logger.Warn("token 刷新失败", "attempt", attempt, "error", err)

		// 检查是否是不可重试的错误
		if tm.isNonRetryableError(err) {
			break
		}
	}

	return fmt.Errorf("刷新 token 失败，已重试 %d 次: %w", tm.maxRetries, lastErr)
}

// doRefresh 执行实际的刷新操作
func (tm *QuickTokenManager) doRefresh(ctx context.Context) (string, time.Time, error) {
	// 这里实现具体的 token 刷新逻辑
	// 返回新的 token 和过期时间

	// 构建请求参数
	data := map[string]string{
		"client_id":     tm.config.Credentials.ClientID,
		"grant_type":    "refresh_token",
		"refresh_token": tm.config.Credentials.RefreshToken,
		"scope":         "https://graph.microsoft.com/.default",
	}

	// 发送请求（这里简化实现，实际需要完整的 HTTP 请求）
	response, err := tm.sendTokenRequest(ctx, data)
	if err != nil {
		return "", time.Time{}, err
	}

	expiry := time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)
	return response.AccessToken, expiry, nil
}

// sendTokenRequest 发送 token 请求
func (tm *QuickTokenManager) sendTokenRequest(ctx context.Context, data map[string]string) (*TokenResponse, error) {
	// 实际的 HTTP 请求实现
	// 这里返回模拟响应
	return &TokenResponse{
		AccessToken: "mock_token",
		ExpiresIn:   3600,
	}, nil
}

// IsValid 检查 token 是否有效
func (tm *QuickTokenManager) IsValid() bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	return tm.accessToken != "" && time.Now().Before(tm.tokenExpiry)
}

// GetExpiry 获取过期时间
func (tm *QuickTokenManager) GetExpiry() time.Time {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	return tm.tokenExpiry
}

// Clear 清除 token
func (tm *QuickTokenManager) Clear() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.accessToken = ""
	tm.tokenExpiry = time.Time{}
	// Debug 级别：频繁调用的操作
	// tm.logger.Info("已清除 token", ...)
}

// GetInfo 获取 token 信息
func (tm *QuickTokenManager) GetInfo() map[string]interface{} {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	info := map[string]interface{}{
		"has_token":  tm.accessToken != "",
		"expires_at": tm.tokenExpiry.Format(time.RFC3339),
		"is_valid":   tm.accessToken != "" && time.Now().Before(tm.tokenExpiry),
		"expires_in": int(time.Until(tm.tokenExpiry).Seconds()),
	}

	if tm.accessToken != "" {
		tokenPreview := tm.accessToken
		if len(tokenPreview) > 10 {
			tokenPreview = tokenPreview[:10] + "..."
		}
		info["token_preview"] = tokenPreview
	}

	return info
}

// isNonRetryableError 判断是否为不可重试的错误
func (tm *QuickTokenManager) isNonRetryableError(err error) bool {
	var tokenErr *TokenError
	if errors.As(err, &tokenErr) {
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

// SetRefreshBuffer 设置刷新缓冲时间
func (tm *QuickTokenManager) SetRefreshBuffer(buffer time.Duration) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.refreshBuffer = buffer
}

// SetRetryConfig 设置重试配置
func (tm *QuickTokenManager) SetRetryConfig(maxRetries int, retryDelay time.Duration) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.maxRetries = maxRetries
	tm.retryDelay = retryDelay
}

// HTTPClient HTTP 客户端接口
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
