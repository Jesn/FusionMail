package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type sessionUserStore interface {
	GetUserByID(id int64) (*model.User, error)
}

// AuthMiddleware JWT 认证中间件
type AuthMiddleware struct {
	jwtSecret string
	userStore sessionUserStore
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return NewAuthMiddlewareWithUserStore(jwtSecret, service.NewInitService())
}

func NewAuthMiddlewareWithUserStore(jwtSecret string, userStore sessionUserStore) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
		userStore: userStore,
	}
}

// RequireAuth JWT 认证中间件
// 支持从 Cookie (fm_session) 或 Authorization 头获取 token
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractAuthToken(c)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "未提供认证信息",
			})
			c.Abort()
			return
		}

		claims, user, err := m.authenticateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "无效的认证信息",
			})
			c.Abort()
			return
		}

		setAuthContext(c, claims, user)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求认证）
// 支持从 Cookie (fm_session) 或 Authorization 头获取 token
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractAuthToken(c)
		if tokenString == "" {
			c.Next()
			return
		}

		claims, user, err := m.authenticateToken(tokenString)
		if err == nil {
			setAuthContext(c, claims, user)
		}

		c.Next()
	}
}

func extractAuthToken(c *gin.Context) string {
	if cookie, err := c.Cookie("fm_session"); err == nil && cookie != "" {
		return cookie
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	if strings.HasPrefix(authHeader, "Bearer ") {
		return authHeader[7:]
	}
	return authHeader
}

func (m *AuthMiddleware) authenticateToken(tokenString string) (jwt.MapClaims, *model.User, error) {
	claims, err := m.parseSignedTokenClaims(tokenString)
	if err != nil {
		return nil, nil, err
	}

	userID, err := parseSubjectClaim(claims)
	if err != nil {
		return nil, nil, err
	}
	if m.userStore == nil {
		return nil, nil, errors.New("user store is not configured")
	}

	user, err := m.userStore.GetUserByID(userID)
	if err != nil || user == nil {
		return nil, nil, errors.New("user not found")
	}
	if !user.IsActive {
		return nil, nil, errors.New("user is disabled")
	}
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, nil, errors.New("user is locked")
	}
	if !sessionVersionMatches(claims, user.SessionVersion) {
		return nil, nil, errors.New("stale token session")
	}

	return claims, user, nil
}

func (m *AuthMiddleware) parseSignedTokenClaims(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(m.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func parseSubjectClaim(claims jwt.MapClaims) (int64, error) {
	switch subject := claims["sub"].(type) {
	case float64:
		return int64(subject), nil
	case string:
		userID, err := strconv.ParseInt(subject, 10, 64)
		if err != nil {
			return 0, err
		}
		return userID, nil
	default:
		return 0, errors.New("invalid subject claim")
	}
}

func parseSessionVersionClaim(claims jwt.MapClaims) (int64, error) {
	switch sessionVersion := claims[service.SessionVersionClaim].(type) {
	case float64:
		return int64(sessionVersion), nil
	case int64:
		return sessionVersion, nil
	case int:
		return int64(sessionVersion), nil
	case string:
		parsed, err := strconv.ParseInt(sessionVersion, 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, errors.New("invalid session version claim")
	}
}

func sessionVersionMatches(claims jwt.MapClaims, currentVersion int64) bool {
	sessionVersion, err := parseSessionVersionClaim(claims)
	return err == nil && sessionVersion == currentVersion
}

func setAuthContext(c *gin.Context, claims jwt.MapClaims, user *model.User) {
	c.Set("userID", strconv.FormatInt(user.ID, 10))
	c.Set("username", user.Username)
	c.Set("role", user.Role)
	c.Set("user_claims", claims)
}
