package service

import (
	"context"
	"fmt"
	"sync"

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

// syncService 邮件同步服务实现
type syncService struct {
	accountRepo          repository.AccountRepository
	emailRepo            repository.EmailRepository
	syncLogRepo          repository.SyncLogRepository
	deletedKeyRepo       *repository.DeletedEmailKeyRepository // 已删除邮件去重标识仓库
	adapterFactory       *adapter.Factory
	webAPIAdapterFactory *webapi.WebAPIAdapterFactory // WebAPI 适配器工厂
	cryptoService        *crypto.Service
	credentialResolver   *CredentialResolver
	ruleService          RuleService
	notifier             SyncNotifier
	schedulerStop        chan struct{}
	spamDetector         SpamDetectorInterface // 垃圾邮件检测器
	dedupeKeyGen         *DedupeKeyGenerator   // 去重标识生成器
	logger               *logger.Logger        // 日志记录器

	// 分布式同步锁（基于 Redis，支持自动过期和续期）
	syncLock *synclock.SyncLock

	// 同步取消支持 (Requirements: 5.1, 5.2)
	// 注意：activeSyncs 现在仅用于本地进度追踪，锁由 Redis 管理
	activeSyncs    map[string]*synclock.LockInfo // 活跃同步的锁信息
	activeTrackers map[string]ProgressTracker    // 活跃同步的进度追踪器
	syncMu         sync.RWMutex                  // 保护 activeSyncs 和 activeTrackers
}

// SpamDetectorInterface 垃圾邮件检测器接口
type SpamDetectorInterface interface {
	DetectSpamSimple(ctx context.Context, email *model.Email) (*spam.SpamSimpleResult, error)
}

// NewSyncService 创建邮件同步服务实例
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

	return &syncService{
		accountRepo:          accountRepo,
		emailRepo:            emailRepo,
		syncLogRepo:          syncLogRepo,
		deletedKeyRepo:       deletedKeyRepo,
		adapterFactory:       adapterFactory,
		webAPIAdapterFactory: webapi.NewWebAPIAdapterFactory(), // 初始化 WebAPI 适配器工厂
		cryptoService:        cryptoService,
		credentialResolver:   credentialResolver,
		ruleService:          ruleService,
		notifier:             resolveSyncNotifier(notifier),
		spamDetector:         spamDetector,
		dedupeKeyGen:         NewDedupeKeyGenerator(),
		logger:               syncLogger,
		syncLock:             sl,
		activeSyncs:          make(map[string]*synclock.LockInfo),
		activeTrackers:       make(map[string]ProgressTracker),
	}
}

// CancelSync 取消指定账户的同步
// Requirements: 5.1 - 支持同步取消
func (s *syncService) CancelSync(accountUID string) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	lockInfo, exists := s.activeSyncs[accountUID]
	if !exists {
		return fmt.Errorf("no active sync for account: %s", accountUID)
	}

	// 调用取消函数
	if lockInfo.CancelFunc != nil {
		lockInfo.CancelFunc()
	}
	s.logger.Info("同步已取消: account=%s", accountUID)

	return nil
}

// GetSyncProgress 获取指定账户的同步进度
func (s *syncService) GetSyncProgress(accountUID string) *model.SyncProgress {
	s.syncMu.RLock()
	defer s.syncMu.RUnlock()

	tracker, exists := s.activeTrackers[accountUID]
	if !exists {
		return nil
	}

	return tracker.GetProgress()
}
