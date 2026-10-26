package middleware

import (
	"fusionmail/internal/dto"

	"github.com/gin-gonic/gin"
)

// RequireAdmin 要求当前会话具备管理员角色。
//
// 背景：项目此前只有 handler/setting_handler.go 里一处内联的角色判断，
// 其余管理端点（日志读取/下载/清空、/dev/* 运维接口）只挂了 RequireAuth，
// 任何已登录用户都能读取他人操作记录、下载服务器日志、清空审计证据。
//
// 角色来源：AuthMiddleware 在鉴权成功后写入 c.Set("role", user.Role)。
// 本中间件必须挂在 RequireAuth 之后。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			dto.ForbiddenResponse(c, "缺少角��信息")
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok || roleStr != "admin" {
			dto.ForbiddenResponse(c, "需要管理员权限")
			c.Abort()
			return
		}

		c.Next()
	}
}
