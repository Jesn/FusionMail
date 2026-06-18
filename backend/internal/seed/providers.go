package seed

import (
	"fmt"

	"fusionmail/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
func seedProviders(db *gorm.DB) error {
	log.Debug("初始化邮箱提供商数据...")

	return db.Transaction(func(tx *gorm.DB) error {
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
