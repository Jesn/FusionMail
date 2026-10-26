package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // 注册 file:// source driver
)

// ErrNoChange 表示数据库已是最新版本（golang-migrate 的哨兵错误）。
var ErrNoChange = migrate.ErrNoChange

// DefaultMigrationsDir 是版本化迁移文件的默认目录（相对 backend 工作目录）。
const DefaultMigrationsDir = "migrations"

// ResolveMigrationsDir 定���迁移目录。
//
// 迁移文件随镜像一起分发，但进程的工作目录可能是 /app 或 backend/，
// 因此按候选顺序探测，避免依赖调用方的 cwd。
func ResolveMigrationsDir() (string, error) {
	candidates := []string{
		DefaultMigrationsDir,
		filepath.Join("backend", DefaultMigrationsDir),
		filepath.Join("..", DefaultMigrationsDir),
	}
	if env := os.Getenv("MIGRATIONS_DIR"); env != "" {
		candidates = append([]string{env}, candidates...)
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, absErr := filepath.Abs(c)
			if absErr != nil {
				return c, nil
			}
			return abs, nil
		}
	}
	return "", fmt.Errorf("未找到迁移目录（已尝试 %v）；可用 MIGRATIONS_DIR 显式指定", candidates)
}

// newMigrator 构造 golang-migrate 实例。
//
// 复用已初始化的 *sql.DB，避免再开一条独立连接池；
// 迁移记录写入 schema_migrations 表（由 golang-migrate 维护）。
func newMigrator(migrationsDir string) (*migrate.Migrate, error) {
	if DB == nil {
		return nil, errors.New("数据库未初始化，请先调用 database.Initialize")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 *sql.DB 失败: %w", err)
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return nil, fmt.Errorf("构造 postgres 迁移驱动失败: %w", err)
	}

	sourceURL := "file://" + migrationsDir
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("构造迁移实例失败（source=%s）: %w", sourceURL, err)
	}
	return m, nil
}

// RunVersionedMigrations 应用全部待执行的版本化迁移。
//
// 这是 schema 的唯一变更入口。返回 ErrNoChange 表示已是最新版本。
func RunVersionedMigrations(migrationsDir string) error {
	m, err := newMigrator(migrationsDir)
	if err != nil {
		return err
	}
	// 注意：不调用 m.Close()，它会关闭共享的 *sql.DB 连接池。

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrNoChange
		}
		return fmt.Errorf("应用迁移失败: %w", err)
	}
	return nil
}

// VersionedMigrationStatus 返回当前 schema 版本与 dirty 标记。
//
// dirty=true 表示上一次迁移中途失败，必须人工介入后才能继续。
func VersionedMigrationStatus(migrationsDir string) (version uint, dirty bool, err error) {
	m, mErr := newMigrator(migrationsDir)
	if mErr != nil {
		return 0, false, mErr
	}
	v, d, vErr := m.Version()
	if vErr != nil {
		if errors.Is(vErr, migrate.ErrNilVersion) {
			return 0, false, nil // 尚未执行过任何迁移
		}
		return 0, false, fmt.Errorf("读取迁移版本失败: %w", vErr)
	}
	return v, d, nil
}

// RollbackVersionedMigrations 回滚 n 个版本（n<=0 表示全部回滚）。
//
// 仅用于测试环境重建；生产回滚请使用备份恢复。
func RollbackVersionedMigrations(migrationsDir string, n int) error {
	m, err := newMigrator(migrationsDir)
	if err != nil {
		return err
	}
	if n <= 0 {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("���滚全部迁移失败: %w", err)
		}
		return nil
	}
	if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("回滚 %d 个版本失败: %w", n, err)
	}
	return nil
}
