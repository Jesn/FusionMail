package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AccountRepository 邮箱账户数据仓库接口
type AccountRepository interface {
	Create(ctx context.Context, account *model.EmailAccount) error
	FindByID(ctx context.Context, id int64) (*model.EmailAccount, error)
	FindByUID(ctx context.Context, uid string) (*model.EmailAccount, error)
	FindByEmail(ctx context.Context, email string) (*model.EmailAccount, error)
	Update(ctx context.Context, account *model.EmailAccount) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*model.EmailAccount, int64, error)
	ListSyncEnabled(ctx context.Context) ([]*model.EmailAccount, error)
	UpdateSyncStatus(ctx context.Context, uid string, status string, errorMsg string) error
	IncrementEmailCount(ctx context.Context, uid string, count int) error
	UpdateUnreadCount(ctx context.Context, uid string, count int) error

	// 系统管理需要的方法
	FindAll(ctx context.Context) ([]*model.EmailAccount, error)
	Count(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)

	// 短期邮箱过期处理相关方法
	IncrementConsecutiveFailures(ctx context.Context, uid string) (int, error)
	ResetConsecutiveFailures(ctx context.Context, uid string) error
	AutoDisableAccount(ctx context.Context, uid string, reason string) error
	AutoSoftDeleteAccount(ctx context.Context, uid string, reason string) error // 自动软删除（放入回收站）

	// 软删除管理方法
	FindAllWithDeleted(ctx context.Context) ([]*model.EmailAccount, error)
	FindDeleted(ctx context.Context) ([]*model.EmailAccount, error)
	FindDeletedByEmail(ctx context.Context, email string) ([]*model.EmailAccount, error)
	FindDeletedBefore(ctx context.Context, cutoffTime time.Time) ([]*model.EmailAccount, error)

	FindByUIDIncludingDeleted(ctx context.Context, uid string) (*model.EmailAccount, error)
	Restore(ctx context.Context, uid string) error
	ForceDelete(ctx context.Context, uid string) error
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
func (r *accountRepository) Create(ctx context.Context, account *model.EmailAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
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
func (r *accountRepository) Update(ctx context.Context, account *model.EmailAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// Delete 删除账户（软删除）
func (r *accountRepository) Delete(ctx context.Context, id int64) error {
	// 使用 Where 条件确保软删除生效
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.EmailAccount{}).Error
}

// List 获取账户列表
func (r *accountRepository) List(ctx context.Context, offset, limit int) ([]*model.EmailAccount, int64, error) {
	var accounts []*model.EmailAccount
	var total int64

	// 获取总数（不包括软删除的）
	if err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&accounts).Error

	return accounts, total, err
}

// ListSyncEnabled 获取启用同步的账户列表
func (r *accountRepository) ListSyncEnabled(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("sync_enabled = ? AND status = ?", true, "active").
		Order("last_sync_at ASC NULLS FIRST").
		Find(&accounts).Error
	return accounts, err
}

// UpdateSyncStatus 更新同步状态
func (r *accountRepository) UpdateSyncStatus(ctx context.Context, uid string, status string, errorMsg string) error {
	updates := map[string]interface{}{
		"last_sync_status": status,
		"last_sync_error":  errorMsg,
	}

	if status == "success" {
		updates["last_sync_at"] = gorm.Expr("NOW()")
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
func (r *accountRepository) FindAll(ctx context.Context) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).Find(&accounts).Error
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
			log.Printf("[ERROR] Failed to find account for increment: %v", err)
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
			log.Printf("[ERROR] Failed to update failure count: %v", result.Error)
			return result.Error
		}

		log.Printf("[DEBUG] Incremented failure count for %s: %d -> %d (rows affected: %d)",
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
