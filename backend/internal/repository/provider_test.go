package repository

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// 每个测试使用独立的内存数据库，避免数据污染
	dbName := fmt.Sprintf("file::memory:?cache=shared&_fk=1&_journal_mode=WAL")
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&model.Provider{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// 测试结束后自动清理数据
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		sqlDB.Close()
	})

	return db
}

func Test_providerRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProviderRepository(db)
	ctx := context.Background()

	t.Run("创建有效的提供商", func(t *testing.T) {
		provider := &model.Provider{
			Name:               "test_provider",
			DisplayName:        "Test Provider",
			SupportedProtocols: `["imap","pop3"]`,
			RecommendedProtocol: "imap",
			Enabled:            true,
		}

		err := repo.Create(ctx, provider)
		if err != nil {
			t.Errorf("Create() error = %v, wantErr nil", err)
		}

		// 验证记录已创建
		found, err := repo.FindByName(ctx, "test_provider")
		if err != nil {
			t.Errorf("FindByName() error = %v, wantErr nil", err)
		}
		if found == nil {
			t.Error("FindByName() returned nil")
		}
	})

	t.Run("创建无效的提供商", func(t *testing.T) {
		provider := &model.Provider{
			Name:               "invalid_provider",
			DisplayName:        "",
			SupportedProtocols: `["imap"]`,
			RecommendedProtocol: "imap",
		}

		err := repo.Create(ctx, provider)
		if err == nil {
			t.Error("Create() wantErr but got nil")
		}
	})

	t.Run("创建重复名称的提供商", func(t *testing.T) {
		// 先创建一个提供商
		provider1 := &model.Provider{
			Name:               "duplicate_test",
			DisplayName:        "Test Provider",
			SupportedProtocols: `["imap","pop3"]`,
			RecommendedProtocol: "imap",
		}
		err := repo.Create(ctx, provider1)
		if err != nil {
			t.Fatalf("First Create() failed: %v", err)
		}

		// 尝试创建同名的提供商
		provider2 := &model.Provider{
			Name:               "duplicate_test",
			DisplayName:        "Test Provider 2",
			SupportedProtocols: `["oauth2"]`,
			RecommendedProtocol: "oauth2",
		}
		err = repo.Create(ctx, provider2)
		if err == nil {
			t.Error("Create() wantErr but got nil for duplicate name")
		}
	})
}

func Test_providerRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProviderRepository(db)
	ctx := context.Background()

	t.Run("更新现有提供商", func(t *testing.T) {
		// 先创建一个提供商
		provider := &model.Provider{
			Name:               "update_test",
			DisplayName:        "Original Name",
			SupportedProtocols: `["imap","pop3"]`,
			RecommendedProtocol: "imap",
		}
		err := repo.Create(ctx, provider)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		// 更新提供商
		updatedProvider := &model.Provider{
			Name:               "update_test",
			DisplayName:        "Updated Name",
			SupportedProtocols: `["oauth2","imap"]`,
			RecommendedProtocol: "oauth2",
		}
		err = repo.Update(ctx, updatedProvider)
		if err != nil {
			t.Errorf("Update() error = %v, wantErr nil", err)
		}

		// 验证更新
		found, err := repo.FindByName(ctx, "update_test")
		if err != nil {
			t.Errorf("FindByName() error = %v, wantErr nil", err)
		}
		if found == nil {
			t.Error("FindByName() returned nil")
		}
		if found.DisplayName != "Updated Name" {
			t.Errorf("DisplayName = %v, want %v", found.DisplayName, "Updated Name")
		}
	})

	t.Run("更新不存在的提供商", func(t *testing.T) {
		provider := &model.Provider{
			Name:               "nonexistent",
			DisplayName:        "Test",
			SupportedProtocols: `["imap"]`,
			RecommendedProtocol: "imap",
		}

		err := repo.Update(ctx, provider)
		if err == nil {
			t.Error("Update() wantErr but got nil")
		}
	})
}

