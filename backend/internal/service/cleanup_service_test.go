package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"

	"gorm.io/gorm"
)

// stubAccountService 只实现清理链路会走到的账户方法（内嵌 nil 接口 = 作用域断言）。
type stubAccountService struct {
	AccountService
	cleaned int
	err     error
	days    []int
}

func (s *stubAccountService) CleanupTrash(_ context.Context, days int) (int, error) {
	s.days = append(s.days, days)
	return s.cleaned, s.err
}

// stubEncryptor 满足 SettingService / CredentialResolver 的加密器依赖。
// 加解密互为逆操作（前缀标记），这样凭据解析用例能验证「解出来的就是明文」。
type stubEncryptor struct{}

func (stubEncryptor) Encrypt(plaintext string) (string, error) { return "enc:" + plaintext, nil }

func (stubEncryptor) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

// cleanupTestEnv 是清理用例的接线：真实 PostgreSQL 仓库 + 替身账户服务。
type cleanupTestEnv struct {
	svc            *CleanupService
	db             *gorm.DB
	accountService *stubAccountService
	emailRepo      repository.EmailRepository
	syncLogRepo    repository.SyncLogRepository
	spamLogRepo    repository.SpamDetectionLogRepository
	deletedKeyRepo *repository.DeletedEmailKeyRepository
	ruleMatchRepo  repository.RuleMatchLogRepository
}

func newCleanupTestEnv(t *testing.T) *cleanupTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	emailRepo := repository.NewEmailRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)
	spamLogRepo := repository.NewSpamDetectionLogRepository(db)
	deletedKeyRepo := repository.NewDeletedEmailKeyRepository(db)
	ruleMatchRepo := repository.NewRuleMatchLogRepository(db)

	accountService := &stubAccountService{}
	settingService := NewSettingService(repository.NewSettingRepository(db), nil, stubEncryptor{})

	svc := NewCleanupService(
		accountService,
		settingService,
		emailRepo,
		syncLogRepo,
		spamLogRepo,
		deletedKeyRepo,
		ruleMatchRepo,
	)

	return &cleanupTestEnv{
		svc:            svc,
		db:             db,
		accountService: accountService,
		emailRepo:      emailRepo,
		syncLogRepo:    syncLogRepo,
		spamLogRepo:    spamLogRepo,
		deletedKeyRepo: deletedKeyRepo,
		ruleMatchRepo:  ruleMatchRepo,
	}
}

// seedSetting 写入系统级配置，并在用例结束时删除（settings 是种子表，不参与 Truncate）。
func seedSetting(t *testing.T, db *gorm.DB, category, key, value string) {
	t.Helper()

	setting := &model.Setting{
		Category:  category,
		Key:       key,
		Value:     value,
		ValueType: "number",
	}
	if err := db.Create(setting).Error; err != nil {
		t.Fatalf("写入配置 %s/%s 失败: %v", category, key, err)
	}
	t.Cleanup(func() {
		db.Unscoped().Delete(&model.Setting{}, setting.ID)
	})
}

// seedCleanupEmail 写入一封邮件，可指定回收站状态与时间戳。
// 时间戳用 UpdateColumns 写入：它跳过 GORM 的自动时间戳，才能造出「很久以前删除」的数据。
func seedCleanupEmail(t *testing.T, db *gorm.DB, providerID string, opts emailSeedOptions) *model.Email {
	t.Helper()

	email := &model.Email{
		ProviderID:  providerID,
		AccountUID:  "acct-cleanup",
		MessageID:   providerID + "@example.com",
		Subject:     "清理用例 " + providerID,
		FromAddress: "sender@example.com",
		ToAddress:   "me@example.com",
		TextBody:    "正文",
		DedupeKey:   "dedupe-" + providerID,
		IsDeleted:   opts.isDeleted,
		IsSpam:      opts.isSpam,
		SentAt:      time.Now(),
		ReceivedAt:  time.Now(),
	}
	if opts.spamDetectedAt != nil {
		email.SpamDetectedAt = opts.spamDetectedAt
	}
	if err := db.Create(email).Error; err != nil {
		t.Fatalf("写入邮件 fixture 失败: %v", err)
	}

	if opts.updatedAt != nil {
		if err := db.Model(&model.Email{}).Where("id = ?", email.ID).
			UpdateColumns(map[string]interface{}{"updated_at": *opts.updatedAt}).Error; err != nil {
			t.Fatalf("改写 updated_at 失败: %v", err)
		}
	}
	return email
}

