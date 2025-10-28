package main

import (
	"flag"
	"log"
	"os"

	"fusionmail/config"
	"fusionmail/pkg/database"
)

func main() {
	// 定义命令行参数
	action := flag.String("action", "up", "Migration action: up (migrate) or status (check)")
	flag.Parse()

	log.Println("FusionMail Database Migration Tool")
	log.Printf("Action: %s", *action)

	// 加载配置
	cfg := config.Load()
	log.Printf("Database: %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 初始化数据库连接
	if err := database.Initialize(&cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 执行迁移操作
	switch *action {
	case "up":
		log.Println("Running database migrations...")
		if err := database.AutoMigrate(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}

		log.Println("Seeding initial data...")
		if err := database.SeedInitialData(); err != nil {
			log.Fatalf("Seeding failed: %v", err)
		}

		log.Println("Migration completed successfully!")

	case "status":
		log.Println("Checking database status...")
		db := database.GetDB()

		// 检查数据库连接
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("Failed to get database instance: %v", err)
		}

		if err := sqlDB.Ping(); err != nil {
			log.Fatalf("Database connection failed: %v", err)
		}

		log.Println("Database connection: OK")

		// 检查表是否存在
		tables := []string{
			"users", "accounts", "emails", "email_attachments",
			"email_labels", "email_label_relations", "email_rules",
			"webhooks", "webhook_logs", "sync_logs", "api_keys",
		}

		for _, table := range tables {
			var exists bool
			err := db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)", table).Scan(&exists).Error
			if err != nil {
				log.Printf("Table %s: ERROR (%v)", table, err)
			} else if exists {
				log.Printf("Table %s: EXISTS", table)
			} else {
				log.Printf("Table %s: NOT FOUND", table)
			}
		}

	default:
		log.Fatalf("Unknown action: %s (use 'up' or 'status')", *action)
	}

	os.Exit(0)
}
