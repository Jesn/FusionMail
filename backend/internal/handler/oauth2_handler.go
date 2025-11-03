package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"fusionmail/internal/dto/response"
	"fusionmail/internal/service"
	"fusionmail/pkg/logger"
)

// OAuth2Handler OAuth2 认证处理器
type OAuth2Handler struct {
	oauth2Service *service.OAuth2Service
	logger        *logger.Logger
}

// NewOAuth2Handler 创建 OAuth2 处理器实例
func NewOAuth2Handler(oauth2Service *service.OAuth2Service, logger *logger.Logger) *OAuth2Handler {
	return &OAuth2Handler{
		oauth2Service: oauth2Service,
		logger:        logger,
	}
}

// GoogleAuthorize 生成 Google OAuth2 授权 URL
// @Summary 生成 Google OAuth2 授权 URL
// @Description 生成 Google OAuth2 授权 URL，用户需要访问此 URL 进行授权
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param email query string false "邮箱地址（可选，用于预填充）"
// @Success 200 {object} response.Response{data=service.OAuth2AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/google/authorize [get]
func (h *OAuth2Handler) GoogleAuthorize(c *gin.Context) {
	email := c.Query("email")

	req := &service.OAuth2AuthRequest{
		Provider: service.OAuth2ProviderGoogle,
		Email:    email,
	}

	resp, err := h.oauth2Service.GenerateAuthURL(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to generate Google OAuth2 auth URL", "error", err)
		response.Error(c, http.StatusInternalServerError, "生成授权链接失败")
		return
	}

	response.Success(c, resp)
}

// GoogleCallback 处理 Google OAuth2 授权回调
// @Summary 处理 Google OAuth2 授权回调
// @Description 处理 Google OAuth2 授权回调，完成账户创建或更新
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param code query string true "授权码"
// @Param state query string true "状态参数"
// @Success 200 {object} response.Response{data=service.OAuth2CallbackResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/google/callback [get]
func (h *OAuth2Handler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	error := c.Query("error")

	h.logger.Info("Google OAuth2 callback received", "code_length", len(code), "state", state, "error", error)

	// 检查是否有错误
	if error != "" {
		h.logger.Error("OAuth2 authorization error", "error", error)
		// 返回错误页面
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>授权失败</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .error { color: #dc3545; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="error">❌ 授权失败</div>
    <div class="info">错误：%s</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
            success: false,
            error: '%s'
        }));
        setTimeout(function() { window.close(); }, 2000);
    </script>
</body>
</html>`, error, error)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	if code == "" || state == "" {
		h.logger.Error("Missing required parameters", "code_empty", code == "", "state_empty", state == "")
		// 返回错误页面
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>授权失败</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .error { color: #dc3545; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="error">❌ 授权失败</div>
    <div class="info">缺少必要参数</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
            success: false,
            error: '缺少必要参数'
        }));
        setTimeout(function() { window.close(); }, 2000);
    </script>
</body>
</html>`
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	req := &service.OAuth2CallbackRequest{
		Provider: service.OAuth2ProviderGoogle,
		Code:     code,
		State:    state,
	}

	h.logger.Info("Processing OAuth2 callback", "provider", req.Provider)

	resp, err := h.oauth2Service.HandleCallback(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to handle Google OAuth2 callback", "error", err, "state", state, "code_length", len(code))
		// 返回错误页面
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>授权失败</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .error { color: #dc3545; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="error">❌ 授权处理失败</div>
    <div class="info">请稍后重试</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
            success: false,
            error: '授权处理失败'
        }));
        setTimeout(function() { window.close(); }, 2000);
    </script>
