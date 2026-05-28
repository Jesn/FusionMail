package cloudmail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"fusionmail/internal/model"
)

const testJWTToken = "test-jwt-token"

func validCloudMailConfig(baseURL string) *model.CloudMailAuthData {
	return &model.CloudMailAuthData{
		BaseURL:  baseURL,
		JWTToken: testJWTToken,
	}
}

func writeCloudMailJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("写入测试响应失败: %v", err)
	}
}

func cloudMailTime(ts time.Time) string {
	return ts.Format("2006-01-02 15:04:05")
}

func newConnectedCloudMailAdapter(t *testing.T, baseURL string, accounts []CloudMailAccount) *CloudMailAdapter {
	t.Helper()

	adapter, err := NewCloudMailAdapter(validCloudMailConfig(baseURL))
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}
	adapter.accounts = accounts
	adapter.SetConnected(true)
	return adapter
}

// TestNewCloudMailAdapter 测试适配器创建。
func TestNewCloudMailAdapter(t *testing.T) {
	t.Run("nil 配置", func(t *testing.T) {
		_, err := NewCloudMailAdapter(nil)
		if err == nil {
			t.Fatal("nil 配置应返回错误")
		}
	})

	t.Run("空配置", func(t *testing.T) {
		_, err := NewCloudMailAdapter(&model.CloudMailAuthData{})
		if err == nil {
			t.Fatal("空配置应返回错误")
		}
	})

	t.Run("有效 JWT 配置", func(t *testing.T) {
		adapter, err := NewCloudMailAdapter(validCloudMailConfig("https://cloudmail.example.com"))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}
		if adapter == nil {
			t.Fatal("适配器不应为 nil")
		}
	})

	t.Run("有效账号密码配置", func(t *testing.T) {
		config := &model.CloudMailAuthData{
			BaseURL:  "https://cloudmail.example.com",
			Email:    "user@example.com",
			Password: "secret",
		}
		adapter, err := NewCloudMailAdapter(config)
		if err != nil {
			t.Fatalf("账号密码配置应可创建适配器: %v", err)
		}
		if adapter == nil {
			t.Fatal("适配器不应为 nil")
		}
	})

	t.Run("缺少认证信息", func(t *testing.T) {
		config := &model.CloudMailAuthData{BaseURL: "https://cloudmail.example.com"}
		_, err := NewCloudMailAdapter(config)
		if err == nil {
			t.Fatal("缺少 JWT Token 或账号密码时应返回错误")
		}
	})
}

// TestCloudMailAdapter_GetProviderType 测试获取提供商类型。
func TestCloudMailAdapter_GetProviderType(t *testing.T) {
	adapter, err := NewCloudMailAdapter(validCloudMailConfig("https://cloudmail.example.com"))
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	if adapter.GetProviderType() != model.WebAPIServiceTypeCloudMail {
		t.Errorf("GetProviderType() = %q, want %q", adapter.GetProviderType(), model.WebAPIServiceTypeCloudMail)
	}
}

// TestCloudMailAdapter_GetProtocol 测试获取协议类型。
func TestCloudMailAdapter_GetProtocol(t *testing.T) {
	adapter, err := NewCloudMailAdapter(validCloudMailConfig("https://cloudmail.example.com"))
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	if adapter.GetProtocol() != "webapi" {
		t.Errorf("GetProtocol() = %q, want %q", adapter.GetProtocol(), "webapi")
	}
}

// TestCloudMailAdapter_TestConnection 测试连接探测。
func TestCloudMailAdapter_TestConnection(t *testing.T) {
	t.Run("连接成功", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/account/list" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}
			if r.URL.Query().Get("accountId") != "0" || r.URL.Query().Get("size") != "20" {
				t.Errorf("请求参数错误: %s", r.URL.RawQuery)
			}
			if r.Header.Get("Authorization") != testJWTToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			writeCloudMailJSON(t, w, AccountListResponse{
				Code:    200,
				Message: "ok",
				Data:    []AccountItem{},
			})
		}))
		defer server.Close()

		adapter, err := NewCloudMailAdapter(validCloudMailConfig(server.URL))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		if err := adapter.TestConnection(context.Background()); err != nil {
			t.Fatalf("TestConnection 失败: %v", err)
		}
	})

	t.Run("认证失败", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		adapter, err := NewCloudMailAdapter(validCloudMailConfig(server.URL))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		if err := adapter.TestConnection(context.Background()); err == nil {
			t.Fatal("认证失败应返回错误")
		}
	})

	t.Run("服务器错误", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		adapter, err := NewCloudMailAdapter(validCloudMailConfig(server.URL))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		if err := adapter.TestConnection(context.Background()); err == nil {
			t.Fatal("服务器错误应返回错误")
		}
	})
}

