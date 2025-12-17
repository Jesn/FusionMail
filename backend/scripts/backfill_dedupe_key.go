//go:build ignore
// +build ignore

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Email 简化的邮件模型（仅包含迁移需要的字段）
type Email struct {
	ID          int64     `gorm:"primaryKey"`
	AccountUID  string    `gorm:"size:64"`
	MessageID   string    `gorm:"size:255"`
	FromAddress string    `gorm:"size:255"`
	Subject     string    `gorm:"type:text"`
	SentAt      time.Time `gorm:"not null"`
	DedupeKey   string    `gorm:"size:64"`
}

func (Email) TableName() string {
	return "emails"
}

// 系统通知域名列表
var systemNotificationDomains = []string{
	"@139.com",
	"@10086.cn",
	"@10086.com",
	"@chinamobile.com",
}

func main() {
	// 从环境变量获取数据库连接
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL 环境变量未设置")
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	ctx := context.Background()

	// 统计需要迁移的邮件数量
	var totalCount int64
	if err := db.WithContext(ctx).Model(&Email{}).Where("dedupe_key IS NULL OR dedupe_key = ''").Count(&totalCount).Error; err != nil {
		log.Fatalf("统计邮件数量失败: %v", err)
	}

	fmt.Printf("需要迁移的邮件数量: %d\n", totalCount)

	if totalCount == 0 {
		fmt.Println("没有需要迁移的邮件")
		return
	}

	// 分批处理
	batchSize := 1000
	processed := 0
	startTime := time.Now()

	for {
		var emails []Email
		if err := db.WithContext(ctx).
			Where("dedupe_key IS NULL OR dedupe_key = ''").
			Limit(batchSize).
			Find(&emails).Error; err != nil {
			log.Fatalf("查询邮件失败: %v", err)
		}

		if len(emails) == 0 {
			break
		}

		// 批量更新
		for _, email := range emails {
			dedupeKey := generateDedupeKey(email.MessageID, email.FromAddress, email.Subject, email.SentAt)
			if err := db.WithContext(ctx).Model(&Email{}).Where("id = ?", email.ID).Update("dedupe_key", dedupeKey).Error; err != nil {
				log.Printf("更新邮件 %d 失败: %v", email.ID, err)
				continue
			}
		}

		processed += len(emails)
		elapsed := time.Since(startTime)
		rate := float64(processed) / elapsed.Seconds()
		remaining := float64(int(totalCount)-processed) / rate

		fmt.Printf("进度: %d/%d (%.1f%%), 速率: %.1f/s, 预计剩余: %.0fs\n",
			processed, totalCount, float64(processed)/float64(totalCount)*100, rate, remaining)
	}

	fmt.Printf("迁移完成! 共处理 %d 封邮件, 耗时 %v\n", processed, time.Since(startTime))

	// 验证
	var remainingCount int64
	if err := db.WithContext(ctx).Model(&Email{}).Where("dedupe_key IS NULL OR dedupe_key = ''").Count(&remainingCount).Error; err != nil {
		log.Printf("验证失败: %v", err)
	} else {
		fmt.Printf("验证: 剩余未迁移邮件数量: %d\n", remainingCount)
	}
}

// generateDedupeKey 生成去重标识
func generateDedupeKey(messageID, fromAddress, subject string, sentAt time.Time) string {
	// 检查是否为系统通知域名
	if isSystemNotificationDomain(fromAddress) {
		return generateHashKey(fromAddress, subject, sentAt)
	}

	// 有 Message-ID 时使用 mid: 前缀
	if messageID != "" {
		return generateMessageIDKey(messageID)
	}

	// 无 Message-ID 时使用 hash: 前缀
	return generateHashKey(fromAddress, subject, sentAt)
}

// isSystemNotificationDomain 检查是否为系统通知域名
func isSystemNotificationDomain(fromAddress string) bool {
	fromLower := strings.ToLower(fromAddress)
	for _, domain := range systemNotificationDomains {
		if strings.HasSuffix(fromLower, strings.ToLower(domain)) {
			return true
		}
	}
	return false
}

// generateMessageIDKey 生成基于 Message-ID 的去重标识
func generateMessageIDKey(messageID string) string {
	cleanID := strings.TrimSpace(messageID)
	cleanID = strings.TrimPrefix(cleanID, "<")
	cleanID = strings.TrimSuffix(cleanID, ">")

	if len(cleanID) > 60 {
		hash := sha256.Sum256([]byte(cleanID))
		return "mid:" + hex.EncodeToString(hash[:])[:32]
	}

	return "mid:" + cleanID
}

// generateHashKey 生成基于 hash 的去重标识
func generateHashKey(fromAddress, subject string, sentAt time.Time) string {
	from := strings.ToLower(strings.TrimSpace(fromAddress))
	subj := strings.TrimSpace(subject)
	timeStr := sentAt.UTC().Format("2006-01-02T15:04")

	data := fmt.Sprintf("%s|%s|%s", from, subj, timeStr)
	hash := sha256.Sum256([]byte(data))

	return "hash:" + hex.EncodeToString(hash[:])[:32]
}
