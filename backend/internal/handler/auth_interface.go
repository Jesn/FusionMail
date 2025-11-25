package handler

import "github.com/gin-gonic/gin"

// AuthHandlerInterface 认证处理器接口
type AuthHandlerInterface interface {
	Login(c *gin.Context)
	Logout(c *gin.Context)
	RefreshToken(c *gin.Context)
	Verify(c *gin.Context)
	ChangePassword(c *gin.Context)
	GetCurrentUser(c *gin.Context)
}