package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"fusionmail/internal/repository"

	"github.com/gin-gonic/gin"
)

// APIKeyMiddleware API Key 认证中间件
type APIKeyMiddleware struct {
	apiKeyRepo *repository.APIKeyRepository
}

// NewAPIKeyMiddleware 创建 API Key 认证中间件
func NewAPIKeyMiddleware(apiKeyRepo *repository.APIKeyRepository) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		apiKeyRepo: apiKeyRepo,
	}
}

// RequireAPIKey API Key 认证中间件
func (m *APIKeyMiddleware) RequireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 API Key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// 也支持从 Authorization 头获取
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "ApiKey ") {
				apiKey = authHeader[7:]
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "未提供 API Key",
			})
			c.Abort()
			return
		}

		// 计算 API Key 的哈希值
		hash := sha256.Sum256([]byte(apiKey))
		keyHash := hex.EncodeToString(hash[:])

		// 验证 API Key
		key, err := m.apiKeyRepo.FindByHash(c.Request.Context(), keyHash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "无效的 API Key",
			})
			c.Abort()
			return
		}

		// 检查是否启用
		if !key.Enabled {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "API Key 已被禁用",
			})
			c.Abort()
			return
		}

		// 检查是否过期
		if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "API Key 已过期",
			})
			c.Abort()
			return
		}

		// 更新使用统计
		go func() {
			_ = m.apiKeyRepo.UpdateUsage(c.Request.Context(), key.ID, c.ClientIP())
		}()

		// 将 API Key 信息存储到上下文
		c.Set("api_key_id", key.ID)
		c.Set("api_key_name", key.Name)
		c.Set("api_key_permissions", key.Permissions)

		c.Next()
	}
}

// AllowAPIKeyOrJWT 允许 API Key 或 JWT 认证
func (m *APIKeyMiddleware) AllowAPIKeyOrJWT(jwtMiddleware *AuthMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先检查 API Key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "ApiKey ") {
				apiKey = authHeader[7:]
			}
		}

		if apiKey != "" {
			// 使用 API Key 认证
			m.RequireAPIKey()(c)
			return
		}

		// 使用 JWT 认证
		jwtMiddleware.RequireAuth()(c)
	}
}
