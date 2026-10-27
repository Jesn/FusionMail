package service

import (
	"context"
	"testing"
)

// TestMarkAsSpamSetsUserFlagsAndPersists 覆盖手动标记垃圾邮件：
// is_spam / user_marked_spam / user_marked_at 三个字段必须一起落库。
func TestMarkAsSpamSetsUserFlagsAndPersists(t *testing.T) {
	env := newSpamTestEnv(t)
	email := seedSpamTestEmail(t, env.db, "acct-1", "p-1", false)

	if err := env.svc.MarkAsSpam(context.Background(), []int64{email.ID}); err != nil {
		t.Fatalf("标记垃圾邮件失败: %v", err)
	}

	got := reloadEmail(t, env, email.ID)
	if !got.IsSpam {
		t.Error("IsSpam 未置位")
	}
	if !got.UserMarkedSpam {
		t.Error("UserMarkedSpam 未置位（无法区分用户标记与规则判定）")
	}
	if got.UserMarkedAt == nil {
		t.Error("UserMarkedAt 未记录")
	}
}

// TestMarkAsSpamIgnoresMissingEmail 覆盖边界：不存在的邮件被跳过而不是报错，
// 也不影响同一批次里其他邮件的标记。
func TestMarkAsSpamIgnoresMissingEmail(t *testing.T) {
	env := newSpamTestEnv(t)
	email := seedSpamTestEmail(t, env.db, "acct-1", "p-1", false)

	if err := env.svc.MarkAsSpam(context.Background(), []int64{999999, email.ID}); err != nil {
		t.Fatalf("批次中含不存在邮件时不应报错: %v", err)
	}
	if !reloadEmail(t, env, email.ID).IsSpam {
		t.Error("批次中存在的邮件未被标记")
	}
}

// TestUnmarkAsSpamClearsFlags 覆盖取消标记：三个字段都要清掉，
// 否则邮件会一直带着「用户标记过」的痕迹。
func TestUnmarkAsSpamClearsFlags(t *testing.T) {
	env := newSpamTestEnv(t)
	email := seedSpamTestEmail(t, env.db, "acct-1", "p-1", true)

	if err := env.svc.MarkAsSpam(context.Background(), []int64{email.ID}); err != nil {
		t.Fatalf("标记垃圾邮件失败: %v", err)
	}
	if err := env.svc.UnmarkAsSpam(context.Background(), []int64{email.ID}); err != nil {
		t.Fatalf("取消标记失败: %v", err)
	}

	got := reloadEmail(t, env, email.ID)
	if got.IsSpam {
		t.Error("IsSpam 未清除")
	}
	if got.UserMarkedSpam {
		t.Error("UserMarkedSpam 未清除")
	}
	if got.UserMarkedAt != nil {
		t.Error("UserMarkedAt 未清除")
	}
}

// TestUserFeedbackUpdatesReputationAndTraining 覆盖用户反馈的副作用：
// 标记垃圾 → 发件人信誉降分 + 记一次 spam 训练；取消标记 → 信誉升分 + 记一次 ham 训练。
//
// 直接调用同步内部方法而不是走 MarkAsSpam：后者把副作用放在 goroutine 里，
// 断言会变成竞态。这里验证的是副作用本身的语义。
func TestUserFeedbackUpdatesReputationAndTraining(t *testing.T) {
	env := newSpamSideEffectEnv(t)
	ctx := context.Background()
	email := seedSpamTestEmail(t, env.db, "acct-1", "p-1", false)

	env.svc.updateReputationForSpam(ctx, "spammer@example.com", true)
	env.svc.addBayesianTraining(ctx, email, true)

	reputation, err := env.reputationRepo.FindByEmail(ctx, "spammer@example.com")
	if err != nil {
		t.Fatalf("读取发件人信誉失败: %v", err)
	}
	if reputation == nil {
		t.Fatal("标记垃圾后没有建立发件人信誉记录")
	}
	if reputation.SpamCount != 1 {
		t.Errorf("SpamCount = %d，期望 1", reputation.SpamCount)
	}
	if reputation.ReputationScore >= 50 {
		t.Errorf("ReputationScore = %v，标记垃圾后应低于初始 50 分", reputation.ReputationScore)
	}

	spamTraining, err := env.trainingRepo.CountByUserAndType(ctx, email.AccountUID, true)
	if err != nil {
		t.Fatalf("统计训练数据失败: %v", err)
	}
	if spamTraining != 1 {
		t.Errorf("spam 训练样本数 = %d，期望 1", spamTraining)
	}

	// 反向：取消标记应升分并记录 ham 样本
	env.svc.updateReputationForSpam(ctx, "spammer@example.com", false)
	env.svc.addBayesianTraining(ctx, email, false)

	reputation, err = env.reputationRepo.FindByEmail(ctx, "spammer@example.com")
	if err != nil {
		t.Fatalf("读取发件人信誉失败: %v", err)
	}
	if reputation.HamCount != 1 {
		t.Errorf("HamCount = %d，期望 1", reputation.HamCount)
	}
	if reputation.ReputationScore <= 40 {
		t.Errorf("ReputationScore = %v，取消标记后应升分", reputation.ReputationScore)
	}

	hamTraining, err := env.trainingRepo.CountByUserAndType(ctx, email.AccountUID, false)
	if err != nil {
		t.Fatalf("统计训练数据失败: %v", err)
	}
	if hamTraining != 1 {
		t.Errorf("ham 训练样本数 = %d，期望 1", hamTraining)
	}
}

