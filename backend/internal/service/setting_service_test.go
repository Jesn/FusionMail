package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// settingTestEnv 是配置服务用例的接线：真实 PostgreSQL 仓库 + 无 Redis 的本地缓存。
type settingTestEnv struct {
	svc *SettingService
	db  *gorm.DB
}

func newSettingTestEnv(t *testing.T) *settingTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	return &settingTestEnv{
		svc: NewSettingService(repository.NewSettingRepository(db), settingTestRedis(t), stubEncryptor{}),
		db:  db,
	}
}

// settingTestRedis 返回一个可用的 Redis 客户端（独立 DB，用前清空）。
//
// 配置服务的写路径必须落 Redis 缓存（cache.Set 直接调 redis），nil 客户端会 panic，
// 因此这里用真实 Redis；连不上时跳过用例而不是失败——CI 容器里没有 Redis 服务。
func settingTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	// 用独立 DB，避免 FlushDB 影响本机其它用途的数据。
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 15})

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("Redis 不可用（%s），跳过配置服务缓存用例: %v", addr, err)
	}
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("清空测试 Redis 库失败: %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })
	return client
}

// cleanupSettingRow 删除用例写入的配置��。
// settings 是受保护的种子表（pgtest.Truncate 不会清它），必须自己收拾干净。
func cleanupSettingRow(t *testing.T, db *gorm.DB, userID *int64, category, key string) {
	t.Helper()

	t.Cleanup(func() {
		query := db.Unscoped().Where("category = ? AND key = ?", category, key)
		if userID == nil {
			query = query.Where("user_id IS NULL")
		} else {
			query = query.Where("user_id = ?", *userID)
		}
		query.Delete(&model.Setting{})
	})
}

// TestSettingSetGetRoundTrip 覆盖非敏感配置的写入与读取。
func TestSettingSetGetRoundTrip(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()

	cleanupSettingRow(t, env.db, nil, "ui", "default_view")
	if err := env.svc.Set(ctx, nil, "ui", "default_view", "card", &GetOptions{}); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	got, err := env.svc.Get(ctx, nil, "ui", "default_view", &GetOptions{})
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	if got != "card" {
		t.Errorf("配置值 = %q，期望 card", got)
	}

	// 不存在的配置返回空字符串而不是错误（调用方靠空值走默认值）
	got, err = env.svc.Get(ctx, nil, "ui", "not_exists_key", &GetOptions{})
	if err != nil {
		t.Fatalf("读取不存在的配置不应报错: %v", err)
	}
	if got != "" {
		t.Errorf("不存在的配置 = %q，期望空字符串", got)
	}
}

// TestSettingSensitiveValueIsEncryptedAtRest 覆盖敏感配置：
// 落库必须是密文，读取���解密；未显式要求时按分类接口不得泄露。
func TestSettingSensitiveValueIsEncryptedAtRest(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()

	cleanupSettingRow(t, env.db, nil, "security", "jwt_secret")
	if err := env.svc.Set(ctx, nil, "security", "jwt_secret", "super-secret", &GetOptions{}); err != nil {
		t.Fatalf("写入敏感配置失败: %v", err)
	}

	var stored model.Setting
	if err := env.db.Where("user_id IS NULL AND category = ? AND key = ?", "security", "jwt_secret").
		First(&stored).Error; err != nil {
		t.Fatalf("读取落库配置失败: %v", err)
	}
	if !stored.IsSensitive {
		t.Error("jwt_secret 未被标记为敏感配置")
	}
	if stored.Value == "super-secret" {
		t.Error("敏感配置以明文落库")
	}
	if !strings.HasPrefix(stored.Value, "enc:") {
		t.Errorf("敏感配置未经过加密器: %q", stored.Value)
	}

	got, err := env.svc.Get(ctx, nil, "security", "jwt_secret", &GetOptions{IncludeSensitive: true})
	if err != nil {
		t.Fatalf("读取敏感配置失败: %v", err)
	}
	if got != "super-secret" {
		t.Errorf("解密���果 = %q，期望 super-secret", got)
	}

	// 分类接口默认不返回敏感配置
	all, err := env.svc.GetByCategory(ctx, nil, "security", &GetOptions{})
	if err != nil {
		t.Fatalf("按分类读取失败: %v", err)
	}
	if _, leaked := all["jwt_secret"]; leaked {
		t.Error("未要求敏感配置时仍返回了 jwt_secret")
	}

	all, err = env.svc.GetByCategory(ctx, nil, "security", &GetOptions{IncludeSensitive: true})
	if err != nil {
		t.Fatalf("按分类读取（含敏感）失败: %v", err)
	}
	if all["jwt_secret"] != "super-secret" {
		t.Errorf("含敏感读取结果 = %q，期望解密后的明文", all["jwt_secret"])
	}
}

