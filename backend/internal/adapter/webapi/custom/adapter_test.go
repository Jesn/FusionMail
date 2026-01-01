package custom

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fusionmail/internal/model"
)

// defaultFieldMapping 返回默认的字段映射配置
func defaultFieldMapping() model.CustomWebAPIFieldMapping {
	return model.CustomWebAPIFieldMapping{
		ID:      "id",
		Subject: "subject",
		From:    "from",
		Date:    "received_at",
		Body:    "text",
	}
}

// createTestConfig 创建测试配置
func createTestConfig(baseURL string) *model.CustomWebAPIAuthData {
	return &model.CustomWebAPIAuthData{
		BaseURL:      baseURL,
		ServiceName:  "TestService",
		ListEndpoint: "/api/mails",
		DataPath:     "data",
		Auth: model.CustomWebAPIAuthConfig{
			Type:  model.WebAPIAuthTypeJWT,
			Token: "test-jwt-token",
		},
		FieldMapping: defaultFieldMapping(),
	}
}

// createTLSTestServer 创建 TLS 测试服务器
func createTLSTestServer(handler http.Handler) *httptest.Server {
	server := httptest.NewTLSServer(handler)
	return server
}

// createTLSHTTPClient 创建跳过证书验证的 HTTP 客户端
func createTLSHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// TestNewCustomWebAPIAdapter 测试适配器创建
func TestNewCustomWebAPIAdapter(t *testing.T) {
	t.Run("nil 配置", func(t *testing.T) {
		_, err := NewCustomWebAPIAdapter(nil)
		if err == nil {
			t.Error("nil 配置应返回错误")
		}
	})

	t.Run("空配置", func(t *testing.T) {
		config := &model.CustomWebAPIAuthData{}
		_, err := NewCustomWebAPIAdapter(config)
		if err == nil {
			t.Error("空配置应返回错误")
		}
	})

	t.Run("有效配置 - JWT 认证", func(t *testing.T) {
		config := createTestConfig("https://api.example.com")
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Errorf("创建适配器失败: %v", err)
		}
		if adapter == nil {
			t.Error("适配器不应为 nil")
		}
	})

	t.Run("有效配置 - API Key 认证", func(t *testing.T) {
		config := &model.CustomWebAPIAuthData{
			BaseURL:      "https://api.example.com",
			ServiceName:  "TestService",
			ListEndpoint: "/api/mails",
			Auth: model.CustomWebAPIAuthConfig{
				Type:       model.WebAPIAuthTypeAPIKey,
				APIKey:     "test-api-key",
				APIKeyName: "X-API-Key",
			},
			FieldMapping: defaultFieldMapping(),
		}
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Errorf("创建适配器失败: %v", err)
		}
		if adapter == nil {
			t.Error("适配器不应为 nil")
		}
	})

	t.Run("有效配置 - Basic 认证", func(t *testing.T) {
		config := &model.CustomWebAPIAuthData{
			BaseURL:      "https://api.example.com",
			ServiceName:  "TestService",
			ListEndpoint: "/api/mails",
			Auth: model.CustomWebAPIAuthConfig{
				Type:     model.WebAPIAuthTypeBasic,
				Username: "user",
				Password: "pass",
			},
			FieldMapping: defaultFieldMapping(),
		}
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Errorf("创建适配器失败: %v", err)
		}
		if adapter == nil {
			t.Error("适配器不应为 nil")
		}
	})

	t.Run("缺少 BaseURL", func(t *testing.T) {
		config := &model.CustomWebAPIAuthData{
			ServiceName:  "TestService",
			ListEndpoint: "/api/mails",
			Auth: model.CustomWebAPIAuthConfig{
				Type:  model.WebAPIAuthTypeJWT,
				Token: "test-jwt-token",
			},
			FieldMapping: defaultFieldMapping(),
		}
		_, err := NewCustomWebAPIAdapter(config)
		if err == nil {
			t.Error("缺少 BaseURL 应返回错误")
		}
	})

	t.Run("缺少 ListEndpoint", func(t *testing.T) {
		config := &model.CustomWebAPIAuthData{
			BaseURL:     "https://api.example.com",
			ServiceName: "TestService",
			Auth: model.CustomWebAPIAuthConfig{
				Type:  model.WebAPIAuthTypeJWT,
				Token: "test-jwt-token",
			},
			FieldMapping: defaultFieldMapping(),
		}
		_, err := NewCustomWebAPIAdapter(config)
		if err == nil {
			t.Error("缺少 ListEndpoint 应返回错误")
		}
	})
}

