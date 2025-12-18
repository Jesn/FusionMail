package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SystemService 系统管理服务
type SystemService struct {
	db           *gorm.DB
	redis        *redis.Client
	accountRepo  repository.AccountRepository
	emailRepo    repository.EmailRepository
	ruleRepo     repository.RuleRepository
	webhookRepo  repository.WebhookRepository
	syncLogRepo  repository.SyncLogRepository
	providerRepo repository.ProviderRepository
	logger       *logger.Logger
	startTime    time.Time
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
	providerRepo repository.ProviderRepository,
	logger *logger.Logger,
) *SystemService {
	return &SystemService{
		db:           db,
		redis:        redis,
		accountRepo:  accountRepo,
		emailRepo:    emailRepo,
		ruleRepo:     ruleRepo,
		webhookRepo:  webhookRepo,
		syncLogRepo:  syncLogRepo,
		providerRepo: providerRepo,
		logger:       logger,
		startTime:    time.Now(),
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
		if account, err := s.accountRepo.FindByUID(ctx, log.AccountUID); err == nil && account != nil {
			item.AccountName = account.Email // 使用邮箱地址作为账户名称
			item.Provider = account.Provider
		} else {
			// 账户已删除，使用占位符
			item.AccountName = "已删除的账户"
			item.Provider = "unknown"
		}

		logItems = append(logItems, item)
	}

	return logItems, total, nil
}

// GetSupportedProviders 获取支持的邮箱提供商列表
// 优先从缓存读取，缓存未命中时从数据库读取，数据库查询失败时使用硬编码配置作为降级
func (s *SystemService) GetSupportedProviders(ctx context.Context) ([]ProviderInfo, error) {
	// 1. 尝试从 Redis 缓存读取
	cacheKey := "providers:cache"
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var providers []ProviderInfo
			if err := s.redis.Get(ctx, cacheKey).Scan(&providers); err == nil {
				s.logger.Debug("从缓存获取提供商配置", "count", len(providers))
				return providers, nil
			}
		}
	}

	s.logger.Info("缓存未命中，从数据库获取提供商配置")

	// 2. 从数据库读取启用状态的提供商
	dbProviders, err := s.providerRepo.FindEnabled(ctx)
	if err != nil {
		s.logger.Error("从数据库获取提供商失败", "error", err)
		s.logger.Warn("使用硬编码的降级配置")
		// 降级：使用硬编码的默认配置
		return s.getFallbackProviders(), nil
	}

	// 3. 转换为 DTO
	var providers []ProviderInfo
	for _, p := range dbProviders {
		// 解析 supported_protocols JSON
		protocols, err := p.GetSupportedProtocols()
		if err != nil {
			s.logger.Warn("解析协议配置失败", "provider", p.Name, "error", err)
			continue
		}

		providerInfo := ProviderInfo{
			ID:                  p.ID,
			Name:                p.Name,
			DisplayName:         p.DisplayName,
			ProviderType:        providerNameToType(p.Name), // 从 name 推断类型（向后兼容）
			SupportedProtocols:  protocols,
			RecommendedProtocol: p.RecommendedProtocol,
			RequiresOAuth:       p.RequiresOAuth,
			Enabled:             p.Enabled,
			IMAPHost:            p.IMAPHost,
			IMAPPort:            p.IMAPPort,
			POP3Host:            p.POP3Host,
			POP3Port:            p.POP3Port,
			SMTPHost:            p.SMTPHost,
			SMTPPort:            p.SMTPPort,
			IMAPEncryption:      p.IMAPEncryption,
			POP3Encryption:      p.POP3Encryption,
			SMTPEncryption:      p.SMTPEncryption,
			EmailDomains:        p.EmailDomains,
			Description:         p.Description,
		}
		providers = append(providers, providerInfo)
	}

	s.logger.Info("从数据库获取提供商配置成功", "count", len(providers))

	// 4. 写入缓存（1小时 TTL）
	if s.redis != nil && len(providers) > 0 {
		err := s.redis.Set(ctx, cacheKey, providers, time.Hour).Err()
		if err != nil {
			s.logger.Warn("写入提供商配置缓存失败", "error", err)
		} else {
			s.logger.Debug("写入提供商配置到缓存", "cache_key", cacheKey, "ttl", "1h")
		}
	}

	return providers, nil
}

