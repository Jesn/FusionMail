package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/service"
	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// DBAuthHandler 数据库认证处理器
type DBAuthHandler struct {
	jwtSecret    string
	initService  *service.InitService
	cookieSecure *bool // Cookie Secure 配置（nil=自动检测，true/false=强制设置）
	logger       *logger.Logger
}

// NewDBAuthHandler 创建数据库认证处理器
func NewDBAuthHandler(jwtSecret string, cookieSecure *bool) *DBAuthHandler {
	return &DBAuthHandler{
		jwtSecret:    jwtSecret,
		initService:  service.NewInitService(),
		cookieSecure: cookieSecure,
		logger:       logger.NewWithModule("Auth"),
	}
}

const (
	accessTokenTTL        = 24 * time.Hour
	refreshThreshold      = 10 * time.Minute
	maxRefreshSessionTTL  = 7 * 24 * time.Hour
	refreshSessionClaim   = "session_exp"
	twoFactorChallengeTTL = 5 * time.Minute
)

var twoFactorChallengeStore = newTwoFactorChallengeStore()

type twoFactorLoginChallenge struct {
	userID         int64
	sessionVersion int64
	expiresAt      time.Time
}

type twoFactorChallengeStoreState struct {
	mu         sync.Mutex
	challenges map[string]twoFactorLoginChallenge
}

func newTwoFactorChallengeStore() *twoFactorChallengeStoreState {
	return &twoFactorChallengeStoreState{
		challenges: make(map[string]twoFactorLoginChallenge),
	}
}

func issueTwoFactorLoginChallenge(userID, sessionVersion int64, now time.Time) (string, time.Time, error) {
	return twoFactorChallengeStore.issue(userID, sessionVersion, now)
}

func consumeTwoFactorLoginChallenge(token string, userID, sessionVersion int64, now time.Time) error {
	return twoFactorChallengeStore.consume(token, userID, sessionVersion, now)
}

func (s *twoFactorChallengeStoreState) issue(userID, sessionVersion int64, now time.Time) (string, time.Time, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", time.Time{}, err
	}

	token := hex.EncodeToString(randomBytes)
	expiresAt := now.Add(twoFactorChallengeTTL)

	s.mu.Lock()
	defer s.mu.Unlock()

	for existingToken, challenge := range s.challenges {
		if !challenge.expiresAt.After(now) {
			delete(s.challenges, existingToken)
		}
	}

	s.challenges[token] = twoFactorLoginChallenge{
		userID:         userID,
		sessionVersion: sessionVersion,
		expiresAt:      expiresAt,
	}

	return token, expiresAt, nil
}

func (s *twoFactorChallengeStoreState) consume(token string, userID, sessionVersion int64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge, ok := s.challenges[token]
	if !ok {
		return fmt.Errorf("2fa challenge not found")
	}
	delete(s.challenges, token)

	if challenge.userID != userID {
		return fmt.Errorf("2fa challenge user mismatch")
	}
	if challenge.sessionVersion != sessionVersion {
		return fmt.Errorf("2fa challenge session version mismatch")
	}
	if !challenge.expiresAt.After(now) {
		return fmt.Errorf("2fa challenge expired")
	}

	return nil
}

func issueSessionToken(jwtSecret string, user *model.User, now, sessionExpiresAt time.Time) (string, time.Time, error) {
	expiresAt := now.Add(accessTokenTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                       strconv.FormatInt(user.ID, 10),
		"exp":                       expiresAt.Unix(),
		"iat":                       now.Unix(),
		"username":                  user.Username,
		"role":                      user.Role,
		refreshSessionClaim:         sessionExpiresAt.Unix(),
		service.SessionVersionClaim: user.SessionVersion,
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func parseSignedTokenClaims(tokenString, jwtSecret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenMalformed
	}

	return claims, nil
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

func parseSubjectClaim(claims jwt.MapClaims) (int64, error) {
	subject, ok := claims["sub"].(string)
	if !ok || subject == "" {
		return 0, jwt.ErrTokenMalformed
	}

	userID, err := strconv.ParseInt(subject, 10, 64)
	if err != nil {
		return 0, jwt.ErrTokenMalformed
	}

	return userID, nil
}

func parseInt64Claim(claims jwt.MapClaims, key string) (int64, error) {
	switch value := claims[key].(type) {
	case float64:
		return int64(value), nil
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, jwt.ErrTokenMalformed
		}
		return parsed, nil
	default:
		return 0, jwt.ErrTokenMalformed
	}
}

func sessionVersionClaimMatches(claims jwt.MapClaims, sessionVersion int64) bool {
	claimVersion, err := parseInt64Claim(claims, service.SessionVersionClaim)
	return err == nil && claimVersion == sessionVersion
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
		return jwt.ErrTokenMalformed
	}
	if !expiresAt.After(now) {
		return jwt.ErrTokenExpired
	}
	if expiresAt.Sub(now) > refreshThreshold {
		return jwt.ErrTokenNotValidYet
	}

	sessionExpiresAt, ok := getRefreshSessionExpiresAt(claims)
	if !ok {
		return jwt.ErrTokenMalformed
	}
	if !sessionExpiresAt.After(now) {
		return jwt.ErrTokenExpired
	}

	return nil
}

