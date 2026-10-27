package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"fusionmail/internal/testutil/pgtest"

	"gorm.io/gorm"
)

// fakeSpamDetector 是垃圾检测替身：返回预设结果或错误。
type fakeSpamDetector struct {
	result *spam.SpamSimpleResult
	err    error
	calls  []*model.Email
}

func (f *fakeSpamDetector) DetectSpamSimple(_ context.Context, email *model.Email) (*spam.SpamSimpleResult, error) {
	f.calls = append(f.calls, email)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// fakeRuleService 只实现同步阶段会调用的 ApplyRules（内嵌 nil 接口 = 作用域断言）。
type fakeRuleService struct {
	RuleService
	applied []int64
	err     error
}

func (f *fakeRuleService) ApplyRules(_ context.Context, email *model.Email) error {
	f.applied = append(f.applied, email.ID)
	return f.err
}

// failingCreateRepo 让指定主题的邮件在 Create 时失败，用于验证批次容错。
type failingCreateRepo struct {
	repository.EmailRepository
	failSubject string
}

func (r *failingCreateRepo) Create(ctx context.Context, email *model.Email) error {
	if email.Subject == r.failSubject {
		return errors.New("模拟写入失败")
	}
	return r.EmailRepository.Create(ctx, email)
}

// syncProcessorTestEnv 是同步处理器用例的接线。
type syncProcessorTestEnv struct {
	processor      *SyncProcessor
	db             *gorm.DB
	emailRepo      repository.EmailRepository
	deletedKeyRepo *repository.DeletedEmailKeyRepository
	spamDetector   *fakeSpamDetector
	ruleService    *fakeRuleService
}

func newSyncProcessorTestEnv(t *testing.T) *syncProcessorTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	emailRepo := repository.NewEmailRepository(db)
	deletedKeyRepo := repository.NewDeletedEmailKeyRepository(db)
	spamDetector := &fakeSpamDetector{}
	ruleService := &fakeRuleService{}

	processor := NewSyncProcessor(
		emailRepo,
		deletedKeyRepo,
		NewDedupeKeyGenerator(),
		ruleService,
		spamDetector,
	)

	return &syncProcessorTestEnv{
		processor:      processor,
		db:             db,
		emailRepo:      emailRepo,
		deletedKeyRepo: deletedKeyRepo,
		spamDetector:   spamDetector,
		ruleService:    ruleService,
	}
}

// processorTestEmail 构造一封适配器邮件。
func processorTestEmail(providerID, subject string) *adapter.Email {
	return &adapter.Email{
		ProviderID:   providerID,
		MessageID:    providerID + "@example.com",
		Subject:      subject,
		FromAddress:  "sender@example.com",
		FromName:     "Sender",
		ToAddresses:  []string{"me@example.com"},
		CcAddresses:  []string{"cc@example.com"},
		TextBody:     subject + " 正文",
		SourceLabels: []string{"inbox"},
		SentAt:       time.Now().Add(-time.Hour),
		ReceivedAt:   time.Now().Add(-time.Hour),
		SizeBytes:    2048,
	}
}

func newProcessorSyncLog() *model.SyncLog {
	return &model.SyncLog{AccountUID: "acct-processor", SyncType: "manual", Status: "running", StartedAt: time.Now()}
}

