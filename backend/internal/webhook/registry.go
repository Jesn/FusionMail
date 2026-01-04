package webhook

import (
	"sort"
	"sync"
)

// AdapterRegistry Webhook 适配器注册表
// 管理所有已注册的 Webhook 适配器，支持并发安全访问
type AdapterRegistry struct {
	// adapters 存储所有已注册的适配器
	// key: provider type, value: adapter instance
	adapters map[string]WebhookAdapter

	// mu 读写锁，保证并发安全
	mu sync.RWMutex
}

// NewAdapterRegistry 创建新的适配器注册表
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]WebhookAdapter),
	}
}

// Register 注册适配器
// 如果已存在相同 provider type 的适配器，将被覆盖
func (r *AdapterRegistry) Register(adapter WebhookAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.GetProviderType()] = adapter
}

// Unregister 注销适配器
// 返回是否成功注销（如果不存在则返回 false）
func (r *AdapterRegistry) Unregister(providerType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.adapters[providerType]; ok {
		delete(r.adapters, providerType)
		return true
	}
	return false
}

// Get 获取指定 provider type 的适配器
// 返回适配器实例和是否存在的标志
func (r *AdapterRegistry) Get(providerType string) (WebhookAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[providerType]
	return adapter, ok
}

// MustGet 获取指定 provider type 的适配器
// 如果不存在则返回错误
func (r *AdapterRegistry) MustGet(providerType string) (WebhookAdapter, error) {
	adapter, ok := r.Get(providerType)
	if !ok {
		return nil, NewUnsupportedProviderError(providerType)
	}
	return adapter, nil
}

// Has 检查是否存在指定 provider type 的适配器
func (r *AdapterRegistry) Has(providerType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.adapters[providerType]
	return ok
}

// List 列出所有已注册的 provider type
// 返回按字母顺序排序的列表
func (r *AdapterRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}

	// 按字母顺序排序，保证输出稳定
	sort.Strings(types)
	return types
}

// Count 返回已注册的适配器数量
func (r *AdapterRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.adapters)
}

// GetAll 获取所有已注册的适配器
// 返回 provider type 到适配器的映射（副本）
func (r *AdapterRegistry) GetAll() map[string]WebhookAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make(map[string]WebhookAdapter, len(r.adapters))
	for k, v := range r.adapters {
		result[k] = v
	}
	return result
}

// AdapterInfo 适配器信息（用于 API 响应）
type AdapterInfo struct {
	// ProviderType 服务商类型
	ProviderType string `json:"provider_type"`

	// SignatureHeader 签名 Header 名称
	SignatureHeader string `json:"signature_header"`
}

// ListInfo 列出所有已注册适配器的详细信息
func (r *AdapterRegistry) ListInfo() []AdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]AdapterInfo, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		infos = append(infos, AdapterInfo{
			ProviderType:    adapter.GetProviderType(),
			SignatureHeader: adapter.GetSignatureHeader(),
		})
	}

	// 按 provider type 排序
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ProviderType < infos[j].ProviderType
	})

	return infos
}

// 全局默认注册表（可选使用）
var defaultRegistry = NewAdapterRegistry()

// DefaultRegistry 获取全局默认注册表
func DefaultRegistry() *AdapterRegistry {
	return defaultRegistry
}

// RegisterDefault 向全局默认注册表注册适配器
func RegisterDefault(adapter WebhookAdapter) {
	defaultRegistry.Register(adapter)
}

// GetDefault 从全局默认注册表获取适配器
func GetDefault(providerType string) (WebhookAdapter, bool) {
	return defaultRegistry.Get(providerType)
}