// TestCustomWebAPIAdapter_GetProviderType 测试获取提供商类型
func TestCustomWebAPIAdapter_GetProviderType(t *testing.T) {
	config := createTestConfig("https://api.example.com")
	adapter, _ := NewCustomWebAPIAdapter(config)

	if adapter.GetProviderType() != model.WebAPIServiceTypeCustom {
		t.Errorf("GetProviderType() = %q, want %q", adapter.GetProviderType(), model.WebAPIServiceTypeCustom)
	}
}

// TestCustomWebAPIAdapter_GetProtocol 测试获取协议类型
func TestCustomWebAPIAdapter_GetProtocol(t *testing.T) {
	config := createTestConfig("https://api.example.com")
	adapter, _ := NewCustomWebAPIAdapter(config)

	if adapter.GetProtocol() != "webapi" {
		t.Errorf("GetProtocol() = %q, want %q", adapter.GetProtocol(), "webapi")
	}
}

// TestCustomWebAPIAdapter_TestConnection 测试连接
func TestCustomWebAPIAdapter_TestConnection(t *testing.T) {
	t.Run("JWT 认证连接成功", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-jwt-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())

		err = adapter.TestConnection(context.Background())
		if err != nil {
			t.Errorf("TestConnection 失败: %v", err)
		}
	})

	t.Run("API Key 认证连接成功", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "test-api-key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}))
		defer server.Close()

		config := &model.CustomWebAPIAuthData{
			BaseURL:      server.URL,
			ServiceName:  "TestService",
			ListEndpoint: "/api/mails",
			Auth: model.CustomWebAPIAuthConfig{
				Type:       model.WebAPIAuthTypeAPIKey,
				APIKey:     "test-api-key",
				APIKeyName: "X-API-Key",
			},
			FieldMapping: defaultFieldMapping(),
		}
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())

		err = adapter.TestConnection(context.Background())
		if err != nil {
			t.Errorf("TestConnection 失败: %v", err)
		}
	})

	t.Run("Basic 认证连接成功", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Basic dXNlcjpwYXNz" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}))
		defer server.Close()

		config := &model.CustomWebAPIAuthData{
			BaseURL:      server.URL,
			ServiceName:  "TestService",
			ListEndpoint: "/api/mails",
			Auth: model.CustomWebAPIAuthConfig{
				Type:     model.WebAPIAuthTypeBasic,
				Username: "user",
				Password: "pass",
			},
			FieldMapping: defaultFieldMapping(),
		}
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())

		err = adapter.TestConnection(context.Background())
		if err != nil {
			t.Errorf("TestConnection 失败: %v", err)
		}
	})

	t.Run("认证失败", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())

		err = adapter.TestConnection(context.Background())
		if err == nil {
			t.Error("认证失败应返回错误")
		}
	})

	t.Run("服务器错误", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())

		err = adapter.TestConnection(context.Background())
		if err == nil {
			t.Error("服务器错误应返回错误")
		}
	})
}

