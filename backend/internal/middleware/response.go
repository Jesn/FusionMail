package middleware

import (
	"fusionmail/internal/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResponseMiddleware 响应中间件
func ResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 如果已经有响应，不再处理
		if c.Writer.Written() {
			return
		}

		// 处理未处理的错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// 根据错误类型返回不同的状态码
			switch err.Type {
			case gin.ErrorTypeBind:
				dto.BadRequestResponse(c, "请求参数格式错误")
			case gin.ErrorTypePublic:
				dto.BadRequestResponse(c, err.Error())
			default:
				dto.InternalServerErrorResponse(c, "服务器内部错误")
			}
			return
		}

		// 如果没有设置状态码，默认返回 404
		if c.Writer.Status() == http.StatusOK && !c.Writer.Written() {
			dto.NotFoundResponse(c, "接口不存在")
		}
	}
}

// ErrorHandler 全局错误处理器
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 信息
				c.Error(gin.Error{
					Err:  err.(error),
					Type: gin.ErrorTypePrivate,
				})

				dto.InternalServerErrorResponse(c, "服务器内部错误")
				c.Abort()
			}
		}()

		c.Next()
	}
}
