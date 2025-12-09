package database

import (
	"fmt"
	"time"

	"fusionmail/config"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 模块日志记录器
var log = logger.NewWithModule("Database")

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
	log.Info("正在连接数据库: %s", hiddenDSN)
	log.Debug("数据库配置: DisablePrepareStmt=%v, MaxIdleConns=%d, MaxOpenConns=%d, ConnMaxLifetime=%d min",
		cfg.DisablePrepareStmt, cfg.MaxIdleConns, cfg.MaxOpenConns, cfg.ConnMaxLifetime)

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// 禁用 PrepareStmt 以支持 Supabase Transaction 模式（端口 6543）
		// Transaction 模式的连接池不支持 prepared statements
		PrepareStmt: !cfg.DisablePrepareStmt,
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

	// 设置连接池参数（从配置读取，支持 Transaction 模式优化）
	// Transaction 模式建议：MaxIdleConns=2-5, MaxOpenConns=10-20
	// Session 模式/直连：MaxIdleConns=10, MaxOpenConns=100
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 100
	}
	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 60
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)

	DB = db
	log.Info("数据库连接成功 (PrepareStmt=%v)", !cfg.DisablePrepareStmt)

	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	log.Info("开始数据库自动迁移...")

	// 执行表重命名迁移（oauth2_clients -> email_oauth2_tokens）
	if err := migrateOAuth2ClientsTable(); err != nil {
		log.Warn("oauth2_clients 表迁移失败: %v", err)
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

	log.Info("数据库自动迁移完成")

	// 创建全文搜索索引（PostgreSQL 特定）
	if err := createFullTextSearchIndex(); err != nil {
		log.Warn("全文搜索索引创建失败: %v", err)
		// 不返回错误，因为这不是致命的
	}

	// 创建Setting表优化索引
	if err := createSettingIndexes(); err != nil {
		log.Warn("Setting 表索引创建失败: %v", err)
		// 不返回错误，因为这不是致命的
	}

	// 初始化 Provider 种子数据
	if err := seedProviders(); err != nil {
		log.Warn("Provider 种子数据初始化失败: %v", err)
		// 不返回错误，因为这不是致命的
	}

	// 初始化 Settings 种子数据
	if err := seedSettings(); err != nil {
		log.Warn("Settings 种子数据初始化失败: %v", err)
		// 不返回错误，因为这不是致命的
	}

	return nil
}

