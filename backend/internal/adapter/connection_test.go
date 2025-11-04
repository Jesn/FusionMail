package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestGraphQuickAdapter_TestConnection 测试连接测试功能
func TestGraphQuickAdapter_TestConnection(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorType      string
	}{
		{
			name: "成功连接测试",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					// 模拟 token 响应
					response := TokenResponse{
						AccessToken: "test_access_token",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				} else if r.URL.Path == "/v1.0/me" {
					// 模拟用户信息响应
					userInfo := ConnectionTestResult{
						UserID:        "12345",
						DisplayName:   "Test User",
						Mail:          "test@example.com",
						UserPrincipal: "test@example.com",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(userInfo)
				}
			},
			expectError: false,
		},
		{
			name: "认证失败",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					w.WriteHeader(http.StatusBadRequest)
					errorResp := map[string]interface{}{
						"error":             "invalid_grant",
						"error_description": "The provided authorization grant is invalid",
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			},
			expectError: true,
			errorType:   "authentication",
		},
		{
			name: "API 权限不足",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					response := TokenResponse{
						AccessToken: "test_access_token",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				} else if r.URL.Path == "/v1.0/me" {
					w.WriteHeader(http.StatusForbidden)
					errorResp := map[string]interface{}{
						"error": map[string]interface{}{
							"code":    "Forbidden",
							"message": "Insufficient privileges to complete the operation",
						},
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			},
			expectError: true,
			errorType:   "api",
		},
		{
			name: "服务不可用",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					response := TokenResponse{
						AccessToken: "test_access_token",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				} else if r.URL.Path == "/v1.0/me" {
					w.WriteHeader(http.StatusServiceUnavailable)
					errorResp := map[string]interface{}{
						"error": map[string]interface{}{
							"code":    "ServiceNotAvailable",
							"message": "The service is temporarily unavailable",
						},
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			},
			expectError: true,
			errorType:   "api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试服务器
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// 创建适配器
			adapter := createTestAdapter(t, server.URL)

			// 执行连接测试
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := adapter.TestConnection(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("期望出现错误，但没有错误")
					return
				}

				var connErr *ConnectionTestError
				if !errors.As(err, &connErr) {
					t.Errorf("期望 ConnectionTestError 类型，得到: %T", err)
					return
				}

				if connErr.Type != tt.errorType {
					t.Errorf("期望错误类型 %s，得到 %s", tt.errorType, connErr.Type)
				}
			} else {
				if err != nil {
					t.Errorf("不期望出现错误，但得到: %v", err)
				}
			}
		})
	}
}

// TestGraphQuickAdapter_TestConnectionWithDetails 测试详细连接测试
func TestGraphQuickAdapter_TestConnectionWithDetails(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			response := TokenResponse{
				AccessToken: "test_access_token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/v1.0/me" {
			userInfo := ConnectionTestResult{
				UserID:        "test-user-123",
				DisplayName:   "Test User",
				Mail:          "test@example.com",
				UserPrincipal: "test@example.com",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userInfo)
		}
	}))
	defer server.Close()

	// 创建适配器
	adapter := createTestAdapter(t, server.URL)

	// 执行详细连接测试
	ctx := context.Background()
	result, err := adapter.TestConnectionWithDetails(ctx)

	if err != nil {
		t.Fatalf("详细连接测试失败: %v", err)
	}

	// 验证结果
	if result.UserID != "test-user-123" {
		t.Errorf("期望用户 ID 'test-user-123'，得到 '%s'", result.UserID)
	}

	if result.DisplayName != "Test User" {
		t.Errorf("期望显示名称 'Test User'，得到 '%s'", result.DisplayName)
	}

	if result.Mail != "test@example.com" {
		t.Errorf("期望邮箱 'test@example.com'，得到 '%s'", result.Mail)
	}

	if result.ResponseTimeMs < 0 {
		t.Error("响应时间不应该为负数")
	}
}

