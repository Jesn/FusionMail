package model

import (
	"testing"
)

// TestCloudflareTempEmailAuthData_GetDomainList 测试域名列表解析
func TestCloudflareTempEmailAuthData_GetDomainList(t *testing.T) {
	tests := []struct {
		name     string
		domains  string
		expected []string
	}{
		{
			name:     "空字符串",
			domains:  "",
			expected: nil,
		},
		{
			name:     "单个域名",
			domains:  "example.com",
			expected: []string{"example.com"},
		},
		{
			name:     "多个域名逗号分隔",
			domains:  "example.com, test.org, demo.net",
			expected: []string{"example.com", "test.org", "demo.net"},
		},
		{
			name:     "带空格的域名",
			domains:  "  example.com  ,  test.org  ",
			expected: []string{"example.com", "test.org"},
		},
		{
			name:     "大小写混合",
			domains:  "Example.COM, TEST.org",
			expected: []string{"example.com", "test.org"},
		},
		{
			name:     "重复域名去重",
			domains:  "example.com, test.org, example.com",
			expected: []string{"example.com", "test.org"},
		},
		{
			name:     "空值过滤",
			domains:  "example.com, , test.org, ",
			expected: []string{"example.com", "test.org"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CloudflareTempEmailAuthData{
				Domains: tt.domains,
			}
			result := config.GetDomainList()

			if len(result) != len(tt.expected) {
				t.Errorf("GetDomainList() 长度 = %d, 期望 %d", len(result), len(tt.expected))
				return
			}

			for i, domain := range result {
				if domain != tt.expected[i] {
					t.Errorf("GetDomainList()[%d] = %q, 期望 %q", i, domain, tt.expected[i])
				}
			}
		})
	}
}

// TestCloudflareTempEmailAuthData_HasDomainFilter 测试是否配置了域名过滤
func TestCloudflareTempEmailAuthData_HasDomainFilter(t *testing.T) {
	tests := []struct {
		name     string
		domains  string
		expected bool
	}{
		{"空字符串", "", false},
		{"只有空格", "   ", false},
		{"只有逗号", ",,,", false},
		{"有效域名", "example.com", true},
		{"多个域名", "example.com, test.org", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CloudflareTempEmailAuthData{
				Domains: tt.domains,
			}
			if result := config.HasDomainFilter(); result != tt.expected {
				t.Errorf("HasDomainFilter() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// TestCloudflareTempEmailAuthData_MatchesDomain 测试邮箱域名匹配
func TestCloudflareTempEmailAuthData_MatchesDomain(t *testing.T) {
	tests := []struct {
		name     string
		domains  string
		email    string
		expected bool
	}{
		// 无过滤配置，全部通过
		{"无过滤-任意邮箱", "", "user@any.com", true},
		{"无过滤-空邮箱", "", "", true}, // 无过滤时，即使空邮箱也通过（由调用方处理）

		// 单域名过滤
		{"单域名-匹配", "example.com", "user@example.com", true},
		{"单域名-不匹配", "example.com", "user@other.com", false},
		{"单域名-大小写匹配", "example.com", "user@EXAMPLE.COM", true},

		// 多域名过滤
		{"多域名-匹配第一个", "example.com, test.org", "user@example.com", true},
		{"多域名-匹配第二个", "example.com, test.org", "user@test.org", true},
		{"多域名-不匹配", "example.com, test.org", "user@other.com", false},

		// 边界情况
		{"无效邮箱-无@", "example.com", "invalid-email", false},
		{"无效邮箱-@结尾", "example.com", "user@", false},
		{"子域名不匹配", "example.com", "user@sub.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CloudflareTempEmailAuthData{
				Domains: tt.domains,
			}
			if result := config.MatchesDomain(tt.email); result != tt.expected {
				t.Errorf("MatchesDomain(%q) = %v, 期望 %v", tt.email, result, tt.expected)
			}
		})
	}
}
