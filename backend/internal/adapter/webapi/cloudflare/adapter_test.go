package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fusionmail/internal/model"
)

// TestNewCloudflareTempEmailAdapter 测试适配器创建
func TestNewCloudflareTempEmailAdapter(t *testing.T) {
	t.Run("nil 配置", func(t *testing.T) {
		_, err := NewCloudflareTempEmailAdapter(nil)
		if err == nil {
			t.Error("nil 配置应返回错误")
		}
	})

	t.Run("空配置", func(t *testing.T) {
		config := &model.CloudflareTempEmailAuthData{}
		_, err := NewCloudflareTempEmailAdapter(config)
		if err == nil {
			t.Error("空配置应返回错误")
		}
	})

	t.Run("Single 模式有效配置", func(t *testing.T) {
		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    "https://temp-email.example.com",
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "test@example.com",
		}
		adapter, err := NewCloudflareTempEmailAdapter(config)
		if err != nil {
			t.Errorf("创建适配器失败: %v", err)
		}
		if adapter == nil {
			t.Error("适配器不应为 nil")
		}
	})

	t.Run("Admin 模式有效配置", func(t *testing.T) {
		config := &model.CloudflareTempEmailAuthData{
			BaseURL:       "https://temp-email.example.com",
			AccessMode:    model.WebAPIAccessModeAdmin,
			AdminPassword: "admin-password",
		}
		adapter, err := NewCloudflareTempEmailAdapter(config)
		if err != nil {
			t.Errorf("创建适配器失败: %v", err)
		}
		if adapter == nil {
			t.Error("适配器不应为 nil")
		}
	})

	t.Run("Single 模式缺少 JWT Token", func(t *testing.T) {
		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    "https://temp-email.example.com",
			AccessMode: model.WebAPIAccessModeSingle,
			Email:      "test@example.com",
		}
		_, err := NewCloudflareTempEmailAdapter(config)
		if err == nil {
			t.Error("缺少 JWT Token 应返回错误")
		}
	})

	t.Run("Admin 模式缺少密码", func(t *testing.T) {
		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    "https://temp-email.example.com",
			AccessMode: model.WebAPIAccessModeAdmin,
		}
		_, err := NewCloudflareTempEmailAdapter(config)
		if err == nil {
			t.Error("缺少管理员密码应返回错误")
		}
	})
}

// TestCloudflareTempEmailAdapter_GetProviderType 测试获取提供商类型
func TestCloudflareTempEmailAdapter_GetProviderType(t *testing.T) {
	config := &model.CloudflareTempEmailAuthData{
		BaseURL:    "https://temp-email.example.com",
		AccessMode: model.WebAPIAccessModeSingle,
		JWTToken:   "test-jwt-token",
		Email:      "test@example.com",
	}
	adapter, _ := NewCloudflareTempEmailAdapter(config)

	if adapter.GetProviderType() != model.WebAPIServiceTypeCloudflareTempEmail {
		t.Errorf("GetProviderType() = %q, want %q", adapter.GetProviderType(), model.WebAPIServiceTypeCloudflareTempEmail)
	}
}

// TestCloudflareTempEmailAdapter_GetProtocol 测试获取协议类型
func TestCloudflareTempEmailAdapter_GetProtocol(t *testing.T) {
	config := &model.CloudflareTempEmailAuthData{
		BaseURL:    "https://temp-email.example.com",
		AccessMode: model.WebAPIAccessModeSingle,
		JWTToken:   "test-jwt-token",
		Email:      "test@example.com",
	}
	adapter, _ := NewCloudflareTempEmailAdapter(config)

	if adapter.GetProtocol() != "webapi" {
		t.Errorf("GetProtocol() = %q, want %q", adapter.GetProtocol(), "webapi")
	}
}

