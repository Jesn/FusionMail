package adapter

import (
	"fmt"
	"testing"
	"time"
)

// TestFactory_CreateProvider 测试适配器工厂创建功能
func TestFactory_CreateProvider(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		expectType  string
	}{
		{
			name: "创建短效 Graph 适配器",
			config: &Config{
				Email:    "test@outlook.com",
				Provider: "outlook",
				Protocol: "graph_quick",
				Credentials: &Credentials{
					ClientID:     "test_client_id",
					RefreshToken: "test_refresh_token",
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
			expectType:  "*adapter.GraphQuickAdapter",
		},
		{
			name: "创建标准 Graph 适配器",
			config: &Config{
				Email:    "test@outlook.com",
				Provider: "outlook",
				Protocol: "graph",
				Credentials: &Credentials{
					AccessToken:  "test_access_token", // 标准适配器需要 AccessToken
					ClientID:     "test_client_id",
					ClientSecret: "test_client_secret",
					RefreshToken: "test_refresh_token",
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
			expectType:  "*adapter.GraphAdapter",
		},
		{
			name: "创建 IMAP 适配器",
			config: &Config{
				Email:    "test@example.com",
				Provider: "generic",
				Protocol: "imap",
				Credentials: &Credentials{
					Email:    "test@example.com",
					Password: "test_password",
					Host:     "imap.example.com",
					Port:     993,
					TLS:      true,
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
			expectType:  "*adapter.IMAPAdapter",
		},
		{
			name: "不支持的协议",
			config: &Config{
				Email:    "test@example.com",
				Provider: "generic",
				Protocol: "unsupported",
				Credentials: &Credentials{
					Email:    "test@example.com",
					Password: "test_password",
				},
				Timeout: 30 * time.Second,
			},
			expectError: true,
		},
		{
			name:        "空配置",
			config:      nil,
			expectError: true,
		},
		{
			name: "缺少凭据",
			config: &Config{
				Email:       "test@example.com",
				Provider:    "generic",
				Protocol:    "imap",
				Credentials: nil,
				Timeout:     30 * time.Second,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("Expected provider but got nil")
				return
			}

			// 验证适配器类型
			actualType := fmt.Sprintf("%T", provider)
			if actualType != tt.expectType {
				t.Errorf("Expected type %s, got %s", tt.expectType, actualType)
			}
		})
	}
}

// TestFactory_CreateProviderAuto 测试自动适配器选择
func TestFactory_CreateProviderAuto(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		expectType  string
	}{
		{
			name: "自动选择短效适配器 - 有 RefreshToken 和 ClientID，无 ClientSecret",
			config: &Config{
				Email:    "test@outlook.com",
				Provider: "outlook",
				Credentials: &Credentials{
					ClientID:     "test_client_id",
					RefreshToken: "test_refresh_token",
					// 注意：没有 ClientSecret
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
			expectType:  "*adapter.GraphQuickAdapter",
		},
		{
			name: "自动选择标准适配器 - 有完整 OAuth2 凭据",
			config: &Config{
				Email:    "test@outlook.com",
				Provider: "outlook",
				Credentials: &Credentials{
					AccessToken:  "test_access_token", // 需要 AccessToken
					ClientID:     "test_client_id",
					ClientSecret: "test_client_secret",
					RefreshToken: "test_refresh_token",
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
			expectType:  "*adapter.GraphAdapter",
		},
		{
			name: "自动选择标准适配器 - 有 AccessToken",
			config: &Config{
				Email:    "test@outlook.com",
				Provider: "outlook",
				Credentials: &Credentials{
					AccessToken: "test_access_token",
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
			expectType:  "*adapter.GraphAdapter",
		},
		{
			name: "Gmail 自动选择",
			config: &Config{
				Email:    "test@gmail.com",
				Provider: "gmail",
				Credentials: &Credentials{
					AccessToken: "test_access_token",
				},
				Timeout: 30 * time.Second,
			},
			expectError: false,
			expectType:  "*adapter.GmailAdapter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProviderAuto(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("Expected provider but got nil")
				return
			}

			// 验证适配器类型
			actualType := fmt.Sprintf("%T", provider)
			if actualType != tt.expectType {
				t.Errorf("Expected type %s, got %s", tt.expectType, actualType)
			}
		})
	}
}

// TestFactory_GetSupportedProtocols 测试获取支持的协议
func TestFactory_GetSupportedProtocols(t *testing.T) {
	factory := NewFactory()
	protocols := factory.GetSupportedProtocols()

	expectedProtocols := []string{
		"imap",
		"pop3",
		"gmail_api",
		"graph",
		"graph_quick",
	}

	if len(protocols) != len(expectedProtocols) {
		t.Errorf("Expected %d protocols, got %d", len(expectedProtocols), len(protocols))
	}

	for _, expected := range expectedProtocols {
		found := false
		for _, actual := range protocols {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected protocol %s not found", expected)
		}
	}
}

// TestFactory_GetProviderInfo 测试获取提供商信息
func TestFactory_GetProviderInfo(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		provider                string
		expectedName            string
		expectedDisplayName     string
		expectedRecommended     string
		shouldSupportGraphQuick bool
	}{
		{
			provider:                "outlook",
			expectedName:            "outlook",
			expectedDisplayName:     "Outlook / Hotmail",
			expectedRecommended:     "graph",
			shouldSupportGraphQuick: true,
		},
		{
			provider:                "gmail",
			expectedName:            "gmail",
			expectedDisplayName:     "Gmail",
			expectedRecommended:     "gmail_api",
			shouldSupportGraphQuick: false,
		},
		{
			provider:            "unknown",
			expectedName:        "generic",
			expectedDisplayName: "通用邮箱",
			expectedRecommended: "imap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			info := factory.GetProviderInfo(tt.provider)

			if info.Name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, info.Name)
			}

			if info.DisplayName != tt.expectedDisplayName {
				t.Errorf("Expected display name %s, got %s", tt.expectedDisplayName, info.DisplayName)
			}

			if info.RecommendedProtocol != tt.expectedRecommended {
				t.Errorf("Expected recommended protocol %s, got %s", tt.expectedRecommended, info.RecommendedProtocol)
			}

			if tt.shouldSupportGraphQuick {
				found := false
				for _, protocol := range info.SupportedProtocols {
					if protocol == "graph_quick" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected graph_quick protocol to be supported")
				}
			}
		})
	}
}

// TestFactory_GetRecommendedProtocol 测试获取推荐协议
func TestFactory_GetRecommendedProtocol(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		provider string
		expected string
	}{
		{"gmail", "gmail_api"},
		{"outlook", "graph"},
		{"icloud", "imap"},
		{"qq", "imap"},
		{"163", "imap"},
		{"unknown", "imap"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			protocol := factory.GetRecommendedProtocol(tt.provider)
			if protocol != tt.expected {
				t.Errorf("Expected protocol %s for provider %s, got %s", tt.expected, tt.provider, protocol)
			}
		})
	}
}
