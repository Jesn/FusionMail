package adapter

import (
	"testing"
)

// TestParseQuickAccountString 测试短效账户字符串解析
func TestParseQuickAccountString(t *testing.T) {
	tests := []struct {
		name             string
		accountString    string
		expectError      bool
		expectedEmail    string
		expectedProvider string
	}{
		{
			name:             "有效的 Outlook 账户",
			accountString:    "user@outlook.com----password123----refresh_token_abc----client_id_xyz",
			expectError:      false,
			expectedEmail:    "user@outlook.com",
			expectedProvider: "outlook",
		},
		{
			name:             "有效的 Gmail 账户",
			accountString:    "user@gmail.com----app_password----refresh_token_def----client_id_123",
			expectError:      false,
			expectedEmail:    "user@gmail.com",
			expectedProvider: "gmail",
		},
		{
			name:             "有效的 QQ 邮箱",
			accountString:    "user@qq.com--------refresh_token_ghi----client_id_456",
			expectError:      false,
			expectedEmail:    "user@qq.com",
			expectedProvider: "qq",
		},
		{
			name:             "通用邮箱",
			accountString:    "user@example.com----password----refresh_token_jkl----client_id_789",
			expectError:      false,
			expectedEmail:    "user@example.com",
			expectedProvider: "generic",
		},
		{
			name:          "空字符串",
			accountString: "",
			expectError:   true,
		},
		{
			name:          "字段数量不足",
			accountString: "user@outlook.com----password----refresh_token",
			expectError:   true,
		},
		{
			name:          "字段数量过多",
			accountString: "user@outlook.com----password----refresh_token----client_id----extra",
			expectError:   true,
		},
		{
			name:          "缺少邮箱",
			accountString: "----password----refresh_token----client_id",
			expectError:   true,
		},
		{
			name:          "缺少 refresh_token",
			accountString: "user@outlook.com----password--------client_id",
			expectError:   true,
		},
		{
			name:          "缺少 client_id",
			accountString: "user@outlook.com----password----refresh_token----",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseQuickAccountString(tt.accountString)

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

			if config == nil {
				t.Error("Expected config but got nil")
				return
			}

			if config.Email != tt.expectedEmail {
				t.Errorf("Expected email %s, got %s", tt.expectedEmail, config.Email)
			}

			if config.Provider != tt.expectedProvider {
				t.Errorf("Expected provider %s, got %s", tt.expectedProvider, config.Provider)
			}

			if config.AuthType != "quick" {
				t.Errorf("Expected auth type 'quick', got %s", config.AuthType)
			}

			if config.Credentials == nil {
				t.Error("Expected credentials but got nil")
				return
			}

			if config.Credentials.RefreshToken == "" {
				t.Error("Expected refresh token but got empty")
			}

			if config.Credentials.ClientID == "" {
				t.Error("Expected client ID but got empty")
			}
		})
	}
}

// TestInferProviderFromEmail 测试从邮箱地址推断提供商
func TestInferProviderFromEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		// Microsoft 系列
		{"user@outlook.com", "outlook"},
		{"user@hotmail.com", "outlook"},
		{"user@live.com", "outlook"},
		{"user@msn.com", "outlook"},

		// Google 系列
		{"user@gmail.com", "gmail"},
		{"user@googlemail.com", "gmail"},

		// Apple 系列
		{"user@icloud.com", "icloud"},
		{"user@me.com", "icloud"},
		{"user@mac.com", "icloud"},

		// 中国邮箱
		{"user@qq.com", "qq"},
		{"user@foxmail.com", "qq"},
		{"user@163.com", "163"},
		{"user@126.com", "163"},
		{"user@yeah.net", "163"},

		// 通用邮箱
		{"user@example.com", "generic"},
		{"user@company.org", "generic"},

		// 边界情况
		{"", "generic"},
		{"invalid-email", "generic"},
		{"@domain.com", "generic"},
		{"user@", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := inferProviderFromEmail(tt.email)
			if result != tt.expected {
				t.Errorf("Expected provider %s for email %s, got %s", tt.expected, tt.email, result)
			}
		})
	}
}

