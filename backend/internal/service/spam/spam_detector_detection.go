package spam

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/model"
)

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
