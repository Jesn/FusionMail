package router

import (
	"fusionmail/internal/handler"
	"fusionmail/internal/middleware"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SetupRouter 配置路由
func SetupRouter(
	authHandler *handler.AuthHandler,
	accountHandler *handler.AccountHandler,
	emailHandler *handler.EmailHandler,
	ruleHandler *handler.RuleHandler,
	webhookHandler *handler.WebhookHandler,
	systemHandler *handler.SystemHandler,
	syncManager *service.SyncManager,
	redisClient *redis.Client,
	jwtSecret string,
) *gin.Engine {
	// 创建路由器
	router := gin.New()

	// 全局中间件
	router.Use(middleware.Recovery())           // 错误恢复
	router.Use(middleware.Logger())             // 日志
	router.Use(middleware.CORS())               // CORS
	router.Use(middleware.ResponseMiddleware()) // 统一响应格式

	// 创建认证中间件
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)

	// 创建速率限制中间件
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, 200) // 默认每分钟 200 次（测试环境）

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 健康检查端点（无需认证）
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "fusionmail",
				"version": "0.1.0",
			})
		})

		// 认证接口（无需认证，但有速率限制）
		auth := api.Group("/auth")
		auth.Use(rateLimitMiddleware.LimitWithRate(100)) // 登录接口限制（测试环境）
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/verify", authHandler.Verify)
		}

		// 需要认证的接口
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		protected.Use(rateLimitMiddleware.Limit())
		{
			// 账户管理接口
			accounts := protected.Group("/accounts")
			{
				accounts.POST("", accountHandler.Create)
				accounts.GET("", accountHandler.List)
				accounts.GET("/:uid", accountHandler.GetByUID)
				accounts.PUT("/:uid", accountHandler.Update)
				accounts.DELETE("/:uid", accountHandler.Delete)
				accounts.POST("/:uid/test", accountHandler.TestConnection)
				accounts.POST("/:uid/sync", accountHandler.SyncAccount)
			}

			// 邮件管理接口
			emails := protected.Group("/emails")
			{
				emails.GET("", emailHandler.GetEmailList)
				emails.GET("/search", emailHandler.SearchEmails)
				emails.GET("/unread-count", emailHandler.GetUnreadCount)
				emails.GET("/stats/:account_uid", emailHandler.GetAccountStats)
				emails.GET("/:id", emailHandler.GetEmailByID)
				emails.POST("/mark-read", emailHandler.MarkAsRead)
				emails.POST("/mark-unread", emailHandler.MarkAsUnread)
				emails.POST("/:id/toggle-star", emailHandler.ToggleStar)
				emails.POST("/:id/archive", emailHandler.ArchiveEmail)
				emails.DELETE("/:id", emailHandler.DeleteEmail)
			}

			// 规则管理接口
			rules := protected.Group("/rules")
			{
				rules.POST("", ruleHandler.CreateRule)
				rules.GET("", ruleHandler.ListRules)
				rules.GET("/:id", ruleHandler.GetRuleByID)
				rules.PUT("/:id", ruleHandler.UpdateRule)
				rules.DELETE("/:id", ruleHandler.DeleteRule)
				rules.POST("/:id/toggle", ruleHandler.ToggleRule)
				rules.POST("/:id/test", ruleHandler.TestRule)
			}

			// Webhook 管理接口
			webhooks := protected.Group("/webhooks")
			{
				webhooks.POST("", webhookHandler.CreateWebhook)
				webhooks.GET("", webhookHandler.GetWebhookList)
				webhooks.GET("/:id", webhookHandler.GetWebhookByID)
				webhooks.PUT("/:id", webhookHandler.UpdateWebhook)
				webhooks.DELETE("/:id", webhookHandler.DeleteWebhook)
				webhooks.POST("/:id/toggle", webhookHandler.ToggleWebhook)
				webhooks.POST("/:id/test", webhookHandler.TestWebhook)
				webhooks.GET("/:id/logs", webhookHandler.GetWebhookLogs)
			}

			// 附件管理接口（待实现）
			// attachments := protected.Group("/attachments")
			// {
			// 	attachments.GET("/:id", attachmentHandler.GetAttachment)
			// 	attachments.GET("/:id/download", attachmentHandler.DownloadAttachment)
			// 	attachments.DELETE("/:id", attachmentHandler.DeleteAttachment)
			// }

			// 邮件附件接口（待实现）
			// emails.GET("/:id/attachments", attachmentHandler.GetEmailAttachments)

			// 同步管理接口
			sync := protected.Group("/sync")
			{
				sync.POST("/accounts/:uid", func(c *gin.Context) {
					accountUID := c.Param("uid")
					if err := syncManager.SyncAccount(c.Request.Context(), accountUID); err != nil {
						c.JSON(500, gin.H{
							"success": false,
							"error":   err.Error(),
						})
						return
					}
					c.JSON(200, gin.H{
						"success": true,
						"message": "同步任务已启动",
					})
				})

				sync.POST("/all", func(c *gin.Context) {
					if err := syncManager.SyncAllAccounts(c.Request.Context()); err != nil {
						c.JSON(500, gin.H{
							"success": false,
							"error":   err.Error(),
						})
						return
					}
					c.JSON(200, gin.H{
						"success": true,
						"message": "所有账户同步任务已启动",
					})
				})

				sync.GET("/status", func(c *gin.Context) {
					c.JSON(200, gin.H{
						"success": true,
						"data": gin.H{
							"running": syncManager.IsRunning(),
						},
					})
				})

				// 同步日志接口
				sync.GET("/logs", systemHandler.GetSyncLogs)
			}

			// 系统管理接口
			system := protected.Group("/system")
			{
				system.GET("/health", systemHandler.GetHealth)
				system.GET("/stats", systemHandler.GetStats)
			}
		}
	}

	return router
}
