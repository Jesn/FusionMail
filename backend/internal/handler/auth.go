package handler

import (
	"bytes"
	"fusionmail/internal/dto"
	cryptoutil "fusionmail/pkg/crypto"
	"io"
	"log"
	"net/http"
	"os"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	jwtSecret string
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtSecret string) *AuthHandler {
	return &AuthHandler{
		jwtSecret: jwtSecret,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用主密码登录系统
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	log.Printf("[AUTH DEBUG] Login request started")

	// Debug: Log raw request body
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	log.Printf("[AUTH DEBUG] Raw request body: %s", string(bodyBytes))
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[AUTH DEBUG] JSON binding failed: %v", err)
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	log.Printf("[AUTH DEBUG] Parsed request - Password: '%s'", req.Password)
	log.Printf("[AUTH DEBUG] Password length: %d", len(req.Password))
	log.Printf("[AUTH DEBUG] Password bytes: %v", []byte(req.Password))

	// 使用 bcrypt 哈希进行校验（通过环境变量 ADMIN_PASSWORD_HASH 提供）
	hash := os.Getenv("ADMIN_PASSWORD_HASH")
	log.Printf("[AUTH DEBUG] ADMIN_PASSWORD_HASH from env: '%s'", hash)
	log.Printf("[AUTH DEBUG] ADMIN_PASSWORD_HASH length: %d", len(hash))

	if hash == "" {
		log.Printf("[AUTH DEBUG] ADMIN_PASSWORD_HASH is empty")
		dto.InternalServerErrorResponse(c, "管理员密码哈希未配置")
		return
	}

	log.Printf("[AUTH DEBUG] About to verify password")
	if !cryptoutil.VerifyPassword(req.Password, hash) {
		log.Printf("[AUTH DEBUG] Password verification failed")
		log.Printf("[AUTH DEBUG] Expected hash: '%s'", hash)
		log.Printf("[AUTH DEBUG] Provided password: '%s'", req.Password)
		dto.UnauthorizedResponseWithCode(c, dto.ErrInvalidCredentials)
		return
	}

	log.Printf("[AUTH DEBUG] Password verification succeeded")

	// 生成 JWT token
	expiresAt := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "admin",
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成 token 失败")
		return
	}

	// 设置 HttpOnly 会话 Cookie，供 SSE 使用
	secure := c.Request.TLS != nil
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "fm_session",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})

	dto.SuccessResponse(c, LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// Logout 用户登出
// @Summary 用户登出
// @Description 登出系统（客户端清除 token）
// @Tags 认证
// @Success 200 {object} map[string]string
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
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
// @Summary 验证 token
// @Description 验证当前 token 是否有效
// @Tags 认证
// @Success 200 {object} map[string]bool
// @Failure 401 {object} map[string]string
// @Router /auth/verify [get]
func (h *AuthHandler) Verify(c *gin.Context) {
	// 从请求头获取 token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		dto.UnauthorizedResponse(c, "未提供认证信息")
		return
	}

	// 解析 Bearer token
	tokenString := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	}

	// 验证 token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	dto.SuccessResponse(c, gin.H{"valid": true})
}

// RefreshTokenRequest 刷新 token 请求
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// RefreshToken 刷新 token
// @Summary 刷新 token
// @Description 使用旧 token 获取新 token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "刷新请求"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	// 解析旧 token（不验证过期时间）
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		dto.UnauthorizedResponseWithCode(c, dto.ErrTokenInvalid)
		return
	}

	// 生成新 token
	expiresAt := time.Now().Add(24 * time.Hour)
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": claims["sub"],
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := newToken.SignedString([]byte(h.jwtSecret))
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成 token 失败")
		return
	}

	dto.SuccessResponse(c, LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}
