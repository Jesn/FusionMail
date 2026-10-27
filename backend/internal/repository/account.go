package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"
	"time"

	"gorm.io/gorm"
)

// 模块日志记录器
var accountRepoLog = logger.NewWithModule("AccountRepo")

// AccountReader 账户只读查询接口
type AccountReader interface {
	FindByID(ctx context.Context, id int64) (*model.EmailAccount, error)
	FindByUID(ctx context.Context, uid string) (*model.EmailAccount, error)
	FindByEmail(ctx context.Context, email string) (*model.EmailAccount, error)
	List(ctx context.Context, offset, limit int) ([]*model.EmailAccount, int64, error)
	ListWithFilter(ctx context.Context, filter *AccountListFilter) ([]*model.EmailAccount, int64, error)
	FindAll(ctx context.Context) ([]*model.EmailAccount, error)
	Count(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)
	FindByUIDWithRelations(ctx context.Context, uid string) (*model.EmailAccount, error)
	FindByIDWithRelations(ctx context.Context, id int64) (*model.EmailAccount, error)
	ListWithRelations(ctx context.Context, offset, limit int) ([]*model.EmailAccount, int64, error)
	FindByProviderID(ctx context.Context, providerID int64) ([]*model.EmailAccount, error)
	FindByAdapterID(ctx context.Context, adapterID int64) ([]*model.EmailAccount, error)
	FindByProviderIDs(ctx context.Context, providerIDs []int64, page, pageSize int) ([]*model.EmailAccount, int64, error)
	FindByParentAccountUID(ctx context.Context, parentUID string) ([]*model.EmailAccount, error)
	// FindChildrenByParent 分页查询父账户下的子邮箱，支持 include 状态与邮箱关键词
	FindChildrenByParent(ctx context.Context, filter *ChildAccountListFilter) ([]*model.EmailAccount, int64, error)
	FindByDomain(ctx context.Context, domain string) ([]*model.EmailAccount, error)
}

// AccountWriter 账户写入接口
type AccountWriter interface {
	Create(ctx context.Context, account *model.EmailAccount) error
	Update(ctx context.Context, account *model.EmailAccount) error
	Delete(ctx context.Context, id int64) error
	Restore(ctx context.Context, uid string) error
	ForceDelete(ctx context.Context, uid string) error
}

// AccountSyncRepository 同步相关接口
type AccountSyncRepository interface {
	ListSyncEnabled(ctx context.Context) ([]*model.EmailAccount, error)
	ListSyncEnabledWithRelations(ctx context.Context) ([]*model.EmailAccount, error)
	HealWebhookChildPollingFlags(ctx context.Context) (int64, error)
	UpdateSyncStatus(ctx context.Context, uid string, status string, errorMsg string) error
	IncrementEmailCount(ctx context.Context, uid string, count int) error
	UpdateUnreadCount(ctx context.Context, uid string, count int) error
	IncrementConsecutiveFailures(ctx context.Context, uid string) (int, error)
	ResetConsecutiveFailures(ctx context.Context, uid string) error
	AutoDisableAccount(ctx context.Context, uid string, reason string) error
	MarkRemoteMailboxDeleted(ctx context.Context, uid string) error
	ReactivateFromRemoteOrphan(ctx context.Context, uid string) error
	AutoSoftDeleteAccount(ctx context.Context, uid string, reason string) error
	UpdateSyncProgress(ctx context.Context, uid string, cursor string, progressJSON string) error
	UpdateUIDSyncState(ctx context.Context, uid string, uidValidity, lastUID int64) error
}

// AccountGroupRepository 分组相关接口
type AccountGroupRepository interface {
	FindByGroupID(ctx context.Context, groupID int64) ([]*model.EmailAccount, error)
	FindAllByGroupID(ctx context.Context, groupID int64) ([]*model.EmailAccount, error)
	FindUngrouped(ctx context.Context) ([]*model.EmailAccount, error)
	UpdateGroupID(ctx context.Context, uid string, groupID *int64) error
	BatchUpdateGroupID(ctx context.Context, uids []string, groupID *int64) error
}

// AccountTrashRepository 软删除/回收站相关接口
type AccountTrashRepository interface {
	FindAllWithDeleted(ctx context.Context) ([]*model.EmailAccount, error)
	FindDeleted(ctx context.Context) ([]*model.EmailAccount, error)
	FindDeletedByEmail(ctx context.Context, email string) ([]*model.EmailAccount, error)
	FindDeletedBefore(ctx context.Context, cutoffTime time.Time) ([]*model.EmailAccount, error)
	FindByUIDIncludingDeleted(ctx context.Context, uid string) (*model.EmailAccount, error)
}

// AccountRepository 邮箱账户数据仓库接口（组合接口，供需要全量方法的消费方使用）
type AccountRepository interface {
	AccountReader
	AccountWriter
	AccountSyncRepository
	AccountGroupRepository
	AccountTrashRepository
}

// accountRepository 邮箱账户数据仓库实现
type accountRepository struct {
	db *gorm.DB
}

// NewAccountRepository 创建邮箱账户数据仓库实例
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

