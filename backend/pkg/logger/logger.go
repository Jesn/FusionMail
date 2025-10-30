package logger

import (
	"log"
	"os"
)

// Logger 简单的日志记录器
type Logger struct {
	*log.Logger
}

// New 创建新的日志记录器
func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[FusionMail] ", log.LstdFlags|log.Lshortfile),
	}
}

// Info 记录信息日志
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Printf("[INFO] "+msg, args...)
}

// Error 记录错误日志
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Printf("[ERROR] "+msg, args...)
}

// Warn 记录警告日志
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Printf("[WARN] "+msg, args...)
}

// Debug 记录调试日志
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Printf("[DEBUG] "+msg, args...)
}
