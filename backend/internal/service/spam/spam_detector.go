package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
	"time"
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
	spamThreshold      int // 垃圾邮件评分阈值，默认 60
	cacheManager       *CacheManager
	fallbackManager    *FallbackManager
	errorHandler       *ErrorHandler
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

// NewSpamDetectorWithFallback 创建带降级策略的垃圾邮件检测器
func NewSpamDetectorWithFallback(
	whitelistChecker *WhitelistChecker,
	preFilter *PreFilter,
	ruleEngine *RuleEngine,
	reputationManager *ReputationManager,
	bayesianClassifier *BayesianClassifier,
	logRepo repository.SpamDetectionLogRepository,
	cacheManager *CacheManager,
	fallbackManager *FallbackManager,
) *SpamDetector {
	return &SpamDetector{
		whitelistChecker:   whitelistChecker,
		preFilter:          preFilter,
		ruleEngine:         ruleEngine,
		reputationManager:  reputationManager,
		bayesianClassifier: bayesianClassifier,
		logRepo:            logRepo,
		spamThreshold:      60,
		cacheManager:       cacheManager,
		fallbackManager:    fallbackManager,
		errorHandler:       NewErrorHandler(nil),
	}
}

// NewSpamDetectorFull 创建完整配置的垃圾邮件检测器
func NewSpamDetectorFull(
	whitelistChecker *WhitelistChecker,
	preFilter *PreFilter,
	ruleEngine *RuleEngine,
	reputationManager *ReputationManager,
	bayesianClassifier *BayesianClassifier,
	logRepo repository.SpamDetectionLogRepository,
	cacheManager *CacheManager,
	fallbackManager *FallbackManager,
	errorHandler *ErrorHandler,
) *SpamDetector {
	if errorHandler == nil {
		errorHandler = NewErrorHandler(nil)
	}
	return &SpamDetector{
		whitelistChecker:   whitelistChecker,
		preFilter:          preFilter,
		ruleEngine:         ruleEngine,
		reputationManager:  reputationManager,
		bayesianClassifier: bayesianClassifier,
		logRepo:            logRepo,
		spamThreshold:      60,
		cacheManager:       cacheManager,
		fallbackManager:    fallbackManager,
		errorHandler:       errorHandler,
		performanceMonitor: NewPerformanceMonitor(logRepo),
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
	cacheManager *CacheManager,
	fallbackManager *FallbackManager,
	errorHandler *ErrorHandler,
	performanceMonitor *PerformanceMonitor,
) *SpamDetector {
	if errorHandler == nil {
		errorHandler = NewErrorHandler(nil)
	}
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
		cacheManager:       cacheManager,
		fallbackManager:    fallbackManager,
		errorHandler:       errorHandler,
		performanceMonitor: performanceMonitor,
	}
}

// SetSpamThreshold 设置垃圾邮件评分阈值
func (s *SpamDetector) SetSpamThreshold(threshold int) {
	s.spamThreshold = threshold
}

