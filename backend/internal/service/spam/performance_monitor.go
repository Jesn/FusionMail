package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceMonitor 性能监控器
// 负责监控垃圾邮件检测的性能指标，确保检测延迟在 200ms 内
type PerformanceMonitor struct {
	logRepo repository.SpamDetectionLogRepository

	// 性能指标
	totalDetections  int64         // 总检测次数
	slowDetections   int64         // 慢检测次数（>200ms）
	totalLatency     int64         // 总延迟（纳秒）
	maxLatency       int64         // 最大延迟（纳秒）
	minLatency       int64         // 最小延迟（纳秒）
	latencyThreshold time.Duration // 延迟阈值（默认 200ms）

	// 各层性能统计
	layerStats map[string]*LayerPerformanceStats
	layerMu    sync.RWMutex

	// 错误统计
	errorCount   int64
	errorLogs    []*PerformanceErrorLog
	errorLogMu   sync.RWMutex
	maxErrorLogs int

	// 检测结果统计
	spamCount int64
	hamCount  int64

	// 时间窗口统计（最近 1 小时）
	windowStats    []*WindowStats
	windowMu       sync.RWMutex
	windowSize     time.Duration
	maxWindowSlots int

	// 告警配置
	alertThreshold float64 // 慢检测比例告警阈值（默认 5%）
	alertCallback  func(alert *PerformanceAlert)
	lastAlertTime  time.Time
	alertCooldown  time.Duration
}

// LayerPerformanceStats 各检测层性能统计
type LayerPerformanceStats struct {
	Name         string `json:"name"`
	TotalCalls   int64  `json:"total_calls"`
	TotalLatency int64  `json:"total_latency_ns"`
	MaxLatency   int64  `json:"max_latency_ns"`
	MinLatency   int64  `json:"min_latency_ns"`
	ErrorCount   int64  `json:"error_count"`
	SkippedCount int64  `json:"skipped_count"` // 被跳过的次数（降级）
}

// WindowStats 时间窗口统计
type WindowStats struct {
	Timestamp      time.Time `json:"timestamp"`
	DetectionCount int64     `json:"detection_count"`
	SlowCount      int64     `json:"slow_count"`
	ErrorCount     int64     `json:"error_count"`
	AvgLatencyMs   float64   `json:"avg_latency_ms"`
	SpamCount      int64     `json:"spam_count"`
	HamCount       int64     `json:"ham_count"`
}

// PerformanceErrorLog 性能错误日志
type PerformanceErrorLog struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"`
	Error     string    `json:"error"`
	LatencyMs float64   `json:"latency_ms"`
	EmailID   string    `json:"email_id"`
	Layer     string    `json:"layer"`
}

// PerformanceAlert 性能告警
type PerformanceAlert struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  string                 `json:"severity"` // warning, critical
	Metrics   map[string]interface{} `json:"metrics"`
}

// PerformanceMetrics 性能指标汇总
type PerformanceMetrics struct {
	TotalDetections int64                             `json:"total_detections"`
	SlowDetections  int64                             `json:"slow_detections"`
	SlowRate        float64                           `json:"slow_rate_percent"`
	AvgLatencyMs    float64                           `json:"avg_latency_ms"`
	MaxLatencyMs    float64                           `json:"max_latency_ms"`
	MinLatencyMs    float64                           `json:"min_latency_ms"`
	P95LatencyMs    float64                           `json:"p95_latency_ms"`
	P99LatencyMs    float64                           `json:"p99_latency_ms"`
	SpamCount       int64                             `json:"spam_count"`
	HamCount        int64                             `json:"ham_count"`
	SpamRate        float64                           `json:"spam_rate_percent"`
	ErrorCount      int64                             `json:"error_count"`
	ErrorRate       float64                           `json:"error_rate_percent"`
	LayerStats      map[string]*LayerPerformanceStats `json:"layer_stats"`
	Timestamp       time.Time                         `json:"timestamp"`
}

