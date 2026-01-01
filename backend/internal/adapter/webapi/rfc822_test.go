package webapi

import (
	"strings"
	"testing"
	"time"
)

// TestRFC822Parser_Parse 测试 RFC822 解析器基本功能
func TestRFC822Parser_Parse(t *testing.T) {
	parser := NewRFC822Parser()

	tests := []struct {
		name        string
		rawContent  string
		wantSubject string
		wantFrom    string
		wantTo      []string
		wantErr     bool
	}{
		{
			name: "简单纯文本邮件",
			rawContent: `From: sender@example.com
To: receiver@example.com
Subject: Test Subject
Date: Mon, 30 Dec 2024 10:00:00 +0800
Message-ID: <test123@example.com>

This is the email body.`,
			wantSubject: "Test Subject",
			wantFrom:    "sender@example.com",
			wantTo:      []string{"receiver@example.com"},
			wantErr:     false,
		},
		{
			name: "带发件人名称的邮件",
			rawContent: `From: "John Doe" <john@example.com>
To: "Jane Doe" <jane@example.com>
Subject: Hello World
Date: Mon, 30 Dec 2024 10:00:00 +0800

Hello!`,
			wantSubject: "Hello World",
			wantFrom:    "john@example.com",
			wantTo:      []string{"jane@example.com"},
			wantErr:     false,
		},
		{
			name: "多收件人邮件",
			rawContent: `From: sender@example.com
To: user1@example.com, user2@example.com
Subject: Multi Recipients
Date: Mon, 30 Dec 2024 10:00:00 +0800

Content`,
			wantSubject: "Multi Recipients",
			wantFrom:    "sender@example.com",
			wantTo:      []string{"user1@example.com", "user2@example.com"},
			wantErr:     false,
		},
		{
			name:       "空内容",
			rawContent: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := parser.Parse(tt.rawContent)

			if tt.wantErr {
				if err == nil {
					t.Errorf("期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("解析失败: %v", err)
				return
			}

			if email.Subject != tt.wantSubject {
				t.Errorf("Subject = %q, want %q", email.Subject, tt.wantSubject)
			}

			if email.FromAddress != tt.wantFrom {
				t.Errorf("FromAddress = %q, want %q", email.FromAddress, tt.wantFrom)
			}

			if len(email.ToAddresses) != len(tt.wantTo) {
				t.Errorf("ToAddresses 数量 = %d, want %d", len(email.ToAddresses), len(tt.wantTo))
			} else {
				for i, addr := range tt.wantTo {
					if email.ToAddresses[i] != addr {
						t.Errorf("ToAddresses[%d] = %q, want %q", i, email.ToAddresses[i], addr)
					}
				}
			}
		})
	}
}

// TestRFC822Parser_ParseRFC2047Subject 测试 RFC2047 编码主题解析
func TestRFC822Parser_ParseRFC2047Subject(t *testing.T) {
	parser := NewRFC822Parser()

	tests := []struct {
		name        string
		rawContent  string
		wantSubject string
	}{
		{
			name: "UTF-8 Base64 编码主题",
			rawContent: `From: sender@example.com
To: receiver@example.com
Subject: =?UTF-8?B?5rWL6K+V5Li76aKY?=
Date: Mon, 30 Dec 2024 10:00:00 +0800

Body`,
			wantSubject: "测试主题",
		},
		{
			name: "UTF-8 Quoted-Printable 编码主题",
			rawContent: `From: sender@example.com
To: receiver@example.com
Subject: =?UTF-8?Q?=E6=B5=8B=E8=AF=95?=
Date: Mon, 30 Dec 2024 10:00:00 +0800

Body`,
			wantSubject: "测试",
		},
		{
			name: "普通 ASCII 主题",
			rawContent: `From: sender@example.com
To: receiver@example.com
Subject: Plain ASCII Subject
Date: Mon, 30 Dec 2024 10:00:00 +0800

Body`,
			wantSubject: "Plain ASCII Subject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := parser.Parse(tt.rawContent)
			if err != nil {
				t.Errorf("解析失败: %v", err)
				return
			}

			if email.Subject != tt.wantSubject {
				t.Errorf("Subject = %q, want %q", email.Subject, tt.wantSubject)
			}
		})
	}
}