// DBLoginRequest 数据库登录请求
type DBLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// DBLoginResponse 数据库登录响应
type DBLoginResponse struct {
	Token                    string      `json:"token,omitempty"`
	ExpiresAt                string      `json:"expiresAt"`
	User                     *DBUserInfo `json:"user"`
	Requires2FA              bool        `json:"requires_2fa,omitempty"`                // 是否需要 2FA 验证
	TwoFactorUserID          int64       `json:"two_factor_user_id,omitempty"`          // 2FA 验证用的用户 ID
	TwoFactorChallengeToken  string      `json:"two_factor_challenge_token,omitempty"`  // 登录第一步成功后签发的一次性 2FA 挑战令牌
	TwoFactorChallengeExpiry string      `json:"two_factor_challenge_expiry,omitempty"` // 一次性 2FA 挑战令牌过期时间
}

// DBUserInfo 数据库用户信息
type DBUserInfo struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Theme       string `json:"theme"`
}

// Login 用户登录（新系统）
// @Summary 用户登录
// @Description 使用用户名和密码登录系统
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body DBLoginRequest true "登录请求"
// @Success 200 {object} DBLoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *DBAuthHandler) Login(c *gin.Context) {
	var req DBLoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("JSON 绑定失败: %v", err)
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	// 验证用户凭据
	user, err := h.initService.ValidateUserCredentials(req.Username, req.Password)
	if err != nil {
		h.logger.Debug("登录失败: user=%s, error=%v", req.Username, err)
		dto.UnauthorizedResponseWithCode(c, dto.ErrInvalidCredentials)
		return
	}

	// 检查是否启用了 2FA
	if user.TwoFactorEnabled && user.TwoFactorVerified {
		h.logger.Debug("用户需要 2FA 验证: %s", user.Username)

		now := time.Now()
		challengeToken, challengeExpiresAt, err := issueTwoFactorLoginChallenge(user.ID, user.SessionVersion, now)
		if err != nil {
			dto.InternalServerErrorResponse(c, "生成 2FA 挑战令牌失败")
			return
		}

		dto.SuccessResponse(c, DBLoginResponse{
			Requires2FA:              true,
			TwoFactorUserID:          user.ID,
			TwoFactorChallengeToken:  challengeToken,
			TwoFactorChallengeExpiry: challengeExpiresAt.Format(time.RFC3339),
		})
		return
	}

	h.logger.Info("用户登录成功: %s", user.Username)

	// 更新最后登录信息
	user.LastLoginAt = &time.Time{}
	*user.LastLoginAt = time.Now()
	if ip := c.ClientIP(); ip != "" {
		user.LastLoginIP = ip
	}
	h.initService.UpdateLastLogin(user.ID, user.LastLoginAt, user.LastLoginIP)

	// 生成 JWT token
	now := time.Now()
	tokenString, expiresAt, err := issueSessionToken(h.jwtSecret, user, now, now.Add(maxRefreshSessionTTL))
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成 token 失败")
		return
	}

	h.setSessionCookie(c, tokenString, expiresAt)

	dto.SuccessResponse(c, DBLoginResponse{
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User: &DBUserInfo{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			Theme:       user.Theme,
		},
	})
}

