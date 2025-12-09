package adapter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AdapterType 适配器类型枚举
type AdapterType string

const (
	AdapterTypeStandard AdapterType = "standard" // 标准 OAuth2 适配器
	AdapterTypeQuick    AdapterType = "quick"    // 短效适配器
	AdapterTypeAuto     AdapterType = "auto"     // 自动选择
)

// AdapterManager 适配器管理器
// 负责管理适配器的创建、切换和生命周期
type AdapterManager struct {
	factory     *Factory
	adapters    map[string]MailProvider // 缓存的适配器实例
	configs     map[string]*Config      // 适配器配置
	mutex       sync.RWMutex            // 读写锁保护并发访问
	logger      Logger                  // 日志记录器
	maxAdapters int                     // 最大适配器数量
}

// AdapterManagerConfig 适配器管理器配置
type AdapterManagerConfig struct {
	MaxAdapters int    // 最大适配器数量（默认 100）
	Logger      Logger // 日志记录器
}

// NewAdapterManager 创建适配器管理器
func NewAdapterManager(config *AdapterManagerConfig) *AdapterManager {
	if config == nil {
		config = &AdapterManagerConfig{}
	}

	if config.MaxAdapters <= 0 {
		config.MaxAdapters = 100
	}

	if config.Logger == nil {
		config.Logger = NewSimpleLogger("AdapterManager")
	}

	return &AdapterManager{
		factory:     NewFactory(),
		adapters:    make(map[string]MailProvider),
		configs:     make(map[string]*Config),
		logger:      config.Logger,
		maxAdapters: config.MaxAdapters,
	}
}

// GetAdapter 获取或创建适配器
// accountID: 账户唯一标识
// config: 适配器配置
// adapterType: 适配器类型（可选，为空时自动选择）
func (m *AdapterManager) GetAdapter(accountID string, config *Config, adapterType AdapterType) (MailProvider, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查是否已存在适配器
	if adapter, exists := m.adapters[accountID]; exists {
		// 检查配置是否发生变化
		if m.isConfigChanged(accountID, config) {
			m.logger.Info("配置发生变化，重新创建适配器", "account_id", accountID)

			// 清理旧适配器
			if err := m.cleanupAdapter(accountID); err != nil {
				m.logger.Warn("清理旧适配器失败", "account_id", accountID, "error", err)
			}
		} else {
			// Debug 级别：频繁调用的操作
			// m.logger.Info("使用缓存的适配器", ...)
			return adapter, nil
		}
	}

	// 创建新适配器
	adapter, err := m.createAdapter(config, adapterType)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter for account %s: %w", accountID, err)
	}

	// 检查适配器数量限制
	if len(m.adapters) >= m.maxAdapters {
		m.logger.Warn("达到最大适配器数量限制，清理最旧的适配器", "max_adapters", m.maxAdapters)
		if err := m.evictOldestAdapter(); err != nil {
			m.logger.Error("清理最旧适配器失败", "error", err)
		}
	}

	// 缓存适配器和配置
	m.adapters[accountID] = adapter
	m.configs[accountID] = m.cloneConfig(config)

	m.logger.Info("成功创建并缓存适配器", "account_id", accountID, "type", fmt.Sprintf("%T", adapter))
	return adapter, nil
}

// SwitchAdapter 切换适配器类型
func (m *AdapterManager) SwitchAdapter(accountID string, config *Config, newType AdapterType) (MailProvider, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Info("开始切换适配器类型", "account_id", accountID, "new_type", newType)

	// 清理现有适配器
	if err := m.cleanupAdapter(accountID); err != nil {
		m.logger.Warn("清理现有适配器失败", "account_id", accountID, "error", err)
	}

	// 创建新适配器
	adapter, err := m.createAdapter(config, newType)
	if err != nil {
		return nil, fmt.Errorf("failed to switch adapter for account %s: %w", accountID, err)
	}

	// 缓存新适配器
	m.adapters[accountID] = adapter
	m.configs[accountID] = m.cloneConfig(config)

	m.logger.Info("成功切换适配器类型", "account_id", accountID, "new_type", fmt.Sprintf("%T", adapter))
	return adapter, nil
}

