package database

import (
	"fmt"
	"time"

	"fusionmail/config"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
)

// 模块日志记录器
var log = logger.NewWithModule("Database")

// gormLogWriter 自定义 GORM 日志写入器，将日志输出到项目日志系统
type gormLogWriter struct {
	log *logger.Logger
}

// Printf 实现 gormlogger.Writer 接口
func (w *gormLogWriter) Printf(format string, args ...interface{}) {
	w.log.Warn(format, args...)
}

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

	// 配置 GORM 日志（含慢查询监控）
	// 慢查询阈值：200ms，超过此时间的查询会被记录为 Warn 级别
	gormLogConfig := gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond, // 慢查询阈值
		LogLevel:                  gormlogger.Warn,        // 日志级别
		IgnoreRecordNotFoundError: true,                   // 忽略记录未找到错误
		Colorful:                  false,                  // 生产环境关闭颜色
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.New(
			&gormLogWriter{log: log}, // 使用自定义日志写入器
			gormLogConfig,
		),
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

func seedAdapters(db *gorm.DB) (map[string]int64, error) {
	log.Debug("初始化邮箱适配器数据...")

	adapters := []model.Adapter{
		{
			Name:        model.AdapterNameGmail,
			DisplayName: "Gmail API",
			AuthType:    model.AdapterAuthTypeOAuth2,
			Description: "Gmail OAuth2 API 适配器",
			IsEnabled:   true,
		},
		{
			Name:        model.AdapterNameGraph,
			DisplayName: "Microsoft Graph",
			AuthType:    model.AdapterAuthTypeOAuth2,
			Description: "Microsoft Graph OAuth2 API 适配器",
			IsEnabled:   true,
		},
		{
			Name:        model.AdapterNameIMAP,
			DisplayName: "IMAP/POP3",
			AuthType:    model.AdapterAuthTypePassword,
			Description: "通用 IMAP/POP3 协议适配器",
			IsEnabled:   true,
		},
		{
			Name:        model.AdapterNameWebAPI,
			DisplayName: "Web API",
			AuthType:    model.AdapterAuthTypeToken,
			Description: "通用 Web API 邮箱适配器",
			IsEnabled:   true,
		},
	}

	adapterIDs := make(map[string]int64, len(adapters))
	for _, adapter := range adapters {
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&adapter).Error; err != nil {
			return nil, fmt.Errorf("创建 Adapter 失败 %s: %w", adapter.Name, err)
		}
		if err := updateAdapterSeedDefaults(db, adapter); err != nil {
			return nil, err
		}

		var adapterID int64
		if err := db.Model(&model.Adapter{}).
			Where("name = ?", adapter.Name).
			Select("id").
			Scan(&adapterID).Error; err != nil {
			return nil, fmt.Errorf("查询 Adapter ID 失败 %s: %w", adapter.Name, err)
		}
		if adapterID == 0 {
			return nil, fmt.Errorf("查询 Adapter ID 为空 %s", adapter.Name)
		}
		adapterIDs[adapter.Name] = adapterID
	}

	return adapterIDs, nil
}

func updateAdapterSeedDefaults(db *gorm.DB, adapter model.Adapter) error {
	updates := map[string]any{
		"display_name": gorm.Expr("COALESCE(NULLIF(display_name, ''), ?)", adapter.DisplayName),
		"auth_type":    gorm.Expr("COALESCE(NULLIF(auth_type, ''), ?)", adapter.AuthType),
		"description":  gorm.Expr("COALESCE(NULLIF(description, ''), ?)", adapter.Description),
	}
	if err := db.Model(&model.Adapter{}).
		Where("name = ? AND (display_name IS NULL OR display_name = '' OR auth_type IS NULL OR auth_type = '' OR description IS NULL OR description = '')", adapter.Name).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("更新 Adapter 默认字段失败 %s: %w", adapter.Name, err)
	}
	return nil
}

