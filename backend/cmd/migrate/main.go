package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"fusionmail/config"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 定义命令行参数
	action := flag.String("action", "up", "Migration action: up (migrate), status (check), or migrate-settings (migrate env vars)")
	flag.Parse()

	log.Println("FusionMail Database Migration Tool")
	log.Printf("Action: %s", *action)

	// 加载配置
	cfg := config.Load()
	log.Printf("Database: %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 初始化数据库连接
	if err := database.Initialize(&cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 执行迁移操作
	switch *action {
	case "up":
		log.Println("Running database migrations...")
		if err := database.AutoMigrate(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}

		log.Println("Seeding initial data...")
		if err := database.SeedInitialData(); err != nil {
			log.Fatalf("Seeding failed: %v", err)
		}

		log.Println("Migration completed successfully!")

	case "migrate-settings":
		log.Println("Starting environment variable migration to settings table...")
		if err := migrateEnvironmentToSettings(cfg); err != nil {
			log.Fatalf("Settings migration failed: %v", err)
		}
		log.Println("Settings migration completed successfully!")

	case "status":
		log.Println("Checking database status...")
		db := database.GetDB()

		// 检查数据库连接
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("Failed to get database instance: %v", err)
		}

		if err := sqlDB.Ping(); err != nil {
			log.Fatalf("Database connection failed: %v", err)
		}

		log.Println("Database connection: OK")

		// 检查表是否存在
		tables := []string{
			"users", "accounts", "emails", "email_attachments",
			"email_labels", "email_label_relations", "email_rules",
			"webhooks", "webhook_logs", "sync_logs", "api_keys", "settings",
		}

		for _, table := range tables {
			var exists bool
			err := db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)", table).Scan(&exists).Error
			if err != nil {
				log.Printf("Table %s: ERROR (%v)", table, err)
			} else if exists {
				log.Printf("Table %s: EXISTS", table)
			} else {
				log.Printf("Table %s: NOT FOUND", table)
			}
		}

	default:
		log.Fatalf("Unknown action: %s (use 'up', 'status', or 'migrate-settings')", *action)
	}

	os.Exit(0)
}