type emailSeedOptions struct {
	isDeleted      bool
	isSpam         bool
	updatedAt      *time.Time
	spamDetectedAt *time.Time
}

// reloadCleanupEmail 从数据库重新读取邮件（避免断言到内存里被就地修改的对象）。
func reloadCleanupEmail(t *testing.T, env *cleanupTestEnv, id int64) *model.Email {
	t.Helper()

	email, err := env.emailRepo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if email == nil {
		t.Fatalf("邮件 %d 不存在", id)
	}
	return email
}

// emailExists 判断邮件是否还在库里（含软删除行）。
func emailExists(t *testing.T, db *gorm.DB, id int64) bool {
	t.Helper()

	var count int64
	if err := db.Unscoped().Model(&model.Email{}).Where("id = ?", id).Count(&count).Error; err != nil {
		t.Fatalf("统计邮件失败: %v", err)
	}
	return count > 0
}

func daysAgo(days int) time.Time {
	return time.Now().AddDate(0, 0, -days)
}

// TestCleanupDeletedEmailsPhysicallyRemovesOldTrash 覆盖回收站物理清理：
// 超过保留期的回收站邮件必须从库里消失（不是软删除），未到期与正常邮件不受影响，
// 且物理删除前必须留下 dedupe_key 防止重新同步回来。
func TestCleanupDeletedEmailsPhysicallyRemovesOldTrash(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	old := daysAgo(30)
	oldTrash := seedCleanupEmail(t, env.db, "old-trash", emailSeedOptions{isDeleted: true, updatedAt: &old})
	recent := time.Now()
	recentTrash := seedCleanupEmail(t, env.db, "recent-trash", emailSeedOptions{isDeleted: true, updatedAt: &recent})
	normal := seedCleanupEmail(t, env.db, "normal", emailSeedOptions{})

	cleaned, err := env.svc.cleanupDeletedEmails(ctx, 7)
	if err != nil {
		t.Fatalf("清理回收站失败: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("清理数量 = %d，期望 1", cleaned)
	}

	if emailExists(t, env.db, oldTrash.ID) {
		t.Error("超过保留期的回收站邮件仍留在库中（清理未真正生效）")
	}
	if !emailExists(t, env.db, recentTrash.ID) {
		t.Error("未到期的回收站邮件被误删")
	}
	if !emailExists(t, env.db, normal.ID) {
		t.Error("正常邮件被误删")
	}

	isDeleted, err := env.deletedKeyRepo.IsDeleted(ctx, oldTrash.AccountUID, oldTrash.DedupeKey)
	if err != nil {
		t.Fatalf("查询已删除标识失败: %v", err)
	}
	if !isDeleted {
		t.Error("物理删除前未记录 dedupe_key，邮件可能被重新同步回来")
	}
}

// TestCleanupDeletedEmailsWithoutDedupeKeyGeneratesOne 覆盖邮件缺少 dedupe_key 的情况：
// 必须现算一个再记录，否则这类邮件删掉后仍会被同步回来。
func TestCleanupDeletedEmailsWithoutDedupeKeyGeneratesOne(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	old := daysAgo(30)
	email := seedCleanupEmail(t, env.db, "no-dedupe", emailSeedOptions{isDeleted: true, updatedAt: &old})
	if err := env.db.Model(&model.Email{}).Where("id = ?", email.ID).
		UpdateColumns(map[string]interface{}{"dedupe_key": ""}).Error; err != nil {
		t.Fatalf("清空 dedupe_key 失败: %v", err)
	}

	cleaned, err := env.svc.cleanupDeletedEmails(ctx, 7)
	if err != nil {
		t.Fatalf("清理回收站失败: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("清理数量 = %d，期望 1", cleaned)
	}

	generated := env.svc.dedupeKeyGen.GenerateFromRaw(
		email.MessageID, email.FromAddress, email.Subject, email.SentAt)
	isDeleted, err := env.deletedKeyRepo.IsDeleted(ctx, email.AccountUID, generated)
	if err != nil {
		t.Fatalf("查询已删除标识失败: %v", err)
	}
	if !isDeleted {
		t.Error("缺少 dedupe_key 的邮件未被记录，删除后会被重新同步")
	}
}

// TestCleanupDeletedEmailsKeepsEverythingWhenNothingIsDue 覆盖边界：
// 没有到期邮件时返回 0 且不报错。
func TestCleanupDeletedEmailsKeepsEverythingWhenNothingIsDue(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	recent := time.Now()
	trash := seedCleanupEmail(t, env.db, "fresh-trash", emailSeedOptions{isDeleted: true, updatedAt: &recent})

	cleaned, err := env.svc.cleanupDeletedEmails(ctx, 7)
	if err != nil {
		t.Fatalf("清理回收站失败: %v", err)
	}
	if cleaned != 0 {
		t.Errorf("清理数量 = %d，期望 0", cleaned)
	}
	if !emailExists(t, env.db, trash.ID) {
		t.Error("未到期的回收站邮件被删除")
	}
}

// TestManualCleanupUsesConfiguredRetentionDays 覆盖手动清理：
// 保留天数来自配置，且账户清理与��件清理的结果都要如实返回。
func TestManualCleanupUsesConfiguredRetentionDays(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "trash_auto_cleanup_days", "7")
	env.accountService.cleaned = 3

	old := daysAgo(30)
	trash := seedCleanupEmail(t, env.db, "manual-old", emailSeedOptions{isDeleted: true, updatedAt: &old})

	accounts, emails, err := env.svc.ManualCleanup(ctx)
	if err != nil {
		t.Fatalf("手动清理失败: %v", err)
	}
	if accounts != 3 || emails != 1 {
		t.Errorf("清理结果 = 账户 %d / 邮件 %d，期望 3/1", accounts, emails)
	}
	if len(env.accountService.days) != 1 || env.accountService.days[0] != 7 {
		t.Errorf("传给账户清理的天数 = %v，期望 [7]", env.accountService.days)
	}
	if emailExists(t, env.db, trash.ID) {
		t.Error("手动清理未物理删除到期邮件")
	}
}

// TestManualCleanupDisabledByNegativeSetting 覆盖 days=-1：
// 表示关闭自动清理，必须一封都不删且不报错。
func TestManualCleanupDisabledByNegativeSetting(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "trash_auto_cleanup_days", "-1")
	old := daysAgo(30)
	trash := seedCleanupEmail(t, env.db, "disabled", emailSeedOptions{isDeleted: true, updatedAt: &old})

	accounts, emails, err := env.svc.ManualCleanup(ctx)
	if err != nil {
		t.Fatalf("关闭清理时不应报错: %v", err)
	}
	if accounts != 0 || emails != 0 {
		t.Errorf("关闭清理却清理了 账户 %d / 邮件 %d", accounts, emails)
	}
	if len(env.accountService.days) != 0 {
		t.Error("关闭清理时仍调用了账户清理")
	}
	if !emailExists(t, env.db, trash.ID) {
		t.Error("关闭清理却删除了邮件")
	}
}