// TestCustomWebAPIAdapter_FetchEmails 测试拉取邮件
func TestCustomWebAPIAdapter_FetchEmails(t *testing.T) {
	t.Run("成功拉取邮件", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"data": []map[string]any{
					{
						"id":          "email-1",
						"subject":     "测试邮件 1",
						"from":        "sender@example.com",
						"text":        "邮件内容 1",
						"received_at": time.Now().Format(time.RFC3339),
					},
					{
						"id":          "email-2",
						"subject":     "测试邮件 2",
						"from":        "sender2@example.com",
						"text":        "邮件内容 2",
						"received_at": time.Now().Format(time.RFC3339),
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())
		adapter.SetConnected(true)

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err != nil {
			t.Errorf("FetchEmails 失败: %v", err)
		}

		if len(emails) != 2 {
			t.Errorf("邮件数量 = %d, want 2", len(emails))
		}

		if emails[0].ProviderID != "email-1" {
			t.Errorf("ProviderID = %q, want %q", emails[0].ProviderID, "email-1")
		}
		if emails[0].Subject != "测试邮件 1" {
			t.Errorf("Subject = %q, want %q", emails[0].Subject, "测试邮件 1")
		}
	})

	t.Run("未连接时拉取", func(t *testing.T) {
		config := createTestConfig("https://api.example.com")
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		_, err = adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err == nil {
			t.Error("未连接时应返回错误")
		}
	})

	t.Run("时间过滤", func(t *testing.T) {
		now := time.Now()
		oldTime := now.Add(-24 * time.Hour)
		newTime := now.Add(-1 * time.Hour)

		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"data": []map[string]any{
					{
						"id":          "old-email",
						"subject":     "旧邮件",
						"from":        "sender@example.com",
						"received_at": oldTime.Format(time.RFC3339),
					},
					{
						"id":          "new-email",
						"subject":     "新邮件",
						"from":        "sender@example.com",
						"received_at": newTime.Format(time.RFC3339),
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())
		adapter.SetConnected(true)

		since := now.Add(-12 * time.Hour)
		emails, err := adapter.FetchEmails(context.Background(), since, 10)
		if err != nil {
			t.Errorf("FetchEmails 失败: %v", err)
		}

		if len(emails) != 1 {
			t.Errorf("邮件数量 = %d, want 1", len(emails))
		}
		if len(emails) > 0 && emails[0].ProviderID != "new-email" {
			t.Errorf("ProviderID = %q, want %q", emails[0].ProviderID, "new-email")
		}
	})

	t.Run("数量限制", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"data": []map[string]any{
					{"id": "email-1", "subject": "邮件 1", "from": "sender@example.com", "received_at": time.Now().Format(time.RFC3339)},
					{"id": "email-2", "subject": "邮件 2", "from": "sender@example.com", "received_at": time.Now().Format(time.RFC3339)},
					{"id": "email-3", "subject": "邮件 3", "from": "sender@example.com", "received_at": time.Now().Format(time.RFC3339)},
					{"id": "email-4", "subject": "邮件 4", "from": "sender@example.com", "received_at": time.Now().Format(time.RFC3339)},
					{"id": "email-5", "subject": "邮件 5", "from": "sender@example.com", "received_at": time.Now().Format(time.RFC3339)},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())
		adapter.SetConnected(true)

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 3)
		if err != nil {
			t.Errorf("FetchEmails 失败: %v", err)
		}

		if len(emails) != 3 {
			t.Errorf("邮件数量 = %d, want 3", len(emails))
		}
	})
}

