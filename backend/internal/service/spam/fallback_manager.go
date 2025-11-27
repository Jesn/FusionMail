package spam

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// FallbackManager 降级策略管理器
// 管理外部服务的健康状态和降级策略
type FallbackManager struct {
	services    map[string]*ServiceHealth
	config      *FallbackConfig
	mu          sync.RWMutex
	loadMonitor *LoadMonitor
}

// FallbackConfig 降级配置
type FallbackConfig struct {
	// RBL 服务超时时间（默认 5 秒）
	RBLTimeout time.Duration
	// SURBL 服务超时时间（默认 3 秒）
	SURBLTimeout time.Duration
	// 服务熔断阈值（连续失败次数）
	CircuitBreakerThreshold int
	// 熔断恢复时间（默认 30 秒）
	CircuitBreakerRecoveryTime time.Duration
	// 高负载阈值（检测队列长度）
	HighLoadThreshold int
	// 高负载时跳过贝叶斯分类
	SkipBayesianOnHighLoad bool
}

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	Name              string    `json:"name"`
	IsHealthy         bool      `json:"is_healthy"`
	ConsecutiveErrors int32     `json:"consecutive_errors"`
	LastError         error     `json:"-"`
	LastErrorTime     time.Time `json:"last_error_time"`
	LastSuccessTime   time.Time `json:"last_success_time"`
	TotalRequests     int64     `json:"total_requests"`
	TotalErrors       int64     `json:"total_errors"`
	CircuitOpen       bool      `json:"circuit_open"`
	CircuitOpenTime   time.Time `json:"circuit_open_time"`
	mu                sync.RWMutex
}

// LoadMonitor 系统负载监控
type LoadMonitor struct {
	currentLoad   int64
	highLoadCount int64
	lastCheckTime time.Time
	mu            sync.RWMutex
}

// DefaultFallbackConfig 返回默认降级配置
func DefaultFallbackConfig() *FallbackConfig {
	return &FallbackConfig{
		RBLTimeout:                 5 * time.Second,
		SURBLTimeout:               3 * time.Second,
		CircuitBreakerThreshold:    5,
		CircuitBreakerRecoveryTime: 30 * time.Second,
		HighLoadThreshold:          100,
		SkipBayesianOnHighLoad:     true,
	}
}

// NewFallbackManager 创建降级策略管理器
func NewFallbackManager(config *FallbackConfig) *FallbackManager {
	if config == nil {
		config = DefaultFallbackConfig()
	}

	fm := &FallbackManager{
		services: make(map[string]*ServiceHealth),
		config:   config,
		loadMonitor: &LoadMonitor{
			lastCheckTime: time.Now(),
		},
	}

	// 初始化服务健康状态
	fm.services["rbl"] = &ServiceHealth{Name: "RBL", IsHealthy: true}
	fm.services["surbl"] = &ServiceHealth{Name: "SURBL", IsHealthy: true}
	fm.services["bayesian"] = &ServiceHealth{Name: "Bayesian", IsHealthy: true}

	// 启动健康检查协程
	go fm.healthCheckLoop()

	return fm
}

// ==================== 服务健康管理 ====================

