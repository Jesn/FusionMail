package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
)

// TestConnection 测试账户连接
func (s *accountService) TestConnection(ctx context.Context, uid string) error {
	// 获取账户（预加载 Provider 关联以获取服务器配置）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return dto.NewAPIErrorWithMessage(dto.ErrAccountNotFound, "账户不存在")
	}

	// 解密密码
	decryptedData, err := s.cryptoService.Decrypt(account.EncryptedCredentials)
	if err != nil {
		s.logger.Error("解密凭证失败: uid=%s, error=%v", uid, err)
		return fmt.Errorf("decryption error: %w", err)
	}

	// 创建凭证
	authType := account.GetAuthType()
	credentials := &adapter.Credentials{
		Email:    account.Email,
		Password: string(decryptedData),
		AuthType: authType,
	}

	// 设置服务器配置（从 Provider 获取）
	protocol := account.GetProtocol()
	if protocol == "imap" {
		host, port, encryption := account.GetIMAPConfig()
		credentials.Host = host
		credentials.Port = port

		// 设置加密方式
		switch encryption {
		case "ssl", "":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true
		}
	} else if protocol == "pop3" {
		host, port, encryption := account.GetPOP3Config()
		credentials.Host = host
		credentials.Port = port

		// 设置加密方式
		switch encryption {
		case "ssl", "":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true
		}
	}

	// 智能修复常见的配置错误
	if credentials.Host == "mail.linuxdo.org" {
		s.logger.Debug("自动修复主机地址: %s -> mail.linux.do", credentials.Host)
		credentials.Host = "mail.linux.do"
	}

	// 验证必要的配置
	if credentials.Host == "" || credentials.Port == 0 {
		return dto.NewAPIErrorWithMessage(
			dto.ErrAccountInvalid,
			fmt.Sprintf("服务器配置缺失: host=%s, port=%d", credentials.Host, credentials.Port),
		)
	}

	// 创建适配器
	providerName := account.GetProviderName()
	provider, err := s.adapterFactory.CreateProviderFromAccount(
		providerName,
		protocol,
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		return dto.NewAPIErrorWithMessage(
			dto.ErrAccountInvalid,
			"账户配置无效: "+err.Error(),
		)
	}

	// 测试连接
	if err := provider.TestConnection(ctx); err != nil {
		return dto.NewAPIErrorWithMessage(
			dto.ErrConnectionFailed,
			"连接失败: "+err.Error(),
		)
	}

	return nil
}

// SetStatus 设置账户状态
func (s *accountService) SetStatus(ctx context.Context, uid string, status string) error {
	// 获取账户（直接从 repo 获取 model，需要修改字段）
	account, err := s.accountRepo.FindByUID(ctx, uid)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 更新状态
	account.Status = status
	account.UpdatedAt = time.Now()

	// 保存更新
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("failed to update account status: %w", err)
	}

	return nil
}

// DisableAccount 禁用账户
func (s *accountService) DisableAccount(ctx context.Context, uid string) error {
	return s.SetStatus(ctx, uid, "disabled")
}

// EnableAccount 启用账户
// 重置所有自动禁用相关字段，允许账号重新同步
func (s *accountService) EnableAccount(ctx context.Context, uid string) error {
	// 获取账户（直接从 repo 获取 model，需要修改字段）
	account, err := s.accountRepo.FindByUID(ctx, uid)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 重置所有禁用相关字段
	account.Status = "active"
	account.ConsecutiveAuthFailures = 0
	account.AutoDisabledAt = nil
	account.DisableReason = ""
	account.LastSyncError = ""
	account.UpdatedAt = time.Now()

	// 保存更新
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("failed to enable account: %w", err)
	}

	s.logger.Info("手动重新启用账户: uid=%s, email=%s", account.UID, account.Email)
	return nil
}

// BatchEnableAccounts 批量启用账户
func (s *accountService) BatchEnableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error) {
	result := &BatchOperationResult{
		Total:       len(uids),
		FailedItems: make([]BatchOperationFailedItem, 0),
	}

	for _, uid := range uids {
		// 获取账户信息（用于错误报告）
		account, err := s.GetByUID(ctx, uid)
		if err != nil {
			result.Failed++
			result.FailedItems = append(result.FailedItems, BatchOperationFailedItem{
				UID:   uid,
				Email: "",
				Error: err.Error(),
			})
			continue
		}

		// 启用账户
		if err := s.EnableAccount(ctx, uid); err != nil {
			result.Failed++
			result.FailedItems = append(result.FailedItems, BatchOperationFailedItem{
				UID:   uid,
				Email: account.Email,
				Error: err.Error(),
			})
		} else {
			result.Success++
		}
	}

	s.logger.Info("批量启用账户完成: 总数=%d, 成功=%d, 失败=%d", result.Total, result.Success, result.Failed)
	return result, nil
}

// BatchDisableAccounts 批量禁用账户
func (s *accountService) BatchDisableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error) {
	result := &BatchOperationResult{
		Total:       len(uids),
		FailedItems: make([]BatchOperationFailedItem, 0),
	}

	for _, uid := range uids {
		// 获取账户信息（用于错误报告）
		account, err := s.GetByUID(ctx, uid)
		if err != nil {
			result.Failed++
			result.FailedItems = append(result.FailedItems, BatchOperationFailedItem{
				UID:   uid,
				Email: "",
				Error: err.Error(),
			})
			continue
		}

		// 禁用账户
		if err := s.DisableAccount(ctx, uid); err != nil {
			result.Failed++
			result.FailedItems = append(result.FailedItems, BatchOperationFailedItem{
				UID:   uid,
				Email: account.Email,
				Error: err.Error(),
			})
		} else {
			result.Success++
		}
	}

	s.logger.Info("批量禁用账户完成: 总数=%d, 成功=%d, 失败=%d", result.Total, result.Success, result.Failed)
	return result, nil
}

// ClearSyncError 清除同步错误状态
func (s *accountService) ClearSyncError(ctx context.Context, uid string) error {
	// 使用 repository 的 UpdateSyncStatus 方法清除错误
	return s.accountRepo.UpdateSyncStatus(ctx, uid, "", "")
}
