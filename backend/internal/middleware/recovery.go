package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery 错误恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 打印堆栈信息
				fmt.Printf("\033[31m[PANIC RECOVERED] %v\n%s\033[0m\n", err, debug.Stack())

				// 返回 500 错误
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "服务器内部错误",
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
