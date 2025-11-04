package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestGraphQuickAdapter_FetchEmails 测试邮件列表获取
func TestGraphQuickAdapter_FetchEmails(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			// 模拟 token 响应
			response := TokenResponse{
				AccessToken: "test_access_token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/v1.0/me/messages" {
			// 模拟邮件列表响应
			messageList := GraphMessageList{
				Value: []GraphMessage{
					{
						ID:               "message1",
						Subject:          "Test Email 1",
						BodyPreview:      "This is a test email",
						SentDateTime:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
						ReceivedDateTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
						IsRead:           false,
						HasAttachments:   false,
						From: GraphRecipient{
							EmailAddress: GraphEmailAddress{
								Name:    "Test Sender",
								Address: "sender@example.com",
							},
						},
						ToRecipients: []GraphRecipient{
							{
								EmailAddress: GraphEmailAddress{
									Name:    "Test Recipient",
									Address: "recipient@example.com",
								},
							},
						},
					},
					{
						ID:               "message2",
						Subject:          "Test Email 2",
						BodyPreview:      "This is another test email",
						SentDateTime:     time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
						ReceivedDateTime: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
						IsRead:           true,
						HasAttachments:   true,
						From: GraphRecipient{
							EmailAddress: GraphEmailAddress{
								Name:    "Another Sender",
								Address: "another@example.com",
							},
						},
					},
				},
				NextLink: "https://graph.microsoft.com/v1.0/me/messages?$skip=2",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messageList)
		}
	}))
	defer server.Close()

	// 创建适配器
	adapter := createTestAdapter(t, server.URL)

	// 测试获取邮件列表
	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)
	emails, err := adapter.FetchEmails(ctx, since, 10)

	if err != nil {
		t.Fatalf("FetchEmails failed: %v", err)
	}

	// 验证结果
	if len(emails) != 2 {
		t.Errorf("Expected 2 emails, got %d", len(emails))
	}

	// 验证第一封邮件
	if emails[0].Subject != "Test Email 1" {
		t.Errorf("Expected subject 'Test Email 1', got '%s'", emails[0].Subject)
	}

	if emails[0].FromAddress != "sender@example.com" {
		t.Errorf("Expected from 'sender@example.com', got '%s'", emails[0].FromAddress)
	}

	if *emails[0].SourceIsRead != false {
		t.Errorf("Expected first email to be unread")
	}

	// 验证第二封邮件
	if emails[1].HasAttachments != true {
		t.Errorf("Expected second email to have attachments")
	}

	if *emails[1].SourceIsRead != true {
		t.Errorf("Expected second email to be read")
	}
}

