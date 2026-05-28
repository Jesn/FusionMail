package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/internal/service"
	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
)

// 模块日志记录器
var twoFactorLog = logger.NewWithModule("2FA")

// TwoFactorHandler 2FA 处理器
type TwoFactorHandler struct {
	initService *service.InitService
	totpService *service.TOTPService
}

// NewTwoFactorHandler 创建 2FA 处理器
func NewTwoFactorHandler(initService *service.InitService) *TwoFactorHandler {
	return &TwoFactorHandler{
		initService: initService,
		totpService: service.NewTOTPService("FusionMail"),
	}
}

// Setup2FAResponse 设置 2FA 响应
type Setup2FAResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// Validate2FARequest 登录时验证 2FA 请求
type Validate2FARequest struct {
	UserID int64  `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

// Setup2FA 设置 2FA（生成密钥和二维码）
// @Summary 设置双因素认证
// @Description 生成 TOTP 密钥和二维码 URL
// @Tags 认证
// @Produce json
// @Success 200 {object} Setup2FAResponse
// @Failure 401 {object} map[string]string
// @Router /auth/2fa/setup [post]
func (h *TwoFactorHandler) Setup2FA(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		dto.UnauthorizedResponse(c, "未找到用户信息")
		return
	}

	// 获取用户信息
	user, err := h.initService.GetUserByID(userID)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取用户信息失败")
		return
	}

	// 如果已启用 2FA，需要先禁用
	if user.TwoFactorEnabled && user.TwoFactorVerified {
		dto.BadRequestResponse(c, "2FA 已启用，请先禁用后再重新设置")
		return
	}

	// 生成 TOTP 密钥
	secret, err := h.totpService.GenerateSecret()
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成密钥失败")
		return
	}

	// 生成恢复码
	backupCodes, err := h.totpService.GenerateBackupCodes(10)
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成恢复码失败")
		return
	}

	// 生成二维码 URL
	qrCodeURL := h.totpService.GenerateOTPAuthURL(secret, user.Username)

	encryptedSecret, err := h.totpService.EncryptSecret(secret)
	if err != nil {
		dto.InternalServerErrorResponse(c, "加密 2FA 密钥失败")
		return
	}
	hashedBackupCodes, err := h.totpService.HashBackupCodes(backupCodes)
	if err != nil {
		dto.InternalServerErrorResponse(c, "处理恢复码失败")
		return
	}
	backupCodesJSON, err := json.Marshal(hashedBackupCodes)
	if err != nil {
		dto.InternalServerErrorResponse(c, "处理恢复码失败")
		return
	}
	if err := h.initService.Update2FASetup(userID, encryptedSecret, string(backupCodesJSON)); err != nil {
		dto.InternalServerErrorResponse(c, "保存 2FA 设置失败")
		return
	}

	twoFactorLog.Info("用户 %s 开始设置 2FA", user.Username)

	dto.SuccessResponse(c, Setup2FAResponse{
		Secret:      secret,
		QRCodeURL:   qrCodeURL,
		BackupCodes: backupCodes,
	})
}

// Verify2FARequest 验证 2FA 请求
type Verify2FARequest struct {
	Code string `json:"code" binding:"required"`
}

// Verify2FA 验证并启用 2FA
// @Summary 验证并启用双因素认证
// @Description 验证 TOTP 码并启用 2FA
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body Verify2FARequest true "验证请求"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /auth/2fa/verify [post]
func (h *TwoFactorHandler) Verify2FA(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	userID, err := h.getUserID(c)
	if err != nil {
		dto.UnauthorizedResponse(c, "未找到用户信息")
		return
	}

	// 获取用户信息
	user, err := h.initService.GetUserByID(userID)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取用户信息失败")
		return
	}

	// 检查是否有待验证的密钥
	if user.TwoFactorSecret == "" {
		dto.BadRequestResponse(c, "请先调用设置接口生成密钥")
		return
	}

	secret, err := h.totpService.DecryptSecret(user.TwoFactorSecret)
	if err != nil {
		dto.InternalServerErrorResponse(c, "读取 2FA 密钥失败")
		return
	}

	// 验证 TOTP 码
	if !h.totpService.ValidateCode(secret, req.Code) {
		dto.BadRequestResponse(c, "验证码错误，请重试")
		return
	}

	// 启用 2FA
	now := time.Now()
	if err := h.initService.Enable2FA(userID, &now); err != nil {
		dto.InternalServerErrorResponse(c, "启用 2FA 失败")
		return
	}

	twoFactorLog.Info("用户 %s 成功启用 2FA", user.Username)

	dto.SuccessWithMessage(c, nil, "双因素认证已启用")
}

// Disable2FARequest 禁用 2FA 请求
type Disable2FARequest struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"` // TOTP 码或恢复码（必填）
}

