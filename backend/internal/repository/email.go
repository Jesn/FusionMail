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
	AccountUIDs []string // 多账号筛选（用于分组筛选）
	GroupID     *int64   // 分组 ID：nil 表示不过滤，0 表示未分组，>0 表示具体分组
	IsRead      *bool
	IsStarred   *bool
	IsArchived  *bool
	IsDeleted   *bool
	IsSpam      *bool // 垃圾邮件过滤
	FromAddress string
	Subject     string
	StartDate   string
	EndDate     string
	SearchQuery string
}

// EmailRepository 邮件数据仓库接口
// EmailReader 邮件只读查询接口
type EmailReader interface {
	FindByID(ctx context.Context, id int64) (*model.Email, error)
	FindByIDs(ctx context.Context, ids []int64) ([]*model.Email, error)
	FindByProviderID(ctx context.Context, providerID, accountUID string) (*model.Email, error)
	FindByDedupeKey(ctx context.Context, accountUID, dedupeKey string) (*model.Email, error)
	List(ctx context.Context, filter *EmailFilter, offset, limit int) ([]*model.Email, int64, error)
	Search(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error)
	Count(ctx context.Context, filter *EmailFilter) (int64, error)
	CountByDateRange(ctx context.Context, startTime, endTime time.Time) (int64, error)
	CountByAccount(ctx context.Context, accountUID string) (int64, error)
	GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error)
	GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error)
}

// EmailWriter 邮件写入接口
type EmailWriter interface {
	Create(ctx context.Context, email *model.Email) error
	CreateBatch(ctx context.Context, emails []*model.Email) error
	Update(ctx context.Context, email *model.Email) error
	UpdateLocalStatus(ctx context.Context, id int64, isRead, isStarred, isArchived, isDeleted *bool) error
	BatchUpdateLocalDeleted(ctx context.Context, ids []int64, deleted bool) (int64, error)
	Delete(ctx context.Context, id int64) error
	DeleteByAccountUID(ctx context.Context, accountUID string) error
}

// EmailStatusRepository 邮件状态管理接口
type EmailStatusRepository interface {
	CountUnread(ctx context.Context, accountUID string) (int64, error)
	MarkAsRead(ctx context.Context, ids []int64) error
	MarkAsUnread(ctx context.Context, ids []int64) error
	MarkAllAsRead(ctx context.Context, accountUID *string) (int64, error)
	SoftDeleteByAccountUID(ctx context.Context, accountUID string) error
	RestoreByAccountUID(ctx context.Context, accountUID string) error
}

// EmailRepository 邮件数据仓库接口（组合接口，供需要全量方法的消费方使用）
type EmailRepository interface {
	EmailReader
	EmailWriter
	EmailStatusRepository
}

// GlobalEmailStats 全局邮件统计
type GlobalEmailStats struct {
	TotalCount    int64 `json:"total_count"`
	UnreadCount   int64 `json:"unread_count"`
	StarredCount  int64 `json:"starred_count"`
	ArchivedCount int64 `json:"archived_count"`
	DeletedCount  int64 `json:"deleted_count"`
	SpamCount     int64 `json:"spam_count"`
}

// AccountEmailStats 账户邮件统计
type AccountEmailStats struct {
	TotalCount    int64 `json:"total_count"`
	UnreadCount   int64 `json:"unread_count"`
	StarredCount  int64 `json:"starred_count"`
	ArchivedCount int64 `json:"archived_count"`
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

// FindByIDs 根据 ID 列表批量查找邮件
func (r *emailRepository) FindByIDs(ctx context.Context, ids []int64) ([]*model.Email, error) {
	if len(ids) == 0 {
		return []*model.Email{}, nil
	}

	var emails []*model.Email
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&emails).Error
	return emails, err
}

// FindByProviderID 根据 Provider ID 和 Account UID 查找邮件
// 使用 Unscoped() 包含软删除的记录，防止已删除邮件被重新同步创建
func (r *emailRepository) FindByProviderID(ctx context.Context, providerID, accountUID string) (*model.Email, error) {
	var email model.Email
	err := r.db.WithContext(ctx).
		Unscoped(). // 包含软删除的记录，避免重复同步
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

// FindByDedupeKey 根据去重标识查找邮件
// 使用 Unscoped() 包含软删除的记录，防止已删除邮件被重新同步创建
func (r *emailRepository) FindByDedupeKey(ctx context.Context, accountUID, dedupeKey string) (*model.Email, error) {
	if dedupeKey == "" {
		return nil, nil
	}
	var email model.Email
	err := r.db.WithContext(ctx).
		Unscoped(). // 包含软删除的记录，避免重复同步
		Where("account_uid = ? AND dedupe_key = ?", accountUID, dedupeKey).
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

// BatchUpdateLocalDeleted 批量更新本地删除状态
func (r *emailRepository) BatchUpdateLocalDeleted(ctx context.Context, ids []int64, deleted bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).
		Model(&model.Email{}).
		Where("id IN ?", ids).
		Update("is_deleted", deleted)

	return result.RowsAffected, result.Error
}

// Delete 删除邮件
func (r *emailRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Email{}, id).Error
}

// DeleteByAccountUID 根据账号 UID 批量删除该账号下的所有邮件（物理删除）
func (r *emailRepository) DeleteByAccountUID(ctx context.Context, accountUID string) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("account_uid = ?", accountUID).
		Delete(&model.Email{}).Error
}