</body>
</html>`
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	h.logger.Info("OAuth2 callback processed successfully", "account_uid", resp.AccountUID, "email", resp.Email)

	// 返回一个简单的 HTML 页面，用于关闭弹窗并传递结果
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>授权成功</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .success { color: #28a745; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="success">✅ 授权成功！</div>
    <div class="info">账户 %s 已成功添加</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        console.log('OAuth2 授权成功页面加载');
        
        // 将结果存储到 sessionStorage
        const authResult = {
            success: true,
            account_uid: '%s',
            email: '%s'
        };
        
        console.log('存储授权结果到 sessionStorage:', authResult);
        
        try {
            // 方法1：使用 sessionStorage
            sessionStorage.setItem('oauth2_auth_result', JSON.stringify(authResult));
            
            // 验证存储是否成功
            const stored = sessionStorage.getItem('oauth2_auth_result');
            console.log('验证存储结果:', stored);
            
            // 方法2：使用 postMessage 向父窗口发送消息
            if (window.opener) {
                console.log('向父窗口发送消息:', authResult);
                window.opener.postMessage({
                    type: 'oauth2_result',
                    data: authResult
                }, '*');
            }
            
        } catch (error) {
            console.error('存储授权结果时出错:', error);
        }
        
        // 关闭窗口
        setTimeout(function() {
            console.log('准备关闭窗口');
            window.close();
        }, 2000);
    </script>
</body>
</html>`, resp.Email, resp.AccountUID, resp.Email)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// GoogleRefresh 刷新 Google OAuth2 访问令牌
// @Summary 刷新 Google OAuth2 访问令牌
// @Description 刷新指定账户的 Google OAuth2 访问令牌
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param account_uid path string true "账户 UID"
// @Success 200 {object} response.Response{data=service.OAuth2TokenRefreshResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/google/refresh/{account_uid} [post]
func (h *OAuth2Handler) GoogleRefresh(c *gin.Context) {
	accountUID := c.Param("account_uid")
	if accountUID == "" {
		response.Error(c, http.StatusBadRequest, "缺少账户 UID")
		return
	}

	req := &service.OAuth2TokenRefreshRequest{
		AccountUID: accountUID,
	}

	resp, err := h.oauth2Service.RefreshToken(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to refresh Google OAuth2 token", "account_uid", accountUID, "error", err)
		response.Error(c, http.StatusInternalServerError, "刷新访问令牌失败")
		return
	}

	response.Success(c, resp)
}

// GoogleRevoke 撤销 Google OAuth2 访问令牌
// @Summary 撤销 Google OAuth2 访问令牌
// @Description 撤销指定账户的 Google OAuth2 访问令牌
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param account_uid path string true "账户 UID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/google/revoke/{account_uid} [post]
func (h *OAuth2Handler) GoogleRevoke(c *gin.Context) {
	accountUID := c.Param("account_uid")
	if accountUID == "" {
		response.Error(c, http.StatusBadRequest, "缺少账户 UID")
		return
	}

	err := h.oauth2Service.RevokeToken(c.Request.Context(), accountUID)
	if err != nil {
		h.logger.Error("Failed to revoke Google OAuth2 token", "account_uid", accountUID, "error", err)
		response.Error(c, http.StatusInternalServerError, "撤销访问令牌失败")
		return
	}

	response.Success(c, "访问令牌已撤销")
}

// MicrosoftAuthorize 生成 Microsoft OAuth2 授权 URL
// @Summary 生成 Microsoft OAuth2 授权 URL
// @Description 生成 Microsoft OAuth2 授权 URL，用户需要访问此 URL 进行授权
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param email query string false "邮箱地址（可选，用于预填充）"
// @Success 200 {object} response.Response{data=service.OAuth2AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/microsoft/authorize [get]
func (h *OAuth2Handler) MicrosoftAuthorize(c *gin.Context) {
	email := c.Query("email")

	req := &service.OAuth2AuthRequest{
		Provider: service.OAuth2ProviderMicrosoft,
		Email:    email,
	}

	resp, err := h.oauth2Service.GenerateAuthURL(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to generate Microsoft OAuth2 auth URL", "error", err)
		response.Error(c, http.StatusInternalServerError, "生成授权链接失败")
		return
	}

	response.Success(c, resp)
}

// MicrosoftCallback 处理 Microsoft OAuth2 授权回调
// @Summary 处理 Microsoft OAuth2 授权回调
// @Description 处理 Microsoft OAuth2 授权回调，完成账户创建或更新
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param code query string true "授权码"
// @Param state query string true "状态参数"
// @Success 200 {object} response.Response{data=service.OAuth2CallbackResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/microsoft/callback [get]
func (h *OAuth2Handler) MicrosoftCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	error := c.Query("error")

	h.logger.Info("Microsoft OAuth2 callback received", "code_length", len(code), "state", state, "error", error)

	// 检查是否有错误
	if error != "" {
		h.logger.Error("Microsoft OAuth2 authorization error", "error", error)
		// 返回错误页面
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>授权失败</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .error { color: #dc3545; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="error">❌ Microsoft 授权失败</div>
    <div class="info">错误：%s</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
            success: false,
            error: '%s'
        }));
        setTimeout(function() { window.close(); }, 2000);
    </script>
