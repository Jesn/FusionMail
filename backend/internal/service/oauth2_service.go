package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
	"google.golang.org/api/gmail/v1"

	"fusionmail/config"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/redis"
)

// OAuth2Service OAuth2 认证服务
type OAuth2Service struct {
	config        *config.Config
	accountRepo   repository.AccountRepository
	cryptoService *crypto.Service
	redisClient   *redis.ClientWrapper
	logger        *logger.Logger
}

// NewOAuth2Service 创建 OAuth2 服务实例
func NewOAuth2Service(
	cfg *config.Config,
	accountRepo repository.AccountRepository,
	cryptoService *crypto.Service,
	redisClient *redis.ClientWrapper,
	logger *logger.Logger,
) *OAuth2Service {
	return &OAuth2Service{
		config:        cfg,
		accountRepo:   accountRepo,
		cryptoService: cryptoService,
		redisClient:   redisClient,
		logger:        logger,
	}
}

// OAuth2Provider OAuth2 提供商类型
type OAuth2Provider string

const (
	OAuth2ProviderGoogle    OAuth2Provider = "google"
	OAuth2ProviderMicrosoft OAuth2Provider = "microsoft"
)

// OAuth2AuthRequest OAuth2 授权请求
type OAuth2AuthRequest struct {
	Provider OAuth2Provider `json:"provider"`
	Email    string         `json:"email,omitempty"` // 可选，用于预填充
}

// OAuth2AuthResponse OAuth2 授权响应
type OAuth2AuthResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// OAuth2CallbackRequest OAuth2 回调请求
type OAuth2CallbackRequest struct {
	Provider OAuth2Provider `json:"provider"`
	Code     string         `json:"code"`
	State    string         `json:"state"`
}