// Detect 执行完整的垃圾邮件检测
func (s *SpamDetector) Detect(ctx context.Context, email *model.Email) (*SpamDetectionResult, error) {
	startTime := time.Now()
	layerLatencies := make(map[string]time.Duration)

	result := &SpamDetectionResult{
		IsSpam:          false,
		Score:           0,
		Confidence:      0,
		Reasons:         make([]string, 0),
		DetectionLayers: make([]DetectionLayerInfo, 0),
		Timestamp:       time.Now(),
	}

	// 第 0 层：白名单/黑名单检查（优先级最高）
	// 使用 AccountUID 作为用户标识
	userUID := email.AccountUID
	if s.whitelistChecker != nil && userUID != "" {
		whitelistStart := time.Now()

		// 检查白名单
		isWhitelisted, _ := s.whitelistChecker.CheckWhitelist(ctx, userUID, email.FromAddress)
		if isWhitelisted {
			whitelistLatency := time.Since(whitelistStart)
			layerLatencies["Whitelist"] = whitelistLatency

			// 白名单优先放行，直接返回非垃圾邮件结果
			result.IsSpam = false
			result.Score = 0
			result.Confidence = 0
			result.Reasons = append(result.Reasons, "发件人在白名单中，直接放行")
			result.DetectionLayers = append(result.DetectionLayers, DetectionLayerInfo{
				Layer:       "Whitelist",
				Score:       0,
				CheckedTime: whitelistLatency,
				Details:     "白名单匹配，跳过后续检测",
			})
			result.CheckedTime = time.Since(startTime)

			// 记录检测日志
			s.logDetection(ctx, email, result)
			return result, nil
		}

		// 检查黑名单
		isBlacklisted, _ := s.whitelistChecker.CheckBlacklist(ctx, userUID, email.FromAddress)
		if isBlacklisted {
			blacklistLatency := time.Since(whitelistStart)
			layerLatencies["Blacklist"] = blacklistLatency

			// 黑名单直接拦截，标记为垃圾邮件
			result.IsSpam = true
			result.Score = 100 // 黑名单直接给最高分
			result.Confidence = 1.0
			result.Reasons = append(result.Reasons, "发件人在黑名单中，直接拦截")
			result.DetectionLayers = append(result.DetectionLayers, DetectionLayerInfo{
				Layer:       "Blacklist",
				Score:       100,
				CheckedTime: blacklistLatency,
				Details:     "黑名单匹配，直接标记为垃圾邮件",
			})
			result.CheckedTime = time.Since(startTime)

			// 记录检测日志
			s.logDetection(ctx, email, result)
			return result, nil
		}

		// 记录白名单/黑名单检查耗时
		if s.performanceMonitor != nil {
			s.performanceMonitor.RecordLayerPerformance("WhitelistBlacklist", time.Since(whitelistStart), nil, false)
		}
	}

	// 第 1 层：预过滤层检测
	preFilterStart := time.Now()

	// 从邮件中提取附件类型
	attachmentTypes := make([]string, 0)
	if email.Attachments != nil {
		for _, att := range email.Attachments {
			if att.ContentType != "" {
				attachmentTypes = append(attachmentTypes, att.ContentType)
			}
		}
	}

	emailData := &EmailData{
		From:            email.FromAddress,
		Size:            email.SizeBytes,
		AttachmentTypes: attachmentTypes,
	}
	preFilterResult, err := s.preFilter.Filter(ctx, emailData)
	preFilterLatency := time.Since(preFilterStart)
	layerLatencies["PreFilter"] = preFilterLatency

	// 记录预过滤层性能
	if s.performanceMonitor != nil {
		s.performanceMonitor.RecordLayerPerformance("PreFilter", preFilterLatency, err, false)
	}

	if err == nil {
		layerInfo := DetectionLayerInfo{
			Layer:       "PreFilter",
			Score:       preFilterResult.TotalScore,
			CheckedTime: preFilterLatency,
			Details:     fmt.Sprintf("预过滤评分: %d", preFilterResult.TotalScore),
		}
		result.DetectionLayers = append(result.DetectionLayers, layerInfo)
		result.Score += preFilterResult.TotalScore

		// 添加检测原因
		for _, detail := range preFilterResult.Details {
			result.Reasons = append(result.Reasons, detail)
		}
	} else if s.performanceMonitor != nil {
		s.performanceMonitor.RecordError("pre_filter", "PreFilter", fmt.Sprintf("%d", email.ID), err, preFilterLatency)
	}

	// 第 2 层：规则引擎检测
	ruleEngineStart := time.Now()
	ruleEngineResult, err := s.ruleEngine.Check(ctx, email)
	ruleEngineLatency := time.Since(ruleEngineStart)
	layerLatencies["RuleEngine"] = ruleEngineLatency

	// 记录规则引擎层性能
	if s.performanceMonitor != nil {
		s.performanceMonitor.RecordLayerPerformance("RuleEngine", ruleEngineLatency, err, false)
	}

	if err == nil {
		layerInfo := DetectionLayerInfo{
			Layer:       "RuleEngine",
			Score:       ruleEngineResult.Score,
			CheckedTime: ruleEngineLatency,
			Details:     fmt.Sprintf("命中 %d 条规则", len(ruleEngineResult.HitRules)),
		}
		result.DetectionLayers = append(result.DetectionLayers, layerInfo)
		result.Score += ruleEngineResult.Score

		// 添加命中的规则作为检测原因
		for _, hitRule := range ruleEngineResult.HitRules {
			reason := fmt.Sprintf("%s: %s (+%d分)", hitRule.Category, hitRule.RuleName, hitRule.Score)
			result.Reasons = append(result.Reasons, reason)
		}
	} else if s.performanceMonitor != nil {
		s.performanceMonitor.RecordError("rule_engine", "RuleEngine", fmt.Sprintf("%d", email.ID), err, ruleEngineLatency)
	}

	// 第 3 层：发件人信誉调整
	reputationStart := time.Now()
	adjustedScore, err := s.reputationManager.AdjustScoreByReputation(ctx, email.FromAddress, result.Score)
	reputationLatency := time.Since(reputationStart)
	layerLatencies["Reputation"] = reputationLatency

	// 记录信誉层性能
	if s.performanceMonitor != nil {
		s.performanceMonitor.RecordLayerPerformance("Reputation", reputationLatency, err, false)
	}

	if err == nil {
		adjustment := adjustedScore - result.Score
		if adjustment != 0 {
			layerInfo := DetectionLayerInfo{
				Layer:       "Reputation",
				Score:       adjustment,
				CheckedTime: reputationLatency,
				Details:     fmt.Sprintf("根据发件人信誉调整评分 %+d分", adjustment),
			}
			result.DetectionLayers = append(result.DetectionLayers, layerInfo)
			result.Score = adjustedScore

			if adjustment > 0 {
				result.Reasons = append(result.Reasons, fmt.Sprintf("低信誉发件人 (+%d分)", adjustment))
			} else if adjustment < 0 {
				result.Reasons = append(result.Reasons, fmt.Sprintf("高信誉发件人 (%d分)", adjustment))
			}
		}
	} else if s.performanceMonitor != nil {
		s.performanceMonitor.RecordError("reputation", "Reputation", fmt.Sprintf("%d", email.ID), err, reputationLatency)
	}

	// 第 4 层：贝叶斯分类（支持高负载降级）
	if s.bayesianClassifier != nil {
		// 检查是否应该跳过贝叶斯分类（高负载时）
		shouldSkipBayesian := false
		if s.fallbackManager != nil {
			shouldSkipBayesian = s.fallbackManager.ShouldSkipBayesian()
		}

		if !shouldSkipBayesian {
			bayesianStart := time.Now()
			// 从 email 中获取 userUID（通过 AccountUID 关联）
			userUID := email.AccountUID // 使用账户 UID 作为用户标识
			if userUID != "" {
				bayesianResult, err := s.bayesianClassifier.Classify(ctx, userUID, email)
				bayesianLatency := time.Since(bayesianStart)
				layerLatencies["Bayesian"] = bayesianLatency

				// 记录贝叶斯层性能
				if s.performanceMonitor != nil {
					s.performanceMonitor.RecordLayerPerformance("Bayesian", bayesianLatency, err, false)
				}

				if err == nil && bayesianResult.ModelUsed {
					layerInfo := DetectionLayerInfo{
						Layer:       "Bayesian",
						Score:       bayesianResult.Score,
						CheckedTime: bayesianLatency,
						Details:     bayesianResult.Description,
					}
					result.DetectionLayers = append(result.DetectionLayers, layerInfo)
					result.Score += bayesianResult.Score

					// 添加贝叶斯分类结果作为检测原因
					if bayesianResult.Score != 0 {
						result.Reasons = append(result.Reasons, bayesianResult.Description)
					}
				} else if err != nil && s.performanceMonitor != nil {
					s.performanceMonitor.RecordError("bayesian", "Bayesian", fmt.Sprintf("%d", email.ID), err, bayesianLatency)
				}
			}
		} else {
			// 记录跳过贝叶斯分类的原因
			result.Reasons = append(result.Reasons, "高负载模式：跳过贝叶斯分类")
			// 记录被跳过的性能指标
			if s.performanceMonitor != nil {
				s.performanceMonitor.RecordLayerPerformance("Bayesian", 0, nil, true)
			}
		}
	}

	// 计算置信度
	result.Confidence = s.calculateConfidence(result.Score)

	// 判定是否为垃圾邮件
	result.IsSpam = result.Score >= s.spamThreshold

	// 记录总耗时
	result.CheckedTime = time.Since(startTime)

	// 记录性能指标
	if s.performanceMonitor != nil {
		metrics := &DetectionMetrics{
			EmailID:        fmt.Sprintf("%d", email.ID),
			TotalLatency:   result.CheckedTime,
			LayerLatencies: layerLatencies,
			IsSpam:         result.IsSpam,
			Score:          result.Score,
			Timestamp:      time.Now(),
		}
		s.performanceMonitor.RecordDetection(metrics)
	}

	// 记录检测日志
	s.logDetection(ctx, email, result)

	// 异步更新发件人信誉
	go s.updateReputationAsync(context.Background(), email.FromAddress, result.Score)

	return result, nil
}