func (h *DBAuthHandler) setSessionCookie(c *gin.Context, tokenString string, expiresAt time.Time) {
	// 浏览器用户会话只通过 HttpOnly Cookie 暴露，避免 JWT 进入前端 JS 存储。
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	if h.cookieSecure != nil {
		secure = *h.cookieSecure
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "fm_session",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户的密码
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/change-password [post]
func (h *DBAuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	// 从token中获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未找到用户信息")
		return
	}

	id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64)
	if err != nil {
		dto.InternalServerErrorResponse(c, "用户ID格式错误")
		return
	}

	// 修改密码
	if err := h.initService.ChangePassword(id, req.OldPassword, req.NewPassword); err != nil {
		h.logger.Debug("修改密码失败: %v", err)
		dto.BadRequestResponse(c, err.Error())
		return
	}

	dto.SuccessWithMessage(c, nil, "密码修改成功")
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的信息
// @Tags 认证
// @Produce json
// @Success 200 {object} DBUserInfo
// @Failure 401 {object} map[string]string
// @Router /auth/me [get]
func (h *DBAuthHandler) GetCurrentUser(c *gin.Context) {
	// 从token中获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未找到用户信息")
		return
	}

	id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64)
	if err != nil {
		dto.InternalServerErrorResponse(c, "用户ID格式错误")
		return
	}

	// 获取用户信息
	user, err := h.initService.GetUserByID(id)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取用户信息失败")
		return
	}

	dto.SuccessResponse(c, DBUserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Theme:       user.Theme,
	})
}

func extractSessionToken(c *gin.Context) string {
	if cookie, err := c.Cookie("fm_session"); err == nil && cookie != "" {
		return cookie
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return authHeader
}

func (h *DBAuthHandler) revokeSessionFromToken(tokenString string) {
	claims, err := parseSignedTokenClaims(tokenString, h.jwtSecret)
	if err != nil {
		return
	}

	userID, err := parseSubjectClaim(claims)
	if err != nil {
		return
	}
	user, err := h.initService.GetUserByID(userID)
	if err != nil || !sessionVersionClaimMatches(claims, user.SessionVersion) {
		return
	}
	if err := h.initService.IncrementSessionVersion(userID); err != nil {
		h.logger.Debug("撤销会话失败: %v", err)
	}
}

// Logout 用户登出
func (h *DBAuthHandler) Logout(c *gin.Context) {
	if tokenString := extractSessionToken(c); tokenString != "" {
		h.revokeSessionFromToken(tokenString)
	}

	// 清除 Cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "fm_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	dto.SuccessWithMessage(c, nil, "登出成功")
}

// Verify 验证 token
func (h *DBAuthHandler) Verify(c *gin.Context) {
	tokenString := extractSessionToken(c)
	if tokenString == "" {
		dto.UnauthorizedResponse(c, "未提供认证信息")
		return
	}

	claims, err := parseSignedTokenClaims(tokenString, h.jwtSecret)
	if err != nil {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}
	userID, err := parseSubjectClaim(claims)
	if err != nil {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}
	user, err := h.initService.GetUserByID(userID)
	if err != nil || !user.IsActive || !sessionVersionClaimMatches(claims, user.SessionVersion) {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	dto.SuccessResponse(c, gin.H{"valid": true})
}

// RefreshTokenRequest 刷新 token 请求
type RefreshTokenRequest struct {
	Token string `json:"token"`
}

// RefreshToken 刷新 token
func (h *DBAuthHandler) RefreshToken(c *gin.Context) {
	tokenString := extractSessionToken(c)
	if tokenString == "" && c.Request.Body != nil && c.Request.ContentLength != 0 {
		var req RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequestResponse(c, "请求参数格式错误")
			return
		}
		tokenString = req.Token
	}
	if tokenString == "" {
		dto.UnauthorizedResponse(c, "未提供认证信息")
		return
	}

	claims, err := parseSignedTokenClaims(tokenString, h.jwtSecret)
	if err != nil {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	now := time.Now()
	if err := validateRefreshClaims(claims, now); err != nil {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	userID, err := parseSubjectClaim(claims)
	if err != nil {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	user, err := h.initService.GetUserByID(userID)
	if err != nil || !user.IsActive || !sessionVersionClaimMatches(claims, user.SessionVersion) {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	sessionExpiresAt, ok := getRefreshSessionExpiresAt(claims)
	if !ok {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	tokenString, expiresAt, err := issueSessionToken(h.jwtSecret, user, now, sessionExpiresAt)
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成 token 失败")
		return
	}

	h.setSessionCookie(c, tokenString, expiresAt)

	dto.SuccessResponse(c, DBLoginResponse{
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// 确保 DBAuthHandler 实现了 AuthHandlerInterface 接口
var _ AuthHandlerInterface = (*DBAuthHandler)(nil)
