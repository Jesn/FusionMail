package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"time"

	"github.com/redis/go-redis/v9"
)

// ReputationManager 发件人信誉管理器
type ReputationManager struct {
	reputationRepo repository.SenderReputationRepository
	redisClient    *redis.Client
	cacheTTL       time.Duration
}

// NewReputationManager 创建发件人信誉管理器
func NewReputationManager(
	reputationRepo repository.SenderReputationRepository,
	redisClient *redis.Client,
) *ReputationManager {
	return &ReputationManager{
		reputationRepo: reputationRepo,
		redisClient:    redisClient,
		cacheTTL:       1 * time.Hour, // 缓存 1 小时
	}
}

// GetOrCreateReputation 获取或创建发件人信誉
func (r *ReputationManager) GetOrCreateReputation(ctx context.Context, senderEmail string) (*model.SenderReputation, error) {
	// 1. 从数据库查询
	reputation, err := r.reputationRepo.FindByEmail(ctx, senderEmail)
	if err == nil && reputation != nil {
		// 缓存结果
		r.cacheReputation(ctx, reputation)
		return reputation, nil
	}

	// 2. 如果不存在，创建新的信誉记录
	domain := extractDomain(senderEmail)
	reputation = &model.SenderReputation{
		Email:           senderEmail,
		Domain:          domain,
		ReputationScore: 50, // 初始评分 50
		TrustLevel:      "neutral",
		TotalEmails:     0,
		SpamCount:       0,
		HamCount:        0,
		RBLStatus:       "unknown",
	}

	if err := r.reputationRepo.Create(ctx, reputation); err != nil {
		return nil, fmt.Errorf("failed to create reputation: %w", err)
	}

	// 缓存新创建的信誉
	r.cacheReputation(ctx, reputation)

	return reputation, nil
}

// UpdateReputationByUserFeedback 根据用户反馈更新信誉
func (r *ReputationManager) UpdateReputationByUserFeedback(ctx context.Context, senderEmail string, isSpam bool) error {
	reputation, err := r.GetOrCreateReputation(ctx, senderEmail)
	if err != nil {
		return err
	}

	// 更新评分
	if isSpam {
		// 用户标记为垃圾邮件，降低信誉评分 10 分
		reputation.ReputationScore -= 10
		reputation.SpamCount++
	} else {
		// 用户标记为正常邮件，提高信誉评分 5 分
		reputation.ReputationScore += 5
		reputation.HamCount++
	}

	// 确保评分在 0-100 范围内
	if reputation.ReputationScore < 0 {
		reputation.ReputationScore = 0
	}
	if reputation.ReputationScore > 100 {
		reputation.ReputationScore = 100
	}

	reputation.TotalEmails++

	// 更新信任级别
	reputation.TrustLevel = r.calculateTrustLevel(reputation.ReputationScore)

	// 更新数据库
	if err := r.reputationRepo.Update(ctx, reputation); err != nil {
		return fmt.Errorf("failed to update reputation: %w", err)
	}

	// 使缓存失效
	r.invalidateCache(ctx, senderEmail)

	return nil
}

// UpdateReputationByDetection 根据检测结果更新信誉
func (r *ReputationManager) UpdateReputationByDetection(ctx context.Context, senderEmail string, spamScore int) error {
	reputation, err := r.GetOrCreateReputation(ctx, senderEmail)
	if err != nil {
		return err
	}

	// 根据垃圾邮件评分调整信誉
	// 评分越高，信誉下降越多
	if spamScore >= 60 {
		// 确认为垃圾邮件，降低信誉 5 分
		reputation.ReputationScore -= 5
		reputation.SpamCount++
	} else if spamScore >= 40 {
		// 可疑邮件，降低信誉 2 分
		reputation.ReputationScore -= 2
	} else {
		// 正常邮件，提高信誉 1 分
		reputation.ReputationScore += 1
		reputation.HamCount++
	}

	// 确保评分在 0-100 范围内
	if reputation.ReputationScore < 0 {
		reputation.ReputationScore = 0
	}
	if reputation.ReputationScore > 100 {
		reputation.ReputationScore = 100
	}

	reputation.TotalEmails++

	// 更新信任级别
	reputation.TrustLevel = r.calculateTrustLevel(reputation.ReputationScore)

	// 更新数据库
	if err := r.reputationRepo.Update(ctx, reputation); err != nil {
		return fmt.Errorf("failed to update reputation: %w", err)
	}

	// 使缓存失效
	r.invalidateCache(ctx, senderEmail)

	return nil
}

// GetReputationScore 获取发件人信誉评分
func (r *ReputationManager) GetReputationScore(ctx context.Context, senderEmail string) (float64, error) {
	reputation, err := r.GetOrCreateReputation(ctx, senderEmail)
	if err != nil {
		return 50, err // 默认返回 50 分
	}

	return reputation.ReputationScore, nil
}