// TestProcessEmailCreatesNewEmailWithMappedFields 覆盖新建路径：
// 去重标识、地址 JSON、标签、附件元数据都要落库，计数加在 EmailsNew。
func TestProcessEmailCreatesNewEmailWithMappedFields(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()

	email := processorTestEmail("p-new", "新邮件")
	email.Attachments = []adapter.Attachment{
		{Filename: "报告.pdf", ContentType: "application/pdf", SizeBytes: 1024},
		{Filename: "logo.png", ContentType: "image/png", SizeBytes: 32, IsInline: true, ContentID: "cid-1"},
	}
	email.HasAttachments = true
	email.AttachmentsCount = 2

	syncLog := newProcessorSyncLog()
	result, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog)
	if err != nil {
		t.Fatalf("处理新邮件失败: %v", err)
	}
	if result != "new" {
		t.Fatalf("处理结果 = %q，期望 new", result)
	}
	if syncLog.EmailsNew != 1 || syncLog.EmailsUpdated != 0 {
		t.Errorf("计数 = new %d / updated %d，期望 1/0", syncLog.EmailsNew, syncLog.EmailsUpdated)
	}

	stored, err := env.emailRepo.FindByDedupeKey(ctx, "acct-processor", env.processor.dedupeKeyGen.GenerateFromRaw(
		email.MessageID, email.FromAddress, email.Subject, email.SentAt))
	if err != nil {
		t.Fatalf("查询邮件失败: %v", err)
	}
	if stored == nil {
		t.Fatal("新邮件未落库")
	}
	if stored.Subject != "新邮件" || stored.AccountUID != "acct-processor" {
		t.Errorf("邮件字段映射错误: %+v", stored)
	}
	if stored.ToAddresses == "" || stored.CcAddresses == "" || stored.SourceLabels == "" {
		t.Errorf("地址/标签字段被丢弃: to=%q cc=%q labels=%q", stored.ToAddresses, stored.CcAddresses, stored.SourceLabels)
	}
	if stored.SyncedAt.IsZero() {
		t.Error("SyncedAt 未记录")
	}

	var attachments []model.EmailAttachment
	if err := env.db.Where("email_id = ?", stored.ID).Order("id asc").Find(&attachments).Error; err != nil {
		t.Fatalf("读取附件失败: %v", err)
	}
	if len(attachments) != 2 {
		t.Fatalf("附件行数 = %d，期望 2", len(attachments))
	}
	for _, att := range attachments {
		if att.StorageType != "external" || att.StoragePath != "" {
			t.Errorf("附件存储语义错误（应只存元数据）: %+v", att)
		}
	}
	if !attachments[1].IsInline || attachments[1].ContentID != "cid-1" {
		t.Errorf("内联附件字段丢失: %+v", attachments[1])
	}
}

// TestProcessEmailUpdatesExistingWithoutDuplicating 覆盖更新路径：
// 同一封邮件再次同步只更新不新增，且回填缺失的去重标识。
func TestProcessEmailUpdatesExistingWithoutDuplicating(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()

	email := processorTestEmail("p-update", "原始主题")
	syncLog := newProcessorSyncLog()
	if _, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog); err != nil {
		t.Fatalf("首次处理失败: %v", err)
	}

	// 清掉去重标识，验证「兼容旧数据」路径：靠 provider_id 找到并回填
	if err := env.db.Model(&model.Email{}).Where("provider_id = ?", "p-update").
		UpdateColumns(map[string]interface{}{"dedupe_key": ""}).Error; err != nil {
		t.Fatalf("清空去重标识失败: %v", err)
	}

	updated := processorTestEmail("p-update", "更新后的主题")
	updated.TextBody = "更新后的正文"
	result, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", updated, syncLog)
	if err != nil {
		t.Fatalf("二次处理失败: %v", err)
	}
	if result != "updated" {
		t.Fatalf("处理结果 = %q，期望 updated", result)
	}
	if syncLog.EmailsNew != 1 || syncLog.EmailsUpdated != 1 {
		t.Errorf("计数 = new %d / updated %d，期望 1/1", syncLog.EmailsNew, syncLog.EmailsUpdated)
	}

	var emails []model.Email
	if err := env.db.Find(&emails).Error; err != nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if len(emails) != 1 {
		t.Fatalf("邮件行数 = %d，期望 1（去重失效）", len(emails))
	}
	if emails[0].Subject != "更新后的主题" || emails[0].TextBody != "更新后的正文" {
		t.Errorf("更新未生效: %+v", emails[0])
	}
	if emails[0].DedupeKey == "" {
		t.Error("更新时未回填缺失的去重标识")
	}
}

// TestProcessEmailSkipsDeletedKey 覆盖「已删除邮件不复活」：
// dedupe_key 在已删除列表中时直接跳过，不得重新落库。
func TestProcessEmailSkipsDeletedKey(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()

	email := processorTestEmail("p-deleted", "已删除邮件")
	dedupeKey := env.processor.dedupeKeyGen.GenerateFromRaw(
		email.MessageID, email.FromAddress, email.Subject, email.SentAt)
	if err := env.deletedKeyRepo.CreateIfNotExists(ctx, "acct-processor", dedupeKey); err != nil {
		t.Fatalf("写入已删除标识��败: %v", err)
	}

	syncLog := newProcessorSyncLog()
	result, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog)
	if err != nil {
		t.Fatalf("处理失败: %v", err)
	}
	if result != "skipped" {
		t.Fatalf("处理结果 = %q，期望 skipped", result)
	}
	if syncLog.EmailsNew != 0 || syncLog.EmailsUpdated != 0 {
		t.Errorf("跳过却计数: new %d / updated %d", syncLog.EmailsNew, syncLog.EmailsUpdated)
	}

	var count int64
	if err := env.db.Unscoped().Model(&model.Email{}).Count(&count).Error; err != nil {
		t.Fatalf("统计邮件失败: %v", err)
	}
	if count != 0 {
		t.Errorf("已删除邮件被重新创建: %d 行", count)
	}
}

