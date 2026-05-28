package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
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

const (
	refreshThreshold     = 10 * time.Minute
	maxRefreshSessionTTL = 7 * 24 * time.Hour
	refreshSessionClaim  = "session_exp"
	SessionVersionClaim  = "session_version"
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
	if err != nil || user == nil {
		return "", time.Time{}, ErrInvalidCredentials
	}

	// 检查用户状态
	if !user.IsActive {
		return "", time.Time{}, ErrUserDisabled
	}

	// 检查账户是否被锁定
	now := time.Now()
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
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

	return s.issueToken(user, now, now.Add(maxRefreshSessionTTL))
}

// RefreshToken 刷新 token
func (s *AuthService) RefreshToken(ctx context.Context, oldToken string) (string, time.Time, error) {
	claims, err := s.parseSignedTokenClaims(oldToken)
	if err != nil {
		return "", time.Time{}, err
	}

	now := time.Now()
	if err := validateRefreshClaims(claims, now); err != nil {
		return "", time.Time{}, err
	}

	userID, err := parseUserIDClaim(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return "", time.Time{}, ErrUserNotFound
	}
	if !user.IsActive {
		return "", time.Time{}, ErrUserDisabled
	}
	if !sessionVersionMatches(claims, user.SessionVersion) {
		return "", time.Time{}, errors.New("stale token session")
	}

	sessionExpiresAt, ok := getRefreshSessionExpiresAt(claims)
	if !ok {
		return "", time.Time{}, errors.New("invalid refresh session")
	}

	return s.issueToken(user, now, sessionExpiresAt)
}

// VerifyToken 验证 token
func (s *AuthService) issueToken(user *model.User, now, sessionExpiresAt time.Time) (string, time.Time, error) {
	expiresAt := now.Add(s.jwtExpiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":               user.ID,
		"username":          user.Username,
		"email":             user.Email,
		"exp":               expiresAt.Unix(),
		"iat":               now.Unix(),
		refreshSessionClaim: sessionExpiresAt.Unix(),
		SessionVersionClaim: user.SessionVersion,
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (s *AuthService) parseSignedTokenClaims(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func parseUserIDClaim(claims jwt.MapClaims) (int64, error) {
	switch userID := claims["sub"].(type) {
	case float64:
		return int64(userID), nil
	case string:
		parsed, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return 0, errors.New("invalid user id in token")
		}
		return parsed, nil
	default:
		return 0, errors.New("invalid user id in token")
	}
}

func parseSessionVersionClaim(claims jwt.MapClaims) (int64, error) {
	switch sessionVersion := claims[SessionVersionClaim].(type) {
	case float64:
		return int64(sessionVersion), nil
	case int64:
		return sessionVersion, nil
	case int:
		return int64(sessionVersion), nil
	case string:
		parsed, err := strconv.ParseInt(sessionVersion, 10, 64)
		if err != nil {
			return 0, errors.New("invalid session version in token")
		}
		return parsed, nil
	default:
		return 0, errors.New("invalid session version in token")
	}
}

func sessionVersionMatches(claims jwt.MapClaims, currentVersion int64) bool {
	sessionVersion, err := parseSessionVersionClaim(claims)
	return err == nil && sessionVersion == currentVersion
}

func parseUnixClaim(claims jwt.MapClaims, key string) (time.Time, bool) {
	value, ok := claims[key]
	if !ok {
		return time.Time{}, false
	}

	switch typed := value.(type) {
	case float64:
		return time.Unix(int64(typed), 0), true
	case int64:
		return time.Unix(typed, 0), true
	case int:
		return time.Unix(int64(typed), 0), true
	}

	return time.Time{}, false
}

func getRefreshSessionExpiresAt(claims jwt.MapClaims) (time.Time, bool) {
	if sessionExpiresAt, ok := parseUnixClaim(claims, refreshSessionClaim); ok {
		return sessionExpiresAt, true
	}
	return parseUnixClaim(claims, "exp")
}

func validateRefreshClaims(claims jwt.MapClaims, now time.Time) error {
	expiresAt, ok := parseUnixClaim(claims, "exp")
	if !ok {
		return errors.New("invalid token expiry")
	}
	if !expiresAt.After(now) {
		return jwt.ErrTokenExpired
	}
	if expiresAt.Sub(now) > refreshThreshold {
		return errors.New("token not eligible for refresh yet")
	}

	sessionExpiresAt, ok := getRefreshSessionExpiresAt(claims)
	if !ok {
		return errors.New("invalid refresh session")
	}
	if !sessionExpiresAt.After(now) {
		return jwt.ErrTokenExpired
	}

	return nil
}

func (s *AuthService) VerifyToken(ctx context.Context, tokenString string) (*jwt.MapClaims, error) {
	claims, err := s.parseSignedTokenClaims(tokenString)
	if err != nil {
		return nil, err
	}

	userID, err := parseUserIDClaim(claims)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if !user.IsActive {
		return nil, ErrUserDisabled
	}
	if !sessionVersionMatches(claims, user.SessionVersion) {
		return nil, errors.New("stale token session")
	}

	return &claims, nil
}

// CreateAPIKey 创建 API Key（已简化：不支持 per-key 速率限制）
func (s *AuthService) CreateAPIKey(ctx context.Context, name, description string, permissions []string, expiresAt *time.Time) (string, *model.APIKey, error) {
	// 生成随机 API Key（32 字节 = 256 位）
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, err
	}
	apiKey := base64.URLEncoding.EncodeToString(keyBytes)

	// 计算哈希值
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// 序列化 permissions 为 JSON
	permissionsJSON := ""
	if len(permissions) > 0 {
		if data, err := json.Marshal(permissions); err == nil {
			permissionsJSON = string(data)
		}
	}

	// 创建 API Key 记录
	key := &model.APIKey{
		KeyHash:     keyHash,
		Name:        name,
		Description: description,
		Permissions: permissionsJSON,
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

// UpdateAPIKey 更新 API Key（已简化：不支持 per-key 速率限制）
func (s *AuthService) UpdateAPIKey(ctx context.Context, id int64, name, description string) error {
	key, err := s.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	key.Name = name
	key.Description = description

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