// TestGraphQuickAdapter_ValidateCredentials 测试凭据验证
func TestGraphQuickAdapter_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorType   string
	}{
		{
			name: "有效凭据",
			config: &Config{
				Email:    "test@example.com",
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					ClientID:     "valid_client_id",
					RefreshToken: "valid_refresh_token",
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "缺少客户端 ID",
			config: &Config{
				Email:    "test@example.com",
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					RefreshToken: "valid_refresh_token",
				},
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorType:   "validation",
		},
		{
			name: "缺少刷新令牌",
			config: &Config{
				Email:    "test@example.com",
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					ClientID: "valid_client_id",
				},
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorType:   "validation",
		},
		{
			name: "凭据为空",
			config: &Config{
				Email:    "test@example.com",
				Provider: "outlook",
				AuthType: "quick",
				Timeout:  30 * time.Second,
			},
			expectError: true,
			errorType:   "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewGraphQuickAdapter(tt.config)

			// 对于缺少必需字段的情况，期望在创建时就失败
			if tt.config.Credentials == nil ||
				tt.config.Credentials.ClientID == "" ||
				tt.config.Credentials.RefreshToken == "" {
				if err == nil {
					t.Error("期望创建适配器失败，但成功了")
				}
				return
			}

			if err != nil && !tt.expectError {
				t.Fatalf("创建适配器失败: %v", err)
			}
			if adapter == nil {
				t.Fatal("适配器为 nil")
			}

			ctx := context.Background()
			err = adapter.ValidateCredentials(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("期望出现错误，但没有错误")
					return
				}

				var connErr *ConnectionTestError
				if !errors.As(err, &connErr) {
					t.Errorf("期望 ConnectionTestError 类型，得到: %T", err)
					return
				}

				if connErr.Type != tt.errorType {
					t.Errorf("期望错误类型 %s，得到 %s", tt.errorType, connErr.Type)
				}
			} else {
				// 对于有效凭据，我们期望验证失败（因为是测试凭据）
				// 但错误类型应该是 authentication 而不是 validation
				if err != nil {
					var connErr *ConnectionTestError
					if errors.As(err, &connErr) && connErr.Type == "validation" {
						t.Errorf("不应该出现验证错误，得到: %v", err)
					}
				}
			}
		})
	}
}

// TestConnectionTestError 测试连接测试错误类型
func TestConnectionTestError(t *testing.T) {
	tests := []struct {
		name     string
		error    *ConnectionTestError
		expected string
	}{
		{
			name: "带错误码的错误",
			error: &ConnectionTestError{
				Type:    "api",
				Message: "访问被拒绝",
				Details: "Insufficient privileges",
				Code:    "Forbidden",
				Status:  403,
			},
			expected: "[api] 访问被拒绝 (code: Forbidden): Insufficient privileges",
		},
		{
			name: "不带错误码的错误",
			error: &ConnectionTestError{
				Type:    "network",
				Message: "网络连接失败",
				Details: "Connection timeout",
			},
			expected: "[network] 网络连接失败: Connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.error.Error()
			if actual != tt.expected {
				t.Errorf("期望错误消息 '%s'，得到 '%s'", tt.expected, actual)
			}
		})
	}
}

// TestGraphQuickAdapter_ConnectionTimeout 测试连接超时
func TestGraphQuickAdapter_ConnectionTimeout(t *testing.T) {
	// 创建一个会延迟响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			response := TokenResponse{
				AccessToken: "test_access_token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/v1.0/me" {
			// 延迟 2 秒响应
			time.Sleep(2 * time.Second)
			userInfo := ConnectionTestResult{
				UserID:      "test-user",
				DisplayName: "Test User",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userInfo)
		}
	}))
	defer server.Close()

	// 创建适配器
	adapter := createTestAdapter(t, server.URL)

	// 使用短超时测试
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := adapter.TestConnection(ctx)
	if err == nil {
		t.Error("期望超时错误，但没有错误")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("期望超时错误，得到: %v", err)
	}
}

// createTestAdapter 创建测试用的适配器
func createTestAdapter(t testing.TB, serverURL string) *GraphQuickAdapter {
	config := &Config{
		Email:    "test@example.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout:  30 * time.Second,
		BaseURL:  serverURL + "/v1.0",  // 使用配置字段
		TokenURL: serverURL + "/token", // 使用配置字段
	}

	adapter, err := NewGraphQuickAdapter(config)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	return adapter
}

// BenchmarkGraphQuickAdapter_TestConnection 连接测试性能基准
func BenchmarkGraphQuickAdapter_TestConnection(b *testing.B) {
	// 创建快速响应的模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			response := TokenResponse{
				AccessToken: "test_access_token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/v1.0/me" {
			userInfo := ConnectionTestResult{
				UserID:      "test-user",
				DisplayName: "Test User",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userInfo)
		}
	}))
	defer server.Close()

	adapter := createTestAdapter(b, server.URL)
	ctx := context.Background()

	// 预热
	adapter.TestConnection(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		adapter.TestConnection(ctx)
	}
}
