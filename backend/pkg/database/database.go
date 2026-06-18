package database

import (
	"fmt"
	"time"

	"fusionmail/config"
	"fusionmail/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 模块日志记录器
var log = logger.NewWithModule("Database")

// gormLogWriter 自定义 GORM 日志写入器，将日志输出到项目日志系统
type gormLogWriter struct {
	log *logger.Logger
}

// Printf 实现 gormlogger.Writer 接口
func (w *gormLogWriter) Printf(format string, args ...interface{}) {
	w.log.Warn(format, args...)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DB 全局数据库实例
var DB *gorm.DB

// Initialize 初始化数据库连接
func Initialize(cfg *config.DatabaseConfig) error {
	dsn := cfg.GetDSN()

	// 打印DSN信息（隐藏密码）
	hiddenDSN := fmt.Sprintf("host=%s port=%s user=%s password=*** dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.DBName, cfg.SSLMode)
	log.Info("正在连接数据库: %s", hiddenDSN)
	log.Debug("数据库配置: DisablePrepareStmt=%v, MaxIdleConns=%d, MaxOpenConns=%d, ConnMaxLifetime=%d min",
		cfg.DisablePrepareStmt, cfg.MaxIdleConns, cfg.MaxOpenConns, cfg.ConnMaxLifetime)

	// 配置 GORM 日志（含慢查询监控）
	// 慢查询阈值：200ms，超过此时间的查询会被记录为 Warn 级别
	gormLogConfig := gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond, // 慢查询阈值
		LogLevel:                  gormlogger.Warn,        // 日志级别
		IgnoreRecordNotFoundError: true,                   // 忽略记录未找到错误
		Colorful:                  false,                  // 生产环境关闭颜色
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.New(
			&gormLogWriter{log: log}, // 使用自定义日志写入器
			gormLogConfig,
		),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// 禁用 PrepareStmt 以支持 Supabase Transaction 模式（端口 6543）
		// Transaction 模式的连接池不支持 prepared statements
		PrepareStmt: !cfg.DisablePrepareStmt,
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 设置连接池参数（从配置读取，支持 Transaction 模式优化）
	// Transaction 模式建议：MaxIdleConns=2-5, MaxOpenConns=10-20
	// Session 模式/直连：MaxIdleConns=10, MaxOpenConns=100
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 100
	}
	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 60
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)

	DB = db
	log.Info("数据库连接成功 (PrepareStmt=%v)", !cfg.DisablePrepareStmt)

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
