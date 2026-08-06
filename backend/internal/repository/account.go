package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// ListSyncEnabled 获取启用同步的账户列表
// 排除：Webhook 模式、webhook_ 子账户、有父账户的子邮箱（均不参与独立轮询）
func (r *accountRepository) ListSyncEnabled(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("sync_enabled = ? AND status = ?", true, "active").
		Where("(sync_mode IS NULL OR sync_mode = '' OR sync_mode = ?)", model.SyncModePolling).
		Where("parent_account_uid IS NULL").
		Where("uid NOT LIKE ?", "webhook_%").
		Order("last_sync_at ASC NULLS FIRST").
		Find(&accounts).Error
	return accounts, err
}

// HealWebhookChildPollingFlags 自愈：关闭不应轮询的子账户/webhook 子账户的 sync_enabled
// 并修正 webhook_ 账户的 sync_mode。返回受影响行数。
// 背景：GORM Create 对 bool 零值会跳过写入，导致 DB default(true) 把子账户写成可同步。
func (r *accountRepository) HealWebhookChildPollingFlags(ctx context.Context) (int64, error) {
	var affected int64

	// 所有子账户 + webhook_ 前缀：关闭轮询
	res := r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("sync_enabled = ?", true).
		Where("parent_account_uid IS NOT NULL OR uid LIKE ?", "webhook_%").
		Update("sync_enabled", false)
	if res.Error != nil {
		return 0, res.Error
	}
	affected += res.RowsAffected

	// webhook_ 子账户：强制 sync_mode=webhook
	res2 := r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid LIKE ?", "webhook_%").
		Where("sync_mode IS NULL OR sync_mode = '' OR sync_mode = ?", model.SyncModePolling).
		Update("sync_mode", model.SyncModeWebhook)
	if res2.Error != nil {
		return affected, res2.Error
	}
	affected += res2.RowsAffected

	return affected, nil
}

// UpdateSyncStatus 更新同步状态
// 无论成功还是失败，都更新 last_sync_at，确保调度器能正确计算下次同步时间
func (r *accountRepository) UpdateSyncStatus(ctx context.Context, uid string, status string, errorMsg string) error {
	updates := map[string]interface{}{
		"last_sync_status": status,
		"last_sync_error":  errorMsg,
		"last_sync_at":     gorm.Expr("NOW()"), // 无论成功失败都更新，避免失败后不再重试
	}

	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Updates(updates).Error
}

// IncrementEmailCount 增加邮件数量
func (r *accountRepository) IncrementEmailCount(ctx context.Context, uid string, count int) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		UpdateColumn("total_emails", gorm.Expr("total_emails + ?", count)).Error
}

// UpdateUnreadCount 更新未读数量
func (r *accountRepository) UpdateUnreadCount(ctx context.Context, uid string, count int) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Update("unread_count", count).Error
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

// IncrementConsecutiveFailures 增加连续失败计数
// 使用事务和行锁确保原子性和并发安全
func (r *accountRepository) IncrementConsecutiveFailures(ctx context.Context, uid string) (int, error) {
	var newCount int

	// 使用事务确保原子性
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account model.EmailAccount

		// 锁定行，防止并发问题
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", uid).
			First(&account).Error; err != nil {
			accountRepoLog.Error("查找账户失败（增加失败计数）: %v", err)
			return err
		}

		oldCount := account.ConsecutiveAuthFailures
		newCount = oldCount + 1

		// 使用 Updates 直接更新字段
		result := tx.Model(&account).Updates(map[string]interface{}{
			"consecutive_auth_failures": newCount,
			"updated_at":                time.Now(),
		})

		if result.Error != nil {
			accountRepoLog.Error("更新失败计数失败: %v", result.Error)
			return result.Error
		}

		accountRepoLog.Debug("增加失败计数: uid=%s, %d -> %d (影响行数: %d)",
			uid, oldCount, newCount, result.RowsAffected)
		return nil
	})

	return newCount, err
}

// ResetConsecutiveFailures 重置连续失败计数
func (r *accountRepository) ResetConsecutiveFailures(ctx context.Context, uid string) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Updates(map[string]interface{}{
			"consecutive_auth_failures": 0,
			"updated_at":                time.Now(),
		}).Error
}

// AutoDisableAccount 自动禁用账号
// 仅禁用状态为 active 的账号
func (r *accountRepository) AutoDisableAccount(ctx context.Context, uid string, reason string) error {
	now := time.Now()

	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ? AND status = ?", uid, "active").
		Updates(map[string]interface{}{
			"status":           "disabled",
			"disable_reason":   reason,
			"auto_disabled_at": now,
			"last_sync_error":  "账号已自动禁用（连续认证失败）",
			"updated_at":       now,
		}).Error
}

// MarkRemoteMailboxDeleted 将子账户标记为「远端邮箱已删除」（保留本地邮件）
func (r *accountRepository) MarkRemoteMailboxDeleted(ctx context.Context, uid string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Updates(map[string]interface{}{
			"status":           model.AccountStatusDisabled,
			"disable_reason":   model.DisableReasonRemoteMailboxDeleted,
			"auto_disabled_at": now,
			"sync_enabled":     false,
			"updated_at":       now,
		}).Error
}

