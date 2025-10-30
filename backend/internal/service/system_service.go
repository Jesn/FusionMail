package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SystemService 系统管理服务
type SystemService struct {
	db          *gorm.DB
	redis       *redis.Client
	accountRepo repository.AccountRepository
	emailRepo   repository.EmailRepository
	ruleRepo    repository.RuleRepository
	webhookRepo repository.WebhookRepository
	syncLogRepo repository.SyncLogRepository
	logger      *logger.Logger
	startTime   time.Time
}

// NewSystemService 创建系统管理服务
func NewSystemService(
	db *gorm.DB,
	redis *redis.Client,
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	ruleRepo repository.RuleRepository,
	webhookRepo repository.WebhookRepository,
	syncLogRepo repository.SyncLogRepository,
	logger *logger.Logger,
) *SystemService {
	return &SystemService{
		db:          db,
		redis:       redis,
		accountRepo: accountRepo,
		emailRepo:   emailRepo,
		ruleRepo:    ruleRepo,
		webhookRepo: webhookRepo,
		syncLogRepo: syncLogRepo,
		logger:      logger,
		startTime:   time.Now(),
	}
}

// GetSystemHealth 获取系统健康状态
func (s *SystemService) GetSystemHealth(ctx context.Context) (*SystemHealthResponse, error) {
	components := make(map[string]HealthCheck)
	overallStatus := "healthy"

	// 检查数据库连接
	dbCheck := s.checkDatabase(ctx)
	components["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// 检查 Redis 连接
	redisCheck := s.checkRedis(ctx)
	components["redis"] = redisCheck
	if redisCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// 检查存储
	storageCheck := s.checkStorage(ctx)
	components["storage"] = storageCheck
	if storageCheck.Status != "healthy" && overallStatus == "healthy" {
		overallStatus = "degraded"
	}

	return &SystemHealthResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Version:    "1.0.0", // TODO: 从配置或构建信息获取
		Uptime:     int64(time.Since(s.startTime).Seconds()),
		Components: components,
	}, nil
}

// GetSystemStats 获取系统统计信息
func (s *SystemService) GetSystemStats(ctx context.Context) (*SystemStatsResponse, error) {
	stats := &SystemStatsResponse{}

	// 邮件统计
	var err error
	stats.TotalEmails, err = s.emailRepo.Count(ctx, nil)
	if err != nil {
		s.logger.Error("获取邮件总数失败", "error", err)
		return nil, fmt.Errorf("获取邮件统计失败: %w", err)
	}

	stats.UnreadEmails, err = s.emailRepo.CountUnread(ctx, "")
	if err != nil {
		s.logger.Error("获取未读邮件数失败", "error", err)
		return nil, fmt.Errorf("获取未读邮件统计失败: %w", err)
	}

	// 今日邮件数
	today := time.Now().Truncate(24 * time.Hour)
	stats.TodayEmails, err = s.emailRepo.CountByDateRange(ctx, today, time.Now())
	if err != nil {
		s.logger.Error("获取今日邮件数失败", "error", err)
		return nil, fmt.Errorf("获取今日邮件统计失败: %w", err)
	}

	// 账户统计
	stats.TotalAccounts, err = s.accountRepo.Count(ctx)
	if err != nil {
		s.logger.Error("获取账户总数失败", "error", err)
		return nil, fmt.Errorf("获取账户统计失败: %w", err)
	}

	stats.ActiveAccounts, err = s.accountRepo.CountActive(ctx)
	if err != nil {
		s.logger.Error("获取活跃账户数失败", "error", err)
		return nil, fmt.Errorf("获取活跃账户统计失败: %w", err)
	}

	// 同步统计
	stats.TotalSyncs, err = s.syncLogRepo.Count(ctx, "")
	if err != nil {
		s.logger.Error("获取同步总数失败", "error", err)
		return nil, fmt.Errorf("获取同步统计失败: %w", err)
	}

	stats.SuccessSyncs, err = s.syncLogRepo.Count(ctx, "success")
	if err != nil {
		s.logger.Error("获取成功同步数失败", "error", err)
		return nil, fmt.Errorf("获取成功同步统计失败: %w", err)
	}

	stats.FailedSyncs, err = s.syncLogRepo.Count(ctx, "failed")
	if err != nil {
		s.logger.Error("获取失败同步数失败", "error", err)
		return nil, fmt.Errorf("获取失败同步统计失败: %w", err)
	}

	// 最后同步时间
	lastSync, err := s.syncLogRepo.GetLatest(ctx)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("获取最后同步时间失败", "error", err)
		return nil, fmt.Errorf("获取最后同步时间失败: %w", err)
	}
	if lastSync != nil {
		stats.LastSyncTime = &lastSync.StartedAt
	}

	// 规则统计
	stats.TotalRules, err = s.ruleRepo.Count(ctx)
	if err != nil {
		s.logger.Error("获取规则总数失败", "error", err)
		return nil, fmt.Errorf("获取规则统计失败: %w", err)
	}

	stats.ActiveRules, err = s.ruleRepo.CountActive(ctx)
	if err != nil {
		s.logger.Error("获取活跃规则数失败", "error", err)
		return nil, fmt.Errorf("获取活跃规则统计失败: %w", err)
	}

	// Webhook 统计
	stats.TotalWebhooks, err = s.webhookRepo.Count(ctx)
	if err != nil {
		s.logger.Error("获取Webhook总数失败", "error", err)
		return nil, fmt.Errorf("获取Webhook统计失败: %w", err)
	}

	stats.ActiveWebhooks, err = s.webhookRepo.CountActive(ctx)
	if err != nil {
		s.logger.Error("获取活跃Webhook数失败", "error", err)
		return nil, fmt.Errorf("获取活跃Webhook统计失败: %w", err)
	}

	// 系统资源统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats.MemoryUsage = int64(m.Alloc)
	stats.GoroutineCount = runtime.NumGoroutine()
	// CPU 使用率需要更复杂的计算，暂时设为0
	stats.CPUUsage = 0.0

	// 存储统计
	stats.AttachmentCount, stats.AttachmentSize, err = s.getStorageStats(ctx)
	if err != nil {
		s.logger.Error("获取存储统计失败", "error", err)
		// 存储统计失败不影响整体响应
		stats.AttachmentCount = 0
		stats.AttachmentSize = 0
	}
	stats.StorageUsed = stats.AttachmentSize

	return stats, nil
}

