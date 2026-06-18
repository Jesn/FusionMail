package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/config"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/oauth2config"
	"fusionmail/pkg/redis"

	"golang.org/x/oauth2"
)

// OAuth2Service OAuth2 认证服务
type OAuth2Service struct {
	config               *config.Config
	accountRepo          repository.AccountRepository
	emailRepo            repository.EmailRepository
	providerRepo         repository.ProviderRepository // 提供商仓库
	adapterRepo          repository.AdapterRepository  // 适配器仓库
	cryptoService        *crypto.Service
	redisClient          *redis.ClientWrapper
	logger               *logger.Logger
	oauth2ConfigProvider *oauth2config.Provider // OAuth2配置提供者
}

// NewOAuth2Service 创建 OAuth2 服务实例
func NewOAuth2Service(
	cfg *config.Config,
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	cryptoService *crypto.Service,
	redisClient *redis.ClientWrapper,
	logger *logger.Logger,
	oauth2ClientRepo repository.OAuth2ClientRepository,
	providerRepo repository.ProviderRepository,
	adapterRepo repository.AdapterRepository,
) *OAuth2Service {
	// 创建OAuth2配置提供者
	oauth2Provider := oauth2config.NewProvider(oauth2ClientRepo, providerRepo, cryptoService, logger)

	return &OAuth2Service{
		config:               cfg,
		accountRepo:          accountRepo,
		emailRepo:            emailRepo,
		providerRepo:         providerRepo,
		adapterRepo:          adapterRepo,
		cryptoService:        cryptoService,
		redisClient:          redisClient,
		logger:               logger,
		oauth2ConfigProvider: oauth2Provider,
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
	Provider   OAuth2Provider `json:"provider"`
	Email      string         `json:"email,omitempty"`       // 可选，用于预填充
	AccountUID string         `json:"account_uid,omitempty"` // 可选，用于重新授权已存在的账户
	GroupID    *int64         `json:"group_id,omitempty"`    // 可选，账户分组 ID
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

	// 将 state 存储到 Redis（15分钟过期）
	stateKey := fmt.Sprintf("oauth2:state:%s", state)
	stateData := map[string]interface{}{
		"provider":    string(req.Provider),
		"email":       req.Email,
		"account_uid": req.AccountUID, // 用于重新授权已存在的账户
		"created":     time.Now().Unix(),
	}
	// 保存分组 ID（如果有）
	if req.GroupID != nil {
		stateData["group_id"] = *req.GroupID
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

	// 检查是否是重新授权（stateData 中包含 account_uid）
	accountUID, hasAccountUID := stateData["account_uid"].(string)
	if hasAccountUID && accountUID != "" {
		s.logger.Info("Reauthorization detected, updating existing account",
			"provider", req.Provider,
			"account_uid", accountUID)

		// 重新授权：直接更新现有账户的 token
		account, err := s.reauthorizeAccount(ctx, accountUID, token)
		if err != nil {
			s.logger.Error("Failed to reauthorize account",
				"provider", req.Provider,
				"account_uid", accountUID,
				"error", err)
			return nil, fmt.Errorf("failed to reauthorize account: %w", err)
		}

		s.logger.Info("Account reauthorized successfully",
			"provider", req.Provider,
			"account_uid", account.UID,
			"email", account.Email)

		return &OAuth2CallbackResponse{
			AccountUID:   account.UID,
			Email:        account.Email,
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresAt:    token.Expiry,
		}, nil
	}

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

	// 从 stateData 中提取 groupID
	var groupID *int64
	if gid, ok := stateData["group_id"]; ok {
		if gidFloat, ok := gid.(float64); ok {
			gidInt := int64(gidFloat)
			groupID = &gidInt
		}
	}

	// 创建或更新账户
	s.logger.Info("Creating or updating account", "provider", req.Provider, "group_id", groupID)

	account, err := s.createOrUpdateAccount(ctx, req.Provider, userInfo, token, groupID)
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