</body>
</html>`, error, error)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	if code == "" || state == "" {
		h.logger.Error("Missing required parameters for Microsoft OAuth2", "code_empty", code == "", "state_empty", state == "")
		// 返回错误页面
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>授权失败</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .error { color: #dc3545; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="error">❌ Microsoft 授权失败</div>
    <div class="info">缺少必要参数</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
            success: false,
            error: '缺少必要参数'
        }));
        setTimeout(function() { window.close(); }, 2000);
    </script>
</body>
</html>`
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	req := &service.OAuth2CallbackRequest{
		Provider: service.OAuth2ProviderMicrosoft,
		Code:     code,
		State:    state,
	}

	h.logger.Info("Processing Microsoft OAuth2 callback", "provider", req.Provider)

	resp, err := h.oauth2Service.HandleCallback(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to handle Microsoft OAuth2 callback", "error", err, "state", state, "code_length", len(code))
		// 返回错误页面
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>授权失败</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .error { color: #dc3545; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="error">❌ Microsoft 授权处理失败</div>
    <div class="info">请稍后重试</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
            success: false,
            error: 'Microsoft 授权处理失败'
        }));
        setTimeout(function() { window.close(); }, 2000);
    </script>
</body>
</html>`
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	h.logger.Info("Microsoft OAuth2 callback processed successfully", "account_uid", resp.AccountUID, "email", resp.Email)

	// 返回一个简单的 HTML 页面，用于关闭弹窗并传递结果
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Microsoft 授权成功</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .success { color: #28a745; font-size: 18px; margin-bottom: 20px; }
        .info { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="success">✅ Microsoft 授权成功！</div>
    <div class="info">账户 %s 已成功添加</div>
    <div class="info">窗口将自动关闭...</div>
    <script>
        console.log('Microsoft OAuth2 授权成功页面加载');
        
        // 将结果存储到 sessionStorage
        const authResult = {
            success: true,
            account_uid: '%s',
            email: '%s',
            provider: 'microsoft'
        };
        
        console.log('存储 Microsoft 授权结果到 sessionStorage:', authResult);
        
        try {
            // 方法1：使用 sessionStorage
            sessionStorage.setItem('oauth2_auth_result', JSON.stringify(authResult));
            
            // 验证存储是否成功
            const stored = sessionStorage.getItem('oauth2_auth_result');
            console.log('验证存储结果:', stored);
            
            // 方法2：使用 postMessage 向父窗口发送消息
            if (window.opener) {
                console.log('向父窗口发送 Microsoft 授权消息:', authResult);
                window.opener.postMessage({
                    type: 'oauth2_result',
                    data: authResult
                }, '*');
            }
            
        } catch (error) {
            console.error('存储 Microsoft 授权结果时出错:', error);
        }
        
        // 关闭窗口
        setTimeout(function() {
            console.log('准备关闭 Microsoft 授权窗口');
            window.close();
        }, 2000);
    </script>
</body>
</html>`, resp.Email, resp.AccountUID, resp.Email)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// MicrosoftRefresh 刷新 Microsoft OAuth2 访问令牌
// @Summary 刷新 Microsoft OAuth2 访问令牌
// @Description 刷新指定账户的 Microsoft OAuth2 访问令牌
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param account_uid path string true "账户 UID"
// @Success 200 {object} response.Response{data=service.OAuth2TokenRefreshResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/microsoft/refresh/{account_uid} [post]
func (h *OAuth2Handler) MicrosoftRefresh(c *gin.Context) {
	accountUID := c.Param("account_uid")
	if accountUID == "" {
		response.Error(c, http.StatusBadRequest, "缺少账户 UID")
		return
	}

	req := &service.OAuth2TokenRefreshRequest{
		AccountUID: accountUID,
	}

	resp, err := h.oauth2Service.RefreshToken(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to refresh Microsoft OAuth2 token", "account_uid", accountUID, "error", err)
		response.Error(c, http.StatusInternalServerError, "刷新访问令牌失败")
		return
	}

	response.Success(c, resp)
}

// MicrosoftRevoke 撤销 Microsoft OAuth2 访问令牌
// @Summary 撤销 Microsoft OAuth2 访问令牌
// @Description 撤销指定账户的 Microsoft OAuth2 访问令牌
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param account_uid path string true "账户 UID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/microsoft/revoke/{account_uid} [post]
func (h *OAuth2Handler) MicrosoftRevoke(c *gin.Context) {
	accountUID := c.Param("account_uid")
	if accountUID == "" {
		response.Error(c, http.StatusBadRequest, "缺少账户 UID")
		return
	}

	err := h.oauth2Service.RevokeToken(c.Request.Context(), accountUID)
	if err != nil {
		h.logger.Error("Failed to revoke Microsoft OAuth2 token", "account_uid", accountUID, "error", err)
		response.Error(c, http.StatusInternalServerError, "撤销访问令牌失败")
		return
	}

	response.Success(c, "访问令牌已撤销")
}
