package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ResolveLogDir 解析日志目录：
// 1) 环境变量 LOG_DIR
// 2) 若存在 /data（Fly.io 数据卷），使用 /data/logs
// 3) 否则使用 fallback（本地开发一般为 ../logs）
func ResolveLogDir(fallback string) string {
	if dir := strings.TrimSpace(os.Getenv("LOG_DIR")); dir != "" {
		return dir
	}
	if _, err := os.Stat("/data"); err == nil {
		return filepath.Join("/data", "logs")
	}
	if fallback != "" {
		return fallback
	}
	return "logs"
}

// GetLogDir 返回当前文件日志目录（未启用文件输出时为空）
func GetLogDir() string {
	fileOutputMu.Lock()
	defer fileOutputMu.Unlock()
	return logDirPath
}

// BackendLogPath 返回 backend.log 完整路径
func BackendLogPath() string {
	dir := GetLogDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, backendLogFileName)
}

// AddFileOutput 为所有日志记录器添加文件输出（同时保持 stdout 供容器平台采集）
// 使用 lumberjack 按体积滚动；保留天数可由 UpdateFileRetentionDays 动态调整
func AddFileOutput(logDir string) error {
	return AddFileOutputWithRetention(logDir, DefaultAppLogRetentionDays)
}

// AddFileOutputWithRetention 启用文件日志并设置初始保留天数（-1 表示不按天数删备份）
func AddFileOutputWithRetention(logDir string, retentionDays int) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	logPath := filepath.Join(logDir, backendLogFileName)
	lj := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    DefaultAppLogMaxSizeMB,
		MaxBackups: DefaultAppLogMaxBackups,
		LocalTime:  true,
		Compress:   false, // 保持明文，便于 /api/v1/logs 与人工排查
	}
	applyRetentionToLumberjack(lj, retentionDays)

	fileOutputMu.Lock()
	if fileOutput != nil {
		_ = fileOutput.Close()
	}
	fileOutput = lj
	logDirPath = logDir
	fileOutputMu.Unlock()

	// 所有通过 New/NewWithModule 创建的 logger 共享 globalOutput
	globalOutput.Set(io.MultiWriter(os.Stdout, lj))
	return nil
}

func applyRetentionToLumberjack(lj *lumberjack.Logger, retentionDays int) {
	if lj == nil {
		return
	}
	if retentionDays < 0 {
		// 0 = 不按年龄删除（lumberjack 语义）
		lj.MaxAge = 0
		lj.MaxBackups = DefaultAppLogMaxBackups
		return
	}
	if retentionDays == 0 {
		// 0 天：尽快丢掉备份，主文件仍受 MaxSize 约束
		lj.MaxAge = 1
		lj.MaxBackups = 1
		return
	}
	lj.MaxAge = retentionDays
	backups := retentionDays + 1
	if backups < 3 {
		backups = 3
	}
	if backups > 30 {
		backups = 30
	}
	lj.MaxBackups = backups
}

// UpdateFileRetentionDays 根据系统设置更新滚动保留策略（无需重启）
// days < 0：永不按天数删除备份；days >= 0：按天数与备份数上限清理
func UpdateFileRetentionDays(days int) {
	fileOutputMu.Lock()
	defer fileOutputMu.Unlock()
	if fileOutput == nil {
		return
	}
	applyRetentionToLumberjack(fileOutput, days)
}

// CleanupRotatedAppLogs 清理超过保留天数的滚动日志备份，并触发 lumberjack 自身清理逻辑。
// 返回删除的文件数。retentionDays < 0 时不删除。
func CleanupRotatedAppLogs(retentionDays int) (int, error) {
	fileOutputMu.Lock()
	dir := logDirPath
	lj := fileOutput
	fileOutputMu.Unlock()

	if dir == "" {
		return 0, nil
	}

	// 检查主文件大小，超过阈值时强制滚动，使其变为备份文件
	// 这样主文件不会无限增长，滚动后的备份会被保留天数策略清理
	if lj != nil {
		if fi, err := os.Stat(lj.Filename); err == nil {
			if fi.Size() >= int64(DefaultAppLogMaxSizeMB)*1024*1024 {
				_ = lj.Rotate()
			}
		}
	}

	if retentionDays < 0 {
		return 0, nil
	}

	// 触发 lumberjack 按 MaxAge/MaxBackups 清理（通过空转 Rotate 不合适）；
	// 直接扫描目录删除过期备份更可控。
	UpdateFileRetentionDays(retentionDays)

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	// retentionDays==0 时 cutoff≈now，几乎删掉所有旧备份
	if retentionDays == 0 {
		cutoff = time.Now()
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("读取日志目录失败: %w", err)
	}

	deleted := 0
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		// 主文件 backend.log 不删；只清 lumberjack 备份：backend-*.log / backend.log.*
		if name == backendLogFileName || name == "frontend.log" {
			continue
		}
		if !isRotatedBackendLogName(name) {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, name)
			if err := os.Remove(path); err != nil {
				return deleted, fmt.Errorf("删除过期日志 %s: %w", name, err)
			}
			deleted++
		}
	}

	// 同步 lumberjack 内部状态：下次写入时按新 MaxAge 生效
	_ = lj
	return deleted, nil
}

// isRotatedBackendLogName 判断是否为 backend 日志的滚动备份名
func isRotatedBackendLogName(name string) bool {
	// lumberjack LocalTime 命名类似：backend-2026-08-06T10-18-47.123.log
	if strings.HasPrefix(name, "backend-") && strings.HasSuffix(name, ".log") {
		return true
	}
	// 兼容 backend.log.1 / backend.log.20260806 等
	if strings.HasPrefix(name, backendLogFileName+".") {
		return true
	}
	return false
}

// ClearBackendLog 清空当前 backend.log（用于管理页「清空日志」）
// 关闭后写入空文件，下次日志写入时 lumberjack 会重新打开
func ClearBackendLog() error {
	fileOutputMu.Lock()
	defer fileOutputMu.Unlock()

	path := ""
	if fileOutput != nil {
		path = fileOutput.Filename
		_ = fileOutput.Close()
	} else if logDirPath != "" {
		path = filepath.Join(logDirPath, backendLogFileName)
	}
	if path == "" {
		return fmt.Errorf("文件日志未启用")
	}
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		return fmt.Errorf("清空日志文件失败: %w", err)
	}
	return nil
}
