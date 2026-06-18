package main

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"fusionmail/config"
	"fusionmail/internal/model"
	"fusionmail/internal/seed"
	"fusionmail/internal/service"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/goroutine"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/runtimeenv"

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

func getEnvBool(key string, defaultValue bool) bool {
	return runtimeenv.EnvBool(key, defaultValue)
}

func currentGinMode() string {
	mode := runtimeenv.CurrentGinMode()
	rawMode := strings.TrimSpace(os.Getenv("GIN_MODE"))
	if rawMode != "" && mode == gin.ReleaseMode {
		normalized := strings.ToLower(rawMode)
		if normalized != gin.ReleaseMode {
			log.Warn("GIN_MODE=%q 无效，按 release 模式处理", rawMode)
		}
	}
	return mode
}

func shouldRunStartupMigrate() bool {
	if currentGinMode() == gin.DebugMode {
		return getEnvBool("ENABLE_AUTO_MIGRATE", true)
	}
	return getEnvBool("ENABLE_AUTO_MIGRATE", false)
}

func shouldRunStartupSeed() bool {
	if currentGinMode() == gin.DebugMode {
		return getEnvBool("ENABLE_STARTUP_SEED", true)
	}
	return getEnvBool("ENABLE_STARTUP_SEED", false)
}

func validateProductionSecrets(cfg *config.Config) error {
	if currentGinMode() != gin.ReleaseMode {
		return nil
	}
	if crypto.IsDefaultEncryptionKey(cfg.Security.EncryptionKey) || !hasExactByteLength(cfg.Security.EncryptionKey, 32) {
		return fmt.Errorf("ENCRYPTION_KEY 未配置、仍为默认值或不是 32 字节，release 模式必须设置 32 字节强随机密钥")
	}
	if isDefaultJWTSecret(cfg.JWT.Secret) || !hasMinimumByteLength(cfg.JWT.Secret, 32) {
		return fmt.Errorf("JWT_SECRET 未配置、仍为默认值或长度不足 32 字节，release 模式必须设置强随机密钥")
	}
	return nil
}

func isDefaultJWTSecret(secret string) bool {
	trimmed := strings.TrimSpace(secret)
	return trimmed == "" || trimmed == config.DefaultJWTSecret
}

func hasExactByteLength(value string, length int) bool {
	return value == strings.TrimSpace(value) && len([]byte(value)) == length
}

func hasMinimumByteLength(value string, length int) bool {
	return value == strings.TrimSpace(value) && len([]byte(value)) >= length
}

func ensureStartupSchemaReady() error {
	if database.GetDB() == nil {
		return fmt.Errorf("database is not initialized")
	}

	requiredTables := []struct {
		name  string
		model interface{}
	}{
		{name: "users", model: &model.User{}},
		{name: "accounts", model: &model.EmailAccount{}},
		{name: "emails", model: &model.Email{}},
		{name: "email_attachments", model: &model.EmailAttachment{}},
		{name: "email_labels", model: &model.EmailLabel{}},
		{name: "email_label_relations", model: &model.EmailLabelRelation{}},
		{name: "email_rules", model: &model.EmailRule{}},
		{name: "webhooks", model: &model.Webhook{}},
		{name: "webhook_logs", model: &model.WebhookLog{}},
		{name: "sync_logs", model: &model.SyncLog{}},
		{name: "api_keys", model: &model.APIKey{}},
		{name: "settings", model: &model.Setting{}},
		{name: "providers", model: &model.Provider{}},
		{name: "email_oauth2_tokens", model: &model.OAuth2Client{}},
		{name: "account_groups", model: &model.AccountGroup{}},
		{name: "email_lists", model: &model.EmailList{}},
		{name: "sender_reputations", model: &model.SenderReputation{}},
		{name: "spam_rules", model: &model.SpamRule{}},
		{name: "bayesian_trainings", model: &model.BayesianTraining{}},
		{name: "spam_detection_logs", model: &model.SpamDetectionLog{}},
	}

	for _, table := range requiredTables {
		if !database.GetDB().Migrator().HasTable(table.model) {
			return fmt.Errorf("数据库结构不完整，缺少表 %s，请先执行 go run cmd/migrate/main.go -action=up 或显式设置 ENABLE_AUTO_MIGRATE=true", table.name)
		}
	}

	if err := ensureTwoFactorSecretColumnReady(); err != nil {
		return err
	}
	return ensureUserSessionVersionColumnReady()
}