// seedProviders 初始化邮箱提供商种子数据
func seedProviders() error {
	log.Debug("初始化邮箱提供商数据...")

	// 定义所有邮箱提供商种子数据
	providers := []model.Provider{
		{
			Name:                "gmail",
			DisplayName:         "Gmail",
			ProviderType:        1,
			SupportedProtocols:  `["oauth2","imap"]`,
			RecommendedProtocol: "oauth2",
			RequiresOAuth:       true,
			IMAPHost:            "imap.gmail.com",
			IMAPPort:            993,
			SMTPHost:            "smtp.gmail.com",
			SMTPPort:            587,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "starttls",
			Enabled:             true,
			SortOrder:           1,
			Description:         "Google Gmail 邮箱服务",
		},
		{
			Name:                "outlook",
			DisplayName:         "Outlook / Hotmail",
			ProviderType:        2,
			SupportedProtocols:  `["oauth2","imap"]`,
			RecommendedProtocol: "oauth2",
			RequiresOAuth:       true,
			IMAPHost:            "outlook.office365.com",
			IMAPPort:            993,
			SMTPHost:            "smtp.office365.com",
			SMTPPort:            587,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "starttls",
			Enabled:             true,
			SortOrder:           2,
			Description:         "Microsoft Outlook / Hotmail 邮箱服务",
		},
		{
			Name:                "icloud",
			DisplayName:         "iCloud Mail",
			ProviderType:        3,
			SupportedProtocols:  `["imap"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.mail.me.com",
			IMAPPort:            993,
			SMTPHost:            "smtp.mail.me.com",
			SMTPPort:            587,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "starttls",
			Enabled:             true,
			SortOrder:           3,
			Description:         "Apple iCloud 邮箱服务",
		},
		{
			Name:                "qq",
			DisplayName:         "QQ 邮箱",
			ProviderType:        4,
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.qq.com",
			IMAPPort:            993,
			POP3Host:            "pop.qq.com",
			POP3Port:            995,
			SMTPHost:            "smtp.qq.com",
			SMTPPort:            465,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "ssl",
			Enabled:             true,
			SortOrder:           4,
			Description:         "腾讯 QQ 邮箱服务，需要使用授权码登录",
		},
		{
			Name:                "163",
			DisplayName:         "163 邮箱",
			ProviderType:        5,
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.163.com",
			IMAPPort:            993,
			POP3Host:            "pop.163.com",
			POP3Port:            995,
			SMTPHost:            "smtp.163.com",
			SMTPPort:            465,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "ssl",
			Enabled:             true,
			SortOrder:           5,
			Description:         "网易 163 邮箱服务，需要使用授权码登录",
		},
		{
			Name:                "139",
			DisplayName:         "139 邮箱 (中国移动)",
			ProviderType:        1,
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.139.com",
			IMAPPort:            993,
			POP3Host:            "pop.139.com",
			POP3Port:            995,
			SMTPHost:            "smtp.139.com",
			SMTPPort:            465,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "ssl",
			Enabled:             true,
			SortOrder:           6,
			Description:         "中国移动 139 邮箱服务，需要使用授权码登录",
		},
		{
			Name:                "126",
			DisplayName:         "126 邮箱 (网易)",
			ProviderType:        1,
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.126.com",
			IMAPPort:            993,
			POP3Host:            "pop.126.com",
			POP3Port:            995,
			SMTPHost:            "smtp.126.com",
			SMTPPort:            465,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "ssl",
			Enabled:             true,
			SortOrder:           7,
			Description:         "网易 126 邮箱服务，需要使用授权码登录",
		},
		{
			Name:                "189",
			DisplayName:         "189 邮箱 (中国电信)",
			ProviderType:        1,
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.189.cn",
			IMAPPort:            993,
			POP3Host:            "pop.189.cn",
			POP3Port:            995,
			SMTPHost:            "smtp.189.cn",
			SMTPPort:            465,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "ssl",
			Enabled:             true,
			SortOrder:           8,
			Description:         "中国电信 189 邮箱服务",
		},
		{
			Name:                "generic",
			DisplayName:         "通用邮箱 (IMAP/POP3)",
			ProviderType:        1,
			SupportedProtocols:  `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPPort:            993,
			POP3Port:            995,
			SMTPPort:            587,
			IMAPEncryption:      "ssl",
			POP3Encryption:      "ssl",
			SMTPEncryption:      "starttls",
			Enabled:             true,
			SortOrder:           99,
			Description:         "支持标准 IMAP/POP3 协议的通用邮箱",
		},
	}

	// 使用 FirstOrCreate 确保不会重复插入
	for _, provider := range providers {
		var existing model.Provider
		result := DB.Where("name = ?", provider.Name).First(&existing)
		if result.Error != nil {
			// 记录不存在，创建新记录
			if err := DB.Create(&provider).Error; err != nil {
				log.Warn("创建 Provider 失败: %s, %v", provider.Name, err)
			} else {
				log.Debug("创建 Provider: %s", provider.Name)
			}
		} else {
			// 记录已存在，更新加密字段（如果为空）
			updates := make(map[string]interface{})
			if existing.IMAPEncryption == "" {
				updates["imap_encryption"] = provider.IMAPEncryption
			}
			if existing.POP3Encryption == "" {
				updates["pop3_encryption"] = provider.POP3Encryption
			}
			if existing.SMTPEncryption == "" {
				updates["smtp_encryption"] = provider.SMTPEncryption
			}
			if len(updates) > 0 {
				DB.Model(&existing).Updates(updates)
				log.Debug("更新 Provider 加密字段: %s", provider.Name)
			}
		}
	}

	log.Debug("邮箱提供商数据初始化完成")
	return nil
}

