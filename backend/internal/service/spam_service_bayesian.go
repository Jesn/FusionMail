package service

import (
	"context"
	"fmt"
	"fusionmail/internal/service/spam"
)

// GetBayesianStatus 获取贝叶斯模型状态
func (s *spamService) GetBayesianStatus(ctx context.Context, userUID string) (*spam.ModelStatus, error) {
	if s.bayesianClassifier == nil {
		return nil, fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.GetModelStatus(ctx, userUID)
}

// TrainBayesianModel 手动训练贝叶斯模型
func (s *spamService) TrainBayesianModel(ctx context.Context, userUID string) error {
	if s.bayesianClassifier == nil {
		return fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.Train(ctx, userUID)
}

// ResetBayesianModel 重置贝叶斯模型
func (s *spamService) ResetBayesianModel(ctx context.Context, userUID string) error {
	if s.bayesianClassifier == nil {
		return fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.Reset(ctx, userUID)
}

// GetBayesianTrainingStats 获取贝叶斯训练统计
func (s *spamService) GetBayesianTrainingStats(ctx context.Context, userUID string) (*spam.TrainingStats, error) {
	if s.bayesianClassifier == nil {
		return nil, fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.GetTrainingStats(ctx, userUID)
}