// TestCloudflareTempEmailAdapter_TestConnection 测试连接
func TestCloudflareTempEmailAdapter_TestConnection(t *testing.T) {
	t.Run("Single 模式连接成功", func(t *testing.T) {
		// 创建模拟服务器
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 验证请求路径
			if r.URL.Path != "/api/mails" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}
			// 验证认证头
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-jwt-token" {
				t.Errorf("认证头错误: %s", auth)
			}
			// 返回成功响应
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(SingleModeResponse{Results: []EmailItem{}})
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "test@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)

		err := adapter.TestConnection(context.Background())
		if err != nil {
			t.Errorf("TestConnection 失败: %v", err)
		}
	})

	t.Run("Admin 模式连接成功", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/admin/mails" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}
			auth := r.Header.Get("x-admin-auth")
			if auth != "admin-password" {
				t.Errorf("认证头错误: %s", auth)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(AdminModeResponse{Results: []AdminEmailItem{}})
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:       server.URL,
			AccessMode:    model.WebAPIAccessModeAdmin,
			AdminPassword: "admin-password",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)

		err := adapter.TestConnection(context.Background())
		if err != nil {
			t.Errorf("TestConnection 失败: %v", err)
		}
	})

	t.Run("认证失败", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "invalid-token",
			Email:      "test@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)

		err := adapter.TestConnection(context.Background())
		if err == nil {
			t.Error("认证失败应返回错误")
		}
	})

	t.Run("服务器错误", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "test@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)

		err := adapter.TestConnection(context.Background())
		if err == nil {
			t.Error("服务器错误应返回错误")
		}
	})
}

