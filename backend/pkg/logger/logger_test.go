package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveLogDirPrefersEnv(t *testing.T) {
	t.Setenv("LOG_DIR", "/custom/logs")
	got := ResolveLogDir("/fallback/logs")
	if got != "/custom/logs" {
		t.Fatalf("ResolveLogDir = %q, want /custom/logs", got)
	}
}

func TestResolveLogDirFallbackWhenNoEnv(t *testing.T) {
	t.Setenv("LOG_DIR", "")
	// 无 LOG_DIR 时：若本机无 /data 则走 fallback
	if _, err := os.Stat("/data"); err == nil {
		t.Skip("本机存在 /data，跳过 fallback 断言")
	}
	got := ResolveLogDir("/fallback/logs")
	if got != "/fallback/logs" {
		t.Fatalf("ResolveLogDir = %q, want /fallback/logs", got)
	}
}

func resetFileOutputForTest() {
	globalOutput.Set(os.Stdout)
	fileOutputMu.Lock()
	if fileOutput != nil {
		_ = fileOutput.Close()
		fileOutput = nil
	}
	logDirPath = ""
	fileOutputMu.Unlock()
}

func TestAddFileOutputWritesForModuleAndPackageLoggers(t *testing.T) {
	t.Cleanup(resetFileOutputForTest)

	// 在 AddFileOutput 之前创建的模块 logger 也必须能落到文件
	early := NewWithModule("EarlyModule")

	dir := t.TempDir()
	if err := AddFileOutput(dir); err != nil {
		t.Fatalf("AddFileOutput: %v", err)
	}

	early.Info("early module message")
	late := NewWithModule("LateModule")
	late.Info("late module message")
	Info("package level message")

	data, err := os.ReadFile(filepath.Join(dir, "backend.log"))
	if err != nil {
		t.Fatalf("读取 backend.log 失败: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"early module message",
		"late module message",
		"package level message",
		"[EarlyModule]",
		"[LateModule]",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("backend.log 缺少 %q:\n%s", want, content)
		}
	}
}

func TestCleanupRotatedAppLogsDeletesOldBackups(t *testing.T) {
	t.Cleanup(resetFileOutputForTest)

	dir := t.TempDir()
	if err := AddFileOutputWithRetention(dir, 7); err != nil {
		t.Fatalf("AddFileOutputWithRetention: %v", err)
	}

	// 模拟 lumberjack 风格的过期备份
	oldBackup := filepath.Join(dir, "backend-2020-01-01T00-00-00.000.log")
	if err := os.WriteFile(oldBackup, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// 修改 mtime 为很久以前
	oldTime := time.Now().AddDate(0, 0, -30)
	if err := os.Chtimes(oldBackup, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// 近期备份应保留
	recentBackup := filepath.Join(dir, "backend-2099-01-01T00-00-00.000.log")
	if err := os.WriteFile(recentBackup, []byte("recent\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 主文件不应被删
	mainLog := filepath.Join(dir, "backend.log")
	if err := os.WriteFile(mainLog, []byte("main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	deleted, err := CleanupRotatedAppLogs(7)
	if err != nil {
		t.Fatalf("CleanupRotatedAppLogs: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted=%d, want 1", deleted)
	}
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Fatalf("过期备份应已删除")
	}
	if _, err := os.Stat(recentBackup); err != nil {
		t.Fatalf("近期备份应保留: %v", err)
	}
	if _, err := os.Stat(mainLog); err != nil {
		t.Fatalf("主日志文件应保留: %v", err)
	}
}

func TestClearBackendLog(t *testing.T) {
	t.Cleanup(resetFileOutputForTest)

	dir := t.TempDir()
	if err := AddFileOutput(dir); err != nil {
		t.Fatal(err)
	}
	Info("to be cleared")
	if err := ClearBackendLog(); err != nil {
		t.Fatalf("ClearBackendLog: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "backend.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("清空后应为空, got %q", string(data))
	}
	// 清空后仍可写入
	Info("after clear")
	data, err = os.ReadFile(filepath.Join(dir, "backend.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "after clear") {
		t.Fatalf("清空后写入失败: %s", string(data))
	}
}

func TestFormatMessageSupportsStructuredFields(t *testing.T) {
	log := NewWithModule("Test")
	args := []interface{}{
		"provider", "google",
		"code_length", 12,
		"error", errors.New("boom"),
	}
	message := log.formatMessage(LevelInfo, "OAuth2 callback received",
		args...,
	)

	if strings.Contains(message, "%!(EXTRA") {
		t.Fatalf("结构化日志不应按 printf 参数渲染: %s", message)
	}
	for _, want := range []string{
		"OAuth2 callback received",
		"provider=google",
		"code_length=12",
		"error=boom",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("日志缺少 %q: %s", want, message)
		}
	}
}

func TestFormatMessageKeepsPrintfStyle(t *testing.T) {
	log := NewWithModule("Test")
	message := log.formatMessage(LevelWarn, "连接失败: count=%d, err=%v", 3, errors.New("boom"))

	if strings.Contains(message, "%!(EXTRA") {
		t.Fatalf("printf 风格日志不应产生额外参数: %s", message)
	}
	for _, want := range []string{
		"连接失败: count=3, err=boom",
		"[WARN][Test]",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("日志缺少 %q: %s", want, message)
		}
	}
}