// TestManualCleanupRejectsInvalidSetting 覆盖配置非法：
// 必须把错误暴露出来，而不是悄悄按默认值删数据。
func TestManualCleanupRejectsInvalidSetting(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "trash_auto_cleanup_days", "not-a-number")

	if _, _, err := env.svc.ManualCleanup(ctx); err == nil {
		t.Fatal("配置非法时应返回 error")
	}
}

// TestManualCleanupPropagatesAccountCleanupError 覆盖账户清理失败：
// 错误必须上抛，不能只清理邮件却报告成功。
func TestManualCleanupPropagatesAccountCleanupError(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "trash_auto_cleanup_days", "7")
	env.accountService.err = errors.New("账户清理炸了")

	if _, _, err := env.svc.ManualCleanup(ctx); err == nil {
		t.Fatal("账户清理失败时应返回 error")
	}
}

// TestManualSpamCleanupDeletesOnlyExpiredSpam 覆盖垃圾邮件清理：
// 只清理超过保留期的垃圾邮件，未到期垃圾邮件与正常邮件都不能动。
func TestManualSpamCleanupDeletesOnlyExpiredSpam(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "spam", "auto_cleanup_days", "30")

	oldDetected := daysAgo(40)
	oldSpam := seedCleanupEmail(t, env.db, "old-spam", emailSeedOptions{isSpam: true, spamDetectedAt: &oldDetected})
	recentDetected := time.Now()
	recentSpam := seedCleanupEmail(t, env.db, "recent-spam", emailSeedOptions{isSpam: true, spamDetectedAt: &recentDetected})
	ham := seedCleanupEmail(t, env.db, "old-ham", emailSeedOptions{updatedAt: ptrTime(daysAgo(40))})

	cleaned, err := env.svc.ManualSpamCleanup(ctx)
	if err != nil {
		t.Fatalf("垃圾邮件清理失败: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("清理数量 = %d，期望 1", cleaned)
	}

	if got := reloadCleanupEmail(t, env, oldSpam.ID); !got.IsDeleted {
		t.Error("过期垃圾邮件未被软删除")
	}
	if got := reloadCleanupEmail(t, env, recentSpam.ID); got.IsDeleted {
		t.Error("未过期垃圾邮件被误删")
	}
	if got := reloadCleanupEmail(t, env, ham.ID); got.IsDeleted {
		t.Error("正常邮件被垃圾清理误删")
	}
}