// DetectionMetrics 单次检测的性能指标
type DetectionMetrics struct {
	EmailID        string                   `json:"email_id"`
	TotalLatency   time.Duration            `json:"total_latency"`
	LayerLatencies map[string]time.Duration `json:"layer_latencies"`
	IsSpam         bool                     `json:"is_spam"`
	Score          int                      `json:"score"`
	IsSlow         bool                     `json:"is_slow"`
	Timestamp      time.Time                `json:"timestamp"`
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(logRepo repository.SpamDetectionLogRepository) *PerformanceMonitor {
	return &PerformanceMonitor{
		logRepo:          logRepo,
		latencyThreshold: 200 * time.Millisecond,
		minLatency:       int64(^uint64(0) >> 1), // 初始化为最大值
		layerStats:       make(map[string]*LayerPerformanceStats),
		errorLogs:        make([]*PerformanceErrorLog, 0),
		maxErrorLogs:     1000,
		windowStats:      make([]*WindowStats, 0),
		windowSize:       time.Minute,
		maxWindowSlots:   60,  // 保留最近 60 分钟的数据
		alertThreshold:   5.0, // 5% 慢检测告警
		alertCooldown:    5 * time.Minute,
	}
}

// SetLatencyThreshold 设置延迟阈值
func (pm *PerformanceMonitor) SetLatencyThreshold(threshold time.Duration) {
	pm.latencyThreshold = threshold
}

// SetAlertCallback 设置告警回调
func (pm *PerformanceMonitor) SetAlertCallback(callback func(alert *PerformanceAlert)) {
	pm.alertCallback = callback
}

// SetAlertThreshold 设置告警阈值
func (pm *PerformanceMonitor) SetAlertThreshold(threshold float64) {
	pm.alertThreshold = threshold
}

// RecordDetection 记录检测性能
func (pm *PerformanceMonitor) RecordDetection(metrics *DetectionMetrics) {
	latencyNs := metrics.TotalLatency.Nanoseconds()

	// 更新总体统计
	atomic.AddInt64(&pm.totalDetections, 1)
	atomic.AddInt64(&pm.totalLatency, latencyNs)

	// 更新最大/最小延迟
	for {
		old := atomic.LoadInt64(&pm.maxLatency)
		if latencyNs <= old || atomic.CompareAndSwapInt64(&pm.maxLatency, old, latencyNs) {
			break
		}
	}
	for {
		old := atomic.LoadInt64(&pm.minLatency)
		if latencyNs >= old || atomic.CompareAndSwapInt64(&pm.minLatency, old, latencyNs) {
			break
		}
	}

	// 检查是否为慢检测
	if metrics.TotalLatency > pm.latencyThreshold {
		atomic.AddInt64(&pm.slowDetections, 1)
		metrics.IsSlow = true
		pm.logSlowDetection(metrics)
	}

	// 更新垃圾邮件统计
	if metrics.IsSpam {
		atomic.AddInt64(&pm.spamCount, 1)
	} else {
		atomic.AddInt64(&pm.hamCount, 1)
	}

	// 更新各层统计
	pm.updateLayerStats(metrics.LayerLatencies)

	// 更新时间窗口统计
	pm.updateWindowStats(metrics)

	// 检查是否需要告警
	pm.checkAlerts()
}

// RecordLayerPerformance 记录单层性能
func (pm *PerformanceMonitor) RecordLayerPerformance(layer string, latency time.Duration, err error, skipped bool) {
	pm.layerMu.Lock()
	defer pm.layerMu.Unlock()

	stats, exists := pm.layerStats[layer]
	if !exists {
		stats = &LayerPerformanceStats{
			Name:       layer,
			MinLatency: int64(^uint64(0) >> 1),
		}
		pm.layerStats[layer] = stats
	}

	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&stats.TotalCalls, 1)
	atomic.AddInt64(&stats.TotalLatency, latencyNs)

	if latencyNs > stats.MaxLatency {
		stats.MaxLatency = latencyNs
	}
	if latencyNs < stats.MinLatency {
		stats.MinLatency = latencyNs
	}

	if err != nil {
		atomic.AddInt64(&stats.ErrorCount, 1)
	}
	if skipped {
		atomic.AddInt64(&stats.SkippedCount, 1)
	}
}

