package spam

import (
	"context"
	"time"

	"fusionmail/internal/model"
)

// DetectSafe 安全的垃圾邮件检测（带错误处理和重试）
func (s *SpamDetector) DetectSafe(ctx context.Context, email *model.Email) (*SpamDetectionResult, error) {
	if s.errorHandler == nil {
		// 没有错误处理器，直接调用普通检测
		return s.Detect(ctx, email)
	}

	// 使用错误处理器执行检测
	result, err := s.errorHandler.SafeExecute(ctx, "spam_detection", func(ctx context.Context) (interface{}, error) {
		return s.Detect(ctx, email)
	})

	if err != nil {
		// 检测失败，返回默认结果（降级策略）
		return &SpamDetectionResult{
			IsSpam:          false,
			Score:           0,
			Confidence:      0,
			Reasons:         []string{"检测失败，使用默认结果"},
			DetectionLayers: []DetectionLayerInfo{},
			CheckedTime:     0,
			Timestamp:       time.Now(),
		}, nil
	}

	return result.(*SpamDetectionResult), nil
}

// DetectWithRetry 带重试的垃圾邮件检测
func (s *SpamDetector) DetectWithRetry(ctx context.Context, email *model.Email) (*SpamDetectionResult, error) {
	if s.errorHandler == nil {
		return s.Detect(ctx, email)
	}

	result, err := s.errorHandler.ExecuteWithRetry(ctx, "spam_detection", func(ctx context.Context) (interface{}, error) {
		return s.Detect(ctx, email)
	})

	if err != nil {
		return nil, err
	}

	return result.(*SpamDetectionResult), nil
}

// GetErrorStats 获取错误统计信息
func (s *SpamDetector) GetErrorStats() *ErrorStats {
	if s.errorHandler == nil {
		return &ErrorStats{}
	}
	return s.errorHandler.GetErrorStats()
}

// GetRecentErrors 获取最近的错误日志
func (s *SpamDetector) GetRecentErrors(count int) []*ErrorLogEntry {
	if s.errorHandler == nil {
		return []*ErrorLogEntry{}
	}
	return s.errorHandler.GetRecentErrors(count)
}

// GetSystemHealth 获取系统健康状态
func (s *SpamDetector) GetSystemHealth() *SystemHealth {
	health := &SystemHealth{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	return health
}

// SystemHealth 系统健康状态
type SystemHealth struct {
	Status    string            `json:"status"` // healthy, degraded, unhealthy
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// ResetAllStats 重置所有统计信息
func (s *SpamDetector) ResetAllStats() {
	if s.errorHandler != nil {
		s.errorHandler.ClearErrorLog()
	}
	if s.performanceMonitor != nil {
		s.performanceMonitor.Reset()
	}
}