// TestSpamCleanupFallsBackToSentAt 覆盖时间戳兜底：
// 既无 spam_detected_at 也无 user_marked_at 时用 sent_at 判断，避免老垃圾邮件永不清理。
func TestSpamCleanupFallsBackToSentAt(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "spam", "auto_cleanup_days", "30")

	email := seedCleanupEmail(t, env.db, "spam-no-timestamp", emailSeedOptions{isSpam: true})
	if err := env.db.Model(&model.Email{}).Where("id = ?", email.ID).
		UpdateColumns(map[string]interface{}{"sent_at": daysAgo(60)}).Error; err != nil {
		t.Fatalf("改写 sent_at 失败: %v", err)
	}

	cleaned, err := env.svc.ManualSpamCleanup(ctx)
	if err != nil {
		t.Fatalf("垃圾邮件清理失败: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("清理数量 = %d，期望 1（应按 sent_at 判定过期）", cleaned)
	}
}

// TestSpamCleanupDisabledByNegativeSetting 覆盖 days=-1：关闭垃圾邮件自动清理。
func TestSpamCleanupDisabledByNegativeSetting(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "spam", "auto_cleanup_days", "-1")
	oldDetected := daysAgo(400)
	spam := seedCleanupEmail(t, env.db, "spam-disabled", emailSeedOptions{isSpam: true, spamDetectedAt: &oldDetected})

	cleaned, err := env.svc.ManualSpamCleanup(ctx)
	if err != nil {
		t.Fatalf("关闭清理时不应报错: %v", err)
	}
	if cleaned != 0 {
		t.Errorf("关闭清理却清理了 %d 封", cleaned)
	}
	if got := reloadCleanupEmail(t, env, spam.ID); got.IsDeleted {
		t.Error("关闭清理却删除了垃圾邮件")
	}

	// 定时任务路径同样受配置约束
	env.svc.cleanupSpamEmails(ctx)
	if got := reloadCleanupEmail(t, env, spam.ID); got.IsDeleted {
		t.Error("定时清理忽略了 days=-1 配置")
	}
}

// TestCleanupSyncLogsRespectsRetention 覆盖同步日志清理：按 started_at 保留 N 天。
func TestCleanupSyncLogsRespectsRetention(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "sync_logs_retention_days", "7")

	oldLog := &model.SyncLog{AccountUID: "acct-cleanup", SyncType: "manual", Status: "success", StartedAt: daysAgo(30)}
	freshLog := &model.SyncLog{AccountUID: "acct-cleanup", SyncType: "manual", Status: "success", StartedAt: time.Now()}
	for _, log := range []*model.SyncLog{oldLog, freshLog} {
		if err := env.syncLogRepo.Create(ctx, log); err != nil {
			t.Fatalf("写入同步日志失败: %v", err)
		}
	}

	env.svc.cleanupSyncLogs(ctx)

	logs := loadSyncLogs(t, env.db)
	if len(logs) != 1 || logs[0].ID != freshLog.ID {
		t.Errorf("同步日志清理结果 = %+v，期望只保留新日志", logs)
	}
}

