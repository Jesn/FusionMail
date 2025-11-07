package middleware

import (
	"runtime/debug"

	"fusionmail/internal/dto"
	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware 统一错误处理中间件
// 捕获 panic 并处理未处理的错误
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 信息和堆栈
				stack := string(debug.Stack())

				// 使用结构化日志记录 panic
				reqLogger := logger.WithRequestID(c)
				reqLogger.Error("panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"stack", stack,
				)

				// 返回 500 错误
				dto.InternalServerErrorResponse(c, "服务器内部错误")
				c.Abort()
			}
		}()

		c.Next()

		// 处理错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// 返回错误响应
			dto.HandleServiceError(c, err)
			c.Abort()
		}
	}
}
