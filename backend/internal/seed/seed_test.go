package seed

import (
	"testing"

	"fusionmail/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedProvidersSeedsAdaptersAndProviderAdapters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:seed_provider_test?mode=memory&cache=shared&_foreign_keys=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(&model.Adapter{}, &model.Provider{}, &model.ProviderAdapter{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	adapterIDs, err := seedAdapters(db)
	if err != nil {
		t.Fatalf("初始化测试 Adapter 失败: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO providers (name, display_name, default_adapter_id, supported_protocols, recommended_protocol, requires_o_auth, enabled, sort_order, description)
		VALUES (?, ?, NULL, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "generic", "历史通用邮箱", `["imap"]`, "imap", false, true, 99, "历史 Provider 数据",
		"webapi_cloud_mail", "历史 Cloud Mail", adapterIDs[model.AdapterNameIMAP], `["webapi"]`, "webapi", false, true, 101, "历史 WebAPI Provider 数据").Error; err != nil {
		t.Fatalf("插入历史 Provider 数据失败: %v", err)
	}
	var webapiProviderID int64
	if err := db.Model(&model.Provider{}).
		Where("name = ?", "webapi_cloud_mail").
		Select("id").
		Scan(&webapiProviderID).Error; err != nil {
		t.Fatalf("查询历史 WebAPI Provider 失败: %v", err)
	}
	if err := db.Create(&model.ProviderAdapter{
		ProviderID: webapiProviderID,
		AdapterID:  adapterIDs[model.AdapterNameIMAP],
		Priority:   0,
	}).Error; err != nil {
		t.Fatalf("插入历史 WebAPI Provider 错误 Adapter 关联失败: %v", err)
	}

	if err := seedProviders(db); err != nil {
		t.Fatalf("初始化 Provider 种子失败: %v", err)
	}

	var adapterCount int64
	if err := db.Model(&model.Adapter{}).Count(&adapterCount).Error; err != nil {
		t.Fatalf("统计 Adapter 失败: %v", err)
	}
	if adapterCount != 4 {
		t.Fatalf("期望 4 个 Adapter，实际 %d 个", adapterCount)
	}

	var providerCount int64
	if err := db.Model(&model.Provider{}).Count(&providerCount).Error; err != nil {
		t.Fatalf("统计 Provider 失败: %v", err)
	}
	if providerCount == 0 {
		t.Fatal("期望写入 Provider 种子数据")
	}

	var missingDefaultCount int64
	if err := db.Model(&model.Provider{}).Where("default_adapter_id IS NULL OR default_adapter_id = 0").Count(&missingDefaultCount).Error; err != nil {
		t.Fatalf("统计默认 Adapter 缺失的 Provider 失败: %v", err)
	}
	if missingDefaultCount != 0 {
		t.Fatalf("期望所有 Provider 都有默认 Adapter，缺失 %d 个", missingDefaultCount)
	}

	assertProviderDefaultAdapter(t, db, "gmail", model.AdapterNameGmail)
	assertProviderDefaultAdapter(t, db, "outlook", model.AdapterNameGraph)
	assertProviderDefaultAdapter(t, db, "generic", model.AdapterNameIMAP)
	assertProviderDefaultAdapter(t, db, "webapi_cloud_mail", model.AdapterNameWebAPI)
	assertProviderAdapter(t, db, "gmail", model.AdapterNameIMAP)
	assertProviderAdapter(t, db, "outlook", model.AdapterNameIMAP)
	assertProviderAdapter(t, db, "webapi_cloudflare_temp_email", model.AdapterNameWebAPI)
	assertProviderAdapter(t, db, "webapi_cloud_mail", model.AdapterNameWebAPI)
	assertProviderAdapterMissing(t, db, "webapi_cloud_mail", model.AdapterNameIMAP)

	var providerAdapterCount int64
	if err := db.Model(&model.ProviderAdapter{}).Count(&providerAdapterCount).Error; err != nil {
		t.Fatalf("统计 ProviderAdapter 失败: %v", err)
	}
	if providerAdapterCount == 0 {
		t.Fatal("期望写入 ProviderAdapter 关联数据")
	}

	if err := seedProviders(db); err != nil {
		t.Fatalf("重复初始化 Provider 种子失败: %v", err)
	}

	var providerAdapterCountAfterSecondSeed int64
	if err := db.Model(&model.ProviderAdapter{}).Count(&providerAdapterCountAfterSecondSeed).Error; err != nil {
		t.Fatalf("重复初始化后统计 ProviderAdapter 失败: %v", err)
	}
	if providerAdapterCountAfterSecondSeed != providerAdapterCount {
		t.Fatalf("ProviderAdapter 种子应保持幂等，第一次 %d 个，第二次 %d 个", providerAdapterCount, providerAdapterCountAfterSecondSeed)
	}
}

func TestBackfillProviderDefaultAdaptersFixesZeroAndInvalidValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:provider_backfill_test?mode=memory&cache=shared&_foreign_keys=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(&model.Adapter{}); err != nil {
		t.Fatalf("迁移 Adapter 表失败: %v", err)
	}
	adapterIDs, err := seedAdapters(db)
	if err != nil {
		t.Fatalf("初始化测试 Adapter 失败: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE providers (
			id integer primary key autoincrement,
			name text unique,
			display_name text,
			default_adapter_id integer,
			supported_protocols text,
			recommended_protocol text,
			requires_o_auth boolean,
			enabled boolean,
			sort_order integer,
			description text
		)
	`).Error; err != nil {
		t.Fatalf("创建旧 Provider 表失败: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO providers (name, display_name, default_adapter_id, supported_protocols, recommended_protocol, requires_o_auth, enabled, sort_order, description)
		VALUES (?, ?, 0, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "gmail", "Gmail", `["oauth2"]`, "oauth2", true, true, 1, "Gmail",
		"webapi_cloud_mail", "Cloud Mail", adapterIDs[model.AdapterNameIMAP], `["webapi"]`, "webapi", false, true, 2, "Cloud Mail",
		"generic", "Generic", 999999, `["imap"]`, "imap", false, true, 3, "Generic").Error; err != nil {
		t.Fatalf("插入旧 Provider 数据失败: %v", err)
	}

	if err := BackfillProviderDefaultAdapters(db); err != nil {
		t.Fatalf("回填 Provider 默认 Adapter 失败: %v", err)
	}

	assertProviderDefaultAdapter(t, db, "gmail", model.AdapterNameGmail)
	assertProviderDefaultAdapter(t, db, "webapi_cloud_mail", model.AdapterNameWebAPI)
	assertProviderDefaultAdapter(t, db, "generic", model.AdapterNameIMAP)
}

func assertProviderDefaultAdapter(t *testing.T, db *gorm.DB, providerName, adapterName string) {
	t.Helper()
	var count int64
	if err := db.Table("providers").
		Joins("JOIN adapters ON adapters.id = providers.default_adapter_id").
		Where("providers.name = ? AND adapters.name = ?", providerName, adapterName).
		Count(&count).Error; err != nil {
		t.Fatalf("查询 Provider 默认 Adapter 失败: provider=%s adapter=%s err=%v", providerName, adapterName, err)
	}
	if count != 1 {
		t.Fatalf("Provider %s 期望默认 Adapter %s", providerName, adapterName)
	}
}

func assertProviderAdapter(t *testing.T, db *gorm.DB, providerName, adapterName string) {
	t.Helper()
	var count int64
	if err := db.Model(&model.ProviderAdapter{}).
		Joins("JOIN providers ON providers.id = provider_adapters.provider_id").
		Joins("JOIN adapters ON adapters.id = provider_adapters.adapter_id").
		Where("providers.name = ? AND adapters.name = ?", providerName, adapterName).
		Count(&count).Error; err != nil {
		t.Fatalf("查询 ProviderAdapter 失败: provider=%s adapter=%s err=%v", providerName, adapterName, err)
	}
	if count != 1 {
		t.Fatalf("Provider %s 期望关联 Adapter %s", providerName, adapterName)
	}
}

func assertProviderAdapterMissing(t *testing.T, db *gorm.DB, providerName, adapterName string) {
	t.Helper()
	var count int64
	if err := db.Model(&model.ProviderAdapter{}).
		Joins("JOIN providers ON providers.id = provider_adapters.provider_id").
		Joins("JOIN adapters ON adapters.id = provider_adapters.adapter_id").
		Where("providers.name = ? AND adapters.name = ?", providerName, adapterName).
		Count(&count).Error; err != nil {
		t.Fatalf("查询 ProviderAdapter 失败: provider=%s adapter=%s err=%v", providerName, adapterName, err)
	}
	if count != 0 {
		t.Fatalf("Provider %s 不应继续关联 Adapter %s", providerName, adapterName)
	}
}