// TestGraphQuickAdapter_FetchEmailDetail 测试邮件详情获取
func TestGraphQuickAdapter_FetchEmailDetail(t *testing.T) {
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
		} else if strings.HasPrefix(r.URL.Path, "/v1.0/me/messages/") && strings.HasSuffix(r.URL.Path, "/attachments") {
			// 模拟附件响应
			attachmentList := GraphAttachmentList{
				Value: []GraphAttachment{
					{
						ID:          "attachment1",
						Name:        "document.pdf",
						ContentType: "application/pdf",
						Size:        1024,
						IsInline:    false,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(attachmentList)
		} else if strings.HasPrefix(r.URL.Path, "/v1.0/me/messages/") {
			// 模拟邮件详情响应
			message := GraphMessage{
				ID:               "test_message_id",
				Subject:          "Detailed Test Email",
				BodyPreview:      "This is a detailed test email",
				SentDateTime:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				ReceivedDateTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				IsRead:           false,
				HasAttachments:   true,
				Body: GraphItemBody{
					ContentType: "html",
					Content:     "<html><body>This is the HTML body</body></html>",
				},
				From: GraphRecipient{
					EmailAddress: GraphEmailAddress{
						Name:    "Detailed Sender",
						Address: "detailed@example.com",
					},
				},
				ToRecipients: []GraphRecipient{
					{
						EmailAddress: GraphEmailAddress{
							Name:    "Detailed Recipient",
							Address: "recipient@example.com",
						},
					},
				},
				CcRecipients: []GraphRecipient{
					{
						EmailAddress: GraphEmailAddress{
							Name:    "CC Recipient",
							Address: "cc@example.com",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(message)
		}
	}))
	defer server.Close()

	// 创建适配器
	adapter := createTestAdapter(t, server.URL)

	// 测试获取邮件详情
	ctx := context.Background()
	email, err := adapter.FetchEmailDetail(ctx, "test_message_id")

	if err != nil {
		t.Fatalf("FetchEmailDetail failed: %v", err)
	}

	// 验证结果
	if email.Subject != "Detailed Test Email" {
		t.Errorf("Expected subject 'Detailed Test Email', got '%s'", email.Subject)
	}

	if email.FromAddress != "detailed@example.com" {
		t.Errorf("Expected from 'detailed@example.com', got '%s'", email.FromAddress)
	}

	if email.HTMLBody != "<html><body>This is the HTML body</body></html>" {
		t.Errorf("Expected HTML body, got '%s'", email.HTMLBody)
	}

	if len(email.ToAddresses) != 1 {
		t.Errorf("Expected 1 to address, got %d", len(email.ToAddresses))
	}

	if len(email.CcAddresses) != 1 {
		t.Errorf("Expected 1 cc address, got %d", len(email.CcAddresses))
	}

	// 验证附件
	if !email.HasAttachments {
		t.Error("Expected email to have attachments")
	}

	if email.AttachmentsCount != 1 {
		t.Errorf("Expected 1 attachment, got %d", email.AttachmentsCount)
	}

	if len(email.Attachments) != 1 {
		t.Errorf("Expected 1 attachment in list, got %d", len(email.Attachments))
	}

	if email.Attachments[0].Filename != "document.pdf" {
		t.Errorf("Expected attachment filename 'document.pdf', got '%s'", email.Attachments[0].Filename)
	}
}

// TestGraphQuickAdapter_FetchEmailsWithFilter 测试过滤邮件获取
func TestGraphQuickAdapter_FetchEmailsWithFilter(t *testing.T) {
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
		} else if r.URL.Path == "/v1.0/me/messages" {
			// 检查过滤参数
			filter := r.URL.Query().Get("$filter")
			if !strings.Contains(filter, "isRead eq false") {
				t.Errorf("Expected filter to contain 'isRead eq false', got: %s", filter)
			}

			// 模拟过滤后的邮件列表
			messageList := GraphMessageList{
				Value: []GraphMessage{
					{
						ID:               "unread_message",
						Subject:          "Unread Email",
						BodyPreview:      "This is an unread email",
						SentDateTime:     time.Now().Format(time.RFC3339),
						ReceivedDateTime: time.Now().Format(time.RFC3339),
						IsRead:           false,
						HasAttachments:   false,
						From: GraphRecipient{
							EmailAddress: GraphEmailAddress{
								Address: "sender@example.com",
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messageList)
		}
	}))
	defer server.Close()

	// 创建适配器
	adapter := createTestAdapter(t, server.URL)

	// 创建过滤条件
	isRead := false
	filter := &EmailFilter{
		IsRead: &isRead,
		Since:  time.Now().Add(-24 * time.Hour),
	}

	// 测试过滤邮件获取
	ctx := context.Background()
	emails, err := adapter.FetchEmailsWithFilter(ctx, filter, 10)

	if err != nil {
		t.Fatalf("FetchEmailsWithFilter failed: %v", err)
	}

	// 验证结果
	if len(emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(emails))
	}

	if emails[0].Subject != "Unread Email" {
		t.Errorf("Expected subject 'Unread Email', got '%s'", emails[0].Subject)
	}

	if *emails[0].SourceIsRead != false {
		t.Error("Expected email to be unread")
	}
}

// TestGraphQuickAdapter_FetchEmailsWithPagination 测试分页邮件获取
func TestGraphQuickAdapter_FetchEmailsWithPagination(t *testing.T) {
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
		} else if r.URL.Path == "/v1.0/me/messages" {
			// 检查分页参数
			top := r.URL.Query().Get("$top")
			if top != "2" {
				t.Errorf("Expected $top=2, got: %s", top)
			}

			// 模拟第一页响应
			messageList := GraphMessageList{
				Value: []GraphMessage{
					{
						ID:      "page1_message1",
						Subject: "Page 1 Message 1",
						From: GraphRecipient{
							EmailAddress: GraphEmailAddress{Address: "sender1@example.com"},
						},
						SentDateTime:     time.Now().Format(time.RFC3339),
						ReceivedDateTime: time.Now().Format(time.RFC3339),
					},
					{
						ID:      "page1_message2",
						Subject: "Page 1 Message 2",
						From: GraphRecipient{
							EmailAddress: GraphEmailAddress{Address: "sender2@example.com"},
						},
						SentDateTime:     time.Now().Format(time.RFC3339),
						ReceivedDateTime: time.Now().Format(time.RFC3339),
					},
				},
				NextLink: "https://graph.microsoft.com/v1.0/me/messages?$skip=2",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messageList)
		}
	}))
	defer server.Close()

	// 创建适配器
	adapter := createTestAdapter(t, server.URL)

	// 测试分页邮件获取
	ctx := context.Background()
	page, err := adapter.FetchEmailsWithPagination(ctx, 2, "")

	if err != nil {
		t.Fatalf("FetchEmailsWithPagination failed: %v", err)
	}

	// 验证结果
	if page.PageSize != 2 {
		t.Errorf("Expected page size 2, got %d", page.PageSize)
	}

	if !page.HasNextPage {
		t.Error("Expected to have next page")
	}

	if page.NextPageToken == "" {
		t.Error("Expected next page token")
	}

	if len(page.Emails) != 2 {
		t.Errorf("Expected 2 emails, got %d", len(page.Emails))
	}

	if page.Emails[0].Subject != "Page 1 Message 1" {
		t.Errorf("Expected first email subject 'Page 1 Message 1', got '%s'", page.Emails[0].Subject)
	}
}

