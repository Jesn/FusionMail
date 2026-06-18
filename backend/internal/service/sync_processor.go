package service

import (
	"context"
	"encoding/json"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// processEmailWithDedupe 处理单封邮件（支持 dedupe_key）
// Requirements: 2.2, 3.1, 3.2, 3.3, 3.4
// 返回值: "new", "updated", "skipped"
func (s *syncService) processEmailWithDedupe(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) (string, error) {
	// 生成 dedupe_key (Requirements: 3.1, 3.2)
	dedupeKey := s.dedupeKeyGen.GenerateFromRaw(
		adapterEmail.MessageID,
		adapterEmail.FromAddress,
		adapterEmail.Subject,
		adapterEmail.SentAt,
	)

	// 检查是否在已删除列表中 (Requirements: 2.2)
	if s.deletedKeyRepo != nil {
		isDeleted, err := s.deletedKeyRepo.IsDeleted(ctx, accountUID, dedupeKey)
		if err != nil {
			s.logger.Warn("检查已删除标识失败: account=%s, key=%s, err=%v", accountUID, dedupeKey, err)
		} else if isDeleted {
			// 邮件已被删除，跳过
			return "skipped", nil
		}
	}

	// 先通过 dedupe_key 查找（优先）
	existingEmail, err := s.emailRepo.FindByDedupeKey(ctx, accountUID, dedupeKey)
	if err != nil {
		return "", err
	}

	// 如果 dedupe_key 没找到，再通过 provider_id 查找（兼容旧数据）
	if existingEmail == nil {
		existingEmail, err = s.emailRepo.FindByProviderID(ctx, adapterEmail.ProviderID, accountUID)
		if err != nil {
			return "", err
		}
	}

	if existingEmail != nil {
		// 如果邮件已被软删除，跳过更新 (Requirements: 2.4)
		if existingEmail.DeletedAt.Valid {
			return "skipped", nil
		}

		// 邮件已存在且未删除，更新
		s.updateEmailFromAdapter(existingEmail, adapterEmail, accountUID)
		// 更新 dedupe_key（如果之前没有）
		if existingEmail.DedupeKey == "" {
			existingEmail.DedupeKey = dedupeKey
		}
		if err := s.emailRepo.Update(ctx, existingEmail); err != nil {
			return "", err
		}
		// 应用规则到已存在邮件（更新后）
		if err := s.applyRulesForEmail(ctx, existingEmail); err != nil {
			s.logger.Warn("应用规则失败(更新): email=%d, err=%v", existingEmail.ID, err)
		}
		syncLog.EmailsUpdated++
		return "updated", nil
	}

	// 新邮件，创建 (Requirements: 3.3)
	newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)
	newEmail.DedupeKey = dedupeKey

	// 先保存邮件到数据库，获取正确的 ID
	if err := s.emailRepo.Create(ctx, newEmail); err != nil {
		return "", err
	}

	// 垃圾邮件检测（在邮件保存后执行，确保 email.ID 正确）
	if s.spamDetector != nil {
		spamResult, spamErr := s.spamDetector.DetectSpamSimple(ctx, newEmail)
		if spamErr != nil {
			s.logger.Warn("垃圾邮件检测失败: emailId=%d, msgId=%s, err=%v", newEmail.ID, newEmail.MessageID, spamErr)
		} else if spamResult != nil {
			// 更新邮件的垃圾检测结果
			newEmail.IsSpam = spamResult.IsSpam
			newEmail.SpamScore = float64(spamResult.Score)
			newEmail.SpamConfidence = spamResult.Confidence
			newEmail.SpamReason = spamResult.Reason
			newEmail.SpamDetectedBy = spamResult.DetectedBy
			if spamResult.IsSpam {
				now := time.Now()
				newEmail.SpamDetectedAt = &now
				s.logger.Info("检测到垃圾邮件: emailId=%d, subject=%s, score=%d", newEmail.ID, newEmail.Subject, spamResult.Score)
			}
			// 更新数据库中的垃圾检测结果
			if err := s.emailRepo.Update(ctx, newEmail); err != nil {
				s.logger.Warn("更新垃圾检测结果失败: emailId=%d, err=%v", newEmail.ID, err)
			}
		}
	}

	// 应用规则到新邮件
	if err := s.applyRulesForEmail(ctx, newEmail); err != nil {
		s.logger.Warn("应用规则失败(新建): email=%d, err=%v", newEmail.ID, err)
	}
	syncLog.EmailsNew++
	return "new", nil
}

