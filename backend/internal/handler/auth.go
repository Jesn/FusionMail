package handler

import (
	"fusionmail/internal/dto"
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
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	// TODO: 实现真正的密码验证
	// 目前为了 MVP，我们使用一个简单的固定密码
	// 在生产环境中，应该从配置或数据库中读取加密后的密码
	masterPassword := "admin123" // 临时密码，应该从环境变量读取

	if req.Password != masterPassword {
		dto.UnauthorizedResponseWithCode(c, dto.ErrInvalidCredentials)
		return
	}

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
