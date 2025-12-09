package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"fusionmail/internal/dto"
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

// DBLoginRequest 数据库登录请求
type DBLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// DBLoginResponse 数据库登录响应
type DBLoginResponse struct {
	Token           string      `json:"token"`
	ExpiresAt       string      `json:"expiresAt"`
	User            *DBUserInfo `json:"user"`
	Requires2FA     bool        `json:"requires_2fa,omitempty"`       // 是否需要 2FA 验证
	TwoFactorUserID int64       `json:"two_factor_user_id,omitempty"` // 2FA 验证用的用户 ID
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
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *DBAuthHandler) Login(c *gin.Context) {
	var req DBLoginRequest

	// Debug: Log raw request body (仅在 DEBUG 级别)
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	h.logger.Debug("登录请求: body=%s", string(bodyBytes))
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body

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
		// 返回需要 2FA 验证的响应
		dto.SuccessResponse(c, DBLoginResponse{
			Requires2FA:     true,
			TwoFactorUserID: user.ID,
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
	expiresAt := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      strconv.FormatInt(user.ID, 10),
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
		"username": user.Username,
		"role":     user.Role,
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成 token 失败")
		return
	}

	// 设置 HttpOnly 会话 Cookie，供 SSE 使用
	// 判断 Cookie Secure 属性：
	// 1. 如果配置了 cookieSecure，使用配置值
	// 2. 否则自动检测：TLS 连接或反向代理 HTTPS
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	if h.cookieSecure != nil {
		secure = *h.cookieSecure // 使用配置的强制值
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

	dto.SuccessResponse(c, DBLoginResponse{
		Token:     tokenString,
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

// Logout 用户登出
func (h *DBAuthHandler) Logout(c *gin.Context) {
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
func (h *DBAuthHandler) RefreshToken(c *gin.Context) {
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
		"sub":      claims["sub"],
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
		"username": claims["username"],
		"role":     claims["role"],
	})

	tokenString, err := newToken.SignedString([]byte(h.jwtSecret))
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成 token 失败")
		return
	}

	dto.SuccessResponse(c, DBLoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// 确保 DBAuthHandler 实现了 AuthHandlerInterface 接口
var _ AuthHandlerInterface = (*DBAuthHandler)(nil)