// TestBatchDeleteSpamCountsOnlyExistingEmails 覆盖批量删除：
// 计数只统计真正删除成功的邮件（不存在的 ID 不得计入）。
func TestBatchDeleteSpamCountsOnlyExistingEmails(t *testing.T) {
	env := newSpamTestEnv(t)
	first := seedSpamTestEmail(t, env.db, "acct-1", "p-1", true)
	second := seedSpamTestEmail(t, env.db, "acct-1", "p-2", true)

	deleted, err := env.svc.BatchDeleteSpam(context.Background(), []int64{first.ID, second.ID, 999999})
	if err != nil {
		t.Fatalf("批量删除失败: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("删除计数 = %d，期望 2", deleted)
	}
	for _, id := range []int64{first.ID, second.ID} {
		if !reloadEmail(t, env, id).IsDeleted {
			t.Errorf("邮件 %d 未被软删除", id)
		}
	}
}

// TestEmptySpamFolderOnlyAffectsSpamInAccount 覆盖清空垃圾箱：
// 只清指定账户的垃圾邮件，正常邮件与其他账户的垃圾邮件都不能动。
func TestEmptySpamFolderOnlyAffectsSpamInAccount(t *testing.T) {
	env := newSpamTestEnv(t)
	spamA := seedSpamTestEmail(t, env.db, "acct-1", "p-1", true)
	spamA2 := seedSpamTestEmail(t, env.db, "acct-1", "p-2", true)
	hamA := seedSpamTestEmail(t, env.db, "acct-1", "p-3", false)
	spamB := seedSpamTestEmail(t, env.db, "acct-2", "p-4", true)

	deleted, err := env.svc.EmptySpamFolder(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("清空垃圾箱失败: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("清空计数 = %d，期望 2", deleted)
	}

	if !reloadEmail(t, env, spamA.ID).IsDeleted || !reloadEmail(t, env, spamA2.ID).IsDeleted {
		t.Error("acct-1 的垃圾邮件未被清空")
	}
	if reloadEmail(t, env, hamA.ID).IsDeleted {
		t.Error("正常邮件被误删")
	}
	if reloadEmail(t, env, spamB.ID).IsDeleted {
		t.Error("其他账户的垃圾邮件被误删")
	}
}

// TestGetSpamEmailsAndStatsScopedByAccount 覆盖查询与统计：
// 只统计未删除的垃圾邮件，且按账户隔离；分页参数越界时被夹到合法范围。
func TestGetSpamEmailsAndStatsScopedByAccount(t *testing.T) {
	env := newSpamTestEnv(t)
	seedSpamTestEmail(t, env.db, "acct-1", "p-1", true)
	seedSpamTestEmail(t, env.db, "acct-1", "p-2", true)
	seedSpamTestEmail(t, env.db, "acct-1", "p-3", false)
	seedSpamTestEmail(t, env.db, "acct-2", "p-4", true)

	emails, total, err := env.svc.GetSpamEmails(context.Background(), "acct-1", 1, 10)
	if err != nil {
		t.Fatalf("查询垃圾邮件失败: %v", err)
	}
	if total != 2 || len(emails) != 2 {
		t.Fatalf("acct-1 垃圾邮件 total=%d len=%d，期望 2/2", total, len(emails))
	}

	stats, err := env.svc.GetSpamStats(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("统计垃圾邮件失败: %v", err)
	}
	if stats.TotalCount != 2 {
		t.Errorf("TotalCount = %d，期望 2", stats.TotalCount)
	}
	if stats.UnreadCount != 2 {
		t.Errorf("UnreadCount = %d，期望 2（fixture 均为未读）", stats.UnreadCount)
	}

	// 边界：page/pageSize 越界应被夹到合法值而不是报错或返回空
	emails, total, err = env.svc.GetSpamEmails(context.Background(), "acct-1", 0, 0)
	if err != nil {
		t.Fatalf("越界分页参数不应报错: %v", err)
	}
	if total != 2 || len(emails) != 2 {
		t.Errorf("越界分页返回 total=%d len=%d，期望 2/2", total, len(emails))
	}
}
