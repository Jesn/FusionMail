package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrAccountLocked      = errors.New("account is locked")
)

// AuthService 认证服务
type AuthService struct {
	userRepo   repository.UserRepository
	apiKeyRepo *repository.APIKeyRepository
	jwtSecret  string
	jwtExpiry  time.Duration
}

// NewAuthService 创建认证服务
func NewAuthService(
	userRepo repository.UserRepository,
	apiKeyRepo *repository.APIKeyRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		apiKeyRepo: apiKeyRepo,
		jwtSecret:  jwtSecret,
		jwtExpiry:  jwtExpiry,
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, username, password string) (string, time.Time, error) {
	// 查找用户
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}

	// 检查用户状态
	if !user.IsActive {
		return "", time.Time{}, ErrUserDisabled
	}

	// 检查账户是否被锁定
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return "", time.Time{}, ErrAccountLocked
	}

	// 验证密码
	if !crypto.VerifyPassword(password, user.PasswordHash) {
		// 增加失败次数
		_ = s.userRepo.IncrementFailedAttempts(ctx, user.ID)
		return "", time.Time{}, ErrInvalidCredentials
	}

	// 重置失败次数
	_ = s.userRepo.ResetFailedAttempts(ctx, user.ID)

	// 更新最后登录信息
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, "")

	// 生成 JWT token
	expiresAt := time.Now().Add(s.jwtExpiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// RefreshToken 刷新 token
func (s *AuthService) RefreshToken(ctx context.Context, oldToken string) (string, time.Time, error) {
	// 解析旧 token（不验证过期时间）
	token, err := jwt.Parse(oldToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		return "", time.Time{}, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", time.Time{}, errors.New("invalid token claims")
	}

	// 获取用户 ID
	userID, ok := claims["sub"].(float64)
	if !ok {
		return "", time.Time{}, errors.New("invalid user id in token")
	}

	// 验证用户是否存在且活跃
	user, err := s.userRepo.FindByID(ctx, int64(userID))
	if err != nil {
		return "", time.Time{}, ErrUserNotFound
	}

	if !user.IsActive {
		return "", time.Time{}, ErrUserDisabled
	}

	// 生成新 token
	expiresAt := time.Now().Add(s.jwtExpiry)
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := newToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// VerifyToken 验证 token
func (s *AuthService) VerifyToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return &claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// CreateAPIKey 创建 API Key
func (s *AuthService) CreateAPIKey(ctx context.Context, name, description string, permissions []string, rateLimit int, expiresAt *time.Time) (string, *model.APIKey, error) {
	// 生成随机 API Key（32 字节 = 256 位）
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, err
	}
	apiKey := base64.URLEncoding.EncodeToString(keyBytes)

	// 计算哈希值
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// 创建 API Key 记录
	key := &model.APIKey{
		KeyHash:     keyHash,
		Name:        name,
		Description: description,
		Permissions: "", // TODO: 将 permissions 序列化为 JSON
		RateLimit:   rateLimit,
		Enabled:     true,
		ExpiresAt:   expiresAt,
	}

	if err := s.apiKeyRepo.Create(ctx, key); err != nil {
		return "", nil, err
	}

	// 返回原始 API Key（只在创建时返回一次）
	return apiKey, key, nil
}

// ListAPIKeys 列出所有 API Key
func (s *AuthService) ListAPIKeys(ctx context.Context) ([]*model.APIKey, error) {
	return s.apiKeyRepo.FindAll(ctx)
}

// GetAPIKey 获取 API Key 详情
func (s *AuthService) GetAPIKey(ctx context.Context, id int64) (*model.APIKey, error) {
	return s.apiKeyRepo.FindByID(ctx, id)
}

// UpdateAPIKey 更新 API Key
func (s *AuthService) UpdateAPIKey(ctx context.Context, id int64, name, description string, rateLimit int) error {
	key, err := s.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	key.Name = name
	key.Description = description
	key.RateLimit = rateLimit

	return s.apiKeyRepo.Update(ctx, key)
}

// DeleteAPIKey 删除 API Key
func (s *AuthService) DeleteAPIKey(ctx context.Context, id int64) error {
	return s.apiKeyRepo.Delete(ctx, id)
}

// EnableAPIKey 启用 API Key
func (s *AuthService) EnableAPIKey(ctx context.Context, id int64) error {
	return s.apiKeyRepo.Enable(ctx, id)
}

// DisableAPIKey 禁用 API Key
func (s *AuthService) DisableAPIKey(ctx context.Context, id int64) error {
	return s.apiKeyRepo.Disable(ctx, id)
}
