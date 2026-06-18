package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/internal/model"
)

// ListDeleted 获取回收站中的账号（仅软删除的）
func (s *accountService) ListDeleted(ctx context.Context) ([]*model.EmailAccount, error) {
	accounts, err := s.accountRepo.FindDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list deleted accounts: %w", err)
	}
	return accounts, nil
}

// Restore 恢复软删除的账号
func (s *accountService) Restore(ctx context.Context, uid string) error {
	// 先检查账号是否存在（包括软删除的）
	account, err := s.accountRepo.FindByUIDIncludingDeleted(ctx, uid)
	if err != nil {
		return err
	}
	if account == nil {
		return dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 检查是否已恢复
	if !account.DeletedAt.Valid {
		return dto.NewAPIErrorWithMessage(
			dto.ErrAccountInvalid,
			"账号未被删除",
		)
	}

	// 恢复该账号下的所有邮件
	if err := s.emailRepo.RestoreByAccountUID(ctx, uid); err != nil {
		s.logger.Warn("恢复账户邮件失败: uid=%s, error=%v", uid, err)
		// 不返回错误，继续恢复账号
	}

	// 恢复账号
	if err := s.accountRepo.Restore(ctx, uid); err != nil {
		return fmt.Errorf("failed to restore account: %w", err)
	}

	s.logger.Info("账户恢复成功: uid=%s, email=%s", uid, account.Email)
	return nil
}

// ForceDelete 永久删除账号（包括所有相关数据）
func (s *accountService) ForceDelete(ctx context.Context, uid string) error {
	// 先检查账号是否存在（包括软删除的）
	account, err := s.accountRepo.FindByUIDIncludingDeleted(ctx, uid)
	if err != nil {
		return err
	}
	if account == nil {
		return dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 删除该账号下的所有邮件（包括附件等相关数据）
	if err := s.emailRepo.DeleteByAccountUID(ctx, uid); err != nil {
		return fmt.Errorf("failed to delete emails for account before force delete: %w", err)
	}

	// TODO: 删除附件文件（如果有本地存储）
	// TODO: 删除相关的规则、Webhook 日志等

	// 执行硬删除账号
	if err := s.accountRepo.ForceDelete(ctx, uid); err != nil {
		return fmt.Errorf("failed to force delete account: %w", err)
	}

	s.logger.Info("账户永久删除: uid=%s, email=%s", uid, account.Email)
	return nil
}

// CleanupTrash 清理回收站（删除超过指定天数的软删除账号）
// days: 保留天数，-1 表示不清理
// 返回清理的账号数量
func (s *accountService) CleanupTrash(ctx context.Context, days int) (int, error) {
	if days < 0 {
		s.logger.Debug("回收站清理已禁用: days=%d", days)
		return 0, nil
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 获取需要清理的账号
	accounts, err := s.accountRepo.FindDeletedBefore(ctx, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to find accounts to cleanup: %w", err)
	}

	if len(accounts) == 0 {
		return 0, nil
	}

	// 逐个永久删除
	cleanedCount := 0
	for _, account := range accounts {
		if err := s.ForceDelete(ctx, account.UID); err != nil {
			s.logger.Warn("清理时删除账户失败: uid=%s, error=%v", account.UID, err)
			continue
		}
		cleanedCount++
	}

	s.logger.Info("回收站清理完成: cleaned=%d, total=%d", cleanedCount, len(accounts))
	return cleanedCount, nil
}
