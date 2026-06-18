package seed

import (
	"fmt"

	"fusionmail/internal/model"
	"fusionmail/pkg/logger"

	"gorm.io/gorm"
)

var log = logger.NewWithModule("Seed")

// SeedInitialData 添加初始数据（如果需要）
func SeedInitialData(db *gorm.DB) error {
	log.Debug("检查初始数据...")

	if db == nil {
		return fmt.Errorf("database is not initialized")
	}

	// 暂时跳过初始数据检查，因为 User 模型有问题
	// TODO: 修复 User 模型后重新启用
	log.Debug("初始数据跳过 (User 模型已禁用)")

	// 初始化提供商数据
	if err := seedProviders(db); err != nil {
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

func SeedProviders(db *gorm.DB) error {
	return seedProviders(db)
}

func SeedSettings(db *gorm.DB) error {
	return seedSettings(db)
}

func SeedAdapters(db *gorm.DB) (map[string]int64, error) {
	return seedAdapters(db)
}

func BackfillProviderDefaultAdapters(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.Provider{}) || !db.Migrator().HasColumn(&model.Provider{}, "default_adapter_id") {
		return nil
	}

	adapterIDs, err := seedAdapters(db)
	if err != nil {
		return err
	}

	result := db.Exec(`
		UPDATE providers
		SET default_adapter_id = CASE
			WHEN LOWER(name) = 'gmail' THEN CAST(? AS bigint)
			WHEN LOWER(name) = 'outlook' THEN CAST(? AS bigint)
			WHEN name LIKE 'webapi_%' OR name IN ('cloudflare_temp_email', 'cloud_mail') THEN CAST(? AS bigint)
			ELSE CAST(? AS bigint)
		END
		WHERE default_adapter_id IS NULL
		   OR default_adapter_id = 0
		   OR NOT EXISTS (
			   SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id
		   )
		   OR (
			   (name LIKE 'webapi_%' OR name IN ('cloudflare_temp_email', 'cloud_mail'))
			   AND default_adapter_id = CAST(? AS bigint)
		   )
	`, adapterIDs[model.AdapterNameGmail], adapterIDs[model.AdapterNameGraph], adapterIDs[model.AdapterNameWebAPI], adapterIDs[model.AdapterNameIMAP], adapterIDs[model.AdapterNameIMAP])
	if result.Error != nil {
		return fmt.Errorf("迁移前修复 Provider 默认 Adapter 失败: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		log.Info("迁移前修复 Provider 默认 Adapter: %d 条", result.RowsAffected)
	}
	return nil
}

// seedOAuth2Clients 初始化 OAuth2 客户端数据
// 注意：不插入占位符数据，让用户通过前端界面创建真实的配置
func seedOAuth2Clients() error {
	log.Debug("OAuth2 客户端跳过 (无默认占位符)")
	return nil
}