// getFallbackProviders 获取降级配置（保留原来的硬编码配置作为fallback）
func (s *SystemService) getFallbackProviders() []ProviderInfo {
	s.logger.Info("使用硬编码的降级配置")

	factory := adapter.NewFactory()
	// 获取所有支持的提供商
	providerNames := factory.GetSupportedProviders()

	var providers []ProviderInfo
	for _, name := range providerNames {
		info := factory.GetProviderInfo(name)
		if info != nil {
			providerInfo := ProviderInfo{
				ID:          0, // Fallback providers don't have real IDs
				Name:        name,
				DisplayName: info.DisplayName,
				ProviderType: func(name string) int {
					switch name {
					case "gmail":
						return 1 // ProviderTypeGmail
					case "outlook":
						return 2 // ProviderTypeOutlook
					case "icloud":
						return 3 // ProviderTypeIcloud
					case "qq":
						return 4 // ProviderTypeQQ
					case "163":
						return 5 // ProviderType163
					case "generic":
						return 6 // ProviderTypeGeneric
					default:
						return 6 // 默认使用通用类型
					}
				}(name), // 使用映射函数
				SupportedProtocols:  info.SupportedProtocols,
				RecommendedProtocol: info.RecommendedProtocol,
				RequiresOAuth:       info.RequiresOAuth,
				Enabled:             true, // Fallback providers are always enabled
				IMAPHost:            info.IMAPHost,
				IMAPPort:            info.IMAPPort,
				POP3Host:            info.POP3Host,
				POP3Port:            info.POP3Port,
			}
			providers = append(providers, providerInfo)
		}
	}

	s.logger.Warn("降级配置返回", "count", len(providers), "source", "hardcoded")
	return providers
}

// InvalidateProviderCache 失效提供商配置缓存
// 当提供商配置发生变更时调用此方法清除缓存
func (s *SystemService) InvalidateProviderCache(ctx context.Context) {
	cacheKey := "providers:cache"
	if s.redis != nil {
		err := s.redis.Del(ctx, cacheKey).Err()
		if err != nil {
			s.logger.Error("清除提供商配置缓存失败", "cache_key", cacheKey, "error", err)
		} else {
			s.logger.Info("清除提供商配置缓存成功", "cache_key", cacheKey)
		}
	}
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
	EmailsAdded  int64      `json:"emails_added"`  // 新增邮件数
	EmailsTotal  int64      `json:"emails_total"`  // 总邮件数
	ErrorMessage string     `json:"error_message"` // 错误信息
}

// ProviderInfo 邮箱提供商信息
type ProviderInfo struct {
	ID                  int64    `json:"id"`                        // 提供商ID
	Name                string   `json:"name"`                      // 提供商标识
	DisplayName         string   `json:"display_name"`              // 显示名称
	ProviderType        int      `json:"provider_type"`             // 提供商类型（枚举值）
	SupportedProtocols  []string `json:"supported_protocols"`       // 支持的协议
	RecommendedProtocol string   `json:"recommended_protocol"`      // 推荐协议
	RequiresOAuth       bool     `json:"requires_oauth"`            // 是否需要OAuth
	Enabled             bool     `json:"enabled"`                   // 是否启用
	IMAPHost            string   `json:"imap_host,omitempty"`       // IMAP服务器地址
	IMAPPort            int      `json:"imap_port,omitempty"`       // IMAP端口
	POP3Host            string   `json:"pop3_host,omitempty"`       // POP3服务器地址
	POP3Port            int      `json:"pop3_port,omitempty"`       // POP3端口
	SMTPHost            string   `json:"smtp_host,omitempty"`       // SMTP服务器地址
	SMTPPort            int      `json:"smtp_port,omitempty"`       // SMTP端口
	IMAPEncryption      string   `json:"imap_encryption,omitempty"` // IMAP加密方式
	POP3Encryption      string   `json:"pop3_encryption,omitempty"` // POP3加密方式
	SMTPEncryption      string   `json:"smtp_encryption,omitempty"` // SMTP加密方式
	EmailDomains        []string `json:"email_domains,omitempty"`   // 支持的邮箱域名列表
	Description         string   `json:"description,omitempty"`     // 描述信息
}

// providerNameToType 将提供商名称转换为类型枚举值（用于向后兼容）
func providerNameToType(name string) int {
	switch name {
	case "gmail":
		return 1 // ProviderTypeGmail
	case "outlook":
		return 2 // ProviderTypeOutlook
	case "icloud":
		return 3 // ProviderTypeIcloud
	case "qq":
		return 4 // ProviderTypeQQ
	case "163":
		return 5 // ProviderType163
	default:
		return 6 // ProviderTypeGeneric
	}
}