// TestCleanupSyncLogsDisabledByNegativeSetting 覆盖 days=-1：同步日志不清理。
func TestCleanupSyncLogsDisabledByNegativeSetting(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "sync_logs_retention_days", "-1")
	oldLog := &model.SyncLog{AccountUID: "acct-cleanup", SyncType: "manual", Status: "success", StartedAt: daysAgo(300)}
	if err := env.syncLogRepo.Create(ctx, oldLog); err != nil {
		t.Fatalf("写入同步日志失败: %v", err)
	}

	env.svc.cleanupSyncLogs(ctx)

	if logs := loadSyncLogs(t, env.db); len(logs) != 1 {
		t.Errorf("关闭清理却删除了同步日志: %+v", logs)
	}
}

// TestCleanupSpamDetectionLogsRespectsRetention 覆盖检测日志清理：按 created_at 保留 N 天。
func TestCleanupSpamDetectionLogsRespectsRetention(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "spam_detection_logs_retention_days", "7")

	oldLog := &model.SpamDetectionLog{EmailID: "1", IsSpam: true, FinalScore: 60}
	if err := env.spamLogRepo.Create(ctx, oldLog); err != nil {
		t.Fatalf("写入检测日志失败: %v", err)
	}
	if err := env.db.Model(&model.SpamDetectionLog{}).Where("id = ?", oldLog.ID).
		UpdateColumns(map[string]interface{}{"created_at": daysAgo(30)}).Error; err != nil {
		t.Fatalf("改写 created_at 失败: %v", err)
	}
	freshLog := &model.SpamDetectionLog{EmailID: "2", IsSpam: false, FinalScore: 10}
	if err := env.spamLogRepo.Create(ctx, freshLog); err != nil {
		t.Fatalf("写入检测日志失败: %v", err)
	}

	env.svc.cleanupSpamDetectionLogs(ctx)

	var logs []model.SpamDetectionLog
	if err := env.db.Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("读取检测日志失败: %v", err)
	}
	if len(logs) != 1 || logs[0].ID != freshLog.ID {
		t.Errorf("检测日志清理结果 = %+v，期望只保留新日志", logs)
	}
}

// TestCleanupRuleMatchLogsDeletesOrphans 覆盖孤儿命中记录清理：
// 邮件已被物理删除的命中记录无意义，必须清掉；邮件仍在的记录必须保留。
func TestCleanupRuleMatchLogsDeletesOrphans(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	email := seedCleanupEmail(t, env.db, "rule-match", emailSeedOptions{})
	kept := &model.RuleMatchLog{EmailID: email.ID, RuleID: 1, RuleName: "关键词", Actions: `["archive"]`}
	orphan := &model.RuleMatchLog{EmailID: 999999, RuleID: 2, RuleName: "正则", Actions: `["mark_read"]`}
	if err := env.ruleMatchRepo.CreateBatch(ctx, []*model.RuleMatchLog{kept, orphan}); err != nil {
		t.Fatalf("写入命中记录失败: %v", err)
	}

	env.svc.cleanupRuleMatchLogs(ctx)

	var logs []model.RuleMatchLog
	if err := env.db.Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("读取命中记录失败: %v", err)
	}
	if len(logs) != 1 || logs[0].ID != kept.ID {
		t.Errorf("孤儿清理结果 = %+v，期望只保留有效记录", logs)
	}
}

// TestCleanupDeletedEmailKeysRemovesExpired 覆盖已删除标识清理：
// 保留 90 天，过期记录必须清掉（否则表无限增长），新记录保留。
func TestCleanupDeletedEmailKeysRemovesExpired(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	oldKey := &model.DeletedEmailKey{AccountUID: "acct-cleanup", DedupeKey: "old-key"}
	freshKey := &model.DeletedEmailKey{AccountUID: "acct-cleanup", DedupeKey: "fresh-key"}
	for _, key := range []*model.DeletedEmailKey{oldKey, freshKey} {
		if err := env.db.Create(key).Error; err != nil {
			t.Fatalf("写入已删除标识失败: %v", err)
		}
	}
	if err := env.db.Model(&model.DeletedEmailKey{}).Where("id = ?", oldKey.ID).
		UpdateColumns(map[string]interface{}{"deleted_at": daysAgo(120)}).Error; err != nil {
		t.Fatalf("改写 deleted_at 失败: %v", err)
	}

	env.svc.cleanupDeletedEmailKeys(ctx)

	var keys []model.DeletedEmailKey
	if err := env.db.Order("id asc").Find(&keys).Error; err != nil {
		t.Fatalf("读取已删除标识失败: %v", err)
	}
	if len(keys) != 1 || keys[0].ID != freshKey.ID {
		t.Errorf("标识清理结果 = %+v，期望只保留新标识", keys)
	}
}

