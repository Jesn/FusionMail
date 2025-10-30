package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

// EmailFilter 邮件过滤条件
type EmailFilter struct {
	AccountUID  string
	IsRead      *bool
	IsStarred   *bool
	IsArchived  *bool
	IsDeleted   *bool
	FromAddress string
	Subject     string
	StartDate   string
	EndDate     string
	SearchQuery string
}

// EmailRepository 邮件数据仓库接口
type EmailRepository interface {
	Create(ctx context.Context, email *model.Email) error
	CreateBatch(ctx context.Context, emails []*model.Email) error
	FindByID(ctx context.Context, id int64) (*model.Email, error)
	FindByProviderID(ctx context.Context, providerID, accountUID string) (*model.Email, error)
	Update(ctx context.Context, email *model.Email) error
	UpdateLocalStatus(ctx context.Context, id int64, isRead, isStarred, isArchived, isDeleted *bool) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter *EmailFilter, offset, limit int) ([]*model.Email, int64, error)
	Search(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error)
	CountUnread(ctx context.Context, accountUID string) (int64, error)
	MarkAsRead(ctx context.Context, ids []int64) error
	MarkAsUnread(ctx context.Context, ids []int64) error

	// 系统管理需要的方法
	Count(ctx context.Context, filter *EmailFilter) (int64, error)
	CountByDateRange(ctx context.Context, startTime, endTime time.Time) (int64, error)
	CountByAccount(ctx context.Context, accountUID string) (int64, error)
}

// emailRepository 邮件数据仓库实现
type emailRepository struct {
	db *gorm.DB
}

// NewEmailRepository 创建邮件数据仓库实例
func NewEmailRepository(db *gorm.DB) EmailRepository {
	return &emailRepository{db: db}
}

// Create 创建邮件
func (r *emailRepository) Create(ctx context.Context, email *model.Email) error {
	return r.db.WithContext(ctx).Create(email).Error
}

// CreateBatch 批量创建邮件
func (r *emailRepository) CreateBatch(ctx context.Context, emails []*model.Email) error {
	if len(emails) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(emails, 100).Error
}

// FindByID 根据 ID 查找邮件
func (r *emailRepository) FindByID(ctx context.Context, id int64) (*model.Email, error) {
	var email model.Email
	err := r.db.WithContext(ctx).
		Preload("Attachments").
		First(&email, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &email, nil
}

// FindByProviderID 根据 Provider ID 和 Account UID 查找邮件
func (r *emailRepository) FindByProviderID(ctx context.Context, providerID, accountUID string) (*model.Email, error) {
	var email model.Email
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND account_uid = ?", providerID, accountUID).
		First(&email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &email, nil
}

// Update 更新邮件
func (r *emailRepository) Update(ctx context.Context, email *model.Email) error {
	return r.db.WithContext(ctx).Save(email).Error
}

// UpdateLocalStatus 更新本地状态
func (r *emailRepository) UpdateLocalStatus(ctx context.Context, id int64, isRead, isStarred, isArchived, isDeleted *bool) error {
	updates := make(map[string]interface{})

	if isRead != nil {
		updates["is_read"] = *isRead
	}
	if isStarred != nil {
		updates["is_starred"] = *isStarred
	}
	if isArchived != nil {
		updates["is_archived"] = *isArchived
	}
	if isDeleted != nil {
		updates["is_deleted"] = *isDeleted
	}

	if len(updates) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Model(&model.Email{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Delete 删除邮件
func (r *emailRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Email{}, id).Error
}

// List 获取邮件列表
func (r *emailRepository) List(ctx context.Context, filter *EmailFilter, offset, limit int) ([]*model.Email, int64, error) {
	var emails []*model.Email
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Email{})

	// 应用过滤条件
	query = r.applyFilter(query, filter)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := query.
		Offset(offset).
		Limit(limit).
		Order("sent_at DESC").
		Find(&emails).Error

	return emails, total, err
}

// Search 全文搜索邮件
func (r *emailRepository) Search(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error) {
	var emails []*model.Email
	var total int64

	// 使用 PostgreSQL 全文搜索
	searchQuery := r.db.WithContext(ctx).Model(&model.Email{}).
		Where("to_tsvector('english', coalesce(subject, '') || ' ' || coalesce(from_name, '') || ' ' || coalesce(text_body, '')) @@ plainto_tsquery('english', ?)", query)

	if accountUID != "" {
		searchQuery = searchQuery.Where("account_uid = ?", accountUID)
	}

	// 获取总数
	if err := searchQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := searchQuery.
		Offset(offset).
		Limit(limit).
		Order("sent_at DESC").
		Find(&emails).Error

	return emails, total, err
}

// CountUnread 统计未读邮件数
func (r *emailRepository) CountUnread(ctx context.Context, accountUID string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.Email{}).
		Where("is_read = ?", false).
		Where("is_deleted = ?", false)

	if accountUID != "" {
		query = query.Where("account_uid = ?", accountUID)
	}

	err := query.Count(&count).Error
	return count, err
}

// MarkAsRead 标记为已读
func (r *emailRepository) MarkAsRead(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.Email{}).
		Where("id IN ?", ids).
		Update("is_read", true).Error
}

// MarkAsUnread 标记为未读
func (r *emailRepository) MarkAsUnread(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.Email{}).
		Where("id IN ?", ids).
		Update("is_read", false).Error
}

// applyFilter 应用过滤条件
func (r *emailRepository) applyFilter(query *gorm.DB, filter *EmailFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	if filter.AccountUID != "" {
		query = query.Where("account_uid = ?", filter.AccountUID)
	}

	if filter.IsRead != nil {
		query = query.Where("is_read = ?", *filter.IsRead)
	}

	if filter.IsStarred != nil {
		query = query.Where("is_starred = ?", *filter.IsStarred)
	}

	if filter.IsArchived != nil {
		query = query.Where("is_archived = ?", *filter.IsArchived)
	}

	if filter.IsDeleted != nil {
		query = query.Where("is_deleted = ?", *filter.IsDeleted)
	}

	if filter.FromAddress != "" {
		query = query.Where("from_address LIKE ?", "%"+filter.FromAddress+"%")
	}

	if filter.Subject != "" {
		query = query.Where("subject LIKE ?", "%"+filter.Subject+"%")
	}

	if filter.StartDate != "" {
		query = query.Where("sent_at >= ?", filter.StartDate)
	}

	if filter.EndDate != "" {
		query = query.Where("sent_at <= ?", filter.EndDate)
	}

	return query
}

// Count 统计邮件数量
func (r *emailRepository) Count(ctx context.Context, filter *EmailFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.Email{})
	query = r.applyFilter(query, filter)
	err := query.Count(&count).Error
	return count, err
}

// CountByDateRange 按日期范围统计邮件数量
func (r *emailRepository) CountByDateRange(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Email{}).
		Where("sent_at >= ? AND sent_at <= ?", startTime, endTime).
		Count(&count).Error
	return count, err
}

// CountByAccount 按账户统计邮件数量
func (r *emailRepository) CountByAccount(ctx context.Context, accountUID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Email{}).
		Where("account_uid = ?", accountUID).
		Count(&count).Error
	return count, err
}