// RecordSuccess 记录服务调用成功
func (fm *FallbackManager) RecordSuccess(serviceName string) {
	fm.mu.RLock()
	service, ok := fm.services[serviceName]
	fm.mu.RUnlock()

	if !ok {
		return
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	atomic.StoreInt32(&service.ConsecutiveErrors, 0)
	atomic.AddInt64(&service.TotalRequests, 1)
	service.LastSuccessTime = time.Now()
	service.IsHealthy = true

	// 如果熔断器打开，检查是否可以关闭
	if service.CircuitOpen {
		if time.Since(service.CircuitOpenTime) > fm.config.CircuitBreakerRecoveryTime {
			service.CircuitOpen = false
		}
	}
}

// RecordError 记录服务调用失败
func (fm *FallbackManager) RecordError(serviceName string, err error) {
	fm.mu.RLock()
	service, ok := fm.services[serviceName]
	fm.mu.RUnlock()

	if !ok {
		return
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	atomic.AddInt32(&service.ConsecutiveErrors, 1)
	atomic.AddInt64(&service.TotalRequests, 1)
	atomic.AddInt64(&service.TotalErrors, 1)
	service.LastError = err
	service.LastErrorTime = time.Now()

	// 检查是否需要打开熔断器
	if int(atomic.LoadInt32(&service.ConsecutiveErrors)) >= fm.config.CircuitBreakerThreshold {
		service.CircuitOpen = true
		service.CircuitOpenTime = time.Now()
		service.IsHealthy = false
	}
}

// IsServiceAvailable 检查服务是否可用
func (fm *FallbackManager) IsServiceAvailable(serviceName string) bool {
	fm.mu.RLock()
	service, ok := fm.services[serviceName]
	fm.mu.RUnlock()

	if !ok {
		return true // 未知服务默认可用
	}

	service.mu.RLock()
	defer service.mu.RUnlock()

	// 如果熔断器打开，检查是否可以尝试恢复
	if service.CircuitOpen {
		if time.Since(service.CircuitOpenTime) > fm.config.CircuitBreakerRecoveryTime {
			// 允许半开状态尝试
			return true
		}
		return false
	}

	return service.IsHealthy
}

// GetServiceHealth 获取服务健康状态
func (fm *FallbackManager) GetServiceHealth(serviceName string) *ServiceHealth {
	fm.mu.RLock()
	service, ok := fm.services[serviceName]
	fm.mu.RUnlock()

	if !ok {
		return nil
	}

	service.mu.RLock()
	defer service.mu.RUnlock()

	// 返回副本
	return &ServiceHealth{
		Name:              service.Name,
		IsHealthy:         service.IsHealthy,
		ConsecutiveErrors: atomic.LoadInt32(&service.ConsecutiveErrors),
		LastErrorTime:     service.LastErrorTime,
		LastSuccessTime:   service.LastSuccessTime,
		TotalRequests:     atomic.LoadInt64(&service.TotalRequests),
		TotalErrors:       atomic.LoadInt64(&service.TotalErrors),
		CircuitOpen:       service.CircuitOpen,
		CircuitOpenTime:   service.CircuitOpenTime,
	}
}

// GetAllServicesHealth 获取所有服务健康状态
func (fm *FallbackManager) GetAllServicesHealth() map[string]*ServiceHealth {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make(map[string]*ServiceHealth)
	for name := range fm.services {
		result[name] = fm.GetServiceHealth(name)
	}
	return result
}

// ==================== 负载监控 ====================

// RecordLoad 记录当前负载
func (fm *FallbackManager) RecordLoad(queueLength int) {
	fm.loadMonitor.mu.Lock()
	defer fm.loadMonitor.mu.Unlock()

	atomic.StoreInt64(&fm.loadMonitor.currentLoad, int64(queueLength))
	fm.loadMonitor.lastCheckTime = time.Now()

	if queueLength > fm.config.HighLoadThreshold {
		atomic.AddInt64(&fm.loadMonitor.highLoadCount, 1)
	}
}

// IsHighLoad 检查是否处于高负载状态
func (fm *FallbackManager) IsHighLoad() bool {
	return atomic.LoadInt64(&fm.loadMonitor.currentLoad) > int64(fm.config.HighLoadThreshold)
}

// GetCurrentLoad 获取当前负载
func (fm *FallbackManager) GetCurrentLoad() int64 {
	return atomic.LoadInt64(&fm.loadMonitor.currentLoad)
}

// ShouldSkipBayesian 是否应该跳过贝叶斯分类（高负载时）
func (fm *FallbackManager) ShouldSkipBayesian() bool {
	if !fm.config.SkipBayesianOnHighLoad {
		return false
	}
	return fm.IsHighLoad()
}

// ==================== 降级策略执行 ====================

// ExecuteWithFallback 带降级策略执行操作
func (fm *FallbackManager) ExecuteWithFallback(
	ctx context.Context,
	serviceName string,
	operation func(ctx context.Context) (interface{}, error),
	fallback func(ctx context.Context) (interface{}, error),
) (interface{}, error) {
	// 检查服务是否可用
	if !fm.IsServiceAvailable(serviceName) {
		// 服务不可用，执行降级逻辑
		if fallback != nil {
			return fallback(ctx)
		}
		return nil, fmt.Errorf("service %s is unavailable and no fallback provided", serviceName)
	}

	// 获取超时配置
	timeout := fm.getServiceTimeout(serviceName)

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 执行操作
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := operation(timeoutCtx)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// 等待结果或超时
	select {
	case result := <-resultChan:
		fm.RecordSuccess(serviceName)
		return result, nil
	case err := <-errChan:
		fm.RecordError(serviceName, err)
		// 执行降级逻辑
		if fallback != nil {
			return fallback(ctx)
		}
		return nil, err
	case <-timeoutCtx.Done():
		fm.RecordError(serviceName, timeoutCtx.Err())
		// 超时，执行降级逻辑
		if fallback != nil {
			return fallback(ctx)
		}
		return nil, fmt.Errorf("service %s timeout", serviceName)
	}
}

// getServiceTimeout 获取服务超时配置
func (fm *FallbackManager) getServiceTimeout(serviceName string) time.Duration {
	switch serviceName {
	case "rbl":
		return fm.config.RBLTimeout
	case "surbl":
		return fm.config.SURBLTimeout
	default:
		return 5 * time.Second
	}
}

// ==================== 健康检查 ====================

// healthCheckLoop 定期健康检查
func (fm *FallbackManager) healthCheckLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fm.checkAndRecoverServices()
	}
}

