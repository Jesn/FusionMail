package service

import (
	"testing"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
)

// TestGroupByTargetAddress 测试按目标地址分组
func TestGroupByTargetAddress(t *testing.T) {
	service := &WebAPISyncService{}

	emails := []*webapi.WebAPIEmail{
		{Email: &adapter.Email{ProviderID: "1"}, TargetAddress: "user1@example.com"},
		{Email: &adapter.Email{ProviderID: "2"}, TargetAddress: "user2@example.com"},
		{Email: &adapter.Email{ProviderID: "3"}, TargetAddress: "user1@example.com"},
		{Email: &adapter.Email{ProviderID: "4"}, TargetAddress: ""},
		{Email: &adapter.Email{ProviderID: "5"}, TargetAddress: "USER1@EXAMPLE.COM"}, // 大写
	}

	grouped := service.groupByTargetAddress(emails)

	// user1@example.com 应该有 3 封（包括大写的）
	if len(grouped["user1@example.com"]) != 3 {
		t.Errorf("user1 邮件数量 = %d, want 3", len(grouped["user1@example.com"]))
	}

	// user2@example.com 应该有 1 封
	if len(grouped["user2@example.com"]) != 1 {
		t.Errorf("user2 邮件数量 = %d, want 1", len(grouped["user2@example.com"]))
	}

	// 空地址应该归类到 _unknown_
	if len(grouped["_unknown_"]) != 1 {
		t.Errorf("unknown 邮件数量 = %d, want 1", len(grouped["_unknown_"]))
	}
}

// TestCreateEmailFromAdapter 测试从适配器邮件创建数据库模型
func TestCreateEmailFromAdapter(t *testing.T) {
	service := &WebAPISyncService{}

	now := time.Now()
	adapterEmail := &adapter.Email{
		ProviderID:       "provider-123",
		MessageID:        "msg-123",
		Subject:          "测试邮件",
		FromAddress:      "sender@example.com",
		FromName:         "发件人",
		ToAddresses:      []string{"user1@example.com", "user2@example.com"},
		CcAddresses:      []string{"cc@example.com"},
		TextBody:         "邮件正文",
		HTMLBody:         "<p>邮件正文</p>",
		Snippet:          "邮件摘要",
		HasAttachments:   true,
		AttachmentsCount: 2,
		SentAt:           now,
		ReceivedAt:       now,
	}

	accountUID := "acc-123"
	email := service.createEmailFromAdapter(adapterEmail, accountUID)

	if email.ProviderID != "provider-123" {
		t.Errorf("ProviderID = %q, want %q", email.ProviderID, "provider-123")
	}
	if email.AccountUID != accountUID {
		t.Errorf("AccountUID = %q, want %q", email.AccountUID, accountUID)
	}
	if email.Subject != "测试邮件" {
		t.Errorf("Subject = %q, want %q", email.Subject, "测试邮件")
	}
	if email.FromAddress != "sender@example.com" {
		t.Errorf("FromAddress = %q, want %q", email.FromAddress, "sender@example.com")
	}
	if email.ToAddresses != "user1@example.com,user2@example.com" {
		t.Errorf("ToAddresses = %q, want %q", email.ToAddresses, "user1@example.com,user2@example.com")
	}
	if email.HasAttachments != true {
		t.Error("HasAttachments should be true")
	}
	if email.AttachmentsCount != 2 {
		t.Errorf("AttachmentsCount = %d, want 2", email.AttachmentsCount)
	}
	if email.IsRead != false {
		t.Error("新邮件 IsRead 应为 false")
	}
}

// TestUpdateEmailFromAdapter 测试从适配器邮件更新数据库模型
func TestUpdateEmailFromAdapter(t *testing.T) {
	service := &WebAPISyncService{}

	// 原始邮件
	existingEmail := &model.Email{
		ID:         1,
		ProviderID: "provider-123",
		AccountUID: "acc-123",
		Subject:    "旧标题",
		TextBody:   "旧正文",
		IsRead:     true, // 已读
	}

	// 更新数据
	adapterEmail := &adapter.Email{
		Subject:          "新标题",
		TextBody:         "新正文",
		HTMLBody:         "<p>新正文</p>",
		HasAttachments:   true,
		AttachmentsCount: 3,
	}

	service.updateEmailFromAdapter(existingEmail, adapterEmail, "acc-123")

	if existingEmail.Subject != "新标题" {
		t.Errorf("Subject = %q, want %q", existingEmail.Subject, "新标题")
	}
	if existingEmail.TextBody != "新正文" {
		t.Errorf("TextBody = %q, want %q", existingEmail.TextBody, "新正文")
	}
	// IsRead 不应被覆盖
	if existingEmail.IsRead != true {
		t.Error("IsRead 不应被覆盖")
	}
}

// TestWebAPIEmail 测试 WebAPIEmail 结构
func TestWebAPIEmail(t *testing.T) {
	t.Run("创建 WebAPIEmail", func(t *testing.T) {
		email := &adapter.Email{
			ProviderID:  "email-1",
			Subject:     "测试",
			ToAddresses: []string{"user@example.com"},
		}

		webEmail := webapi.NewWebAPIEmail(email, "target@example.com")

		if webEmail.TargetAddress != "target@example.com" {
			t.Errorf("TargetAddress = %q, want %q", webEmail.TargetAddress, "target@example.com")
		}
		if webEmail.Email.ProviderID != "email-1" {
			t.Errorf("ProviderID = %q, want %q", webEmail.Email.ProviderID, "email-1")
		}
	})

	t.Run("ToEmail 转换", func(t *testing.T) {
		email := &adapter.Email{
			ProviderID: "email-1",
			Subject:    "测试",
		}

		webEmail := webapi.NewWebAPIEmail(email, "target@example.com")
		converted := webEmail.ToEmail()

		if converted.ProviderID != "email-1" {
			t.Errorf("ProviderID = %q, want %q", converted.ProviderID, "email-1")
		}
	})
}

// TestExtractTargetAddress 测试提取目标地址
func TestExtractTargetAddress(t *testing.T) {
	tests := []struct {
		name     string
		email    *adapter.Email
		expected string
	}{
		{
			name: "单个收件人",
			email: &adapter.Email{
				ToAddresses: []string{"user@example.com"},
			},
			expected: "user@example.com",
		},
		{
			name: "多个收件人取第一个",
			email: &adapter.Email{
				ToAddresses: []string{"user1@example.com", "user2@example.com"},
			},
			expected: "user1@example.com",
		},
		{
			name: "无收件人",
			email: &adapter.Email{
				ToAddresses: []string{},
			},
			expected: "",
		},
		{
			name:     "nil 收件人",
			email:    &adapter.Email{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := webapi.ExtractTargetAddress(tt.email)
			if result != tt.expected {
				t.Errorf("ExtractTargetAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSyncResult 测试同步结果
func TestSyncResult(t *testing.T) {
	result := webapi.NewSyncResult()

	if result.TotalCount != 0 {
		t.Errorf("初始 TotalCount = %d, want 0", result.TotalCount)
	}

	// 添加邮件
	email := &webapi.WebAPIEmail{
		Email:         &adapter.Email{ProviderID: "1"},
		TargetAddress: "user@example.com",
	}
	result.AddEmail(email)

	if result.TotalCount != 1 {
		t.Errorf("添加后 TotalCount = %d, want 1", result.TotalCount)
	}
}
