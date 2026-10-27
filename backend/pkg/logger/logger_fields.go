package logger

import (
	"strings"

	"github.com/gin-gonic/gin"
)

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
