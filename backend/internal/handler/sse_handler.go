package handler

import (
	"fmt"
	"net/http"
	"time"

	"fusionmail/internal/sse"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SSEHandler 负责处理基于 Cookie 的 SSE 长连接（校验 fm_session 或 Bearer）
type SSEHandler struct {
	jwtSecret string
}

func NewSSEHandler(jwtSecret string) *SSEHandler { return &SSEHandler{jwtSecret: jwtSecret} }

// Stream 建立 SSE 连接并将服务器事件推送给客户端
func (h *SSEHandler) Stream(c *gin.Context) {
	// 记录请求信息
	fmt.Printf("[SSE] 收到连接请求 - Origin: %s, Cookie: %v\n",
		c.GetHeader("Origin"),
		c.Request.Header.Get("Cookie") != "")

	// 认证：优先从 Cookie 读取 fm_session，其次尝试 Bearer 头，最后回退到 query token（用于无 Cookie 场景）
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

	// 支持通过 query 参数传递 token（例如 /events?token=xxx，用于 polyfill / 无 Cookie 环境）
	if tokenString == "" {
		if qToken := c.Query("token"); qToken != "" {
			tokenString = qToken
			fmt.Printf("[SSE] 使用 Query token\n")
		}
	}

	if tokenString == "" {
		fmt.Printf("[SSE] 认证失败: 未找到 token\n")
		c.Status(http.StatusUnauthorized)
		return
	}

	if token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	}); err != nil || !token.Valid {
		fmt.Printf("[SSE] Token 验证失败: %v\n", err)
		c.Status(http.StatusUnauthorized)
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
		c.Status(http.StatusInternalServerError)
		return
	}

	// 订阅事件
	ch, unsubscribe := sse.Subscribe()
	defer unsubscribe()

	// 初始握手：发送一条注释，避免某些代理超时
	fmt.Fprintf(c.Writer, ": connected\n\n")
	flusher.Flush()

	// 心跳（ping）避免链路被中间件关闭
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	clientGone := c.Writer.CloseNotify()

	for {
		select {
		case <-clientGone:
			return
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			fmt.Fprintf(c.Writer, "event: ping\n")
			fmt.Fprintf(c.Writer, "data: {}\n\n")
			flusher.Flush()
		case ev := <-ch:
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