func Test_providerRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProviderRepository(db)
	ctx := context.Background()

	t.Run("删除现有提供商", func(t *testing.T) {
		// 先创建一个提供商
		provider := &model.Provider{
			Name:               "delete_test",
			DisplayName:        "Delete Test",
			SupportedProtocols: `["imap"]`,
			RecommendedProtocol: "imap",
		}
		err := repo.Create(ctx, provider)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		// 删除提供商
		err = repo.Delete(ctx, "delete_test")
		if err != nil {
			t.Errorf("Delete() error = %v, wantErr nil", err)
		}

		// 验证已删除
		_, err = repo.FindByName(ctx, "delete_test")
		if err == nil {
			t.Error("FindByName() wantErr but got nil after delete")
		}
	})

	t.Run("删除不存在的提供商", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if err == nil {
			t.Error("Delete() wantErr but got nil")
		}
	})
}

func Test_providerRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProviderRepository(db)
	ctx := context.Background()

	t.Run("获取所有提供商", func(t *testing.T) {
		// 创建多个提供商
		providers := []*model.Provider{
			{
				Name:               "provider_a",
				DisplayName:        "Provider A",
				SupportedProtocols: `["imap"]`,
				RecommendedProtocol: "imap",
				SortOrder:          2,
			},
			{
				Name:               "provider_b",
				DisplayName:        "Provider B",
				SupportedProtocols: `["pop3"]`,
				RecommendedProtocol: "pop3",
				SortOrder:          1,
			},
		}

		for _, p := range providers {
			err := repo.Create(ctx, p)
			if err != nil {
				t.Fatalf("Create() failed: %v", err)
			}
		}

		// 获取所有提供商
		found, err := repo.FindAll(ctx)
		if err != nil {
			t.Errorf("FindAll() error = %v, wantErr nil", err)
		}
		if len(found) != 2 {
			t.Errorf("FindAll() got %d providers, want %d", len(found), 2)
		}

		// 验证排序
		if found[0].Name != "provider_b" {
			t.Errorf("First provider name = %v, want %v", found[0].Name, "provider_b")
		}
		if found[1].Name != "provider_a" {
			t.Errorf("Second provider name = %v, want %v", found[1].Name, "provider_a")
		}
	})
}

func Test_providerRepository_FindByName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProviderRepository(db)
	ctx := context.Background()

	t.Name()

	t.Run("查找存在的提供商", func(t *testing.T) {
		provider := &model.Provider{
			Name:               "find_test",
			DisplayName:        "Find Test",
			SupportedProtocols: `["imap"]`,
			RecommendedProtocol: "imap",
		}
		err := repo.Create(ctx, provider)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		found, err := repo.FindByName(ctx, "find_test")
		if err != nil {
			t.Errorf("FindByName() error = %v, wantErr nil", err)
		}
		if found == nil {
			t.Error("FindByName() returned nil")
		}
		if found.Name != "find_test" {
			t.Errorf("Name = %v, want %v", found.Name, "find_test")
		}
	})

	t.Run("查找不存在的提供商", func(t *testing.T) {
		_, err := repo.FindByName(ctx, "nonexistent")
		if err == nil {
			t.Error("FindByName() wantErr but got nil")
		}
	})
}

func Test_providerRepository_FindEnabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProviderRepository(db)
	ctx := context.Background()

	t.Run("获取启用的提供商", func(t *testing.T) {
		// 创建多个提供商，其中一些启用，一些禁用
		providers := []*model.Provider{
			{
				Name:               "enabled_provider",
				DisplayName:        "Enabled",
				SupportedProtocols: `["imap"]`,
				RecommendedProtocol: "imap",
				Enabled:            true,
			},
			{
				Name:               "disabled_provider",
				DisplayName:        "Disabled",
				SupportedProtocols: `["pop3"]`,
				RecommendedProtocol: "pop3",
				Enabled:            false,
			},
		}

		for _, p := range providers {
			err := repo.Create(ctx, p)
			if err != nil {
				t.Fatalf("Create() failed: %v", err)
			}
		}

		// 只获取启用的提供商
		found, err := repo.FindEnabled(ctx)
		if err != nil {
			t.Errorf("FindEnabled() error = %v, wantErr nil", err)
		}
		if len(found) != 1 {
			t.Errorf("FindEnabled() got %d providers, want %d", len(found), 1)
		}
		if found[0].Name != "enabled_provider" {
			t.Errorf("First provider name = %v, want %v", found[0].Name, "enabled_provider")
		}
	})
}