// TestRFC822Parser_ParseMultipart 测试 multipart 邮件解析
func TestRFC822Parser_ParseMultipart(t *testing.T) {
	parser := NewRFC822Parser()

	rawContent := `From: sender@example.com
To: receiver@example.com
Subject: Multipart Test
Date: Mon, 30 Dec 2024 10:00:00 +0800
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset="utf-8"

This is plain text.
--boundary123
Content-Type: text/html; charset="utf-8"

<html><body><p>This is HTML.</p></body></html>
--boundary123--`

	email, err := parser.Parse(rawContent)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if email.TextBody == "" {
		t.Error("TextBody 不应为空")
	}

	if !strings.Contains(email.TextBody, "plain text") {
		t.Errorf("TextBody 应包含 'plain text', got %q", email.TextBody)
	}

	if email.HTMLBody == "" {
		t.Error("HTMLBody 不应为空")
	}

	if !strings.Contains(email.HTMLBody, "This is HTML") {
		t.Errorf("HTMLBody 应包含 'This is HTML', got %q", email.HTMLBody)
	}
}

// TestRFC822Parser_ParseDate 测试日期解析
func TestRFC822Parser_ParseDate(t *testing.T) {
	parser := NewRFC822Parser()

	rawContent := `From: sender@example.com
To: receiver@example.com
Subject: Date Test
Date: Mon, 30 Dec 2024 10:30:00 +0800

Body`

	email, err := parser.Parse(rawContent)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if email.SentAt.IsZero() {
		t.Error("SentAt 不应为零值")
	}

	// 验证日期（考虑时区）
	expectedYear := 2024
	expectedMonth := time.December
	expectedDay := 30

	if email.SentAt.Year() != expectedYear {
		t.Errorf("Year = %d, want %d", email.SentAt.Year(), expectedYear)
	}
	if email.SentAt.Month() != expectedMonth {
		t.Errorf("Month = %v, want %v", email.SentAt.Month(), expectedMonth)
	}
	if email.SentAt.Day() != expectedDay {
		t.Errorf("Day = %d, want %d", email.SentAt.Day(), expectedDay)
	}
}

// TestRFC822Parser_ParseMessageID 测试 Message-ID 解析
func TestRFC822Parser_ParseMessageID(t *testing.T) {
	parser := NewRFC822Parser()

	tests := []struct {
		name          string
		rawContent    string
		wantMessageID string
	}{
		{
			name: "带尖括号的 Message-ID",
			rawContent: `From: sender@example.com
To: receiver@example.com
Subject: Test
Date: Mon, 30 Dec 2024 10:00:00 +0800
Message-ID: <unique123@example.com>

Body`,
			wantMessageID: "unique123@example.com",
		},
		{
			name: "不带尖括号的 Message-ID",
			rawContent: `From: sender@example.com
To: receiver@example.com
Subject: Test
Date: Mon, 30 Dec 2024 10:00:00 +0800
Message-ID: simple123@example.com

Body`,
			wantMessageID: "simple123@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := parser.Parse(tt.rawContent)
			if err != nil {
				t.Errorf("解析失败: %v", err)
				return
			}

			if email.MessageID != tt.wantMessageID {
				t.Errorf("MessageID = %q, want %q", email.MessageID, tt.wantMessageID)
			}
		})
	}
}

