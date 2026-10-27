package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupProviderServiceTestDB 创建 Provider 服务测试用的内存数据库
// 每个测试使用独立的内存数据库，避免数据污染（与 repository/batch_count_test.go 的 setup 模式同风格）
func setupProviderServiceTestDB(t *testing.T) *gorm.DB {
	dbName := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	err = db.AutoMigrate(&model.Provider{}, &model.Adapter{}, &model.ProviderAdapter{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		sqlDB.Close()
	})

	return db
}

// newProviderServiceWithTestDB 基于测试库构造 ProviderService（带 AdapterRepository）
func newProviderServiceWithTestDB(db *gorm.DB) *ProviderService {
	return NewProviderServiceWithAdapterRepo(
		repository.NewProviderRepository(db),
		repository.NewAdapterRepository(db),
	)
}

// createTestAdapter 创建测试适配器
func createTestAdapter(t *testing.T, db *gorm.DB, name string) *model.Adapter {
	t.Helper()
	adapter := &model.Adapter{
		Name:        name,
		DisplayName: name + " Adapter",
		AuthType:    model.AdapterAuthTypePassword,
		IsEnabled:   true,
	}
	if err := db.Create(adapter).Error; err != nil {
		t.Fatalf("创建测试适配器失败: %v", err)
	}
	return adapter
}

// newTestProvider 构造一个能通过 Validate 的 Provider
func newTestProvider(name string) *model.Provider {
	return &model.Provider{
		Name:                name,
		DisplayName:         "Test Provider " + name,
		SupportedProtocols:  `["imap"]`,
		RecommendedProtocol: "imap",
		Enabled:             true,
	}
}

// countProviderAdapters 统计指定 Provider 的关联行数
func countProviderAdapters(t *testing.T, db *gorm.DB, providerID int64) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&model.ProviderAdapter{}).
		Where("provider_id = ?", providerID).
		Count(&count).Error; err != nil {
		t.Fatalf("统计 ProviderAdapter 失败: %v", err)
	}
	return count
}

// TestProviderService_Create_WithAdapterIDs 创建时 adapter_ids 关联落库
func TestProviderService_Create_WithAdapterIDs(t *testing.T) {
	db := setupProviderServiceTestDB(t)
	svc := newProviderServiceWithTestDB(db)
	ctx := context.Background()

	imap := createTestAdapter(t, db, "imap")
	gmail := createTestAdapter(t, db, "gmail")

	// 请求顺序 imap -> gmail，priority 应按顺序编号（0 为最高）
	created, err := svc.Create(ctx, newTestProvider("test_create"), []int64{imap.ID, gmail.ID})
	if err != nil {
		t.Fatalf("Create() error = %v, wantErr nil", err)
	}

	// 关联落库：恰好 2 行
	if count := countProviderAdapters(t, db, created.ID); count != 2 {
		t.Errorf("provider_adapters 行数 = %d, want 2", count)
	}

	// 返回值已预加载关联，且按 priority 升序、嵌套 Adapter 信息完整
	if len(created.SupportedAdapters) != 2 {
		t.Fatalf("返回的 SupportedAdapters 长度 = %d, want 2", len(created.SupportedAdapters))
	}
	if created.SupportedAdapters[0].AdapterID != imap.ID || created.SupportedAdapters[0].Priority != 0 {
		t.Errorf("第一个关联 = (adapter_id=%d, priority=%d), want (adapter_id=%d, priority=0)",
			created.SupportedAdapters[0].AdapterID, created.SupportedAdapters[0].Priority, imap.ID)
	}
	if created.SupportedAdapters[1].AdapterID != gmail.ID || created.SupportedAdapters[1].Priority != 1 {
		t.Errorf("第二个关联 = (adapter_id=%d, priority=%d), want (adapter_id=%d, priority=1)",
			created.SupportedAdapters[1].AdapterID, created.SupportedAdapters[1].Priority, gmail.ID)
	}
	if created.SupportedAdapters[0].Adapter == nil || created.SupportedAdapters[0].Adapter.Name != "imap" {
		t.Errorf("嵌套 Adapter 未预加载，got %+v", created.SupportedAdapters[0].Adapter)
	}
}

// TestProviderService_Create_NilAdapterIDs 创建时 nil 与 [] 语义相同：无关联
func TestProviderService_Create_NilAdapterIDs(t *testing.T) {
	db := setupProviderServiceTestDB(t)
	svc := newProviderServiceWithTestDB(db)
	ctx := context.Background()

	created, err := svc.Create(ctx, newTestProvider("test_no_adapter"), nil)
	if err != nil {
		t.Fatalf("Create() error = %v, wantErr nil", err)
	}

	if count := countProviderAdapters(t, db, created.ID); count != 0 {
		t.Errorf("provider_adapters 行数 = %d, want 0", count)
	}
	if len(created.SupportedAdapters) != 0 {
		t.Errorf("返回的 SupportedAdapters 长度 = %d, want 0", len(created.SupportedAdapters))
	}
}

