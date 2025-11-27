package service

import (
	"context"
	"fusionmail/internal/repository"
	"log"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

// CleanupService 清理服务
type CleanupService struct {
	accountService AccountService
	settingService *SettingService
	emailRepo      repository.EmailRepository
	cron           *cron.Cron
	isRunning      bool
}

// NewCleanupService 创建清理服务实例
func NewCleanupService(accountService AccountService, settingService *SettingService, emailRepo repository.EmailRepository) *CleanupService {
	return &CleanupService{
		accountService: accountService,
		settingService: settingService,
		emailRepo:      emailRepo,
		cron:           cron.New(),
		isRunning:      false,
	}
}

// Start 启动清理服务
func (s *CleanupService) Start(ctx context.Context) error {
	if s.isRunning {
		log.Println("[CleanupService] Already running")
		return nil
	}

	// 添加定时任务：每天凌晨 2 点执行账户回收站清理
	_, err := s.cron.AddFunc("0 2 * * *", func() {
		s.cleanupTrash(ctx)
	})
	if err != nil {
		return err
	}

	// 添加定时任务：每天凌晨 3 点执行垃圾邮件清理
	_, err = s.cron.AddFunc("0 3 * * *", func() {
		s.cleanupSpamEmails(ctx)
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	s.isRunning = true
	log.Println("[CleanupService] Started successfully, scheduled to run daily at 2:00 AM (trash) and 3:00 AM (spam)")

	// 启动时立即执行一次清理（可选）
	// go s.cleanupTrash(ctx)

	return nil
}

// Stop 停止清理服务
func (s *CleanupService) Stop() {
	if !s.isRunning {
		return
	}

	s.cron.Stop()
	s.isRunning = false
	log.Println("[CleanupService] Stopped")
}

// cleanupTrash 清理回收站
func (s *CleanupService) cleanupTrash(ctx context.Context) {
	log.Println("[CleanupService] Starting trash cleanup...")

	// 获取配置
	value, err := s.settingService.Get(ctx, nil, "system", "trash_auto_cleanup_days", nil)
	if err != nil {
		log.Printf("[CleanupService] Failed to get cleanup setting: %v", err)
		return
	}

	// 如果配置为空，使用默认值 7
	if value == "" {
		value = "7"
	}

	// 解析天数
	days, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("[CleanupService] Invalid cleanup days value: %s, error: %v", value, err)
		return
	}

	// 如果设置为 -1，表示不自动清理
	if days < 0 {
		log.Println("[CleanupService] Auto cleanup is disabled (days=-1)")
		return
	}

	// 执行清理
	cleanedCount, err := s.accountService.CleanupTrash(ctx, days)
	if err != nil {
		log.Printf("[CleanupService] Failed to cleanup trash: %v", err)
		return
	}

	log.Printf("[CleanupService] Trash cleanup completed: cleaned=%d accounts, retention_days=%d", cleanedCount, days)
}

// ManualCleanup 手动触发清理（用于测试或管理接口）
func (s *CleanupService) ManualCleanup(ctx context.Context) (int, error) {
	log.Println("[CleanupService] Manual cleanup triggered")

	// 获取配置
	value, err := s.settingService.Get(ctx, nil, "system", "trash_auto_cleanup_days", nil)
	if err != nil {
		return 0, err
	}

	// 如果配置为空，使用默认值 7
	if value == "" {
		value = "7"
	}

	// 解析天数
	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	// 如果设置为 -1，返回 0
	if days < 0 {
		log.Println("[CleanupService] Auto cleanup is disabled (days=-1)")
		return 0, nil
	}

	// 执行清理
	return s.accountService.CleanupTrash(ctx, days)
}

// cleanupSpamEmails 清理垃圾邮件
func (s *CleanupService) cleanupSpamEmails(ctx context.Context) {
	log.Println("[CleanupService] Starting spam emails cleanup...")

	// 获取垃圾邮件自动清理天数配置
	value, err := s.settingService.Get(ctx, nil, "spam", "auto_cleanup_days", nil)
	if err != nil {
		log.Printf("[CleanupService] Failed to get spam cleanup setting: %v", err)
		// 使用默认值 30 天
		value = "30"
	}

	// 如果配置为空，使用默认值 30
	if value == "" {
		value = "30"
	}

	// 解析天数
	days, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("[CleanupService] Invalid spam cleanup days value: %s, error: %v", value, err)
		return
	}

	// 如果设置为 -1，表示不自动清理
	if days < 0 {
		log.Println("[CleanupService] Spam auto cleanup is disabled (days=-1)")
		return
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 构建过滤条件：查询超过指定天数的垃圾邮件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	// 查询所有垃圾邮件
	emails, _, err := s.emailRepo.List(ctx, filter, 0, 10000)
	if err != nil {
		log.Printf("[CleanupService] Failed to list spam emails: %v", err)
		return
	}

	// 过滤出超过指定天数的邮件并删除
	cleanedCount := 0
	for _, email := range emails {
		// 检查邮件是否超过指定天数
		var checkTime time.Time
		if email.SpamDetectedAt != nil {
			checkTime = *email.SpamDetectedAt
		} else if email.UserMarkedAt != nil {
			checkTime = *email.UserMarkedAt
		} else {
			checkTime = email.SentAt
		}

		if checkTime.Before(cutoffTime) {
			// 软删除邮件
			deleted := true
			if err := s.emailRepo.UpdateLocalStatus(ctx, email.ID, nil, nil, nil, &deleted); err != nil {
				log.Printf("[CleanupService] Failed to delete spam email %d: %v", email.ID, err)
				continue
			}
			cleanedCount++
		}
	}

	log.Printf("[CleanupService] Spam emails cleanup completed: cleaned=%d emails, retention_days=%d", cleanedCount, days)
}

// ManualSpamCleanup 手动触发垃圾邮件清理（用于测试或管理接口）
func (s *CleanupService) ManualSpamCleanup(ctx context.Context) (int, error) {
	log.Println("[CleanupService] Manual spam cleanup triggered")

	// 获取配置
	value, err := s.settingService.Get(ctx, nil, "spam", "auto_cleanup_days", nil)
	if err != nil {
		return 0, err
	}

	// 如果配置为空，使用默认值 30
	if value == "" {
		value = "30"
	}

	// 解析天数
	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	// 如果设置为 -1，返回 0
	if days < 0 {
		log.Println("[CleanupService] Spam auto cleanup is disabled (days=-1)")
		return 0, nil
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 构建过滤条件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	// 查询所有垃圾邮件
	emails, _, err := s.emailRepo.List(ctx, filter, 0, 10000)
	if err != nil {
		return 0, err
	}

	// 过滤出超过指定天数的邮件并删除
	cleanedCount := 0
	for _, email := range emails {
		var checkTime time.Time
		if email.SpamDetectedAt != nil {
			checkTime = *email.SpamDetectedAt
		} else if email.UserMarkedAt != nil {
			checkTime = *email.UserMarkedAt
		} else {
			checkTime = email.SentAt
		}

		if checkTime.Before(cutoffTime) {
			deleted := true
			if err := s.emailRepo.UpdateLocalStatus(ctx, email.ID, nil, nil, nil, &deleted); err != nil {
				log.Printf("[CleanupService] Failed to delete spam email %d: %v", email.ID, err)
				continue
			}
			cleanedCount++
		}
	}

	return cleanedCount, nil
}