// calculateConfidence 计算置信度
func (s *SpamDetector) calculateConfidence(score int) float64 {
	// 简单的线性映射：0-100 分映射到 0-1.0
	confidence := float64(score) / 100.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0 {
		confidence = 0
	}
	return confidence
}

// logDetection 记录检测日志
func (s *SpamDetector) logDetection(ctx context.Context, email *model.Email, result *SpamDetectionResult) {
	// 构造检测原因字符串
	reasons := ""
	for i, reason := range result.Reasons {
		if i > 0 {
			reasons += "; "
		}
		reasons += reason
	}

	// 构造检测层级信息
	layers := ""
	for i, layer := range result.DetectionLayers {
		if i > 0 {
			layers += ", "
		}
		layers += fmt.Sprintf("%s(%d分)", layer.Layer, layer.Score)
	}

	detectionLog := &model.SpamDetectionLog{
		EmailID:          fmt.Sprintf("%d", email.ID),
		IsSpam:           result.IsSpam,
		FinalScore:       float64(result.Score),
		DetectionDetails: reasons,
		ProcessingTimeMs: result.CheckedTime.Milliseconds(),
	}

	// 异步保存日志，不阻塞主流程
	go func() {
		if err := s.logRepo.Create(context.Background(), detectionLog); err != nil {
			spamDetectorLog.Warn("保存垃圾邮件检测日志失败: %v", err)
		}
	}()
}

