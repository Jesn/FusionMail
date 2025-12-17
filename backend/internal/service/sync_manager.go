package service

import (
	"context"
	"fmt"
	"sync"

	"fusionmail/internal/adapter"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/logger"
	pkgredis "fusionmail/pkg/redis"
)

// 模块日志记录器
var syncManagerLog = logger.NewWithModule("SyncManager")

// SyncManager 同步管理器
type SyncManager struct {
	syncService SyncService
	running     bool
	mu          sync.RWMutex
	cancel      context.CancelFunc
}

// NewSyncManager 创建同步管理器实例
func NewSyncManager(cryptoService *crypto.Service, spamDetector SpamDetectorInterface) *SyncManager {
	// 创建 Repository 实例
	db := database.GetDB()
	accountRepo := repository.NewAccountRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)
	deletedKeyRepo := repository.NewDeletedEmailKeyRepository(db)
	oauth2ClientRepo := repository.NewOAuth2ClientRepository(db)
	providerRepo := repository.NewProviderRepository(db)

	// 创建日志器
	appLogger := logger.New()

	// 创建适配器工厂
	adapterFactory := adapter.NewFactory()

	// 获取 Redis 客户端（用于分布式同步锁）
	redisClient := pkgredis.GetClient()

	// 创建同步服务（传入 Redis 客户端以启用分布式锁）
	syncService := NewSyncService(accountRepo, emailRepo, syncLogRepo, deletedKeyRepo, adapterFactory, oauth2ClientRepo, providerRepo, appLogger, cryptoService, spamDetector, redisClient)

	return &SyncManager{
		syncService: syncService,
	}
}

// Start 启动同步管理器
func (m *SyncManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("sync manager is already running")
	}

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.running = true

	// 启动定时同步调度器
	if err := m.syncService.StartScheduler(ctx); err != nil {
		m.running = false
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	syncManagerLog.Info("同步管理器已启动")
	return nil
}

// Stop 停止同步管理器
func (m *SyncManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	// 停止调度器
	if err := m.syncService.StopScheduler(); err != nil {
		syncManagerLog.Error("停止调度器失败: %v", err)
	}

	// 取消上下文
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	m.running = false
	syncManagerLog.Info("同步管理器已停止")
	return nil
}

// IsRunning 检查是否正在运行
func (m *SyncManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetSyncService 获取同步服务实例
// 用于需要直接访问 SyncService 的场景（如取消同步、获取进度）
func (m *SyncManager) GetSyncService() SyncService {
	return m.syncService
}

// SyncAccount 手动同步指定账户
func (m *SyncManager) SyncAccount(ctx context.Context, accountUID string) error {
	return m.syncService.SyncAccount(ctx, accountUID)
}

// SyncAllAccounts 手动同步所有账户
func (m *SyncManager) SyncAllAccounts(ctx context.Context) error {
	return m.syncService.SyncAllAccounts(ctx)
}

// TestAccountConnection 测试账户连接
func (m *SyncManager) TestAccountConnection(ctx context.Context, accountUID string) error {
	// 获取账户信息
	db := database.GetDB()
	accountRepo := repository.NewAccountRepository(db)

	account, err := accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountUID)
	}

	// 创建适配器
	adapterFactory := adapter.NewFactory()

	// 解析凭证（简化版本）
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: account.AuthType,
	}

	// 设置服务器配置
	// 如果用户手动配置了服务器地址，优先使用用户配置
	if account.IMAPHost != "" && account.IMAPPort != 0 {
		credentials.Host = account.IMAPHost
		credentials.Port = account.IMAPPort
		credentials.TLS = true // 默认开启 TLS
	} else {
		switch account.Provider {
		case "icloud":
			credentials.Host = "imap.mail.me.com"
			credentials.Port = 993
		case "qq":
			credentials.Host = "imap.qq.com"
			credentials.Port = 993
		case "163":
			credentials.Host = "imap.163.com"
			credentials.Port = 993
		case "gmail":
			credentials.Host = "imap.gmail.com"
			credentials.Port = 993
		case "outlook":
			credentials.Host = "outlook.office365.com"
			credentials.Port = 993
		case "generic":
			// generic 必须配置服务器信息
		default:
			return fmt.Errorf("unsupported provider: %s", account.Provider)
		}
	}

	// 对于 generic 或手动配置的情况，进行额外检查和设置
	if account.Provider == "generic" || (account.IMAPHost != "" && account.IMAPPort != 0) {
		if account.Protocol == "imap" {
			if credentials.Host == "" {
				credentials.Host = account.IMAPHost
				credentials.Port = account.IMAPPort
			}
		} else if account.Protocol == "pop3" {
			credentials.Host = account.POP3Host
			credentials.Port = account.POP3Port
		}

		// 智能修复常见的配置错误
		if credentials.Host == "mail.linuxdo.org" {
			syncManagerLog.Debug("自动修复错误主机配置: %s -> mail.linux.do", credentials.Host)
			credentials.Host = "mail.linux.do"
		}

		// 验证必要的配置
		if credentials.Host == "" || credentials.Port == 0 {
			return fmt.Errorf("provider requires host and port configuration")
		}
	}

	provider, err := adapterFactory.CreateProviderFromAccount(
		account.Provider,
		account.Protocol,
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// 测试连接
	return provider.TestConnection(ctx)
}