// TestProviderService_UpdateByID_ReplaceAdapters 更新时先删旧关联再插新
func TestProviderService_UpdateByID_ReplaceAdapters(t *testing.T) {
	db := setupProviderServiceTestDB(t)
	svc := newProviderServiceWithTestDB(db)
	ctx := context.Background()

	imap := createTestAdapter(t, db, "imap")
	gmail := createTestAdapter(t, db, "gmail")
	graph := createTestAdapter(t, db, "graph")

	created, err := svc.Create(ctx, newTestProvider("test_update"), []int64{imap.ID})
	if err != nil {
		t.Fatalf("Create() error = %v, wantErr nil", err)
	}

	// 更新为 gmail -> graph，旧关联（imap）应被删除
	updated, err := svc.UpdateByID(ctx, created.ID, newTestProvider("test_update"), []int64{gmail.ID, graph.ID})
	if err != nil {
		t.Fatalf("UpdateByID() error = %v, wantErr nil", err)
	}

	if count := countProviderAdapters(t, db, created.ID); count != 2 {
		t.Errorf("provider_adapters 行数 = %d, want 2", count)
	}

	// 旧关联已删、新关联按请求顺序编号
	if len(updated.SupportedAdapters) != 2 {
		t.Fatalf("返回的 SupportedAdapters 长度 = %d, want 2", len(updated.SupportedAdapters))
	}
	if updated.SupportedAdapters[0].AdapterID != gmail.ID || updated.SupportedAdapters[0].Priority != 0 {
		t.Errorf("第一个关联 = (adapter_id=%d, priority=%d), want (adapter_id=%d, priority=0)",
			updated.SupportedAdapters[0].AdapterID, updated.SupportedAdapters[0].Priority, gmail.ID)
	}
	if updated.SupportedAdapters[1].AdapterID != graph.ID || updated.SupportedAdapters[1].Priority != 1 {
		t.Errorf("第二个关联 = (adapter_id=%d, priority=%d), want (adapter_id=%d, priority=1)",
			updated.SupportedAdapters[1].AdapterID, updated.SupportedAdapters[1].Priority, graph.ID)
	}
	for _, pa := range updated.SupportedAdapters {
		if pa.AdapterID == imap.ID {
			t.Errorf("旧关联 imap 未被删除")
		}
	}
}

// TestProviderService_UpdateByID_EmptyAdapterIDs_Clears 更新时空/nil AdapterIDs 清空关联（合法操作）
func TestProviderService_UpdateByID_EmptyAdapterIDs_Clears(t *testing.T) {
	db := setupProviderServiceTestDB(t)
	svc := newProviderServiceWithTestDB(db)
	ctx := context.Background()

	imap := createTestAdapter(t, db, "imap")
	created, err := svc.Create(ctx, newTestProvider("test_clear"), []int64{imap.ID})
	if err != nil {
		t.Fatalf("Create() error = %v, wantErr nil", err)
	}

	// 空切片：清空全部关联
	if _, err := svc.UpdateByID(ctx, created.ID, newTestProvider("test_clear"), []int64{}); err != nil {
		t.Fatalf("UpdateByID() error = %v, wantErr nil", err)
	}
	if count := countProviderAdapters(t, db, created.ID); count != 0 {
		t.Errorf("空切片更新后 provider_adapters 行数 = %d, want 0", count)
	}

	// 重新关联后再用 nil 更新：同样清空
	if _, err := svc.UpdateByID(ctx, created.ID, newTestProvider("test_clear"), []int64{imap.ID}); err != nil {
		t.Fatalf("UpdateByID() error = %v, wantErr nil", err)
	}
	if _, err := svc.UpdateByID(ctx, created.ID, newTestProvider("test_clear"), nil); err != nil {
		t.Fatalf("UpdateByID() error = %v, wantErr nil", err)
	}
	if count := countProviderAdapters(t, db, created.ID); count != 0 {
		t.Errorf("nil 更新后 provider_adapters 行数 = %d, want 0", count)
	}
}

// TestProviderService_UpdateByID_InvalidAdapterID 适配器 ID 不存在时报错且不破坏已有关联（防外键悬挂）
func TestProviderService_UpdateByID_InvalidAdapterID(t *testing.T) {
	db := setupProviderServiceTestDB(t)
	svc := newProviderServiceWithTestDB(db)
	ctx := context.Background()

	imap := createTestAdapter(t, db, "imap")
	created, err := svc.Create(ctx, newTestProvider("test_invalid"), []int64{imap.ID})
	if err != nil {
		t.Fatalf("Create() error = %v, wantErr nil", err)
	}

	// 不存在的适配器 ID：更新必须失败
	if _, err := svc.UpdateByID(ctx, created.ID, newTestProvider("test_invalid"), []int64{99999}); err == nil {
		t.Fatal("UpdateByID() error = nil, want 适配器不存在的错误")
	}

	// 失败的更新不得影响已有关联
	if count := countProviderAdapters(t, db, created.ID); count != 1 {
		t.Errorf("失败更新后 provider_adapters 行数 = %d, want 1（原关联保持��变）", count)
	}
}

// TestProviderService_Create_DuplicateAdapterIDs 重复的适配器 ID 去重后落库（避免复合主键冲突）
func TestProviderService_Create_DuplicateAdapterIDs(t *testing.T) {
	db := setupProviderServiceTestDB(t)
	svc := newProviderServiceWithTestDB(db)
	ctx := context.Background()

	imap := createTestAdapter(t, db, "imap")
	gmail := createTestAdapter(t, db, "gmail")

	created, err := svc.Create(ctx, newTestProvider("test_dup"), []int64{imap.ID, gmail.ID, imap.ID})
	if err != nil {
		t.Fatalf("Create() error = %v, wantErr nil", err)
	}

	if count := countProviderAdapters(t, db, created.ID); count != 2 {
		t.Errorf("provider_adapters 行数 = %d, want 2（重复 ID 已去重）", count)
	}
}