// Create 创建账户
// 使用 Omit 排除 ProviderRef 和 AdapterRef，避免 GORM 尝试插入这些关联字段。
// 注意：GORM 对 bool 零值会跳过写入并落到列 default；创建后强制回写关键布尔/模式字段。
func (r *accountRepository) Create(ctx context.Context, account *model.EmailAccount) error {
	if err := r.db.WithContext(ctx).Omit("ProviderRef", "AdapterRef").Create(account).Error; err != nil {
		return err
	}
	// 强制写入可能为零值的字段（尤其是 sync_enabled=false）
	return r.db.WithContext(ctx).Model(account).Updates(map[string]interface{}{
		"sync_enabled":  account.SyncEnabled,
		"sync_mode":     account.SyncModeField,
		"smtp_enabled":  account.SMTPEnabled,
		"proxy_enabled": account.ProxyEnabled,
	}).Error
}

// FindByID 根据 ID 查找账户
func (r *accountRepository) FindByID(ctx context.Context, id int64) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).First(&account, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// FindByUID 根据 UID 查找账户
func (r *accountRepository) FindByUID(ctx context.Context, uid string) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).Where("uid = ?", uid).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// FindByEmail 根据邮箱地址查找账户（不包括软删除的）
func (r *accountRepository) FindByEmail(ctx context.Context, email string) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// Update 更新账户
// 使用 Omit 排除 ProviderRef 和 AdapterRef，避免 GORM 尝试更新这些关联字段
func (r *accountRepository) Update(ctx context.Context, account *model.EmailAccount) error {
	return r.db.WithContext(ctx).Omit("ProviderRef", "AdapterRef").Save(account).Error
}

// Delete 删除账户（软删除）
func (r *accountRepository) Delete(ctx context.Context, id int64) error {
	// 使用 Where 条件确保软删除生效
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.EmailAccount{}).Error
}

// List 获取账户列表
// 注意：不包括子账户（parent_account_uid 不为空的账户）
func (r *accountRepository) List(ctx context.Context, offset, limit int) ([]*model.EmailAccount, int64, error) {
	var accounts []*model.EmailAccount
	var total int64

	// 获取总数（不包括软删除的，不包括子账户）
	if err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("parent_account_uid IS NULL").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表（不包括子账户）
	err := r.db.WithContext(ctx).
		Where("parent_account_uid IS NULL").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&accounts).Error

	return accounts, total, err
}

// AccountListFilter 账户列表筛选参数
type AccountListFilter struct {
	GroupID  *int64 // 分组 ID：nil 表示所有，0 表示未分组，>0 表示具体分组
	Email    string // 邮箱搜索（模糊匹配）
	Provider string // 提供商筛选
	Status   string // 状态筛选
	Page     int    // 页码（从 1 开始）
	PageSize int    // 每页数量
}

// ListWithFilter 带筛选条件的账户列表
// 注意：不包括子账户（parent_account_uid 不为空的账户）
func (r *accountRepository) ListWithFilter(ctx context.Context, filter *AccountListFilter) ([]*model.EmailAccount, int64, error) {
	var accounts []*model.EmailAccount
	var total int64

	query := r.db.WithContext(ctx).Model(&model.EmailAccount{})

	// 排除子账户（Webhook 模式创建的子邮箱）
	query = query.Where("parent_account_uid IS NULL")

	// 分组筛选
	if filter.GroupID != nil {
		if *filter.GroupID == 0 {
			// 未分组
			query = query.Where("group_id IS NULL")
		} else if *filter.GroupID > 0 {
			// 具体分组
			query = query.Where("group_id = ?", *filter.GroupID)
		}
		// GroupID < 0 表示所有账号，不添加条件
	}

	// 邮箱搜索
	if filter.Email != "" {
		query = query.Where("email ILIKE ?", "%"+filter.Email+"%")
	}

	// 提供商筛选（通过关联查询）
	if filter.Provider != "" {
		query = query.Joins("JOIN providers ON providers.id = email_accounts.provider_id").
			Where("providers.name = ?", filter.Provider)
	}

	// 状态筛选
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (filter.Page - 1) * filter.PageSize
	if offset < 0 {
		offset = 0
	}

	// 获取列表，预加载 ProviderRef 和 AdapterRef
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("parent_account_uid IS NULL"). // 排除子账户
		Offset(offset).
		Limit(filter.PageSize).
		Order("created_at DESC")

	// 重新应用筛选条件（因为 Preload 需要新的查询）
	if filter.GroupID != nil {
		if *filter.GroupID == 0 {
			err = err.Where("group_id IS NULL")
		} else if *filter.GroupID > 0 {
			err = err.Where("group_id = ?", *filter.GroupID)
		}
	}
	if filter.Email != "" {
		err = err.Where("email ILIKE ?", "%"+filter.Email+"%")
	}
	if filter.Provider != "" {
		err = err.Joins("JOIN providers ON providers.id = email_accounts.provider_id").
			Where("providers.name = ?", filter.Provider)
	}
	if filter.Status != "" {
		err = err.Where("status = ?", filter.Status)
	}

	if findErr := err.Find(&accounts).Error; findErr != nil {
		return nil, 0, findErr
	}

	return accounts, total, nil
}

// FindAll 获取所有账户
// 预加载 ProviderRef 和 AdapterRef，确保 MarshalJSON 能正确生成 provider/protocol 字段
func (r *accountRepository) FindAll(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Find(&accounts).Error
	return accounts, err
}

// Count 统计账户总数
func (r *accountRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).Count(&count).Error
	return count, err
}

// CountActive 统计活跃账户数（启用同步的账户）
func (r *accountRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("sync_enabled = ?", true).
		Count(&count).Error
	return count, err
}
