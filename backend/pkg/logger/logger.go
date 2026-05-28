package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	logger    *log.Logger
	level     LogLevel
	module    string
	mu        sync.RWMutex
	fields    map[string]interface{}
	calldepth int
}

// 全局默认日志记录器
var (
	defaultLogger *Logger
	once          sync.Once
)

// GetDefault 获取默认日志记录器（单例）
func GetDefault() *Logger {
	once.Do(func() {
		defaultLogger = New()
	})
	return defaultLogger
}

// New 创建新的日志记录器
func New() *Logger {
	level := LevelInfo
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		if l, ok := levelFromString[strings.ToLower(envLevel)]; ok {
			level = l
		}
	}
	return &Logger{
		logger:    log.New(os.Stdout, "", 0),
		level:     level,
		fields:    make(map[string]interface{}),
		calldepth: 3,
	}
}

// NewWithModule 创建带模块名的日志记录器
func NewWithModule(module string) *Logger {
	l := New()
	l.module = module
	return l
}

// SetOutput 设置日志输出目标
func (l *Logger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
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
		logger:    l.logger,
		level:     l.level,
		module:    module,
		fields:    copyFields(l.fields),
		calldepth: l.calldepth,
	}
}

// WithField 添加单个字段
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newFields := copyFields(l.fields)
	newFields[key] = value
	return &Logger{
		logger:    l.logger,
		level:     l.level,
		module:    l.module,
		fields:    newFields,
		calldepth: l.calldepth,
	}
}

// WithFields 添加多个字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := copyFields(l.fields)
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		logger:    l.logger,
		level:     l.level,
		module:    l.module,
		fields:    newFields,
		calldepth: l.calldepth,
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

func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	fmt.Println(l.formatMessage(level, msg, args...))
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
	fmt.Printf(format+"\n", args...)
}

// Println 兼容标准库
func (l *Logger) Println(args ...interface{}) {
	if !l.shouldLog(LevelInfo) {
		return
	}
	fmt.Println(args...)
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