// processBatchEmails 处理一批邮件
func (s *syncService) processBatchEmails(ctx context.Context, accountUID string, emails []*adapter.Email, syncLog *model.SyncLog) (newCount, updatedCount, failedCount int) {
	newBefore := syncLog.EmailsNew
	updatedBefore := syncLog.EmailsUpdated

	for _, email := range emails {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			return int(syncLog.EmailsNew - newBefore), int(syncLog.EmailsUpdated - updatedBefore), failedCount
		default:
		}

		if err := s.processEmail(ctx, accountUID, email, syncLog); err != nil {
			failedCount++
			continue
		}
	}

	return int(syncLog.EmailsNew - newBefore), int(syncLog.EmailsUpdated - updatedBefore), failedCount
}

// processEmail 处理单封邮件（兼容旧模式，同时支持 dedupe_key）
func (s *syncService) processEmail(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) error {
	// 生成 dedupe_key (Requirements: 3.1, 3.2)
	dedupeKey := s.dedupeKeyGen.GenerateFromRaw(
		adapterEmail.MessageID,
		adapterEmail.FromAddress,
		adapterEmail.Subject,
		adapterEmail.SentAt,
	)

	// 检查是否在已删除列表中 (Requirements: 2.2)
	if s.deletedKeyRepo != nil {
		isDeleted, err := s.deletedKeyRepo.IsDeleted(ctx, accountUID, dedupeKey)
		if err != nil {
			s.logger.Warn("检查已删除标识失败: account=%s, key=%s, err=%v", accountUID, dedupeKey, err)
		} else if isDeleted {
			// 邮件已被删除，跳过
			return nil
		}
	}

	// 先通过 dedupe_key 查找（优先）
	existingEmail, err := s.emailRepo.FindByDedupeKey(ctx, accountUID, dedupeKey)
	if err != nil {
		return err
	}

	// 如果 dedupe_key 没找到，再通过 provider_id 查找（兼容旧数据）
	if existingEmail == nil {
		existingEmail, err = s.emailRepo.FindByProviderID(ctx, adapterEmail.ProviderID, accountUID)
		if err != nil {
			return err
		}
	}

	if existingEmail != nil {
		// 如果邮件已被软删除，跳过更新（不恢复已删除的邮件）
		if existingEmail.DeletedAt.Valid {
			// 邮件已被用户删除，跳过同步
			return nil
		}

		// 邮件已存在且未删除，更新
		s.updateEmailFromAdapter(existingEmail, adapterEmail, accountUID)
		// 更新 dedupe_key（如果之前没有）
		if existingEmail.DedupeKey == "" {
			existingEmail.DedupeKey = dedupeKey
		}
		if err := s.emailRepo.Update(ctx, existingEmail); err != nil {
			return err
		}
		// 应用规则到已存在邮件（更新后）
		if err := s.applyRulesForEmail(ctx, existingEmail); err != nil {
			s.logger.Warn("应用规则失败(更新): email=%d, err=%v", existingEmail.ID, err)
		}
		syncLog.EmailsUpdated++
	} else {
		// 新邮件，创建
		newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)
		newEmail.DedupeKey = dedupeKey // 设置 dedupe_key

		// 先保存邮件到数据库，获取正确的 ID
		if err := s.emailRepo.Create(ctx, newEmail); err != nil {
			return err
		}

		// 垃圾邮件检测（在邮件保存后执行，确保 email.ID 正确）
		if s.spamDetector != nil {
			spamResult, spamErr := s.spamDetector.DetectSpamSimple(ctx, newEmail)
			if spamErr != nil {
				s.logger.Warn("垃圾邮件检测失败: emailId=%d, msgId=%s, err=%v", newEmail.ID, newEmail.MessageID, spamErr)
			} else if spamResult != nil {
				// 更新邮件的垃圾检测结果
				newEmail.IsSpam = spamResult.IsSpam
				newEmail.SpamScore = float64(spamResult.Score)
				newEmail.SpamConfidence = spamResult.Confidence
				newEmail.SpamReason = spamResult.Reason
				newEmail.SpamDetectedBy = spamResult.DetectedBy
				if spamResult.IsSpam {
					now := time.Now()
					newEmail.SpamDetectedAt = &now
					s.logger.Info("检测到垃圾邮件: emailId=%d, subject=%s, score=%d", newEmail.ID, newEmail.Subject, spamResult.Score)
				}
				// 更新数据库中的垃圾检测结果
				if err := s.emailRepo.Update(ctx, newEmail); err != nil {
					s.logger.Warn("更新垃圾检测结果失败: emailId=%d, err=%v", newEmail.ID, err)
				}
			}
		}

		// 应用规则到新邮件
		if err := s.applyRulesForEmail(ctx, newEmail); err != nil {
			s.logger.Warn("应用规则失败(新建): email=%d, err=%v", newEmail.ID, err)
		}
		syncLog.EmailsNew++
	}

	return nil
}

