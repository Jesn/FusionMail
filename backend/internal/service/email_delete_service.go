package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// DeleteEmail 删除邮件（软删除）
func (s *emailService) DeleteEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 本地删除
	deleted := true
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, nil, &deleted); err != nil {
		s.logger.Error("删除邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}

	// 后台执行服务器软删除
	go s.tryServerSoftDelete(context.Background(), email)

	return nil
}

// BatchDeleteEmails 批量删除邮件（软删除）
func (s *emailService) BatchDeleteEmails(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "邮件 ID 列表不能为空")
	}

	uniqueIDs := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return 0, dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "邮件 ID 格式无效")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}

	emails, err := s.emailRepo.FindByIDs(ctx, uniqueIDs)
	if err != nil {
		s.logger.Error("批量查询邮件失败: count=%d, error=%v", len(uniqueIDs), err)
		return 0, fmt.Errorf("database error: %w", err)
	}
	if len(emails) == 0 {
		return 0, nil
	}

	deleteIDs := make([]int64, 0, len(emails))
	serverDeleteEmails := make([]*model.Email, 0, len(emails))
	for _, email := range emails {
		if email == nil || email.IsDeleted {
			continue
		}
		deleteIDs = append(deleteIDs, email.ID)
		serverDeleteEmails = append(serverDeleteEmails, email)
	}
	if len(deleteIDs) == 0 {
		return 0, nil
	}

	deletedCount, err := s.emailRepo.BatchUpdateLocalDeleted(ctx, deleteIDs, true)
	if err != nil {
		s.logger.Error("批量删除邮件失败: count=%d, error=%v", len(deleteIDs), err)
		return 0, fmt.Errorf("database error: %w", err)
	}

	if deletedCount > 0 {
		go func(emails []*model.Email) {
			for _, email := range emails {
				s.tryServerSoftDelete(context.Background(), email)
			}
		}(serverDeleteEmails)
	}

	return deletedCount, nil
}

// RestoreEmail 恢复已删除邮件（从垃圾箱恢复到收件箱）
func (s *emailService) RestoreEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 恢复：取消删除，同时取消归档，确保回到收件箱
	deleted := false
	archived := false
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, &archived, &deleted); err != nil {
		s.logger.Error("恢复邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// tryServerSoftDelete 尝试在服务器上软删除邮件
func (s *emailService) tryServerSoftDelete(ctx context.Context, email *model.Email) {
	// 获取账号信息
	account, err := s.accountRepo.FindByUID(ctx, email.AccountUID)
	if err != nil || account == nil {
		s.logger.Debug("获取账户失败: email_id=%d, error=%v", email.ID, err)
		return
	}

	// 检查是否启用服务器软删除
	if account.ServerDeletePolicy != "soft" {
		return
	}

	// 解析凭证
	credentials, err := s.credentialResolver.Resolve(account)
	if err != nil {
		s.logger.Debug("解析凭证失败: account=%s, error=%v", account.UID, err)
		return
	}

	// 创建适配器
	providerName := account.GetProviderName()
	protocol := account.GetProtocol()
	mailAdapter, err := s.adapterFactory.CreateProviderFromAccount(
		providerName,
		protocol,
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		s.logger.Debug("创建适配器失败: account=%s, error=%v", account.UID, err)
		return
	}

	// 连接到邮箱服务器
	if err := mailAdapter.Connect(ctx); err != nil {
		s.logger.Debug("连接邮箱服务器失败: account=%s, error=%v", account.UID, err)
		return
	}
	defer mailAdapter.Disconnect()

	// 检查是否支持软删除
	softDeleter, ok := mailAdapter.(adapter.SoftDeleter)
	if !ok {
		s.logger.Debug("适配器不支持软删除: account=%s", account.UID)
		return
	}

	// 执行软删除（带重试）
	if err := s.softDeleteWithRetry(ctx, softDeleter, email.ProviderID); err != nil {
		s.logger.Debug("服务器软删除失败: email_id=%d, error=%v", email.ID, err)
	}
}

// softDeleteWithRetry 带重试的软删除
func (s *emailService) softDeleteWithRetry(ctx context.Context, deleter adapter.SoftDeleter, providerID string) error {
	maxRetries := 3
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := deleter.MoveToTrash(ctx, providerID)
		if err == nil {
			return nil
		}

		// 404 视为成功
		if strings.Contains(err.Error(), "404") {
			return nil
		}

		if attempt < maxRetries {
			select {
			case <-time.After(backoff):
				backoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// PermanentDeleteEmail 永久删除单封邮件（物理删除）
func (s *emailService) PermanentDeleteEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 只允许删除已在回收站中的邮件
	if !email.IsDeleted {
		return dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "只能永久删除回收站中的邮件")
	}

	// 物理删除邮件
	if err := s.emailRepo.Delete(ctx, id); err != nil {
		s.logger.Error("永久删除邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}

	return nil
}

// BatchPermanentDeleteEmails 批量永久删除邮件（物理删除）
func (s *emailService) BatchPermanentDeleteEmails(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	deletedCount := int64(0)
	for _, id := range ids {
		// 验证邮件是否存在且在回收站中
		email, err := s.emailRepo.FindByID(ctx, id)
		if err != nil {
			s.logger.Debug("查询邮件失败: id=%d, error=%v", id, err)
			continue
		}
		if email == nil {
			continue
		}
		if !email.IsDeleted {
			continue // 跳过不在回收站中的邮件
		}

		// 物理删除
		if err := s.emailRepo.Delete(ctx, id); err != nil {
			s.logger.Debug("永久删除邮件失败: id=%d, error=%v", id, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// EmptyTrash 清空回收站（永久删除所有已删除邮件）
func (s *emailService) EmptyTrash(ctx context.Context) (int64, error) {
	// 获取所有已删除的邮件
	trueVal := true
	filter := &repository.EmailFilter{
		IsDeleted: &trueVal,
	}

	// 分批获取并删除
	deletedCount := int64(0)
	batchSize := 100
	offset := 0

	for {
		emails, _, err := s.emailRepo.List(ctx, filter, offset, batchSize)
		if err != nil {
			return deletedCount, fmt.Errorf("failed to list deleted emails: %w", err)
		}

		if len(emails) == 0 {
			break
		}

		for _, email := range emails {
			if err := s.emailRepo.Delete(ctx, email.ID); err != nil {
				s.logger.Debug("清空回收站时删除邮件失败: id=%d, error=%v", email.ID, err)
				continue
			}
			deletedCount++
		}

		// 由于删除后数据减少，不需要增加 offset
		if len(emails) < batchSize {
			break
		}
	}

	return deletedCount, nil
}
