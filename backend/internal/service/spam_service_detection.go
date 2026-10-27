package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"time"
)

// MarkAsSpam 标记邮件为垃圾邮件
func (s *spamService) MarkAsSpam(ctx context.Context, emailIDs []int64) error {
	for _, emailID := range emailIDs {
		// 获取邮件信息
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil || email == nil {
			spamServiceLog.Warn("无法找到邮件: id=%d, err=%v", emailID, err)
			continue
		}

		// 更新邮件的垃圾邮件状态
		email.IsSpam = true
		email.UserMarkedSpam = true
		now := time.Now()
		email.UserMarkedAt = &now

		if err := s.emailRepo.Update(ctx, email); err != nil {
			spamServiceLog.Warn("标记邮件为垃圾邮件失败: id=%d, err=%v", emailID, err)
			continue
		}

		// 异步更新发件人信誉（降低）
		if s.reputationManager != nil {
			go s.updateReputationForSpam(context.Background(), email.FromAddress, true)
		}

		// 异步添加贝叶斯训练数据
		if s.bayesianClassifier != nil {
			go s.addBayesianTraining(context.Background(), email, true)
		}
	}

	return nil
}

// UnmarkAsSpam 取消垃圾邮件标记
func (s *spamService) UnmarkAsSpam(ctx context.Context, emailIDs []int64) error {
	for _, emailID := range emailIDs {
		// 获取邮件信息
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil || email == nil {
			spamServiceLog.Warn("无法找到邮件: id=%d, err=%v", emailID, err)
			continue
		}

		// 取消垃圾邮件标记
		email.IsSpam = false
		email.UserMarkedSpam = false
		email.UserMarkedAt = nil

		if err := s.emailRepo.Update(ctx, email); err != nil {
			spamServiceLog.Warn("取消邮件垃圾邮件标记失败: id=%d, err=%v", emailID, err)
			continue
		}

		// 异步更新发件人信誉（提高）
		if s.reputationManager != nil {
			go s.updateReputationForSpam(context.Background(), email.FromAddress, false)
		}

		// 异步添加贝叶斯训练数据
		if s.bayesianClassifier != nil {
			go s.addBayesianTraining(context.Background(), email, false)
		}
	}

	return nil
}

// BatchDeleteSpam 批量删除垃圾邮件
func (s *spamService) BatchDeleteSpam(ctx context.Context, emailIDs []int64) (int64, error) {
	deletedCount := int64(0)

	for _, emailID := range emailIDs {
		// 先确认邮件存在：UpdateLocalStatus 走 UPDATE，命中 0 行同样返回 nil，
		// 不校验就会把不存在的 ID 计成「已删除」，前端看到的删除数会虚高。
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil || email == nil {
			spamServiceLog.Warn("待删除的邮件不存在: id=%d, err=%v", emailID, err)
			continue
		}

		// 软删除邮件
		deleted := true
		if err := s.emailRepo.UpdateLocalStatus(ctx, emailID, nil, nil, nil, &deleted); err != nil {
			spamServiceLog.Warn("删除邮件失败: id=%d, err=%v", emailID, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// EmptySpamFolder 清空垃圾箱
func (s *spamService) EmptySpamFolder(ctx context.Context, accountUID string) (int64, error) {
	// 构建过滤条件：查询所有垃圾邮件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 查询所有垃圾邮件
	emails, _, err := s.emailRepo.List(ctx, filter, 0, 10000) // 限制最多 10000 封
	if err != nil {
		return 0, fmt.Errorf("failed to list spam emails: %w", err)
	}

	// 批量删除
	deletedCount := int64(0)
	for _, email := range emails {
		deleted := true
		if err := s.emailRepo.UpdateLocalStatus(ctx, email.ID, nil, nil, nil, &deleted); err != nil {
			spamServiceLog.Warn("删除邮件失败: id=%d, err=%v", email.ID, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// updateReputationForSpam 更新发件人信誉（用户反馈）
func (s *spamService) updateReputationForSpam(ctx context.Context, senderEmail string, isSpam bool) {
	if s.reputationManager == nil {
		return
	}
	if err := s.reputationManager.UpdateReputationByUserFeedback(ctx, senderEmail, isSpam); err != nil {
		spamServiceLog.Warn("更新发件人信誉失败: sender=%s, err=%v", senderEmail, err)
	}
}

// addBayesianTraining 添加贝叶斯训练数据
func (s *spamService) addBayesianTraining(ctx context.Context, email *model.Email, isSpam bool) {
	if s.bayesianClassifier == nil {
		return
	}

	// 使用贝叶斯分类器的方法添加训练数据
	if err := s.bayesianClassifier.AddTrainingData(ctx, email.AccountUID, email, isSpam); err != nil {
		spamServiceLog.Warn("添加贝叶斯训练数据失败: emailId=%d, err=%v", email.ID, err)
	}
}
