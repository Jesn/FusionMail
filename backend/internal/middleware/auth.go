package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AuthMiddleware JWT 认证中间件
type AuthMiddleware struct {
	jwtSecret string
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// RequireAuth JWT 认证中间件
// 支持从 Cookie (fm_session) 或 Authorization 头获取 token
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 Cookie 读取 fm_session
		if cookie, err := c.Cookie("fm_session"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// 如果 Cookie 中没有，尝试从 Authorization 头获取
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				// 解析 Bearer token
				if strings.HasPrefix(authHeader, "Bearer ") {
					tokenString = authHeader[7:]
				} else {
					tokenString = authHeader
				}
			}
		}

		// 如果都没有，返回未认证错误
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "未提供认证信息",
			})
			c.Abort()
			return
		}

		// 验证 token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "无效的认证信息",
			})
			c.Abort()
			return
		}

		// 提取 claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// 将用户信息存储到上下文
			c.Set("userID", claims["sub"])        // 用户ID
			c.Set("username", claims["username"]) // 用户名
			c.Set("role", claims["role"])         // 用户角色
			c.Set("user_claims", claims)
		}

		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求认证）
// 支持从 Cookie (fm_session) 或 Authorization 头获取 token
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 Cookie 读取 fm_session
		if cookie, err := c.Cookie("fm_session"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// 如果 Cookie 中没有，尝试从 Authorization 头获取
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				if strings.HasPrefix(authHeader, "Bearer ") {
					tokenString = authHeader[7:]
				} else {
					tokenString = authHeader
				}
			}
		}

		// 如果没有 token，继续处理（可选认证）
		if tokenString == "" {
			c.Next()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.jwtSecret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				c.Set("userID", claims["sub"])
				c.Set("username", claims["username"])
				c.Set("role", claims["role"])
				c.Set("user_claims", claims)
			}
		}

		c.Next()
	}
}
