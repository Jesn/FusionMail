package integration

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openSQLiteMemoryDB 为集成测试创建一个隔离的内存 SQLite 数据库。
//
// 每个测试用例使用以测试名命名的独立内存库，避免并发测试之间互相污染。
//
// 必须使用 cache=shared：GORM 走连接池，而 cache=private 会让每条连接拿到各自
// 独立的内存库——AutoMigrate 建在连接 A、后续查询落到连接 B，就会报
// "no such table"。共享缓存 + 以测试名命名的唯一库名同时满足隔离与可见性。
//
// 外键约束保持 SQLite 默认（关闭）：这些用例是业务逻辑属性测试，fixture 只构造
// 被测聚合所需的最少字段（例如��接构造 EmailAccount 而不建 Provider/Adapter），
// 打开外键会因 fixture 不完整而失败，掩盖真正要验证的属性。外键与约束行为
// 应由接入真实 PostgreSQL 的用例覆盖。
//
// 注意：本 helper 供 *testing.T 与 *testing.B 共用，因此参数类型为 testing.TB。
// 生产环境使用 PostgreSQL，SQLite 无法覆盖 PG 专有行为（部分唯一索引、
// jsonb、GIN 全文索引、ILIKE、DISTINCT ON 等），相关用例需另行接入真实 PG。
func openSQLiteMemoryDB(tb testing.TB) *gorm.DB {
	tb.Helper()

	dbName := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=10000", tb.Name())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		tb.Fatalf("打开内存测试数据库失败: %v", err)
	}

	// SQLite 只有一个写者；共享缓存下多连接并发写会返回
	// "database table is locked"（-race 放慢执行后尤其明显）。
	// 收敛到单连接串行化访问，行为可预测且不再抖动。
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		tb.Cleanup(func() { _ = sqlDB.Close() })
	}

	return db
}
