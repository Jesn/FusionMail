package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

// SentEmailFilter 已发送邮件过滤条件
type SentEmailFilter struct {
	AccountUID  string     // 账户 UID
	Status      string     // 发送状态：sent/failed
	StartDate   *time.Time // 开始时间
	EndDate     *time.Time // 结束时间
	SearchQuery string     // 搜索关键词（主题、收件人）
}

// SentEmailRepository 已发送邮件数据仓库接口
// Requirements: 1.4, 7.1, 7.4
type SentEmailRepository interface {
	// Create 创建已发送邮件记录
	Create(ctx context.Context, email *model.SentEmail) error

	// FindByID 根据 ID 查找已发送邮件
	FindByID(ctx context.Context, id int64) (*model.SentEmail, error)

	// FindByMessageID 根据 Message-ID 查找已发送邮件
	FindByMessageID(ctx context.Context, messageID string) (*model.SentEmail, error)

	// Update 更新已发送邮件
	Update(ctx context.Context, email *model.SentEmail) error

	// Delete 删除已发送邮件（软删除）
	Delete(ctx context.Context, id int64) error

	// List 列出已发送邮件
	List(ctx context.Context, filter *SentEmailFilter, offset, limit int) ([]*model.SentEmail, int64, error)

	// Count 统计已发送邮件数量
	Count(ctx context.Context, filter *SentEmailFilter) (int64, error)

	// CountByAccount 统计账户的已发送邮件数量
	CountByAccount(ctx context.Context, accountUID string) (int64, error)

	// DeleteByAccountUID 删除账户的所有已发送邮件
	DeleteByAccountUID(ctx context.Context, accountUID string) error
}

// sentEmailRepository 已发送邮件数据仓库实现
type sentEmailRepository struct {
	db *gorm.DB
}

// NewSentEmailRepository 创建已发送邮件数据仓库实例
func NewSentEmailRepository(db *gorm.DB) SentEmailRepository {
	return &sentEmailRepository{db: db}
}

// Create 创建已发送邮件记录
func (r *sentEmailRepository) Create(ctx context.Context, email *model.SentEmail) error {
	return r.db.WithContext(ctx).Create(email).Error
}

// FindByID 根据 ID 查找已发送邮件
func (r *sentEmailRepository) FindByID(ctx context.Context, id int64) (*model.SentEmail, error) {
	var email model.SentEmail
	err := r.db.WithContext(ctx).First(&email, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &email, nil
}

// FindByMessageID 根据 Message-ID 查找已发送邮件
func (r *sentEmailRepository) FindByMessageID(ctx context.Context, messageID string) (*model.SentEmail, error) {
	var email model.SentEmail
	err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		First(&email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &email, nil
}

// Update 更新已发送邮件
func (r *sentEmailRepository) Update(ctx context.Context, email *model.SentEmail) error {
	return r.db.WithContext(ctx).Save(email).Error
}

// Delete 删除已发送邮件（软删除）
func (r *sentEmailRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.SentEmail{}, id).Error
}

// List 列出已发送邮件
// Requirements: 7.1, 7.4
func (r *sentEmailRepository) List(ctx context.Context, filter *SentEmailFilter, offset, limit int) ([]*model.SentEmail, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.SentEmail{})

	// 应用过滤条件
	query = r.applyFilter(query, filter)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	var emails []*model.SentEmail
	err := query.
		Order("sent_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&emails).Error
	if err != nil {
		return nil, 0, err
	}

	return emails, total, nil
}

// Count 统计已发送邮件数量
func (r *sentEmailRepository) Count(ctx context.Context, filter *SentEmailFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&model.SentEmail{})
	query = r.applyFilter(query, filter)

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// CountByAccount 统计账户的已发送邮件数量
func (r *sentEmailRepository) CountByAccount(ctx context.Context, accountUID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.SentEmail{}).
		Where("account_uid = ?", accountUID).
		Count(&count).Error
	return count, err
}

// DeleteByAccountUID 删除账户的所有已发送邮件
func (r *sentEmailRepository) DeleteByAccountUID(ctx context.Context, accountUID string) error {
	return r.db.WithContext(ctx).
		Where("account_uid = ?", accountUID).
		Delete(&model.SentEmail{}).Error
}

// applyFilter 应用过滤条件
func (r *sentEmailRepository) applyFilter(query *gorm.DB, filter *SentEmailFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	if filter.AccountUID != "" {
		query = query.Where("account_uid = ?", filter.AccountUID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.StartDate != nil {
		query = query.Where("sent_at >= ?", filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("sent_at <= ?", filter.EndDate)
	}

	if filter.SearchQuery != "" {
		searchPattern := "%" + filter.SearchQuery + "%"
		query = query.Where(
			"subject ILIKE ? OR to_addresses ILIKE ? OR from_address ILIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	return query
}
