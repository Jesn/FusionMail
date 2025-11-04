package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGraphQuickAdapter_TokenRefresh 测试 token 刷新机制
func TestGraphQuickAdapter_TokenRefresh(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			// 模拟 token 刷新响应
			response := TokenResponse{
				AccessToken: "new_access_token_123",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
				Scope:       "https://graph.microsoft.com/.default",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 创建适配器配置，使用测试服务器 URL
	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout:  30 * time.Second,
		TokenURL: server.URL + "/token", // 使用测试服务器的 token 端点
	}

	// 创建适配器
	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	ctx := context.Background()

	// 测试 token 刷新
	err = adapter.refreshAccessToken(ctx)
	if err != nil {
		t.Fatalf("Token refresh failed: %v", err)
	}

	// 验证 token 信息
	if !adapter.IsTokenValid() {
		t.Error("Token should be valid after refresh")
	}

	tokenInfo := adapter.GetTokenInfo()
	if tokenInfo["has_token"] != true {
		t.Error("Should have token after refresh")
	}

	if tokenInfo["token_preview"] != "new_access..." {
		t.Errorf("Unexpected token preview: %v", tokenInfo["token_preview"])
	}
}

// TestGraphQuickAdapter_EnsureValidToken 测试 token 有效性检查
func TestGraphQuickAdapter_EnsureValidToken(t *testing.T) {
	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// 初始状态应该没有 token
	if adapter.IsTokenValid() {
		t.Error("Token should not be valid initially")
	}

	// 设置一个过期的 token
	adapter.tokenMutex.Lock()
	adapter.accessToken = "expired_token"
	adapter.tokenExpiry = time.Now().Add(-time.Hour) // 1小时前过期
	adapter.tokenMutex.Unlock()

	// 检查 token 是否被识别为无效
	if adapter.IsTokenValid() {
		t.Error("Expired token should not be valid")
	}

	// 设置一个有效的 token
	adapter.tokenMutex.Lock()
	adapter.accessToken = "valid_token"
	adapter.tokenExpiry = time.Now().Add(time.Hour) // 1小时后过期
	adapter.tokenMutex.Unlock()

	// 检查 token 是否被识别为有效
	if !adapter.IsTokenValid() {
		t.Error("Valid token should be recognized as valid")
	}
}

// TestGraphQuickAdapter_TokenError 测试 token 错误处理
func TestGraphQuickAdapter_TokenError(t *testing.T) {
	// 测试 TokenError 类型
	tokenErr := &TokenError{
		Code:        "invalid_grant",
		Description: "The provided authorization grant is invalid",
		StatusCode:  400,
	}

	expectedMsg := "token error invalid_grant: The provided authorization grant is invalid (status: 400)"
	if tokenErr.Error() != expectedMsg {
		t.Errorf("Unexpected error message: %v", tokenErr.Error())
	}

	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// 测试不可重试错误的识别
	if !adapter.isNonRetryableError(tokenErr) {
		t.Error("invalid_grant should be identified as non-retryable")
	}

	// 测试可重试错误
	retryableErr := &TokenError{
		Code:        "temporarily_unavailable",
		Description: "Service temporarily unavailable",
		StatusCode:  503,
	}

	if adapter.isNonRetryableError(retryableErr) {
		t.Error("temporarily_unavailable should be identified as retryable")
	}
}

// TestGraphQuickAdapter_ConcurrentTokenRefresh 测试并发 token 刷新
func TestGraphQuickAdapter_ConcurrentTokenRefresh(t *testing.T) {
	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// 设置一个即将过期的 token
	adapter.tokenMutex.Lock()
	adapter.accessToken = "expiring_token"
	adapter.tokenExpiry = time.Now().Add(time.Minute) // 1分钟后过期
	adapter.tokenMutex.Unlock()

	ctx := context.Background()

	// 并发调用 ensureValidToken
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// 这应该不会导致竞态条件
			adapter.ensureValidToken(ctx)
		}()
	}

	// 等待所有协程完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证 token 状态一致
	tokenInfo := adapter.GetTokenInfo()
	if tokenInfo["has_token"] != true {
		t.Error("Should have token after concurrent access")
	}
}