// seedSettings 初始化系统设置种子数据
func seedSettings() error {
	log.Debug("初始化系统设置数据...")

	// 定义所有默认设置种子数据（系统级，userID 为 nil）
	defaultSettings := []model.Setting{
		// UI 设置
		{Category: "ui", Key: "theme", Value: "system", ValueType: "string", IsPublic: true, Description: "界面主题：light/dark/system"},
		{Category: "ui", Key: "language", Value: "zh-CN", ValueType: "string", IsPublic: true, Description: "界面语言"},
		{Category: "ui", Key: "email_page_size", Value: "50", ValueType: "number", IsPublic: true, Description: "每页显示邮件数量"},
		{Category: "ui", Key: "default_view", Value: "list", ValueType: "string", IsPublic: true, Description: "默认视图模式：list/card"},

		// 同步设置
		{Category: "sync", Key: "enable_auto_sync", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用自动同步"},
		{Category: "sync", Key: "sync_interval", Value: "300", ValueType: "number", IsPublic: true, Description: "同步间隔（秒）"},
		{Category: "sync", Key: "max_concurrent_syncs", Value: "3", ValueType: "number", IsPublic: false, Description: "最大并发同步数"},

		// 通知设置
		{Category: "notification", Key: "enable_desktop_notification", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用桌面通知"},
		{Category: "notification", Key: "enable_email_notification", Value: "false", ValueType: "boolean", IsPublic: true, Description: "启用邮件通知"},
		{Category: "notification", Key: "notification_sound", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用通知声音"},
		{Category: "notification", Key: "unread_only", Value: "true", ValueType: "boolean", IsPublic: true, Description: "仅通知未读邮件"},

		// 安全设置
		{Category: "security", Key: "session_timeout", Value: "1440", ValueType: "number", IsPublic: false, Description: "会话超时时间（分钟）"},
		{Category: "security", Key: "login_max_attempts", Value: "5", ValueType: "number", IsPublic: false, Description: "最大登录尝试次数"},
		{Category: "security", Key: "password_complexity", Value: "true", ValueType: "boolean", IsPublic: false, Description: "启用密码复杂度检查"},
		{Category: "security", Key: "jwt_expiry", Value: "24", ValueType: "number", IsPublic: false, Description: "JWT 过期时间（小时）"},

		// API 设置
		{Category: "api", Key: "rate_limit_enabled", Value: "true", ValueType: "boolean", IsPublic: false, Description: "启用 API 速率限制"},
		{Category: "api", Key: "rate_limit_site", Value: "100", ValueType: "number", IsPublic: false, Description: "站点 API 速率限制（次/分钟）"},
		{Category: "api", Key: "rate_limit_public", Value: "200", ValueType: "number", IsPublic: false, Description: "公开 API 速率限制（次/分钟）"},

		// 系统设置
		{Category: "system", Key: "trash_auto_cleanup_days", Value: "7", ValueType: "number", IsPublic: false, Description: "回收站自动清理天数，-1 表示永不清理"},

		// 垃圾邮件设置
		{Category: "spam", Key: "spam_detection_enabled", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用垃圾邮件检测"},
		{Category: "spam", Key: "user_spam_detection_enabled", Value: "true", ValueType: "boolean", IsPublic: true, Description: "用户级垃圾邮件检测"},
		{Category: "spam", Key: "spam_threshold", Value: "60", ValueType: "number", IsPublic: true, Description: "垃圾邮件评分阈值（0-100）"},
		{Category: "spam", Key: "spam_auto_cleanup_days", Value: "30", ValueType: "number", IsPublic: true, Description: "垃圾邮件自动清理天数，-1 表示永不清理"},
		{Category: "spam", Key: "bayesian_enabled", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用贝叶斯过滤"},
		{Category: "spam", Key: "rbl_enabled", Value: "false", ValueType: "boolean", IsPublic: true, Description: "启用 RBL 检查"},
		{Category: "spam", Key: "surbl_enabled", Value: "false", ValueType: "boolean", IsPublic: true, Description: "启用 SURBL 检查"},
	}

	// 使用 FirstOrCreate 确保不会重复插入
	for _, setting := range defaultSettings {
		var existing model.Setting
		// 系统级设置 user_id 为 NULL
		result := DB.Where("user_id IS NULL AND category = ? AND key = ?", setting.Category, setting.Key).First(&existing)
		if result.Error != nil {
			// 记录不存在，创建新记录
			if err := DB.Create(&setting).Error; err != nil {
				log.Warn("创建设置失败: %s/%s, %v", setting.Category, setting.Key, err)
			} else {
				log.Debug("创建设置: %s/%s = %s", setting.Category, setting.Key, setting.Value)
			}
		}
		// 如果记录已存在，不更新（保留用户自定义值）
	}

	log.Debug("系统设置数据初始化完成")
	return nil
}

// createFullTextSearchIndex 创建全文搜索索引
func createFullTextSearchIndex() error {
	log.Debug("创建全文搜索索引...")

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
		log.Debug("全文搜索索引已存在，跳过")
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

	log.Debug("全文搜索索引创建成功")
	return nil
}

// createSettingIndexes 创建Setting表优化索引
func createSettingIndexes() error {
	log.Debug("创建 Setting 表索引...")

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
			log.Warn("检查索引失败: %s, %v", idx.name, err)
			continue
		}

		if exists {
			log.Debug("索引已存在，跳过: %s", idx.name)
			continue
		}

		// 创建索引
		if err := DB.Exec(idx.query).Error; err != nil {
			return fmt.Errorf("创建索引失败 %s: %w", idx.name, err)
		}

		log.Debug("索引创建成功: %s", idx.name)
	}

	log.Debug("Setting 表索引创建完成")
	return nil
}

// SeedInitialData 添加初始数据（如果需要）
func SeedInitialData() error {
	log.Debug("检查初始数据...")

	// 暂时跳过初始数据检查，因为 User 模型有问题
	// TODO: 修复 User 模型后重新启用
	log.Debug("初始数据跳过 (User 模型已禁用)")

	// 初始化提供商数据
	if err := seedProviders(); err != nil {
		log.Warn("Provider 种子数据初始化失败: %v", err)
		// 不返回错误，因为这不是致命的
	}

	// 初始化 OAuth2 客户端数据
	if err := seedOAuth2Clients(); err != nil {
		log.Warn("OAuth2 客户端种子数据初始化失败: %v", err)
		// 不返回错误，因为这不是致命的
	}

	log.Debug("初始数据初始化完成")
	return nil
}

// seedOAuth2Clients 初始化 OAuth2 客户端数据
// 注意：不插入占位符数据，让用户通过前端界面创建真实的配置
func seedOAuth2Clients() error {
	log.Debug("OAuth2 客户端跳过 (无默认占位符)")
	return nil
}

// migrateOAuth2ClientsTable 迁移 o_auth2_clients 表到 email_oauth2_tokens
// 注意：GORM 自动将 OAuth2Client 转换为 o_auth2_clients（蛇形命名）
func migrateOAuth2ClientsTable() error {
	log.Debug("检查 o_auth2_clients 表迁移...")

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

	log.Debug("旧表 (o_auth2_clients) 存在: %v, 新表 (email_oauth2_tokens) 存在: %v", oldTableExists, newTableExists)

	// 如果旧表存在且新表不存在，执行重命名
	if oldTableExists && !newTableExists {
		log.Info("重命名 o_auth2_clients 到 email_oauth2_tokens...")

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
				log.Debug("索引重命名跳过: %v", err)
				// 继续执行，不中断
			}
		}

		// 重命名外键约束
		DB.Exec(`ALTER TABLE email_oauth2_tokens DROP CONSTRAINT IF EXISTS fk_o_auth2_clients_provider`)
		DB.Exec(`ALTER TABLE email_oauth2_tokens DROP CONSTRAINT IF EXISTS fk_oauth2_clients_provider`)
		DB.Exec(`ALTER TABLE email_oauth2_tokens ADD CONSTRAINT fk_email_oauth2_tokens_provider FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE`)

		log.Info("表 o_auth2_clients 重命名为 email_oauth2_tokens 成功")
	} else if newTableExists {
		log.Debug("表 email_oauth2_tokens 已存在，跳过迁移")
	} else {
		log.Debug("表 o_auth2_clients 不存在，将创建为 email_oauth2_tokens")
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
