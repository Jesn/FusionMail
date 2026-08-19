package router

import (
	"context"
	"fusionmail/internal/handler"
	"fusionmail/internal/middleware"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"
	"fusionmail/pkg/database"
	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	_ "fusionmail/docs" // 导入 Swagger 文档
)

// 模块日志记录器
var routerLog = logger.NewWithModule("Router")

type Handlers struct {
	Auth           handler.AuthHandlerInterface
	Account        *handler.AccountHandler
	Email          *handler.EmailHandler
	Rule           *handler.RuleHandler
	Webhook        *handler.WebhookHandler
	System         *handler.SystemHandler
	OAuth2         *handler.OAuth2Handler
	APIKey         *handler.APIKeyHandler
	Public         *handler.PublicHandler
	Setting        *handler.SettingHandler
	OAuth2Client   *handler.OAuth2ClientHandler
	Provider       *handler.ProviderHandler
	Adapter        *handler.AdapterHandler
	DevSync        *handler.DevSyncHandler
	EmailList      *handler.EmailListHandler
	Spam           *handler.SpamHandler
	Reputation     *handler.ReputationHandler
	TwoFactor      *handler.TwoFactorHandler
	TwoFactorLogin *handler.TwoFactorLoginHandler
}

type RateLimitConfig struct {
	Enabled      bool
	SitePerMin   int
	PublicPerMin int
}

type RouterDeps struct {
	Handlers       Handlers
	SyncManager    *service.SyncManager
	RedisClient    *redis.Client
	JWTSecret      string
	CookieSecure   *bool
	AuthMiddleware *middleware.AuthMiddleware
	APIKeyRepo     *repository.APIKeyRepository
	RateLimit      RateLimitConfig
	AppCtx         context.Context
}

func (deps RouterDeps) authMiddleware() *middleware.AuthMiddleware {
	if deps.AuthMiddleware != nil {
		return deps.AuthMiddleware
	}
	return middleware.NewAuthMiddleware(deps.JWTSecret)
}

