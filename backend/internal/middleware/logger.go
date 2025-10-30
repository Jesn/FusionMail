package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 请求信息
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 获取用户信息（如果已认证）
		userID := ""
		if uid, exists := c.Get("user_id"); exists {
			userID = fmt.Sprintf("%v", uid)
		}

		// 获取 API Key 信息（如果使用 API Key）
		apiKeyName := ""
		if name, exists := c.Get("api_key_name"); exists {
			apiKeyName = fmt.Sprintf("%v", name)
		}

		// 构建日志消息
		logMsg := fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %s",
			endTime.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)

		// 添加用户信息
		if userID != "" {
			logMsg += fmt.Sprintf(" | user:%s", userID)
		}
		if apiKeyName != "" {
			logMsg += fmt.Sprintf(" | apikey:%s", apiKeyName)
		}

		// 添加 User-Agent（可选）
		if userAgent != "" && len(userAgent) > 50 {
			logMsg += fmt.Sprintf(" | UA:%s...", userAgent[:50])
		}

		// 根据状态码选择日志级别
		if statusCode >= 500 {
			fmt.Printf("\033[31m%s\033[0m\n", logMsg) // 红色
		} else if statusCode >= 400 {
			fmt.Printf("\033[33m%s\033[0m\n", logMsg) // 黄色
		} else if statusCode >= 300 {
			fmt.Printf("\033[36m%s\033[0m\n", logMsg) // 青色
		} else {
			fmt.Printf("\033[32m%s\033[0m\n", logMsg) // 绿色
		}
	}
}
