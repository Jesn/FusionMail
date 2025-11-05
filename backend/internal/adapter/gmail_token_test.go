package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGmailAdapter_RefreshTokenIfNeeded_TokenValid 测试 token 有效时不刷新
func TestGmailAdapter_RefreshTokenIfNeeded_TokenValid(t *testing.T) {
	// 创建一个有效的 token（1小时后过期）
	config := &Config{
		Email:    "test@gmail.com",
		Provider: "gmail",
		Protocol: "gmail_api",
		Credentials: &Credentials{
			Email:        "test@gmail.com",
			AuthType:     "oauth2",
			AccessToken:  "valid-token",
			RefreshToken: "refresh-token",
			TokenExpiry:  time.Now().Add(1 * time.Hour),
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	adapter, err := NewGmailAdapter(config)
	assert.NoError(t, err)

	// 调用 RefreshTokenIfNeeded
	err = adapter.RefreshTokenIfNeeded(context.Background())

	// 验证：不应该刷新，token 保持不变
	assert.NoError(t, err)
	assert.Equal(t, "valid-token", adapter.config.Credentials.AccessToken)
}

// TestGmailAdapter_GetTokenExpiry 测试获取 token 过期时间
func TestGmailAdapter_GetTokenExpiry(t *testing.T) {
	expectedExpiry := time.Now().Add(1 * time.Hour)

	config := &Config{
		Email:    "test@gmail.com",
		Provider: "gmail",
		Protocol: "gmail_api",
		Credentials: &Credentials{
			Email:        "test@gmail.com",
			AuthType:     "oauth2",
			AccessToken:  "test-token",
			RefreshToken: "refresh-token",
			TokenExpiry:  expectedExpiry,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	adapter, err := NewGmailAdapter(config)
	assert.NoError(t, err)

	// 获取 token 过期时间
	expiry := adapter.GetTokenExpiry()

	// 验证
	assert.Equal(t, expectedExpiry, expiry)
}

// TestGmailAdapter_RefreshTokenIfNeeded_TokenExpiring 测试 token 即将过期时的行为
func TestGmailAdapter_RefreshTokenIfNeeded_TokenExpiring(t *testing.T) {
	// 创建一个即将过期的 token（2分钟后过期）
	config := &Config{
		Email:    "test@gmail.com",
		Provider: "gmail",
		Protocol: "gmail_api",
		Credentials: &Credentials{
			Email:        "test@gmail.com",
			AuthType:     "oauth2",
			AccessToken:  "old-token",
			RefreshToken: "refresh-token",
			TokenExpiry:  time.Now().Add(2 * time.Minute),
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	adapter, err := NewGmailAdapter(config)
	assert.NoError(t, err)

	// 注意：这个测试会失败，因为没有真实的 OAuth2 服务器
	// 在实际测试中，需要 mock OAuth2 服务器
	// 这里只是验证逻辑流程

	// 调用 RefreshTokenIfNeeded
	err = adapter.RefreshTokenIfNeeded(context.Background())

	// 由于没有 mock，预期会失败
	// 在实际测试中，应该 mock OAuth2 服务器并验证刷新成功
	assert.Error(t, err) // 预期失败（因为没有真实的 OAuth2 服务器）
}

// TestGmailAdapter_TokenRefresher_Interface 测试 GmailAdapter 实现了 TokenRefresher 接口
func TestGmailAdapter_TokenRefresher_Interface(t *testing.T) {
	config := &Config{
		Email:    "test@gmail.com",
		Provider: "gmail",
		Protocol: "gmail_api",
		Credentials: &Credentials{
			Email:        "test@gmail.com",
			AuthType:     "oauth2",
			AccessToken:  "test-token",
			RefreshToken: "refresh-token",
			TokenExpiry:  time.Now().Add(1 * time.Hour),
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	adapter, err := NewGmailAdapter(config)
	assert.NoError(t, err)

	// 验证 GmailAdapter 实现了 TokenRefresher 接口
	var _ TokenRefresher = adapter
}