// SoftDeleteByAccountUID 软删除指定账户的所有邮件
func (r *emailRepository) SoftDeleteByAccountUID(ctx context.Context, accountUID string) error {
	// 使用 GORM 的软删除，只需要调用 Delete 方法
	return r.db.WithContext(ctx).Where("account_uid = ?", accountUID).Delete(&model.Email{}).Error
}

// RestoreByAccountUID 恢复指定账户的所有邮件
func (r *emailRepository) RestoreByAccountUID(ctx context.Context, accountUID string) error {
	// 恢复软删除的邮件，将 deleted_at 设置为 NULL
	return r.db.WithContext(ctx).Unscoped().Model(&model.Email{}).
		Where("account_uid = ? AND deleted_at IS NOT NULL", accountUID).
		Update("deleted_at", nil).Error
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
// 优化：优先使用 PostgreSQL 全文搜索索引，性能提升 10-100 倍
// 降级：如果全文搜索无结果，回退到 ILIKE 模糊匹配
func (r *emailRepository) Search(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error) {
	var emails []*model.Email
	var total int64

	// 构建基础查询条件
	baseQuery := r.db.WithContext(ctx).Model(&model.Email{}).
		Where("is_deleted = ?", false)

	if accountUID != "" {
		baseQuery = baseQuery.Where("account_uid = ?", accountUID)
	}

	// 优先使用全文搜索（性能更好）
	// 使用 plainto_tsquery 自动处理查询词
	fullTextQuery := baseQuery.Session(&gorm.Session{}).
		Where("to_tsvector('english', coalesce(subject,'') || ' ' || coalesce(from_name,'') || ' ' || coalesce(text_body,'')) @@ plainto_tsquery('english', ?)", query)

	// 尝试全文搜索
	if err := fullTextQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 如果全文搜索有结果，使用全文搜索
	if total > 0 {
		err := fullTextQuery.
			Offset(offset).
			Limit(limit).
			Order("sent_at DESC").
			Find(&emails).Error
		return emails, total, err
	}

	// 降级：全文搜索无结果时，使用 ILIKE 模糊匹配（支持部分匹配和中文）
	likeQuery := baseQuery.Session(&gorm.Session{}).
		Where("(subject ILIKE ? OR from_name ILIKE ? OR from_address ILIKE ? OR text_body ILIKE ?)",
			"%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")

	if err := likeQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := likeQuery.
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

// MarkAllAsRead 批量标记所有未读邮件为已读
func (r *emailRepository) MarkAllAsRead(ctx context.Context, accountUID *string) (int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Email{}).
		Where("is_read = ?", false).
		Where("is_deleted = ?", false)

	// 如果指定了账号，添加账号过滤
	if accountUID != nil && *accountUID != "" {
		query = query.Where("account_uid = ?", *accountUID)
	}

	result := query.Update("is_read", true)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// applyFilter 应用过滤条件
func (r *emailRepository) applyFilter(query *gorm.DB, filter *EmailFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	// 账号筛选：优先使用 AccountUIDs（多账号），其次使用 AccountUID（单账号）
	if len(filter.AccountUIDs) > 0 {
		query = query.Where("account_uid IN ?", filter.AccountUIDs)
	} else if filter.AccountUID != "" {
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

	if filter.IsSpam != nil {
		query = query.Where("is_spam = ?", *filter.IsSpam)
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

// GetGlobalStats 获取全局邮件统计（单条 SQL 聚合查询，性能优化）
func (r *emailRepository) GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error) {
	var stats GlobalEmailStats

	// 使用 PostgreSQL FILTER 子句进行单次聚合查询
	// 相比多次 COUNT 查询，减少 5 次数据库往返
	sql := `
		SELECT 
			COUNT(*) FILTER (WHERE is_deleted = false AND deleted_at IS NULL) as total_count,
			COUNT(*) FILTER (WHERE is_read = false AND is_deleted = false AND deleted_at IS NULL) as unread_count,
			COUNT(*) FILTER (WHERE is_starred = true AND is_deleted = false AND deleted_at IS NULL) as starred_count,
			COUNT(*) FILTER (WHERE is_archived = true AND is_deleted = false AND deleted_at IS NULL) as archived_count,
			COUNT(*) FILTER (WHERE is_deleted = true OR deleted_at IS NOT NULL) as deleted_count,
			COUNT(*) FILTER (WHERE is_spam = true AND is_deleted = false AND deleted_at IS NULL) as spam_count
		FROM emails
	`

	err := r.db.WithContext(ctx).Raw(sql).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetAccountStats 获取账户邮件统计（单条 SQL 聚合查询，性能优化）
func (r *emailRepository) GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error) {
	var stats AccountEmailStats

	// 使用 PostgreSQL FILTER 子句进行单次聚合查询
	sql := `
		SELECT 
			COUNT(*) FILTER (WHERE is_deleted = false AND deleted_at IS NULL) as total_count,
			COUNT(*) FILTER (WHERE is_read = false AND is_deleted = false AND deleted_at IS NULL) as unread_count,
			COUNT(*) FILTER (WHERE is_starred = true AND is_deleted = false AND deleted_at IS NULL) as starred_count,
			COUNT(*) FILTER (WHERE is_archived = true AND is_deleted = false AND deleted_at IS NULL) as archived_count
		FROM emails
		WHERE account_uid = ?
	`

	err := r.db.WithContext(ctx).Raw(sql, accountUID).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