// TestSettingGetByCategoryPrefersUserValue 覆盖用户级覆盖系统级：
// 同一个 key 同时存在系统级与用户级时，用户级必须��出。
func TestSettingGetByCategoryCachesSensitiveValuesOnce(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()

	cleanupSettingRow(t, env.db, nil, "security", "webhook_secret")
	// 直接落库（绕过 Set/refreshCache），保证缓存是冷的
	row := &model.Setting{
		Category:    "security",
		Key:         "webhook_secret",
		Value:       "enc:hook-secret",
		ValueType:   "string",
		IsSensitive: true,
	}
	if err := env.db.Create(row).Error; err != nil {
		t.Fatalf("写入敏感配置失败: %v", err)
	}

	// 第一次：缓存未命中，走数据库路径
	first, err := env.svc.GetByCategory(ctx, nil, "security", &GetOptions{IncludeSensitive: true})
	if err != nil {
		t.Fatalf("首次读取失败: %v", err)
	}
	if first["webhook_secret"] != "hook-secret" {
		t.Fatalf("首次读取 = %q，期望解密后的明文", first["webhook_secret"])
	}

	// 第二次：命中缓存，必须得到同样的明文。
	// 缓存里存的应当是数据库中的密文���只加密一次），否则这里只会解密一层。
	second, err := env.svc.GetByCategory(ctx, nil, "security", &GetOptions{IncludeSensitive: true})
	if err != nil {
		t.Fatalf("二次读取失败: %v", err)
	}
	if second["webhook_secret"] != "hook-secret" {
		t.Errorf("命中缓存后 = %q，期望解密后的明文（缓存写入了重复加密的值）", second["webhook_secret"])
	}
}

func TestSettingGetByCategoryPrefersUserValue(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()
	userID := int64(4242)

	cleanupSettingRow(t, env.db, nil, "ui", "theme")
	cleanupSettingRow(t, env.db, &userID, "ui", "theme")

	if err := env.svc.Set(ctx, nil, "ui", "theme", "light", &GetOptions{}); err != nil {
		t.Fatalf("写入系统级配置失败: %v", err)
	}
	if err := env.svc.Set(ctx, &userID, "ui", "theme", "dark", &GetOptions{}); err != nil {
		t.Fatalf("写入用户级配置失败: %v", err)
	}

	settings, err := env.svc.GetByCategory(ctx, &userID, "ui", &GetOptions{})
	if err != nil {
		t.Fatalf("按分类读取失败: %v", err)
	}
	if settings["theme"] != "dark" {
		t.Errorf("theme = %q，期望用户级 dark 覆盖系统级 light", settings["theme"])
	}

	// 系统级视角��受用户级影响
	systemSettings, err := env.svc.GetByCategory(ctx, nil, "ui", &GetOptions{})
	if err != nil {
		t.Fatalf("系统级读取失败: %v", err)
	}
	if systemSettings["theme"] != "light" {
		t.Errorf("系统级 theme = %q，期望 light", systemSettings["theme"])
	}
}