// RecordError 记录错误
func (pm *PerformanceMonitor) RecordError(operation, layer, emailID string, err error, latency time.Duration) {
	atomic.AddInt64(&pm.errorCount, 1)

	errorLog := &PerformanceErrorLog{
		Timestamp: time.Now(),
		Operation: operation,
		Error:     err.Error(),
		LatencyMs: float64(latency.Milliseconds()),
		EmailID:   emailID,
		Layer:     layer,
	}

	pm.errorLogMu.Lock()
	defer pm.errorLogMu.Unlock()

	pm.errorLogs = append(pm.errorLogs, errorLog)

	// 限制错误日志数量
	if len(pm.errorLogs) > pm.maxErrorLogs {
		pm.errorLogs = pm.errorLogs[len(pm.errorLogs)-pm.maxErrorLogs:]
	}
}

// updateLayerStats 更新各层统计
func (pm *PerformanceMonitor) updateLayerStats(layerLatencies map[string]time.Duration) {
	pm.layerMu.Lock()
	defer pm.layerMu.Unlock()

	for layer, latency := range layerLatencies {
		stats, exists := pm.layerStats[layer]
		if !exists {
			stats = &LayerPerformanceStats{
				Name:       layer,
				MinLatency: int64(^uint64(0) >> 1),
			}
			pm.layerStats[layer] = stats
		}

		latencyNs := latency.Nanoseconds()
		atomic.AddInt64(&stats.TotalCalls, 1)
		atomic.AddInt64(&stats.TotalLatency, latencyNs)

		if latencyNs > stats.MaxLatency {
			stats.MaxLatency = latencyNs
		}
		if latencyNs < stats.MinLatency {
			stats.MinLatency = latencyNs
		}
	}
}

// updateWindowStats 更新时间窗口统计
func (pm *PerformanceMonitor) updateWindowStats(metrics *DetectionMetrics) {
	pm.windowMu.Lock()
	defer pm.windowMu.Unlock()

	now := time.Now().Truncate(pm.windowSize)

	// 查找或创建当前时间窗口
	var currentWindow *WindowStats
	if len(pm.windowStats) > 0 {
		lastWindow := pm.windowStats[len(pm.windowStats)-1]
		if lastWindow.Timestamp.Equal(now) {
			currentWindow = lastWindow
		}
	}

	if currentWindow == nil {
		currentWindow = &WindowStats{
			Timestamp: now,
		}
		pm.windowStats = append(pm.windowStats, currentWindow)

		// 限制窗口数量
		if len(pm.windowStats) > pm.maxWindowSlots {
			pm.windowStats = pm.windowStats[len(pm.windowStats)-pm.maxWindowSlots:]
		}
	}

	// 更新窗口统计
	currentWindow.DetectionCount++
	if metrics.IsSlow {
		currentWindow.SlowCount++
	}
	if metrics.IsSpam {
		currentWindow.SpamCount++
	} else {
		currentWindow.HamCount++
	}

	// 更新平均延迟
	totalLatency := currentWindow.AvgLatencyMs * float64(currentWindow.DetectionCount-1)
	totalLatency += float64(metrics.TotalLatency.Milliseconds())
	currentWindow.AvgLatencyMs = totalLatency / float64(currentWindow.DetectionCount)
}

// logSlowDetection 记录慢检测
func (pm *PerformanceMonitor) logSlowDetection(metrics *DetectionMetrics) {
	// 记录慢检测日志
	fmt.Printf("[性能警告] 慢检测: EmailID=%s, 延迟=%v, 阈值=%v\n",
		metrics.EmailID, metrics.TotalLatency, pm.latencyThreshold)

	// 记录各层延迟
	for layer, latency := range metrics.LayerLatencies {
		fmt.Printf("  - %s: %v\n", layer, latency)
	}
}

