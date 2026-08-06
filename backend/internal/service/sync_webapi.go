package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// isWebhookOnlyAuthData 判断凭证是否为仅 Webhook 推送配置（无 base_url，无法轮询）
func isWebhookOnlyAuthData(authDataJSON string) bool {
	if strings.TrimSpace(authDataJSON) == "" {
		return false
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(authDataJSON), &m); err != nil {
		return false
	}
	baseURL, _ := m["base_url"].(string)
	if strings.TrimSpace(baseURL) != "" {
		return false
	}
	syncMode, _ := m["sync_mode"].(string)
	secret, _ := m["webhook_secret"].(string)
	// 明确 webhook 模式，或只有 webhook_secret 而无 base_url
	if syncMode == model.SyncModeWebhook {
		return true
	}
	return strings.TrimSpace(secret) != ""
}

// doSyncWebAPI 执行 WebAPI 协议的同步逻辑
// 专门处理 WebAPI 类型的邮箱账户同步
func (s *syncService) doSyncWebAPI(ctx context.Context, account *model.EmailAccount, syncLog *model.SyncLog, syncConfig *model.SyncConfig, isFirstSync bool) error {
	s.logger.Info("WebAPI 同步开始: account=%s, provider=%s, isFirstSync=%v", account.UID, account.GetProviderName(), isFirstSync)

	// 获取 Provider 信息（包含 WebAPI 服务类型）
	if account.ProviderRef == nil {
		return fmt.Errorf("WebAPI 账户缺少 Provider 信息: %s", account.UID)
	}

	// 从 Provider 获取 WebAPI 服务类型
	serviceType, err := model.GetWebAPIServiceType(account.ProviderRef)
	if err != nil {
		return fmt.Errorf("获取 WebAPI 服务类型失败: %w", err)
	}

	s.logger.Debug("WebAPI 服务类型: %s", serviceType)

	// 解密认证数据
	authDataJSON := account.EncryptedCredentials
	if s.cryptoService != nil && authDataJSON != "" {
		decrypted, err := s.cryptoService.Decrypt(authDataJSON)
		if err != nil {
			s.logger.Warn("解密 AuthData 失败，尝试使用原始数据: %v", err)
		} else {
			authDataJSON = string(decrypted)
		}
	}

	// 凭证缺失 base_url 且为 webhook 推送类配置时，不应进入轮询适配器创建
	// （兜底：防止脏数据仍漏进调度）
	if isWebhookOnlyAuthData(authDataJSON) {
		s.logger.Debug("WebAPI 凭证为 webhook-only（无 base_url），跳过轮询: account=%s", account.UID)
		return nil
	}

	// 创建 WebAPI 适配器
	webAPIProvider, err := s.webAPIAdapterFactory.CreateAdapter(serviceType, authDataJSON)
	if err != nil {
		return fmt.Errorf("创建 WebAPI 适配器失败: %w", err)
	}

	// 连接到 WebAPI 服务
	if err := webAPIProvider.Connect(ctx); err != nil {
		return s.handleSyncError(ctx, account, fmt.Errorf("WebAPI 连接失败: %w", err))
	}
	defer webAPIProvider.Disconnect()

	// 创建进度追踪器
	tracker := NewProgressTrackerWithNotifier(syncConfig.ProgressInterval, s.notifier)

	// 注册进度追踪器
	s.syncMu.Lock()
	s.activeTrackers[account.UID] = tracker
	s.syncMu.Unlock()

	// 计算同步起始时间
	// WebAPI 首次同步使用全量模式（since 为零值），因为 WebAPI 服务通常数据量较小
	// 且邮件时间可能较早，使用 FirstSyncDays 可能会过滤掉所有邮件
	var since time.Time
	if isFirstSync || account.LastSyncAt == nil {
		// 首次同步：使用零值时间，拉取所有邮件
		s.logger.Info("WebAPI 首次同步，使用全量模式: account=%s", account.UID)
		since = time.Time{}
	} else {
		// 增量同步：从上次同步时间开始，减去 1 小时缓冲
		since = account.LastSyncAt.Add(-1 * time.Hour)
		s.logger.Info("WebAPI 增量同步: account=%s, since=%s", account.UID, since.Format(time.RFC3339))
	}

	// 拉取邮件
	limit := 100 // 默认拉取数量
	if syncConfig.BatchSize > 0 {
		limit = syncConfig.BatchSize
	}

	emails, err := webAPIProvider.FetchEmails(ctx, since, limit)
	if err != nil {
		return fmt.Errorf("WebAPI 拉取邮件失败: %w", err)
	}

	s.logger.Info("WebAPI 拉取邮件完成: account=%s, count=%d", account.UID, len(emails))

	// 处理邮件
	newCount := 0
	updatedCount := 0
	skippedCount := 0

	for _, email := range emails {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 转换为数据库模型
		dbEmail := s.convertWebAPIEmailToModel(email, account)

		// 生成去重标识
		dedupeKey := s.dedupeKeyGen.Generate(dbEmail)
		dbEmail.DedupeKey = dedupeKey

		// 检查是否已存在
		existing, err := s.emailRepo.FindByDedupeKey(ctx, account.UID, dedupeKey)
		if err != nil {
			s.logger.Warn("检查邮件是否存在失败: %v", err)
		}

		if existing != nil {
			// 更新现有邮件
			dbEmail.ID = existing.ID
			if err := s.emailRepo.Update(ctx, dbEmail); err != nil {
				s.logger.Warn("更新邮件失败: %v", err)
				skippedCount++
				continue
			}
			updatedCount++
		} else {
			// 创建新邮件
			if err := s.emailRepo.Create(ctx, dbEmail); err != nil {
				s.logger.Warn("创建邮件失败: %v", err)
				skippedCount++
				continue
			}
			newCount++
		}

		// 更新进度
		tracker.Update(newCount+updatedCount+skippedCount, newCount, updatedCount, skippedCount)
	}

	// 更新同步日志
	syncLog.EmailsNew = int64(newCount)
	syncLog.EmailsUpdated = int64(updatedCount)
	syncLog.EmailsFetched = int64(len(emails))

	// 更新账户的最后同步时间
	now := time.Now()
	account.LastSyncAt = &now
	if err := s.accountRepo.Update(ctx, account); err != nil {
		s.logger.Warn("更新账户最后同步时间失败: %v", err)
	}

	// 同步成功，重置失败计数（所有账号类型）
	if account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			s.logger.Error("重置失败计数失败: account=%s, err=%v", account.UID, resetErr)
		} else {
			s.logger.Info("已重置失败计数: account=%s, 原失败次数=%d", account.UID, account.ConsecutiveAuthFailures)
		}
	}

	s.logger.Info("WebAPI 同步完成: account=%s, new=%d, updated=%d, skipped=%d",
		account.UID, newCount, updatedCount, skippedCount)

	// 通知前端刷新统计/列表缓存
	NotifyEmailCountsMaybeChanged(s.notifier, map[string]any{
		"account_uid": account.UID,
		"new_count":   newCount,
	})

	return nil
}

