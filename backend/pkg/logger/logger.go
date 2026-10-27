package logger

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"

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
	DefaultAppLogMaxSizeMB = 10
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
