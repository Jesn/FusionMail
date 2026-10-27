package repository

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupBatchCountTestDB 创建批量统计测试用的内存数据库
// 每个测试使用独立的内存数据库，避免数据污染（与 provider_test.go 的 setupTestDB 同风格）
func setupBatchCountTestDB(t *testing.T) *gorm.DB {
	dbName := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	err = db.AutoMigrate(&model.AccountGroup{}, &model.EmailAccount{}, &model.Email{})
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

// createBatchTestGroup 创建测试分组
func createBatchTestGroup(t *testing.T, db *gorm.DB, name string) *model.AccountGroup {
	t.Helper()
	group := &model.AccountGroup{Name: name}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}
	return group
}

// createBatchTestAccount 创建测试账号，groupID 为 nil 表示未分组
func createBatchTestAccount(t *testing.T, db *gorm.DB, uid, email string, groupID *int64, parentUID *string) *model.EmailAccount {
	t.Helper()
	acc := &model.EmailAccount{
		UID:                  uid,
		Email:                email,
		ProviderID:           1,
		AdapterID:            1,
		EncryptedCredentials: "{}",
		GroupID:              groupID,
		ParentAccountUID:     parentUID,
	}
	if err := db.Create(acc).Error; err != nil {
		t.Fatalf("创建测试账号失败: %v", err)
	}
	return acc
}

// createBatchTestEmail 创建测试邮件
func createBatchTestEmail(t *testing.T, db *gorm.DB, id int64, accountUID string, isRead, isDeleted bool) *model.Email {
	t.Helper()
	email := &model.Email{
		ProviderID:  fmt.Sprintf("provider-%d", id),
		AccountUID:  accountUID,
		Subject:     fmt.Sprintf("subject-%d", id),
		FromAddress: "from@example.com",
		SentAt:      time.Now(),
		ReceivedAt:  time.Now(),
		IsRead:      isRead,
		IsDeleted:   isDeleted,
	}
	if err := db.Create(email).Error; err != nil {
		t.Fatalf("创建测试邮件失败: %v", err)
	}
	return email
}

// TestCountAccountsByGroupIDs 批量账号计数：一次 GROUP BY 聚合，口径与单组版本一致
func TestCountAccountsByGroupIDs(t *testing.T) {
	db := setupBatchCountTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	g1 := createBatchTestGroup(t, db, "分组1")
	g2 := createBatchTestGroup(t, db, "分组2")
	g3 := createBatchTestGroup(t, db, "分组3") // 无账号

	// 分组1：2 个直连账号
	createBatchTestAccount(t, db, "acc-1-1", "a11@example.com", &g1.ID, nil)
	createBatchTestAccount(t, db, "acc-1-2", "a12@example.com", &g1.ID, nil)

	// 分组2：1 个直连账号 + 1 个子账户（应排除）+ 1 个软删除账户（应排除）
	createBatchTestAccount(t, db, "acc-2-1", "a21@example.com", &g2.ID, nil)
	createBatchTestAccount(t, db, "acc-2-child", "a2c@example.com", &g2.ID, strPtr("parent-uid"))
	childAcc := createBatchTestAccount(t, db, "acc-2-del", "a2d@example.com", &g2.ID, nil)
	if err := db.Delete(childAcc).Error; err != nil {
		t.Fatalf("软删除测试账号失败: %v", err)
	}

	t.Run("空入参返回空 map", func(t *testing.T) {
		result, err := repo.CountAccountsByGroupIDs(ctx, []int64{})
		if err != nil {
			t.Fatalf("CountAccountsByGroupIDs() error = %v", err)
		}
		if len(result) != 0 {
			t.Errorf("空入参应返回空 map，得到 %v", result)
		}
	})

	t.Run("按分组聚合且排除子账户与软删除账户", func(t *testing.T) {
		result, err := repo.CountAccountsByGroupIDs(ctx, []int64{g1.ID, g2.ID, g3.ID})
		if err != nil {
			t.Fatalf("CountAccountsByGroupIDs() error = %v", err)
		}
		if result[g1.ID] != 2 {
			t.Errorf("分组1 账号数 = %d, want 2", result[g1.ID])
		}
		if result[g2.ID] != 1 {
			t.Errorf("分组2 账号数 = %d, want 1（排除子账户与软删除账户）", result[g2.ID])
		}
		if _, ok := result[g3.ID]; ok {
			t.Errorf("无账号的分组不应出现在结果中，得到 %v", result)
		}
	})

	t.Run("与单组版本口径一致", func(t *testing.T) {
		single1, _ := repo.CountAccountsByGroupID(ctx, g1.ID)
		single2, _ := repo.CountAccountsByGroupID(ctx, g2.ID)
		batch, err := repo.CountAccountsByGroupIDs(ctx, []int64{g1.ID, g2.ID})
		if err != nil {
			t.Fatalf("CountAccountsByGroupIDs() error = %v", err)
		}
		if batch[g1.ID] != single1 || batch[g2.ID] != single2 {
			t.Errorf("批量结果 (%d,%d) 与单组结果 (%d,%d) 不一致", batch[g1.ID], batch[g2.ID], single1, single2)
		}
	})
}