func ensureTwoFactorSecretColumnReady() error {
	var column struct {
		DataType               string
		CharacterMaximumLength stdsql.NullInt64 `gorm:"column:character_maximum_length"`
	}

	err := database.GetDB().Raw(`
		SELECT data_type, character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = ?
		  AND column_name = ?
	`, "users", "two_factor_secret").Scan(&column).Error
	if err != nil {
		return fmt.Errorf("检查 users.two_factor_secret 字段失败: %w", err)
	}
	if column.DataType == "" {
		return fmt.Errorf("数据库结构不完整，缺少 users.two_factor_secret 字段，请先执行迁移 035")
	}
	if column.DataType == "text" {
		return nil
	}
	if column.DataType == "character varying" && column.CharacterMaximumLength.Valid && column.CharacterMaximumLength.Int64 >= 128 {
		return nil
	}
	return fmt.Errorf("users.two_factor_secret 字段类型为 %s，请先执行迁移 035_harden_2fa_secret_storage.sql", column.DataType)
}

func ensureUserSessionVersionColumnReady() error {
	if !database.GetDB().Migrator().HasColumn(&model.User{}, "session_version") {
		return fmt.Errorf("数据库结构不完整，缺少 users.session_version 字段，请先执行迁移 036_add_user_session_version.sql")
	}
	return nil
}

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

	gin.SetMode(currentGinMode())

	// 加载配置
	cfg := config.Load()
	if err := validateProductionSecrets(cfg); err != nil {
		log.Fatal("安全配置校验失败: %v", err)
	}
	log.Info("配置已加载: DB=%s:%s, Server=%s:%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Server.Host, cfg.Server.Port)

	// 初始化数据库连接
	if err := database.Initialize(&cfg.Database); err != nil {
		log.Fatal("数据库初始化失败: %v", err)
	}
	defer database.Close()

	// 自动迁移数据库表结构
	if shouldRunStartupMigrate() {
		if err := database.AutoMigrate(); err != nil {
			log.Fatal("数据库迁移失败: %v", err)
		}
	} else {
		log.Info("已跳过启动时自动迁移，请使用显式迁移命令执行数据库变更")
	}

	if !shouldRunStartupMigrate() {
		if err := ensureStartupSchemaReady(); err != nil {
			log.Fatal("启动前数据库校验失败: %v", err)
		}
	}

	initService := service.NewInitService()
	if err := initService.NormalizeTwoFactorStorage(); err != nil {
		log.Fatal("2FA 敏感数据升级失败: %v", err)
	}

	// 添加初始数据（如果需要）
	if shouldRunStartupSeed() {
		if err := seed.SeedInitialData(database.GetDB()); err != nil {
			log.Fatal("初始数据添加失败: %v", err)
		}
	} else {
		log.Info("已跳过启动时种子初始化，请在需要时显式执行")
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
	if err := initService.InitializeSystem(); err != nil {
		log.Fatal("系统初始化失败: %v", err)
	}

	logDir := filepath.Join(pwd, "..", "logs")
	app, err := buildAppContainer(cfg, database.GetDB(), logDir)
	if err != nil {
		log.Fatal("应用容器创建失败: %v", err)
	}
	ginRouter := app.Router

	ctx := context.Background()
	if err := app.Runtime.SyncManager.Start(ctx); err != nil {
		log.Warn("同步管理器启动失败: %v", err)
	} else {
		log.Info("同步管理器启动成功")
	}
	if err := app.Runtime.CleanupService.Start(ctx); err != nil {
		log.Warn("清理服务启动失败: %v", err)
	} else {
		log.Info("清理服务启动成功")
	}

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
	if currentGinMode() == gin.DebugMode {
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
		WriteTimeout:   90 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
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
	app.Runtime.CleanupService.Stop()

	// 停止同步管理器
	if err := app.Runtime.SyncManager.Stop(); err != nil {
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