// checkAlerts 检查是否需要告警
func (pm *PerformanceMonitor) checkAlerts() {
	if pm.alertCallback == nil {
		return
	}

	// 检查告警冷却时间
	if time.Since(pm.lastAlertTime) < pm.alertCooldown {
		return
	}

	total := atomic.LoadInt64(&pm.totalDetections)
	slow := atomic.LoadInt64(&pm.slowDetections)

	if total < 100 {
		return // 样本量太小，不告警
	}

	slowRate := float64(slow) / float64(total) * 100

	if slowRate > pm.alertThreshold {
		alert := &PerformanceAlert{
			Type:      "slow_detection_rate",
			Message:   fmt.Sprintf("慢检测比例过高: %.2f%% (阈值: %.2f%%)", slowRate, pm.alertThreshold),
			Timestamp: time.Now(),
			Severity:  "warning",
			Metrics: map[string]interface{}{
				"slow_rate":        slowRate,
				"total_detections": total,
				"slow_detections":  slow,
				"threshold":        pm.alertThreshold,
			},
		}

		if slowRate > pm.alertThreshold*2 {
			alert.Severity = "critical"
		}

		pm.alertCallback(alert)
		pm.lastAlertTime = time.Now()
	}
}

// GetMetrics 获取性能指标汇总
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	total := atomic.LoadInt64(&pm.totalDetections)
	slow := atomic.LoadInt64(&pm.slowDetections)
	totalLatency := atomic.LoadInt64(&pm.totalLatency)
	maxLatency := atomic.LoadInt64(&pm.maxLatency)
	minLatency := atomic.LoadInt64(&pm.minLatency)
	spam := atomic.LoadInt64(&pm.spamCount)
	ham := atomic.LoadInt64(&pm.hamCount)
	errors := atomic.LoadInt64(&pm.errorCount)

	metrics := &PerformanceMetrics{
		TotalDetections: total,
		SlowDetections:  slow,
		SpamCount:       spam,
		HamCount:        ham,
		ErrorCount:      errors,
		Timestamp:       time.Now(),
	}

	if total > 0 {
		metrics.SlowRate = float64(slow) / float64(total) * 100
		metrics.AvgLatencyMs = float64(totalLatency) / float64(total) / 1e6
		metrics.SpamRate = float64(spam) / float64(total) * 100
		metrics.ErrorRate = float64(errors) / float64(total) * 100
	}

	if maxLatency > 0 {
		metrics.MaxLatencyMs = float64(maxLatency) / 1e6
	}
	if minLatency < int64(^uint64(0)>>1) {
		metrics.MinLatencyMs = float64(minLatency) / 1e6
	}

	// 复制各层统计
	pm.layerMu.RLock()
	metrics.LayerStats = make(map[string]*LayerPerformanceStats)
	for name, stats := range pm.layerStats {
		metrics.LayerStats[name] = &LayerPerformanceStats{
			Name:         stats.Name,
			TotalCalls:   atomic.LoadInt64(&stats.TotalCalls),
			TotalLatency: atomic.LoadInt64(&stats.TotalLatency),
			MaxLatency:   stats.MaxLatency,
			MinLatency:   stats.MinLatency,
			ErrorCount:   atomic.LoadInt64(&stats.ErrorCount),
			SkippedCount: atomic.LoadInt64(&stats.SkippedCount),
		}
	}
	pm.layerMu.RUnlock()

	return metrics
}