// SetupRouter 配置路由
func SetupRouter(deps RouterDeps) *gin.Engine {
	authHandler := deps.Handlers.Auth
	accountHandler := deps.Handlers.Account
	emailHandler := deps.Handlers.Email
	ruleHandler := deps.Handlers.Rule
	webhookHandler := deps.Handlers.Webhook
	systemHandler := deps.Handlers.System
	oauth2Handler := deps.Handlers.OAuth2
	apiKeyHandler := deps.Handlers.APIKey
	publicHandler := deps.Handlers.Public
	settingHandler := deps.Handlers.Setting
	oauth2ClientHandler := deps.Handlers.OAuth2Client
	providerHandler := deps.Handlers.Provider
	adapterHandler := deps.Handlers.Adapter
	devSyncHandler := deps.Handlers.DevSync
	emailListHandler := deps.Handlers.EmailList
	spamHandler := deps.Handlers.Spam
	reputationHandler := deps.Handlers.Reputation
	syncManager := deps.SyncManager
	redisClient := deps.RedisClient
	jwtSecret := deps.JWTSecret
	apiKeyRepo := deps.APIKeyRepo
	rateLimitEnabled := deps.RateLimit.Enabled
	siteRatePerMin := deps.RateLimit.SitePerMin
	publicRatePerMin := deps.RateLimit.PublicPerMin
	// 创建路由器
	router := gin.New()

	// 禁用自动重定向（避免 Swagger 路由被重定向）
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// 全局中间件
	router.Use(middleware.Recovery())            // 错误恢复
	router.Use(otelgin.Middleware("fusionmail")) // OpenTelemetry 分布式追踪
	router.Use(middleware.MetricsMiddleware())   // Prometheus 指标
	router.Use(middleware.Logger())              // 日志
	router.Use(middleware.CORS())                // CORS
	router.Use(middleware.CSP())                 // CSP 安全策略
	router.Use(middleware.ResponseMiddleware())  // 统一响应格式

	// 创建认证中间件
	authMiddleware := deps.authMiddleware()

	// SSE 处理器（Cookie/Bearer 校验在处理器内）
	sseHandler := handler.NewSSEHandler(jwtSecret)

	// 创建 API Key 中间件
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(apiKeyRepo)

	// 创建速率限制中间件（默认使用站点限速作为默认值）
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, siteRatePerMin)

	// Swagger 文档路由已移至 main.go 中注册（在静态文件服务之前）
	// 这样可以确保路由优先级正确，避免被 NoRoute 捕获

	// Prometheus metrics 端点（无需认证）
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 健康检查端点（liveness，无需认证）
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "fusionmail",
				"version": "0.1.0",
			})
		})

		// 就绪检查端点（readiness，无需认证）
		// 检查 DB 和 Redis 连通性，用于 K8s readiness probe
		api.GET("/ready", func(c *gin.Context) {
			checks := gin.H{}
			allReady := true

			// 检查 DB
			if db := database.GetDB(); db != nil {
				sqlDB, err := db.DB()
				if err == nil {
					if err := sqlDB.Ping(); err == nil {
						checks["database"] = "ok"
					} else {
						checks["database"] = "fail"
						allReady = false
					}
				} else {
					checks["database"] = "fail"
					allReady = false
				}
			} else {
				checks["database"] = "fail"
				allReady = false
			}

			// 检查 Redis
			if redisClient != nil {
				if err := redisClient.Ping(c.Request.Context()).Err(); err == nil {
					checks["redis"] = "ok"
				} else {
					checks["redis"] = "degraded"
					// Redis 降级不阻止就绪（fail-open）
				}
			} else {
				checks["redis"] = "not_configured"
			}

			status := "ready"
			code := 200
			if !allReady {
				status = "not_ready"
				code = 503
			}
			c.JSON(code, gin.H{
				"status": status,
				"checks": checks,
			})
		})

		// 获取邮箱提供商列表（无需认证）
		api.GET("/system/providers", systemHandler.GetProviders)

		// SSE
		if rateLimitEnabled {
			api.GET("/events", rateLimitMiddleware.LimitWithRate(siteRatePerMin), sseHandler.Stream)
		} else {
			api.GET("/events", sseHandler.Stream)
		}

		// 2FA 处理器（从依赖注入获取）
		twoFactorHandler := deps.Handlers.TwoFactor
		twoFactorLoginHandler := deps.Handlers.TwoFactorLogin

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

			// 2FA 登录验证（无需认证）- 使用带 JWT 功能的处理器
			auth.POST("/2fa/validate", twoFactorLoginHandler.Validate2FAAndLogin)

			// 需要认证的认证相关接口
			authWithAuth := auth.Group("")
			authWithAuth.Use(authMiddleware.RequireAuth())
			{
				authWithAuth.POST("/change-password", authHandler.ChangePassword)
				authWithAuth.GET("/me", authHandler.GetCurrentUser)

				// 2FA 管理接口（需要认证）
				authWithAuth.GET("/2fa/status", twoFactorHandler.Get2FAStatus)
				authWithAuth.POST("/2fa/setup", twoFactorHandler.Setup2FA)
				authWithAuth.POST("/2fa/verify", twoFactorHandler.Verify2FA)
				authWithAuth.POST("/2fa/disable", twoFactorHandler.Disable2FA)
				authWithAuth.POST("/2fa/backup-codes", twoFactorHandler.RegenerateBackupCodes)
			}

			// Google OAuth2 端点
			auth.GET("/google/authorize", oauth2Handler.GoogleAuthorize)
			auth.GET("/google/callback", oauth2Handler.GoogleCallback)                    // Google 重定向使用 GET
			auth.POST("/google/callback", oauth2Handler.GoogleCallback)                   // 前端调用使用 POST
			auth.GET("/google/reauthorize/:account_uid", oauth2Handler.GoogleReauthorize) // 重新授权
			auth.POST("/google/refresh/:account_uid", oauth2Handler.GoogleRefresh)
			auth.POST("/google/revoke/:account_uid", oauth2Handler.GoogleRevoke)

			// Microsoft OAuth2 端点
			auth.GET("/microsoft/authorize", oauth2Handler.MicrosoftAuthorize)
			auth.GET("/microsoft/callback", oauth2Handler.MicrosoftCallback)                    // 修改为 GET
			auth.GET("/microsoft/reauthorize/:account_uid", oauth2Handler.MicrosoftReauthorize) // 重新授权
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
				mail.POST("/mark-read", publicHandler.MarkMailAsRead)
				mail.POST("/send", publicHandler.SendMail)
				mail.GET("/detail", publicHandler.GetMailDetail)
				mail.DELETE("/delete", publicHandler.DeleteMail)
				mail.DELETE("/clear", publicHandler.ClearMailbox)
				mail.GET("/sent", publicHandler.ListSentEmails)
				mail.POST("/import-accounts", publicHandler.BatchImportAccounts)
			}
		}

		// 需要认证的接口（仅允许 JWT）
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		protected.Use(middleware.CSRFMiddleware())
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
				accounts.GET("/filter", accountHandler.ListWithFilter) // 带筛选条件的账户列表
				accounts.GET("/trash", accountHandler.ListDeleted)     // 获取回收站账号
				accounts.GET("/:uid", accountHandler.GetByUID)
				accounts.PUT("/:uid", accountHandler.Update)
				accounts.DELETE("/:uid", accountHandler.Delete)
				accounts.POST("/:uid/restore", accountHandler.Restore)
				accounts.DELETE("/:uid/force", accountHandler.ForceDelete)
				accounts.POST("/:uid/test", accountHandler.TestConnection)
				accounts.POST("/:uid/sync", accountHandler.SyncAccount)
				accounts.POST("/:uid/sync/cancel", accountHandler.CancelSync)       // 取消同步 (Requirements: 5.1)
				accounts.GET("/:uid/sync/progress", accountHandler.GetSyncProgress) // 获取同步进度
				accounts.POST("/:uid/disable", accountHandler.DisableAccount)
				accounts.POST("/:uid/enable", accountHandler.EnableAccount)
				accounts.POST("/:uid/clear-error", accountHandler.ClearSyncError)

				// 批量操作
				accounts.POST("/batch/enable", accountHandler.BatchEnableAccounts)   // 批量启用账户
				accounts.POST("/batch/disable", accountHandler.BatchDisableAccounts) // 批量禁用账户
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

			// Provider 管理接口
			providers := protected.Group("/providers")
			{
				providers.POST("", providerHandler.Create)
				providers.GET("", providerHandler.ListWithPagination)             // 分页列表
				providers.GET("/all", providerHandler.List)                       // 全部列表（无分页）
				providers.GET("/by-domain", providerHandler.FindByDomain)         // 根据域名查找
				providers.GET("/by-email", providerHandler.FindByEmail)           // 根据邮箱查找
				providers.GET("/with-adapters", providerHandler.ListWithAdapters) // 带适配器列表
				providers.GET("/:id", providerHandler.GetByID)
				providers.GET("/:id/adapters", providerHandler.GetWithAdapters) // 获取 Provider 的适配器
				providers.PUT("/:id", providerHandler.UpdateByID)
				providers.DELETE("/:id", providerHandler.DeleteByID)
			}

			// Adapter 管理接口
			adapters := protected.Group("/adapters")
			{
				adapters.GET("", adapterHandler.List)                 // 获取所有适配器
				adapters.GET("/enabled", adapterHandler.ListEnabled)  // 获取启用的适配器
				adapters.GET("/:id", adapterHandler.GetByID)          // 根据 ID 获取
				adapters.GET("/name/:name", adapterHandler.GetByName) // 根据名称获取
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
				emails.POST("/batch-delete", emailHandler.BatchDeleteEmails)
				emails.POST("/:id/toggle-star", emailHandler.ToggleStar)
				emails.POST("/:id/archive", emailHandler.ArchiveEmail)
				emails.DELETE("/:id", emailHandler.DeleteEmail)
				emails.POST("/:id/restore", emailHandler.RestoreEmail)
				emails.DELETE("/:id/permanent", emailHandler.PermanentDeleteEmail)
				emails.POST("/permanent-delete", emailHandler.BatchPermanentDeleteEmails)
				emails.POST("/empty-trash", emailHandler.EmptyTrash)
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

			// 垃圾邮件管理接口
			spam := protected.Group("/spam")
			{
				spam.POST("/mark", spamHandler.MarkAsSpam)         // 标记为垃圾邮件
				spam.POST("/unmark", spamHandler.UnmarkAsSpam)     // 取消垃圾邮件标记
				spam.DELETE("/batch", spamHandler.BatchDeleteSpam) // 批量删除垃圾邮件
				spam.POST("/empty", spamHandler.EmptySpamFolder)   // 清空垃圾箱
				spam.GET("/emails", spamHandler.GetSpamEmails)     // 获取垃圾邮件列表
				spam.GET("/stats", spamHandler.GetSpamStats)       // 获取垃圾邮件统计

				// 贝叶斯分类器接口
				bayesian := spam.Group("/bayesian")
				{
					bayesian.GET("/status", spamHandler.GetBayesianStatus)       // 获取模型状态
					bayesian.POST("/train", spamHandler.TrainBayesianModel)      // 训练模型
					bayesian.POST("/reset", spamHandler.ResetBayesianModel)      // 重置模型
					bayesian.GET("/stats", spamHandler.GetBayesianTrainingStats) // 获取训练统计
				}

				// 规则管理接口
				rules := spam.Group("/rules")
				{
					rules.GET("", spamHandler.GetRules)              // 获取规则列表
					rules.GET("/stats", spamHandler.GetRuleStats)    // 获取规则统计
					rules.POST("/test", spamHandler.TestRule)        // 测试规则
					rules.GET("/:id", spamHandler.GetRule)           // 获取单个规则
					rules.POST("", spamHandler.CreateRule)           // 创建规则
					rules.PUT("/:id", spamHandler.UpdateRule)        // 更新规则
					rules.DELETE("/:id", spamHandler.DeleteRule)     // 删除规则
					rules.PUT("/:id/toggle", spamHandler.ToggleRule) // 切换规则状态
				}
			}

			// 发件人信誉管理接口
			reputation := protected.Group("/reputation")
			{
				reputation.GET("/sender/:email", reputationHandler.GetSenderReputation) // 查询发件人信誉
				reputation.POST("/update", reputationHandler.UpdateReputation)          // 更新信誉评分
				reputation.GET("/stats", reputationHandler.GetReputationStats)          // 信誉统计
				reputation.GET("/list", reputationHandler.ListSenderReputations)        // 发件人信誉列表
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

			// 分组管理接口（需要在 SetupRouter 中注入 groupHandler）
			// 注意：groupHandler 需要在 main.go 中创建并传入

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
					// 异步执行同步任务，避免 HTTP 请求超时
					// 使用 context.Background() 而非 c.Request.Context()
					// 因为 HTTP 请求返回后 c.Request.Context() 会被取消
					go func() {
						if err := syncManager.SyncAccount(deps.AppCtx, accountUID); err != nil {
							routerLog.Error("异步同步失败: account=%s, err=%v", accountUID, err)
						}
					}()
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
				systemSettings.GET("/:category", settingHandler.GetSystemByCategory)
				systemSettings.POST("/:category", settingHandler.BatchSetSystem)
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

// RegisterGroupRoutes 注册分组管理路由
func RegisterGroupRoutes(router *gin.Engine, groupHandler *handler.GroupHandler, deps RouterDeps) {
	authMiddleware := deps.authMiddleware()

	api := router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())

	// 分组管理接口
	groups := protected.Group("/groups")
	{
		groups.POST("", groupHandler.CreateGroup)
		groups.GET("", groupHandler.GetGroups)
		groups.GET("/ungrouped/accounts", groupHandler.GetUngroupedAccounts)
		groups.POST("/batch-assign", groupHandler.BatchAssignAccounts)
		groups.PUT("/reorder", groupHandler.ReorderGroups)
		groups.GET("/:id", groupHandler.GetGroupByID)
		groups.PUT("/:id", groupHandler.UpdateGroup)
		groups.DELETE("/:id", groupHandler.DeleteGroup)
	}

	// 账号分组分配接口
	accounts := protected.Group("/accounts")
	{
		accounts.PUT("/:uid/group", groupHandler.AssignAccountToGroup)
	}
}

// RegisterWebhookReceiverRoutes 注册 Webhook 接收路由
// 这些路由无需认证，由外部邮件服务商调用
func RegisterWebhookReceiverRoutes(router *gin.Engine, webhookReceiverHandler *handler.WebhookReceiverHandler) {
	// Webhook 接收路由组（无需认证）
	// 外部邮件服务商（如 Cloudflare Temp Email）会调用这些端点推送邮件
	webhookReceiver := router.Group("/api/v1/webhook/receive")
	{
		// 通用 webhook 入口，支持所有已注册的 provider 类型
		// POST /api/v1/webhook/receive/:provider_type
		webhookReceiver.POST("/:provider_type", webhookReceiverHandler.HandleWebhook)

		// 获取支持的 provider 类型列表
		// GET /api/v1/webhook/receive/providers
		webhookReceiver.GET("/providers", webhookReceiverHandler.GetSupportedProviders)

		// 获取指定 provider 的 webhook 配置信息
		// GET /api/v1/webhook/receive/info/:provider_type
		webhookReceiver.GET("/info/:provider_type", webhookReceiverHandler.GetWebhookInfo)
	}

	routerLog.Info("Webhook 接收路由已注册")
}

// RegisterLogRoutes 注册日志查询路由
func RegisterLogRoutes(router *gin.Engine, logHandler *handler.LogHandler, deps RouterDeps) {
	authMiddleware := deps.authMiddleware()

	api := router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())

	// 日志管理接口
	logs := protected.Group("/logs")
	{
		logs.GET("", logHandler.GetLogs)              // 获取日志列表
		logs.GET("/files", logHandler.GetLogFiles)    // 获取日志文件列表
		logs.GET("/stats", logHandler.GetLogStats)    // 获取日志统计
		logs.GET("/tail", logHandler.GetLogTail)      // 获取最新日志
		logs.GET("/download", logHandler.DownloadLog) // 下载日志
		logs.POST("/clear", logHandler.ClearLogs)     // 清空日志
	}

	routerLog.Info("日志查询路由已注册")
}

