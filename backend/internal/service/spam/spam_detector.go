package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"time"
)

// SpamDetector 垃圾邮件检测器主组件
type SpamDetector struct {
	whitelistChecker   *WhitelistChecker
	preFilter          *PreFilter
	ruleEngine         *RuleEngine
	reputationManager  *ReputationManager
	bayesianClassifier *BayesianClassifier
	logRepo            repository.SpamDetectionLogRepository
	spamThreshold      int // 垃圾邮件评分阈值，默认 60
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

// SetSpamThreshold 设置垃圾邮件评分阈值
func (s *SpamDetector) SetSpamThreshold(threshold int) {
	s.spamThreshold = threshold
}

// Detect 执行完整的垃圾邮件检测
func (s *SpamDetector) Detect(ctx context.Context, email *model.Email) (*SpamDetectionResult, error) {
	startTime := time.Now()

	result := &SpamDetectionResult{
		IsSpam:          false,
		Score:           0,
		Confidence:      0,
		Reasons:         make([]string, 0),
		DetectionLayers: make([]DetectionLayerInfo, 0),
		Timestamp:       time.Now(),
	}

	// 第 0 层：白名单/黑名单检查（优先级最高）
	// 注意：这里需要 userUID，暂时简化处理，后续需要从 email 中获取
	// whitelistStart := time.Now()
	// isWhitelisted, _ := s.whitelistChecker.CheckWhitelist(ctx, userUID, email.FromAddress)
	// isBlacklisted, _ := s.whitelistChecker.CheckBlacklist(ctx, userUID, email.FromAddress)
	// 暂时跳过白名单/黑名单检查，后续集成时补充

	// 第 1 层：预过滤层检测
	preFilterStart := time.Now()
	emailData := &EmailData{
		From:            email.FromAddress,
		Size:            0, // 暂时设为 0，后续从 email 中获取
		AttachmentTypes: []string{},
	}
	preFilterResult, err := s.preFilter.Filter(ctx, emailData)
	if err == nil {
		layerInfo := DetectionLayerInfo{
			Layer:       "PreFilter",
			Score:       preFilterResult.TotalScore,
			CheckedTime: time.Since(preFilterStart),
			Details:     fmt.Sprintf("预过滤评分: %d", preFilterResult.TotalScore),
		}
		result.DetectionLayers = append(result.DetectionLayers, layerInfo)
		result.Score += preFilterResult.TotalScore

		// 添加检测原因
		for _, detail := range preFilterResult.Details {
			result.Reasons = append(result.Reasons, detail)
		}
	}

	// 第 2 层：规则引擎检测
	ruleEngineStart := time.Now()
	ruleEngineResult, err := s.ruleEngine.Check(ctx, email)
	if err == nil {
		layerInfo := DetectionLayerInfo{
			Layer:       "RuleEngine",
			Score:       ruleEngineResult.Score,
			CheckedTime: time.Since(ruleEngineStart),
			Details:     fmt.Sprintf("命中 %d 条规则", len(ruleEngineResult.HitRules)),
		}
		result.DetectionLayers = append(result.DetectionLayers, layerInfo)
		result.Score += ruleEngineResult.Score

		// 添加命中的规则作为检测原因
		for _, hitRule := range ruleEngineResult.HitRules {
			reason := fmt.Sprintf("%s: %s (+%d分)", hitRule.Category, hitRule.RuleName, hitRule.Score)
			result.Reasons = append(result.Reasons, reason)
		}
	}

	// 第 3 层：发件人信誉调整
	reputationStart := time.Now()
	adjustedScore, err := s.reputationManager.AdjustScoreByReputation(ctx, email.FromAddress, result.Score)
	if err == nil {
		adjustment := adjustedScore - result.Score
		if adjustment != 0 {
			layerInfo := DetectionLayerInfo{
				Layer:       "Reputation",
				Score:       adjustment,
				CheckedTime: time.Since(reputationStart),
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
	}

	// 第 4 层：贝叶斯分类
	if s.bayesianClassifier != nil {
		bayesianStart := time.Now()
		// 从 email 中获取 userUID（通过 AccountUID 关联）
		userUID := email.AccountUID // 使用账户 UID 作为用户标识
		if userUID != "" {
			bayesianResult, err := s.bayesianClassifier.Classify(ctx, userUID, email)
			if err == nil && bayesianResult.ModelUsed {
				layerInfo := DetectionLayerInfo{
					Layer:       "Bayesian",
					Score:       bayesianResult.Score,
					CheckedTime: time.Since(bayesianStart),
					Details:     bayesianResult.Description,
				}
				result.DetectionLayers = append(result.DetectionLayers, layerInfo)
				result.Score += bayesianResult.Score

				// 添加贝叶斯分类结果作为检测原因
				if bayesianResult.Score != 0 {
					result.Reasons = append(result.Reasons, bayesianResult.Description)
				}
			}
		}
	}

	// 计算置信度
	result.Confidence = s.calculateConfidence(result.Score)

	// 判定是否为垃圾邮件
	result.IsSpam = result.Score >= s.spamThreshold

	// 记录总耗时
	result.CheckedTime = time.Since(startTime)

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

	log := &model.SpamDetectionLog{
		EmailID:          fmt.Sprintf("%d", email.ID),
		IsSpam:           result.IsSpam,
		FinalScore:       float64(result.Score),
		DetectionDetails: reasons,
		ProcessingTimeMs: result.CheckedTime.Milliseconds(),
	}

	// 异步保存日志，不阻塞主流程
	go func() {
		if err := s.logRepo.Create(context.Background(), log); err != nil {
			fmt.Printf("警告: 保存垃圾邮件检测日志失败: %v\n", err)
		}
	}()
}

// updateReputationAsync 异步更新发件人信誉
func (s *SpamDetector) updateReputationAsync(ctx context.Context, senderEmail string, spamScore int) {
	if err := s.reputationManager.UpdateReputationByDetection(ctx, senderEmail, spamScore); err != nil {
		fmt.Printf("警告: 更新发件人信誉失败 [%s]: %v\n", senderEmail, err)
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
