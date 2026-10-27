package repository

import (
	"context"
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
	// 批量按账户统计邮件数量：一次 GROUP BY 聚合返回 map[accountUID]count，消除列表页 N+1 查询
	CountByAccounts(ctx context.Context, accountUIDs []string) (map[string]int64, error)
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
	BatchSoftDeleteByAccountUID(ctx context.Context, accountUID string) (int64, error)
	// Delete 走 GORM 软删除（写 deleted_at），邮件仍留在库中，可从回收站恢复。
	Delete(ctx context.Context, id int64) error
	// ReplaceAttachments 整体替换邮件的附件元数据行（删旧插新，单事务）。
	// 附件元数据随同步落库（StorageType=external，本地不存内容）。
	ReplaceAttachments(ctx context.Context, emailID int64, atts []model.EmailAttachment) error
	// PermanentDelete 物理删除（Unscoped），用于"永久删除"和"清空回收站"。
	PermanentDelete(ctx context.Context, id int64) error
	// PermanentDeleteByIDs 批量物理删除，返回实际删除行数。
	// 一次 DELETE ... WHERE id IN 替代循环内逐封删除（2N → 1 条查询）。
	PermanentDeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	// PermanentDeleteAllTrash 一次物理删除所有 is_deleted = true 的邮件（清空回收站），
	// 返回删除行数。原子单条 SQL，替代原"分页 List + 循环逐封删"。
	PermanentDeleteAllTrash(ctx context.Context) (int64, error)
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

// BatchSoftDeleteByAccountUID 批量软删除指定账户下所有未删除的邮件（设置 is_deleted = true）
func (r *emailRepository) BatchSoftDeleteByAccountUID(ctx context.Context, accountUID string) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.Email{}).
		Where("account_uid = ? AND is_deleted = false", accountUID).
		Update("is_deleted", true)
	return result.RowsAffected, result.Error
}

// Delete 删除邮件（GORM 软删除：写 deleted_at，数据仍在库中）
func (r *emailRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Email{}, id).Error
}

// PermanentDelete 物理删除邮件。
//
// 历史缺陷：service 层的 PermanentDeleteEmail / BatchPermanentDeleteEmails /
// EmptyTrash 都声明"物理删除"，但实际调用的是上面的软删除 Delete，
// 用户以为已彻底删除的邮件其实永久留存（隐私与合规问题）。
func (r *emailRepository) PermanentDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.Email{}, id).Error
}

// ReplaceAttachments 整体替换邮件的附件元数据行
func (r *emailRepository) ReplaceAttachments(ctx context.Context, emailID int64, atts []model.EmailAttachment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("email_id = ?", emailID).Delete(&model.EmailAttachment{}).Error; err != nil {
			return err
		}
		if len(atts) == 0 {
			return nil
		}
		for i := range atts {
			atts[i].EmailID = emailID
		}
		return tx.Create(&atts).Error
	})
}

// PermanentDeleteByIDs 批量物理删除，返回实际删除行数
func (r *emailRepository) PermanentDeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Delete(&model.Email{})
	return res.RowsAffected, res.Error
}

// PermanentDeleteAllTrash 清空回收站：一次物理删除所有 is_deleted = true 的邮件
func (r *emailRepository) PermanentDeleteAllTrash(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Unscoped().Where("is_deleted = ?", true).Delete(&model.Email{})
	return res.RowsAffected, res.Error
}

// DeleteByAccountUID 根据账号 UID 批量删除该账号下的所有邮件（物理删除）
func (r *emailRepository) DeleteByAccountUID(ctx context.Context, accountUID string) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("account_uid = ?", accountUID).
		Delete(&model.Email{}).Error
}
