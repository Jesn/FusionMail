package pgtest_test

import (
	"os"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/testutil/pgtest"
)

func TestMain(m *testing.M) {
	pgtest.Init("pgtest")
	os.Exit(m.Run())
}

// TestFixtureCreatesAndResetsDatabase 验证夹具有效性：
// AutoMigrate 建出表 → 写入可见 → Truncate 后归零。
//
// 这是夹具自身的守卫：夹具坏了，后面所有 PG 用例都会以误导性的方式失败。
func TestFixtureCreatesAndResetsDatabase(t *testing.T) {
	db := pgtest.DB(t)
	pgtest.Truncate(t)

	email := &model.Email{
		ProviderID:  "fixture-1",
		AccountUID:  "acct-fixture",
		Subject:     "夹具冒烟",
		FromAddress: "fixture@example.com",
		SentAt:      time.Now(),
		ReceivedAt:  time.Now(),
	}
	if err := db.Create(email).Error; err != nil {
		t.Fatalf("写入邮件失败: %v", err)
	}
	if email.ID == 0 {
		t.Fatal("写入后主键仍为 0，AutoMigrate 可能没建出 emails 表")
	}

	var before int64
	if err := db.Model(&model.Email{}).Count(&before).Error; err != nil {
		t.Fatalf("计数失败: %v", err)
	}
	if before != 1 {
		t.Fatalf("写入后 emails 行数 = %d，期望 1", before)
	}

	pgtest.Truncate(t)

	var after int64
	if err := db.Model(&model.Email{}).Count(&after).Error; err != nil {
		t.Fatalf("计数失败: %v", err)
	}
	if after != 0 {
		t.Fatalf("Truncate 后 emails 行数 = %d，期望 0", after)
	}
}

// TestTruncateKeepsSeedTables 保证默认 Truncate 不会把 AutoMigrate 灌入的种子
// 数据（providers/adapters/settings）一起清掉——清掉会让依赖种子数据的用例
// 以"账户找不到适配器"这种无关原因失败。
func TestTruncateKeepsSeedTables(t *testing.T) {
	db := pgtest.DB(t)
	pgtest.Truncate(t)

	var providers int64
	if err := db.Table("providers").Count(&providers).Error; err != nil {
		t.Fatalf("统计 providers 失败: %v", err)
	}
	if providers == 0 {
		t.Fatal("默认 Truncate 清掉了种子表 providers；应只清业务表")
	}
}

// TestDedupeKeyUniqueIndexIsReal 验证 PG 才能体现的约束确实生效：
// (provider_id, account_uid) 部分唯一索引在 SQLite 上不存在，
// 这类行为是本夹具存在的理由。
func TestDedupeKeyUniqueIndexIsReal(t *testing.T) {
	db := pgtest.DB(t)
	pgtest.Truncate(t, "emails")

	first := &model.Email{
		ProviderID:  "dup-1",
		AccountUID:  "acct-dup",
		Subject:     "首次",
		FromAddress: "a@example.com",
		SentAt:      time.Now(),
		ReceivedAt:  time.Now(),
	}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}

	dup := *first
	dup.ID = 0
	if err := db.Create(&dup).Error; err == nil {
		t.Fatal("重复 (provider_id, account_uid) 写入应当被唯一索引拒绝，但成功了")
	}
}
