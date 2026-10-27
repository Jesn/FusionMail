// Package pgtest 提供基于真实 PostgreSQL 的测试夹具。
//
// 为什么不用 SQLite：父任务的三个地基错误之一是「集成测试跑 SQLite、生产跑
// PostgreSQL」——ILIKE / jsonb / 部分唯一索引 / DISTINCT ON / 外键约束这些
// 行为在 SQLite 上测不出来，回归保护是假的。
//
// 隔离策略（见 .trellis/tasks/10-27-2.0-p19-core-flow-tests/design.md D1）：
//
//   - 每个测试包一个独立库 fm_test_<pkg>，建库前先 DROP（清理上次崩溃的残留）
//   - 首次 DB() 调用时才建库 + AutoMigrate，且只做一次（lazy + sync.Once），
//     不做 DB 用例的包完全不受影响
//   - 每个用例用 Truncate 清理自己写入的表；种子表（providers/adapters/
//     provider_adapters/settings）默认保留，避免把 AutoMigrate 灌进来的
//     种子数据一起清掉
//   - PostgreSQL 不可达时 DB() 直接 t.Skip（不是 fail）：CI 没有 PG 服务，
//     跳过才能让质量门禁保持绿，跳过原因会打印出来
//
// 用法规约：
//
//	func TestMain(m *testing.M) { pgtest.Init("service"); os.Exit(m.Run()) }
//
//	func TestXxx(t *testing.T) {
//	    db := pgtest.DB(t)          // 不可达 → t.Skip
//	    pgtest.Truncate(t, "emails") // 清掉本用例会写入的表
//	    ...
//	}
//
// 注意：database.DB 是包级全局，夹具会改写它，因此 DB 用例禁止 t.Parallel()。
package pgtest

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"fusionmail/pkg/database"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	// DBNamePrefix 是测试库统一前缀，和人工库（fm_baseline/fm_automigrate）区分开。
	DBNamePrefix = "fm_test_"

	// pingTimeout 是判定 PG 是否可达的超时。CI 里没有 PG 时会白等这么久，
	// 所以保持 1 秒级。
	pingTimeout = 1500 * time.Millisecond

	defaultHost = "127.0.0.1"
	defaultPort = "5432"
	defaultUser = "fusionmail"
	// defaultPassword 与 .trellis/spec/backend/testing.md 记录的本地测试库一致。
	defaultPassword = "fusionmail_test_pw"
)

// seedTables 是 AutoMigrate 会灌入种子数据的表，默认不参与 Truncate。
// 需要清它们的用例显式传表名即可。
var seedTables = map[string]bool{
	"providers":         true,
	"adapters":          true,
	"provider_adapters": true,
	"settings":          true,
}

var (
	initOnce sync.Once
	pkgName  string

	stateMu sync.Mutex
	current *state
)

type state struct {
	db  *gorm.DB
	dsn string
	err error
}

// Init 声明当前测试包的库名后缀，应在 TestMain 里调用。
// 不调用时回退到测试二进制文件名（<pkg>.test → <pkg>）。
func Init(pkg string) {
	if pkg != "" {
		pkgName = pkg
	}
}

// DB 返回当前包的测试库连接。PostgreSQL 不可达时调用 t.Skip。
func DB(tb testing.TB) *gorm.DB {
	tb.Helper()

	st := ensure()
	if st.err != nil {
		tb.Skipf("跳过 PostgreSQL 用例：%v（设置 DB_HOST/DB_USER/DB_PASSWORD 或 TEST_DATABASE_URL，见 .trellis/spec/backend/testing.md）", st.err)
	}
	return st.db
}

// Available 报告 PG 测试库是否可用，不可用时返回原因。
func Available() (bool, error) {
	st := ensure()
	return st.err == nil, st.err
}

func ensure() *state {
	initOnce.Do(func() {
		stateMu.Lock()
		defer stateMu.Unlock()
		current = setup()
	})
	stateMu.Lock()
	defer stateMu.Unlock()
	return current
}