// RemoveAdapter 移除适配器
func (m *AdapterManager) RemoveAdapter(accountID string) error {
	if accountID == "" {
		return fmt.Errorf("account ID is required")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.cleanupAdapter(accountID)
}

// GetAdapterInfo 获取适配器信息
func (m *AdapterManager) GetAdapterInfo(accountID string) (*AdapterInfo, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	adapter, exists := m.adapters[accountID]
	if !exists {
		return nil, fmt.Errorf("adapter not found for account %s", accountID)
	}

	config := m.configs[accountID]

	info := &AdapterInfo{
		AccountID:    accountID,
		AdapterType:  fmt.Sprintf("%T", adapter),
		ProviderType: adapter.GetProviderType(),
		Protocol:     adapter.GetProtocol(),
		Email:        config.Email,
		Provider:     config.Provider,
		AuthType:     config.AuthType,
		IsConnected:  false,      // 需要实际测试连接
		CreatedAt:    time.Now(), // 简化实现，实际应该记录创建时间
	}

	return info, nil
}

// ListAdapters 列出所有适配器
func (m *AdapterManager) ListAdapters() ([]*AdapterInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	infos := make([]*AdapterInfo, 0, len(m.adapters))

	for accountID := range m.adapters {
		info, err := m.getAdapterInfoUnsafe(accountID)
		if err != nil {
			m.logger.Warn("获取适配器信息失败", "account_id", accountID, "error", err)
			continue
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// TestAllConnections 测试所有适配器连接
func (m *AdapterManager) TestAllConnections(ctx context.Context) (map[string]error, error) {
	m.mutex.RLock()
	adapters := make(map[string]MailProvider)
	for accountID, adapter := range m.adapters {
		adapters[accountID] = adapter
	}
	m.mutex.RUnlock()

	results := make(map[string]error)

	for accountID, adapter := range adapters {
		err := adapter.TestConnection(ctx)
		results[accountID] = err

		if err != nil {
			m.logger.Warn("适配器连接测试失败", "account_id", accountID, "error", err)
		} else {
			m.logger.Info("适配器连接测试成功", "account_id", accountID)
		}
	}

	return results, nil
}

// Cleanup 清理所有适配器
func (m *AdapterManager) Cleanup() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var errors []error

	for accountID := range m.adapters {
		if err := m.cleanupAdapter(accountID); err != nil {
			errors = append(errors, fmt.Errorf("failed to cleanup adapter %s: %w", accountID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	m.logger.Info("所有适配器清理完成")
	return nil
}

// GetStats 获取管理器统计信息
func (m *AdapterManager) GetStats() *ManagerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := &ManagerStats{
		TotalAdapters: len(m.adapters),
		MaxAdapters:   m.maxAdapters,
		AdapterTypes:  make(map[string]int),
	}

	for _, adapter := range m.adapters {
		adapterType := fmt.Sprintf("%T", adapter)
		stats.AdapterTypes[adapterType]++
	}

	return stats
}

// 私有方法

// createAdapter 创建适配器
func (m *AdapterManager) createAdapter(config *Config, adapterType AdapterType) (MailProvider, error) {
	switch adapterType {
	case AdapterTypeStandard:
		// 强制使用标准适配器
		if config.Provider == "outlook" {
			config.Protocol = "graph"
		} else if config.Provider == "gmail" {
			config.Protocol = "gmail_api"
		}
		return m.factory.CreateProvider(config)

	case AdapterTypeQuick:
		// 强制使用短效适配器
		if config.Provider == "outlook" {
			config.Protocol = "graph_quick"
		} else {
			return nil, fmt.Errorf("quick adapter not supported for provider: %s", config.Provider)
		}
		return m.factory.CreateProvider(config)

	case AdapterTypeAuto:
		fallthrough
	default:
		// 自动选择适配器
		return m.factory.CreateProviderAuto(config)
	}
}

// isConfigChanged 检查配置是否发生变化
func (m *AdapterManager) isConfigChanged(accountID string, newConfig *Config) bool {
	oldConfig, exists := m.configs[accountID]
	if !exists {
		return true
	}

	// 简化的配置比较，实际应该比较所有重要字段
	return oldConfig.Email != newConfig.Email ||
		oldConfig.Provider != newConfig.Provider ||
		oldConfig.Protocol != newConfig.Protocol ||
		oldConfig.AuthType != newConfig.AuthType ||
		!m.isCredentialsEqual(oldConfig.Credentials, newConfig.Credentials)
}

// isCredentialsEqual 比较凭据是否相等
func (m *AdapterManager) isCredentialsEqual(old, new *Credentials) bool {
	if old == nil && new == nil {
		return true
	}
	if old == nil || new == nil {
		return false
	}

	return old.Email == new.Email &&
		old.Password == new.Password &&
		old.AccessToken == new.AccessToken &&
		old.RefreshToken == new.RefreshToken &&
		old.ClientID == new.ClientID &&
		old.ClientSecret == new.ClientSecret
}

// cleanupAdapter 清理适配器
func (m *AdapterManager) cleanupAdapter(accountID string) error {
	adapter, exists := m.adapters[accountID]
	if !exists {
		return nil
	}

	// 断开连接
	if err := adapter.Disconnect(); err != nil {
		m.logger.Warn("断开适配器连接失败", "account_id", accountID, "error", err)
	}

	// 从缓存中移除
	delete(m.adapters, accountID)
	delete(m.configs, accountID)

	m.logger.Info("适配器清理完成", "account_id", accountID)
	return nil
}

// evictOldestAdapter 清理最旧的适配器（简化实现）
func (m *AdapterManager) evictOldestAdapter() error {
	if len(m.adapters) == 0 {
		return nil
	}

	// 简化实现：清理第一个适配器
	// 实际应该基于 LRU 或创建时间
	for accountID := range m.adapters {
		return m.cleanupAdapter(accountID)
	}

	return nil
}

// cloneConfig 克隆配置
func (m *AdapterManager) cloneConfig(config *Config) *Config {
	if config == nil {
		return nil
	}

	cloned := *config

	if config.Credentials != nil {
		credentials := *config.Credentials
		cloned.Credentials = &credentials
	}

	if config.Proxy != nil {
		proxy := *config.Proxy
		cloned.Proxy = &proxy
	}

	return &cloned
}

// getAdapterInfoUnsafe 获取适配器信息（不加锁）
func (m *AdapterManager) getAdapterInfoUnsafe(accountID string) (*AdapterInfo, error) {
	adapter, exists := m.adapters[accountID]
	if !exists {
		return nil, fmt.Errorf("adapter not found for account %s", accountID)
	}

	config := m.configs[accountID]

	info := &AdapterInfo{
		AccountID:    accountID,
		AdapterType:  fmt.Sprintf("%T", adapter),
		ProviderType: adapter.GetProviderType(),
		Protocol:     adapter.GetProtocol(),
		Email:        config.Email,
		Provider:     config.Provider,
		AuthType:     config.AuthType,
		IsConnected:  false,
		CreatedAt:    time.Now(),
	}

	return info, nil
}

// AdapterInfo 适配器信息
type AdapterInfo struct {
	AccountID    string    `json:"account_id"`
	AdapterType  string    `json:"adapter_type"`
	ProviderType string    `json:"provider_type"`
	Protocol     string    `json:"protocol"`
	Email        string    `json:"email"`
	Provider     string    `json:"provider"`
	AuthType     string    `json:"auth_type"`
	IsConnected  bool      `json:"is_connected"`
	CreatedAt    time.Time `json:"created_at"`
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	TotalAdapters int            `json:"total_adapters"`
	MaxAdapters   int            `json:"max_adapters"`
	AdapterTypes  map[string]int `json:"adapter_types"`
}