// TestGraphQuickAdapter_GetEmailCount 测试邮件计数
func TestGraphQuickAdapter_GetEmailCount(t *testing.T) {
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
		} else if r.URL.Path == "/v1.0/me/messages/$count" {
			// 模拟计数响应
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("42"))
		}
	}))
	defer server.Close()

	// 创建适配器
	adapter := createTestAdapter(t, server.URL)

	// 测试获取邮件计数
	ctx := context.Background()
	count, err := adapter.GetEmailCount(ctx)

	if err != nil {
		t.Fatalf("GetEmailCount failed: %v", err)
	}

	if count != 42 {
		t.Errorf("Expected count 42, got %d", count)
	}
}

// TestGraphQuickAdapter_ErrorHandling 测试错误处理
func TestGraphQuickAdapter_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorContains  string
	}{
		{
			name: "邮件不存在",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					response := TokenResponse{
						AccessToken: "test_access_token",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				} else if r.URL.Path == "/v1.0/me/messages" {
					// 返回空的邮件列表而不是 404 错误
					messageList := GraphMessageList{
						Value:    []GraphMessage{},
						NextLink: "",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(messageList)
				}
			},
			expectError:   false, // 空列表不应该是错误
			errorContains: "",
		},
		{
			name: "权限不足",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					response := TokenResponse{
						AccessToken: "test_access_token",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				} else if r.URL.Path == "/v1.0/me/messages" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Insufficient privileges"))
				}
			},
			expectError:   true,
			errorContains: "status 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			adapter := createTestAdapter(t, server.URL)
			ctx := context.Background()

			// 测试 FetchEmails 错误处理
			_, err := adapter.FetchEmails(ctx, time.Time{}, 10)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// BenchmarkGraphQuickAdapter_FetchEmails 邮件获取性能基准
func BenchmarkGraphQuickAdapter_FetchEmails(b *testing.B) {
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
		} else if r.URL.Path == "/v1.0/me/messages" {
			messageList := GraphMessageList{
				Value: []GraphMessage{
					{
						ID:      "benchmark_message",
						Subject: "Benchmark Email",
						From: GraphRecipient{
							EmailAddress: GraphEmailAddress{Address: "bench@example.com"},
						},
						SentDateTime:     time.Now().Format(time.RFC3339),
						ReceivedDateTime: time.Now().Format(time.RFC3339),
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messageList)
		}
	}))
	defer server.Close()

	adapter := createTestAdapter(b, server.URL)
	ctx := context.Background()

	// 预热
	adapter.FetchEmails(ctx, time.Time{}, 10)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		adapter.FetchEmails(ctx, time.Time{}, 10)
	}
}
