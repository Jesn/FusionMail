package router

import (
	"fusionmail/internal/handler"
	"fusionmail/internal/middleware"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SetupRouter 配置路由
func SetupRouter(
	authHandler handler.AuthHandlerInterface,
	accountHandler *handler.AccountHandler,
	emailHandler *handler.EmailHandler,
	ruleHandler *handler.RuleHandler,
	webhookHandler *handler.WebhookHandler,
	systemHandler *handler.SystemHandler,
	oauth2Handler *handler.OAuth2Handler, // 新增 OAuth2 处理器
	apiKeyHandler *handler.APIKeyHandler, // 新增 API Key 处理器
	publicHandler *handler.PublicHandler, // 新增公共接口处理器
	settingHandler *handler.SettingHandler, // 新增 Setting 处理器
	oauth2ClientHandler *handler.OAuth2ClientHandler, // 新增 OAuth2Client 处理器
	providerHandler *handler.ProviderHandler, // 新增 Provider 处理器
	devSyncHandler *handler.DevSyncHandler, // 新增开发环境同步处理器
	emailListHandler *handler.EmailListHandler, // 新增白名单/黑名单处理器
	syncManager *service.SyncManager,
	redisClient *redis.Client,
	jwtSecret string,
	apiKeyRepo *repository.APIKeyRepository, // 新增 API Key 仓库
	rateLimitEnabled bool,
	siteRatePerMin int,
	publicRatePerMin int,
) *gin.Engine {
	// 创建路由器
	router := gin.New()

	// 全局中间件
	router.Use(middleware.Recovery())           // 错误恢复
	router.Use(middleware.Logger())             // 日志
	router.Use(middleware.CORS())               // CORS
	router.Use(middleware.CSP())                // CSP 安全策略
	router.Use(middleware.ResponseMiddleware()) // 统一响应格式

	// 创建认证中间件
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)

	// SSE 处理器（Cookie/Bearer 校验在处理器内）
	sseHandler := handler.NewSSEHandler(jwtSecret)

	// 创建 API Key 中间件
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(apiKeyRepo)

	// 创建速率限制中间件（默认使用站点限速作为默认值）
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, siteRatePerMin)

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

		// 获取邮箱提供商列表（无需认证）
		api.GET("/system/providers", systemHandler.GetProviders)

		// SSE
		api.GET("/events", sseHandler.Stream)

		// 认证接口（无需认证，但按站点限速配置）
		auth := api.Group("/auth")
		if rateLimitEnabled {
			auth.Use(rateLimitMiddleware.LimitWithRate(siteRatePerMin))
		}
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/verify", authHandler.Verify)

			// 需要认证的认证相关接口
			authWithAuth := auth.Group("")
			authWithAuth.Use(authMiddleware.RequireAuth())
			{
				authWithAuth.POST("/change-password", authHandler.ChangePassword)
				authWithAuth.GET("/me", authHandler.GetCurrentUser)
			}

			// Google OAuth2 端点
			auth.GET("/google/authorize", oauth2Handler.GoogleAuthorize)
			auth.GET("/google/callback", oauth2Handler.GoogleCallback)  // Google 重定向使用 GET
			auth.POST("/google/callback", oauth2Handler.GoogleCallback) // 前端调用使用 POST
			auth.POST("/google/refresh/:account_uid", oauth2Handler.GoogleRefresh)
			auth.POST("/google/revoke/:account_uid", oauth2Handler.GoogleRevoke)

			// Microsoft OAuth2 端点
			auth.GET("/microsoft/authorize", oauth2Handler.MicrosoftAuthorize)
			auth.GET("/microsoft/callback", oauth2Handler.MicrosoftCallback) // 修改为 GET
			auth.POST("/microsoft/refresh/:account_uid", oauth2Handler.MicrosoftRefresh)
			auth.POST("/microsoft/revoke/:account_uid", oauth2Handler.MicrosoftRevoke)
		}

		// 公共接口（仅允许 API Key）
		public := api.Group("/public")
		public.Use(apiKeyMiddleware.RequireAPIKeyOnly())
		if rateLimitEnabled {
			public.Use(rateLimitMiddleware.LimitWithRate(publicRatePerMin))
		}
		{
			// 邮件接口
			mail := public.Group("/mail")
			{
				mail.GET("/receive", publicHandler.ReceiveMail)
				mail.GET("/search", publicHandler.SearchMail)
			}
		}

		// 需要认证的接口（仅允许 JWT）
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		if rateLimitEnabled {
			protected.Use(rateLimitMiddleware.LimitWithRate(siteRatePerMin))
		}
		{
			// 账户管理接口
			accounts := protected.Group("/accounts")
			{
				accounts.POST("", accountHandler.Create)
				accounts.POST("/batch-import", accountHandler.BatchImport) // 批量导入（必须在 "" 之后，避免路由冲突）
				accounts.GET("", accountHandler.List)
				accounts.GET("/trash", accountHandler.ListDeleted) // 获取回收站账号
				accounts.GET("/:uid", accountHandler.GetByUID)
				accounts.PUT("/:uid", accountHandler.Update)
				accounts.DELETE("/:uid", accountHandler.Delete)
				accounts.POST("/:uid/restore", accountHandler.Restore)
				accounts.DELETE("/:uid/force", accountHandler.ForceDelete)
				accounts.POST("/:uid/test", accountHandler.TestConnection)
				accounts.POST("/:uid/sync", accountHandler.SyncAccount)
				accounts.POST("/:uid/disable", accountHandler.DisableAccount)
				accounts.POST("/:uid/enable", accountHandler.EnableAccount)
				accounts.POST("/:uid/clear-error", accountHandler.ClearSyncError)
			}

			// OAuth2 客户端管理接口
			oauth2Clients := protected.Group("/oauth2/clients")
			{
				oauth2Clients.POST("", oauth2ClientHandler.Create)
				oauth2Clients.GET("", oauth2ClientHandler.List)
				oauth2Clients.GET("/:id", oauth2ClientHandler.GetByID)
				oauth2Clients.PUT("/:id", oauth2ClientHandler.Update)
				oauth2Clients.DELETE("/:id", oauth2ClientHandler.Delete)
				oauth2Clients.GET("/provider/:provider_type", oauth2ClientHandler.GetByProvider)
				oauth2Clients.GET("/provider/:provider_type/default", oauth2ClientHandler.GetDefault)
				oauth2Clients.POST("/:id/default/:provider_type", oauth2ClientHandler.SetDefault)
				oauth2Clients.GET("/smart-select/:provider_type", oauth2ClientHandler.SmartSelect)
			}

			// Provider 管理接口（仅支持ID查询）
			providers := protected.Group("/providers")
			{
				providers.POST("", providerHandler.Create)
				providers.GET("", providerHandler.ListWithPagination) // 分页列表
				providers.GET("/all", providerHandler.List)           // 全部列表（无分页）
				providers.GET("/:id", providerHandler.GetByID)
				providers.PUT("/:id", providerHandler.UpdateByID)
				providers.DELETE("/:id", providerHandler.DeleteByID)
			}

			// 开发环境数据同步接口（仅用于开发测试）
			dev := protected.Group("/dev")
			{
				dev.POST("/sync-from-env", devSyncHandler.SyncFromEnv)
			}

			// 邮件管理接口
			emails := protected.Group("/emails")
			{
				emails.GET("", emailHandler.GetEmailList)
				emails.GET("/search", emailHandler.SearchEmails)
				emails.GET("/unread-count", emailHandler.GetUnreadCount)
				emails.GET("/stats", emailHandler.GetGlobalStats)
				emails.GET("/stats/:account_uid", emailHandler.GetAccountStats)
				emails.GET("/:id", emailHandler.GetEmailByID)
				emails.POST("/mark-read", emailHandler.MarkAsRead)
				emails.POST("/mark-unread", emailHandler.MarkAsUnread)
				emails.POST("/mark-all-read", emailHandler.MarkAllAsRead)
				emails.POST("/:id/toggle-star", emailHandler.ToggleStar)
				emails.POST("/:id/archive", emailHandler.ArchiveEmail)
				emails.DELETE("/:id", emailHandler.DeleteEmail)
				emails.POST("/:id/restore", emailHandler.RestoreEmail)

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

			// 白名单/黑名单管理接口
			emailList := protected.Group("/emaillist")
			{
				// 白名单接口
				emailList.GET("/whitelist", emailListHandler.GetWhitelist)
				emailList.POST("/whitelist", emailListHandler.AddToWhitelist)
				emailList.DELETE("/whitelist/:id", emailListHandler.DeleteFromWhitelist)

				// 黑名单接口
				emailList.GET("/blacklist", emailListHandler.GetBlacklist)
				emailList.POST("/blacklist", emailListHandler.AddToBlacklist)
				emailList.DELETE("/blacklist/:id", emailListHandler.DeleteFromBlacklist)
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

			// API Key 管理接口
			apiKeys := protected.Group("/api-keys")
			{
				apiKeys.POST("", apiKeyHandler.Create)
				apiKeys.GET("", apiKeyHandler.List)
				apiKeys.GET("/:id", apiKeyHandler.GetByID)
				apiKeys.PUT("/:id", apiKeyHandler.Update)
				apiKeys.DELETE("/:id", apiKeyHandler.Delete)
				apiKeys.POST("/:id/enable", apiKeyHandler.Enable)
				apiKeys.POST("/:id/disable", apiKeyHandler.Disable)
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

				sync.GET("/status", systemHandler.GetSyncStatus)

				// 同步日志接口（用于问题排查和历史记录查询）
				sync.GET("/logs", systemHandler.GetSyncLogs)
			}

			// 系统管理接口（用于运维监控和系统诊断）
			system := protected.Group("/system")
			{
				// 健康检查接口（可用于 K8s liveness/readiness probe）
				system.GET("/health", systemHandler.GetHealth)
				// 系统统计接口（用于监控和运营分析）
				system.GET("/stats", systemHandler.GetStats)
			}

			// 设置管理接口（用户级和系统级配置）
			settings := protected.Group("/settings")
			{
				// 按分类获取配置
				settings.GET("/:category", settingHandler.GetSettingsByCategory)

				// 批量设置配置
				settings.POST("/:category", settingHandler.SetSettings)

				// 获取单个配置
				settings.GET("/:category/:key", settingHandler.GetSetting)

				// 设置单个配置
				settings.PUT("/:category/:key", settingHandler.SetSetting)

				// 删除配置
				settings.DELETE("/:category/:key", settingHandler.DeleteSetting)

				// 重置配置为默认值
				settings.POST("/:category/:key/reset", settingHandler.ResetSetting)

				// 搜索配置
				settings.GET("/search", settingHandler.SearchSettings)

				// 获取缓存统计（仅管理员）
				settings.GET("/stats", settingHandler.GetStats)

				// 预热缓存（仅管理员）
				settings.POST("/warmup", settingHandler.WarmUp)

				// 导出配置（仅管理员）
				settings.GET("/export", settingHandler.ExportSettings)

				// 导入配置（仅管理员）
				settings.POST("/import", settingHandler.ImportSettings)
			}

			// 系统级配置管理（仅管理员）
			systemSettings := protected.Group("/settings/system")
			{
				systemSettings.GET("/:category/:key", settingHandler.GetSystem)
				systemSettings.POST("/:category/:key", settingHandler.SetSystem)
			}
		}

		// 公开配置接口（无需认证，但有限速）
		publicSettings := api.Group("/settings")
		{
			// 获取公开配置（前端可访问）
			publicSettings.GET("/public", settingHandler.GetPublicSettings)
		}
	}

	return router
}
