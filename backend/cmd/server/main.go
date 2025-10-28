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
	"fusionmail/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting FusionMail server...")

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

		// TODO: 添加其他 API 路由
		// api.GET("/emails", handlers.GetEmails)
		// api.GET("/accounts", handlers.GetAccounts)
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