// Disable2FA 禁用 2FA
// @Summary 禁用双因素认证
// @Description 验证密码后禁用 2FA
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body Disable2FARequest true "禁用请求"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /auth/2fa/disable [post]
func (h *TwoFactorHandler) Disable2FA(c *gin.Context) {
	var req Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	userID, err := h.getUserID(c)
	if err != nil {
		dto.UnauthorizedResponse(c, "未找到用户信息")
		return
	}

	// 获取用户信息
	user, err := h.initService.GetUserByID(userID)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取用户信息失败")
		return
	}

	// 验证密码
	if _, err := h.initService.ValidateUserCredentials(user.Username, req.Password); err != nil {
		dto.BadRequestResponse(c, "密码错误")
		return
	}

	// 如果已启用 2FA，必须验证 2FA 验证码
	if user.TwoFactorEnabled && user.TwoFactorVerified {
		secret, err := h.totpService.DecryptSecret(user.TwoFactorSecret)
		if err != nil {
			dto.InternalServerErrorResponse(c, "读取 2FA 密钥失败")
			return
		}

		valid := h.totpService.ValidateCode(secret, req.Code)

		// 如果 TOTP 验证失败，尝试恢复码
		if !valid {
			var remainingCount int
			valid, remainingCount, err = h.initService.ConsumeBackupCode(user.ID, req.Code)
			if err != nil {
				dto.InternalServerErrorResponse(c, "恢复码处理失败")
				return
			}
			if valid {
				twoFactorLog.Info("用户 %s 使用恢复码禁用 2FA，剩余 %d 个", user.Username, remainingCount)
			}
		}

		// 如果验证码和恢复码都无效，返回错误
		if !valid {
			dto.BadRequestResponse(c, "验证码或恢复码错误")
			return
		}
	}

	// 禁用 2FA
	if err := h.initService.Disable2FA(userID); err != nil {
		dto.InternalServerErrorResponse(c, "禁用 2FA 失败")
		return
	}

	twoFactorLog.Info("用户 %s 禁用了 2FA", user.Username)

	dto.SuccessWithMessage(c, nil, "双因素认证已禁用")
}

// Get2FAStatus 获取 2FA 状态
// @Summary 获取双因素认证状态
// @Description 获取当前用户的 2FA 启用状态
// @Tags 认证
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /auth/2fa/status [get]
func (h *TwoFactorHandler) Get2FAStatus(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		dto.UnauthorizedResponse(c, "未找到用户信息")
		return
	}

	user, err := h.initService.GetUserByID(userID)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取用户信息失败")
		return
	}

	// 计算剩余恢复码数量
	backupCodesCount := 0
	if user.TwoFactorBackup != "" {
		var codes []string
		if err := json.Unmarshal([]byte(user.TwoFactorBackup), &codes); err == nil {
			backupCodesCount = len(codes)
		}
	}

	dto.SuccessResponse(c, gin.H{
		"enabled":            user.TwoFactorEnabled && user.TwoFactorVerified,
		"enabled_at":         user.TwoFactorEnabledAt,
		"backup_codes_count": backupCodesCount,
	})
}

// RegenerateBackupCodes 重新生成恢复码
// @Summary 重新生成恢复码
// @Description 重新生成 2FA 恢复码
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body Verify2FARequest true "验证请求"
// @Success 200 {object} map[string]interface{}
// @Router /auth/2fa/backup-codes [post]
func (h *TwoFactorHandler) RegenerateBackupCodes(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	userID, err := h.getUserID(c)
	if err != nil {
		dto.UnauthorizedResponse(c, "未找到用户信息")
		return
	}

	user, err := h.initService.GetUserByID(userID)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取用户信息失败")
		return
	}

	if !user.TwoFactorEnabled || !user.TwoFactorVerified {
		dto.BadRequestResponse(c, "2FA 未启用")
		return
	}

	secret, err := h.totpService.DecryptSecret(user.TwoFactorSecret)
	if err != nil {
		dto.InternalServerErrorResponse(c, "读取 2FA 密钥失败")
		return
	}

	// 验证 TOTP 码
	if !h.totpService.ValidateCode(secret, req.Code) {
		dto.BadRequestResponse(c, "验证码错误")
		return
	}

	// 生成新的恢复码
	backupCodes, err := h.totpService.GenerateBackupCodes(10)
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成恢复码失败")
		return
	}

	// 保存新的恢复码
	hashedBackupCodes, err := h.totpService.HashBackupCodes(backupCodes)
	if err != nil {
		dto.InternalServerErrorResponse(c, "处理恢复码失败")
		return
	}
	backupCodesJSON, err := json.Marshal(hashedBackupCodes)
	if err != nil {
		dto.InternalServerErrorResponse(c, "处理恢复码失败")
		return
	}
	if err := h.initService.UpdateBackupCodes(userID, string(backupCodesJSON)); err != nil {
		dto.InternalServerErrorResponse(c, "保存恢复码失败")
		return
	}

	twoFactorLog.Info("用户 %s 重新生成了恢复码", user.Username)

	dto.SuccessResponse(c, gin.H{
		"backup_codes": backupCodes,
	})
}