// TestProcessEmailSkipsSoftDeletedEmail 覆盖「回收站里的邮件不复活」：
// 邮件已被软删除时同步必须跳过，且不能把它恢复出来。
func TestProcessEmailSkipsSoftDeletedEmail(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()

	email := processorTestEmail("p-trash", "回收站邮件")
	syncLog := newProcessorSyncLog()
	if _, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog); err != nil {
		t.Fatalf("首次处理失败: %v", err)
	}

	stored, err := env.emailRepo.FindByProviderID(ctx, "p-trash", "acct-processor")
	if err != nil || stored == nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if err := env.emailRepo.Delete(ctx, stored.ID); err != nil {
		t.Fatalf("软删除邮件失败: %v", err)
	}

	result, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog)
	if err != nil {
		t.Fatalf("再次处理失败: %v", err)
	}
	if result != "skipped" {
		t.Fatalf("处理结果 = %q，期望 skipped", result)
	}

	var restored model.Email
	if err := env.db.Unscoped().Where("id = ?", stored.ID).First(&restored).Error; err != nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if !restored.DeletedAt.Valid {
		t.Error("回收站里的邮件被同步恢复了")
	}
}

// TestProcessEmailAppliesSpamDetection 覆盖垃圾检测副作用：
// 判定为垃圾时必须写入评分/原因/检测时���，非垃圾时不写检测时间。
func TestProcessEmailAppliesSpamDetection(t *testing.T) {
	cases := []struct {
		name         string
		result       *spam.SpamSimpleResult
		wantSpam     bool
		wantDetected bool
	}{
		{
			name:         "判定为垃圾",
			result:       &spam.SpamSimpleResult{IsSpam: true, Score: 75, Confidence: 0.9, Reason: "命中关键词", DetectedBy: "rule"},
			wantSpam:     true,
			wantDetected: true,
		},
		{
			name:         "判定为正常",
			result:       &spam.SpamSimpleResult{IsSpam: false, Score: 5, Confidence: 0.8, Reason: "未命中", DetectedBy: "bayesian"},
			wantSpam:     false,
			wantDetected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newSyncProcessorTestEnv(t)
			ctx := context.Background()
			env.spamDetector.result = c.result

			email := processorTestEmail("p-spam", "垃圾检测")
			syncLog := newProcessorSyncLog()
			if _, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog); err != nil {
				t.Fatalf("处理失败: %v", err)
			}

			if len(env.spamDetector.calls) != 1 {
				t.Fatalf("检测调用次数 = %d，期望 1", len(env.spamDetector.calls))
			}
			if env.spamDetector.calls[0].ID == 0 {
				t.Error("检测时邮件尚未落库（ID 为 0），垃圾规则无法关联邮件")
			}

			stored, err := env.emailRepo.FindByProviderID(ctx, "p-spam", "acct-processor")
			if err != nil || stored == nil {
				t.Fatalf("读取邮件失败: %v", err)
			}
			if stored.IsSpam != c.wantSpam {
				t.Errorf("IsSpam = %v，期望 %v", stored.IsSpam, c.wantSpam)
			}
			if stored.SpamScore != float64(c.result.Score) || stored.SpamReason != c.result.Reason {
				t.Errorf("垃圾字段未落库: score=%v reason=%q", stored.SpamScore, stored.SpamReason)
			}
			if stored.SpamDetectedBy != c.result.DetectedBy {
				t.Errorf("SpamDetectedBy = %q，期望 %q", stored.SpamDetectedBy, c.result.DetectedBy)
			}
			if c.wantDetected && stored.SpamDetectedAt == nil {
				t.Error("判为垃圾却未记录检测时间（垃圾清理依赖它）")
			}
			if !c.wantDetected && stored.SpamDetectedAt != nil {
				t.Error("非垃圾邮件却记录了检测时间")
			}
		})
	}
}

