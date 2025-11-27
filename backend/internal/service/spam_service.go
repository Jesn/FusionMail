package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"log"
	"time"
)

// SpamService 垃圾邮件服务接口
type SpamService interface {
	// 垃圾邮件标记
	MarkAsSpam(ctx context.Context, emailIDs []int64) error
	UnmarkAsSpam(ctx context.Context, emailIDs []int64) error

	// 垃圾邮件管理
	BatchDeleteSpam(ctx context.Context, emailIDs []int64) (int64, error)
	EmptySpamFolder(ctx context.Context, accountUID string) (int64, error)

	// 垃圾邮件查询
	GetSpamEmails(ctx context.Context, accountUID string, page, pageSize int) ([]*model.Email, int64, error)
	GetSpamStats(ctx context.Context, accountUID string) (*SpamStats, error)
}

// SpamStats 垃圾邮件统计
type SpamStats struct {
	TotalCount   int64 `json:"total_count"`   // 总垃圾邮件数
	UnreadCount  int64 `json:"unread_count"`  // 未读垃圾邮件数
	TodayCount   int64 `json:"today_count"`   // 今日垃圾邮件数
	WeekCount    int64 `json:"week_count"`    // 本周垃圾邮件数
	MonthCount   int64 `json:"month_count"`   // 本月垃圾邮件数
	BlockedCount int64 `json:"blocked_count"` // 拦截的垃圾邮件数
}

// spamService 垃圾邮件服务实现
type spamService struct {
	emailRepo            repository.EmailRepository
	reputationManager    *spam.ReputationManager
	bayesianTrainingRepo repository.BayesianTrainingRepository
}

// NewSpamService 创建垃圾邮件服务
func NewSpamService(
	emailRepo repository.EmailRepository,
	reputationManager *spam.ReputationManager,
	bayesianTrainingRepo repository.BayesianTrainingRepository,
) SpamService {
	return &spamService{
		emailRepo:            emailRepo,
		reputationManager:    reputationManager,
		bayesianTrainingRepo: bayesianTrainingRepo,
	}
}

// MarkAsSpam 标记邮件为垃圾邮件
func (s *spamService) MarkAsSpam(ctx context.Context, emailIDs []int64) error {
	for _, emailID := range emailIDs {
		// 获取邮件信息
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil || email == nil {
			log.Printf("警告: 无法找到邮件 %d: %v", emailID, err)
			continue
		}

		// 更新邮件的垃圾邮件状态
		email.IsSpam = true
		email.UserMarkedSpam = true
		now := time.Now()
		email.UserMarkedAt = &now

		if err := s.emailRepo.Update(ctx, email); err != nil {
			log.Printf("警告: 标记邮件 %d 为垃圾邮件失败: %v", emailID, err)
			continue
		}

		// 异步更新发件人信誉（降低）
		if s.reputationManager != nil {
			go s.updateReputationForSpam(context.Background(), email.FromAddress, true)
		}

		// 异步记录贝叶斯训练数据
		if s.bayesianTrainingRepo != nil {
			go s.recordBayesianTraining(context.Background(), email, true)
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
			log.Printf("警告: 无法找到邮件 %d: %v", emailID, err)
			continue
		}

		// 取消垃圾邮件标记
		email.IsSpam = false
		email.UserMarkedSpam = false
		email.UserMarkedAt = nil

		if err := s.emailRepo.Update(ctx, email); err != nil {
			log.Printf("警告: 取消邮件 %d 的垃圾邮件标记失败: %v", emailID, err)
			continue
		}

		// 异步更新发件人信誉（提高）
		if s.reputationManager != nil {
			go s.updateReputationForSpam(context.Background(), email.FromAddress, false)
		}

		// 异步记录贝叶斯训练数据
		if s.bayesianTrainingRepo != nil {
			go s.recordBayesianTraining(context.Background(), email, false)
		}
	}

	return nil
}

// BatchDeleteSpam 批量删除垃圾邮件
func (s *spamService) BatchDeleteSpam(ctx context.Context, emailIDs []int64) (int64, error) {
	deletedCount := int64(0)

	for _, emailID := range emailIDs {
		// 软删除邮件
		deleted := true
		if err := s.emailRepo.UpdateLocalStatus(ctx, emailID, nil, nil, nil, &deleted); err != nil {
			log.Printf("警告: 删除邮件 %d 失败: %v", emailID, err)
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
			log.Printf("警告: 删除邮件 %d 失败: %v", email.ID, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// GetSpamEmails 获取垃圾邮件列表
func (s *spamService) GetSpamEmails(ctx context.Context, accountUID string, page, pageSize int) ([]*model.Email, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建过滤条件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 查询垃圾邮件列表
	emails, total, err := s.emailRepo.List(ctx, filter, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get spam emails: %w", err)
	}

	return emails, total, nil
}

// GetSpamStats 获取垃圾邮件统计
func (s *spamService) GetSpamStats(ctx context.Context, accountUID string) (*SpamStats, error) {
	stats := &SpamStats{}

	// 构建基础过滤条件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 统计总数
	total, err := s.emailRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count spam emails: %w", err)
	}
	stats.TotalCount = total

	// 统计未读数
	isRead := false
	unreadFilter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
		IsRead:    &isRead,
	}
	if accountUID != "" {
		unreadFilter.AccountUID = accountUID
	}
	unreadCount, err := s.emailRepo.Count(ctx, unreadFilter)
	if err == nil {
		stats.UnreadCount = unreadCount
	}

	// 其他统计暂时设为 0，后续实现
	stats.TodayCount = 0
	stats.WeekCount = 0
	stats.MonthCount = 0
	stats.BlockedCount = 0

	return stats, nil
}

// updateReputationForSpam 更新发件人信誉（用户反馈）
func (s *spamService) updateReputationForSpam(ctx context.Context, senderEmail string, isSpam bool) {
	if s.reputationManager == nil {
		return
	}
	if err := s.reputationManager.UpdateReputationByUserFeedback(ctx, senderEmail, isSpam); err != nil {
		log.Printf("警告: 更新发件人信誉失败 [%s]: %v", senderEmail, err)
	}
}

// recordBayesianTraining 记录贝叶斯训练数据
func (s *spamService) recordBayesianTraining(ctx context.Context, email *model.Email, isSpam bool) {
	if s.bayesianTrainingRepo == nil {
		return
	}

	// 简化实现：提取特征词（暂时使用简单的空格分割）
	// 后续可以使用更复杂的分词算法
	tokens := "[]" // 暂时使用空数组

	trainingData := &model.BayesianTraining{
		UserUID: email.AccountUID,
		EmailID: fmt.Sprintf("%d", email.ID),
		IsSpam:  isSpam,
		Tokens:  tokens,
	}

	if err := s.bayesianTrainingRepo.Create(ctx, trainingData); err != nil {
		log.Printf("警告: 记录贝叶斯训练数据失败 [邮件ID: %d]: %v", email.ID, err)
	}
}
