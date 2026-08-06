package database

import (
	"fmt"

	"fusionmail/internal/model"
	"fusionmail/internal/seed"
)

// AutoMigrate 执行开发或显式维护场景下的 GORM 自动迁移。
// 生产发布仍应以 backend/migrations 下的版本化 SQL 为审计入口。
func AutoMigrate() error {
	log.Info("开始数据库自动迁移...")

	// 执行表重命名迁移（oauth2_clients -> email_oauth2_tokens）
	if err := migrateOAuth2ClientsTable(); err != nil {
		log.Warn("oauth2_clients 表迁移失败: %v", err)
		// 不返回错误，继续执行其他迁移
	}

	// 先创建并填充 adapters，避免历史 default_adapter_id=0 在创建外键前阻断迁移
	if err := DB.AutoMigrate(&model.Adapter{}); err != nil {
		return fmt.Errorf("failed to auto migrate adapters: %w", err)
	}
	if err := backfillProviderDefaultAdaptersBeforeAutoMigrate(); err != nil {
		return err
	}

	models := []interface{}{
		&model.User{},
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
		&model.Adapter{},
		&model.Provider{},
		&model.ProviderAdapter{},
		&model.OAuth2Client{},
		&model.AccountGroup{},
		&model.EmailList{},
		&model.SenderReputation{},
		&model.SpamRule{},
		&model.BayesianTraining{},
		&model.SpamDetectionLog{},
	}

	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Info("数据库自动迁移完成")

	if err := createFullTextSearchIndex(); err != nil {
		log.Warn("全文搜索索引创建失败: %v", err)
	}

	if err := createSettingIndexes(); err != nil {
		log.Warn("Setting 表索引创建失败: %v", err)
	}

	if err := seed.SeedProviders(DB); err != nil {
		return fmt.Errorf("Provider 种子数据初始化失败: %w", err)
	}

	if err := seed.SeedSettings(DB); err != nil {
		log.Warn("Settings 种子数据初始化失败: %v", err)
	}

	return nil
}

func backfillProviderDefaultAdaptersBeforeAutoMigrate() error {
	return seed.BackfillProviderDefaultAdapters(DB)
}

// createFullTextSearchIndex 创建全文搜索索引
func createFullTextSearchIndex() error {
	log.Debug("创建全文搜索索引...")

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

// createSettingIndexes 创建 Setting 表优化索引
func createSettingIndexes() error {
	// 清理系统级配置的重复行（PostgreSQL NULL!=NULL 导致 ON CONFLICT 产生的历史脏数据）
	if err := cleanupDuplicateSystemSettings(); err != nil {
		log.Warn("清理系统配置重复行失败: %v", err)
	}

	log.Debug("创建 Setting 表索引...")

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

	for _, idx := range indexes {
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

		if err := DB.Exec(idx.query).Error; err != nil {
			return fmt.Errorf("创建索引失败 %s: %w", idx.name, err)
		}

		log.Debug("索引创建成功: %s", idx.name)
	}

	log.Debug("Setting 表索引创建完成")
	return nil
}

// migrateOAuth2ClientsTable 迁移 o_auth2_clients 表到 email_oauth2_tokens
// 注意：GORM 自动将 OAuth2Client 转换为 o_auth2_clients（蛇形命名）
func migrateOAuth2ClientsTable() error {
	log.Debug("检查 o_auth2_clients 表迁移...")

	var oldTableExists bool
	if err := DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'o_auth2_clients'
		)
	`).Scan(&oldTableExists).Error; err != nil {
		return fmt.Errorf("failed to check old table: %w", err)
	}

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

	if oldTableExists && !newTableExists {
		log.Info("重命名 o_auth2_clients 到 email_oauth2_tokens...")

		if err := DB.Exec(`ALTER TABLE o_auth2_clients RENAME TO email_oauth2_tokens`).Error; err != nil {
			return fmt.Errorf("failed to rename table: %w", err)
		}

		indexRenames := []string{
			`ALTER INDEX IF EXISTS idx_o_auth2_clients_provider_id RENAME TO idx_email_oauth2_tokens_provider_id`,
			`ALTER INDEX IF EXISTS idx_o_auth2_clients_enabled RENAME TO idx_email_oauth2_tokens_enabled`,
			`ALTER INDEX IF EXISTS idx_o_auth2_clients_is_default RENAME TO idx_email_oauth2_tokens_is_default`,
			`ALTER INDEX IF EXISTS idx_oauth2_clients_provider_id RENAME TO idx_email_oauth2_tokens_provider_id`,
		}

		for _, sql := range indexRenames {
			if err := DB.Exec(sql).Error; err != nil {
				log.Debug("索引重命名跳过: %v", err)
			}
		}

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

// cleanupDuplicateSystemSettings 清理系统级配置（user_id IS NULL）的重复行
// PostgreSQL 中 NULL != NULL，ON CONFLICT (user_id, category, key) 对 user_id IS NULL 的行不触发冲突，
// 导致历史数据中可能存在重复行。此函数保留每组 (category, key) 中 updated_at 最新（或 id 最大）的那条。
func cleanupDuplicateSystemSettings() error {
	result := DB.Exec(`
		DELETE FROM settings
		WHERE user_id IS NULL
		  AND id NOT IN (
		    SELECT DISTINCT ON (category, key) id
		    FROM settings
		    WHERE user_id IS NULL
		    ORDER BY category, key, updated_at DESC, id DESC
		  )
	`)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Info("清理系统配置重复行: 删除 %d 行", result.RowsAffected)
	}
	return nil
}
