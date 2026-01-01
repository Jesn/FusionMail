package cloudmail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fusionmail/internal/model"
)

// TestNewCloudMailAdapter 测试适配器创建
func TestNewCloudMailAdapter(t *testing.T) {
	t.Run("nil 配置", func(t *testing.T) {
		_, err := NewCloudMailAdapter(nil)
		if err == nil {
			t.Error("nil 配置应返回错误")
		}
	})

	t.Run("空配置", func(t *testing.T) {
		config := &model.CloudMailAuthData{}
		_, err := NewCloudMailAdapter(config)
		if err == nil {
			t.Error("空配置应返回错误")
		}
	})

	t.Run("有效配置", func(t *testing.T) {
		config := &model.CloudMailAuthData{
			BaseURL:  "https://cloudmail.example.com",
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
				{Email: "user2@example.com"},
			},
		}
		adapter, err := NewCloudMailAdapter(config)
		if err != nil {
			t.Errorf("创建适配器失败: %v", err)
		}
		if adapter == nil {
			t.Error("适配器不应为 nil")
		}
	})

	t.Run("缺少 JWT Token", func(t *testing.T) {
		config := &model.CloudMailAuthData{
			BaseURL: "https://cloudmail.example.com",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		_, err := NewCloudMailAdapter(config)
		if err == nil {
			t.Error("缺少 JWT Token 应返回错误")
		}
	})

	t.Run("缺少账户列表", func(t *testing.T) {
		config := &model.CloudMailAuthData{
			BaseURL:  "https://cloudmail.example.com",
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{},
		}
		_, err := NewCloudMailAdapter(config)
		if err == nil {
			t.Error("缺少账户列表应返回错误")
		}
	})
}

// TestCloudMailAdapter_GetProviderType 测试获取提供商类型
func TestCloudMailAdapter_GetProviderType(t *testing.T) {
	config := &model.CloudMailAuthData{
		BaseURL:  "https://cloudmail.example.com",
		JWTToken: "test-jwt-token",
		Accounts: []model.CloudMailAccount{
			{Email: "user1@example.com"},
		},
	}
	adapter, _ := NewCloudMailAdapter(config)

	if adapter.GetProviderType() != model.WebAPIServiceTypeCloudMail {
		t.Errorf("GetProviderType() = %q, want %q", adapter.GetProviderType(), model.WebAPIServiceTypeCloudMail)
	}
}

// TestCloudMailAdapter_GetProtocol 测试获取协议类型
func TestCloudMailAdapter_GetProtocol(t *testing.T) {
	config := &model.CloudMailAuthData{
		BaseURL:  "https://cloudmail.example.com",
		JWTToken: "test-jwt-token",
		Accounts: []model.CloudMailAccount{
			{Email: "user1@example.com"},
		},
	}
	adapter, _ := NewCloudMailAdapter(config)

	if adapter.GetProtocol() != "webapi" {
		t.Errorf("GetProtocol() = %q, want %q", adapter.GetProtocol(), "webapi")
	}
}

// TestCloudMailAdapter_TestConnection 测试连接
func TestCloudMailAdapter_TestConnection(t *testing.T) {
	t.Run("连接成功", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 验证认证头
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-jwt-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MailListResponse{Data: []MailItem{}})
		}))
		defer server.Close()

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)

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

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "invalid-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)

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

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)

		err := adapter.TestConnection(context.Background())
		if err == nil {
			t.Error("服务器错误应返回错误")
		}
	})
}