// migrateEnvironmentToSettings 将环境变量迁移到Setting表
func migrateEnvironmentToSettings(cfg *config.Config) error {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, using environment variables")
	}

	// 创建Repository和Service
	db := database.GetDB()
	settingRepo := repository.NewSettingRepository(db)

	// 初始化加密器
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	// 初始化Redis（用于缓存）
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// 验证Redis连接
	ctx := database.GetDB().Statement.Context
	if ctx == nil {
		ctx = database.GetDB().WithContext(nil).Statement.Context
	}
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	// 定义迁移映射（阶段1：UI配置）
	stage1 := []struct {
		envVar    string
		category  string
		key       string
		sensitive bool
		isPublic  bool
	}{
		// UI配置
		{"DEFAULT_THEME", "ui", "theme", false, true},
		{"DEFAULT_LANGUAGE", "ui", "language", false, true},
		{"EMAIL_PAGE_SIZE", "ui", "email_page_size", false, true},
		{"DEFAULT_EMAIL_VIEW", "ui", "default_view", false, true},
	}

	// 阶段2：功能开关配置
	stage2 := []struct {
		envVar    string
		category  string
		key       string
		sensitive bool
		isPublic  bool
	}{
		// 同步配置
		{"AUTO_SYNC_ENABLED", "sync", "enable_auto_sync", false, true},
		{"SYNC_INTERVAL_SECONDS", "sync", "sync_interval", false, true},
		{"MAX_CONCURRENT_SYNCS", "sync", "max_concurrent_syncs", false, true},
		{"ENABLE_SPAM_FILTER", "email", "enable_spam_filter", false, true},
		{"ENABLE_EMAIL_PUSH", "notification", "enable_email_push", false, true},
		{"ENABLE_DESKTOP_NOTIFICATION", "notification", "enable_desktop_notification", false, true},
	}

	// 阶段3：OAuth配置（敏感）
	stage3 := []struct {
		envVar    string
		category  string
		key       string
		sensitive bool
		isPublic  bool
	}{
		// OAuth2配置
		{"GMAIL_CLIENT_ID", "oauth", "gmail_client_id", false, false},
		{"GMAIL_CLIENT_SECRET", "oauth", "gmail_client_secret", true, false},
		{"MICROSOFT_CLIENT_ID", "oauth", "microsoft_client_id", false, false},
		{"MICROSOFT_CLIENT_SECRET", "oauth", "microsoft_client_secret", true, false},
	}

	// 阶段4：安全配置（敏感）
	stage4 := []struct {
		envVar    string
		category  string
		key       string
		sensitive bool
		isPublic  bool
	}{
		// 安全配置
		{"JWT_SECRET", "security", "jwt_secret", true, false},
		{"JWT_EXPIRY_HOURS", "security", "jwt_expiry", false, false},
		{"MASTER_PASSWORD", "security", "master_password", true, false},
		{"SESSION_TIMEOUT_MINUTES", "security", "session_timeout", false, false},
		{"LOGIN_MAX_ATTEMPTS", "security", "login_max_attempts", false, false},
		{"PASSWORD_COMPLEXITY_REQUIRED", "security", "password_complexity", false, false},
	}

	// 阶段5：性能配置
	stage5 := []struct {
		envVar    string
		category  string
		key       string
		sensitive bool
		isPublic  bool
	}{
		// API配置
		{"RATE_LIMIT_ENABLED", "api", "rate_limit_enabled", false, false},
		{"RATE_LIMIT_SITE_REQUESTS_PER_MINUTE", "api", "rate_limit_site", false, false},
		{"RATE_LIMIT_PUBLIC_REQUESTS_PER_MINUTE", "api", "rate_limit_public", false, false},
	}

	// 阶段6：SMTP配置
	stage6 := []struct {
		envVar    string
		category  string
		key       string
		sensitive bool
		isPublic  bool
	}{
		// SMTP配置
		{"SMTP_HOST", "smtp", "smtp_host", false, false},
		{"SMTP_PORT", "smtp", "smtp_port", false, false},
		{"SMTP_USERNAME", "smtp", "smtp_username", false, false},
		{"SMTP_PASSWORD", "smtp", "smtp_password", true, false},
		{"SMTP_FROM", "smtp", "smtp_from", false, false},
		{"SMTP_FROM_NAME", "smtp", "smtp_from_name", false, false},
	}

	// 执行分阶段迁移
	stages := []struct {
		name string
		list []struct {
			envVar    string
			category  string
			key       string
			sensitive bool
			isPublic  bool
		}
	}{
		{"UI配置", stage1},
		{"功能开关配置", stage2},
		{"OAuth2配置", stage3},
		{"安全配置", stage4},
		{"性能配置", stage5},
		{"SMTP配置", stage6},
	}

	// 执行迁移
	for _, stage := range stages {
		log.Printf("\n迁移阶段: %s", stage.name)
		migratedCount := 0
		skippedCount := 0

		for _, item := range stage.list {
			// 检查环境变量是否存在
			envValue := os.Getenv(item.envVar)
			if envValue == "" {
				log.Printf("  跳过 %s: 环境变量未设置", item.envVar)
				skippedCount++
				continue
			}

			// 检查是否已存在
			exists, err := settingRepo.Exists(ctx, nil, item.category, item.key)
			if err != nil {
				log.Printf("  错误 %s: 检查存在性失败: %v", item.envVar, err)
				continue
			}

			if exists {
				log.Printf("  跳过 %s: 配置已存在", item.envVar)
				skippedCount++
				continue
			}

			// 如果是敏感配置，且不是密码类，先加密
			valueToStore := envValue
			if item.sensitive && !strings.Contains(strings.ToLower(item.key), "password") && !strings.Contains(strings.ToLower(item.key), "secret") {
				// 这些配置需要加密存储
				encrypted, err := encryptor.Encrypt(envValue)
				if err != nil {
					log.Printf("  错误 %s: 加密失败: %v", item.envVar, err)
					continue
				}
				valueToStore = encrypted
			}

			// 设置到数据库
			valueType := getValueType(item.category, item.key)
			if err := settingRepo.Set(ctx, nil, item.category, item.key, valueToStore, item.sensitive, valueType); err != nil {
				log.Printf("  错误 %s: 写入失败: %v", item.envVar, err)
				continue
			}

			log.Printf("  ✓ 迁移 %s -> %s:%s", item.envVar, item.category, item.key)
			migratedCount++
		}

		log.Printf("阶段 %s 完成: 迁移 %d 个，跳过 %d 个", stage.name, migratedCount, skippedCount)
	}

	log.Println("\n✓ 所有阶段迁移完成")
	return nil
}

// getValueType 根据配置项推断类型
func getValueType(category, key string) string {
	if strings.Contains(key, "enable") || strings.Contains(key, "enabled") {
		return "boolean"
	}
	if strings.Contains(key, "size") || strings.Contains(key, "count") || strings.Contains(key, "timeout") || strings.Contains(key, "interval") || strings.Contains(key, "port") || strings.Contains(key, "attempts") {
		return "number"
	}
	if strings.Contains(key, "json") || strings.Contains(key, "config") {
		return "json"
	}
	return "string"
}