// TestCustomWebAPIAdapter_FetchEmailDetail 测试获取邮件详情
func TestCustomWebAPIAdapter_FetchEmailDetail(t *testing.T) {
	t.Run("成功获取详情", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/mails/email-123" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}

			// 返回数组格式，与解析器配置的 DataPath 一致
			response := map[string]any{
				"data": []map[string]any{
					{
						"id":          "email-123",
						"subject":     "详情测试邮件",
						"from":        "sender@example.com",
						"text":        "这是邮件详情内容",
						"received_at": time.Now().Format(time.RFC3339),
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		config.DetailEndpoint = "/api/mails/{id}"
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())
		adapter.SetConnected(true)

		email, err := adapter.FetchEmailDetail(context.Background(), "email-123")
		if err != nil {
			t.Errorf("FetchEmailDetail 失败: %v", err)
			return
		}

		if email.ProviderID != "email-123" {
			t.Errorf("ProviderID = %q, want %q", email.ProviderID, "email-123")
		}
		if email.Subject != "详情测试邮件" {
			t.Errorf("Subject = %q, want %q", email.Subject, "详情测试邮件")
		}
	})

	t.Run("未配置详情端点", func(t *testing.T) {
		config := createTestConfig("https://api.example.com")
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetConnected(true)

		_, err = adapter.FetchEmailDetail(context.Background(), "email-123")
		if err == nil {
			t.Error("未配置详情端点应返回错误")
		}
	})

	t.Run("邮件不存在", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		config.DetailEndpoint = "/api/mails/{id}"
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())
		adapter.SetConnected(true)

		_, err = adapter.FetchEmailDetail(context.Background(), "not-exist")
		if err == nil {
			t.Error("邮件不存在应返回错误")
		}
	})

	t.Run("未连接时获取详情", func(t *testing.T) {
		config := createTestConfig("https://api.example.com")
		config.DetailEndpoint = "/api/mails/{id}"
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		_, err = adapter.FetchEmailDetail(context.Background(), "email-123")
		if err == nil {
			t.Error("未连接时应返回错误")
		}
	})
}

// TestCustomWebAPIAdapter_Connect 测试连接
func TestCustomWebAPIAdapter_Connect(t *testing.T) {
	t.Run("连接成功", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())

		err = adapter.Connect(context.Background())
		if err != nil {
			t.Errorf("Connect 失败: %v", err)
		}

		if !adapter.IsConnected() {
			t.Error("连接后 IsConnected() 应返回 true")
		}
	})

	t.Run("连接失败", func(t *testing.T) {
		server := createTLSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		config := createTestConfig(server.URL)
		adapter, err := NewCustomWebAPIAdapter(config)
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		adapter.SetHTTPClient(createTLSHTTPClient())

		err = adapter.Connect(context.Background())
		if err == nil {
			t.Error("连接失败应返回错误")
		}

		if adapter.IsConnected() {
			t.Error("连接失败后 IsConnected() 应返回 false")
		}
	})
}

// TestCustomWebAPIAdapter_Disconnect 测试断开连接
func TestCustomWebAPIAdapter_Disconnect(t *testing.T) {
	config := createTestConfig("https://api.example.com")
	adapter, err := NewCustomWebAPIAdapter(config)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}
	adapter.SetConnected(true)

	err = adapter.Disconnect()
	if err != nil {
		t.Errorf("Disconnect 失败: %v", err)
	}

	if adapter.IsConnected() {
		t.Error("断开连接后 IsConnected() 应返回 false")
	}
}

// TestCustomWebAPIAdapter_GetConfig 测试获取配置
func TestCustomWebAPIAdapter_GetConfig(t *testing.T) {
	config := createTestConfig("https://api.example.com")
	adapter, err := NewCustomWebAPIAdapter(config)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	gotConfig := adapter.GetConfig()
	if gotConfig != config {
		t.Error("GetConfig() 应返回原始配置")
	}
}

// TestReplacePathParam 测试路径参数替换
func TestReplacePathParam(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		param    string
		value    string
		expected string
	}{
		{
			name:     "花括号格式",
			path:     "/api/mails/{id}",
			param:    "id",
			value:    "email-123",
			expected: "/api/mails/email-123",
		},
		{
			name:     "冒号格式",
			path:     "/api/mails/:id",
			param:    "id",
			value:    "email-123",
			expected: "/api/mails/email-123",
		},
		{
			name:     "多个参数",
			path:     "/api/accounts/{account_id}/mails/{id}",
			param:    "id",
			value:    "email-123",
			expected: "/api/accounts/{account_id}/mails/email-123",
		},
		{
			name:     "无匹配",
			path:     "/api/mails/list",
			param:    "id",
			value:    "email-123",
			expected: "/api/mails/list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replacePathParam(tt.path, tt.param, tt.value)
			if result != tt.expected {
				t.Errorf("replacePathParam() = %q, want %q", result, tt.expected)
			}
		})
	}
}