// TestCloudMailAdapter_FetchEmails 测试拉取邮件
func TestCloudMailAdapter_FetchEmails(t *testing.T) {
	t.Run("成功拉取多账户邮件", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			account := r.URL.Query().Get("account")

			var response MailListResponse
			switch account {
			case "user1@example.com":
				response = MailListResponse{
					Data: []MailItem{
						{
							ID:         "email-1",
							Subject:    "用户1的邮件",
							From:       "sender@example.com",
							ReceivedAt: time.Now().Format(time.RFC3339),
						},
					},
				}
			case "user2@example.com":
				response = MailListResponse{
					Data: []MailItem{
						{
							ID:         "email-2",
							Subject:    "用户2的邮件",
							From:       "sender@example.com",
							ReceivedAt: time.Now().Format(time.RFC3339),
						},
						{
							ID:         "email-3",
							Subject:    "用户2的第二封邮件",
							From:       "sender2@example.com",
							ReceivedAt: time.Now().Format(time.RFC3339),
						},
					},
				}
			default:
				response = MailListResponse{Data: []MailItem{}}
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
				{Email: "user2@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)
		adapter.SetConnected(true)

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err != nil {
			t.Errorf("FetchEmails 失败: %v", err)
		}

		// 应该有 3 封邮件（用户1: 1封，用户2: 2封）
		if len(emails) != 3 {
			t.Errorf("邮件数量 = %d, want 3", len(emails))
		}

		// 验证目标地址正确设置
		foundUser1 := false
		foundUser2 := false
		for _, email := range emails {
			if len(email.ToAddresses) > 0 {
				if email.ToAddresses[0] == "user1@example.com" {
					foundUser1 = true
				}
				if email.ToAddresses[0] == "user2@example.com" {
					foundUser2 = true
				}
			}
		}
		if !foundUser1 {
			t.Error("未找到 user1@example.com 的邮件")
		}
		if !foundUser2 {
			t.Error("未找到 user2@example.com 的邮件")
		}
	})

	t.Run("未连接时拉取", func(t *testing.T) {
		config := &model.CloudMailAuthData{
			BaseURL:  "https://cloudmail.example.com",
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)
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
			response := MailListResponse{
				Data: []MailItem{
					{
						ID:         "old-email",
						Subject:    "旧邮件",
						From:       "sender@example.com",
						ReceivedAt: oldTime.Format(time.RFC3339),
					},
					{
						ID:         "new-email",
						Subject:    "新邮件",
						From:       "sender@example.com",
						ReceivedAt: newTime.Format(time.RFC3339),
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)
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

	t.Run("部分账户失败继续处理", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			account := r.URL.Query().Get("account")

			// 第一个账户返回错误
			if account == "user1@example.com" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// 第二个账户正常返回
			response := MailListResponse{
				Data: []MailItem{
					{
						ID:         "email-from-user2",
						Subject:    "用户2的邮件",
						From:       "sender@example.com",
						ReceivedAt: time.Now().Format(time.RFC3339),
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
				{Email: "user2@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)
		adapter.SetConnected(true)

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		// 即使部分账户失败，也不应返回错误
		if err != nil {
			t.Errorf("FetchEmails 不应返回错误: %v", err)
		}

		// 应该只有第二个账户的邮件
		if len(emails) != 1 {
			t.Errorf("邮件数量 = %d, want 1", len(emails))
		}

		// 验证两个账户都被请求
		if requestCount != 2 {
			t.Errorf("请求次数 = %d, want 2", requestCount)
		}
	})
}

// TestCloudMailAdapter_FetchEmailDetail 测试获取邮件详情
func TestCloudMailAdapter_FetchEmailDetail(t *testing.T) {
	t.Run("成功获取详情", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/mails/email-123" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}

			response := MailItem{
				ID:               "email-123",
				MessageID:        "<msg123@example.com>",
				Subject:          "详情测试邮件",
				From:             "sender@example.com",
				FromName:         "发件人",
				ToAddresses:      []string{"target@example.com"},
				TextBody:         "这是邮件详情内容",
				HTMLBody:         "<p>这是邮件详情内容</p>",
				ReceivedAt:       time.Now().Format(time.RFC3339),
				HasAttachments:   true,
				AttachmentsCount: 2,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)
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
		if !email.HasAttachments {
			t.Error("HasAttachments 应为 true")
		}
		if email.AttachmentsCount != 2 {
			t.Errorf("AttachmentsCount = %d, want 2", email.AttachmentsCount)
		}
	})

	t.Run("邮件不存在", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)
		adapter.SetConnected(true)

		_, err := adapter.FetchEmailDetail(context.Background(), "not-exist")
		if err == nil {
			t.Error("邮件不存在应返回错误")
		}
	})

	t.Run("未连接时获取详情", func(t *testing.T) {
		config := &model.CloudMailAuthData{
			BaseURL:  "https://cloudmail.example.com",
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)

		_, err := adapter.FetchEmailDetail(context.Background(), "email-123")
		if err == nil {
			t.Error("未连接时应返回错误")
		}
	})
}

// TestCloudMailAdapter_Connect 测试连接
func TestCloudMailAdapter_Connect(t *testing.T) {
	t.Run("连接成功", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MailListResponse{Data: []MailItem{}})
		}))
		defer server.Close()

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "test-jwt-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)

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

		config := &model.CloudMailAuthData{
			BaseURL:  server.URL,
			JWTToken: "invalid-token",
			Accounts: []model.CloudMailAccount{
				{Email: "user1@example.com"},
			},
		}
		adapter, _ := NewCloudMailAdapter(config)

		err := adapter.Connect(context.Background())
		if err == nil {
			t.Error("连接失败应返回错误")
		}

		if adapter.IsConnected() {
			t.Error("连接失败后 IsConnected() 应返回 false")
		}
	})
}

// TestCloudMailAdapter_Disconnect 测试断开连接
func TestCloudMailAdapter_Disconnect(t *testing.T) {
	config := &model.CloudMailAuthData{
		BaseURL:  "https://cloudmail.example.com",
		JWTToken: "test-jwt-token",
		Accounts: []model.CloudMailAccount{
			{Email: "user1@example.com"},
		},
	}
	adapter, _ := NewCloudMailAdapter(config)
	adapter.SetConnected(true)

	err := adapter.Disconnect()
	if err != nil {
		t.Errorf("Disconnect 失败: %v", err)
	}

	if adapter.IsConnected() {
		t.Error("断开连接后 IsConnected() 应返回 false")
	}
}

// TestCloudMailAdapter_GetConfig 测试获取配置
func TestCloudMailAdapter_GetConfig(t *testing.T) {
	config := &model.CloudMailAuthData{
		BaseURL:  "https://cloudmail.example.com",
		JWTToken: "test-jwt-token",
		Accounts: []model.CloudMailAccount{
			{Email: "user1@example.com"},
		},
	}
	adapter, _ := NewCloudMailAdapter(config)

	gotConfig := adapter.GetConfig()
	if gotConfig != config {
		t.Error("GetConfig() 应返回原始配置")
	}
}
