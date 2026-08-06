package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel 日志级别
type LogLevel int

const (
	// LevelDebug 调试级别 - 详细的调试信息，生产环境应关闭
	LevelDebug LogLevel = iota
	// LevelInfo 信息级别 - 常规运行信息
	LevelInfo
	// LevelWarn 警告级别 - 潜在问题，但不影响运行
	LevelWarn
	// LevelError 错误级别 - 错误信息，需要关注
	LevelError
	// LevelFatal 致命级别 - 严重错误，程序可能无法继续
	LevelFatal
)

// 日志级别名称映射
var levelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// 从字符串解析日志级别
var levelFromString = map[string]LogLevel{
	"debug": LevelDebug,
	"info":  LevelInfo,
	"warn":  LevelWarn,
	"error": LevelError,
	"fatal": LevelFatal,
}

// Logger 结构化日志记录器
type Logger struct {
	logger     *log.Logger
	level      LogLevel
	module     string
	mu         sync.RWMutex
	fields     map[string]interface{}
	calldepth  int
	jsonFormat bool
}

// switchableWriter 可在运行时切换底层 Writer，使包级 init 创建的 logger 也能在启动后写文件
type switchableWriter struct {
	mu sync.RWMutex
	w  io.Writer
}

func (s *switchableWriter) Write(p []byte) (int, error) {
	s.mu.RLock()
	w := s.w
	s.mu.RUnlock()
	return w.Write(p)
}

func (s *switchableWriter) Set(w io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.w = w
}

// 应用日志文件滚动默认参数
const (
	// DefaultAppLogMaxSizeMB 单个日志文件最大体积（MB），超过后滚动
	DefaultAppLogMaxSizeMB = 50
	// DefaultAppLogMaxBackups 最多保留的滚动备份数
	DefaultAppLogMaxBackups = 14
	// DefaultAppLogRetentionDays 默认保留天数
	DefaultAppLogRetentionDays = 7
	// backendLogFileName 主日志文件名（/api/v1/logs 读取此文件）
	backendLogFileName = "backend.log"
)

// 全局默认日志记录器与共享输出
var (
	defaultLogger *Logger
	once          sync.Once
	globalOutput  = &switchableWriter{w: os.Stdout}
	fileOutputMu  sync.Mutex
	// fileOutput 使用 lumberjack，按大小/天数滚动
	fileOutput *lumberjack.Logger
	logDirPath string
)

// GetDefault 获取默认日志记录器（单例）
func GetDefault() *Logger {
	once.Do(func() {
		defaultLogger = New()
	})
	return defaultLogger
}

// New 创建新的日志记录器（共享 globalOutput，保证后续 AddFileOutput 对全部 logger 生效）
func New() *Logger {
	level := LevelInfo
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		if l, ok := levelFromString[strings.ToLower(envLevel)]; ok {
			level = l
		}
	}
	jsonFormat := false
	if fmt := strings.ToLower(os.Getenv("LOG_FORMAT")); fmt == "json" {
		jsonFormat = true
	}
	return &Logger{
		logger:     log.New(globalOutput, "", 0),
		level:      level,
		fields:     make(map[string]interface{}),
		calldepth:  3,
		jsonFormat: jsonFormat,
	}
}

// NewWithModule 创建带模块名的日志记录器
func NewWithModule(module string) *Logger {
	l := New()
	l.module = module
	return l
}

// SetOutput 设置单个日志记录器的输出目标（一般优先用 AddFileOutput）
func (l *Logger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
}

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

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetLevelFromString 从字符串设置日志级别
func (l *Logger) SetLevelFromString(levelStr string) {
	if level, ok := levelFromString[strings.ToLower(levelStr)]; ok {
		l.SetLevel(level)
	}
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// WithModule 创建带模块名的子日志记录器
func (l *Logger) WithModule(module string) *Logger {
	return &Logger{
		logger:     l.logger,
		level:      l.level,
		module:     module,
		fields:     copyFields(l.fields),
		calldepth:  l.calldepth,
		jsonFormat: l.jsonFormat,
	}
}

// WithField 添加单个字段
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newFields := copyFields(l.fields)
	newFields[key] = value
	return &Logger{
		logger:     l.logger,
		level:      l.level,
		module:     l.module,
		fields:     newFields,
		calldepth:  l.calldepth,
		jsonFormat: l.jsonFormat,
	}
}

// WithFields 添加多个字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := copyFields(l.fields)
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		logger:     l.logger,
		level:      l.level,
		module:     l.module,
		fields:     newFields,
		calldepth:  l.calldepth,
		jsonFormat: l.jsonFormat,
	}
}

// WithRequestID 添加 Request ID 到日志
func (l *Logger) WithRequestID(c *gin.Context) *Logger {
	requestID := ""
	if rid, exists := c.Get("request_id"); exists {
		requestID = rid.(string)
	}
	return l.WithField("request_id", requestID)
}