// seedProviders 初始化邮箱提供商种子数据
func seedProviders() error {
	log.Debug("初始化邮箱提供商数据...")

	return DB.Transaction(func(tx *gorm.DB) error {
		adapterIDs, err := seedAdapters(tx)
		if err != nil {
			return err
		}

		// 定义所有邮箱提供商种子数据
		providers := []model.Provider{
			{
				Name:        "gmail",
				DisplayName: "Gmail",

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
				Name:        "outlook",
				DisplayName: "Outlook / Hotmail",

				SupportedProtocols:  `["oauth2","imap","batch_import"]`,
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
				Name:        "icloud",
				DisplayName: "iCloud Mail",

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
				Name:        "qq",
				DisplayName: "QQ 邮箱",

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
				Name:        "163",
				DisplayName: "163 邮箱",

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
				Name:        "139",
				DisplayName: "139 邮箱 (中国移动)",

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
				Name:        "126",
				DisplayName: "126 邮箱 (网易)",

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
				Name:        "189",
				DisplayName: "189 邮箱 (中国电信)",

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
				Name:        "generic",
				DisplayName: "通用邮箱 (IMAP/POP3)",

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
			// WebAPI Provider - Cloudflare Temp Email
			{
				Name:                "webapi_cloudflare_temp_email",
				DisplayName:         "Cloudflare Temp Email",
				SupportedProtocols:  `["webapi"]`,
				RecommendedProtocol: "webapi",
				RequiresOAuth:       false,
				Enabled:             true,
				SortOrder:           100,
				Description:         "Cloudflare Workers 临时邮箱服务",
				Metadata:            `{"service_type":"cloudflare_temp_email","access_modes":["single","admin"],"github_url":"https://github.com/dreamhunter2333/cloudflare_temp_email"}`,
			},
			// WebAPI Provider - Cloud Mail
			{
				Name:                "webapi_cloud_mail",
				DisplayName:         "Cloud Mail",
				SupportedProtocols:  `["webapi"]`,
				RecommendedProtocol: "webapi",
				RequiresOAuth:       false,
				Enabled:             true,
				SortOrder:           101,
				Description:         "Cloud Mail 邮箱服务 (如 mail.hema.edu.kg)",
				Metadata:            `{"service_type":"cloud_mail","access_modes":["single"],"github_url":"https://github.com/maillab/cloud-mail"}`,
			},
			// 注意：自定义 Web API (webapi_custom) 已移除
			// 原因：自定义 WebAPI 没有通用方案，不同站点需要单独适配
			// 如需支持新的 WebAPI 服务，请创建专门的适配器
		}

		// 使用 FirstOrCreate 确保不会重复插入
		for _, provider := range providers {
			defaultAdapterName := providerDefaultAdapterName(provider)
			defaultAdapterID := adapterIDs[defaultAdapterName]
			if defaultAdapterID == 0 {
				return fmt.Errorf("缺少 Provider 默认 Adapter %s: %s", provider.Name, defaultAdapterName)
			}
			provider.DefaultAdapterID = defaultAdapterID

			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoNothing: true,
			}).Create(&provider).Error; err != nil {
				return fmt.Errorf("创建 Provider 失败 %s: %w", provider.Name, err)
			}
			if err := updateProviderSeedDefaults(tx, provider); err != nil {
				return err
			}

			var providerID int64
			if err := tx.Model(&model.Provider{}).
				Where("name = ?", provider.Name).
				Select("id").
				Scan(&providerID).Error; err != nil {
				return fmt.Errorf("查询 Provider ID 失败 %s: %w", provider.Name, err)
			}
			if providerID == 0 {
				return fmt.Errorf("查询 Provider ID 为空 %s", provider.Name)
			}

			if err := seedProviderAdapters(tx, providerID, providerAdapterNames(provider), adapterIDs); err != nil {
				return err
			}
		}

		if err := repairWebAPIEmailAccountAdapters(tx, adapterIDs[model.AdapterNameWebAPI]); err != nil {
			return err
		}

		log.Debug("邮箱提供商数据初始化完成")
		return nil
	})
}