// ReactivateFromRemoteOrphan 将因远端删除而禁用的账户恢复为 active（例如地址复活后又收到信）
func (r *accountRepository) ReactivateFromRemoteOrphan(ctx context.Context, uid string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ? AND status = ? AND disable_reason = ?",
			uid, model.AccountStatusDisabled, model.DisableReasonRemoteMailboxDeleted).
		Updates(map[string]interface{}{
			"status":           model.AccountStatusActive,
			"disable_reason":   "",
			"auto_disabled_at": nil,
			"updated_at":       now,
		}).Error
}

// AutoSoftDeleteAccount 自动软删除账号（放入回收站）
// 用于批量导入的 OAuth2 账号连续认证失败后的自动清理
func (r *accountRepository) AutoSoftDeleteAccount(ctx context.Context, uid string, reason string) error {
	now := time.Now()

	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ? AND deleted_at IS NULL", uid).
		Updates(map[string]interface{}{
			"status":           "disabled",
			"disable_reason":   reason,
			"auto_disabled_at": now,
			"last_sync_error":  "账号已自动移入回收站（连续认证失败）",
			"deleted_at":       now,
			"updated_at":       now,
		}).Error
}

// FindDeletedByEmail 根据邮箱地址查找已软删除的账户列表
func (r *accountRepository) FindDeletedByEmail(ctx context.Context, email string) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).Unscoped().
		Where("email = ? AND deleted_at IS NOT NULL", email).
		Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// FindAllWithDeleted 获取所有账号（包括软删除的）
func (r *accountRepository) FindAllWithDeleted(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).Unscoped().Find(&accounts).Error
	return accounts, err
}

// FindDeleted 获取回收站中的账号（仅软删除的）
func (r *accountRepository) FindDeleted(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).Unscoped().
		Where("deleted_at IS NOT NULL").
		Order("deleted_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindDeletedBefore 查找在指定时间之前删除的账号
func (r *accountRepository) FindDeletedBefore(ctx context.Context, cutoffTime time.Time) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoffTime).
		Find(&accounts).Error
	return accounts, err
}

// FindByUIDIncludingDeleted 根据 UID 查找账号（包括软删除的）
func (r *accountRepository) FindByUIDIncludingDeleted(ctx context.Context, uid string) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).Unscoped().Where("uid = ?", uid).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// Restore 恢复软删除的账号
func (r *accountRepository) Restore(ctx context.Context, uid string) error {
	return r.db.WithContext(ctx).Unscoped().Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"updated_at": time.Now(),
		}).Error
}

// ForceDelete 永久删除账号（硬删除）
func (r *accountRepository) ForceDelete(ctx context.Context, uid string) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("uid = ?", uid).
		Delete(&model.EmailAccount{}).Error
}

// UpdateSyncProgress 更新同步进度
// Requirements: 3.2 - 每批处理后保存进度
func (r *accountRepository) UpdateSyncProgress(ctx context.Context, uid string, cursor string, progressJSON string) error {
	// 处理空字符串：PostgreSQL JSON 类型不接受空字符串，需要使用 null
	var progressValue interface{}
	if progressJSON == "" {
		progressValue = nil // 使用 NULL
	} else {
		progressValue = progressJSON
	}

	updates := map[string]interface{}{
		"sync_cursor":        cursor,
		"sync_progress_json": progressValue,
		"updated_at":         time.Now(),
	}

	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Updates(updates).Error
}

// UpdateUIDSyncState 更新 UID 增量同步状态
// Requirements: 6.1 - 持久化 uid_validity 和 last_uid
func (r *accountRepository) UpdateUIDSyncState(ctx context.Context, uid string, uidValidity, lastUID int64) error {
	updates := map[string]interface{}{
		"uid_validity": uidValidity,
		"last_uid":     lastUID,
		"updated_at":   time.Now(),
	}

	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Updates(updates).Error
}

// FindByGroupID 根据分组 ID 查找账号列表
// 注意：不包括子账户（parent_account_uid 不为空的账户）
func (r *accountRepository) FindByGroupID(ctx context.Context, groupID int64) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND parent_account_uid IS NULL", groupID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindAllByGroupID 根据分组 ID 查找所有账号（包括子账户）
// 用于邮件列表查询，需要包含子账户的邮件
func (r *accountRepository) FindAllByGroupID(ctx context.Context, groupID int64) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindUngrouped 查找未分组的账号列表
// 注意：不包括子账户（parent_account_uid 不为空的账户）
func (r *accountRepository) FindUngrouped(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("group_id IS NULL AND parent_account_uid IS NULL").
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// UpdateGroupID 更新账号的分组 ID
func (r *accountRepository) UpdateGroupID(ctx context.Context, uid string, groupID *int64) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid = ?", uid).
		Update("group_id", groupID).Error
}

// BatchUpdateGroupID 批量更新账号的分组 ID
func (r *accountRepository) BatchUpdateGroupID(ctx context.Context, uids []string, groupID *int64) error {
	if len(uids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.EmailAccount{}).
		Where("uid IN ?", uids).
		Update("group_id", groupID).Error
}

