package database

import (
	"fmt"
	"log"
	"time"

	"fusionmail/config"
	"fusionmail/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库实例
var DB *gorm.DB

// Initialize 初始化数据库连接
func Initialize(cfg *config.DatabaseConfig) error {
	dsn := cfg.GetDSN()

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
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

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	DB = db
	log.Println("Database connection established successfully")

	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	log.Println("Starting database auto migration...")

	// 定义所有需要迁移的模型
	models := []interface{}{
		// &model.User{}, // 暂时移除，有迁移问题
		&model.Account{},
		&model.Email{},
		&model.EmailAttachment{},
		&model.EmailLabel{},
		&model.EmailLabelRelation{},
		&model.EmailRule{},
		&model.Webhook{},
		&model.WebhookLog{},
		&model.SyncLog{},
		&model.APIKey{},
	}

	// 执行自动迁移
	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Println("Database auto migration completed successfully")

	// 创建全文搜索索引（PostgreSQL 特定）
	if err := createFullTextSearchIndex(); err != nil {
		log.Printf("Warning: failed to create full-text search index: %v", err)
		// 不返回错误，因为这不是致命的
	}

	return nil
}

// createFullTextSearchIndex 创建全文搜索索引
func createFullTextSearchIndex() error {
	log.Println("Creating full-text search index...")

	// 检查索引是否已存在
	var exists bool
	err := DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE indexname = 'idx_emails_fulltext_search'
		)
	`).Scan(&exists).Error

	if err != nil {
		return err
	}

	if exists {
		log.Println("Full-text search index already exists, skipping...")
		return nil
	}

	// 创建全文搜索索引
	sql := `
		CREATE INDEX idx_emails_fulltext_search ON emails 
		USING gin(
			to_tsvector('english', 
				coalesce(subject, '') || ' ' || 
				coalesce(from_name, '') || ' ' || 
				coalesce(text_body, '')
			)
		)
	`

	if err := DB.Exec(sql).Error; err != nil {
		return err
	}

	log.Println("Full-text search index created successfully")
	return nil
}

// SeedInitialData 添加初始数据（如果需要）
func SeedInitialData() error {
	log.Println("Checking for initial data...")

	// 暂时跳过初始数据检查，因为 User 模型有问题
	// TODO: 修复 User 模型后重新启用
	log.Println("Initial data seeding skipped (User model disabled)")

	// 这里可以添加初始数据
	// 例如：创建默认管理员用户、默认标签等
	// 目前暂时不添加初始数据

	log.Println("Initial data seeding completed")
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
