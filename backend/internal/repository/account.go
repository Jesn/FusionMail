package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// AccountRepository 邮箱账户数据仓库接口
type AccountRepository interface {
	Create(ctx context.Context, account *model.Account) error
	FindByID(ctx context.Context, id int64) (*model.Account, error)
	FindByUID(ctx context.Context, uid string) (*model.Account, error)
	FindByEmail(ctx context.Context, email string) (*model.Account, error)
	Update(ctx context.Context, account *model.Account) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*model.Account, int64, error)
	ListSyncEnabled(ctx context.Context) ([]*model.Account, error)
	UpdateSyncStatus(ctx context.Context, uid string, status string, errorMsg string) error
	IncrementEmailCount(ctx context.Context, uid string, count int) error
	UpdateUnreadCount(ctx context.Context, uid string, count int) error

	// 系统管理需要的方法
	FindAll(ctx context.Context) ([]*model.Account, error)
	Count(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)
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
func (r *accountRepository) Create(ctx context.Context, account *model.Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// FindByID 根据 ID 查找账户
func (r *accountRepository) FindByID(ctx context.Context, id int64) (*model.Account, error) {
	var account model.Account
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
func (r *accountRepository) FindByUID(ctx context.Context, uid string) (*model.Account, error) {
	var account model.Account
	err := r.db.WithContext(ctx).Where("uid = ?", uid).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// FindByEmail 根据邮箱地址查找账户
func (r *accountRepository) FindByEmail(ctx context.Context, email string) (*model.Account, error) {
	var account model.Account
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// Update 更新账户
func (r *accountRepository) Update(ctx context.Context, account *model.Account) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// Delete 删除账户（软删除）
func (r *accountRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Account{}, id).Error
}

// List 获取账户列表
func (r *accountRepository) List(ctx context.Context, offset, limit int) ([]*model.Account, int64, error) {
	var accounts []*model.Account
	var total int64

	// 获取总数（不包括软删除的）
	if err := r.db.WithContext(ctx).Model(&model.Account{}).Count(&total).Error; err != nil {
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
func (r *accountRepository) ListSyncEnabled(ctx context.Context) ([]*model.Account, error) {
	var accounts []*model.Account
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
		Model(&model.Account{}).
		Where("uid = ?", uid).
		Updates(updates).Error
}

// IncrementEmailCount 增加邮件数量
func (r *accountRepository) IncrementEmailCount(ctx context.Context, uid string, count int) error {
	return r.db.WithContext(ctx).
		Model(&model.Account{}).
		Where("uid = ?", uid).
		UpdateColumn("total_emails", gorm.Expr("total_emails + ?", count)).Error
}

// UpdateUnreadCount 更新未读数量
func (r *accountRepository) UpdateUnreadCount(ctx context.Context, uid string, count int) error {
	return r.db.WithContext(ctx).
		Model(&model.Account{}).
		Where("uid = ?", uid).
		Update("unread_count", count).Error
}

// FindAll 获取所有账户
func (r *accountRepository) FindAll(ctx context.Context) ([]*model.Account, error) {
	var accounts []*model.Account
	err := r.db.WithContext(ctx).Find(&accounts).Error
	return accounts, err
}

// Count 统计账户总数
func (r *accountRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Account{}).Count(&count).Error
	return count, err
}

// CountActive 统计活跃账户数（启用同步的账户）
func (r *accountRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Account{}).
		Where("sync_enabled = ?", true).
		Count(&count).Error
	return count, err
}