func updateProviderSeedDefaults(db *gorm.DB, provider model.Provider) error {
	if err := repairProviderDefaultAdapter(db, provider); err != nil {
		return err
	}

	condition := "name = ? AND (default_adapter_id IS NULL OR default_adapter_id = 0 OR display_name IS NULL OR display_name = '' OR description IS NULL OR description = '' OR imap_encryption IS NULL OR imap_encryption = '' OR pop3_encryption IS NULL OR pop3_encryption = '' OR smtp_encryption IS NULL OR smtp_encryption = '' OR supported_protocols IS NULL OR supported_protocols = '' OR recommended_protocol IS NULL OR recommended_protocol = '')"
	updates := map[string]any{
		"default_adapter_id":   gorm.Expr("COALESCE(NULLIF(default_adapter_id, 0), ?)", provider.DefaultAdapterID),
		"display_name":         gorm.Expr("COALESCE(NULLIF(display_name, ''), ?)", provider.DisplayName),
		"description":          gorm.Expr("COALESCE(NULLIF(description, ''), ?)", provider.Description),
		"imap_encryption":      gorm.Expr("COALESCE(NULLIF(imap_encryption, ''), ?)", provider.IMAPEncryption),
		"pop3_encryption":      gorm.Expr("COALESCE(NULLIF(pop3_encryption, ''), ?)", provider.POP3Encryption),
		"smtp_encryption":      gorm.Expr("COALESCE(NULLIF(smtp_encryption, ''), ?)", provider.SMTPEncryption),
		"supported_protocols":  gorm.Expr("COALESCE(NULLIF(supported_protocols, ''), ?)", provider.SupportedProtocols),
		"recommended_protocol": gorm.Expr("COALESCE(NULLIF(recommended_protocol, ''), ?)", provider.RecommendedProtocol),
	}
	if provider.Metadata != "" {
		condition = "name = ? AND (default_adapter_id IS NULL OR default_adapter_id = 0 OR display_name IS NULL OR display_name = '' OR description IS NULL OR description = '' OR imap_encryption IS NULL OR imap_encryption = '' OR pop3_encryption IS NULL OR pop3_encryption = '' OR smtp_encryption IS NULL OR smtp_encryption = '' OR supported_protocols IS NULL OR supported_protocols = '' OR recommended_protocol IS NULL OR recommended_protocol = '' OR metadata IS NULL OR metadata = '')"
		updates["metadata"] = gorm.Expr("COALESCE(NULLIF(metadata, ''), ?)", provider.Metadata)
	}

	if err := db.Model(&model.Provider{}).
		Where(condition, provider.Name).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("更新 Provider 默认字段失败 %s: %w", provider.Name, err)
	}
	return nil
}

func repairProviderDefaultAdapter(db *gorm.DB, provider model.Provider) error {
	condition := `name = ? AND (
		default_adapter_id IS NULL
		OR default_adapter_id = 0
		OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)`
	args := []any{provider.Name}
	if provider.RecommendedProtocol == "webapi" {
		condition += " OR default_adapter_id = (SELECT id FROM adapters WHERE name = ?)"
		args = append(args, model.AdapterNameIMAP)
	}
	condition += ")"

	if err := db.Model(&model.Provider{}).
		Where(condition, args...).Update("default_adapter_id", provider.DefaultAdapterID).Error; err != nil {
		return fmt.Errorf("修复 Provider 默认 Adapter 失败 %s: %w", provider.Name, err)
	}
	return nil
}

func providerDefaultAdapterName(provider model.Provider) string {
	switch provider.Name {
	case "gmail":
		return model.AdapterNameGmail
	case "outlook":
		return model.AdapterNameGraph
	default:
		if provider.RecommendedProtocol == "webapi" {
			return model.AdapterNameWebAPI
		}
		return model.AdapterNameIMAP
	}
}

func providerAdapterNames(provider model.Provider) []string {
	switch provider.Name {
	case "gmail":
		return []string{model.AdapterNameGmail, model.AdapterNameIMAP}
	case "outlook":
		return []string{model.AdapterNameGraph, model.AdapterNameIMAP}
	default:
		if provider.RecommendedProtocol == "webapi" {
			return []string{model.AdapterNameWebAPI}
		}
		return []string{model.AdapterNameIMAP}
	}
}