// GetLayerMetrics 获取指定层的性能指标
func (pm *PerformanceMonitor) GetLayerMetrics(layer string) *LayerPerformanceStats {
	pm.layerMu.RLock()
	defer pm.layerMu.RUnlock()

	stats, exists := pm.layerStats[layer]
	if !exists {
		return nil
	}

	return &LayerPerformanceStats{
		Name:         stats.Name,
		TotalCalls:   atomic.LoadInt64(&stats.TotalCalls),
		TotalLatency: atomic.LoadInt64(&stats.TotalLatency),
		MaxLatency:   stats.MaxLatency,
		MinLatency:   stats.MinLatency,
		ErrorCount:   atomic.LoadInt64(&stats.ErrorCount),
		SkippedCount: atomic.LoadInt64(&stats.SkippedCount),
	}
}

// GetWindowStats 获取时间窗口统计
func (pm *PerformanceMonitor) GetWindowStats(duration time.Duration) []*WindowStats {
	pm.windowMu.RLock()
	defer pm.windowMu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]*WindowStats, 0)

	for _, ws := range pm.windowStats {
		if ws.Timestamp.After(cutoff) {
			result = append(result, ws)
		}
	}

	return result
}

// GetRecentErrors 获取最近的错误日志
func (pm *PerformanceMonitor) GetRecentErrors(count int) []*PerformanceErrorLog {
	pm.errorLogMu.RLock()
	defer pm.errorLogMu.RUnlock()

	if count <= 0 || count > len(pm.errorLogs) {
		count = len(pm.errorLogs)
	}

	result := make([]*PerformanceErrorLog, count)
	copy(result, pm.errorLogs[len(pm.errorLogs)-count:])

	return result
}

