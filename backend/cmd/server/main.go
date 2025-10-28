package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"fusionmail/config"
	"fusionmail/pkg/database"
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

	// TODO: 启动 HTTP 服务器
	log.Printf("Server will listen on %s:%s", cfg.Server.Host, cfg.Server.Port)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