// updateReputationAsync 异步更新发件人信誉
func (s *SpamDetector) updateReputationAsync(ctx context.Context, senderEmail string, spamScore int) {
	if err := s.reputationManager.UpdateReputationByDetection(ctx, senderEmail, spamScore); err != nil {
		spamDetectorLog.Warn("更新发件人信誉失败: sender=%s, err=%v", senderEmail, err)
	}
}

// BatchDetect 批量检测邮件
func (s *SpamDetector) BatchDetect(ctx context.Context, emails []*model.Email) ([]*SpamDetectionResult, error) {
	results := make([]*SpamDetectionResult, len(emails))

	// 简单的串行处理，后续可以优化为并发处理
	for i, email := range emails {
		result, err := s.Detect(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to detect email %d: %w", email.ID, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetDetectionStats 获取检测统计信息
func (s *SpamDetector) GetDetectionStats(ctx context.Context, startTime, endTime time.Time) (*DetectionStats, error) {
	// 从日志表查询统计信息
	logs, err := s.logRepo.FindByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	stats := &DetectionStats{
		TotalChecked: int64(len(logs)),
		SpamCount:    0,
		HamCount:     0,
		AvgScore:     0,
		AvgCheckTime: 0,
	}

	totalScore := 0.0
	for _, log := range logs {
		if log.IsSpam {
			stats.SpamCount++
		} else {
			stats.HamCount++
		}
		totalScore += log.FinalScore
	}

	if stats.TotalChecked > 0 {
		stats.AvgScore = totalScore / float64(stats.TotalChecked)
		stats.SpamRate = float64(stats.SpamCount) / float64(stats.TotalChecked) * 100
	}

	return stats, nil
}

// DetectionStats 检测统计信息
type DetectionStats struct {
	TotalChecked int64   `json:"total_checked"`  // 总检测数
	SpamCount    int64   `json:"spam_count"`     // 垃圾邮件数
	HamCount     int64   `json:"ham_count"`      // 正常邮件数
	SpamRate     float64 `json:"spam_rate"`      // 垃圾邮件率
	AvgScore     float64 `json:"avg_score"`      // 平均评分
	AvgCheckTime float64 `json:"avg_check_time"` // 平均检测耗时（毫秒）
}

// SpamSimpleResult 简化的垃圾邮件检测结果（用于同步服务集成）
type SpamSimpleResult struct {
	IsSpam     bool
	Score      int
	Confidence float64
	Reason     string
	DetectedBy string
}

// GetIsSpam 获取是否为垃圾邮件
func (r *SpamSimpleResult) GetIsSpam() bool { return r.IsSpam }

// GetScore 获取评分
func (r *SpamSimpleResult) GetScore() int { return r.Score }

// GetConfidence 获取置信度
func (r *SpamSimpleResult) GetConfidence() float64 { return r.Confidence }

// GetReason 获取原因
func (r *SpamSimpleResult) GetReason() string { return r.Reason }

// GetDetectedBy 获取检测层级
func (r *SpamSimpleResult) GetDetectedBy() string { return r.DetectedBy }

// AddTrainingData 添加贝叶斯训练数据（用户标记邮件时调用）
func (s *SpamDetector) AddTrainingData(ctx context.Context, userUID string, email *model.Email, isSpam bool) error {
	if s.bayesianClassifier == nil {
		return fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.AddTrainingData(ctx, userUID, email, isSpam)
}

// GetBayesianStatus 获取贝叶斯模型状态
func (s *SpamDetector) GetBayesianStatus(ctx context.Context, userUID string) (*ModelStatus, error) {
	if s.bayesianClassifier == nil {
		return nil, fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.GetModelStatus(ctx, userUID)
}

// TrainBayesianModel 手动训练贝叶斯模型
func (s *SpamDetector) TrainBayesianModel(ctx context.Context, userUID string) error {
	if s.bayesianClassifier == nil {
		return fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.Train(ctx, userUID)
}

// ResetBayesianModel 重置贝叶斯模型
func (s *SpamDetector) ResetBayesianModel(ctx context.Context, userUID string) error {
	if s.bayesianClassifier == nil {
		return fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.Reset(ctx, userUID)
}

// GetBayesianTrainingStats 获取贝叶斯训练统计
func (s *SpamDetector) GetBayesianTrainingStats(ctx context.Context, userUID string) (*TrainingStats, error) {
	if s.bayesianClassifier == nil {
		return nil, fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.GetTrainingStats(ctx, userUID)
}

// DetectSpamSimple 返回简化的检测结果，用于同步服务集成
func (s *SpamDetector) DetectSpamSimple(ctx context.Context, email *model.Email) (*SpamSimpleResult, error) {
	result, err := s.Detect(ctx, email)
	if err != nil {
		return nil, err
	}

	// 构造检测原因字符串
	reason := ""
	for i, r := range result.Reasons {
		if i > 0 {
			reason += "; "
		}
		reason += r
	}

	// 构造检测层级字符串
	detectedBy := ""
	for i, layer := range result.DetectionLayers {
		if i > 0 {
			detectedBy += ", "
		}
		detectedBy += layer.Layer
	}

	return &SpamSimpleResult{
		IsSpam:     result.IsSpam,
		Score:      result.Score,
		Confidence: result.Confidence,
		Reason:     reason,
		DetectedBy: detectedBy,
	}, nil
}

// ==================== 带错误处理的安全检测方法 ====================

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

// GetCacheStats 获取缓存统计信息
func (s *SpamDetector) GetCacheStats() *CacheStats {
	if s.cacheManager == nil {
		return &CacheStats{}
	}
	return s.cacheManager.GetStats()
}

// GetFallbackStats 获取降级统计信息
func (s *SpamDetector) GetFallbackStats() *FallbackStats {
	if s.fallbackManager == nil {
		return &FallbackStats{}
	}
	return s.fallbackManager.GetStats()
}

// GetSystemHealth 获取系统健康状态
func (s *SpamDetector) GetSystemHealth() *SystemHealth {
	health := &SystemHealth{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	// 检查各服务状态
	if s.fallbackManager != nil {
		services := s.fallbackManager.GetAllServicesHealth()
		for name, svc := range services {
			if svc.IsHealthy {
				health.Services[name] = "healthy"
			} else if svc.CircuitOpen {
				health.Services[name] = "circuit_open"
				health.Status = "degraded"
			} else {
				health.Services[name] = "unhealthy"
				health.Status = "degraded"
			}
		}
	}

	// 检查缓存状态
	if s.cacheManager != nil {
		hitRate := s.cacheManager.GetHitRate()
		if hitRate < 50 {
			health.Services["cache"] = "low_hit_rate"
		} else {
			health.Services["cache"] = "healthy"
		}
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
	if s.cacheManager != nil {
		s.cacheManager.ResetStats()
	}
	if s.fallbackManager != nil {
		s.fallbackManager.ResetAllServices()
	}
	if s.performanceMonitor != nil {
		s.performanceMonitor.Reset()
	}
}

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
