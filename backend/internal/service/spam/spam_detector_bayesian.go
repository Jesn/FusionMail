package spam

import (
	"context"
	"fmt"

	"fusionmail/internal/model"
)

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
