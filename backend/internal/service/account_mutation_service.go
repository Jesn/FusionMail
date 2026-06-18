package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
)

func (s *accountService) Update(ctx context.Context, uid string, req *UpdateAccountRequest) (*response.AccountResponse, error) {
	// 获取现有账户（直接从 repo 获取 model，需要修改字段）
	account, err := s.accountRepo.FindByUID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return nil, dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 更新字段
	if req.Email != nil {
		account.Email = *req.Email
	}
	if req.Password != nil {
		encryptedPassword, err := s.cryptoService.Encrypt([]byte(*req.Password))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		account.EncryptedCredentials = encryptedPassword
	}
	if req.SyncEnabled != nil {
		account.SyncEnabled = *req.SyncEnabled
	}
	if req.SyncInterval != nil {
		account.SyncInterval = *req.SyncInterval
	}
	// 更新删除策略
	if req.ServerDeletePolicy != nil {
		account.ServerDeletePolicy = *req.ServerDeletePolicy
	}

	// 更新首次同步优化配置 (Requirements: 6.1, 6.2)
	if req.FirstSyncDays != nil {
		// 验证配置值范围
		if *req.FirstSyncDays < model.MinFirstSyncDays || *req.FirstSyncDays > model.MaxFirstSyncDays {
			return nil, fmt.Errorf("first_sync_days must be between %d and %d", model.MinFirstSyncDays, model.MaxFirstSyncDays)
		}
		account.FirstSyncDays = *req.FirstSyncDays
	}
	if req.BatchSize != nil {
		if *req.BatchSize < model.MinBatchSize || *req.BatchSize > model.MaxBatchSize {
			return nil, fmt.Errorf("batch_size must be between %d and %d", model.MinBatchSize, model.MaxBatchSize)
		}
		account.BatchSize = *req.BatchSize
	}
	if req.MaxEmailsPerSync != nil {
		if *req.MaxEmailsPerSync < model.MinMaxEmailsPerSync || *req.MaxEmailsPerSync > model.MaxMaxEmailsPerSync {
			return nil, fmt.Errorf("max_emails_per_sync must be between %d and %d", model.MinMaxEmailsPerSync, model.MaxMaxEmailsPerSync)
		}
		account.MaxEmailsPerSync = *req.MaxEmailsPerSync
	}

	// 更新分组 ID
	// 注意：req.GroupID 为 nil 表示不更新，req.GroupID 指向的值为 0 或 null 表示移出分组
	if req.GroupID != nil {
		if *req.GroupID == 0 {
			// 移出分组
			account.GroupID = nil
		} else {
			// 分配到指定分组
			account.GroupID = req.GroupID
		}
	}

	account.UpdatedAt = time.Now()

	// 保存更新
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	return toAccountResponse(account), nil
}

// Delete 删除账户（软删除）
func (s *accountService) Delete(ctx context.Context, uid string) error {
	// 先获取账户以获得 ID
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	// 软删除该账号下的所有邮件
	if err := s.emailRepo.SoftDeleteByAccountUID(ctx, uid); err != nil {
		s.logger.Warn("软删除账户邮件失败: uid=%s, error=%v", uid, err)
		// 不返回错误，继续删除账号
	}

	// 执行软删除账号
	if err := s.accountRepo.Delete(ctx, account.ID); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	s.logger.Info("账户已删除: uid=%s, email=%s", uid, account.Email)
	return nil
}

// GetByUIDWithRelations 根据 UID 获取账户并预加载 Provider 和 Adapter 关联
func (s *accountService) GetByUIDWithRelations(ctx context.Context, uid string) (*model.EmailAccount, error) {
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, uid)
	if err != nil {
		s.logger.Error("查询账户失败: uid=%s, error=%v", uid, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return nil, dto.NewAPIError(dto.ErrAccountNotFound)
	}
	return account, nil
}

// ListWithRelations 获取账户列表并预加载 Provider 和 Adapter 关联
func (s *accountService) ListWithRelations(ctx context.Context) ([]*model.EmailAccount, error) {
	accounts, _, err := s.accountRepo.ListWithRelations(ctx, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts with relations: %w", err)
	}
	return accounts, nil
}

// ListSyncEnabledWithRelations 获取启用同步的账户列表并预加载 Provider 和 Adapter 关联
func (s *accountService) ListSyncEnabledWithRelations(ctx context.Context) ([]*model.EmailAccount, error) {
	accounts, err := s.accountRepo.ListSyncEnabledWithRelations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync enabled accounts with relations: %w", err)
	}
	return accounts, nil
}
