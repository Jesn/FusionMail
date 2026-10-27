package repository

import (
	"context"
	"fusionmail/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
