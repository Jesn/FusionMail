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
	"fusionmail/internal/service"
	"fusionmail/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting FusionMail server...")

	// 加载 .env 文件（如果存在）
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables or defaults")
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

	// 创建服务实例
	db := database.GetDB()
	accountRepo := repository.NewAccountRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	ruleRepo := repository.NewRuleRepository(db)
	adapterFactory := adapter.NewFactory()

	// 创建账户服务
	accountService, err := service.NewAccountService(accountRepo, adapterFactory)
	if err != nil {
		log.Fatalf("Failed to create account service: %v", err)
	}

	// 创建邮件服务
	emailService := service.NewEmailService(emailRepo, accountRepo)

	// 创建规则服务
	ruleService := service.NewRuleService(ruleRepo, emailRepo)

	// 创建处理器
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-key-for-testing-only" // 开发环境默认密钥
	}
	authHandler := handler.NewAuthHandler(jwtSecret)
	accountHandler := handler.NewAccountHandler(accountService)
	emailHandler := handler.NewEmailHandler(emailService)
	ruleHandler := handler.NewRuleHandler(ruleService)

	// 创建并启动同步管理器
	syncManager := service.NewSyncManager()
	ctx := context.Background()
	if err := syncManager.Start(ctx); err != nil {
		log.Printf("Failed to start sync manager: %v", err)
	} else {
		log.Println("Sync manager started successfully")
	}

	// 设置 Gin 模式
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由
	router := gin.Default()

	// 配置 CORS（允许前端开发服务器访问）
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 健康检查端点
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "fusionmail",
				"version": "0.1.0",
			})
		})

		// 认证接口
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)
		api.GET("/auth/verify", authHandler.Verify)

		// 同步相关接口
		api.POST("/sync/accounts/:uid", func(c *gin.Context) {
			accountUID := c.Param("uid")
			if err := syncManager.SyncAccount(c.Request.Context(), accountUID); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Sync started"})
		})

		api.POST("/sync/all", func(c *gin.Context) {
			if err := syncManager.SyncAllAccounts(c.Request.Context()); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Sync started for all accounts"})
		})

		api.GET("/sync/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"running": syncManager.IsRunning(),
			})
		})

		// 账户管理接口
		api.POST("/accounts", accountHandler.Create)
		api.GET("/accounts", accountHandler.List)
		api.GET("/accounts/:uid", accountHandler.GetByUID)
		api.PUT("/accounts/:uid", accountHandler.Update)
		api.DELETE("/accounts/:uid", accountHandler.Delete)
		api.POST("/accounts/:uid/test", accountHandler.TestConnection)

		// 邮件管理接口
		api.GET("/emails", emailHandler.GetEmailList)
		api.GET("/emails/search", emailHandler.SearchEmails)
		api.GET("/emails/unread-count", emailHandler.GetUnreadCount)
		api.GET("/emails/stats/:account_uid", emailHandler.GetAccountStats)
		api.GET("/emails/:id", emailHandler.GetEmailByID)
		api.POST("/emails/mark-read", emailHandler.MarkAsRead)
		api.POST("/emails/mark-unread", emailHandler.MarkAsUnread)
		api.POST("/emails/:id/toggle-star", emailHandler.ToggleStar)
		api.POST("/emails/:id/archive", emailHandler.ArchiveEmail)
		api.DELETE("/emails/:id", emailHandler.DeleteEmail)

		// 规则管理接口
		api.POST("/rules", ruleHandler.CreateRule)
		api.GET("/rules", ruleHandler.ListRules)
		api.GET("/rules/:id", ruleHandler.GetRuleByID)
		api.PUT("/rules/:id", ruleHandler.UpdateRule)
		api.DELETE("/rules/:id", ruleHandler.DeleteRule)
		api.POST("/rules/:id/toggle", ruleHandler.ToggleRule)
		api.POST("/rules/:id/test", ruleHandler.TestRule)
	}

	// 静态文件服务（前端）
	staticPath := getStaticPath()
	if _, err := os.Stat(staticPath); err == nil {
		log.Printf("Serving static files from: %s", staticPath)

		// 提供静态资源文件
		router.Static("/assets", filepath.Join(staticPath, "assets"))

		// SPA 路由处理：所有非 API 请求返回 index.html
		router.NoRoute(func(c *gin.Context) {
			// 如果是 API 请求，返回 404
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(404, gin.H{"error": "API endpoint not found"})
				return
			}

			// 否则返回前端 index.html（SPA 路由）
			c.File(filepath.Join(staticPath, "index.html"))
		})
	} else {
		log.Printf("Warning: Static files not found at %s, frontend will not be served", staticPath)
	}

	// 创建 HTTP 服务器
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
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