// GetSyncStatus 获取同步状态
func (s *SystemService) GetSyncStatus(ctx context.Context) ([]SyncStatusResponse, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取账户列表失败: %w", err)
	}

	var statusList []SyncStatusResponse
	for _, account := range accounts {
		status := SyncStatusResponse{
			AccountUID:   account.UID,
			AccountName:  account.Email, // 使用邮箱地址作为账户名称
			Provider:     account.Provider,
			SyncInterval: account.SyncInterval,
		}

		// 获取最后同步记录
		lastSync, err := s.syncLogRepo.GetLatestByAccount(ctx, account.UID)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Error("获取账户同步记录失败", "account_uid", account.UID, "error", err)
			status.Status = "unknown"
			status.ErrorMessage = "获取同步状态失败"
		} else if lastSync != nil {
			status.LastSyncTime = &lastSync.StartedAt
			if lastSync.Status == "success" {
				status.Status = "idle"
			} else {
				status.Status = "failed"
				status.ErrorMessage = lastSync.ErrorMessage
			}
		} else {
			status.Status = "idle"
		}

		// 计算下次同步时间
		if status.LastSyncTime != nil && account.SyncEnabled {
			nextSync := status.LastSyncTime.Add(time.Duration(account.SyncInterval) * time.Minute)
			status.NextSyncTime = &nextSync
		}

		// 获取邮件统计
		status.EmailCount, _ = s.emailRepo.CountByAccount(ctx, account.UID)
		status.UnreadCount, _ = s.emailRepo.CountUnread(ctx, account.UID)

		statusList = append(statusList, status)
	}

	return statusList, nil
}

// GetSyncLogs 获取同步日志
func (s *SystemService) GetSyncLogs(ctx context.Context, page, pageSize int, accountUID, status string) ([]SyncLogItem, int64, error) {
	offset := (page - 1) * pageSize

	logs, total, err := s.syncLogRepo.FindWithPagination(ctx, offset, pageSize, accountUID, status)
	if err != nil {
		return nil, 0, fmt.Errorf("获取同步日志失败: %w", err)
	}

	var logItems []SyncLogItem
	for _, log := range logs {
		item := SyncLogItem{
			ID:           log.ID,
			AccountUID:   log.AccountUID,
			Status:       log.Status,
			StartTime:    log.StartedAt,
			EndTime:      log.CompletedAt,
			EmailsAdded:  log.EmailsNew,
			EmailsTotal:  log.EmailsFetched,
			ErrorMessage: log.ErrorMessage,
		}

		// 计算持续时间
		if log.CompletedAt != nil {
			item.Duration = log.CompletedAt.Sub(log.StartedAt).Milliseconds()
		} else {
			item.Duration = int64(log.DurationMs)
		}

		// 获取账户信息
		if account, err := s.accountRepo.FindByUID(ctx, log.AccountUID); err == nil {
			item.AccountName = account.Email // 使用邮箱地址作为账户名称
			item.Provider = account.Provider
		}

		logItems = append(logItems, item)
	}

	return logItems, total, nil
}

// checkDatabase 检查数据库连接
func (s *SystemService) checkDatabase(ctx context.Context) HealthCheck {
	start := time.Now()

	sqlDB, err := s.db.DB()
	if err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("获取数据库连接失败: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("数据库连接失败: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "数据库连接正常",
		Latency: time.Since(start).Milliseconds(),
	}
}

// checkRedis 检查 Redis 连接
func (s *SystemService) checkRedis(ctx context.Context) HealthCheck {
	start := time.Now()

	if err := s.redis.Ping(ctx).Err(); err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Redis连接失败: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Redis连接正常",
		Latency: time.Since(start).Milliseconds(),
	}
}

