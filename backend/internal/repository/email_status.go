package repository

import (
	"context"
	"fusionmail/internal/model"
)

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