// TestManualLogCleanupRunsAllLogCleaners 覆盖手动日志清理入口：
// 一次调用要把同步日志、孤儿命中记录、检测日志都清一遍。
func TestManualLogCleanupRunsAllLogCleaners(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	seedSetting(t, env.db, "system", "sync_logs_retention_days", "7")
	seedSetting(t, env.db, "system", "spam_detection_logs_retention_days", "7")

	oldLog := &model.SyncLog{AccountUID: "acct-cleanup", SyncType: "manual", Status: "success", StartedAt: daysAgo(30)}
	if err := env.syncLogRepo.Create(ctx, oldLog); err != nil {
		t.Fatalf("写入同步日志失败: %v", err)
	}
	orphan := &model.RuleMatchLog{EmailID: 999999, RuleID: 1, RuleName: "孤儿"}
	if err := env.ruleMatchRepo.CreateBatch(ctx, []*model.RuleMatchLog{orphan}); err != nil {
		t.Fatalf("写入命中记录失败: %v", err)
	}
	oldSpamLog := &model.SpamDetectionLog{EmailID: "1", IsSpam: true, FinalScore: 60}
	if err := env.spamLogRepo.Create(ctx, oldSpamLog); err != nil {
		t.Fatalf("写入检测日志失败: %v", err)
	}
	if err := env.db.Model(&model.SpamDetectionLog{}).Where("id = ?", oldSpamLog.ID).
		UpdateColumns(map[string]interface{}{"created_at": daysAgo(30)}).Error; err != nil {
		t.Fatalf("改写 created_at 失败: %v", err)
	}

	if err := env.svc.ManualLogCleanup(ctx); err != nil {
		t.Fatalf("手动日志清理失败: %v", err)
	}

	if logs := loadSyncLogs(t, env.db); len(logs) != 0 {
		t.Errorf("同步日志未被清理: %+v", logs)
	}
	var matchLogs []model.RuleMatchLog
	if err := env.db.Find(&matchLogs).Error; err != nil {
		t.Fatalf("读取命中记录失败: %v", err)
	}
	if len(matchLogs) != 0 {
		t.Errorf("孤儿命中记录未被清理: %+v", matchLogs)
	}
	var spamLogs []model.SpamDetectionLog
	if err := env.db.Find(&spamLogs).Error; err != nil {
		t.Fatalf("读取检测日志失败: %v", err)
	}
	if len(spamLogs) != 0 {
		t.Errorf("检测日志未被清理: %+v", spamLogs)
	}
}

// TestCleanupServiceStartStop 覆盖调度器生命周期：
// 未启动时 Stop 无副作用，重复 Stop 不 panic，Start 后 isRunning 置位。
func TestCleanupServiceStartStop(t *testing.T) {
	env := newCleanupTestEnv(t)
	ctx := context.Background()

	env.svc.Stop() // 未启动
	if env.svc.isRunning {
		t.Fatal("未启动却标记为运行中")
	}

	if err := env.svc.Start(ctx); err != nil {
		t.Fatalf("启动清理服务失败: %v", err)
	}
	if !env.svc.isRunning {
		t.Fatal("启动后未标记为运行中")
	}
	// Start 会异步跑一次日志清理，等它结束再停，避免与后续用例抢同一张表
	time.Sleep(200 * time.Millisecond)

	env.svc.Stop()
	if env.svc.isRunning {
		t.Fatal("停止后仍标记为运行中")
	}
	env.svc.Stop() // 重复停止不得 panic

	// 重复启动不应重复注册任务
	if err := env.svc.Start(ctx); err != nil {
		t.Fatalf("二次启动失败: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	env.svc.Stop()
}

func ptrTime(t time.Time) *time.Time { return &t }