// GetHealthStatus 获取健康状态
func (pm *PerformanceMonitor) GetHealthStatus() *HealthStatus {
	metrics := pm.GetMetrics()

	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 检查慢检测比例
	if metrics.SlowRate > pm.alertThreshold*2 {
		status.Status = "critical"
		status.Details["slow_rate"] = fmt.Sprintf("%.2f%% (阈值: %.2f%%)", metrics.SlowRate, pm.alertThreshold)
	} else if metrics.SlowRate > pm.alertThreshold {
		status.Status = "warning"
		status.Details["slow_rate"] = fmt.Sprintf("%.2f%% (阈值: %.2f%%)", metrics.SlowRate, pm.alertThreshold)
	}

	// 检查错误率
	if metrics.ErrorRate > 5 {
		status.Status = "critical"
		status.Details["error_rate"] = fmt.Sprintf("%.2f%%", metrics.ErrorRate)
	} else if metrics.ErrorRate > 1 {
		if status.Status == "healthy" {
			status.Status = "warning"
		}
		status.Details["error_rate"] = fmt.Sprintf("%.2f%%", metrics.ErrorRate)
	}

	// 检查平均延迟
	if metrics.AvgLatencyMs > float64(pm.latencyThreshold.Milliseconds()) {
		if status.Status == "healthy" {
			status.Status = "warning"
		}
		status.Details["avg_latency"] = fmt.Sprintf("%.2fms (阈值: %dms)", metrics.AvgLatencyMs, pm.latencyThreshold.Milliseconds())
	}

	status.Details["total_detections"] = metrics.TotalDetections
	status.Details["spam_rate"] = fmt.Sprintf("%.2f%%", metrics.SpamRate)

	return status
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string                 `json:"status"` // healthy, warning, critical
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// Reset 重置所有统计
func (pm *PerformanceMonitor) Reset() {
	atomic.StoreInt64(&pm.totalDetections, 0)
	atomic.StoreInt64(&pm.slowDetections, 0)
	atomic.StoreInt64(&pm.totalLatency, 0)
	atomic.StoreInt64(&pm.maxLatency, 0)
	atomic.StoreInt64(&pm.minLatency, int64(^uint64(0)>>1))
	atomic.StoreInt64(&pm.spamCount, 0)
	atomic.StoreInt64(&pm.hamCount, 0)
	atomic.StoreInt64(&pm.errorCount, 0)

	pm.layerMu.Lock()
	pm.layerStats = make(map[string]*LayerPerformanceStats)
	pm.layerMu.Unlock()

	pm.errorLogMu.Lock()
	pm.errorLogs = make([]*PerformanceErrorLog, 0)
	pm.errorLogMu.Unlock()

	pm.windowMu.Lock()
	pm.windowStats = make([]*WindowStats, 0)
	pm.windowMu.Unlock()
}

// SaveDetectionLog 保存检测日志到数据库
func (pm *PerformanceMonitor) SaveDetectionLog(ctx context.Context, log *model.SpamDetectionLog) error {
	if pm.logRepo == nil {
		return nil
	}
	return pm.logRepo.Create(ctx, log)
}

// GetDetectionLogs 获取检测日志
func (pm *PerformanceMonitor) GetDetectionLogs(ctx context.Context, startTime, endTime time.Time) ([]*model.SpamDetectionLog, error) {
	if pm.logRepo == nil {
		return nil, fmt.Errorf("日志仓库未初始化")
	}
	return pm.logRepo.FindByTimeRange(ctx, startTime, endTime)
}

// CleanupOldLogs 清理旧日志
func (pm *PerformanceMonitor) CleanupOldLogs(ctx context.Context, retentionDays int) error {
	if pm.logRepo == nil {
		return nil
	}
	before := time.Now().AddDate(0, 0, -retentionDays)
	return pm.logRepo.DeleteOldLogs(ctx, before)
}

// GetPerformanceSummary 获取性能摘要（用于仪表板显示）
func (pm *PerformanceMonitor) GetPerformanceSummary() *PerformanceSummary {
	metrics := pm.GetMetrics()
	health := pm.GetHealthStatus()

	// 获取最近 5 分钟的窗口统计
	recentWindows := pm.GetWindowStats(5 * time.Minute)
	var recentAvgLatency float64
	var recentSlowRate float64
	if len(recentWindows) > 0 {
		var totalLatency float64
		var totalDetections int64
		var totalSlow int64
		for _, w := range recentWindows {
			totalLatency += w.AvgLatencyMs * float64(w.DetectionCount)
			totalDetections += w.DetectionCount
			totalSlow += w.SlowCount
		}
		if totalDetections > 0 {
			recentAvgLatency = totalLatency / float64(totalDetections)
			recentSlowRate = float64(totalSlow) / float64(totalDetections) * 100
		}
	}

	return &PerformanceSummary{
		HealthStatus:       health.Status,
		TotalDetections:    metrics.TotalDetections,
		AvgLatencyMs:       metrics.AvgLatencyMs,
		RecentAvgLatencyMs: recentAvgLatency,
		SlowRate:           metrics.SlowRate,
		RecentSlowRate:     recentSlowRate,
		SpamRate:           metrics.SpamRate,
		ErrorRate:          metrics.ErrorRate,
		Timestamp:          time.Now(),
	}
}

// PerformanceSummary 性能摘要
type PerformanceSummary struct {
	HealthStatus       string    `json:"health_status"`
	TotalDetections    int64     `json:"total_detections"`
	AvgLatencyMs       float64   `json:"avg_latency_ms"`
	RecentAvgLatencyMs float64   `json:"recent_avg_latency_ms"` // 最近 5 分钟
	SlowRate           float64   `json:"slow_rate_percent"`
	RecentSlowRate     float64   `json:"recent_slow_rate_percent"` // 最近 5 分钟
	SpamRate           float64   `json:"spam_rate_percent"`
	ErrorRate          float64   `json:"error_rate_percent"`
	Timestamp          time.Time `json:"timestamp"`
}

// StartPeriodicCleanup 启动定期清理任务
func (pm *PerformanceMonitor) StartPeriodicCleanup(ctx context.Context, interval time.Duration, retentionDays int) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := pm.CleanupOldLogs(ctx, retentionDays); err != nil {
					fmt.Printf("[性能监控] 清理旧日志失败: %v\n", err)
				}
			}
		}
	}()
}
