package handler

import (
	"fmt"
	"net/http"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/service"
	"fusionmail/internal/sse"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type sseUserStore interface {
	GetUserByID(id int64) (*model.User, error)
}

var sseHeartbeatInterval = 25 * time.Second

// SSEHandler 负责处理基于 Cookie 的 SSE 长连接（校验 fm_session 或 Bearer）
type SSEHandler struct {
	jwtSecret string
	userStore sseUserStore
}

func NewSSEHandler(jwtSecret string) *SSEHandler {
	return NewSSEHandlerWithUserStore(jwtSecret, service.NewInitService())
}

func NewSSEHandlerWithUserStore(jwtSecret string, userStore sseUserStore) *SSEHandler {
	return &SSEHandler{jwtSecret: jwtSecret, userStore: userStore}
}

// Stream 建立 SSE 连接并将服务器事件推送给客户端
func (h *SSEHandler) Stream(c *gin.Context) {
	// 记录请求信息
	fmt.Printf("[SSE] 收到连接请求 - Origin: %s, Cookie: %v\n",
		c.GetHeader("Origin"),
		c.Request.Header.Get("Cookie") != "")

	// 认证：优先从 Cookie 读取 fm_session，其次尝试 Bearer 头
	tokenString, err := c.Cookie("fm_session")
	if err != nil || tokenString == "" {
		fmt.Printf("[SSE] Cookie 认证失败: %v, 尝试 Bearer 认证\n", err)
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			tokenString = auth[7:]
			fmt.Printf("[SSE] 使用 Bearer token\n")
		}
	} else {
		fmt.Printf("[SSE] 使用 Cookie token\n")
	}

	if tokenString == "" {
		fmt.Printf("[SSE] 认证失败: 未找到 token\n")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if err := h.validateSessionToken(tokenString); err != nil {
		fmt.Printf("[SSE] Token 验证失败: %v\n", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	fmt.Printf("[SSE] 认证成功，建立连接\n")

	// 设置必要的 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 兼容 Nginx 关闭缓冲
	c.Header("Vary", "Origin")

	// 确保支持刷新
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	h.disableWriteDeadline(c.Writer)

	// 订阅事件
	ch, unsubscribe := sse.Subscribe()
	defer unsubscribe()

	// 初始握手：发送一条注释，避免某些代理超时
	fmt.Fprintf(c.Writer, ": connected\n\n")
	flusher.Flush()

	// 心跳（ping）避免链路被中间件关闭
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	clientGone := c.Writer.CloseNotify()

	for {
		select {
		case <-clientGone:
			return
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			if err := h.validateSessionToken(tokenString); err != nil {
				fmt.Printf("[SSE] 会话已失效，断开连接: %v\n", err)
				return
			}
			fmt.Fprintf(c.Writer, "event: ping\n")
			fmt.Fprintf(c.Writer, "data: {}\n\n")
			flusher.Flush()
		case ev := <-ch:
			if err := h.validateSessionToken(tokenString); err != nil {
				fmt.Printf("[SSE] 会话已失效，停止推送: %v\n", err)
				return
			}
			// 写入标准 SSE 帧
			fmt.Fprintf(c.Writer, "event: %s\n", ev.Type)
			if ev.Data == "" {
				fmt.Fprintf(c.Writer, "data: {}\n\n")
			} else {
				fmt.Fprintf(c.Writer, "data: %s\n\n", ev.Data)
			}
			flusher.Flush()
		}
	}
}

func (h *SSEHandler) disableWriteDeadline(writer http.ResponseWriter) {
	if err := http.NewResponseController(writer).SetWriteDeadline(time.Time{}); err != nil {
		fmt.Printf("[SSE] 禁用写超时失败: %v\n", err)
	}
}

func (h *SSEHandler) validateSessionToken(tokenString string) error {
	claims, err := h.parseSignedTokenClaims(tokenString)
	if err != nil {
		return err
	}
	userID, err := parseSubjectClaim(claims)
	if err != nil {
		return err
	}
	if h.userStore == nil {
		return fmt.Errorf("user store is not configured")
	}
	user, err := h.userStore.GetUserByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}
	if !user.IsActive {
		return fmt.Errorf("user is disabled")
	}
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return fmt.Errorf("user is locked")
	}
	if !sessionVersionClaimMatches(claims, user.SessionVersion) {
		return fmt.Errorf("stale token session")
	}
	return nil
}

func (h *SSEHandler) parseSignedTokenClaims(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenMalformed
	}
	return claims, nil
}
