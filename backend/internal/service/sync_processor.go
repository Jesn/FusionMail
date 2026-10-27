package service

import (
	"context"
	"encoding/json"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// SyncProcessor 负责把适配器拉回的邮件写入本地库：去重、新建、更新、打标签、跑规则。
//
// 为什么从 syncService 里拆出来：原类型跨 6 个文件、1782 行、30 个方法、15 个字段。
// 本文件是依赖图上的**叶子**——它只调用自己的方法，不反向依赖调度器或执行器，
// 因此可以独立构造与测试。
type SyncProcessor struct {
	emailRepo      repository.EmailRepository
	deletedKeyRepo *repository.DeletedEmailKeyRepository
	dedupeKeyGen   *DedupeKeyGenerator
	ruleService    RuleService
	spamDetector   SpamDetectorInterface
	logger         *logger.Logger
}

// NewSyncProcessor 创建邮件处理服务。
func NewSyncProcessor(
	emailRepo repository.EmailRepository,
	deletedKeyRepo *repository.DeletedEmailKeyRepository,
	dedupeKeyGen *DedupeKeyGenerator,
	ruleService RuleService,
	spamDetector SpamDetectorInterface,
) *SyncProcessor {
	return &SyncProcessor{
		emailRepo:      emailRepo,
		deletedKeyRepo: deletedKeyRepo,
		dedupeKeyGen:   dedupeKeyGen,
		ruleService:    ruleService,
		spamDetector:   spamDetector,
		logger:         logger.NewWithModule("SyncProcessor"),
	}
}

// processEmailWithDedupe 处理单封邮件（支持 dedupe_key）
// Requirements: 2.2, 3.1, 3.2, 3.3, 3.4
// 返回值: "new", "updated", "skipped"
func (s *SyncProcessor) processEmailWithDedupe(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) (string, error) {
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
		// 附件元数据整体替换（Update 不级联关联）
		if err := s.emailRepo.ReplaceAttachments(ctx, existingEmail.ID, buildAttachmentRows(adapterEmail)); err != nil {
			s.logger.Warn("附件元数据更新失败: emailId=%d, err=%v", existingEmail.ID, err)
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
func (s *SyncProcessor) processBatchEmails(ctx context.Context, accountUID string, emails []*adapter.Email, syncLog *model.SyncLog) (newCount, updatedCount, failedCount int) {
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
func (s *SyncProcessor) processEmail(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) error {
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
		// 附件元数据整体替换（Update 不级联关联）
		if err := s.emailRepo.ReplaceAttachments(ctx, existingEmail.ID, buildAttachmentRows(adapterEmail)); err != nil {
			s.logger.Warn("附件元数据更新失败: emailId=%d, err=%v", existingEmail.ID, err)
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

// buildAttachmentRows 把适配器附件元数据转换为落库行。
// 存储语义：同步只持久化元数据 —— StorageType="external"（内容在服务商侧，
// ExternalRef 留作未来按需拉取），StoragePath 留空（本地未存储）。
// 不落附件内容：Fly.io 临时文件系统上本地盘存储会随重启丢失，DB 存字节
// 则膨胀备份；按需下载是适配器层的独立课题。
func buildAttachmentRows(adapterEmail *adapter.Email) []model.EmailAttachment {
	if len(adapterEmail.Attachments) == 0 {
		return nil
	}
	atts := make([]model.EmailAttachment, 0, len(adapterEmail.Attachments))
	for _, a := range adapterEmail.Attachments {
		atts = append(atts, model.EmailAttachment{
			Filename:    a.Filename,
			ContentType: a.ContentType,
			SizeBytes:   a.SizeBytes,
			StorageType: "external",
			StoragePath: "",
			IsInline:    a.IsInline,
			ContentID:   a.ContentID,
		})
	}
	return atts
}

// createEmailFromAdapter 从适配器邮件创建数据库邮件模型
func (s *SyncProcessor) createEmailFromAdapter(adapterEmail *adapter.Email, accountUID string) *model.Email {
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
		// 附件元数据随 Create 级联落库（GORM 自动填 email_id），
		// 垃圾检测的附件类规则与预过滤层也读这份数据
		Attachments: buildAttachmentRows(adapterEmail),
		SentAt:      adapterEmail.SentAt,
		ReceivedAt:  adapterEmail.ReceivedAt,
		SizeBytes:   adapterEmail.SizeBytes,
		ThreadID:    adapterEmail.ThreadID,
		InReplyTo:   adapterEmail.InReplyTo,
		References:  adapterEmail.References,
		SyncedAt:    time.Now(),
	}
}

// updateEmailFromAdapter 从适配器邮件更新数据库邮件模型
func (s *SyncProcessor) updateEmailFromAdapter(dbEmail *model.Email, adapterEmail *adapter.Email, accountUID string) {
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
func (s *SyncProcessor) joinAddresses(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	data, _ := json.Marshal(addresses)
	return string(data)
}

// joinLabels 将标签列表转换为 JSON 字符串
func (s *SyncProcessor) joinLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	data, _ := json.Marshal(labels)
	return string(data)
}

// applyRulesForEmail 在同步阶段对单封邮件应用规则
func (s *SyncProcessor) applyRulesForEmail(ctx context.Context, email *model.Email) error {
	if s.ruleService == nil {
		return nil
	}
	return s.ruleService.ApplyRules(ctx, email)
}