// TestGraphQuickAdapter_FetchEmailsEndpoint 测试邮件获取端点与 micro.py 的一致性
// 这是关键测试：确保使用正确的端点 /me/mailFolders/inbox/messages
func TestGraphQuickAdapter_FetchEmailsEndpoint(t *testing.T) {
	var capturedURL string

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 捕获请求的 URL
		capturedURL = r.URL.Path

		if r.URL.Path == "/token" {
			// 模拟 token 刷新响应
			response := TokenResponse{
				AccessToken: "test_access_token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
				Scope:       "https://graph.microsoft.com/.default",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/v1.0/me/mailFolders/inbox/messages" {
			// 模拟邮件列表响应（与 micro.py 一致的端点）
			response := GraphMessageList{
				Value: []GraphMessage{
					{
						ID:      "msg1",
						Subject: "Test Email 1",
						From: GraphRecipient{
							EmailAddress: GraphEmailAddress{
								Address: "sender@example.com",
								Name:    "Sender Name",
							},
						},
						BodyPreview:      "This is a test email",
						ReceivedDateTime: time.Now().Format(time.RFC3339),
						SentDateTime:     time.Now().Format(time.RFC3339),
						IsRead:           false,
						HasAttachments:   false,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 创建适配器配置
	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout:  30 * time.Second,
		BaseURL:  server.URL + "/v1.0",
		TokenURL: server.URL + "/token",
	}

	// 创建适配器
	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	ctx := context.Background()

	// 调用 FetchEmails
	emails, err := adapter.FetchEmails(ctx, time.Time{}, 10)
	if err != nil {
		t.Fatalf("FetchEmails failed: %v", err)
	}

	// 验证返回的邮件
	if len(emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(emails))
	}

	// 关键验证：确保使用了正确的端点（与 micro.py 一致）
	expectedPath := "/v1.0/me/mailFolders/inbox/messages"
	if capturedURL != expectedPath {
		t.Errorf("Wrong endpoint used!\nExpected: %s (micro.py compatible)\nGot: %s\n\nThis is a critical error - the endpoint must match micro.py exactly!", expectedPath, capturedURL)
	}

	t.Logf("✓ Endpoint verification passed: using %s (matches micro.py)", capturedURL)
}

// TestGraphQuickAdapter_TokenRefreshParameters 测试 token 刷新参数与 micro.py 的一致性
func TestGraphQuickAdapter_TokenRefreshParameters(t *testing.T) {
	var capturedParams map[string]string

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			// 解析请求参数
			r.ParseForm()
			capturedParams = make(map[string]string)
			for key, values := range r.Form {
				if len(values) > 0 {
					capturedParams[key] = values[0]
				}
			}

			// 模拟 token 响应
			response := TokenResponse{
				AccessToken: "test_access_token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
				Scope:       "https://graph.microsoft.com/.default",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// 创建适配器配置
	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id_123",
			RefreshToken: "test_refresh_token_456",
		},
		Timeout:  30 * time.Second,
		TokenURL: server.URL + "/token",
	}

	// 创建适配器
	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	ctx := context.Background()

	// 刷新 token
	err = adapter.refreshAccessToken(ctx)
	if err != nil {
		t.Fatalf("Token refresh failed: %v", err)
	}

	// 验证参数与 micro.py 一致
	expectedParams := map[string]string{
		"client_id":     "test_client_id_123",
		"grant_type":    "refresh_token",
		"refresh_token": "test_refresh_token_456",
		"scope":         "https://graph.microsoft.com/.default",
	}

	for key, expectedValue := range expectedParams {
		actualValue, exists := capturedParams[key]
		if !exists {
			t.Errorf("Missing parameter: %s (required by micro.py)", key)
		} else if actualValue != expectedValue {
			t.Errorf("Parameter %s mismatch:\nExpected: %s (micro.py)\nGot: %s", key, expectedValue, actualValue)
		}
	}

	t.Logf("✓ Token refresh parameters match micro.py")
}

// BenchmarkGraphQuickAdapter_TokenValidation 性能测试
func BenchmarkGraphQuickAdapter_TokenValidation(b *testing.B) {
	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		b.Fatalf("Failed to create adapter: %v", err)
	}

	// 设置一个有效的 token
	adapter.tokenMutex.Lock()
	adapter.accessToken = "valid_token"
	adapter.tokenExpiry = time.Now().Add(time.Hour)
	adapter.tokenMutex.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		adapter.IsTokenValid()
	}
}