func setup() *state {
	if os.Getenv("TEST_PG_DISABLE") == "1" {
		return &state{err: fmt.Errorf("TEST_PG_DISABLE=1")}
	}

	adminURL, err := adminURL()
	if err != nil {
		return &state{err: err}
	}

	admin, err := open(adminURL.String(), "postgres")
	if err != nil {
		return &state{err: fmt.Errorf("连接 PostgreSQL 失败: %w", err)}
	}
	if sqlDB, dbErr := admin.DB(); dbErr == nil {
		defer sqlDB.Close()
	}

	dbName := DBNamePrefix + sanitize(packageName())

	// 上一次运行可能崩溃留下连接，先踢掉再 DROP，否则 DROP 会被占用的库阻塞。
	if err := admin.Exec(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()",
		dbName,
	).Error; err != nil {
		return &state{err: fmt.Errorf("清理旧连接失败: %w", err)}
	}
	if err := admin.Exec("DROP DATABASE IF EXISTS " + quoteIdent(dbName)).Error; err != nil {
		return &state{err: fmt.Errorf("删除旧测试库 %s 失败: %w", dbName, err)}
	}
	if err := admin.Exec("CREATE DATABASE " + quoteIdent(dbName)).Error; err != nil {
		return &state{err: fmt.Errorf("创建测试库 %s 失败: %w", dbName, err)}
	}

	target := *adminURL
	target.Path = "/" + dbName
	dsn := target.String()

	db, err := open(dsn, dbName)
	if err != nil {
		return &state{err: fmt.Errorf("打开测试库 %s 失败: %w", dbName, err)}
	}

	// AutoMigrate 走全局 database.DB（pkg/database 的既有契约），
	// 这里显式接管以免复制一份模型清单当第二真相。
	database.DB = db
	if err := database.AutoMigrate(); err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return &state{err: fmt.Errorf("AutoMigrate 失败: %w", err)}
	}

	return &state{db: db, dsn: dsn}
}

// DSN 返回当前测试库的连接串（调试用）。
func DSN(tb testing.TB) string {
	tb.Helper()
	return ensure().dsn
}

// open 连接并 ping 指定库。
func open(dsn, label string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("%s 不可达: %w", label, err)
	}
	return db, nil
}

// adminURL 解析管理连接：优先 TEST_DATABASE_URL，其次 DB_* 拼装。
// 管理连接固定落在 postgres 库上，因为建库/删库不能连着自己。
func adminURL() (*url.URL, error) {
	if raw := os.Getenv("TEST_DATABASE_URL"); raw != "" {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("TEST_DATABASE_URL 非法: %w", err)
		}
		u.Path = "/postgres"
		return u, nil
	}

	host := firstEnv("DB_HOST", defaultHost)
	port := firstEnv("DB_PORT", defaultPort)
	user := firstEnv("DB_USER", defaultUser)
	pass := firstEnv("DB_PASSWORD", defaultPassword)
	sslmode := firstEnv("DB_SSLMODE", "disable")

	return &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     net.JoinHostPort(host, port),
		Path:     "/postgres",
		RawQuery: "sslmode=" + url.QueryEscape(sslmode),
	}, nil
}

// Truncate 清空指定表（RESTART IDENTITY CASCADE）。
// 不传表名时清空当前 schema 下除种子表外的所有表。
func Truncate(tb testing.TB, tables ...string) {
	tb.Helper()
	db := DB(tb)

	if len(tables) == 0 {
		tables = tableNames(tb, db)
	}
	if len(tables) == 0 {
		return
	}

	quoted := make([]string, 0, len(tables))
	for _, name := range tables {
		quoted = append(quoted, quoteIdent(name))
	}
	stmt := "TRUNCATE TABLE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE"
	if err := db.Exec(stmt).Error; err != nil {
		tb.Fatalf("清空表失败 (%s): %v", stmt, err)
	}
}

// tableNames 列出当前 schema 下需要清空的表（跳过种子表）。
func tableNames(tb testing.TB, db *gorm.DB) []string {
	tb.Helper()

	var all []string
	if err := db.Raw(
		"SELECT tablename FROM pg_tables WHERE schemaname = current_schema() ORDER BY tablename",
	).Scan(&all).Error; err != nil {
		tb.Fatalf("列出表失败: %v", err)
	}

	kept := make([]string, 0, len(all))
	for _, name := range all {
		if seedTables[name] {
			continue
		}
		kept = append(kept, name)
	}
	return kept
}

func packageName() string {
	if pkgName != "" {
		return pkgName
	}
	base := filepath.Base(os.Args[0])
	return strings.TrimSuffix(base, ".test")
}

// sanitize 把包名压成合法库名片段。
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "pkg"
	}
	return out
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func firstEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
