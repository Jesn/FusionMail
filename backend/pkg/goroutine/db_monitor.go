package goroutine

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"fusionmail/pkg/logger"
)

// 模块日志记录器
var dbMonitorLog = logger.NewWithModule("DBPoolMonitor")

// DBPoolStats 数据库连接池统计
type DBPoolStats struct {
	// 最大打开连接数
	MaxOpenConnections int
	// 当前打开连接数
	OpenConnections int
	// 使用中的连接数
	InUse int
	// 空闲连接数
	Idle int
	// 等待连接的总次数
	WaitCount int64
	// 等待连接的总时间
	WaitDuration time.Duration
	// 因空闲超时关闭的连接数
	MaxIdleClosed int64
	// 因生命周期超时关闭的连接数
	MaxLifetimeClosed int64
	// 检查时间
	CheckTime time.Time
}

// DBPoolMonitorConfig 数据库连接池监控配置
type DBPoolMonitorConfig struct {
	// 检查间隔
	CheckInterval time.Duration
	// 使用率告警阈值（0-1）
	UsageWarningThreshold float64
	// 使用率严重告警阈值（0-1）
	UsageCriticalThreshold float64
	// 等待时间告警阈值
	WaitDurationWarningThreshold time.Duration
}

// DefaultDBPoolMonitorConfig 返回默认配置
func DefaultDBPoolMonitorConfig() *DBPoolMonitorConfig {
	return &DBPoolMonitorConfig{
		CheckInterval:                30 * time.Second,
		UsageWarningThreshold:        0.7,
		UsageCriticalThreshold:       0.9,
		WaitDurationWarningThreshold: 100 * time.Millisecond,
	}
}

// DBPoolMonitor 数据库连接池监控器
type DBPoolMonitor struct {
	db     *sql.DB
	config *DBPoolMonitorConfig

	// 历史统计
	lastStats     *DBPoolStats
	peakInUse     int
	totalWaitTime time.Duration

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 回调
	onWarning  func(stats DBPoolStats, message string)
	onCritical func(stats DBPoolStats, message string)

	mu      sync.RWMutex
	running bool
}

// NewDBPoolMonitor 创建数据库连接池监控器
func NewDBPoolMonitor(db *sql.DB, config *DBPoolMonitorConfig) *DBPoolMonitor {
	if config == nil {
		config = DefaultDBPoolMonitorConfig()
	}

	return &DBPoolMonitor{
		db:     db,
		config: config,
	}
}

// SetWarningCallback 设置告警回调
func (m *DBPoolMonitor) SetWarningCallback(fn func(stats DBPoolStats, message string)) {
	m.onWarning = fn
}

// SetCriticalCallback 设置严重告警回调
func (m *DBPoolMonitor) SetCriticalCallback(fn func(stats DBPoolStats, message string)) {
	m.onCritical = fn
}

// Start 启动监控
func (m *DBPoolMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	m.wg.Add(1)
	go m.monitorLoop()

	dbMonitorLog.Info("数据库连接池监控器已启动 (间隔=%v)", m.config.CheckInterval)
	return nil
}

// Stop 停止监控
func (m *DBPoolMonitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.cancel()
	m.mu.Unlock()

	m.wg.Wait()
	dbMonitorLog.Info("数据库连接池监控器已停止")
}

// GetStats 获取当前统计
func (m *DBPoolMonitor) GetStats() DBPoolStats {
	stats := m.db.Stats()
	return DBPoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
		CheckTime:          time.Now(),
	}
}

// GetPeakInUse 获取峰值使用数
func (m *DBPoolMonitor) GetPeakInUse() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.peakInUse
}

// monitorLoop 监控循环
func (m *DBPoolMonitor) monitorLoop() {
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
func (m *DBPoolMonitor) check() {
	stats := m.GetStats()

	m.mu.Lock()
	// 更新峰值
	if stats.InUse > m.peakInUse {
		m.peakInUse = stats.InUse
	}

	// 计算增量等待时间
	var waitDurationDelta time.Duration
	if m.lastStats != nil {
		waitDurationDelta = stats.WaitDuration - m.lastStats.WaitDuration
	}
	m.lastStats = &stats
	m.mu.Unlock()

	// 计算使用率
	var usageRate float64
	if stats.MaxOpenConnections > 0 {
		usageRate = float64(stats.InUse) / float64(stats.MaxOpenConnections)
	}

	// 检查告警条件
	if usageRate >= m.config.UsageCriticalThreshold {
		msg := "数据库连接池使用率严重超标"
		dbMonitorLog.Error("%s: inUse=%d, max=%d, rate=%.1f%%",
			msg, stats.InUse, stats.MaxOpenConnections, usageRate*100)
		if m.onCritical != nil {
			go m.onCritical(stats, msg)
		}
	} else if usageRate >= m.config.UsageWarningThreshold {
		msg := "数据库连接池使用率超过告警阈值"
		dbMonitorLog.Warn("%s: inUse=%d, max=%d, rate=%.1f%%",
			msg, stats.InUse, stats.MaxOpenConnections, usageRate*100)
		if m.onWarning != nil {
			go m.onWarning(stats, msg)
		}
	}

	// 检查等待时间
	if waitDurationDelta > m.config.WaitDurationWarningThreshold {
		msg := "数据库连接等待时间过长"
		dbMonitorLog.Warn("%s: waitDuration=%v, waitCount=%d",
			msg, waitDurationDelta, stats.WaitCount)
		if m.onWarning != nil {
			go m.onWarning(stats, msg)
		}
	}

	// 定期输出状态日志
	dbMonitorLog.Debug("DB Pool: open=%d, inUse=%d, idle=%d, waitCount=%d, waitDuration=%v",
		stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount, stats.WaitDuration)
}
