package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// StructuredLogger 结构化日志记录器
type StructuredLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewStructuredLogger 创建新的结构化日志记录器
func NewStructuredLogger(level LogLevel) *StructuredLogger {
	return &StructuredLogger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// log 记录日志
func (sl *StructuredLogger) log(level LogLevel, msg string, fields ...interface{}) {
	if level < sl.level {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)

	// 添加调用者信息
	caller := ""
	if _, file, line, ok := runtime.Caller(2); ok {
		caller = fmt.Sprintf("%s:%d", getFileName(file), line)
	}

	// 格式化输出
	output := fmt.Sprintf("[%s] %s %s", level.String(), timestamp, msg)

	// 添加调用者信息
	if caller != "" {
		output += fmt.Sprintf(" (%s)", caller)
	}

	// 添加字段
	if len(fields) > 0 {
		var fieldStrs []string
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				key := fmt.Sprintf("%v", fields[i])
				value := fields[i+1]
				fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", key, value))
			}
		}
		if len(fieldStrs) > 0 {
			output += fmt.Sprintf(" | %s", strings.Join(fieldStrs, ", "))
		}
	}

	sl.logger.Println(output)
}

// Debug 记录调试日志
func (sl *StructuredLogger) Debug(msg string, fields ...interface{}) {
	sl.log(DEBUG, msg, fields...)
}

// Info 记录信息日志
func (sl *StructuredLogger) Info(msg string, fields ...interface{}) {
	sl.log(INFO, msg, fields...)
}

// Warn 记录警告日志
func (sl *StructuredLogger) Warn(msg string, fields ...interface{}) {
	sl.log(WARN, msg, fields...)
}

// Error 记录错误日志
func (sl *StructuredLogger) Error(msg string, fields ...interface{}) {
	sl.log(ERROR, msg, fields...)
}

// Fatal 记录致命错误日志并退出
func (sl *StructuredLogger) Fatal(msg string, fields ...interface{}) {
	sl.log(FATAL, msg, fields...)
	os.Exit(1)
}

// WithRequestID 添加 Request ID 到日志
func (sl *StructuredLogger) WithRequestID(c *gin.Context) *RequestLogger {
	requestID := ""
	if rid, exists := c.Get("request_id"); exists {
		requestID = rid.(string)
	}
	return &RequestLogger{
		logger:    sl,
		requestID: requestID,
	}
}

// RequestLogger 带 Request ID 的日志记录器
type RequestLogger struct {
	logger    *StructuredLogger
	requestID string
}

// Debug 记录调试日志（带 Request ID）
func (rl *RequestLogger) Debug(msg string, fields ...interface{}) {
	fields = append(fields, "request_id", rl.requestID)
	rl.logger.Debug(msg, fields...)
}

// Info 记录信息日志（带 Request ID）
func (rl *RequestLogger) Info(msg string, fields ...interface{}) {
	fields = append(fields, "request_id", rl.requestID)
	rl.logger.Info(msg, fields...)
}

// Warn 记录警告日志（带 Request ID）
func (rl *RequestLogger) Warn(msg string, fields ...interface{}) {
	fields = append(fields, "request_id", rl.requestID)
	rl.logger.Warn(msg, fields...)
}

// Error 记录错误日志（带 Request ID）
func (rl *RequestLogger) Error(msg string, fields ...interface{}) {
	fields = append(fields, "request_id", rl.requestID)
	rl.logger.Error(msg, fields...)
}

// getFileName 获取文件名（不包含路径）
func getFileName(file string) string {
	parts := strings.Split(file, "/")
	return parts[len(parts)-1]
}

// 全局日志记录器实例
var defaultLogger = NewStructuredLogger(INFO)

// 全局日志函数
func Debug(msg string, fields ...interface{}) {
	defaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...interface{}) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...interface{}) {
	defaultLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...interface{}) {
	defaultLogger.Error(msg, fields...)
}

func Fatal(msg string, fields ...interface{}) {
	defaultLogger.Fatal(msg, fields...)
}

// WithRequestID 创建带 Request ID 的日志记录器
func WithRequestID(c *gin.Context) *RequestLogger {
	return defaultLogger.WithRequestID(c)
}

// SetLevel 设置全局日志级别
func SetLevel(level LogLevel) {
	defaultLogger.level = level
}