// TestCloudMailAdapter_Connect 测试连接并从 API 获取账户列表。
func TestCloudMailAdapter_Connect(t *testing.T) {
	t.Run("连接成功并忽略已删除账户", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/account/list" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}

			writeCloudMailJSON(t, w, AccountListResponse{
				Code:    200,
				Message: "ok",
				Data: []AccountItem{
					{AccountID: 101, Email: "user1@example.com", Name: "用户1", IsDel: 0},
					{AccountID: 102, Email: "deleted@example.com", Name: "已删除", IsDel: 1},
				},
			})
		}))
		defer server.Close()

		adapter, err := NewCloudMailAdapter(validCloudMailConfig(server.URL))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		if err := adapter.Connect(context.Background()); err != nil {
			t.Fatalf("Connect 失败: %v", err)
		}
		if !adapter.IsConnected() {
			t.Fatal("连接后 IsConnected() 应返回 true")
		}
		if got := adapter.GetAccounts(); len(got) != 1 || got[0].Email != "user1@example.com" {
			t.Fatalf("账户列表 = %#v, want 仅包含 user1@example.com", got)
		}
	})

	t.Run("连接失败", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		adapter, err := NewCloudMailAdapter(validCloudMailConfig(server.URL))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		if err := adapter.Connect(context.Background()); err == nil {
			t.Fatal("连接失败应返回错误")
		}
		if adapter.IsConnected() {
			t.Fatal("连接失败后 IsConnected() 应返回 false")
		}
	})
}

// TestCloudMailAdapter_FetchEmails 测试按运行时账户拉取邮件。
func TestCloudMailAdapter_FetchEmails(t *testing.T) {
	t.Run("成功拉取多账户邮件", func(t *testing.T) {
		now := time.Now()
		requestedAccountIDs := make(map[string]bool)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/email/list" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}

			accountID := r.URL.Query().Get("accountId")
			requestedAccountIDs[accountID] = true

			emailID, err := strconv.Atoi(accountID)
			if err != nil {
				t.Fatalf("accountId 应为数字: %v", err)
			}

			writeCloudMailJSON(t, w, EmailListResponse{
				Code:    200,
				Message: "ok",
				Data: EmailListData{
					List: []EmailItem{
						{
							EmailID:    emailID,
							AccountID:  emailID,
							Subject:    "测试邮件",
							SendEmail:  "sender@example.com",
							ToEmail:    "",
							CreateTime: cloudMailTime(now),
						},
					},
					Total: 1,
				},
			})
		}))
		defer server.Close()

		adapter := newConnectedCloudMailAdapter(t, server.URL, []CloudMailAccount{
			{AccountID: 101, Email: "user1@example.com"},
			{AccountID: 102, Email: "user2@example.com"},
		})

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err != nil {
			t.Fatalf("FetchEmails 失败: %v", err)
		}
		if len(emails) != 2 {
			t.Fatalf("邮件数量 = %d, want 2", len(emails))
		}
		if !requestedAccountIDs["101"] || !requestedAccountIDs["102"] {
			t.Fatalf("请求账户 = %#v, want 101 和 102", requestedAccountIDs)
		}
		if emails[0].ToAddresses[0] != "user1@example.com" || emails[1].ToAddresses[0] != "user2@example.com" {
			t.Fatalf("目标地址 = %#v / %#v, want 运行时账户邮箱", emails[0].ToAddresses, emails[1].ToAddresses)
		}
	})

	t.Run("未连接时拉取", func(t *testing.T) {
		adapter, err := NewCloudMailAdapter(validCloudMailConfig("https://cloudmail.example.com"))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		if _, err := adapter.FetchEmails(context.Background(), time.Time{}, 10); err == nil {
			t.Fatal("未连接时应返回错误")
		}
	})

	t.Run("时间过滤", func(t *testing.T) {
		now := time.Now()
		oldTime := now.Add(-24 * time.Hour)
		newTime := now.Add(-1 * time.Hour)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeCloudMailJSON(t, w, EmailListResponse{
				Code:    200,
				Message: "ok",
				Data: EmailListData{
					List: []EmailItem{
						{
							EmailID:    1,
							Subject:    "旧邮件",
							SendEmail:  "sender@example.com",
							CreateTime: cloudMailTime(oldTime),
						},
						{
							EmailID:    2,
							Subject:    "新邮件",
							SendEmail:  "sender@example.com",
							CreateTime: cloudMailTime(newTime),
						},
					},
					Total: 2,
				},
			})
		}))
		defer server.Close()

		adapter := newConnectedCloudMailAdapter(t, server.URL, []CloudMailAccount{
			{AccountID: 101, Email: "user1@example.com"},
		})

		emails, err := adapter.FetchEmails(context.Background(), now.Add(-12*time.Hour), 10)
		if err != nil {
			t.Fatalf("FetchEmails 失败: %v", err)
		}
		if len(emails) != 1 {
			t.Fatalf("邮件数量 = %d, want 1", len(emails))
		}
		if emails[0].ProviderID != "2" {
			t.Fatalf("ProviderID = %q, want %q", emails[0].ProviderID, "2")
		}
	})

	t.Run("部分账户失败继续处理", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if r.URL.Query().Get("accountId") == "101" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			writeCloudMailJSON(t, w, EmailListResponse{
				Code:    200,
				Message: "ok",
				Data: EmailListData{
					List: []EmailItem{
						{
							EmailID:    2,
							Subject:    "用户2的邮件",
							SendEmail:  "sender@example.com",
							CreateTime: cloudMailTime(time.Now()),
						},
					},
					Total: 1,
				},
			})
		}))
		defer server.Close()

		adapter := newConnectedCloudMailAdapter(t, server.URL, []CloudMailAccount{
			{AccountID: 101, Email: "user1@example.com"},
			{AccountID: 102, Email: "user2@example.com"},
		})

		emails, err := adapter.FetchEmails(context.Background(), time.Time{}, 10)
		if err != nil {
			t.Fatalf("FetchEmails 不应返回错误: %v", err)
		}
		if len(emails) != 1 {
			t.Fatalf("邮件数量 = %d, want 1", len(emails))
		}
		if requestCount != 2 {
			t.Fatalf("请求次数 = %d, want 2", requestCount)
		}
	})
}

