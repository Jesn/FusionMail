package service

import (
	"context"
	"log"
	"strconv"

	"github.com/robfig/cron/v3"
)

// CleanupService 清理服务
type CleanupService struct {
	accountService AccountService
	settingService *SettingService
	cron           *cron.Cron
	isRunning      bool
}

// NewCleanupService 创建清理服务实例
func NewCleanupService(accountService AccountService, settingService *SettingService) *CleanupService {
	return &CleanupService{
		accountService: accountService,
		settingService: settingService,
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

	// 添加定时任务：每天凌晨 2 点执行清理
	_, err := s.cron.AddFunc("0 2 * * *", func() {
		s.cleanupTrash(ctx)
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	s.isRunning = true
	log.Println("[CleanupService] Started successfully, scheduled to run daily at 2:00 AM")

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