// TestCloudflareTempEmailAdapter_FetchEmails_Single 测试 Single 模式拉取邮件
func TestCloudflareTempEmailAdapter_FetchEmails_Single(t *testing.T) {
	t.Run("成功拉取邮件", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := SingleModeResponse{
				Results: []EmailItem{
					{
						ID:        "email-1",
						MessageID: "<msg1@example.com>",
						Subject:   "测试邮件 1",
						From:      "sender@example.com",
						FromName:  "发件人",
						TextBody:  "这是测试邮件内容",
						CreatedAt: time.Now().Format(time.RFC3339),
					},
					{
						ID:        "email-2",
						MessageID: "<msg2@example.com>",
						Subject:   "测试邮件 2",
						From:      "sender2@example.com",
						TextBody:  "这是第二封测试邮件",
						CreatedAt: time.Now().Format(time.RFC3339),
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "target@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)
		adapter.SetConnected(true)

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err != nil {
			t.Errorf("FetchEmails 失败: %v", err)
		}

		if len(emails) != 2 {
			t.Errorf("邮件数量 = %d, want 2", len(emails))
		}

		// 验证第一封邮件
		if emails[0].ProviderID != "email-1" {
			t.Errorf("ProviderID = %q, want %q", emails[0].ProviderID, "email-1")
		}
		if emails[0].Subject != "测试邮件 1" {
			t.Errorf("Subject = %q, want %q", emails[0].Subject, "测试邮件 1")
		}
		if emails[0].FromAddress != "sender@example.com" {
			t.Errorf("FromAddress = %q, want %q", emails[0].FromAddress, "sender@example.com")
		}
		// 验证目标地址
		if len(emails[0].ToAddresses) == 0 || emails[0].ToAddresses[0] != "target@example.com" {
			t.Errorf("ToAddresses = %v, want [target@example.com]", emails[0].ToAddresses)
		}
	})

	t.Run("未连接时拉取", func(t *testing.T) {
		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    "https://temp-email.example.com",
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "test@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)
		// 不调用 Connect，保持未连接状态

		_, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err == nil {
			t.Error("未连接时应返回错误")
		}
	})

	t.Run("时间过滤", func(t *testing.T) {
		now := time.Now()
		oldTime := now.Add(-24 * time.Hour)
		newTime := now.Add(-1 * time.Hour)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := SingleModeResponse{
				Results: []EmailItem{
					{
						ID:        "old-email",
						Subject:   "旧邮件",
						From:      "sender@example.com",
						CreatedAt: oldTime.Format(time.RFC3339),
					},
					{
						ID:        "new-email",
						Subject:   "新邮件",
						From:      "sender@example.com",
						CreatedAt: newTime.Format(time.RFC3339),
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "target@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)
		adapter.SetConnected(true)

		// 只获取 12 小时内的邮件
		since := now.Add(-12 * time.Hour)
		emails, err := adapter.FetchEmails(context.Background(), since, 10)
		if err != nil {
			t.Errorf("FetchEmails 失败: %v", err)
		}

		// 应该只返回新邮件
		if len(emails) != 1 {
			t.Errorf("邮件数量 = %d, want 1", len(emails))
		}
		if len(emails) > 0 && emails[0].ProviderID != "new-email" {
			t.Errorf("ProviderID = %q, want %q", emails[0].ProviderID, "new-email")
		}
	})
}

// TestCloudflareTempEmailAdapter_FetchEmails_Admin 测试 Admin 模式拉取邮件
func TestCloudflareTempEmailAdapter_FetchEmails_Admin(t *testing.T) {
	t.Run("成功拉取邮件", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 验证认证头
			auth := r.Header.Get("x-admin-auth")
			if auth != "admin-password" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			response := AdminModeResponse{
				Results: []AdminEmailItem{
					{
						EmailItem: EmailItem{
							ID:        "email-1",
							Subject:   "邮件给用户1",
							From:      "sender@example.com",
							TextBody:  "内容1",
							CreatedAt: time.Now().Format(time.RFC3339),
						},
						Address: "user1@domain.com",
					},
					{
						EmailItem: EmailItem{
							ID:        "email-2",
							Subject:   "邮件给用户2",
							From:      "sender@example.com",
							TextBody:  "内容2",
							CreatedAt: time.Now().Format(time.RFC3339),
						},
						Address: "user2@domain.com",
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:       server.URL,
			AccessMode:    model.WebAPIAccessModeAdmin,
			AdminPassword: "admin-password",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)
		adapter.SetConnected(true)

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err != nil {
			t.Errorf("FetchEmails 失败: %v", err)
		}

		if len(emails) != 2 {
			t.Errorf("邮件数量 = %d, want 2", len(emails))
		}

		// 验证目标地址从响应中提取
		if len(emails[0].ToAddresses) == 0 || emails[0].ToAddresses[0] != "user1@domain.com" {
			t.Errorf("ToAddresses = %v, want [user1@domain.com]", emails[0].ToAddresses)
		}
		if len(emails[1].ToAddresses) == 0 || emails[1].ToAddresses[0] != "user2@domain.com" {
			t.Errorf("ToAddresses = %v, want [user2@domain.com]", emails[1].ToAddresses)
		}
	})
}

// TestCloudflareTempEmailAdapter_FetchEmailDetail 测试获取邮件详情
func TestCloudflareTempEmailAdapter_FetchEmailDetail(t *testing.T) {
	t.Run("Single 模式获取详情", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/mails/email-123" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}

			response := EmailItem{
				ID:        "email-123",
				MessageID: "<msg123@example.com>",
				Subject:   "详情测试邮件",
				From:      "sender@example.com",
				FromName:  "发件人",
				TextBody:  "这是邮件详情内容",
				HTMLBody:  "<p>这是邮件详情内容</p>",
				CreatedAt: time.Now().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "target@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)
		adapter.SetConnected(true)

		email, err := adapter.FetchEmailDetail(context.Background(), "email-123")
		if err != nil {
			t.Errorf("FetchEmailDetail 失败: %v", err)
		}

		if email.ProviderID != "email-123" {
			t.Errorf("ProviderID = %q, want %q", email.ProviderID, "email-123")
		}
		if email.Subject != "详情测试邮件" {
			t.Errorf("Subject = %q, want %q", email.Subject, "详情测试邮件")
		}
		if email.HTMLBody != "<p>这是邮件详情内容</p>" {
			t.Errorf("HTMLBody = %q", email.HTMLBody)
		}
	})

	t.Run("邮件不存在", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "target@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)
		adapter.SetConnected(true)

		_, err := adapter.FetchEmailDetail(context.Background(), "not-exist")
		if err == nil {
			t.Error("邮件不存在应返回错误")
		}
	})

	t.Run("未连接时获取详情", func(t *testing.T) {
		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    "https://temp-email.example.com",
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "test@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)

		_, err := adapter.FetchEmailDetail(context.Background(), "email-123")
		if err == nil {
			t.Error("未连接时应返回错误")
		}
	})
}

// TestCloudflareTempEmailAdapter_Connect 测试连接
func TestCloudflareTempEmailAdapter_Connect(t *testing.T) {
	t.Run("连接成功", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(SingleModeResponse{Results: []EmailItem{}})
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "test-jwt-token",
			Email:      "test@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)

		err := adapter.Connect(context.Background())
		if err != nil {
			t.Errorf("Connect 失败: %v", err)
		}

		if !adapter.IsConnected() {
			t.Error("连接后 IsConnected() 应返回 true")
		}
	})

	t.Run("连接失败", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		config := &model.CloudflareTempEmailAuthData{
			BaseURL:    server.URL,
			AccessMode: model.WebAPIAccessModeSingle,
			JWTToken:   "invalid-token",
			Email:      "test@example.com",
		}
		adapter, _ := NewCloudflareTempEmailAdapter(config)

		err := adapter.Connect(context.Background())
		if err == nil {
			t.Error("连接失败应返回错误")
		}

		if adapter.IsConnected() {
			t.Error("连接失败后 IsConnected() 应返回 false")
		}
	})
}

// TestCloudflareTempEmailAdapter_Disconnect 测试断开连接
func TestCloudflareTempEmailAdapter_Disconnect(t *testing.T) {
	config := &model.CloudflareTempEmailAuthData{
		BaseURL:    "https://temp-email.example.com",
		AccessMode: model.WebAPIAccessModeSingle,
		JWTToken:   "test-jwt-token",
		Email:      "test@example.com",
	}
	adapter, _ := NewCloudflareTempEmailAdapter(config)
	adapter.SetConnected(true)

	err := adapter.Disconnect()
	if err != nil {
		t.Errorf("Disconnect 失败: %v", err)
	}

	if adapter.IsConnected() {
		t.Error("断开连接后 IsConnected() 应返回 false")
	}
}

// TestCloudflareTempEmailAdapter_ParseEmailWithRFC822 测试 RFC822 解析
func TestCloudflareTempEmailAdapter_ParseEmailWithRFC822(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回包含 RFC822 原始内容的响应
		response := SingleModeResponse{
			Results: []EmailItem{
				{
					ID: "email-rfc822",
					Raw: `From: sender@example.com
To: target@example.com
Subject: RFC822 Test
Date: Mon, 30 Dec 2024 10:00:00 +0800
Content-Type: text/plain; charset=utf-8

This is the email body.`,
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &model.CloudflareTempEmailAuthData{
		BaseURL:    server.URL,
		AccessMode: model.WebAPIAccessModeSingle,
		JWTToken:   "test-jwt-token",
		Email:      "target@example.com",
	}
	adapter, _ := NewCloudflareTempEmailAdapter(config)
	adapter.SetConnected(true)

	emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
	if err != nil {
		t.Errorf("FetchEmails 失败: %v", err)
	}

	if len(emails) != 1 {
		t.Fatalf("邮件数量 = %d, want 1", len(emails))
	}

	email := emails[0]
	if email.Subject != "RFC822 Test" {
		t.Errorf("Subject = %q, want %q", email.Subject, "RFC822 Test")
	}
	if email.FromAddress != "sender@example.com" {
		t.Errorf("FromAddress = %q, want %q", email.FromAddress, "sender@example.com")
	}
	if email.TextBody != "This is the email body." {
		t.Errorf("TextBody = %q", email.TextBody)
	}
}

// TestCloudflareTempEmailAdapter_GetConfig 测试获取配置
func TestCloudflareTempEmailAdapter_GetConfig(t *testing.T) {
	config := &model.CloudflareTempEmailAuthData{
		BaseURL:    "https://temp-email.example.com",
		AccessMode: model.WebAPIAccessModeSingle,
		JWTToken:   "test-jwt-token",
		Email:      "test@example.com",
	}
	adapter, _ := NewCloudflareTempEmailAdapter(config)

	gotConfig := adapter.GetConfig()
	if gotConfig != config {
		t.Error("GetConfig() 应返回原始配置")
	}
}
