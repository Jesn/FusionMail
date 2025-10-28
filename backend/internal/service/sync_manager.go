package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"fusionmail/internal/adapter"
	"fusionmail/internal/repository"
	"fusionmail/pkg/database"
)

// SyncManager 同步管理器
type SyncManager struct {
	syncService SyncService
	running     bool
	mu          sync.RWMutex
	cancel      context.CancelFunc
}

// NewSyncManager 创建同步管理器实例
func NewSyncManager() *SyncManager {
	// 创建 Repository 实例
	db := database.GetDB()
	accountRepo := repository.NewAccountRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)

	// 创建适配器工厂
	adapterFactory := adapter.NewFactory()

	// 创建同步服务
	syncService := NewSyncService(accountRepo, emailRepo, syncLogRepo, adapterFactory)

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

	log.Println("Sync manager started")
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
		log.Printf("Failed to stop scheduler: %v", err)
	}

	// 取消上下文
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	m.running = false
	log.Println("Sync manager stopped")
	return nil
}

// IsRunning 检查是否正在运行
func (m *SyncManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
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
	default:
		return fmt.Errorf("unsupported provider: %s", account.Provider)
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
