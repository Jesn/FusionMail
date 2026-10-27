package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"time"
	"unicode"

	"gorm.io/gorm"
)

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
// isASCIIQuery 判断查询串是否只含 ASCII 字符。
//
// 非 ASCII（中文、日文、韩文、西里尔等）查询走 'english' 全文搜索没有意义：
// 整段会被当成单个 token，匹配率接近零。这类查询应直接走 trigram 索引
// 加速的 ILIKE 路径。
func isASCIIQuery(q string) bool {
	for _, r := range q {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

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

	// 优先使用全文搜索（仅对 ASCII 查询有意义）。
	//
	// 'english' 文本搜索配置对中文整段只产生一个 token，中文查询几乎必然匹配不到，
	// 每次都要白跑一次全表 tsvector 计算再降级到 ILIKE。因此对含非 ASCII 字符的
	// 查询直接跳过全文搜索，走 trigram 索引加速的 ILIKE 路径
	// （见 migrations/000002_email_search_trgm.up.sql）。
	if isASCIIQuery(query) {
		fullTextQuery := baseQuery.Session(&gorm.Session{}).
			Where("to_tsvector('english', coalesce(subject,'') || ' ' || coalesce(from_name,'') || ' ' || coalesce(text_body,'')) @@ plainto_tsquery('english', ?)", query)

		if err := fullTextQuery.Count(&total).Error; err != nil {
			return nil, 0, err
		}

		if total > 0 {
			err := fullTextQuery.
				Offset(offset).
				Limit(limit).
				Order("sent_at DESC").
				Find(&emails).Error
			return emails, total, err
		}
	}

	// 降级：ILIKE 模糊匹配（支持部分匹配和中文，由 GIN trigram 索引加速）
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

// CountByAccounts 批量按账户统计邮件数量
// 一次 GROUP BY 聚合查询替代逐账户 COUNT，消除子邮箱列表页的 N+1 查询
// 返回 map[accountUID]count；统计口径与单账户版本一致，
// 无邮件的账户不会出现在返回的 map 中（调用方取默认值 0）
func (r *emailRepository) CountByAccounts(ctx context.Context, accountUIDs []string) (map[string]int64, error) {
	result := make(map[string]int64, len(accountUIDs))
	if len(accountUIDs) == 0 {
		return result, nil
	}

	var rows []struct {
		AccountUID string
		Count      int64
	}
	err := r.db.WithContext(ctx).
		Model(&model.Email{}).
		Select("account_uid AS account_uid, COUNT(*) AS count").
		Where("account_uid IN ?", accountUIDs).
		Group("account_uid").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.AccountUID] = row.Count
	}
	return result, nil
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
