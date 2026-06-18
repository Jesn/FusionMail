package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"fusionmail/internal/model"

	"golang.org/x/oauth2"
)

// reauthorizeAccount 重新授权账户（更新现有账户的 token）
func (s *OAuth2Service) reauthorizeAccount(ctx context.Context, accountUID string, token *oauth2.Token) (*model.EmailAccount, error) {
	// 获取现有账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	// 解密现有凭证
	credentials, err := s.decryptCredentials(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 更新 token
	credentials.AccessToken = token.AccessToken
	credentials.RefreshToken = token.RefreshToken
	credentials.TokenExpiry = token.Expiry

	// 加密凭证
	encryptedCredentials, err := s.encryptCredentials(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// 更新账户状态
	account.EncryptedCredentials = encryptedCredentials
	account.Status = "active"
	account.LastSyncError = ""          // 清除同步错误
	account.ConsecutiveAuthFailures = 0 // 重置认证失败计数
	account.DisableReason = ""          // 清除禁用原因
	account.AutoDisabledAt = nil        // 清除自动禁用时间

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	s.logger.Info("Account reauthorized and status reset",
		"account_uid", accountUID,
		"email", account.Email)

	return account, nil
}

// createOrUpdateAccount 创建或更新账户
func (s *OAuth2Service) createOrUpdateAccount(ctx context.Context, provider OAuth2Provider, userInfo map[string]interface{}, token *oauth2.Token, groupID *int64) (*model.EmailAccount, error) {
	s.logger.Debug("Processing user info for account creation/update",
		"provider", provider,
		"user_info_keys", getMapKeys(userInfo),
		"user_info", userInfo)

	// 尝试从不同字段获取邮箱地址
	var email string
	var ok bool

	// 优先尝试 "email" 字段（Google使用）
	if email, ok = userInfo["email"].(string); !ok {
		// 如果没有 "email" 字段，尝试 "mail" 字段（Microsoft Graph使用）
		if email, ok = userInfo["mail"].(string); !ok {
			// 如果都没有，尝试 "userPrincipalName" 字段（Microsoft Graph备选）
			if email, ok = userInfo["userPrincipalName"].(string); !ok {
				s.logger.Error("Email not found or invalid in user info",
					"provider", provider,
					"user_info", userInfo,
					"email_value", userInfo["email"],
					"mail_value", userInfo["mail"],
					"userPrincipalName_value", userInfo["userPrincipalName"])
				return nil, fmt.Errorf("email not found in user info")
			}
		}
	}

	s.logger.Info("Email extracted from user info",
		"provider", provider,
		"email", email)

	// 检查账户是否已存在
	s.logger.Debug("Checking if account already exists", "email", email)

	existingAccount, err := s.accountRepo.FindByEmail(ctx, email)
	if err == nil && existingAccount != nil {
		s.logger.Info("Existing account found, updating token",
			"email", email,
			"account_uid", existingAccount.UID)
		// 更新现有账户的 token
		return s.updateAccountToken(ctx, existingAccount, token)
	}

	if err != nil {
		s.logger.Debug("Error finding account by email", "email", email, "error", err)
	}

	// 如果没有找到激活账号，检查是否存在软删除的账号并清理
	deletedAccounts, derr := s.accountRepo.FindDeletedByEmail(ctx, email)
	if derr != nil {
		s.logger.Error("Failed to find soft-deleted accounts by email",
			"email", email,
			"error", derr)
		return nil, fmt.Errorf("failed to check soft-deleted accounts: %w", derr)
	}

	if len(deletedAccounts) > 0 {
		s.logger.Info("Soft-deleted accounts found, cleaning up before creating new one",
			"email", email,
			"count", len(deletedAccounts))

		for _, acc := range deletedAccounts {
			if acc == nil {
				continue
			}

			// 删除该账号的所有邮件
			if err := s.emailRepo.DeleteByAccountUID(ctx, acc.UID); err != nil {
				s.logger.Error("Failed to delete emails for soft-deleted account",
					"email", acc.Email,
					"account_uid", acc.UID,
					"error", err)
				return nil, fmt.Errorf("failed to delete emails for soft-deleted account: %w", err)
			}

			// 永久删除软删除账号
			if err := s.accountRepo.ForceDelete(ctx, acc.UID); err != nil {
				s.logger.Error("Failed to force delete soft-deleted account",
					"email", acc.Email,
					"account_uid", acc.UID,
					"error", err)
				return nil, fmt.Errorf("failed to force delete soft-deleted account: %w", err)
			}
		}
	}

	s.logger.Info("No existing account found, creating new account",
		"email", email,
		"provider", provider)

	// 创建新账户
	return s.createNewAccount(ctx, provider, email, userInfo, token, groupID)
}

// getMapKeys 获取 map 的所有键（用于日志记录）
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// updateAccountToken 更新账户 token
func (s *OAuth2Service) updateAccountToken(ctx context.Context, account *model.EmailAccount, token *oauth2.Token) (*model.EmailAccount, error) {
	// 防御性检查
	if account == nil {
		return nil, fmt.Errorf("account cannot be nil")
	}

	// 解密现有凭证
	credentials, err := s.decryptCredentials(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 更新 token
	credentials.AccessToken = token.AccessToken
	credentials.RefreshToken = token.RefreshToken
	credentials.TokenExpiry = token.Expiry

	// 加密凭证
	encryptedCredentials, err := s.encryptCredentials(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// 更新账户
	account.EncryptedCredentials = encryptedCredentials
	account.Status = "active"

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	return account, nil
}

// createNewAccount 创建新账户
func (s *OAuth2Service) createNewAccount(ctx context.Context, provider OAuth2Provider, email string, userInfo map[string]interface{}, token *oauth2.Token, groupID *int64) (*model.EmailAccount, error) {
	// 生成账户 UID
	accountUID := generateUID()

	// 创建凭证
	credentials := &Credentials{
		Email:        email,
		AuthType:     "oauth2",
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
	}

	// 加密凭证
	encryptedCredentials, err := s.encryptCredentials(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// 确定提供商名称和适配器名称
	var providerName, adapterName string
	switch provider {
	case OAuth2ProviderGoogle:
		providerName = "gmail"
		adapterName = "gmail"
	case OAuth2ProviderMicrosoft:
		providerName = "outlook"
		adapterName = "graph"
	}

	// 查找 Provider
	providerConfig, err := s.providerRepo.FindByName(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider '%s': %w", providerName, err)
	}

	// 查找 Adapter
	adapterConfig, err := s.adapterRepo.FindByName(ctx, adapterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapter '%s': %w", adapterName, err)
	}

	// 创建账户（使用 ProviderID 和 AdapterID）
	account := &model.EmailAccount{
		UID:                  accountUID,
		Email:                email,
		ProviderID:           providerConfig.ID,
		AdapterID:            adapterConfig.ID,
		EncryptedCredentials: encryptedCredentials,
		Status:               "active",
		SyncEnabled:          true,
		SyncInterval:         2,
		GroupID:              groupID, // 设置分组 ID
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	s.logger.Info("OAuth2 account created successfully", "account_uid", accountUID, "email", email, "provider", providerName)

	return account, nil
}

// Credentials 凭证结构（用于序列化）
type Credentials struct {
	Email        string    `json:"email"`
	AuthType     string    `json:"auth_type"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenExpiry  time.Time `json:"token_expiry,omitempty"`
	Password     string    `json:"password,omitempty"`
}

// encryptCredentials 加密凭证
func (s *OAuth2Service) encryptCredentials(credentials *Credentials) (string, error) {
	data, err := json.Marshal(credentials)
	if err != nil {
		return "", fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, err := s.cryptoService.Encrypt(data)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	return encrypted, nil
}

// decryptCredentials 解密凭证
func (s *OAuth2Service) decryptCredentials(encryptedData string) (*Credentials, error) {
	data, err := s.cryptoService.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	var credentials Credentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &credentials, nil
}

// generateUID 生成唯一标识符
func generateUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
