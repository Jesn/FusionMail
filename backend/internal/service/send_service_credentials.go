package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// parseCredentials 解析账户凭证
// 从加密的凭证数据中提取 OAuth2 或密码凭证
func (s *SendService) parseCredentials(ctx context.Context, account *model.EmailAccount) (*adapter.AccountCredentials, error) {
	// 如果没有加密凭证，返回 nil（SMTP 发送可能不需要凭证）
	if account.EncryptedCredentials == "" {
		return nil, nil
	}

	// 解密凭证数据
	decryptedData, err := s.cryptoService.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 解析认证类型：优先从 JSON 数据中读取，fallback 到 adapter 配置
	var credAuthType struct {
		AuthType string `json:"auth_type"`
	}
	authType := account.GetAuthType()
	if json.Unmarshal(decryptedData, &credAuthType) == nil && credAuthType.AuthType != "" {
		authType = credAuthType.AuthType
	}

	switch authType {
	case "quick":
		// quick 导入：仅有 refresh_token + client_id，access_token 运行时刷新
		var quickCreds struct {
			RefreshToken string `json:"refresh_token"`
			ClientID     string `json:"client_id"`
		}
		if err := json.Unmarshal(decryptedData, &quickCreds); err != nil {
			return nil, fmt.Errorf("failed to parse quick credentials: %w", err)
		}
		return &adapter.AccountCredentials{
			RefreshToken: quickCreds.RefreshToken,
			ClientID:     quickCreds.ClientID,
		}, nil

	case "oauth2":
		// OAuth2 凭证是 JSON 格式
		var oauthCreds struct {
			Email        string    `json:"email"`
			AuthType     string    `json:"auth_type"`
			AccessToken  string    `json:"access_token"`
			RefreshToken string    `json:"refresh_token"`
			TokenExpiry  time.Time `json:"token_expiry"`
			ClientID     string    `json:"client_id"`
		}

		if err := json.Unmarshal(decryptedData, &oauthCreds); err != nil {
			return nil, fmt.Errorf("failed to parse OAuth2 credentials: %w", err)
		}

		credentials := &adapter.AccountCredentials{
			AccessToken:  oauthCreds.AccessToken,
			RefreshToken: oauthCreds.RefreshToken,
			TokenExpiry:  oauthCreds.TokenExpiry,
			ClientID:     oauthCreds.ClientID,
		}

		// 如果 JSON 中没有 ClientID，从 OAuth2 配置中获取
		if credentials.ClientID == "" {
			if account.ProviderID > 0 {
				oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2ConfigByProviderID(ctx, account.ProviderID)
				if err != nil {
					s.logger.Warn("获取 OAuth2 配置失败: provider_id=%d, error=%v", account.ProviderID, err)
				} else if oauth2Config != nil {
					credentials.ClientID = oauth2Config.ClientID
					credentials.ClientSecret = oauth2Config.ClientSecret
				}
			} else {
				providerName := account.GetProviderName()
				if providerName != "" {
					oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2ConfigByName(ctx, providerName)
					if err != nil {
						s.logger.Warn("获取 OAuth2 配置失败: provider=%s, error=%v", providerName, err)
					} else if oauth2Config != nil {
						credentials.ClientID = oauth2Config.ClientID
						credentials.ClientSecret = oauth2Config.ClientSecret
					}
				}
			}
		}

		return credentials, nil
	}

	// 密码认证：凭证就是密码本身
	// 对于 SMTP 发送，密码已经存储在 EncryptedSMTPPassword 中
	// 这里返回 nil，让 SenderFactory 使用 SMTP 配置
	return nil, nil
}
