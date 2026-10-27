package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"fusionmail/config"
	"fusionmail/internal/repository"
	"fusionmail/internal/seed"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"gorm.io/gorm"
)

func main() {
	// 定义命令行参数
	action := flag.String("action", "up",
		"Migration action: up (apply versioned migrations) | status (show schema version) | "+
			"down (rollback N versions, test only) | seed (seed initial data) | "+
			"automigrate (GORM AutoMigrate, dev/test only) | migrate-settings (migrate env vars)")
	downSteps := flag.Int("steps", 1, "down 动作回滚的版本数（<=0 表示全部回滚）")
	flag.Parse()

	log.Println("FusionMail Database Migration Tool")
	log.Printf("Action: %s", *action)

	// 加载配置
	cfg := config.Load()
	log.Printf("Database: %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 初始化数据库连接（显式持有句柄，P17：消除 GetDB() 隐式全局读取）
	db, err := database.Initialize(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 执行迁移操作
	switch *action {
	case "up":
		// schema 的唯一变更入口：版本化 SQL 迁移。
		//
		// 历史实现这里调用的是 GORM AutoMigrate + 无条件 seed，而 fly.toml 的
		// release_command 又指向本命令，导致每次部署都在 release 模式做
		// schema 变更与数据写入，绕过 ENABLE_AUTO_MIGRATE，且 migrations/ 下的
		// 39 个 SQL 迁移从未被执行过（实测 24 成功 / 15 失败，无法从零建库）。
		migrationsDir, err := database.ResolveMigrationsDir()
		if err != nil {
			log.Fatalf("定位迁移目录失败: %v", err)
		}
		log.Printf("迁移目录: %s", migrationsDir)

		if err := database.RunVersionedMigrations(migrationsDir); err != nil {
			if errors.Is(err, database.ErrNoChange) {
				log.Println("数据库已是最新版本，无需迁移")
			} else {
				log.Fatalf("Migration failed: %v", err)
			}
		} else {
			log.Println("Migration completed successfully!")
		}

		if v, dirty, vErr := database.VersionedMigrationStatus(migrationsDir); vErr == nil {
			log.Printf("当前 schema 版本: %d (dirty=%v)", v, dirty)
		}

	case "down":
		migrationsDir, err := database.ResolveMigrationsDir()
		if err != nil {
			log.Fatalf("定位迁移目录失败: %v", err)
		}
		if err := database.RollbackVersionedMigrations(migrationsDir, *downSteps); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		log.Println("Rollback completed successfully!")

	case "seed":
		log.Println("Seeding initial data...")
		if err := seed.SeedInitialData(db); err != nil {
			log.Fatalf("Seeding failed: %v", err)
		}
		log.Println("Seeding completed successfully!")

	case "automigrate":
		// 仅限开发与测试：从 GORM 模型推导 schema。
		// 生产环境禁止使用——它会绕过版本化迁移，使 schema 无法审计与回滚。
		log.Println("WARNING: 正在执行 GORM AutoMigrate（仅限开发/测试环境）")
		if err := database.AutoMigrate(); err != nil {
			log.Fatalf("AutoMigrate failed: %v", err)
		}
		log.Println("AutoMigrate completed successfully!")

	case "migrate-settings":
		log.Println("Starting environment variable migration to settings table...")
		if err := migrateEnvironmentToSettings(cfg, db); err != nil {
			log.Fatalf("Settings migration failed: %v", err)
		}
		log.Println("Settings migration completed successfully!")

	case "status":
		log.Println("Checking database status...")

		// 版本化迁移状态（schema 的权威来源）
		if migrationsDir, dirErr := database.ResolveMigrationsDir(); dirErr == nil {
			if v, dirty, vErr := database.VersionedMigrationStatus(migrationsDir); vErr != nil {
				log.Printf("迁移版本: 读取失败 (%v)", vErr)
			} else if dirty {
				log.Printf("迁移版本: %d —— DIRTY，上一次迁移中途失败，需人工介入", v)
			} else {
				log.Printf("迁移版本: %d (dirty=false)", v)
			}
		}

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
func migrateEnvironmentToSettings(cfg *config.Config, db *gorm.DB) error {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, using environment variables")
	}

	// 创建Repository和Service
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
	ctx := context.Background()
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