// checkAndRecoverServices 检查并恢复服务
func (fm *FallbackManager) checkAndRecoverServices() {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, service := range fm.services {
		service.mu.Lock()
		// 如果熔断器打开且超过恢复时间，尝试恢复
		if service.CircuitOpen {
			if time.Since(service.CircuitOpenTime) > fm.config.CircuitBreakerRecoveryTime {
				service.CircuitOpen = false
				service.IsHealthy = true
				atomic.StoreInt32(&service.ConsecutiveErrors, 0)
			}
		}
		service.mu.Unlock()
	}
}

// ResetService 重置服务状态
func (fm *FallbackManager) ResetService(serviceName string) {
	fm.mu.RLock()
	service, ok := fm.services[serviceName]
	fm.mu.RUnlock()

	if !ok {
		return
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	service.IsHealthy = true
	service.CircuitOpen = false
	atomic.StoreInt32(&service.ConsecutiveErrors, 0)
}

// ResetAllServices 重置所有服务状态
func (fm *FallbackManager) ResetAllServices() {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, service := range fm.services {
		service.mu.Lock()
		service.IsHealthy = true
		service.CircuitOpen = false
		atomic.StoreInt32(&service.ConsecutiveErrors, 0)
		service.mu.Unlock()
	}
}

// ==================== 统计信息 ====================

// FallbackStats 降级统计信息
type FallbackStats struct {
	Services      map[string]*ServiceHealth `json:"services"`
	CurrentLoad   int64                     `json:"current_load"`
	HighLoadCount int64                     `json:"high_load_count"`
	IsHighLoad    bool                      `json:"is_high_load"`
}

// GetStats 获取降级统计信息
func (fm *FallbackManager) GetStats() *FallbackStats {
	return &FallbackStats{
		Services:      fm.GetAllServicesHealth(),
		CurrentLoad:   fm.GetCurrentLoad(),
		HighLoadCount: atomic.LoadInt64(&fm.loadMonitor.highLoadCount),
		IsHighLoad:    fm.IsHighLoad(),
	}
}
