// Package goroutine 提供 Goroutine 管理和监控功能
// 包括：Goroutine 数量监控、泄露检测、并发控制、pprof 集成
package goroutine

import (
	"context"
	"fmt"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"fusionmail/pkg/logger"
)

// 模块日志记录器
var monitorLog = logger.NewWithModule("GoroutineMonitor")

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 检查间隔（默认 30 秒）
	CheckInterval time.Duration
	// Goroutine 数量告警阈值（默认 1000）
	WarningThreshold int
	// Goroutine 数量严重告警阈值（默认 5000）
	CriticalThreshold int
	// 是否启用泄露检测（默认 true）
	EnableLeakDetection bool
	// 泄露检测窗口大小（用于计算增长率，默认 10）
	LeakDetectionWindowSize int
	// 泄露检测增长率阈值（默认 0.5，即 50% 增长）
	LeakGrowthRateThreshold float64
}

// DefaultMonitorConfig 返回默认配置
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		CheckInterval:           30 * time.Second,
		WarningThreshold:        1000,
		CriticalThreshold:       5000,
		EnableLeakDetection:     true,
		LeakDetectionWindowSize: 10,
		LeakGrowthRateThreshold: 0.5,
	}
}

// MonitorStats 监控统计数据
type MonitorStats struct {
	// 当前 Goroutine 数量
	CurrentCount int
	// 峰值 Goroutine 数量
	PeakCount int
	// 最小 Goroutine 数量
	MinCount int
	// 平均 Goroutine 数量
	AvgCount float64
	// 检查次数
	CheckCount int64
	// 告警次数
	WarningCount int64
	// 严重告警次数
	CriticalCount int64
	// 疑似泄露次数
	LeakSuspectCount int64
	// 最后检查时间
	LastCheckTime time.Time
	// 启动时间
	StartTime time.Time
}

// Monitor Goroutine 监控器
type Monitor struct {
	config *MonitorConfig
	stats  *MonitorStats

	// 历史记录（用于泄露检测）
	history     []int
	historyIdx  int
	historyFull bool

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 回调函数
	onWarning  func(count int)
	onCritical func(count int)
	onLeak     func(count int, growthRate float64)

	mu      sync.RWMutex
	running bool
}

// NewMonitor 创建新的监控器
func NewMonitor(config *MonitorConfig) *Monitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	// 确保窗口大小至少为 1
	windowSize := config.LeakDetectionWindowSize
	if windowSize <= 0 {
		windowSize = 10
	}

	return &Monitor{
		config: config,
		stats: &MonitorStats{
			MinCount:  int(^uint(0) >> 1), // 初始化为最大值
			StartTime: time.Now(),
		},
		history: make([]int, windowSize),
	}
}

// SetWarningCallback 设置告警回调
func (m *Monitor) SetWarningCallback(fn func(count int)) {
	m.onWarning = fn
}

// SetCriticalCallback 设置严重告警回调
func (m *Monitor) SetCriticalCallback(fn func(count int)) {
	m.onCritical = fn
}

// SetLeakCallback 设置泄露检测回调
func (m *Monitor) SetLeakCallback(fn func(count int, growthRate float64)) {
	m.onLeak = fn
}

// Start 启动监控
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	m.running = true
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	m.wg.Add(1)
	go m.monitorLoop()

	monitorLog.Info("Goroutine 监控器已启动 (间隔=%v, 告警阈值=%d, 严重阈值=%d)",
		m.config.CheckInterval, m.config.WarningThreshold, m.config.CriticalThreshold)

	return nil
}

// Stop 停止监控
func (m *Monitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.cancel()
	m.mu.Unlock()

	m.wg.Wait()
	monitorLog.Info("Goroutine 监控器已停止")
}

// GetStats 获取统计数据
func (m *Monitor) GetStats() MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := *m.stats
	stats.CurrentCount = runtime.NumGoroutine()
	return stats
}

// monitorLoop 监控循环
func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.check()
		case <-m.ctx.Done():
			return
		}
	}
}

// check 执行一次检查
func (m *Monitor) check() {
	count := runtime.NumGoroutine()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新统计
	m.stats.CurrentCount = count
	m.stats.CheckCount++
	m.stats.LastCheckTime = time.Now()

	if count > m.stats.PeakCount {
		m.stats.PeakCount = count
	}
	if count < m.stats.MinCount {
		m.stats.MinCount = count
	}

	// 计算平均值（增量平均）
	m.stats.AvgCount = m.stats.AvgCount + (float64(count)-m.stats.AvgCount)/float64(m.stats.CheckCount)

	// 更新历史记录
	m.history[m.historyIdx] = count
	m.historyIdx = (m.historyIdx + 1) % len(m.history)
	if m.historyIdx == 0 {
		m.historyFull = true
	}

	// 检查告警
	if count >= m.config.CriticalThreshold {
		m.stats.CriticalCount++
		monitorLog.Error("Goroutine 数量严重超标: count=%d, threshold=%d", count, m.config.CriticalThreshold)
		if m.onCritical != nil {
			go m.onCritical(count)
		}
	} else if count >= m.config.WarningThreshold {
		m.stats.WarningCount++
		monitorLog.Warn("Goroutine 数量超过告警阈值: count=%d, threshold=%d", count, m.config.WarningThreshold)
		if m.onWarning != nil {
			go m.onWarning(count)
		}
	}

	// 泄露检测
	if m.config.EnableLeakDetection && m.historyFull {
		m.detectLeak(count)
	}
}

// detectLeak 检测 Goroutine 泄露
func (m *Monitor) detectLeak(currentCount int) {
	// 计算历史窗口的平均值
	var sum int
	for _, v := range m.history {
		sum += v
	}
	avgHistory := float64(sum) / float64(len(m.history))

	// 计算增长率
	if avgHistory > 0 {
		growthRate := (float64(currentCount) - avgHistory) / avgHistory
		if growthRate > m.config.LeakGrowthRateThreshold {
			m.stats.LeakSuspectCount++
			monitorLog.Warn("疑似 Goroutine 泄露: current=%d, avgHistory=%.1f, growthRate=%.2f%%",
				currentCount, avgHistory, growthRate*100)
			if m.onLeak != nil {
				go m.onLeak(currentCount, growthRate)
			}
		}
	}
}

// DumpGoroutineStacks 导出所有 Goroutine 的堆栈信息
func DumpGoroutineStacks() string {
	buf := make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	return string(buf[:n])
}

// GetGoroutineProfile 获取 Goroutine profile
func GetGoroutineProfile() *pprof.Profile {
	return pprof.Lookup("goroutine")
}

// GetCurrentCount 获取当前 Goroutine 数量
func GetCurrentCount() int {
	return runtime.NumGoroutine()
}
