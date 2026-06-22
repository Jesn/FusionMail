package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "fm_csrf"
	csrfHeaderName = "X-CSRF-Token"
)

// CSRFMiddleware 双提交 cookie 模式的 CSRF 防护
// 仅对非安全方法（POST/PUT/PATCH/DELETE）生效
// 仅在使用 cookie 认证时校验（API Key 认证的路由跳过）
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 安全方法不校验
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			ensureCSRFCookie(c)
			c.Next()
			return
		}

		// API Key 认证的路由跳过 CSRF（API Key 通过 header 传递，不受 CSRF 影响）
		if _, hasAPIKey := c.Get("api_key_id"); hasAPIKey {
			c.Next()
			return
		}

		// Bearer token 认证跳过 CSRF（通过 Authorization header 传递，浏览器不会自动附带）
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			c.Next()
			return
		}

		// 无 session cookie 的请求跳过 CSRF（未认证或非浏览器请求）
		if _, err := c.Cookie("fm_session"); err != nil {
			c.Next()
			return
		}

		// 校验双提交 token
		cookieToken, err := c.Cookie(csrfCookieName)
		if err != nil || cookieToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "CSRF token missing",
			})
			c.Abort()
			return
		}

		headerToken := c.GetHeader(csrfHeaderName)
		if headerToken == "" || headerToken != cookieToken {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "CSRF token mismatch",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ensureCSRFCookie 确保客户端有 CSRF token cookie
func ensureCSRFCookie(c *gin.Context) {
	if _, err := c.Cookie(csrfCookieName); err == nil {
		return // cookie 已存在
	}

	token := generateCSRFToken()
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // 前端 JS 需要读取
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "fallback-csrf-token-00000000"
	}
	return hex.EncodeToString(b)
}
