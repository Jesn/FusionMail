package service

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// RefreshToken 刷新访问令牌
func (s *OAuth2Service) RefreshToken(ctx context.Context, req *OAuth2TokenRefreshRequest) (*OAuth2TokenRefreshResponse, error) {
	// 获取账户信息
	account, err := s.accountRepo.FindByUID(ctx, req.AccountUID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	// 解密凭证
	credentials, err := s.decryptCredentials(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 检查是否是 OAuth2 认证
	if credentials.AuthType != "oauth2" {
		return nil, fmt.Errorf("account is not using OAuth2 authentication")
	}

	// 获取 OAuth2 配置
	providerName := account.GetProviderName()
	var provider OAuth2Provider
	switch providerName {
	case "gmail":
		provider = OAuth2ProviderGoogle
	case "outlook":
		provider = OAuth2ProviderMicrosoft
	default:
		return nil, fmt.Errorf("unsupported provider for OAuth2: %s", providerName)
	}

	oauth2Config, err := s.getOAuth2Config(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 config: %w", err)
	}

	// 创建 token 对象
	token := &oauth2.Token{
		AccessToken:  credentials.AccessToken,
		RefreshToken: credentials.RefreshToken,
		Expiry:       credentials.TokenExpiry,
		TokenType:    "Bearer",
	}

	// 刷新 token
	tokenSource := oauth2Config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		s.logger.Error("Failed to refresh OAuth2 token", "account_uid", req.AccountUID, "error", err)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// 更新凭证
	credentials.AccessToken = newToken.AccessToken
	credentials.TokenExpiry = newToken.Expiry
	if newToken.RefreshToken != "" {
		credentials.RefreshToken = newToken.RefreshToken
	}

	// 加密并保存凭证
	encryptedCredentials, err := s.encryptCredentials(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	account.EncryptedCredentials = encryptedCredentials
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	s.logger.Info("OAuth2 token refreshed successfully", "account_uid", req.AccountUID)

	return &OAuth2TokenRefreshResponse{
		AccessToken: newToken.AccessToken,
		ExpiresAt:   newToken.Expiry,
	}, nil
}

// RevokeToken 撤销访问令牌
func (s *OAuth2Service) RevokeToken(ctx context.Context, accountUID string) error {
	// 获取账户信息
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	// 解密凭证
	_, err = s.decryptCredentials(account.EncryptedCredentials)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 撤销 token（这里可以调用相应的撤销 API，但通常删除本地 token 就足够了）
	s.logger.Info("OAuth2 token revoked", "account_uid", accountUID)

	// 删除账户或标记为需要重新授权
	account.Status = "auth_required"
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("failed to update account status: %w", err)
	}

	return nil
}

// ValidateMicrosoftAccount 验证 Microsoft 账户的有效性
func (s *OAuth2Service) ValidateMicrosoftAccount(ctx context.Context, refreshToken, clientID string) error {
	if refreshToken == "" {
		return fmt.Errorf("refresh token is required")
	}

	if clientID == "" {
		return fmt.Errorf("client ID is required")
	}

	// 使用传入的 clientID 创建 OAuth2 配置
	// 批量导入的账户使用自己的 clientID，而不是供应商配置中的
	oauth2Config := &oauth2.Config{
		ClientID: clientID,
		// Microsoft 公共客户端不需要 ClientSecret
		ClientSecret: "",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
		},
		Scopes: []string{
			"offline_access",
			"https://graph.microsoft.com/Mail.Read",
			"https://graph.microsoft.com/Mail.Send",
			"https://graph.microsoft.com/User.Read",
		},
	}

	// 创建 token 对象
	token := &oauth2.Token{
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}

	// 尝试刷新 token 来验证其有效性
	tokenSource := oauth2Config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		s.logger.Error("Failed to validate Microsoft account", "error", err, "client_id", clientID)
		return fmt.Errorf("invalid Microsoft account: failed to refresh token - %w", err)
	}

	// 检查新的访问令牌是否有效
	if newToken.AccessToken == "" {
		return fmt.Errorf("invalid Microsoft account: no access token received")
	}

	s.logger.Info("Microsoft account validation successful",
		"client_id", clientID,
		"has_new_refresh_token", newToken.RefreshToken != "",
		"token_expires_at", newToken.Expiry)

	return nil
}
