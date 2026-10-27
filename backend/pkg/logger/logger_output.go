package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

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
