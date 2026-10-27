package adapter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// RefreshTokenIfNeeded 实现 TokenRefresher 接口
// 如果 token 即将过期（5分钟内），则自动刷新
func (a *GmailAdapter) RefreshTokenIfNeeded(ctx context.Context) error {
	// 检查 token 是否即将过期 (5分钟内)
	if time.Now().Add(5 * time.Minute).Before(a.config.Credentials.TokenExpiry) {
		return nil // token 仍然有效
	}

	// Token 即将过期，执行刷新
	return a.refreshToken(ctx)
}

// GetTokenExpiry 实现 TokenRefresher 接口
// 返回 token 过期时间
func (a *GmailAdapter) GetTokenExpiry() time.Time {
	return a.config.Credentials.TokenExpiry
}

// refreshToken 刷新 OAuth2 token
func (a *GmailAdapter) refreshToken(ctx context.Context) error {
	if a.oauth2Config == nil {
		return fmt.Errorf("oauth2 config not initialized, call Connect first")
	}

	// 创建当前 token
	currentToken := &oauth2.Token{
		AccessToken:  a.config.Credentials.AccessToken,
		RefreshToken: a.config.Credentials.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       a.config.Credentials.TokenExpiry,
	}

	// 使用 TokenSource 自动刷新
	tokenSource := a.oauth2Config.TokenSource(ctx, currentToken)

	// 获取新 token（如果需要会自动刷新）
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// 更新配置中的 token
	a.config.Credentials.AccessToken = newToken.AccessToken
	a.config.Credentials.TokenExpiry = newToken.Expiry

	// 如果返回了新的 refresh token，也更新它
	if newToken.RefreshToken != "" {
		a.config.Credentials.RefreshToken = newToken.RefreshToken
	}

	// 重新创建 HTTP 客户端和 Gmail 服务
	httpClient := a.oauth2Config.Client(ctx, newToken)

	// 如果配置了代理，设置代理
	if a.config.Proxy != nil && a.config.Proxy.Enabled {
		transport := httpClient.Transport.(*oauth2.Transport)
		transport.Base = &http.Transport{
			Proxy: http.ProxyURL(a.getProxyURL()),
		}
	}

	a.httpClient = httpClient

	// 重新创建 Gmail 服务
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to recreate Gmail service: %w", err)
	}

	a.service = service

	return nil
}