// checkStorage 检查存储
func (s *SystemService) checkStorage(ctx context.Context) HealthCheck {
	start := time.Now()

	// 这里可以添加存储健康检查逻辑
	// 例如检查存储目录是否可写，S3连接是否正常等

	return HealthCheck{
		Status:  "healthy",
		Message: "存储正常",
		Latency: time.Since(start).Milliseconds(),
	}
}

// getStorageStats 获取存储统计
func (s *SystemService) getStorageStats(ctx context.Context) (int64, int64, error) {
	var count int64
	var totalSize int64

	err := s.db.WithContext(ctx).Model(&model.EmailAttachment{}).Count(&count).Error
	if err != nil {
		return 0, 0, err
	}

	err = s.db.WithContext(ctx).Model(&model.EmailAttachment{}).
		Select("COALESCE(SUM(size), 0)").Scan(&totalSize).Error
	if err != nil {
		return count, 0, err
	}

	return count, totalSize, nil
}

// SystemHealthResponse 系统健康状态响应
type SystemHealthResponse struct {
	Status     string                 `json:"status"`     // overall, database, redis, storage
	Timestamp  time.Time              `json:"timestamp"`  // 检查时间
	Version    string                 `json:"version"`    // 系统版本
	Uptime     int64                  `json:"uptime"`     // 运行时间（秒）
	Components map[string]HealthCheck `json:"components"` // 各组件健康状态
}

// HealthCheck 健康检查结果
type HealthCheck struct {
	Status  string `json:"status"`  // healthy, unhealthy, unknown
	Message string `json:"message"` // 状态描述
	Latency int64  `json:"latency"` // 响应延迟（毫秒）
}

// SystemStatsResponse 系统统计信息响应
type SystemStatsResponse struct {
	// 邮件统计
	TotalEmails  int64 `json:"total_emails"`  // 总邮件数
	UnreadEmails int64 `json:"unread_emails"` // 未读邮件数
	TodayEmails  int64 `json:"today_emails"`  // 今日邮件数

	// 账户统计
	TotalAccounts  int64 `json:"total_accounts"`  // 总账户数
	ActiveAccounts int64 `json:"active_accounts"` // 活跃账户数

	// 同步统计
	TotalSyncs   int64      `json:"total_syncs"`    // 总同步次数
	SuccessSyncs int64      `json:"success_syncs"`  // 成功同步次数
	FailedSyncs  int64      `json:"failed_syncs"`   // 失败同步次数
	LastSyncTime *time.Time `json:"last_sync_time"` // 最后同步时间

	// 规则统计
	TotalRules  int64 `json:"total_rules"`  // 总规则数
	ActiveRules int64 `json:"active_rules"` // 活跃规则数

	// Webhook 统计
	TotalWebhooks  int64 `json:"total_webhooks"`  // 总 Webhook 数
	ActiveWebhooks int64 `json:"active_webhooks"` // 活跃 Webhook 数

	// 系统资源
	MemoryUsage    int64   `json:"memory_usage"`    // 内存使用量（字节）
	GoroutineCount int     `json:"goroutine_count"` // Goroutine 数量
	CPUUsage       float64 `json:"cpu_usage"`       // CPU 使用率（百分比）

	// 存储统计
	StorageUsed     int64 `json:"storage_used"`     // 已使用存储（字节）
	AttachmentCount int64 `json:"attachment_count"` // 附件数量
	AttachmentSize  int64 `json:"attachment_size"`  // 附件总大小（字节）
}

// SyncStatusResponse 同步状态响应
type SyncStatusResponse struct {
	AccountUID   string     `json:"account_uid"`    // 账户UID
	AccountName  string     `json:"account_name"`   // 账户名称
	Provider     string     `json:"provider"`       // 邮箱服务商
	Status       string     `json:"status"`         // 同步状态：idle, syncing, failed
	LastSyncTime *time.Time `json:"last_sync_time"` // 最后同步时间
	NextSyncTime *time.Time `json:"next_sync_time"` // 下次同步时间
	SyncInterval int        `json:"sync_interval"`  // 同步间隔（分钟）
	ErrorMessage string     `json:"error_message"`  // 错误信息
	EmailCount   int64      `json:"email_count"`    // 邮件总数
	UnreadCount  int64      `json:"unread_count"`   // 未读数
}

// SyncLogItem 同步日志项
type SyncLogItem struct {
	ID           int64      `json:"id"`
	AccountUID   string     `json:"account_uid"`
	AccountName  string     `json:"account_name"`
	Provider     string     `json:"provider"`
	Status       string     `json:"status"` // success, failed
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	Duration     int64      `json:"duration"`      // 持续时间（毫秒）
	EmailsAdded  int        `json:"emails_added"`  // 新增邮件数
	EmailsTotal  int        `json:"emails_total"`  // 总邮件数
	ErrorMessage string     `json:"error_message"` // 错误信息
}