// TwoFactorLoginHandler 2FA 登录处理器（需要 JWT 密钥）
type TwoFactorLoginHandler struct {
	*TwoFactorHandler
	jwtSecret    string
	cookieSecure *bool
}

// NewTwoFactorLoginHandler 创建带 JWT 功能的 2FA 处理器
func NewTwoFactorLoginHandler(initService *service.InitService, jwtSecret string, cookieSecure *bool) *TwoFactorLoginHandler {
	return &TwoFactorLoginHandler{
		TwoFactorHandler: NewTwoFactorHandler(initService),
		jwtSecret:        jwtSecret,
		cookieSecure:     cookieSecure,
	}
}

// Validate2FALoginRequest 登录时验证 2FA 请求（包含 JWT 生成所需信息）
type Validate2FALoginRequest struct {
	UserID                  int64  `json:"user_id" binding:"required"`
	Code                    string `json:"code" binding:"required"`
	TwoFactorChallengeToken string `json:"two_factor_challenge_token" binding:"required"`
	Username                string `json:"username"` // 可选，用于日志
}

// Validate2FALoginResponse 2FA 验证成功后的登录响应
type Validate2FALoginResponse struct {
	Token     string      `json:"token,omitempty"`
	ExpiresAt string      `json:"expiresAt"`
	User      *DBUserInfo `json:"user"`
}

// Validate2FA 登录时验证 2FA（公开接口）- 由 TwoFactorLoginHandler 实现
// 基础 TwoFactorHandler 不实现此方法，需要使用 TwoFactorLoginHandler
func (h *TwoFactorHandler) Validate2FA(c *gin.Context) {
	dto.BadRequestResponse(c, "请使用正确的 2FA 验证端点")
}

// Validate2FAAndLogin 登录时验证 2FA 并生成 JWT（公开接口）
// @Summary 登录时验证双因素认证
// @Description 在登录流程中验证 TOTP 码并完成登录，返回 JWT token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body Validate2FALoginRequest true "验证请求"
// @Success 200 {object} Validate2FALoginResponse
// @Failure 400 {object} map[string]string
// @Router /auth/2fa/validate [post]
func (h *TwoFactorLoginHandler) Validate2FAAndLogin(c *gin.Context) {
	var req Validate2FALoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	now := time.Now()
	user, err := h.initService.GetUserByID(req.UserID)
	if err != nil {
		dto.UnauthorizedResponse(c, "2FA 登录凭证无效或已过期")
		return
	}
	if err := consumeTwoFactorLoginChallenge(req.TwoFactorChallengeToken, req.UserID, user.SessionVersion, now); err != nil {
		dto.UnauthorizedResponse(c, "2FA 登录凭证无效或已过期")
		return
	}

	if !user.IsActive {
		dto.UnauthorizedResponse(c, "用户已被禁用")
		return
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		dto.UnauthorizedResponse(c, "账户已被锁定")
		return
	}

	if !user.TwoFactorEnabled || !user.TwoFactorVerified {
		dto.BadRequestResponse(c, "用户未启用 2FA")
		return
	}

	secret, err := h.totpService.DecryptSecret(user.TwoFactorSecret)
	if err != nil {
		dto.InternalServerErrorResponse(c, "读取 2FA 密钥失败")
		return
	}

	// 验证 TOTP 码
	valid := h.totpService.ValidateCode(secret, req.Code)

	// 如果 TOTP 验证失败，尝试恢复码
	if !valid {
		var remainingCount int
		valid, remainingCount, err = h.initService.ConsumeBackupCode(user.ID, req.Code)
		if err != nil {
			dto.InternalServerErrorResponse(c, "恢复码处理失败")
			return
		}
		if valid {
			twoFactorLog.Info("用户 %s 使用了恢复码，剩余 %d 个", user.Username, remainingCount)
		}
	}

	if !valid {
		dto.BadRequestResponse(c, "验证码错误")
		return
	}

	twoFactorLog.Info("用户 %s 通过 2FA 验证，正在生成 JWT", user.Username)

	// 更新最后登录信息
	user.LastLoginAt = &now
	if ip := c.ClientIP(); ip != "" {
		user.LastLoginIP = ip
	}
	h.initService.UpdateLastLogin(user.ID, user.LastLoginAt, user.LastLoginIP)

	// 生成会话 token，仅通过 HttpOnly Cookie 返回给浏览器。
	tokenString, expiresAt, err := issueSessionToken(h.jwtSecret, user, now, now.Add(maxRefreshSessionTTL))
	if err != nil {
		dto.InternalServerErrorResponse(c, "生成 token 失败")
		return
	}

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

	twoFactorLog.Info("用户 %s 通过 2FA 登录成功", user.Username)

	dto.SuccessResponse(c, Validate2FALoginResponse{
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

// getUserID 从上下文获取用户 ID
func (h *TwoFactorHandler) getUserID(c *gin.Context) (int64, error) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, fmt.Errorf("user ID not found")
	}
	return strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64)
}