// IsLowReputation 判断是否为低信誉发件人
func (r *ReputationManager) IsLowReputation(ctx context.Context, senderEmail string) (bool, error) {
	score, err := r.GetReputationScore(ctx, senderEmail)
	if err != nil {
		return false, err
	}

	// 信誉评分低于 20 视为低信誉
	return score < 20, nil
}

// IsHighReputation 判断是否为高信誉发件人
func (r *ReputationManager) IsHighReputation(ctx context.Context, senderEmail string) (bool, error) {
	score, err := r.GetReputationScore(ctx, senderEmail)
	if err != nil {
		return false, err
	}

	// 信誉评分高于 80 视为高信誉
	return score > 80, nil
}

// AdjustScoreByReputation 根据信誉调整垃圾邮件评分
func (r *ReputationManager) AdjustScoreByReputation(ctx context.Context, senderEmail string, spamScore int) (int, error) {
	reputation, err := r.GetOrCreateReputation(ctx, senderEmail)
	if err != nil {
		return spamScore, nil // 出错时不调整评分
	}

	// 高信誉发件人：降低垃圾评分权重
	if reputation.ReputationScore > 80 {
		// 降低 20% 的评分
		adjustment := int(float64(spamScore) * 0.2)
		spamScore -= adjustment
	}

	// 低信誉发件人：增加垃圾评分权重
	if reputation.ReputationScore < 20 {
		// 增加 20% 的评分
		adjustment := int(float64(spamScore) * 0.2)
		spamScore += adjustment
	}

	// 确保评分在 0-100 范围内
	if spamScore < 0 {
		spamScore = 0
	}
	if spamScore > 100 {
		spamScore = 100
	}

	return spamScore, nil
}

// GetReputationStats 获取发件人信誉统计
func (r *ReputationManager) GetReputationStats(ctx context.Context, senderEmail string) (*ReputationStats, error) {
	reputation, err := r.GetOrCreateReputation(ctx, senderEmail)
	if err != nil {
		return nil, err
	}

	stats := &ReputationStats{
		Email:        reputation.Email,
		Domain:       reputation.Domain,
		Score:        reputation.ReputationScore,
		TrustLevel:   reputation.TrustLevel,
		TotalEmails:  reputation.TotalEmails,
		SpamCount:    reputation.SpamCount,
		HamCount:     reputation.HamCount,
		SpamRate:     0,
		RBLStatus:    reputation.RBLStatus,
		RBLCheckedAt: reputation.RBLCheckedAt,
	}

	// 计算垃圾邮件率
	if reputation.TotalEmails > 0 {
		stats.SpamRate = float64(reputation.SpamCount) / float64(reputation.TotalEmails) * 100
	}

	return stats, nil
}

// ReputationStats 发件人信誉统计
type ReputationStats struct {
	Email        string     `json:"email"`
	Domain       string     `json:"domain"`
	Score        float64    `json:"score"`
	TrustLevel   string     `json:"trust_level"`
	TotalEmails  int64      `json:"total_emails"`
	SpamCount    int64      `json:"spam_count"`
	HamCount     int64      `json:"ham_count"`
	SpamRate     float64    `json:"spam_rate"`
	RBLStatus    string     `json:"rbl_status"`
	RBLCheckedAt *time.Time `json:"rbl_checked_at"`
}

// calculateTrustLevel 计算信任级别
func (r *ReputationManager) calculateTrustLevel(score float64) string {
	if score >= 80 {
		return "trusted"
	} else if score >= 40 {
		return "neutral"
	} else if score >= 20 {
		return "suspicious"
	}
	return "blocked"
}

// cacheReputation 缓存信誉数据
func (r *ReputationManager) cacheReputation(ctx context.Context, reputation *model.SenderReputation) {
	cacheKey := fmt.Sprintf("reputation:%s", reputation.Email)
	// 简化处理：只缓存评分
	r.redisClient.Set(ctx, cacheKey, reputation.ReputationScore, r.cacheTTL)
}

// invalidateCache 使缓存失效
func (r *ReputationManager) invalidateCache(ctx context.Context, senderEmail string) {
	cacheKey := fmt.Sprintf("reputation:%s", senderEmail)
	r.redisClient.Del(ctx, cacheKey)
}

// BatchUpdateReputation 批量更新信誉（用于定期任务）
func (r *ReputationManager) BatchUpdateReputation(ctx context.Context) error {
	// 这里可以实现定期的信誉衰减逻辑
	// 例如：长时间未发邮件的发件人，信誉逐渐回归到 50 分
	// 暂时留空，后续实现
	return nil
}
