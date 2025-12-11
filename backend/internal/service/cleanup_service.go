package service

import (
	"context"
	"strconv"
	"time"

	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"

	"github.com/robfig/cron/v3"
)

// 模块日志记录器
var cleanupLog = logger.NewWithModule("Cleanup")

// CleanupService 清理服务
type CleanupService struct {
	accountService       AccountService
	settingService       *SettingService
	emailRepo            repository.EmailRepository
	syncLogRepo          repository.SyncLogRepository
	webhookLogRepo       repository.WebhookLogRepository
	spamDetectionLogRepo repository.SpamDetectionLogRepository
	cron                 *cron.Cron
	isRunning            bool
}

// NewCleanupService 创建清理服务实例
func NewCleanupService(
	accountService AccountService,
	settingService *SettingService,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	webhookLogRepo repository.WebhookLogRepository,
	spamDetectionLogRepo repository.SpamDetectionLogRepository,
) *CleanupService {
	return &CleanupService{
		accountService:       accountService,
		settingService:       settingService,
		emailRepo:            emailRepo,
		syncLogRepo:          syncLogRepo,
		webhookLogRepo:       webhookLogRepo,
		spamDetectionLogRepo: spamDetectionLogRepo,
		cron:                 cron.New(),
		isRunning:            false,
	}
}

// Start 启动清理服务
func (s *CleanupService) Start(ctx context.Context) error {
	if s.isRunning {
		cleanupLog.Debug("清理服务已在运行")
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

	// 添加定时任务：每天凌晨 4 点执行日志清理
	_, err = s.cron.AddFunc("0 4 * * *", func() {
		s.cleanupLogs(ctx)
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	s.isRunning = true
	cleanupLog.Info("清理服务已启动，定时任务: 02:00 回收站清理, 03:00 垃圾邮件清理, 04:00 日志清理")

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
	cleanupLog.Info("清理服务已停止")
}

// cleanupTrash 清理回收站
func (s *CleanupService) cleanupTrash(ctx context.Context) {
	cleanupLog.Debug("开始回收站清理...")

	// 获取配置
	value, err := s.settingService.Get(ctx, nil, "system", "trash_auto_cleanup_days", nil)
	if err != nil {
		cleanupLog.Warn("获取清理配置失败: %v", err)
		return
	}

	// 如果配置为空，使用默认值 7
	if value == "" {
		value = "7"
	}

	// 解析天数
	days, err := strconv.Atoi(value)
	if err != nil {
		cleanupLog.Warn("清理天数配置无效: %s, %v", value, err)
		return
	}

	// 如果设置为 -1，表示不自动清理
	if days < 0 {
		cleanupLog.Debug("自动清理已禁用 (days=-1)")
		return
	}

	// 执行清理
	cleanedCount, err := s.accountService.CleanupTrash(ctx, days)
	if err != nil {
		cleanupLog.Error("回收站清理失败: %v", err)
		return
	}

	if cleanedCount > 0 {
		cleanupLog.Info("回收站清理完成: 清理=%d, 保留天数=%d", cleanedCount, days)
	}
}

// ManualCleanup 手动触发清理（用于测试或管理接口）
func (s *CleanupService) ManualCleanup(ctx context.Context) (int, error) {
	cleanupLog.Info("手动触发回收站清理")

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
		cleanupLog.Debug("自动清理已禁用 (days=-1)")
		return 0, nil
	}

	// 执行清理
	return s.accountService.CleanupTrash(ctx, days)
}

// cleanupSpamEmails 清理垃圾邮件
func (s *CleanupService) cleanupSpamEmails(ctx context.Context) {
	cleanupLog.Debug("开始垃圾邮件清理...")

	// 获取垃圾邮件自动清理天数配置
	value, err := s.settingService.Get(ctx, nil, "spam", "auto_cleanup_days", nil)
	if err != nil {
		cleanupLog.Warn("获取垃圾邮件清理配置失败: %v", err)
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
		cleanupLog.Warn("垃圾邮件清理天数配置无效: %s, %v", value, err)
		return
	}

	// 如果设置为 -1，表示不自动清理
	if days < 0 {
		cleanupLog.Debug("垃圾邮件自动清理已禁用 (days=-1)")
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
		cleanupLog.Error("查询垃圾邮件失败: %v", err)
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
				cleanupLog.Warn("删除垃圾邮件失败: id=%d, %v", email.ID, err)
				continue
			}
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		cleanupLog.Info("垃圾邮件清理完成: 清理=%d, 保留天数=%d", cleanedCount, days)
	}
}

// ManualSpamCleanup 手动触发垃圾邮件清理（用于测试或管理接口）
func (s *CleanupService) ManualSpamCleanup(ctx context.Context) (int, error) {
	cleanupLog.Info("手动触发垃圾邮件清理")

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
		cleanupLog.Debug("垃圾邮件自动清理已禁用 (days=-1)")
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
				cleanupLog.Warn("删除垃圾邮件失败: id=%d, %v", email.ID, err)
				continue
			}
			cleanedCount++
		}
	}

	return cleanedCount, nil
}

