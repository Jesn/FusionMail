package database

import (
	"fmt"
	"log"
	"time"

	"fusionmail/config"
	"fusionmail/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DB 全局数据库实例
var DB *gorm.DB

// Initialize 初始化数据库连接
func Initialize(cfg *config.DatabaseConfig) error {
	dsn := cfg.GetDSN()

	// 打印DSN信息（隐藏密码）
	hiddenDSN := fmt.Sprintf("host=%s port=%s user=%s password=*** dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.DBName, cfg.SSLMode)
	log.Printf("Attempting database connection with DSN: %s", hiddenDSN)

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	DB = db
	log.Println("Database connection established successfully")

	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	log.Println("Starting database auto migration...")

	// 执行表重命名迁移（oauth2_clients -> email_oauth2_tokens）
	if err := migrateOAuth2ClientsTable(); err != nil {
		log.Printf("Warning: failed to migrate oauth2_clients table: %v", err)
		// 不返回错误，继续执行其他迁移
	}

	// 定义所有需要迁移的模型
	models := []interface{}{
		&model.User{}, // 启用用户模型
		&model.EmailAccount{},
		&model.Email{},
		&model.EmailAttachment{},
		&model.EmailLabel{},
		&model.EmailLabelRelation{},
		&model.EmailRule{},
		&model.Webhook{},
		&model.WebhookLog{},
		&model.SyncLog{},
		&model.APIKey{},
		&model.Setting{},
		&model.Provider{},     // 新增 Provider 模型
		&model.OAuth2Client{}, // OAuth2 客户端模型（新表名：email_oauth2_tokens）
		// 垃圾邮件检测相关模型
		&model.EmailList{},
		&model.SenderReputation{},
		&model.SpamRule{},
		&model.BayesianTraining{},
		&model.SpamDetectionLog{},
	}

	// 执行自动迁移
	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Println("Database auto migration completed successfully")

	// 创建全文搜索索引（PostgreSQL 特定）
	if err := createFullTextSearchIndex(); err != nil {
		log.Printf("Warning: failed to create full-text search index: %v", err)
		// 不返回错误，因为这不是致命的
	}

	// 创建Setting表优化索引
	if err := createSettingIndexes(); err != nil {
		log.Printf("Warning: failed to create setting indexes: %v", err)
		// 不返回错误，因为这不是致命的
	}

	return nil
}

// createFullTextSearchIndex 创建全文搜索索引
func createFullTextSearchIndex() error {
	log.Println("Creating full-text search index...")

	// 检查索引是否已存在
	var exists bool
	err := DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE indexname = 'idx_emails_fulltext_search'
		)
	`).Scan(&exists).Error

	if err != nil {
		return err
	}

	if exists {
		log.Println("Full-text search index already exists, skipping...")
		return nil
	}

	// 创建全文搜索索引
	sql := `
		CREATE INDEX idx_emails_fulltext_search ON emails 
		USING gin(
			to_tsvector('english', 
				coalesce(subject, '') || ' ' || 
				coalesce(from_name, '') || ' ' || 
				coalesce(text_body, '')
			)
		)
	`

	if err := DB.Exec(sql).Error; err != nil {
		return err
	}

	log.Println("Full-text search index created successfully")
	return nil
}

// createSettingIndexes 创建Setting表优化索引
func createSettingIndexes() error {
	log.Println("Creating setting table indexes...")

	// 定义索引列表
	indexes := []struct {
		name  string
		query string
	}{
		{
			name:  "uk_settings_user_category_key",
			query: `CREATE UNIQUE INDEX IF NOT EXISTS uk_settings_user_category_key ON settings (user_id, category, key)`,
		},
		{
			name:  "idx_settings_category",
			query: `CREATE INDEX IF NOT EXISTS idx_settings_category ON settings (category)`,
		},
		{
			name:  "idx_settings_user_category",
			query: `CREATE INDEX IF NOT EXISTS idx_settings_user_category ON settings (user_id, category)`,
		},
		{
			name:  "idx_settings_sensitive",
			query: `CREATE INDEX IF NOT EXISTS idx_settings_sensitive ON settings (is_sensitive) WHERE is_sensitive = true`,
		},
		{
			name:  "idx_settings_public",
			query: `CREATE INDEX IF NOT EXISTS idx_settings_public ON settings (is_public) WHERE is_public = true`,
		},
	}

	// 创建每个索引
	for _, idx := range indexes {
		// 检查索引是否存在
		var exists bool
		if err := DB.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE indexname = ?
			)
		`, idx.name).Scan(&exists).Error; err != nil {
			log.Printf("Warning: failed to check index %s: %v", idx.name, err)
			continue
		}

		if exists {
			log.Printf("Index %s already exists, skipping...", idx.name)
			continue
		}

		// 创建索引
		if err := DB.Exec(idx.query).Error; err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}

		log.Printf("Index %s created successfully", idx.name)
	}

	log.Println("Setting table indexes created successfully")
	return nil
}

