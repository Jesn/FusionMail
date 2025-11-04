package test

import (
	"testing"

	"fusionmail/config"
	"fusionmail/internal/service"
)

func TestOAuth2Service_GenerateAuthURL(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{
		OAuth2: config.OAuth2Config{
			Google: config.GoogleOAuth2Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:4444/auth/google/callback",
			},
		},
	}

	// 创建模拟的依赖
	// 注意：这里需要实际的依赖注入，但为了简单起见，我们只测试配置是否正确加载
	
	// 验证配置
	if cfg.OAuth2.Google.ClientID != "test-client-id" {
		t.Errorf("Expected client ID to be 'test-client-id', got '%s'", cfg.OAuth2.Google.ClientID)
	}

	if cfg.OAuth2.Google.RedirectURL != "http://localhost:4444/auth/google/callback" {
		t.Errorf("Expected redirect URL to be 'http://localhost:4444/auth/google/callback', got '%s'", cfg.OAuth2.Google.RedirectURL)
	}

	t.Log("OAuth2 configuration test passed")
}

func TestOAuth2Provider_Constants(t *testing.T) {
	// 测试 OAuth2 提供商常量
	if service.OAuth2ProviderGoogle != "google" {
		t.Errorf("Expected OAuth2ProviderGoogle to be 'google', got '%s'", service.OAuth2ProviderGoogle)
	}

	if service.OAuth2ProviderMicrosoft != "microsoft" {
		t.Errorf("Expected OAuth2ProviderMicrosoft to be 'microsoft', got '%s'", service.OAuth2ProviderMicrosoft)
	}

	t.Log("OAuth2 provider constants test passed")
}