// TestCloudMailAdapter_FetchEmailDetail 测试获取邮件详情。
func TestCloudMailAdapter_FetchEmailDetail(t *testing.T) {
	t.Run("成功获取详情", func(t *testing.T) {
		now := time.Now()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/email/detail" {
				t.Errorf("请求路径错误: %s", r.URL.Path)
			}
			if r.URL.Query().Get("emailId") != "123" {
				t.Errorf("邮件 ID 参数错误: %s", r.URL.RawQuery)
			}

			writeCloudMailJSON(t, w, EmailDetailResponse{
				Code:    200,
				Message: "ok",
				Data: EmailItem{
					EmailID:    123,
					MessageID:  "<msg123@example.com>",
					Subject:    "详情测试邮件",
					SendEmail:  "sender@example.com",
					Name:       "发件人",
					ToEmail:    "target@example.com",
					Text:       "这是邮件详情内容",
					Content:    "<p>这是邮件详情内容</p>",
					CreateTime: cloudMailTime(now),
					AttList: []AttachmentItem{
						{AttID: 1, FileName: "a.txt"},
						{AttID: 2, FileName: "b.txt"},
					},
				},
			})
		}))
		defer server.Close()

		adapter := newConnectedCloudMailAdapter(t, server.URL, []CloudMailAccount{})

		email, err := adapter.FetchEmailDetail(context.Background(), "123")
		if err != nil {
			t.Fatalf("FetchEmailDetail 失败: %v", err)
		}
		if email.ProviderID != "123" {
			t.Errorf("ProviderID = %q, want %q", email.ProviderID, "123")
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

		adapter := newConnectedCloudMailAdapter(t, server.URL, []CloudMailAccount{})

		if _, err := adapter.FetchEmailDetail(context.Background(), "not-exist"); err == nil {
			t.Fatal("邮件不存在应返回错误")
		}
	})

	t.Run("未连接时获取详情", func(t *testing.T) {
		adapter, err := NewCloudMailAdapter(validCloudMailConfig("https://cloudmail.example.com"))
		if err != nil {
			t.Fatalf("创建适配器失败: %v", err)
		}

		if _, err := adapter.FetchEmailDetail(context.Background(), "123"); err == nil {
			t.Fatal("未连接时应返回错误")
		}
	})
}

// TestCloudMailAdapter_Disconnect 测试断开连接。
func TestCloudMailAdapter_Disconnect(t *testing.T) {
	adapter := newConnectedCloudMailAdapter(t, "https://cloudmail.example.com", []CloudMailAccount{
		{AccountID: 101, Email: "user1@example.com"},
	})

	if err := adapter.Disconnect(); err != nil {
		t.Fatalf("Disconnect 失败: %v", err)
	}
	if adapter.IsConnected() {
		t.Fatal("断开连接后 IsConnected() 应返回 false")
	}
	if len(adapter.GetAccounts()) != 0 {
		t.Fatalf("断开连接后账户列表应清空: %#v", adapter.GetAccounts())
	}
}

// TestCloudMailAdapter_GetConfig 测试获取配置。
func TestCloudMailAdapter_GetConfig(t *testing.T) {
	config := validCloudMailConfig("https://cloudmail.example.com")
	adapter, err := NewCloudMailAdapter(config)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	if adapter.GetConfig() != config {
		t.Fatal("GetConfig() 应返回原始配置")
	}
}
