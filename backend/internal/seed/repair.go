package seed

import (
	"fmt"

	"fusionmail/internal/model"

	"gorm.io/gorm"
)

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