// TestProcessEmailToleratesSpamDetectorFailure 覆盖检测失败：
// 检测报错不能丢掉邮件本身，只是不写垃圾字段。
func TestProcessEmailToleratesSpamDetectorFailure(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()
	env.spamDetector.err = errors.New("检测服务不可用")

	email := processorTestEmail("p-detect-fail", "检测失败")
	syncLog := newProcessorSyncLog()
	result, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog)
	if err != nil {
		t.Fatalf("检测失败不应导致同步失败: %v", err)
	}
	if result != "new" {
		t.Fatalf("处理结果 = %q，期望 new", result)
	}

	stored, err := env.emailRepo.FindByProviderID(ctx, "p-detect-fail", "acct-processor")
	if err != nil || stored == nil {
		t.Fatalf("邮件未落库: %v", err)
	}
	if stored.IsSpam {
		t.Error("检测失败却把邮件标成垃圾")
	}
}

// TestProcessEmailAppliesRules 覆盖规则应用：新建与更新都要跑一次规则，
// 规则报错不能中断同步。
func TestProcessEmailAppliesRules(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()

	email := processorTestEmail("p-rules", "规则")
	syncLog := newProcessorSyncLog()
	if _, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog); err != nil {
		t.Fatalf("处理失败: %v", err)
	}
	if len(env.ruleService.applied) != 1 {
		t.Fatalf("新建时应用规则次数 = %d，期望 1", len(env.ruleService.applied))
	}
	if env.ruleService.applied[0] == 0 {
		t.Error("应用规则时邮件 ID 为 0")
	}

	if _, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", email, syncLog); err != nil {
		t.Fatalf("二次处理失败: %v", err)
	}
	if len(env.ruleService.applied) != 2 {
		t.Errorf("更新时应用规则次数 = %d，期望 2", len(env.ruleService.applied))
	}

	// 规则报错不得让同步失败
	env.ruleService.err = errors.New("规则引擎炸了")
	if _, err := env.processor.processEmailWithDedupe(ctx, "acct-processor", processorTestEmail("p-rules-2", "规则2"), syncLog); err != nil {
		t.Fatalf("规则报错不应导致同步失败: %v", err)
	}
}

// TestProcessEmailWithoutOptionalDependencies 覆盖依赖缺省：
// 未配置垃圾检测/规则/已删除标识仓库时仍能正常落库。
func TestProcessEmailWithoutOptionalDependencies(t *testing.T) {
	db := pgtest.DB(t)
	pgtest.Truncate(t)
	ctx := context.Background()

	processor := NewSyncProcessor(repository.NewEmailRepository(db), nil, NewDedupeKeyGenerator(), nil, nil)
	syncLog := newProcessorSyncLog()

	result, err := processor.processEmailWithDedupe(ctx, "acct-processor", processorTestEmail("p-min", "最小依赖"), syncLog)
	if err != nil {
		t.Fatalf("处理失败: %v", err)
	}
	if result != "new" || syncLog.EmailsNew != 1 {
		t.Errorf("结果 = %q / new = %d，期望 new/1", result, syncLog.EmailsNew)
	}
}

// TestProcessBatchEmailsCountsAndContinuesOnFailure 覆盖批次容错：
// 单封失败只计失败数，其余邮件继续处理。
func TestProcessBatchEmailsCountsAndContinuesOnFailure(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()

	processor := NewSyncProcessor(
		&failingCreateRepo{EmailRepository: env.emailRepo, failSubject: "坏邮件"},
		env.deletedKeyRepo,
		NewDedupeKeyGenerator(),
		nil,
		nil,
	)

	emails := []*adapter.Email{
		processorTestEmail("p-batch-1", "好邮件1"),
		processorTestEmail("p-batch-2", "坏邮件"),
		processorTestEmail("p-batch-3", "好邮件3"),
	}
	syncLog := newProcessorSyncLog()

	newCount, updatedCount, failedCount := processor.processBatchEmails(ctx, "acct-processor", emails, syncLog)
	if newCount != 2 || updatedCount != 0 || failedCount != 1 {
		t.Fatalf("批次结果 = new %d / updated %d / failed %d，期望 2/0/1", newCount, updatedCount, failedCount)
	}

	var count int64
	if err := env.db.Model(&model.Email{}).Count(&count).Error; err != nil {
		t.Fatalf("统计邮件失败: %v", err)
	}
	if count != 2 {
		t.Errorf("落库邮件数 = %d，期望 2", count)
	}
}

