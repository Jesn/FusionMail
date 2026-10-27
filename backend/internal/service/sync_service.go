package service

import (
	"context"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/synclock"

	"github.com/redis/go-redis/v9"
)

// 模块日志记录器
var syncLog = logger.NewWithModule("Sync")

// SyncService 邮件同步服务接口
type SyncService interface {
	// SyncAccount 同步指定账户的邮件
	SyncAccount(ctx context.Context, accountUID string) error

	// SyncAllAccounts 同步所有启用的账户
	SyncAllAccounts(ctx context.Context) error

	// StartScheduler 启动定时同步调度器
	StartScheduler(ctx context.Context) error

	// StopScheduler 停止定时同步调度器
	StopScheduler() error

	// CancelSync 取消指定账户的同步
	// Requirements: 5.1 - 支持同步取消
	CancelSync(accountUID string) error

	// GetSyncProgress 获取指定账户的同步进度
	GetSyncProgress(accountUID string) *model.SyncProgress
}

// SpamDetectorInterface 垃圾邮件检测器接口
type SpamDetectorInterface interface {
	DetectSpamSimple(ctx context.Context, email *model.Email) (*spam.SpamSimpleResult, error)
}

// NewSyncService 创建邮件同步服务实例（组合根入口）。
//
// 内部由三个聚焦的类型组合而成，而不是一个 19 字段 / 24 方法的大类型：
//   - SyncExecutor：执行单账户同步（含 WebAPI 路径）
//   - SyncScheduler：定时调度与全量同步、卡住任务清理
//   - SyncProgressRegistry：活跃同步的共享状态（锁信息 + 进度追踪器）
func NewSyncService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	deletedKeyRepo *repository.DeletedEmailKeyRepository, // 已删除邮件去重标识仓库
	adapterFactory *adapter.Factory,
	appLogger *logger.Logger,
	cryptoService *crypto.Service, // 添加加密服务参数（指针类型）
	credentialResolver *CredentialResolver,
	ruleService RuleService,
	spamDetector SpamDetectorInterface, // 垃圾邮件检测器（可选）
	redisClient *redis.Client, // Redis 客户端，用于分布式锁
	notifier SyncNotifier,
) SyncService {

	// 创建模块日志记录器
	syncLogger := logger.NewWithModule("Sync")

	if appLogger == nil {
		appLogger = syncLogger
	}
	if credentialResolver == nil {
		credentialResolver = NewCredentialResolver(cryptoService, nil)
	}

	// 创建分布式同步锁（如果 Redis 可用）
	var sl *synclock.SyncLock
	if redisClient != nil {
		sl = synclock.NewSyncLock(redisClient)
		syncLogger.Info("分布式同步锁已启用 (Redis)")
	} else {
		syncLogger.Warn("Redis 不可用，使用内存锁（不推荐用于生产环境）")
	}

	// 共享可变状态：executor 写（注册/注销），控制面读（取消/查进度）
	registry := NewSyncProgressRegistry()

	executor := NewSyncExecutor(
		accountRepo,
		emailRepo,
		syncLogRepo,
		adapterFactory,
		webapi.NewWebAPIAdapterFactory(), // 初始化 WebAPI 适配器工厂
		cryptoService,
		credentialResolver,
		resolveSyncNotifier(notifier),
		NewDedupeKeyGenerator(),
		NewSyncProcessor(
			emailRepo,
			deletedKeyRepo,
			NewDedupeKeyGenerator(),
			ruleService,
			spamDetector,
		),
		NewSyncErrorPolicy(accountRepo, syncLogRepo),
		registry,
		sl,
	)

	scheduler := NewSyncScheduler(accountRepo, syncLogRepo, sl, registry, executor)

	return &compositeSyncService{
		scheduler: scheduler,
		executor:  executor,
		registry:  registry,
	}
}

// compositeSyncService 把三个聚焦服务组合成 SyncService。
//
// 它只做转发，不含任何业务逻辑——这样"上帝接口"不会重新长出上帝实现。
type compositeSyncService struct {
	scheduler *SyncScheduler
	executor  *SyncExecutor
	registry  *SyncProgressRegistry
}

func (c *compositeSyncService) SyncAccount(ctx context.Context, accountUID string) error {
	return c.executor.SyncAccount(ctx, accountUID)
}
func (c *compositeSyncService) SyncAllAccounts(ctx context.Context) error {
	return c.scheduler.SyncAllAccounts(ctx)
}
func (c *compositeSyncService) StartScheduler(ctx context.Context) error {
	return c.scheduler.StartScheduler(ctx)
}
func (c *compositeSyncService) StopScheduler() error {
	return c.scheduler.StopScheduler()
}
func (c *compositeSyncService) CancelSync(accountUID string) error {
	return c.registry.Cancel(accountUID)
}
func (c *compositeSyncService) GetSyncProgress(accountUID string) *model.SyncProgress {
	return c.registry.GetProgress(accountUID)
}
