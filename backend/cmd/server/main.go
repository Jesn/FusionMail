package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"fusionmail/config"
	"fusionmail/internal/adapter"
	"fusionmail/internal/handler"
	"fusionmail/internal/repository"
	"fusionmail/internal/router"
	"fusionmail/internal/service"
	"fusionmail/internal/service/spam"
	"fusionmail/internal/webhook"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/goroutine"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/oauth2config"
	redisWrapper "fusionmail/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "fusionmail/docs" // 导入 Swagger 文档

	// 导入 WebAPI 适配器包以触发 init() 注册
	_ "fusionmail/internal/adapter/webapi/cloudflare"
	_ "fusionmail/internal/adapter/webapi/cloudmail"
	_ "fusionmail/internal/adapter/webapi/custom"
)

// 模块日志记录器
var log = logger.NewWithModule("Main")

func main() {
	log.Info("启动 FusionMail 服务器...")

	// 加载 .env 文件（如果存在）
	// 使用绝对路径确保加载backend目录下的.env文件
	pwd, _ := os.Getwd()
	envFile := filepath.Join(pwd, ".env")
	if err := godotenv.Load(envFile); err != nil {
		log.Debug("未找到 .env 文件: %s, 使用环境变量或默认值", envFile)
	} else {
		log.Info("已加载 .env 文件: %s", envFile)
	}

	// 加载配置
	cfg := config.Load()
	log.Info("配置已加载: DB=%s:%s, Server=%s:%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Server.Host, cfg.Server.Port)

	// 初始化数据库连接
	if err := database.Initialize(&cfg.Database); err != nil {
		log.Fatal("数据库初始化失败: %v", err)
	}
	defer database.Close()

	// 自动迁移数据库表结构
	if err := database.AutoMigrate(); err != nil {
		log.Fatal("数据库迁移失败: %v", err)
	}

	// 添加初始数据（如果需要）
	if err := database.SeedInitialData(); err != nil {
		log.Fatal("初始数据添加失败: %v", err)
	}

	log.Info("数据库初始化完成")

	// 启动 Goroutine 监控器
	grMonitor := goroutine.NewMonitor(&goroutine.MonitorConfig{
		CheckInterval:           30 * time.Second,
		WarningThreshold:        500,  // 告警阈值
		CriticalThreshold:       2000, // 严重告警阈值
		EnableLeakDetection:     true,
		LeakDetectionWindowSize: 10,
		LeakGrowthRateThreshold: 0.5,
	})
	grMonitor.SetWarningCallback(func(count int) {
		log.Warn("Goroutine 数量告警: count=%d", count)
	})
	grMonitor.SetCriticalCallback(func(count int) {
		log.Error("Goroutine 数量严重超标: count=%d", count)
	})
	grMonitor.SetLeakCallback(func(count int, growthRate float64) {
		log.Warn("疑似 Goroutine 泄露: count=%d, growthRate=%.2f%%", count, growthRate*100)
	})
	if err := grMonitor.Start(context.Background()); err != nil {
		log.Warn("Goroutine 监控器启动失败: %v", err)
	}
	defer grMonitor.Stop()

	// 启动数据库连接池监控器
	sqlDB, err := database.GetDB().DB()
	if err == nil {
		dbMonitor := goroutine.NewDBPoolMonitor(sqlDB, &goroutine.DBPoolMonitorConfig{
			CheckInterval:                30 * time.Second,
			UsageWarningThreshold:        0.7,
			UsageCriticalThreshold:       0.9,
			WaitDurationWarningThreshold: 100 * time.Millisecond,
		})
		dbMonitor.SetWarningCallback(func(stats goroutine.DBPoolStats, message string) {
			log.Warn("数据库连接池告警: %s, inUse=%d, max=%d", message, stats.InUse, stats.MaxOpenConnections)
		})
		dbMonitor.SetCriticalCallback(func(stats goroutine.DBPoolStats, message string) {
			log.Error("数据库连接池严重告警: %s, inUse=%d, max=%d", message, stats.InUse, stats.MaxOpenConnections)
		})
		if err := dbMonitor.Start(context.Background()); err != nil {
			log.Warn("数据库连接池监控器启动失败: %v", err)
		}
		defer dbMonitor.Stop()
	}

	// 初始化系统（创建管理员用户）
	initService := service.NewInitService()
	if err := initService.InitializeSystem(); err != nil {
		log.Fatal("系统初始化失败: %v", err)
	}

	// 创建服务实例
	db := database.GetDB()
	accountRepo := repository.NewAccountRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	ruleRepo := repository.NewRuleRepository(db)
	webhookRepo := repository.NewWebhookRepository(db)
	webhookLogRepo := repository.NewWebhookLogRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	settingRepo := repository.NewSettingRepository(db)           // 新增 Setting Repository
	providerRepo := repository.NewProviderRepository(db)         // 新增 Provider Repository
	oauth2ClientRepo := repository.NewOAuth2ClientRepository(db) // 新增 OAuth2Client Repository
	adapterRepo := repository.NewAdapterRepository(db)           // 新增 Adapter Repository
	adapterFactory := adapter.NewFactory()

	// 创建加密服务
	cryptoService, err := crypto.NewService(cfg.Security.EncryptionKey)
	if err != nil {
		log.Fatal("加密服务创建失败: %v", err)
	}

	// 创建账户服务
	accountService, err := service.NewAccountService(accountRepo, emailRepo, providerRepo, adapterFactory, cryptoService)
	if err != nil {
		log.Fatal("账户服务创建失败: %v", err)
	}

	// 创建加密器
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		log.Fatal("加密器创建失败: %v", err)
	}

	// 创建邮件服务
	emailService := service.NewEmailService(emailRepo, accountRepo, adapterFactory, encryptor)
	translationService := service.NewTranslationService(service.TranslationServiceConfig{
		APIURL:  cfg.Translation.APIURL,
		Token:   cfg.Translation.Token,
		Timeout: time.Duration(cfg.Translation.TimeoutSeconds) * time.Second,
	})

	// 创建规则服务
	ruleService := service.NewRuleService(ruleRepo, emailRepo)

	// 初始化 Redis 客户端（使用全局初始化，支持 TLS）
	// 这会设置 pkgredis.Client 全局变量，供 SyncManager 等组件使用
	if err := redisWrapper.Initialize(&cfg.Redis); err != nil {
		log.Warn("Redis 连接失败: %v", err)
	}

	// 获取 Redis 客户端实例
	redisClient := redisWrapper.GetClient()

	// 创建 Redis 客户端包装器
	redisClientWrapper := redisWrapper.NewClientWrapper(redisClient)

	// 创建 Webhook 服务
	webhookLogger := logger.New()
	webhookService := service.NewWebhookService(webhookRepo, webhookLogRepo, webhookLogger)

	// 创建 OAuth2 服务
	oauth2Service := service.NewOAuth2Service(cfg, accountRepo, emailRepo, cryptoService, redisClientWrapper, webhookLogger, oauth2ClientRepo, providerRepo, adapterRepo)

	// 创建系统管理服务
	systemService := service.NewSystemService(
		database.GetDB(),
		redisClient,
		accountRepo,
		emailRepo,
		ruleRepo,
		webhookRepo,
		syncLogRepo,
		providerRepo, // 新增 Provider Repository
		webhookLogger,
	)

	// 创建 OAuth2 客户端服务
	oauth2ClientService := service.NewOAuth2ClientService(oauth2ClientRepo, cryptoService)

	// 创建 Provider 服务
	providerService := service.NewProviderService(providerRepo)

	// 创建 WebAPI Provider 服务和处理器
	webAPIProviderService := service.NewWebAPIProviderService(
		accountRepo,
		providerRepo,
		adapterRepo,
		emailRepo,
		syncLogRepo,
		cryptoService,
	)
	webAPIProviderHandler := handler.NewWebAPIProviderHandler(webAPIProviderService)
	webAPIServicesHandler := handler.NewWebAPIServicesHandler()
	// 创建 Adapter 服务
	adapterService := service.NewAdapterService(adapterRepo)

	// 创建认证服务（用于 API Key 管理）
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, apiKeyRepo, cfg.JWT.Secret, time.Duration(cfg.JWT.Expiry)*time.Hour)

	// 创建处理器
	jwtSecret := cfg.JWT.Secret
	// 使用新的认证处理器
	authHandler := handler.NewDBAuthHandler(jwtSecret, cfg.Security.CookieSecure)
	// accountHandler 将在 syncManager 创建后初始化
	var accountHandler *handler.AccountHandler
	// publicHandler 需要同步服务，将在 syncManager 创建后初始化
	var publicHandler *handler.PublicHandler
	emailHandler := handler.NewEmailHandler(emailService)
	translationHandler := handler.NewTranslationHandler(translationService)
	ruleHandler := handler.NewRuleHandler(ruleService)
	webhookHandler := handler.NewWebhookHandler(webhookService, webhookLogRepo)
	systemHandler := handler.NewSystemHandler(systemService)
	oauth2Handler := handler.NewOAuth2Handler(oauth2Service)
	apiKeyHandler := handler.NewAPIKeyHandler(authService)
	oauth2ClientHandler := handler.NewOAuth2ClientHandler(oauth2ClientService, providerService) // 新增 OAuth2Client 处理器
	providerHandler := handler.NewProviderHandler(providerService)                              // 新增 Provider 处理器
	adapterHandler := handler.NewAdapterHandler(adapterService)                                 // 新增 Adapter 处理器
	devSyncHandler := handler.NewDevSyncHandler()                                               // 新增开发环境同步处理器

	// 创建 Setting 服务和处理器
	settingService := service.NewSettingService(settingRepo, redisClient, encryptor)
	settingHandler := handler.NewSettingHandler(settingService)

	// 创建分组服务和处理器
	groupRepo := repository.NewGroupRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)
	groupHandler := handler.NewGroupHandler(groupService)

	// 创建白名单/黑名单服务和处理器
	emailListRepo := repository.NewEmailListRepository(db)
	whitelistChecker := spam.NewWhitelistChecker(emailListRepo, redisClient)
	emailListService := spam.NewEmailListService(emailListRepo, whitelistChecker)
	emailListHandler := handler.NewEmailListHandler(emailListService)

	// 创建垃圾邮件服务和处理器
	senderReputationRepo := repository.NewSenderReputationRepository(db)
	bayesianTrainingRepo := repository.NewBayesianTrainingRepository(db)
	spamRuleRepo := repository.NewSpamRuleRepository(db)
	reputationManager := spam.NewReputationManager(senderReputationRepo, redisClient)
	bayesianClassifier := spam.NewBayesianClassifier(bayesianTrainingRepo)
	spamService := service.NewSpamService(emailRepo, spamRuleRepo, reputationManager, bayesianClassifier)
	spamHandler := handler.NewSpamHandler(spamService)

	// 创建发件人信誉处理器
	reputationHandler := handler.NewReputationHandler(reputationManager, senderReputationRepo)

	// 创建邮件发送服务和处理器 (Requirements: 1.1, 5.1, 5.2, 5.3, 7.1, 3.1)
	sentEmailRepo := repository.NewSentEmailRepository(db)
	senderFactory, err := adapter.NewSenderFactory(cfg.Security.EncryptionKey)
	if err != nil {
		log.Fatal("发送器工厂创建失败: %v", err)
	}
	// 创建 OAuth2 配置提供者（用于发送邮件时获取凭证）
	oauth2ConfigProvider := oauth2config.NewProvider(oauth2ClientRepo, providerRepo, cryptoService, logger.NewWithModule("OAuth2Config"))
	sendService := service.NewSendService(senderFactory, accountRepo, sentEmailRepo, emailRepo, cryptoService, oauth2ConfigProvider, logger.NewWithModule("SendService"))
	sentEmailService := service.NewSentEmailService(sentEmailRepo)
	smtpConfigService, err := service.NewSMTPConfigService(accountRepo, cfg.Security.EncryptionKey)
	if err != nil {
		log.Fatal("SMTP 配置服务创建失败: %v", err)
	}
	sendHandler := handler.NewSendHandler(sendService, sentEmailService, smtpConfigService)

	// 创建垃圾邮件检测器（用于同步时自动检测）
	spamDetectionLogRepo := repository.NewSpamDetectionLogRepository(db)
	rblChecker := spam.NewRBLChecker(redisClient)
	behaviorAnalyzer := spam.NewBehaviorAnalyzer(redisClient)
	surblChecker := spam.NewSURBLChecker(redisClient)
	preFilter := spam.NewPreFilter(rblChecker, behaviorAnalyzer)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, redisClient, surblChecker)
	spamDetector := spam.NewSpamDetector(whitelistChecker, preFilter, ruleEngine, reputationManager, bayesianClassifier, spamDetectionLogRepo)

	// 设置 Gin 模式
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建并启动同步管理器
	syncManager := service.NewSyncManager(cryptoService, spamDetector)
	ctx := context.Background()
	if err := syncManager.Start(ctx); err != nil {
		log.Warn("同步管理器启动失败: %v", err)
	} else {
		log.Info("同步管理器启动成功")
	}

	// 创建 accountHandler（需要 syncService 用于取消同步和获取进度）
	accountHandler = handler.NewAccountHandler(accountService, oauth2Service, syncManager.GetSyncService())
	publicHandler = handler.NewPublicHandler(emailService, accountService, syncManager.GetSyncService())

	// 创建已删除邮件标识仓库
	deletedKeyRepo := repository.NewDeletedEmailKeyRepository(db)

	// 创建并启动清理服务
	cleanupService := service.NewCleanupService(
		accountService,
		settingService,
		emailRepo,
		syncLogRepo,
		webhookLogRepo,
		spamDetectionLogRepo,
		deletedKeyRepo,
	)
	if err := cleanupService.Start(ctx); err != nil {
		log.Warn("清理服务启动失败: %v", err)
	} else {
		log.Info("清理服务启动成功")
	}

	// 使用新的路由配置模块
	ginRouter := router.SetupRouter(
		authHandler,
		accountHandler,
		emailHandler,
		ruleHandler,
		webhookHandler,
		systemHandler,
		oauth2Handler,
		apiKeyHandler,
		publicHandler,
		settingHandler,      // 新增 Setting 处理器
		oauth2ClientHandler, // 新增 OAuth2Client 处理器
		providerHandler,     // 新增 Provider 处理器
		adapterHandler,      // 新增 Adapter 处理器
		devSyncHandler,      // 新增开发环境同步处理器
		emailListHandler,    // 新增白名单/黑名单处理器
		spamHandler,         // 新增垃圾邮件处理器
		reputationHandler,   // 新增发件人信誉处理器
		syncManager,
		redisClient,
		jwtSecret,
		apiKeyRepo,
		cfg.RateLimit.Enabled,
		cfg.RateLimit.SiteDefault,
		cfg.RateLimit.PublicDefault,
		false, // 不在 SetupRouter 中注册 Swagger（改为在 main.go 中注册）
	)

	// 注册分组管理路由
	router.RegisterGroupRoutes(ginRouter, groupHandler, jwtSecret)
	log.Info("分组管理路由已注册")

	// 注册邮件发送路由 (Requirements: 1.1, 5.1, 5.2, 5.3, 7.1, 3.1, 3.2)
	router.RegisterSendRoutes(ginRouter, sendHandler, jwtSecret)
	log.Info("邮件发送路由已注册")
	router.RegisterTranslationRoutes(ginRouter, translationHandler, jwtSecret)

	// 注册 WebAPI Provider 路由
	router.RegisterWebAPIRoutes(ginRouter, webAPIProviderHandler, webAPIServicesHandler, jwtSecret)
	log.Info("WebAPI Provider 路由已注册")

	// 初始化 Webhook 适配器注册表
	webhookRegistry := webhook.NewAdapterRegistry()
	// 注册 Cloudflare Temp Email 适配器
	webhookRegistry.Register(webhook.NewCloudflareAdapter(webhookLogger))
	log.Info("Webhook 适配器已注册: %v", webhookRegistry.List())

	// 创建 Webhook 接收服务和处理器
	webhookReceiverService := service.NewWebhookReceiverService(accountRepo, emailRepo, providerRepo, cryptoService, webhookLogger)
	webhookReceiverHandler := handler.NewWebhookReceiverHandler(webhookRegistry, webhookReceiverService, webhookLogger)

	// 注册 Webhook 接收路由（无需认证，由外部服务商调用）
	router.RegisterWebhookReceiverRoutes(ginRouter, webhookReceiverHandler)
	log.Info("Webhook 接收路由已注册")

	// 注册日志查询路由
	// 日志目录：项目根目录下的 logs 文件夹
	// pwd 是 backend 目录，日志在项目根目录的 logs 文件夹
	logDir := filepath.Join(pwd, "..", "logs")
	logHandler := handler.NewLogHandler(logDir)
	router.RegisterLogRoutes(ginRouter, logHandler, jwtSecret)
	log.Info("日志查询路由已注册，日志目录: %s", logDir)

	// Swagger 文档路由（必须在静态文件服务之前注册）
	log.Debug("Swagger.Enabled = %v", cfg.Swagger.Enabled)
	if cfg.Swagger.Enabled {
		log.Debug("注册 Swagger 路由...")
		// 使用默认配置，不指定 URL
		ginRouter.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		log.Info("Swagger 路由已注册: /swagger/*any")
	} else {
		log.Debug("Swagger 已禁用 (SWAGGER_ENABLED=false)")
	}

	// pprof 路由（仅开发环境启用）
	if os.Getenv("GIN_MODE") != "release" {
		goroutine.RegisterPprofRoutes(ginRouter, "/debug/pprof")
		log.Info("pprof 路由已注册: /debug/pprof/*")
	}

	// 静态文件服务（前端）
	staticPath := getStaticPath()
	if _, err := os.Stat(staticPath); err == nil {
		log.Info("静态文件目录: %s", staticPath)

		// 提供静态资源文件
		ginRouter.Static("/assets", filepath.Join(staticPath, "assets"))

		// SPA 路由处理：所有非 API 请求返回 index.html
		ginRouter.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			// 如果是 API 请求，返回 404
			// 精确匹配 /api/ 开头的路径（避免 /api-docs 等前端路由被误判）
			if len(path) >= 5 && path[:5] == "/api/" {
				c.JSON(404, gin.H{"error": "API endpoint not found"})
				return
			}

			// 如果是 Swagger 文档请求，返回 404（Swagger 路由应该已经处理）
			if len(path) >= 9 && path[:9] == "/swagger/" {
				c.JSON(404, gin.H{"error": "Swagger documentation not enabled or not found"})
				return
			}

			// 检查是否是静态文件请求（如 logo.png, favicon.ico 等）
			// 如果文件存在，直接返回文件
			requestedFile := filepath.Join(staticPath, path)
			if info, err := os.Stat(requestedFile); err == nil && !info.IsDir() {
				c.File(requestedFile)
				return
			}

			// 否则返回前端 index.html（SPA 路由）
			c.File(filepath.Join(staticPath, "index.html"))
		})
	} else {
		log.Warn("静态文件目录不存在: %s, 前端将不可用", staticPath)
	}

	// 创建 HTTP 服务器（SSE 需要更长的超时时间）
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	// 日志输出 Swagger 状态
	if cfg.Swagger.Enabled {
		log.Info("Swagger 文档地址: http://%s/swagger/index.html", addr)
	} else {
		log.Debug("Swagger 文档已禁用 (设置 SWAGGER_ENABLED=true 启用)")
	}
	srv := &http.Server{
		Addr:           addr,
		Handler:        ginRouter,
		ReadTimeout:    90 * time.Second, // SSE 长连接需要更长的读超时
		WriteTimeout:   90 * time.Second, // SSE 长连接需要更长的写超时
		MaxHeaderBytes: 1 << 20,          // 1 MB
	}

	// 在 goroutine 中启动服务器
	go func() {
		log.Info("服务器监听地址: %s", addr)
		log.Info("API 端点: http://%s/api/v1", addr)
		log.Info("前端地址: http://%s", addr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在关闭服务器...")

	// 停止清理服务
	cleanupService.Stop()

	// 停止同步管理器
	if err := syncManager.Stop(); err != nil {
		log.Warn("同步管理器停止失败: %v", err)
	}

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Warn("服务器强制关闭: %v", err)
	}

	log.Info("服务器已退出")
}

// getStaticPath 获取静态文件路径
func getStaticPath() string {
	// 优先使用环境变量
	if path := os.Getenv("STATIC_PATH"); path != "" {
		return path
	}

	// 检查常见路径
	paths := []string{
		"./static",         // Docker 容器中
		"../frontend/dist", // 开发环境
		"./frontend/dist",  // 开发环境（从根目录运行）
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "./static"
}
