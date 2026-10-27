package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
)

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
