package main

import (
	"context"
	"fmt"
	"log"
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
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/logger"
	redisWrapper "fusionmail/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("Starting FusionMail server...")

	// 加载 .env 文件（如果存在）
	// 使用绝对路径确保加载backend目录下的.env文件
	pwd, _ := os.Getwd()
	envFile := filepath.Join(pwd, ".env")
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("No .env file found at %s, using environment variables or defaults: %v", envFile, err)
	} else {
		log.Printf("Successfully loaded .env file: %s", envFile)
	}

	// 加载配置
	cfg := config.Load()
	log.Printf("Configuration loaded: DB=%s:%s, Server=%s:%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Server.Host, cfg.Server.Port)

	// 初始化数据库连接
	if err := database.Initialize(&cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 自动迁移数据库表结构
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}

	// 添加初始数据（如果需要）
	if err := database.SeedInitialData(); err != nil {
		log.Fatalf("Failed to seed initial data: %v", err)
	}

	log.Println("Database initialization completed successfully")

	// 初始化系统（创建管理员用户）
	initService := service.NewInitService()
	if err := initService.InitializeSystem(); err != nil {
		log.Fatalf("Failed to initialize system: %v", err)
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
	adapterFactory := adapter.NewFactory()

	// 创建加密服务
	cryptoService, err := crypto.NewService(cfg.Security.EncryptionKey)
	if err != nil {
		log.Fatalf("Failed to create crypto service: %v", err)
	}

	// 创建账户服务
	accountService, err := service.NewAccountService(accountRepo, emailRepo, adapterFactory, cryptoService)
	if err != nil {
		log.Fatalf("Failed to create account service: %v", err)
	}

	// 创建加密器
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	// 创建邮件服务
	emailService := service.NewEmailService(emailRepo, accountRepo, adapterFactory, encryptor)

	// 创建规则服务
	ruleService := service.NewRuleService(ruleRepo, emailRepo)

	// 初始化 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试 Redis 连接
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Redis connection established successfully")
	}

	// 创建 Redis 客户端包装器
	redisClientWrapper := redisWrapper.NewClientWrapper(redisClient)

	// 创建 Webhook 服务
	logger := logger.New()
	webhookService := service.NewWebhookService(webhookRepo, webhookLogRepo, logger)

	// 创建 OAuth2 服务
	oauth2Service := service.NewOAuth2Service(cfg, accountRepo, emailRepo, cryptoService, redisClientWrapper, logger, oauth2ClientRepo, providerRepo)

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
		logger,
	)

	// 创建 OAuth2 客户端服务
	oauth2ClientService := service.NewOAuth2ClientService(oauth2ClientRepo)

	// 创建 Provider 服务
	providerService := service.NewProviderService(providerRepo)

	// 创建认证服务（用于 API Key 管理）
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, apiKeyRepo, cfg.JWT.Secret, time.Duration(cfg.JWT.Expiry)*time.Hour)

	// 创建处理器
	jwtSecret := cfg.JWT.Secret
	// 使用新的认证处理器
	authHandler := handler.NewDBAuthHandler(jwtSecret)
	accountHandler := handler.NewAccountHandler(accountService, oauth2Service)
	emailHandler := handler.NewEmailHandler(emailService)
	ruleHandler := handler.NewRuleHandler(ruleService)
	webhookHandler := handler.NewWebhookHandler(webhookService, webhookLogRepo)
	systemHandler := handler.NewSystemHandler(systemService)
	oauth2Handler := handler.NewOAuth2Handler(oauth2Service)
	apiKeyHandler := handler.NewAPIKeyHandler(authService)
	publicHandler := handler.NewPublicHandler(emailService, accountService)
	oauth2ClientHandler := handler.NewOAuth2ClientHandler(oauth2ClientService, providerService) // 新增 OAuth2Client 处理器
	providerHandler := handler.NewProviderHandler(providerService)                              // 新增 Provider 处理器
	devSyncHandler := handler.NewDevSyncHandler()                                               // 新增开发环境同步处理器

	// 创建 Setting 服务和处理器
	settingService := service.NewSettingService(settingRepo, redisClient, encryptor)
	settingHandler := handler.NewSettingHandler(settingService)

	// 设置 Gin 模式
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建并启动同步管理器
	syncManager := service.NewSyncManager(cryptoService)
	ctx := context.Background()
	if err := syncManager.Start(ctx); err != nil {
		log.Printf("Failed to start sync manager: %v", err)
	} else {
		log.Println("Sync manager started successfully")
	}

	// 创建并启动清理服务
	cleanupService := service.NewCleanupService(accountService, settingService)
	if err := cleanupService.Start(ctx); err != nil {
		log.Printf("Failed to start cleanup service: %v", err)
	} else {
		log.Println("Cleanup service started successfully")
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
		devSyncHandler,      // 新增开发环境同步处理器
		syncManager,
		redisClient,
		jwtSecret,
		apiKeyRepo,
		cfg.RateLimit.Enabled,
		cfg.RateLimit.SiteDefault,
		cfg.RateLimit.PublicDefault,
	)

	// 静态文件服务（前端）
	staticPath := getStaticPath()
	if _, err := os.Stat(staticPath); err == nil {
		log.Printf("Serving static files from: %s", staticPath)

		// 提供静态资源文件
		ginRouter.Static("/assets", filepath.Join(staticPath, "assets"))

		// SPA 路由处理：所有非 API 请求返回 index.html
		ginRouter.NoRoute(func(c *gin.Context) {
			// 如果是 API 请求，返回 404
			// 精确匹配 /api/ 开头的路径（避免 /api-docs 等前端路由被误判）
			if len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/api/" {
				c.JSON(404, gin.H{"error": "API endpoint not found"})
				return
			}

			// 否则返回前端 index.html（SPA 路由）
			c.File(filepath.Join(staticPath, "index.html"))
		})
	} else {
		log.Printf("Warning: Static files not found at %s, frontend will not be served", staticPath)
	}

	// 创建 HTTP 服务器（SSE 需要更长的超时时间）
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        ginRouter,
		ReadTimeout:    90 * time.Second, // SSE 长连接需要更长的读超时
		WriteTimeout:   90 * time.Second, // SSE 长连接需要更长的写超时
		MaxHeaderBytes: 1 << 20,          // 1 MB
	}

	// 在 goroutine 中启动服务器
	go func() {
		log.Printf("Server listening on %s", addr)
		log.Printf("API endpoint: http://%s/api/v1", addr)
		log.Printf("Frontend: http://%s", addr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 停止清理服务
	cleanupService.Stop()

	// 停止同步管理器
	if err := syncManager.Stop(); err != nil {
		log.Printf("Failed to stop sync manager: %v", err)
	}

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
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