// createEmailFromAdapter 从适配器邮件创建数据库邮件模型
func (s *syncService) createEmailFromAdapter(adapterEmail *adapter.Email, accountUID string) *model.Email {
	return &model.Email{
		ProviderID:       adapterEmail.ProviderID,
		AccountUID:       accountUID,
		MessageID:        adapterEmail.MessageID,
		Subject:          adapterEmail.Subject,
		FromAddress:      adapterEmail.FromAddress,
		FromName:         adapterEmail.FromName,
		ToAddresses:      s.joinAddresses(adapterEmail.ToAddresses),
		CcAddresses:      s.joinAddresses(adapterEmail.CcAddresses),
		BccAddresses:     s.joinAddresses(adapterEmail.BccAddresses),
		ReplyTo:          adapterEmail.ReplyTo,
		TextBody:         adapterEmail.TextBody,
		HTMLBody:         adapterEmail.HTMLBody,
		Snippet:          adapterEmail.Snippet,
		SourceIsRead:     adapterEmail.SourceIsRead,
		SourceLabels:     s.joinLabels(adapterEmail.SourceLabels),
		SourceFolder:     adapterEmail.SourceFolder,
		HasAttachments:   adapterEmail.HasAttachments,
		AttachmentsCount: adapterEmail.AttachmentsCount,
		SentAt:           adapterEmail.SentAt,
		ReceivedAt:       adapterEmail.ReceivedAt,
		SizeBytes:        adapterEmail.SizeBytes,
		ThreadID:         adapterEmail.ThreadID,
		InReplyTo:        adapterEmail.InReplyTo,
		References:       adapterEmail.References,
		SyncedAt:         time.Now(),
	}
}

// updateEmailFromAdapter 从适配器邮件更新数据库邮件模型
func (s *syncService) updateEmailFromAdapter(dbEmail *model.Email, adapterEmail *adapter.Email, accountUID string) {
	// 更新可能变化的字段
	dbEmail.Subject = adapterEmail.Subject
	dbEmail.TextBody = adapterEmail.TextBody
	dbEmail.HTMLBody = adapterEmail.HTMLBody
	dbEmail.Snippet = adapterEmail.Snippet
	dbEmail.SourceIsRead = adapterEmail.SourceIsRead
	dbEmail.SourceLabels = s.joinLabels(adapterEmail.SourceLabels)
	dbEmail.SourceFolder = adapterEmail.SourceFolder
	dbEmail.HasAttachments = adapterEmail.HasAttachments
	dbEmail.AttachmentsCount = adapterEmail.AttachmentsCount
	dbEmail.SizeBytes = adapterEmail.SizeBytes
	dbEmail.SyncedAt = time.Now()
}

// joinAddresses 将地址列表转换为 JSON 字符串
func (s *syncService) joinAddresses(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	data, _ := json.Marshal(addresses)
	return string(data)
}

// joinLabels 将标签列表转换为 JSON 字符串
func (s *syncService) joinLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	data, _ := json.Marshal(labels)
	return string(data)
}

// applyRulesForEmail 在同步阶段对单封邮件应用规则
func (s *syncService) applyRulesForEmail(ctx context.Context, email *model.Email) error {
	if s.ruleService == nil {
		return nil
	}
	return s.ruleService.ApplyRules(ctx, email)
}
