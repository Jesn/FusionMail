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
	// 获取账户信息（预加载 Provider 和 Adapter 关联）
	db := database.GetDB()
	accountRepo := repository.NewAccountRepository(db)

	account, err := accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountUID)
	}

	// 创建适配器
	adapterFactory := adapter.NewFactory()

	// 解析凭证（简化版本）
	authType := account.GetAuthType()
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: authType,
	}

	// 设置服务器配置（从 Provider 获取）
	protocol := account.GetProtocol()
	if protocol == "imap" {
		host, port, encryption := account.GetIMAPConfig()
		credentials.Host = host
		credentials.Port = port

		// 设置加密方式
		switch encryption {
		case "ssl", "":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true // 默认使用 SSL
		}
	} else if protocol == "pop3" {
		host, port, encryption := account.GetPOP3Config()
		credentials.Host = host
		credentials.Port = port

		// 设置加密方式
		switch encryption {
		case "ssl", "":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true // 默认使用 SSL
		}
	}

	// 智能修复常见的配置错误
	if credentials.Host == "mail.linuxdo.org" {
		syncManagerLog.Debug("自动修复错误主机配置: %s -> mail.linux.do", credentials.Host)
		credentials.Host = "mail.linux.do"
	}

	// 验证必要的配置（仅对 IMAP/POP3 协议需要 Host/Port）
	// OAuth2 协议使用 API 访问，不需要 Host/Port
	providerName := account.GetProviderName()
	if protocol == "imap" || protocol == "pop3" {
		if credentials.Host == "" || credentials.Port == 0 {
			return fmt.Errorf("server configuration missing: host=%s, port=%d (provider=%s, protocol=%s)",
				credentials.Host, credentials.Port, providerName, protocol)
		}
	}

	provider, err := adapterFactory.CreateProviderFromAccount(
		providerName,
		protocol,
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// 测试连接
	return provider.TestConnection(ctx)
}