// RegisterSendRoutes 注册邮件发送路由
// Requirements: 1.1, 5.1, 5.2, 5.3, 7.1, 3.1, 3.2
func RegisterSendRoutes(router *gin.Engine, sendHandler *handler.SendHandler, deps RouterDeps) {
	authMiddleware := deps.authMiddleware()

	api := router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())

	// 邮件发送接口
	emails := protected.Group("/emails")
	{
		// 发送邮件
		emails.POST("/send", sendHandler.SendEmail)

		// 回复/转发
		emails.POST("/:id/reply", sendHandler.Reply)
		emails.POST("/:id/reply-all", sendHandler.ReplyAll)
		emails.POST("/:id/forward", sendHandler.Forward)

		// 已发送邮件
		emails.GET("/sent", sendHandler.ListSentEmails)
		emails.GET("/sent/:id", sendHandler.GetSentEmail)
		emails.DELETE("/sent/:id", sendHandler.DeleteSentEmail)

		// 附件上传
		emails.POST("/attachments", sendHandler.UploadAttachment)
	}

	// SMTP 配置接口
	accounts := protected.Group("/accounts")
	{
		accounts.GET("/:uid/smtp", sendHandler.GetSMTPConfig)
		accounts.PUT("/:uid/smtp", sendHandler.UpdateSMTPConfig)
		accounts.POST("/:uid/smtp/test", sendHandler.TestSMTPConnection)
	}

	// SMTP 默认配置
	smtp := protected.Group("/smtp")
	{
		smtp.GET("/defaults", sendHandler.GetDefaultSMTPConfigs)
	}
}

// RegisterTranslationRoutes 注册翻译代理路由
func RegisterTranslationRoutes(router *gin.Engine, translationHandler *handler.TranslationHandler, deps RouterDeps) {
	authMiddleware := deps.authMiddleware()

	api := router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())

	protected.POST("/translate", translationHandler.Translate)

	routerLog.Info("翻译代理路由已注册")
}
