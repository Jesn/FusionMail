package spam

import (
	"time"

	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// 模块日志记录器
var spamDetectorLog = logger.NewWithModule("SpamDetector")

// SpamDetector 垃圾邮件检测器主组件
type SpamDetector struct {
	whitelistChecker   *WhitelistChecker
	preFilter          *PreFilter
	ruleEngine         *RuleEngine
	reputationManager  *ReputationManager
	bayesianClassifier *BayesianClassifier
	logRepo            repository.SpamDetectionLogRepository
	spamThreshold      int                 // 垃圾邮件评分阈值，默认 60
	performanceMonitor *PerformanceMonitor // 性能监控器
}

// SpamDetectionResult 垃圾邮件检测结果
type SpamDetectionResult struct {
	IsSpam          bool                 `json:"is_spam"`          // 是否为垃圾邮件
	Score           int                  `json:"score"`            // 总评分
	Confidence      float64              `json:"confidence"`       // 置信度 (0-1)
	Reasons         []string             `json:"reasons"`          // 检测原因列表
	DetectionLayers []DetectionLayerInfo `json:"detection_layers"` // 各层检测信息
	CheckedTime     time.Duration        `json:"checked_time"`     // 检测耗时
	Timestamp       time.Time            `json:"timestamp"`        // 检测时间
}

// DetectionLayerInfo 检测层信息
type DetectionLayerInfo struct {
	Layer       string        `json:"layer"`        // 层级名称
	Score       int           `json:"score"`        // 该层评分
	Details     string        `json:"details"`      // 详细信息
	CheckedTime time.Duration `json:"checked_time"` // 该层耗时
}

// NewSpamDetector 创建垃圾邮件检测器
func NewSpamDetector(
	whitelistChecker *WhitelistChecker,
	preFilter *PreFilter,
	ruleEngine *RuleEngine,
	reputationManager *ReputationManager,
	bayesianClassifier *BayesianClassifier,
	logRepo repository.SpamDetectionLogRepository,
) *SpamDetector {
	return &SpamDetector{
		whitelistChecker:   whitelistChecker,
		preFilter:          preFilter,
		ruleEngine:         ruleEngine,
		reputationManager:  reputationManager,
		bayesianClassifier: bayesianClassifier,
		logRepo:            logRepo,
		spamThreshold:      60, // 默认阈值 60 分
	}
}

// NewSpamDetectorWithMonitor 创建带性能监控的垃圾邮件检测器
func NewSpamDetectorWithMonitor(
	whitelistChecker *WhitelistChecker,
	preFilter *PreFilter,
	ruleEngine *RuleEngine,
	reputationManager *ReputationManager,
	bayesianClassifier *BayesianClassifier,
	logRepo repository.SpamDetectionLogRepository,
	performanceMonitor *PerformanceMonitor,
) *SpamDetector {
	if performanceMonitor == nil {
		performanceMonitor = NewPerformanceMonitor(logRepo)
	}
	return &SpamDetector{
		whitelistChecker:   whitelistChecker,
		preFilter:          preFilter,
		ruleEngine:         ruleEngine,
		reputationManager:  reputationManager,
		bayesianClassifier: bayesianClassifier,
		logRepo:            logRepo,
		spamThreshold:      60,
		performanceMonitor: performanceMonitor,
	}
}

// SetSpamThreshold 设置垃圾邮件评分阈值
func (s *SpamDetector) SetSpamThreshold(threshold int) {
	s.spamThreshold = threshold
}
