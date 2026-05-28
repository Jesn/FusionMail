package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

// SenderReputationRepository 发件人信誉数据仓库接口
type SenderReputationRepository interface {
	Create(ctx context.Context, reputation *model.SenderReputation) error
	FindByID(ctx context.Context, id int64) (*model.SenderReputation, error)
	FindByEmail(ctx context.Context, email string) (*model.SenderReputation, error)
	FindByDomain(ctx context.Context, domain string) ([]*model.SenderReputation, error)
	Update(ctx context.Context, reputation *model.SenderReputation) error
	UpdateScore(ctx context.Context, email string, delta int) error
	UpdateRBLStatus(ctx context.Context, email string, status string, lists string) error
	List(ctx context.Context, offset, limit int) ([]*model.SenderReputation, int64, error)
	ListByTrustLevel(ctx context.Context, trustLevel string, offset, limit int) ([]*model.SenderReputation, int64, error)
	IncrementSpamCount(ctx context.Context, email string) error
	IncrementHamCount(ctx context.Context, email string) error
	GetOrCreate(ctx context.Context, email string, domain string) (*model.SenderReputation, error)
}

// senderReputationRepository 发件人信誉数据仓库实现
type senderReputationRepository struct {
	db *gorm.DB
}

// NewSenderReputationRepository 创建发件人信誉数据仓库实例
func NewSenderReputationRepository(db *gorm.DB) SenderReputationRepository {
	return &senderReputationRepository{db: db}
}

// Create 创建发件人信誉记录
func (r *senderReputationRepository) Create(ctx context.Context, reputation *model.SenderReputation) error {
	return r.db.WithContext(ctx).Create(reputation).Error
}

// FindByID 根据 ID 查找信誉记录
func (r *senderReputationRepository) FindByID(ctx context.Context, id int64) (*model.SenderReputation, error) {
	var reputation model.SenderReputation
	err := r.db.WithContext(ctx).First(&reputation, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reputation, nil
}

// FindByEmail 根据邮箱地址查找信誉记录
func (r *senderReputationRepository) FindByEmail(ctx context.Context, email string) (*model.SenderReputation, error) {
	var reputation model.SenderReputation
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&reputation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reputation, nil
}

// FindByDomain 根据域名查找信誉记录列表
func (r *senderReputationRepository) FindByDomain(ctx context.Context, domain string) ([]*model.SenderReputation, error) {
	var reputations []*model.SenderReputation
	err := r.db.WithContext(ctx).
		Where("domain = ?", domain).
		Order("reputation_score DESC").
		Find(&reputations).Error
	return reputations, err
}

// Update 更新信誉记录
func (r *senderReputationRepository) Update(ctx context.Context, reputation *model.SenderReputation) error {
	return r.db.WithContext(ctx).Save(reputation).Error
}

// UpdateScore 更新信誉评分（增量更新）
func (r *senderReputationRepository) UpdateScore(ctx context.Context, email string, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.SenderReputation{}).
		Where("email = ?", email).
		Updates(map[string]interface{}{
			"reputation_score": gorm.Expr(`
				CASE
					WHEN reputation_score + ? < 0 THEN 0
					WHEN reputation_score + ? > 100 THEN 100
					ELSE reputation_score + ?
				END
			`, delta, delta, delta),
			"updated_at": time.Now(),
		}).Error
}

// UpdateRBLStatus 更新 RBL 状态
func (r *senderReputationRepository) UpdateRBLStatus(ctx context.Context, email string, status string, lists string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.SenderReputation{}).
		Where("email = ?", email).
		Updates(map[string]interface{}{
			"rbl_status":     status,
			"rbl_lists":      lists,
			"rbl_checked_at": now,
			"updated_at":     now,
		}).Error
}

// List 获取信誉记录列表
func (r *senderReputationRepository) List(ctx context.Context, offset, limit int) ([]*model.SenderReputation, int64, error) {
	var reputations []*model.SenderReputation
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.SenderReputation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("reputation_score DESC").
		Find(&reputations).Error

	return reputations, total, err
}

// ListByTrustLevel 根据信任级别获取信誉记录列表
func (r *senderReputationRepository) ListByTrustLevel(ctx context.Context, trustLevel string, offset, limit int) ([]*model.SenderReputation, int64, error) {
	var reputations []*model.SenderReputation
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SenderReputation{}).
		Where("trust_level = ?", trustLevel)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("reputation_score DESC").
		Find(&reputations).Error

	return reputations, total, err
}

// IncrementSpamCount 增加垃圾邮件计数
func (r *senderReputationRepository) IncrementSpamCount(ctx context.Context, email string) error {
	return r.db.WithContext(ctx).
		Model(&model.SenderReputation{}).
		Where("email = ?", email).
		Updates(map[string]interface{}{
			"spam_count":   gorm.Expr("spam_count + 1"),
			"total_emails": gorm.Expr("total_emails + 1"),
			"updated_at":   time.Now(),
		}).Error
}

// IncrementHamCount 增加正常邮件计数
func (r *senderReputationRepository) IncrementHamCount(ctx context.Context, email string) error {
	return r.db.WithContext(ctx).
		Model(&model.SenderReputation{}).
		Where("email = ?", email).
		Updates(map[string]interface{}{
			"ham_count":    gorm.Expr("ham_count + 1"),
			"total_emails": gorm.Expr("total_emails + 1"),
			"updated_at":   time.Now(),
		}).Error
}

// GetOrCreate 获取或创建发件人信誉记录
func (r *senderReputationRepository) GetOrCreate(ctx context.Context, email string, domain string) (*model.SenderReputation, error) {
	// 先尝试查找
	reputation, err := r.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// 如果找到了，直接返回
	if reputation != nil {
		return reputation, nil
	}

	// 如果没找到，创建新记录
	reputation = &model.SenderReputation{
		Email:           email,
		Domain:          domain,
		ReputationScore: 50.0, // 初始评分 50
		TrustLevel:      "neutral",
		TotalEmails:     0,
		SpamCount:       0,
		HamCount:        0,
		RBLStatus:       "unknown",
	}

	if err := r.Create(ctx, reputation); err != nil {
		return nil, err
	}

	return reputation, nil
}