// FindByUIDWithRelations 根据 UID 查找账户并预加载 Provider 和 Adapter 关联
func (r *accountRepository) FindByUIDWithRelations(ctx context.Context, uid string) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("uid = ?", uid).
		First(&account).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByIDWithRelations 根据 ID 查找账户并预加载 Provider 和 Adapter 关联
func (r *accountRepository) FindByIDWithRelations(ctx context.Context, id int64) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		First(&account, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// ListWithRelations 获取账户列表并预加载 Provider 和 Adapter 关联
// 注意：不包括子账户（parent_account_uid 不为空的账户）
func (r *accountRepository) ListWithRelations(ctx context.Context, offset, limit int) ([]*model.EmailAccount, int64, error) {
	var accounts []*model.EmailAccount
	var total int64

	// 获取总数（不包括软删除的，不包括子账户）
	if err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("parent_account_uid IS NULL").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表并预加载关联（不包括子账户）
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("parent_account_uid IS NULL").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&accounts).Error

	return accounts, total, err
}

// ListSyncEnabledWithRelations 获取启用同步的账户列表并预加载 Provider 和 Adapter 关联
// 过滤规则与 ListSyncEnabled 一致
func (r *accountRepository) ListSyncEnabledWithRelations(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("sync_enabled = ? AND status = ?", true, "active").
		Where("(sync_mode IS NULL OR sync_mode = '' OR sync_mode = ?)", model.SyncModePolling).
		Where("parent_account_uid IS NULL").
		Where("uid NOT LIKE ?", "webhook_%").
		Order("last_sync_at ASC NULLS FIRST").
		Find(&accounts).Error
	return accounts, err
}

// FindByProviderID 根据 Provider ID 查找账户列表
func (r *accountRepository) FindByProviderID(ctx context.Context, providerID int64) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindByAdapterID 根据 Adapter ID 查找账户列表
func (r *accountRepository) FindByAdapterID(ctx context.Context, adapterID int64) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("adapter_id = ?", adapterID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindByProviderIDs 根据多个 Provider ID 查找账户列表（分页）
func (r *accountRepository) FindByProviderIDs(ctx context.Context, providerIDs []int64, page, pageSize int) ([]*model.EmailAccount, int64, error) {
	if len(providerIDs) == 0 {
		return []*model.EmailAccount{}, 0, nil
	}

	var accounts []*model.EmailAccount
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("provider_id IN ?", providerIDs).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("provider_id IN ?", providerIDs).
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&accounts).Error

	return accounts, total, err
}

// FindByParentAccountUID 根据父账户 UID 查找子邮箱账户列表
func (r *accountRepository) FindByParentAccountUID(ctx context.Context, parentUID string) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("parent_account_uid = ?", parentUID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// ChildAccountListFilter 子邮箱分页查询条件
type ChildAccountListFilter struct {
	ParentUID string // 父账户 UID（必填）
	Include   string // active | orphaned | all
	Email     string // 邮箱模糊搜索（可选）
	Page      int    // 页码，从 1 起
	PageSize  int    // 每页数量
}

// FindChildrenByParent 分页查询父账户下的子邮箱
func (r *accountRepository) FindChildrenByParent(ctx context.Context, filter *ChildAccountListFilter) ([]*model.EmailAccount, int64, error) {
	if filter == nil || filter.ParentUID == "" {
		return []*model.EmailAccount{}, 0, nil
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("parent_account_uid = ?", filter.ParentUID)

	switch strings.ToLower(strings.TrimSpace(filter.Include)) {
	case "orphaned":
		query = query.Where("status = ? AND disable_reason = ?",
			model.AccountStatusDisabled, model.DisableReasonRemoteMailboxDeleted)
	case "all":
		// 不过滤状态
	default: // active
		query = query.Where("status = ?", model.AccountStatusActive)
	}

	if kw := strings.TrimSpace(filter.Email); kw != "" {
		// 转义 ILIKE 通配符，避免用户输入 %/_ 扩大匹配面
		escaped := strings.ReplaceAll(kw, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `%`, `\%`)
		escaped = strings.ReplaceAll(escaped, `_`, `\_`)
		query = query.Where("email ILIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var accounts []*model.EmailAccount
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&accounts).Error
	if err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

// FindByDomain 按域名查找账户（用于 Webhook Admin 模式）
// 查找 EncryptedCredentials 中包含指定域名的账户
// 主要用于 Webhook 接收时，根据收件人域名匹配 Admin 模式的账户
func (r *accountRepository) FindByDomain(ctx context.Context, domain string) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount

	// 使用 PostgreSQL 的 JSON 操作符查询
	// 查找 domains 字段包含指定域名的账户
	// 注意：domains 字段格式为 "example.com, test.org"（逗号分隔）
	err := r.db.WithContext(ctx).
		Where("encrypted_credentials LIKE ?", "%"+domain+"%").
		Where("deleted_at IS NULL").
		Find(&accounts).Error

	if err != nil {
		return nil, err
	}

	return accounts, nil
}