// convertWebAPIEmailToModel 将 WebAPI 邮件转换为数据库模型
func (s *syncService) convertWebAPIEmailToModel(email *adapter.Email, account *model.EmailAccount) *model.Email {
	dbEmail := &model.Email{
		AccountUID:       account.UID,
		ProviderID:       email.ProviderID,
		MessageID:        email.MessageID,
		Subject:          email.Subject,
		FromAddress:      email.FromAddress,
		FromName:         email.FromName,
		Snippet:          email.Snippet,
		TextBody:         email.TextBody,
		HTMLBody:         email.HTMLBody,
		SentAt:           email.SentAt,
		ReceivedAt:       email.ReceivedAt,
		HasAttachments:   email.HasAttachments,
		AttachmentsCount: email.AttachmentsCount,
	}

	// 处理收件人地址
	if len(email.ToAddresses) > 0 {
		toJSON, _ := json.Marshal(email.ToAddresses)
		dbEmail.ToAddresses = string(toJSON)
	}

	// 处理抄送地址
	if len(email.CcAddresses) > 0 {
		ccJSON, _ := json.Marshal(email.CcAddresses)
		dbEmail.CcAddresses = string(ccJSON)
	}

	// 处理密送地址
	if len(email.BccAddresses) > 0 {
		bccJSON, _ := json.Marshal(email.BccAddresses)
		dbEmail.BccAddresses = string(bccJSON)
	}

	// 处理标签（从源邮箱标签）
	if len(email.SourceLabels) > 0 {
		labelsJSON, _ := json.Marshal(email.SourceLabels)
		dbEmail.Labels = string(labelsJSON)
	}

	return dbEmail
}
