package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDMiddleware 为每个请求生成唯一 ID
// 用于追踪和关联日志
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取 Request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 如果没有，生成新的 UUID
			requestID = uuid.New().String()
		}

		// 设置到上下文中
		c.Set("request_id", requestID)

		// 添加到响应头
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// GetRequestID 从上下文中获取 Request ID
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		return requestID.(string)
	}
	return ""
}