// cleanupLogs 清理各类日志
func (s *CleanupService) cleanupLogs(ctx context.Context) {
	cleanupLog.Debug("开始日志清理...")

	// 清理同步日志
	s.cleanupSyncLogs(ctx)

	// 清理 Webhook 日志
	s.cleanupWebhookLogs(ctx)

	// 清理垃圾邮件检测日志
	s.cleanupSpamDetectionLogs(ctx)
}

// cleanupSyncLogs 清理同步日志
func (s *CleanupService) cleanupSyncLogs(ctx context.Context) {
	value, err := s.settingService.Get(ctx, nil, "system", "sync_logs_retention_days", nil)
	if err != nil {
		cleanupLog.Warn("获取同步日志清理配置失败: %v", err)
		return
	}

	if value == "" {
		value = "7"
	}

	days, err := strconv.Atoi(value)
	if err != nil {
		cleanupLog.Warn("同步日志清理天数配置无效: %s, %v", value, err)
		return
	}

	if days < 0 {
		cleanupLog.Debug("同步日志自动清理已禁用 (days=-1)")
		return
	}

	if err := s.syncLogRepo.DeleteOldLogs(ctx, days); err != nil {
		cleanupLog.Error("同步日志清理失败: %v", err)
		return
	}

	cleanupLog.Info("同步日志清理完成, 保留天数=%d", days)
}

// cleanupWebhookLogs 清理 Webhook 日志
func (s *CleanupService) cleanupWebhookLogs(ctx context.Context) {
	value, err := s.settingService.Get(ctx, nil, "system", "webhook_logs_retention_days", nil)
	if err != nil {
		cleanupLog.Warn("获取 Webhook 日志清理配置失败: %v", err)
		return
	}

	if value == "" {
		value = "14"
	}

	days, err := strconv.Atoi(value)
	if err != nil {
		cleanupLog.Warn("Webhook 日志清理天数配置无效: %s, %v", value, err)
		return
	}

	if days < 0 {
		cleanupLog.Debug("Webhook 日志自动清理已禁用 (days=-1)")
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)
	if err := s.webhookLogRepo.DeleteOldLogs(ctx, cutoffTime); err != nil {
		cleanupLog.Error("Webhook 日志清理失败: %v", err)
		return
	}

	cleanupLog.Info("Webhook 日志清理完成, 保留天数=%d", days)
}

// cleanupSpamDetectionLogs 清理垃圾邮件检测日志
func (s *CleanupService) cleanupSpamDetectionLogs(ctx context.Context) {
	value, err := s.settingService.Get(ctx, nil, "system", "spam_detection_logs_retention_days", nil)
	if err != nil {
		cleanupLog.Warn("获取垃圾邮件检测日志清理配置失败: %v", err)
		return
	}

	if value == "" {
		value = "7"
	}

	days, err := strconv.Atoi(value)
	if err != nil {
		cleanupLog.Warn("垃圾邮件检测日志清理天数配置无效: %s, %v", value, err)
		return
	}

	if days < 0 {
		cleanupLog.Debug("垃圾邮件检测日志自动清理已禁用 (days=-1)")
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)
	if err := s.spamDetectionLogRepo.DeleteOldLogs(ctx, cutoffTime); err != nil {
		cleanupLog.Error("垃圾邮件检测日志清理失败: %v", err)
		return
	}

	cleanupLog.Info("垃圾邮件检测日志清理完成, 保留天数=%d", days)
}

// ManualLogCleanup 手动触发日志清理
func (s *CleanupService) ManualLogCleanup(ctx context.Context) error {
	cleanupLog.Info("手动触发日志清理")
	s.cleanupLogs(ctx)
	return nil
}