// TestRFC822Parser_ParseCc 测试 Cc 解析
func TestRFC822Parser_ParseCc(t *testing.T) {
	parser := NewRFC822Parser()

	rawContent := `From: sender@example.com
To: receiver@example.com
Cc: cc1@example.com, cc2@example.com
Subject: Cc Test
Date: Mon, 30 Dec 2024 10:00:00 +0800

Body`

	email, err := parser.Parse(rawContent)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(email.CcAddresses) != 2 {
		t.Errorf("CcAddresses 数量 = %d, want 2", len(email.CcAddresses))
	}

	expectedCc := []string{"cc1@example.com", "cc2@example.com"}
	for i, addr := range expectedCc {
		if i < len(email.CcAddresses) && email.CcAddresses[i] != addr {
			t.Errorf("CcAddresses[%d] = %q, want %q", i, email.CcAddresses[i], addr)
		}
	}
}

// TestRFC822Parser_ParseReplyTo 测试 Reply-To 解析
func TestRFC822Parser_ParseReplyTo(t *testing.T) {
	parser := NewRFC822Parser()

	rawContent := `From: sender@example.com
To: receiver@example.com
Reply-To: reply@example.com
Subject: Reply-To Test
Date: Mon, 30 Dec 2024 10:00:00 +0800

Body`

	email, err := parser.Parse(rawContent)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if email.ReplyTo != "reply@example.com" {
		t.Errorf("ReplyTo = %q, want %q", email.ReplyTo, "reply@example.com")
	}
}

// TestGenerateSnippet 测试摘要生成
func TestGenerateSnippet(t *testing.T) {
	parser := NewRFC822Parser()

	rawContent := `From: sender@example.com
To: receiver@example.com
Subject: Snippet Test
Date: Mon, 30 Dec 2024 10:00:00 +0800

This is a test email body with some content that should be used to generate a snippet.`

	email, err := parser.Parse(rawContent)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	snippet := GenerateSnippet(email, 50)

	if len(snippet) > 53 { // 50 + "..."
		t.Errorf("Snippet 长度 = %d, 应该 <= 53", len(snippet))
	}

	if !strings.Contains(snippet, "test email") {
		t.Errorf("Snippet 应包含 'test email', got %q", snippet)
	}
}

// TestStripHTML 测试 HTML 标签移除
func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "简单 HTML",
			html:     "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "嵌套标签",
			html:     "<div><p>Nested <strong>content</strong></p></div>",
			expected: "Nested content",
		},
		{
			name:     "无标签",
			html:     "Plain text",
			expected: "Plain text",
		},
		{
			name:     "空字符串",
			html:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTML(tt.html)
			if result != tt.expected {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.html, result, tt.expected)
			}
		})
	}
}

// TestParseEmailAddress 测试邮箱地址解析
func TestParseEmailAddress(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantAddress string
	}{
		{
			input:       "user@example.com",
			wantName:    "",
			wantAddress: "user@example.com",
		},
		{
			input:       "John Doe <john@example.com>",
			wantName:    "John Doe",
			wantAddress: "john@example.com",
		},
		{
			input:       `"John Doe" <john@example.com>`,
			wantName:    "John Doe",
			wantAddress: "john@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, address := ParseEmailAddress(tt.input)
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if address != tt.wantAddress {
				t.Errorf("address = %q, want %q", address, tt.wantAddress)
			}
		})
	}
}

// TestFormatEmailAddress 测试邮箱地址格式化
func TestFormatEmailAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "",
			address:  "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "John Doe",
			address:  "john@example.com",
			expected: "John Doe <john@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatEmailAddress(tt.name, tt.address)
			if result != tt.expected {
				t.Errorf("FormatEmailAddress(%q, %q) = %q, want %q", tt.name, tt.address, result, tt.expected)
			}
		})
	}
}

// TestSplitAddresses 测试地址分割
func TestSplitAddresses(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "user1@example.com, user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			input:    "John <john@example.com>, Jane <jane@example.com>",
			expected: []string{"john@example.com", "jane@example.com"},
		},
		{
			input:    "single@example.com",
			expected: []string{"single@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitAddresses(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitAddresses(%q) 返回 %d 个地址, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, addr := range tt.expected {
				if result[i] != addr {
					t.Errorf("splitAddresses(%q)[%d] = %q, want %q", tt.input, i, result[i], addr)
				}
			}
		})
	}
}
