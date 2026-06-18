package spam

import (
	"context"
	"time"
)

// GetPerformanceMetrics 获取性能指标
func (s *SpamDetector) GetPerformanceMetrics() *PerformanceMetrics {
	if s.performanceMonitor == nil {
		return &PerformanceMetrics{}
	}
	return s.performanceMonitor.GetMetrics()
}

// GetPerformanceSummary 获取性能摘要
func (s *SpamDetector) GetPerformanceSummary() *PerformanceSummary {
	if s.performanceMonitor == nil {
		return &PerformanceSummary{}
	}
	return s.performanceMonitor.GetPerformanceSummary()
}

// GetPerformanceHealth 获取性能健康状态
func (s *SpamDetector) GetPerformanceHealth() *HealthStatus {
	if s.performanceMonitor == nil {
		return &HealthStatus{Status: "unknown"}
	}
	return s.performanceMonitor.GetHealthStatus()
}

// GetLayerPerformance 获取指定层的性能指标
func (s *SpamDetector) GetLayerPerformance(layer string) *LayerPerformanceStats {
	if s.performanceMonitor == nil {
		return nil
	}
	return s.performanceMonitor.GetLayerMetrics(layer)
}

// GetWindowStats 获取时间窗口统计
func (s *SpamDetector) GetWindowStats(duration time.Duration) []*WindowStats {
	if s.performanceMonitor == nil {
		return nil
	}
	return s.performanceMonitor.GetWindowStats(duration)
}

// GetPerformanceErrors 获取性能相关错误
func (s *SpamDetector) GetPerformanceErrors(count int) []*PerformanceErrorLog {
	if s.performanceMonitor == nil {
		return nil
	}
	return s.performanceMonitor.GetRecentErrors(count)
}

// SetPerformanceAlertCallback 设置性能告警回调
func (s *SpamDetector) SetPerformanceAlertCallback(callback func(alert *PerformanceAlert)) {
	if s.performanceMonitor != nil {
		s.performanceMonitor.SetAlertCallback(callback)
	}
}

// SetLatencyThreshold 设置延迟阈值
func (s *SpamDetector) SetLatencyThreshold(threshold time.Duration) {
	if s.performanceMonitor != nil {
		s.performanceMonitor.SetLatencyThreshold(threshold)
	}
}

// StartPerformanceCleanup 启动性能日志定期清理
func (s *SpamDetector) StartPerformanceCleanup(ctx context.Context, interval time.Duration, retentionDays int) {
	if s.performanceMonitor != nil {
		s.performanceMonitor.StartPeriodicCleanup(ctx, interval, retentionDays)
	}
}
