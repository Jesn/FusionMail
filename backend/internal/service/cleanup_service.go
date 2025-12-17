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
	deletedKeyRepo       *repository.DeletedEmailKeyRepository // 已删除邮件去重标识仓库
	dedupeKeyGen         *DedupeKeyGenerator                   // 去重标识生成器
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
	deletedKeyRepo *repository.DeletedEmailKeyRepository,
) *CleanupService {
	return &CleanupService{
		accountService:       accountService,
		settingService:       settingService,
		emailRepo:            emailRepo,
		syncLogRepo:          syncLogRepo,
		webhookLogRepo:       webhookLogRepo,
		spamDetectionLogRepo: spamDetectionLogRepo,
		deletedKeyRepo:       deletedKeyRepo,
		dedupeKeyGen:         NewDedupeKeyGenerator(),
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

// cleanupTrash 清理回收站（账户和邮件）
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

	// 1. 清理账户回收站
	accountCleanedCount, err := s.accountService.CleanupTrash(ctx, days)
	if err != nil {
		cleanupLog.Error("账户回收站清理失败: %v", err)
	} else if accountCleanedCount > 0 {
		cleanupLog.Info("账户回收站清理完成: 清理=%d, 保留天数=%d", accountCleanedCount, days)
	}

	// 2. 清理邮件回收站
	emailCleanedCount, err := s.cleanupDeletedEmails(ctx, days)
	if err != nil {
		cleanupLog.Error("邮件回收站清理失败: %v", err)
	} else if emailCleanedCount > 0 {
		cleanupLog.Info("邮件回收站清理完成: 清理=%d, 保留天数=%d", emailCleanedCount, days)
	}
}

// cleanupDeletedEmails 清理已删除的邮件（物理删除超过指定天数的邮件）
// Requirements: 2.1 - 物理删除前记录 dedupe_key
func (s *CleanupService) cleanupDeletedEmails(ctx context.Context, days int) (int, error) {
	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 构建过滤条件：查询已删除且超过指定天数的邮件
	isDeleted := true
	filter := &repository.EmailFilter{
		IsDeleted: &isDeleted,
	}

	// 查询所有已删除的邮件
	emails, _, err := s.emailRepo.List(ctx, filter, 0, 10000)
	if err != nil {
		return 0, err
	}

	// 过滤出超过指定天数的邮件并物理删除
	cleanedCount := 0
	for _, email := range emails {
		// 检查邮件删除时间是否超过指定天数
		// gorm.DeletedAt 类型需要使用 .Valid 和 .Time 访问
		if email.DeletedAt.Valid && email.DeletedAt.Time.Before(cutoffTime) {
			// 在物理删除前记录 dedupe_key (Requirements: 2.1)
			if s.deletedKeyRepo != nil {
				dedupeKey := email.DedupeKey
				// 如果邮件没有 dedupe_key，生成一个
				if dedupeKey == "" {
					dedupeKey = s.dedupeKeyGen.GenerateFromRaw(
						email.MessageID,
						email.FromAddress,
						email.Subject,
						email.SentAt,
					)
				}
				// 记录到 deleted_email_keys 表
				if err := s.deletedKeyRepo.CreateIfNotExists(ctx, email.AccountUID, dedupeKey); err != nil {
					cleanupLog.Warn("记录已删除邮件标识失败: id=%d, key=%s, %v", email.ID, dedupeKey, err)
					// 继续删除，不阻塞
				}
			}

			// 物理删除邮件
			if err := s.emailRepo.Delete(ctx, email.ID); err != nil {
				cleanupLog.Warn("物理删除邮件失败: id=%d, %v", email.ID, err)
				continue
			}
			cleanedCount++
		}
	}

	return cleanedCount, nil
}

// ManualCleanup 手动触发清理（用于测试或管理接口）
// 返回清理的账户数和邮件数
func (s *CleanupService) ManualCleanup(ctx context.Context) (int, int, error) {
	cleanupLog.Info("手动触发回收站清理")

	// 获取配置
	value, err := s.settingService.Get(ctx, nil, "system", "trash_auto_cleanup_days", nil)
	if err != nil {
		return 0, 0, err
	}

	// 如果配置为空，使用默认值 7
	if value == "" {
		value = "7"
	}

	// 解析天数
	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, 0, err
	}

	// 如果设置为 -1，返回 0
	if days < 0 {
		cleanupLog.Debug("自动清理已禁用 (days=-1)")
		return 0, 0, nil
	}

	// 1. 清理账户回收站
	accountCount, err := s.accountService.CleanupTrash(ctx, days)
	if err != nil {
		return 0, 0, err
	}

	// 2. 清理邮件回收站
	emailCount, err := s.cleanupDeletedEmails(ctx, days)
	if err != nil {
		return accountCount, 0, err
	}

	return accountCount, emailCount, nil
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

	// 清理过期的已删除邮件标识 (Requirements: 2.3)
	s.cleanupDeletedEmailKeys(ctx)
}

// cleanupDeletedEmailKeys 清理过期的已删除邮件标识
// Requirements: 2.3 - 90 天后清理
func (s *CleanupService) cleanupDeletedEmailKeys(ctx context.Context) {
	if s.deletedKeyRepo == nil {
		return
	}

	// 默认保留 90 天
	retentionDays := 90

	cleanedCount, err := s.deletedKeyRepo.CleanupOldKeys(ctx, retentionDays)
	if err != nil {
		cleanupLog.Error("已删除邮件标识清理失败: %v", err)
		return
	}

	if cleanedCount > 0 {
		cleanupLog.Info("已删除邮件标识清理完成: 清理=%d, 保留天数=%d", cleanedCount, retentionDays)
	}
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