// TestSettingGetPublicFiltersNonPublic 覆盖公开配置过滤：
// 只有白名单里的 key 才算公开，其余一律不返回。
func TestSettingGetPublicFiltersNonPublic(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()

	cleanupSettingRow(t, env.db, nil, "ui", "theme")
	cleanupSettingRow(t, env.db, nil, "ui", "internal_flag")

	if err := env.svc.Set(ctx, nil, "ui", "theme", "dark", &GetOptions{}); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	if err := env.svc.Set(ctx, nil, "ui", "internal_flag", "1", &GetOptions{}); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	public, err := env.svc.GetPublic(ctx, nil, "ui")
	if err != nil {
		t.Fatalf("读取公开配置失败: %v", err)
	}
	if public["theme"] != "dark" {
		t.Errorf("公开配置缺少 theme: %v", public)
	}
	if _, leaked := public["internal_flag"]; leaked {
		t.Error("非公开配置被返回")
	}

	// 单个配置的公开校验
	if _, err := env.svc.Get(ctx, nil, "ui", "internal_flag", &GetOptions{OnlyPublic: true}); err == nil {
		t.Error("以 OnlyPublic 读取非公开配置应报错")
	}
}

// TestSettingBatchSetAndDeleteByCategory 覆盖批量写入与整类删除。
func TestSettingBatchSetAndDeleteByCategory(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()
	category := "ui"

	for _, key := range []string{"theme", "language", "email_page_size"} {
		cleanupSettingRow(t, env.db, nil, category, key)
	}

	if err := env.svc.BatchSet(ctx, nil, category, map[string]string{
		"theme":           "dark",
		"language":        "en-US",
		"email_page_size": "100",
	}, &GetOptions{}); err != nil {
		t.Fatalf("批量写入失败: %v", err)
	}

	settings, err := env.svc.GetByCategory(ctx, nil, category, &GetOptions{})
	if err != nil {
		t.Fatalf("按分类读取失败: %v", err)
	}
	if settings["theme"] != "dark" || settings["language"] != "en-US" || settings["email_page_size"] != "100" {
		t.Errorf("批量写入结果错误: %v", settings)
	}

	if err := env.svc.Delete(ctx, nil, category, "language"); err != nil {
		t.Fatalf("删除配置失败: %v", err)
	}
	got, err := env.svc.Get(ctx, nil, category, "language", &GetOptions{})
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	if got != "" {
		t.Errorf("删除后仍读到 %q", got)
	}

	if err := env.svc.DeleteByCategory(ctx, nil, category); err != nil {
		t.Fatalf("删除分类失败: %v", err)
	}
	settings, err = env.svc.GetByCategory(ctx, nil, category, &GetOptions{})
	if err != nil {
		t.Fatalf("按分类读取失败: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("整类删除后仍有 %d 项: %v", len(settings), settings)
	}
}

// TestSettingResetUsesDefaultValue 覆盖重置为默认值：没有默认值的配置必须报错。
func TestSettingResetUsesDefaultValue(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()

	cleanupSettingRow(t, env.db, nil, "ui", "theme")
	if err := env.svc.Set(ctx, nil, "ui", "theme", "dark", &GetOptions{}); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	if err := env.svc.Reset(ctx, nil, "ui", "theme"); err != nil {
		t.Fatalf("重置配置失败: %v", err)
	}
	got, err := env.svc.Get(ctx, nil, "ui", "theme", &GetOptions{})
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	if got != "light" {
		t.Errorf("重置后 = %q，期望默认值 light", got)
	}

	if err := env.svc.Reset(ctx, nil, "ui", "no_default_key"); err == nil {
		t.Error("没有默认值的配置重置应报错")
	}
}

// TestSettingSearchAndStats 覆盖搜索与统计接口。
func TestSettingSearchAndStats(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()

	cleanupSettingRow(t, env.db, nil, "ui", "theme")
	if err := env.svc.Set(ctx, nil, "ui", "theme", "dark", &GetOptions{}); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	results, err := env.svc.Search(ctx, nil, "theme", "ui", false)
	if err != nil {
		t.Fatalf("搜索配置失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("搜索 theme 没有结果")
	}
	for _, setting := range results {
		if setting.Key != "theme" {
			t.Errorf("搜索结果包含无关项: %+v", setting)
		}
	}

	stats := env.svc.GetStats()
	if stats["service_status"] != "active" {
		t.Errorf("service_status = %v，期望 active", stats["service_status"])
	}
	if _, ok := stats["cache"]; !ok {
		t.Error("统计信息缺少 cache 字段")
	}
}

// TestSettingSystemAndUserHelpers 覆盖系统级/用户级便捷方法。
func TestSettingSystemAndUserHelpers(t *testing.T) {
	env := newSettingTestEnv(t)
	ctx := context.Background()
	userID := int64(777)

	cleanupSettingRow(t, env.db, nil, "ui", "default_view")
	cleanupSettingRow(t, env.db, &userID, "ui", "default_view")

	if err := env.svc.SetSystem(ctx, "ui", "default_view", "list"); err != nil {
		t.Fatalf("写入系统配置失败: %v", err)
	}
	if got, err := env.svc.GetSystem(ctx, "ui", "default_view"); err != nil || got != "list" {
		t.Errorf("GetSystem = %q,%v，期望 list", got, err)
	}

	byCategory, err := env.svc.GetSystemByCategory(ctx, "ui")
	if err != nil {
		t.Fatalf("按分类读取系统配置失败: %v", err)
	}
	if byCategory["default_view"] != "list" {
		t.Errorf("系统分类配置 = %v", byCategory)
	}

	if err := env.svc.SetUser(ctx, userID, "ui", "default_view", "card"); err != nil {
		t.Fatalf("写入用户配置失败: %v", err)
	}
	if got, err := env.svc.GetUser(ctx, userID, "ui", "default_view"); err != nil || got != "card" {
		t.Errorf("GetUser = %q,%v，期望 card", got, err)
	}

	if err := env.svc.BatchSetSystem(ctx, "ui", map[string]string{"default_view": "grid"}); err != nil {
		t.Fatalf("批量写入系统配置失败: %v", err)
	}
	if got, _ := env.svc.GetSystem(ctx, "ui", "default_view"); got != "grid" {
		t.Errorf("批量写入后 = %q，期望 grid", got)
	}
}

// TestSettingClassificationHelpers 覆盖敏感/公开/类型/默认值的判定表。
func TestSettingClassificationHelpers(t *testing.T) {
	svc := &SettingService{}

	sensitiveCases := []struct {
		category, key string
		want          bool
	}{
		{category: "security", key: "jwt_secret", want: true},
		{category: "oauth", key: "gmail_client_secret", want: true},
		{category: "smtp", key: "smtp_password", want: true},
		{category: "api", key: "secret_keys", want: true},
		{category: "security", key: "session_timeout", want: false},
		{category: "unknown", key: "jwt_secret", want: false},
	}
	for _, c := range sensitiveCases {
		if got := svc.isSensitive(c.category, c.key); got != c.want {
			t.Errorf("isSensitive(%s,%s) = %v，期望 %v", c.category, c.key, got, c.want)
		}
	}

	publicCases := []struct {
		category, key string
		want          bool
	}{
		{category: "ui", key: "theme", want: true},
		{category: "sync", key: "sync_interval", want: true},
		{category: "notification", key: "enable_email_notification", want: true},
		{category: "ui", key: "internal_flag", want: false},
		{category: "unknown", key: "theme", want: false},
	}
	for _, c := range publicCases {
		if got := svc.isPublic(c.category, c.key); got != c.want {
			t.Errorf("isPublic(%s,%s) = %v，期望 %v", c.category, c.key, got, c.want)
		}
	}

	typeCases := []struct {
		key  string
		want string
	}{
		{key: "enable_auto_sync", want: "boolean"},
		{key: "email_page_size", want: "number"},
		{key: "secret_keys", want: "json"},
		{key: "theme", want: "string"},
	}
	for _, c := range typeCases {
		if got := svc.getValueType("ui", c.key); got != c.want {
			t.Errorf("getValueType(%s) = %q，期望 %q", c.key, got, c.want)
		}
	}

	if got := svc.getDefaultValue("ui", "theme"); got != "light" {
		t.Errorf("getDefaultValue(ui,theme) = %q，期望 light", got)
	}
	if got := svc.getDefaultValue("ui", "no_such_key"); got != "" {
		t.Errorf("未知配置默认值 = %q，期望空字符串", got)
	}
	if got := svc.getDefaultValue("unknown", "theme"); got != "" {
		t.Errorf("未知分类默认值 = %q，期望空字符串", got)
	}
}