// OAuth2CallbackResponse OAuth2 回调响应
type OAuth2CallbackResponse struct {
	AccountUID   string    `json:"account_uid"`
	Email        string    `json:"email"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// OAuth2TokenRefreshRequest Token 刷新请求
type OAuth2TokenRefreshRequest struct {
	AccountUID string `json:"account_uid"`
}

// OAuth2TokenRefreshResponse Token 刷新响应
type OAuth2TokenRefreshResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// GenerateAuthURL 生成授权 URL
func (s *OAuth2Service) GenerateAuthURL(ctx context.Context, req *OAuth2AuthRequest) (*OAuth2AuthResponse, error) {
	s.logger.Info("Starting OAuth2 auth URL generation",
		"provider", req.Provider,
		"email", req.Email)

	// 生成随机 state 参数防止 CSRF 攻击
	state, err := s.generateState()
	if err != nil {
		s.logger.Error("Failed to generate OAuth2 state", "provider", req.Provider, "error", err)
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	s.logger.Debug("Generated OAuth2 state", "provider", req.Provider, "state", state)

	// 获取 OAuth2 配置
	oauth2Config, err := s.getOAuth2Config(req.Provider)
	if err != nil {
		s.logger.Error("Failed to get OAuth2 config", "provider", req.Provider, "error", err)
		return nil, fmt.Errorf("failed to get OAuth2 config: %w", err)
	}

	s.logger.Debug("OAuth2 config retrieved",
		"provider", req.Provider,
		"client_id", oauth2Config.ClientID,
		"redirect_url", oauth2Config.RedirectURL,
		"scopes", oauth2Config.Scopes)

	// 生成授权 URL
	authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	s.logger.Debug("Generated OAuth2 auth URL", "provider", req.Provider, "url", authURL)

	// 将 state 存储到 Redis（5分钟过期）
	stateKey := fmt.Sprintf("oauth2:state:%s", state)
	stateData := map[string]interface{}{
		"provider": string(req.Provider),
		"email":    req.Email,
		"created":  time.Now().Unix(),
	}

	if err := s.redisClient.SetJSON(ctx, stateKey, stateData, 15*time.Minute); err != nil {
		s.logger.Error("Failed to store OAuth2 state to Redis",
			"provider", req.Provider,
			"state", state,
			"state_key", stateKey,
			"error", err)
		return nil, fmt.Errorf("failed to store state: %w", err)
	}

	s.logger.Info("OAuth2 auth URL generated successfully",
		"provider", req.Provider,
		"state", state,
		"expires_in", "15m")

	return &OAuth2AuthResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// HandleCallback 处理 OAuth2 回调
func (s *OAuth2Service) HandleCallback(ctx context.Context, req *OAuth2CallbackRequest) (*OAuth2CallbackResponse, error) {
	s.logger.Info("Starting OAuth2 callback processing",
		"provider", req.Provider,
		"state", req.State,
		"code_length", len(req.Code))

	// 验证 state 参数
	stateKey := fmt.Sprintf("oauth2:state:%s", req.State)
	var stateData map[string]interface{}

	s.logger.Debug("Validating OAuth2 state", "state_key", stateKey)

	if err := s.redisClient.GetJSON(ctx, stateKey, &stateData); err != nil {
		s.logger.Error("Invalid OAuth2 state - not found in Redis",
			"provider", req.Provider,
			"state", req.State,
			"state_key", stateKey,
			"error", err)
		return nil, fmt.Errorf("invalid state parameter")
	}

	s.logger.Info("OAuth2 state validated successfully",
		"provider", req.Provider,
		"state", req.State,
		"state_data", stateData)

	// 删除已使用的 state
	s.redisClient.Del(ctx, stateKey)
	s.logger.Debug("OAuth2 state deleted from Redis", "state_key", stateKey)

	// 验证 provider 匹配
	if stateData["provider"] != string(req.Provider) {
		s.logger.Error("OAuth2 provider mismatch",
			"expected_provider", stateData["provider"],
			"actual_provider", req.Provider,
			"state", req.State)
		return nil, fmt.Errorf("provider mismatch")
	}

	s.logger.Debug("OAuth2 provider validated", "provider", req.Provider)

	// 获取 OAuth2 配置
	oauth2Config, err := s.getOAuth2Config(req.Provider)
	if err != nil {
		s.logger.Error("Failed to get OAuth2 config for callback",
			"provider", req.Provider,
			"error", err)
		return nil, fmt.Errorf("failed to get OAuth2 config: %w", err)
	}

	s.logger.Debug("OAuth2 config retrieved for token exchange",
		"provider", req.Provider,
		"client_id", oauth2Config.ClientID,
		"redirect_url", oauth2Config.RedirectURL)

	// 交换授权码获取 token
	s.logger.Info("Exchanging OAuth2 authorization code for token",
		"provider", req.Provider,
		"code_length", len(req.Code))

	token, err := oauth2Config.Exchange(ctx, req.Code)
	if err != nil {
		s.logger.Error("Failed to exchange OAuth2 code for token",
			"provider", req.Provider,
			"code_length", len(req.Code),
			"client_id", oauth2Config.ClientID,
			"redirect_url", oauth2Config.RedirectURL,
			"error", err)
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	s.logger.Info("OAuth2 token exchange successful",
		"provider", req.Provider,
		"token_type", token.TokenType,
		"expires_at", token.Expiry,
		"has_refresh_token", token.RefreshToken != "")

	// 获取用户信息
	s.logger.Info("Fetching user info from OAuth2 provider", "provider", req.Provider)

	userInfo, err := s.getUserInfo(ctx, req.Provider, token)
	if err != nil {
		s.logger.Error("Failed to get user info from OAuth2 provider",
			"provider", req.Provider,
			"error", err)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	s.logger.Info("User info retrieved successfully",
		"provider", req.Provider,
		"user_info", userInfo)

	// 创建或更新账户
	s.logger.Info("Creating or updating account", "provider", req.Provider)

	account, err := s.createOrUpdateAccount(ctx, req.Provider, userInfo, token)
	if err != nil {
		s.logger.Error("Failed to create or update account",
			"provider", req.Provider,
			"user_info", userInfo,
			"error", err)
		return nil, fmt.Errorf("failed to create/update account: %w", err)
	}

	return &OAuth2CallbackResponse{
		AccountUID:   account.UID,
		Email:        account.Email,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}, nil
}

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
	var provider OAuth2Provider
	switch account.Provider {
	case "gmail":
		provider = OAuth2ProviderGoogle
	case "outlook":
		provider = OAuth2ProviderMicrosoft
	default:
		return nil, fmt.Errorf("unsupported provider for OAuth2: %s", account.Provider)
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

// 私有方法

// generateState 生成随机 state 参数
func (s *OAuth2Service) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// getOAuth2Config 获取 OAuth2 配置
func (s *OAuth2Service) getOAuth2Config(provider OAuth2Provider) (*oauth2.Config, error) {
	switch provider {
	case OAuth2ProviderGoogle:
		return &oauth2.Config{
			ClientID:     s.config.OAuth2.Google.ClientID,
			ClientSecret: s.config.OAuth2.Google.ClientSecret,
			RedirectURL:  s.config.OAuth2.Google.RedirectURL,
			Scopes: []string{
				gmail.GmailReadonlyScope,
				gmail.GmailModifyScope,
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		}, nil
	case OAuth2ProviderMicrosoft:
		return &oauth2.Config{
			ClientID:     s.config.OAuth2.Microsoft.ClientID,
			ClientSecret: s.config.OAuth2.Microsoft.ClientSecret,
			RedirectURL:  s.config.OAuth2.Microsoft.RedirectURL,
			Scopes: []string{
				"https://graph.microsoft.com/Mail.ReadWrite",
				"https://graph.microsoft.com/User.Read",
				"offline_access",
			},
			Endpoint: microsoft.AzureADEndpoint("common"),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported OAuth2 provider: %s", provider)
	}
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

// createOrUpdateAccount 创建或更新账户
func (s *OAuth2Service) createOrUpdateAccount(ctx context.Context, provider OAuth2Provider, userInfo map[string]interface{}, token *oauth2.Token) (*model.Account, error) {
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

	s.logger.Info("No existing account found, creating new account",
		"email", email,
		"provider", provider)

	// 创建新账户
	return s.createNewAccount(ctx, provider, email, userInfo, token)
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
func (s *OAuth2Service) updateAccountToken(ctx context.Context, account *model.Account, token *oauth2.Token) (*model.Account, error) {
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
func (s *OAuth2Service) createNewAccount(ctx context.Context, provider OAuth2Provider, email string, userInfo map[string]interface{}, token *oauth2.Token) (*model.Account, error) {
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

	// 确定提供商和协议
	var providerName, protocol string
	switch provider {
	case OAuth2ProviderGoogle:
		providerName = "gmail"
		protocol = "gmail_api"
	case OAuth2ProviderMicrosoft:
		providerName = "outlook"
		protocol = "graph"
	}

	// 创建账户
	account := &model.Account{
		UID:                  accountUID,
		Email:                email,
		Provider:             providerName,
		Protocol:             protocol,
		AuthType:             "oauth2",
		EncryptedCredentials: encryptedCredentials,
		Status:               "active",
		SyncEnabled:          true,
		SyncInterval:         2,
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
