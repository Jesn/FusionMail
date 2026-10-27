package cloudflare

// CloudflareTempEmailAdapter user_token 刷新：
// 临近过期自动检查并刷新 user_token，成功后回调通知服务层保存新 token。
// 自 adapter.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"fusionmail/internal/dto/webapiconfig"
)

// checkAndRefreshToken 检查并刷新 user_token
// 仅在配置了 user_token 且距离过期 ≤7 天时才刷新
func (a *CloudflareTempEmailAdapter) checkAndRefreshToken(ctx context.Context) error {
	// 只有配置了 user_token 才需要检查刷新
	if !a.config.HasUserToken() {
		return nil
	}

	// 检查是否需要刷新（距离过期 ≤7 天）
	if !a.config.NeedsTokenRefresh() {
		exp := a.config.GetUserTokenExpiry()
		if exp > 0 {
			remaining := time.Until(time.Unix(exp, 0))
			a.log.Debug("user_token 无需刷新: 剩余有效期 %v", remaining)
		}
		return nil
	}

	a.log.Info("user_token 即将过期，尝试刷新...")

	// 调用 /user_api/settings 获取新 token
	newToken, err := a.refreshUserToken(ctx)
	if err != nil {
		return fmt.Errorf("刷新 user_token 失败: %w", err)
	}

	if newToken == "" {
		a.log.Debug("服务端未返回新 token，当前 token 仍有效")
		return nil
	}

	// 更新本地配置
	oldToken := a.config.UserToken
	a.config.UserToken = newToken

	// 通知服务层保存新 token
	a.tokenUpdateMu.Lock()
	callback := a.tokenUpdateCallback
	a.tokenUpdateMu.Unlock()

	if callback != nil {
		if err := callback(newToken); err != nil {
			// 回滚本地配置
			a.config.UserToken = oldToken
			return fmt.Errorf("保存新 token 失败: %w", err)
		}
	}

	// 解析新 token 的过期时间用于日志
	newExp := webapiconfig.ParseJWTExpiry(newToken)
	if newExp > 0 {
		a.log.Info("user_token 刷新成功: 新过期时间 %v", time.Unix(newExp, 0))
	} else {
		a.log.Info("user_token 刷新成功")
	}

	return nil
}

// refreshUserToken 调用 /user_api/settings 刷新 user_token
// 返回新的 token（如果服务端返回了 new_user_token）
func (a *CloudflareTempEmailAdapter) refreshUserToken(ctx context.Context) (string, error) {
	// 构建请求 URL
	url := a.config.BaseURL + "/user_api/settings"

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	// user_api 只需要 x-user-token 头，不需要 Authorization
	req.Header.Set("x-user-token", a.config.UserToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("认证失败: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("服务器返回错误: HTTP %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 JSON
	// 当 token 距离过期 ≤7 天时，响应中会包含 new_user_token 字段
	var settingsResp struct {
		NewUserToken string `json:"new_user_token"` // 新的 user_token（如果需要刷新）
	}

	if err := json.Unmarshal(body, &settingsResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	return settingsResp.NewUserToken, nil
}
