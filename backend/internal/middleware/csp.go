package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CSP 中间件
// Content Security Policy - 防止 XSS 攻击
func CSP() gin.HandlerFunc {
	return func(c *gin.Context) {
		// CSP 策略配置
		cspDirectives := []string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'", // 'unsafe-inline' 用于 Vite 开发，线上可考虑移除
			"style-src 'self' 'unsafe-inline'",                // 'unsafe-inline' 用于 Tailwind CSS
			"img-src 'self' data: blob: https: http:",         // 允许 data URL 和 blob URL
			"font-src 'self' data:",
			"connect-src 'self' https: http: ws: wss:",        // 允许 API 连接
			"frame-src 'none'",                                // 禁止 iframe
			"object-src 'none'",                               // 禁止 object/embed
			"base-uri 'self'",
			"form-action 'self'",                              // 表单只能提交到同源
			"upgrade-insecure-requests",                       // 自动升级 HTTP 到 HTTPS
		}

		// 设置 CSP 头
		c.Header("Content-Security-Policy", strings.Join(cspDirectives, "; "))

		// 其他安全相关头
		c.Header("X-Content-Type-Options", "nosniff")  // 防止 MIME 类型嗅探
		c.Header("X-Frame-Options", "DENY")             // 禁止点击劫持
		c.Header("X-XSS-Protection", "1; mode=block")   // 启用 XSS 保护
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin") // 引用者策略

		// HSTS - 强制 HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