// TestFormatQuickAccountString 测试格式化短效账户字符串
func TestFormatQuickAccountString(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		expected    string
	}{
		{
			name: "完整配置",
			config: &Config{
				Email:    "user@outlook.com",
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					Email:        "user@outlook.com",
					Password:     "password123",
					RefreshToken: "refresh_token_abc",
					ClientID:     "client_id_xyz",
				},
			},
			expectError: false,
			expected:    "user@outlook.com----password123----refresh_token_abc----client_id_xyz",
		},
		{
			name: "无密码配置",
			config: &Config{
				Email:    "user@gmail.com",
				Provider: "gmail",
				AuthType: "quick",
				Credentials: &Credentials{
					Email:        "user@gmail.com",
					RefreshToken: "refresh_token_def",
					ClientID:     "client_id_123",
				},
			},
			expectError: false,
			expected:    "user@gmail.com--------refresh_token_def----client_id_123",
		},
		{
			name:        "空配置",
			config:      nil,
			expectError: true,
		},
		{
			name: "缺少凭据",
			config: &Config{
				Email:       "user@outlook.com",
				Provider:    "outlook",
				AuthType:    "quick",
				Credentials: nil,
			},
			expectError: true,
		},
		{
			name: "缺少邮箱",
			config: &Config{
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					RefreshToken: "refresh_token",
					ClientID:     "client_id",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatQuickAccountString(tt.config)

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

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestValidateQuickAccountConfig 测试短效账户配置验证
func TestValidateQuickAccountConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "有效配置",
			config: &Config{
				Email:    "user@outlook.com",
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					RefreshToken: "refresh_token",
					ClientID:     "client_id",
				},
			},
			expectError: false,
		},
		{
			name:        "空配置",
			config:      nil,
			expectError: true,
		},
		{
			name: "缺少凭据",
			config: &Config{
				Email:       "user@outlook.com",
				Provider:    "outlook",
				AuthType:    "quick",
				Credentials: nil,
			},
			expectError: true,
		},
		{
			name: "无效邮箱格式",
			config: &Config{
				Email:    "invalid-email",
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					RefreshToken: "refresh_token",
					ClientID:     "client_id",
				},
			},
			expectError: true,
		},
		{
			name: "缺少 refresh_token",
			config: &Config{
				Email:    "user@outlook.com",
				Provider: "outlook",
				AuthType: "quick",
				Credentials: &Credentials{
					ClientID: "client_id",
				},
			},
			expectError: true,
		},
		{
			name: "错误的认证类型",
			config: &Config{
				Email:    "user@outlook.com",
				Provider: "outlook",
				AuthType: "standard",
				Credentials: &Credentials{
					RefreshToken: "refresh_token",
					ClientID:     "client_id",
				},
			},
			expectError: true,
		},
		{
			name: "不支持的提供商",
			config: &Config{
				Email:    "user@example.com",
				Provider: "unsupported",
				AuthType: "quick",
				Credentials: &Credentials{
					RefreshToken: "refresh_token",
					ClientID:     "client_id",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuickAccountConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestParseBatchQuickAccounts 测试批量解析短效账户
func TestParseBatchQuickAccounts(t *testing.T) {
	accountStrings := []string{
		"user1@outlook.com----password1----refresh1----client1",
		"user2@gmail.com----password2----refresh2----client2",
		"invalid-string",
		"user3@qq.com----password3----refresh3----client3",
		"----missing-email----refresh4----client4",
	}

	configs, errors := ParseBatchQuickAccounts(accountStrings)

	// 应该有 3 个有效配置
	if len(configs) != 3 {
		t.Errorf("Expected 3 valid configs, got %d", len(configs))
	}

	// 应该有 2 个错误
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// 验证有效配置
	expectedEmails := []string{"user1@outlook.com", "user2@gmail.com", "user3@qq.com"}
	for i, config := range configs {
		if config.Email != expectedEmails[i] {
			t.Errorf("Expected email %s at index %d, got %s", expectedEmails[i], i, config.Email)
		}
	}
}

// TestExtractQuickAccountInfo 测试提取账户信息
func TestExtractQuickAccountInfo(t *testing.T) {
	tests := []struct {
		name          string
		accountString string
		expectedValid bool
		expectedEmail string
	}{
		{
			name:          "有效账户",
			accountString: "user@outlook.com----password----refresh_token----client_id",
			expectedValid: true,
			expectedEmail: "user@outlook.com",
		},
		{
			name:          "无效账户",
			accountString: "invalid-string",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ExtractQuickAccountInfo(tt.accountString)

			if info.IsValid != tt.expectedValid {
				t.Errorf("Expected valid %v, got %v", tt.expectedValid, info.IsValid)
			}

			if tt.expectedValid && info.Email != tt.expectedEmail {
				t.Errorf("Expected email %s, got %s", tt.expectedEmail, info.Email)
			}

			if !tt.expectedValid && info.ErrorMessage == "" {
				t.Error("Expected error message for invalid account")
			}
		})
	}
}

// TestIsQuickAuthSupported 测试短效认证支持检查
func TestIsQuickAuthSupported(t *testing.T) {
	tests := []struct {
		provider string
		expected bool
	}{
		{"outlook", true},
		{"gmail", false},   // 目前不支持
		{"icloud", false},  // 目前不支持
		{"qq", false},      // 目前不支持
		{"163", false},     // 目前不支持
		{"generic", false}, // 目前不支持
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := IsQuickAuthSupported(tt.provider)
			if result != tt.expected {
				t.Errorf("Expected %v for provider %s, got %v", tt.expected, tt.provider, result)
			}
		})
	}
}