// TestCountUnreadEmailsByGroupIDs 批量未读邮件统计：一次 GROUP BY 聚合，口径与单组版本一致
func TestCountUnreadEmailsByGroupIDs(t *testing.T) {
	db := setupBatchCountTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	g1 := createBatchTestGroup(t, db, "分组1")
	g2 := createBatchTestGroup(t, db, "分组2")
	g3 := createBatchTestGroup(t, db, "分组3") // 无邮件

	acc1 := createBatchTestAccount(t, db, "acc-1", "a1@example.com", &g1.ID, nil)
	acc2 := createBatchTestAccount(t, db, "acc-2", "a2@example.com", &g2.ID, nil)

	// 账号1：2 封未读 + 1 封已读（排除）+ 1 封已删除（排除）
	createBatchTestEmail(t, db, 1, acc1.UID, false, false)
	createBatchTestEmail(t, db, 2, acc1.UID, false, false)
	createBatchTestEmail(t, db, 3, acc1.UID, true, false)
	createBatchTestEmail(t, db, 4, acc1.UID, false, true)

	// 账号2：1 封未读；账号本身软删除后其邮件不应计入
	createBatchTestEmail(t, db, 5, acc2.UID, false, false)
	if err := db.Delete(acc2).Error; err != nil {
		t.Fatalf("软删除测试账号失败: %v", err)
	}

	t.Run("空入参返回空 map", func(t *testing.T) {
		result, err := repo.CountUnreadEmailsByGroupIDs(ctx, []int64{})
		if err != nil {
			t.Fatalf("CountUnreadEmailsByGroupIDs() error = %v", err)
		}
		if len(result) != 0 {
			t.Errorf("空入参应返回空 map，得到 %v", result)
		}
	})

	t.Run("按分组聚合且过滤已读、已删除邮件与软删除账户", func(t *testing.T) {
		result, err := repo.CountUnreadEmailsByGroupIDs(ctx, []int64{g1.ID, g2.ID, g3.ID})
		if err != nil {
			t.Fatalf("CountUnreadEmailsByGroupIDs() error = %v", err)
		}
		if result[g1.ID] != 2 {
			t.Errorf("分组1 未读数 = %d, want 2", result[g1.ID])
		}
		if _, ok := result[g2.ID]; ok {
			t.Errorf("软删除账户的邮件不应计入，得到 %v", result)
		}
		if _, ok := result[g3.ID]; ok {
			t.Errorf("无邮件的分组不应出现在结果中，得到 %v", result)
		}
	})

	t.Run("与单组版本口径一致", func(t *testing.T) {
		single1, _ := repo.CountUnreadEmailsByGroupID(ctx, g1.ID)
		batch, err := repo.CountUnreadEmailsByGroupIDs(ctx, []int64{g1.ID})
		if err != nil {
			t.Fatalf("CountUnreadEmailsByGroupIDs() error = %v", err)
		}
		if batch[g1.ID] != single1 {
			t.Errorf("批量结果 %d 与单组结果 %d 不一致", batch[g1.ID], single1)
		}
	})
}

// TestCountByAccounts 批量按账户统计邮件数量：一次 GROUP BY 聚合，口径与单账户版本一致
func TestCountByAccounts(t *testing.T) {
	db := setupBatchCountTestDB(t)
	repo := NewEmailRepository(db)
	ctx := context.Background()

	createBatchTestAccount(t, db, "acc-a", "a@example.com", nil, nil)
	createBatchTestAccount(t, db, "acc-b", "b@example.com", nil, nil)
	createBatchTestAccount(t, db, "acc-c", "c@example.com", nil, nil) // 无邮件

	// acc-a：2 封（含 1 封 is_deleted=true，与单账户版本口径一致：仅排除软删除记录）
	createBatchTestEmail(t, db, 1, "acc-a", false, false)
	createBatchTestEmail(t, db, 2, "acc-a", true, true)
	// acc-b：1 封正常 + 1 封软删除记录（deleted_at 置位，应排除）
	createBatchTestEmail(t, db, 3, "acc-b", false, false)
	softDeleted := createBatchTestEmail(t, db, 4, "acc-b", false, false)
	if err := db.Delete(softDeleted).Error; err != nil {
		t.Fatalf("软删除测试邮件失败: %v", err)
	}

	t.Run("空入参返回空 map", func(t *testing.T) {
		result, err := repo.CountByAccounts(ctx, []string{})
		if err != nil {
			t.Fatalf("CountByAccounts() error = %v", err)
		}
		if len(result) != 0 {
			t.Errorf("空入参应返回空 map，得到 %v", result)
		}
	})

	t.Run("按账户聚合且排除软删除记录", func(t *testing.T) {
		result, err := repo.CountByAccounts(ctx, []string{"acc-a", "acc-b", "acc-c"})
		if err != nil {
			t.Fatalf("CountByAccounts() error = %v", err)
		}
		if result["acc-a"] != 2 {
			t.Errorf("acc-a 邮件数 = %d, want 2", result["acc-a"])
		}
		if result["acc-b"] != 1 {
			t.Errorf("acc-b 邮件数 = %d, want 1（排除软删除记录）", result["acc-b"])
		}
		if _, ok := result["acc-c"]; ok {
			t.Errorf("无邮件的账户不应出现在结果中，得到 %v", result)
		}
	})

	t.Run("与单账户版本口径一致", func(t *testing.T) {
		singleA, _ := repo.CountByAccount(ctx, "acc-a")
		singleB, _ := repo.CountByAccount(ctx, "acc-b")
		batch, err := repo.CountByAccounts(ctx, []string{"acc-a", "acc-b"})
		if err != nil {
			t.Fatalf("CountByAccounts() error = %v", err)
		}
		if batch["acc-a"] != singleA || batch["acc-b"] != singleB {
			t.Errorf("批量结果 (%d,%d) 与单账户结果 (%d,%d) 不一致", batch["acc-a"], batch["acc-b"], singleA, singleB)
		}
	})
}

// strPtr 返回字符串指针
func strPtr(s string) *string {
	return &s
}