func copyFields(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (l *Logger) shouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

func getFileName(file string) string {
	parts := strings.Split(file, "/")
	return parts[len(parts)-1]
}

func hasPrintfVerb(format string) bool {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i >= len(format) {
			return false
		}
		if format[i] == '%' {
			continue
		}

		if format[i] == '[' {
			for i < len(format) && format[i] != ']' {
				i++
			}
			if i >= len(format) {
				return false
			}
			i++
		}

		for i < len(format) && strings.ContainsRune("#0+- ", rune(format[i])) {
			i++
		}
		for i < len(format) && ((format[i] >= '0' && format[i] <= '9') || format[i] == '*') {
			i++
		}
		if i < len(format) && format[i] == '.' {
			i++
			for i < len(format) && ((format[i] >= '0' && format[i] <= '9') || format[i] == '*') {
				i++
			}
		}
		if i < len(format) && format[i] == '[' {
			for i < len(format) && format[i] != ']' {
				i++
			}
			if i >= len(format) {
				return false
			}
			i++
		}
		if i < len(format) && strings.ContainsRune("vTtbcdoOqxXUeEfFgGspw", rune(format[i])) {
			return true
		}
	}

	return false
}

func formatPrintfMessage(format string, args []interface{}) string {
	return fmt.Sprintf(format, args...)
}

func appendStructuredArgs(sb *strings.Builder, args []interface{}) {
	for i := 0; i < len(args); i += 2 {
		key := fmt.Sprintf("arg%d", i+1)
		value := args[i]
		if keyArg, ok := args[i].(string); ok && keyArg != "" && i+1 < len(args) {
			key = keyArg
			value = args[i+1]
		}

		sb.WriteString(fmt.Sprintf(" %s=%v", key, value))
	}
}

func (l *Logger) formatMessage(level LogLevel, msg string, args ...interface{}) string {
	var sb strings.Builder
	sb.WriteString(time.Now().Format("2006/01/02 15:04:05"))
	sb.WriteString(" [")
	sb.WriteString(levelNames[level])
	sb.WriteString("]")
	if l.module != "" {
		sb.WriteString("[")
		sb.WriteString(l.module)
		sb.WriteString("]")
	}
	sb.WriteString(" ")
	hasStructuredArgs := len(args) > 0 && !hasPrintfVerb(msg)
	if len(args) > 0 && !hasStructuredArgs {
		sb.WriteString(formatPrintfMessage(msg, args))
	} else {
		sb.WriteString(msg)
	}
	if len(l.fields) > 0 || hasStructuredArgs {
		sb.WriteString(" |")
		for k, v := range l.fields {
			sb.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
		if hasStructuredArgs {
			appendStructuredArgs(&sb, args)
		}
	}
	if _, file, line, ok := runtime.Caller(l.calldepth); ok {
		sb.WriteString(fmt.Sprintf(" (%s:%d)", getFileName(file), line))
	}
	return sb.String()
}

func (l *Logger) formatJSONMessage(level LogLevel, msg string, args ...interface{}) string {
	hasStructuredArgs := len(args) > 0 && !hasPrintfVerb(msg)
	if len(args) > 0 && !hasStructuredArgs {
		msg = formatPrintfMessage(msg, args)
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"level":     levelNames[level],
		"message":   msg,
	}
	if l.module != "" {
		entry["module"] = l.module
	}
	for k, v := range l.fields {
		entry[k] = v
	}
	if hasStructuredArgs {
		for i := 0; i < len(args); i += 2 {
			key := fmt.Sprintf("arg%d", i+1)
			value := args[i]
			if keyArg, ok := args[i].(string); ok && keyArg != "" && i+1 < len(args) {
				key = keyArg
				value = args[i+1]
			}
			entry[key] = value
		}
	}
	if _, file, line, ok := runtime.Caller(l.calldepth); ok {
		entry["file"] = getFileName(file)
		entry["line"] = line
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprintf(`{"level":"ERROR","message":"failed to marshal log entry: %v"}`, err)
	}
	return string(data)
}

func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	var output string
	if l.jsonFormat {
		output = l.formatJSONMessage(level, msg, args...)
	} else {
		output = l.formatMessage(level, msg, args...)
	}
	l.logger.Output(2, output)
}

// Debug 记录调试日志
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info 记录信息日志
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn 记录警告日志
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error 记录错误日志
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Fatal 记录致命错误并退出
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(LevelFatal, msg, args...)
	os.Exit(1)
}

// Printf 兼容标准库
func (l *Logger) Printf(format string, args ...interface{}) {
	level := LevelInfo
	upper := strings.ToUpper(format)
	if strings.Contains(upper, "[DEBUG]") {
		level = LevelDebug
	} else if strings.Contains(upper, "[WARN]") {
		level = LevelWarn
	} else if strings.Contains(upper, "[ERROR]") {
		level = LevelError
	}
	if !l.shouldLog(level) {
		return
	}
	l.logger.Output(2, fmt.Sprintf(format, args...))
}

// Println 兼容标准库
func (l *Logger) Println(args ...interface{}) {
	if !l.shouldLog(LevelInfo) {
		return
	}
	l.logger.Output(2, fmt.Sprintln(args...))
}

// ============ 包级别便捷函数 ============

func Debug(msg string, args ...interface{}) {
	GetDefault().Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	GetDefault().Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	GetDefault().Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	GetDefault().Error(msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	GetDefault().Fatal(msg, args...)
}

func SetLevel(level LogLevel) {
	GetDefault().SetLevel(level)
}

func SetLevelFromEnv() {
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		GetDefault().SetLevelFromString(envLevel)
	}
}

func WithRequestID(c *gin.Context) *Logger {
	return GetDefault().WithRequestID(c)
}