// TestProcessBatchEmailsStopsOnContextCancel 覆盖取消语义：
// context 已取消时不再处理后续邮件（避免停机时继续写库）。
func TestProcessBatchEmailsStopsOnContextCancel(t *testing.T) {
	env := newSyncProcessorTestEnv(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	emails := []*adapter.Email{
		processorTestEmail("p-cancel-1", "取消1"),
		processorTestEmail("p-cancel-2", "取消2"),
	}
	syncLog := newProcessorSyncLog()

	newCount, updatedCount, failedCount := env.processor.processBatchEmails(ctx, "acct-processor", emails, syncLog)
	if newCount != 0 || updatedCount != 0 || failedCount != 0 {
		t.Errorf("取消后仍处理了邮件: new %d / updated %d / failed %d", newCount, updatedCount, failedCount)
	}

	var count int64
	if err := env.db.Model(&model.Email{}).Count(&count).Error; err != nil {
		t.Fatalf("统计邮件失败: %v", err)
	}
	if count != 0 {
		t.Errorf("取消后仍落库 %d 封邮件", count)
	}
}

// TestProcessEmailLegacyPathMirrorsDedupePath 覆盖旧入口 processEmail：
// 与 processEmailWithDedupe 共享去重语义（不重复落库、跳过已删除）。
func TestProcessEmailLegacyPathMirrorsDedupePath(t *testing.T) {
	env := newSyncProcessorTestEnv(t)
	ctx := context.Background()

	email := processorTestEmail("p-legacy", "旧入口")
	syncLog := newProcessorSyncLog()

	if err := env.processor.processEmail(ctx, "acct-processor", email, syncLog); err != nil {
		t.Fatalf("旧入口处理失败: %v", err)
	}
	if syncLog.EmailsNew != 1 {
		t.Fatalf("EmailsNew = %d，期望 1", syncLog.EmailsNew)
	}

	if err := env.processor.processEmail(ctx, "acct-processor", email, syncLog); err != nil {
		t.Fatalf("旧入口二次处理失败: %v", err)
	}
	if syncLog.EmailsUpdated != 1 {
		t.Errorf("EmailsUpdated = %d，期望 1", syncLog.EmailsUpdated)
	}

	var count int64
	if err := env.db.Model(&model.Email{}).Count(&count).Error; err != nil {
		t.Fatalf("统计邮件失败: %v", err)
	}
	if count != 1 {
		t.Errorf("邮件行数 = %d，期望 1", count)
	}

	// 已删除标识命中时静默跳过
	dedupeKey := env.processor.dedupeKeyGen.GenerateFromRaw(
		email.MessageID, email.FromAddress, email.Subject, email.SentAt)
	if err := env.deletedKeyRepo.CreateIfNotExists(ctx, "acct-processor", dedupeKey); err != nil {
		t.Fatalf("写入已删除标识失败: %v", err)
	}
	if err := env.processor.processEmail(ctx, "acct-processor", email, syncLog); err != nil {
		t.Fatalf("命中已删除标识时不应报错: %v", err)
	}
	if syncLog.EmailsUpdated != 1 {
		t.Errorf("命中已删除标识后计数被改动: updated = %d", syncLog.EmailsUpdated)
	}
}

// TestBuildAttachmentRows 覆盖附件元数据转换：空列表返回 nil，字段逐一映射。
func TestBuildAttachmentRows(t *testing.T) {
	if rows := buildAttachmentRows(&adapter.Email{}); rows != nil {
		t.Errorf("无附件时应返回 nil，实际 %+v", rows)
	}

	rows := buildAttachmentRows(&adapter.Email{Attachments: []adapter.Attachment{
		{Filename: "a.txt", ContentType: "text/plain", SizeBytes: 10},
	}})
	if len(rows) != 1 {
		t.Fatalf("附件行数 = %d，期望 1", len(rows))
	}
	if rows[0].Filename != "a.txt" || rows[0].ContentType != "text/plain" || rows[0].SizeBytes != 10 {
		t.Errorf("附件字段映射错误: %+v", rows[0])
	}
	if rows[0].StorageType != "external" {
		t.Errorf("StorageType = %q，期望 external", rows[0].StorageType)
	}
}
