package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// migrationsDir 返回 backend/migrations 的绝对路径。
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
}

// versionedNamePattern 匹配 golang-migrate 的版本化文件命名：{version}_{name}.{up|down}.sql
var versionedNamePattern = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.(up|down)\.sql$`)

// TestMigrationRootOnlyContainsVersionedSQLAndDocs 约束迁移根目录的内容。
//
// 根目录只允许 golang-migrate 的版本化文件与说明文档：迁移器会扫描整个目录，
// 任何非版本化文件都会让 `migrate -action=up` 直接失败。
// 历史 SQL（NNN_name.sql 形式）已归档到 migrations/legacy/，仅供查阅，不再执行。
func TestMigrationRootOnlyContainsVersionedSQLAndDocs(t *testing.T) {
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	allowedDirs := map[string]bool{
		"legacy":      true, // 历史 SQL，已失效，仅存档
		"manual":      true, // 需要人工执行的破坏性/数据搬迁 SQL
		"maintenance": true, // 运维 SQL
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if !allowedDirs[name] {
				t.Errorf("migrations 根目录出现未预期的子目录 %q；版本化迁移必须直接放在根目录", name)
			}
			continue
		}
		if filepath.Ext(name) == ".md" {
			continue
		}
		if versionedNamePattern.MatchString(name) {
			continue
		}
		t.Errorf("migrations 根目录包含非版本化文件 %q；"+
			"版本化迁移须命名为 {version}_{name}.up.sql / .down.sql，"+
			"历史 SQL 请放入 migrations/legacy/", name)
	}
}

// TestMigrationVersionsAreUniqueAndPaired 校验版本号唯一且 up/down 成对。
//
// 缺少 down 的迁移无法回滚；重复版本号会让 golang-migrate 直接报错。
func TestMigrationVersionsAreUniqueAndPaired(t *testing.T) {
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	upVersions := map[string]string{}
	downVersions := map[string]string{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		m := versionedNamePattern.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		version, direction := m[1], m[3]
		// 归一化版本号，避免 "1" 与 "000001" 被当成两个版本
		if n, convErr := strconv.Atoi(version); convErr == nil {
			version = strconv.Itoa(n)
		}

		target := upVersions
		if direction == "down" {
			target = downVersions
		}
		if prev, exists := target[version]; exists {
			t.Fatalf("重复的迁移版本 %s: %s 与 %s", version, prev, entry.Name())
		}
		target[version] = entry.Name()
	}

	if len(upVersions) == 0 {
		t.Fatal("未找到任何版本化 up 迁移；schema 将无法从零建立")
	}

	for version, upFile := range upVersions {
		if _, exists := downVersions[version]; !exists {
			t.Errorf("迁移版本 %s 缺少对应的 .down.sql（%s）；无法回滚", version, upFile)
		}
	}
	for version, downFile := range downVersions {
		if _, exists := upVersions[version]; !exists {
			t.Errorf("迁移版本 %s 只有 .down.sql（%s）而没有 .up.sql", version, downFile)
		}
	}
}

// TestBaselineMigrationIsFirstAndNonEmpty 校验 baseline 存在且是第一个版本。
//
// baseline 是从「GORM AutoMigrate 在干净 PostgreSQL 上产出的真实 schema」
// 导出的起点；它必须存在，否则新环境无法建库。
func TestBaselineMigrationIsFirstAndNonEmpty(t *testing.T) {
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	type versioned struct {
		version int
		name    string
	}
	var versions []versioned
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		m := versionedNamePattern.FindStringSubmatch(entry.Name())
		if m == nil || m[3] != "up" {
			continue
		}
		n, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			continue
		}
		versions = append(versions, versioned{version: n, name: entry.Name()})
	}
	if len(versions) == 0 {
		t.Fatal("未找到任何 up 迁移")
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].version < versions[j].version })

	first := versions[0]
	if !strings.Contains(first.name, "baseline") {
		t.Errorf("第一个迁移应为 baseline（当前为 %s）；"+
			"baseline 是从 AutoMigrate 真实产出导出的 schema 起点", first.name)
	}

	info, statErr := os.Stat(filepath.Join(dir, first.name))
	if statErr != nil {
		t.Fatalf("无法读取 %s: %v", first.name, statErr)
	}
	if info.Size() < 1024 {
		t.Errorf("baseline 迁移仅 %d 字节，疑似不完整（完整 schema 应包含全部表定义）", info.Size())
	}
}

// TestLegacyMigrationsAreArchived 确认历史 SQL 已归档。
//
// 历史 39 个 SQL 迁移实测无法从零建库（001 用了 PostgreSQL 保留字 references
// 作列名导致语法错误，后续级联失败），必须留在 legacy/ 下避免被迁移器扫描。
func TestLegacyMigrationsAreArchived(t *testing.T) {
	dir := migrationsDir(t)
	legacyDir := filepath.Join(dir, "legacy")
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		t.Skipf("legacy 目录不存在，跳过：%v", err)
	}
	if len(entries) == 0 {
		t.Error("legacy 目录为空；历史 SQL 应归档于此以便追溯")
	}
}
