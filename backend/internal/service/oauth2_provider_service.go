package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"fusionmail/internal/model"

	"golang.org/x/oauth2"
)

// generateState 生成随机 state 参数
func (s *OAuth2Service) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// getOAuth2Config 获取 OAuth2 配置（完全基于数据库）
func (s *OAuth2Service) getOAuth2Config(provider OAuth2Provider) (*oauth2.Config, error) {
	s.logger.Info("Using OAuth2 config from database", "provider", provider)

	// 将 provider 类型转换为枚举值
	var providerType int
	switch provider {
	case OAuth2ProviderGoogle:
		providerType = int(model.ProviderTypeGmail)
	case OAuth2ProviderMicrosoft:
		providerType = int(model.ProviderTypeOutlook)
	default:
		return nil, fmt.Errorf("unsupported OAuth2 provider: %s", provider)
	}

	// 从数据库获取配置
	return s.oauth2ConfigProvider.GetOAuth2Config(context.Background(), providerType)
}

// getUserInfo 获取用户信息
func (s *OAuth2Service) getUserInfo(ctx context.Context, provider OAuth2Provider, token *oauth2.Token) (map[string]interface{}, error) {
	switch provider {
	case OAuth2ProviderGoogle:
		return s.getGoogleUserInfo(ctx, token)
	case OAuth2ProviderMicrosoft:
		return s.getMicrosoftUserInfo(ctx, token)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// getGoogleUserInfo 获取 Google 用户信息
func (s *OAuth2Service) getGoogleUserInfo(ctx context.Context, token *oauth2.Token) (map[string]interface{}, error) {
	s.logger.Debug("Fetching Google user info",
		"token_type", token.TokenType,
		"expires_at", token.Expiry)

	// 使用 Google API 获取用户信息
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	s.logger.Debug("Making request to Google userinfo API", "url", "https://www.googleapis.com/oauth2/v2/userinfo")

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		s.logger.Error("Failed to request Google user info",
			"url", "https://www.googleapis.com/oauth2/v2/userinfo",
			"error", err)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	s.logger.Debug("Google userinfo API response received",
		"status_code", resp.StatusCode,
		"content_type", resp.Header.Get("Content-Type"))

	if resp.StatusCode != 200 {
		s.logger.Error("Google userinfo API returned error status",
			"status_code", resp.StatusCode,
			"status", resp.Status)
		return nil, fmt.Errorf("Google API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		s.logger.Error("Failed to decode Google user info response", "error", err)
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	s.logger.Info("Google user info retrieved successfully",
		"email", userInfo["email"],
		"verified_email", userInfo["verified_email"],
		"name", userInfo["name"])

	return userInfo, nil
}

// getMicrosoftUserInfo 获取 Microsoft 用户信息
func (s *OAuth2Service) getMicrosoftUserInfo(ctx context.Context, token *oauth2.Token) (map[string]interface{}, error) {
	s.logger.Debug("Fetching Microsoft user info",
		"token_type", token.TokenType,
		"expires_at", token.Expiry)

	// 使用 Microsoft Graph API 获取用户信息
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	s.logger.Debug("Making request to Microsoft Graph API", "url", "https://graph.microsoft.com/v1.0/me")

	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		s.logger.Error("Failed to request Microsoft user info",
			"url", "https://graph.microsoft.com/v1.0/me",
			"error", err)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	s.logger.Debug("Microsoft Graph API response received",
		"status_code", resp.StatusCode,
		"content_type", resp.Header.Get("Content-Type"))

	if resp.StatusCode != 200 {
		s.logger.Error("Microsoft Graph API returned error status",
			"status_code", resp.StatusCode,
			"status", resp.Status)
		return nil, fmt.Errorf("Microsoft Graph API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		s.logger.Error("Failed to decode Microsoft user info response", "error", err)
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// 添加账户类型识别
	accountType := "personal" // 默认为个人账户
	if userPrincipalName, ok := userInfo["userPrincipalName"].(string); ok {
		// 简单的账户类型识别逻辑
		if userInfo["jobTitle"] != nil || userInfo["companyName"] != nil || userInfo["department"] != nil {
			accountType = "work"
		}
		// 基于域名的识别
		if len(userPrincipalName) > 0 {
			// 个人账户通常使用 @outlook.com, @hotmail.com, @live.com 等域名
			personalDomains := []string{"outlook.com", "hotmail.com", "live.com", "msn.com"}
			isPersonal := false
			for _, domain := range personalDomains {
				if len(userPrincipalName) > len(domain) && userPrincipalName[len(userPrincipalName)-len(domain):] == domain {
					isPersonal = true
					break
				}
			}
			if !isPersonal {
				accountType = "work"
			}
		}
	}

	// 添加账户类型到用户信息中
	userInfo["account_type"] = accountType

	s.logger.Info("Microsoft user info retrieved successfully",
		"email", userInfo["mail"],
		"user_principal_name", userInfo["userPrincipalName"],
		"display_name", userInfo["displayName"],
		"account_type", accountType,
		"job_title", userInfo["jobTitle"],
		"company_name", userInfo["companyName"])

	return userInfo, nil
}
