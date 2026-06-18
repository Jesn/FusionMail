package service

import (
	"context"
	"fmt"
	"sync"

	"fusionmail/internal/adapter"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// 模块日志记录器
var syncManagerLog = logger.NewWithModule("SyncManager")

// SyncManager 同步管理器
type SyncManager struct {
	syncService        SyncService
	accountRepo        repository.AccountRepository
	adapterFactory     *adapter.Factory
	credentialResolver *CredentialResolver
	running            bool
	mu                 sync.RWMutex
	cancel             context.CancelFunc
}

func NewSyncManagerWithDeps(
	syncService SyncService,
	accountRepo repository.AccountRepository,
	adapterFactory *adapter.Factory,
	credentialResolver *CredentialResolver,
) *SyncManager {
	return &SyncManager{
		syncService:        syncService,
		accountRepo:        accountRepo,
		adapterFactory:     adapterFactory,
		credentialResolver: credentialResolver,
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
	if m.accountRepo == nil || m.adapterFactory == nil || m.credentialResolver == nil {
		return fmt.Errorf("sync manager dependencies are not configured")
	}

	// 获取账户信息（预加载 Provider 和 Adapter 关联）
	account, err := m.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountUID)
	}

	credentials, err := m.credentialResolver.Resolve(account)
	if err != nil {
		return fmt.Errorf("failed to resolve credentials: %w", err)
	}

	provider, err := m.adapterFactory.CreateProviderFromAccount(
		account.GetProviderName(),
		account.GetProtocol(),
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// 测试连接
	return provider.TestConnection(ctx)
}