func seedProviderAdapters(db *gorm.DB, providerID int64, adapterNames []string, adapterIDs map[string]int64) error {
	if providerID == 0 {
		return fmt.Errorf("Provider ID 不能为空")
	}

	if len(adapterNames) == 1 && adapterNames[0] == model.AdapterNameWebAPI {
		webapiAdapterID := adapterIDs[model.AdapterNameWebAPI]
		if webapiAdapterID == 0 {
			return fmt.Errorf("缺少 Provider Adapter %s", model.AdapterNameWebAPI)
		}
		if err := db.Where("provider_id = ? AND adapter_id <> ?", providerID, webapiAdapterID).
			Delete(&model.ProviderAdapter{}).Error; err != nil {
			return fmt.Errorf("清理 WebAPI Provider 错误 Adapter 关联失败 provider_id=%d: %w", providerID, err)
		}
	}

	for priority, adapterName := range adapterNames {
		adapterID := adapterIDs[adapterName]
		if adapterID == 0 {
			return fmt.Errorf("缺少 Provider Adapter %s", adapterName)
		}

		providerAdapter := model.ProviderAdapter{
			ProviderID: providerID,
			AdapterID:  adapterID,
			Priority:   priority,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider_id"}, {Name: "adapter_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"priority"}),
		}).Create(&providerAdapter).Error; err != nil {
			return fmt.Errorf("创建 ProviderAdapter 失败 provider_id=%d adapter=%s: %w", providerID, adapterName, err)
		}
	}

	return nil
}

func repairWebAPIEmailAccountAdapters(db *gorm.DB, webapiAdapterID int64) error {
	if webapiAdapterID == 0 {
		return fmt.Errorf("WebAPI Adapter ID 不能为空")
	}
	if !db.Migrator().HasTable(&model.EmailAccount{}) {
		return nil
	}
	if !db.Migrator().HasColumn(&model.EmailAccount{}, "provider_id") || !db.Migrator().HasColumn(&model.EmailAccount{}, "adapter_id") {
		return nil
	}

	result := db.Exec(`
		UPDATE email_accounts
		SET adapter_id = ?
		FROM providers p
		WHERE email_accounts.provider_id = p.id
		  AND (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail') OR p.recommended_protocol = 'webapi')
		  AND (
		      email_accounts.adapter_id IS NULL
		      OR email_accounts.adapter_id = 0
		      OR email_accounts.adapter_id = (SELECT id FROM adapters WHERE name = 'imap')
		      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = email_accounts.adapter_id)
		  )
	`, webapiAdapterID)
	if result.Error != nil {
		return fmt.Errorf("修复 WebAPI 账户 Adapter 失败: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		log.Info("修复 WebAPI 账户 Adapter: %d 条", result.RowsAffected)
	}
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
		{Category: "system", Key: "sync_logs_retention_days", Value: "7", ValueType: "number", IsPublic: false, Description: "同步日志保留天数，-1 表示永不清理"},
		{Category: "system", Key: "webhook_logs_retention_days", Value: "14", ValueType: "number", IsPublic: false, Description: "Webhook 日志保留天数，-1 表示永不清理"},
		{Category: "system", Key: "spam_detection_logs_retention_days", Value: "7", ValueType: "number", IsPublic: false, Description: "垃圾邮件检测日志保留天数，-1 表示永不清理"},

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

// SeedInitialData 添加初始数据（如果需要）
func SeedInitialData() error {
	log.Debug("检查初始数据...")

	// 暂时跳过初始数据检查，因为 User 模型有问题
	// TODO: 修复 User 模型后重新启用
	log.Debug("初始数据跳过 (User 模型已禁用)")

	// 初始化提供商数据
	if err := seedProviders(); err != nil {
		return fmt.Errorf("Provider 种子数据初始化失败: %w", err)
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