// SeedInitialData 添加初始数据（如果需要）
func SeedInitialData() error {
	log.Println("Checking for initial data...")

	// 暂时跳过初始数据检查，因为 User 模型有问题
	// TODO: 修复 User 模型后重新启用
	log.Println("Initial data seeding skipped (User model disabled)")

	// 初始化提供商数据
	if err := seedProviders(); err != nil {
		log.Printf("Warning: failed to seed providers: %v", err)
		// 不返回错误，因为这不是致命的
	}

	// 初始化 OAuth2 客户端数据
	if err := seedOAuth2Clients(); err != nil {
		log.Printf("Warning: failed to seed OAuth2 clients: %v", err)
		// 不返回错误，因为这不是致命的
	}

	log.Println("Initial data seeding completed")
	return nil
}

// seedProviders 初始化邮箱提供商数据
func seedProviders() error {
	log.Println("Seeding email providers...")

	// 检查是否已有数据
	var count int64
	if err := DB.Model(&model.Provider{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count providers: %w", err)
	}

	if count > 0 {
		log.Println("Providers already seeded, skipping...")
		return nil
	}

	// 定义默认提供商数据（包含 provider_type 字段）
	providers := []model.Provider{
		{
			Name:                "gmail",
			DisplayName:         "Gmail",
			ProviderType:        1, // Gmail
			SupportedProtocols:  `["oauth2","imap"]`,
			RecommendedProtocol: "oauth2",
			RequiresOAuth:       true,
			IMAPHost:            "imap.gmail.com",
			IMAPPort:            993,
			SortOrder:           1,
			Description:         "Google Gmail 邮箱服务",
		},
		{
			Name:                "outlook",
			DisplayName:         "Outlook / Hotmail",
			ProviderType:        2, // Outlook
			SupportedProtocols:  `["oauth2","imap"]`,
			RecommendedProtocol: "oauth2",
			RequiresOAuth:       true,
			IMAPHost:            "outlook.office365.com",
			IMAPPort:            993,
			SortOrder:           2,
			Description:         "Microsoft Outlook / Hotmail 邮箱服务",
		},
		{
			Name:                "icloud",
			DisplayName:         "iCloud Mail",
			ProviderType:        3, // iCloud
			SupportedProtocols:  `["imap"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.mail.me.com",
			IMAPPort:            993,
			SortOrder:           3,
			Description:         "Apple iCloud 邮箱服务",
		},
		{
			Name:                "qq",
			DisplayName:         "QQ 邮箱",
			ProviderType:        4, // QQ
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.qq.com",
			IMAPPort:            993,
			POP3Host:            "pop.qq.com",
			POP3Port:            995,
			SortOrder:           4,
			Description:         "腾讯 QQ 邮箱服务",
		},
		{
			Name:                "163",
			DisplayName:         "163 邮箱",
			ProviderType:        5, // 163
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.163.com",
			IMAPPort:            993,
			POP3Host:            "pop.163.com",
			POP3Port:            995,
			SortOrder:           5,
			Description:         "网易 163 邮箱服务",
		},
		{
			Name:                "generic",
			DisplayName:         "通用邮箱 (IMAP/POP3)",
			ProviderType:        6, // Generic
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPPort:            993,
			POP3Port:            995,
			SortOrder:           99,
			Description:         "支持标准 IMAP/POP3 协议的通用邮箱",
		},
	}

	// 插入数据
	for _, provider := range providers {
		if err := DB.Create(&provider).Error; err != nil {
			return fmt.Errorf("failed to create provider %s: %w", provider.Name, err)
		}
		log.Printf("Created provider: %s", provider.Name)
	}

	log.Printf("Seeded %d providers successfully", len(providers))
	return nil
}

// seedOAuth2Clients 初始化 OAuth2 客户端数据
// 注意：不插入占位符数据，让用户通过前端界面创建真实的配置
func seedOAuth2Clients() error {
	log.Println("OAuth2 clients seeding skipped (no default placeholders)")
	log.Println("Please create OAuth2 client configurations via the UI")
	return nil
}

// migrateOAuth2ClientsTable 迁移 o_auth2_clients 表到 email_oauth2_tokens
// 注意：GORM 自动将 OAuth2Client 转换为 o_auth2_clients（蛇形命名）
func migrateOAuth2ClientsTable() error {
	log.Println("Checking o_auth2_clients table migration...")

	// 检查旧表是否存在（GORM 生成的表名是 o_auth2_clients）
	var oldTableExists bool
	if err := DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'o_auth2_clients'
		)
	`).Scan(&oldTableExists).Error; err != nil {
		return fmt.Errorf("failed to check old table: %w", err)
	}

	// 检查新表是否存在
	var newTableExists bool
	if err := DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'email_oauth2_tokens'
		)
	`).Scan(&newTableExists).Error; err != nil {
		return fmt.Errorf("failed to check new table: %w", err)
	}

	log.Printf("Old table (o_auth2_clients) exists: %v, New table (email_oauth2_tokens) exists: %v", oldTableExists, newTableExists)

	// 如果旧表存在且新表不存在，执行重命名
	if oldTableExists && !newTableExists {
		log.Println("Renaming o_auth2_clients to email_oauth2_tokens...")

		// 重命名表
		if err := DB.Exec(`ALTER TABLE o_auth2_clients RENAME TO email_oauth2_tokens`).Error; err != nil {
			return fmt.Errorf("failed to rename table: %w", err)
		}

		// 重命名索引（忽略不存在的索引错误）
		indexRenames := []string{
			`ALTER INDEX IF EXISTS idx_o_auth2_clients_provider_id RENAME TO idx_email_oauth2_tokens_provider_id`,
			`ALTER INDEX IF EXISTS idx_o_auth2_clients_enabled RENAME TO idx_email_oauth2_tokens_enabled`,
			`ALTER INDEX IF EXISTS idx_o_auth2_clients_is_default RENAME TO idx_email_oauth2_tokens_is_default`,
			`ALTER INDEX IF EXISTS idx_oauth2_clients_provider_id RENAME TO idx_email_oauth2_tokens_provider_id`,
		}

		for _, sql := range indexRenames {
			if err := DB.Exec(sql).Error; err != nil {
				log.Printf("Warning: failed to rename index: %v", err)
				// 继续执行，不中断
			}
		}

		// 重命名外键约束
		DB.Exec(`ALTER TABLE email_oauth2_tokens DROP CONSTRAINT IF EXISTS fk_o_auth2_clients_provider`)
		DB.Exec(`ALTER TABLE email_oauth2_tokens DROP CONSTRAINT IF EXISTS fk_oauth2_clients_provider`)
		DB.Exec(`ALTER TABLE email_oauth2_tokens ADD CONSTRAINT fk_email_oauth2_tokens_provider FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE`)

		log.Println("Table o_auth2_clients renamed to email_oauth2_tokens successfully")
	} else if newTableExists {
		log.Println("Table email_oauth2_tokens already exists, skipping migration")
	} else {
		log.Println("Table o_auth2_clients does not exist, will be created as email_oauth2_tokens")
	}

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
