package repository

import (
	"context"
	"fusionmail/internal/model"
)

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
