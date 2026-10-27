package spam

import (
	"context"
	"testing"

	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"
)

func newReputationTestEnv(t *testing.T) (*ReputationManager, repository.SenderReputationRepository) {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)
	repo := repository.NewSenderReputationRepository(db)
	return NewReputationManager(repo, nil), repo
}

// TestGetOrCreateReputationDefaults 覆盖首次接触的发件人：
// 默认 50 分 / neutral / 域名解析正确，且重复调用不会写出第二行。
func TestGetOrCreateReputationDefaults(t *testing.T) {
	manager, repo := newReputationTestEnv(t)
	ctx := context.Background()

	reputation, err := manager.GetOrCreateReputation(ctx, "newcomer@sub.example.com")
	if err != nil {
		t.Fatalf("创建信誉记录失败: %v", err)
	}
	if reputation.ReputationScore != 50 {
		t.Errorf("初始评分 = %v，期望 50", reputation.ReputationScore)
	}
	if reputation.TrustLevel != "neutral" {
		t.Errorf("初始信任级别 = %q，期望 neutral", reputation.TrustLevel)
	}
	if reputation.Domain != "sub.example.com" {
		t.Errorf("域名 = %q，期望 sub.example.com", reputation.Domain)
	}
	if reputation.RBLStatus != "unknown" {
		t.Errorf("RBL 状态 = %q，期望 unknown", reputation.RBLStatus)
	}

	again, err := manager.GetOrCreateReputation(ctx, "newcomer@sub.example.com")
	if err != nil {
		t.Fatalf("二次获取失败: %v", err)
	}
	if again.ID != reputation.ID {
		t.Errorf("二次获取创建了新记录: %d != %d", again.ID, reputation.ID)
	}

	_, total, err := repo.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("列出发件人信誉失败: %v", err)
	}
	if total != 1 {
		t.Errorf("信誉记录数 = %d，期望 1", total)
	}
}

// TestUserFeedbackAdjustsScoreAndClamps 覆盖��户反馈：
// 标记垃圾 -10、取消标记 +5，且评分必须夹在 0-100 之间并同步信任级别。
func TestUserFeedbackAdjustsScoreAndClamps(t *testing.T) {
	manager, _ := newReputationTestEnv(t)
	ctx := context.Background()
	sender := "feedback@example.com"

	if err := manager.UpdateReputationByUserFeedback(ctx, sender, true); err != nil {
		t.Fatalf("标记垃圾失败: %v", err)
	}
	reputation, err := manager.GetOrCreateReputation(ctx, sender)
	if err != nil {
		t.Fatalf("读取信誉失败: %v", err)
	}
	if reputation.ReputationScore != 40 || reputation.SpamCount != 1 || reputation.TotalEmails != 1 {
		t.Errorf("标记垃圾后 = score %v / spam %d / total %d，期望 40/1/1",
			reputation.ReputationScore, reputation.SpamCount, reputation.TotalEmails)
	}

	if err := manager.UpdateReputationByUserFeedback(ctx, sender, false); err != nil {
		t.Fatalf("取消标记失败: %v", err)
	}
	reputation, _ = manager.GetOrCreateReputation(ctx, sender)
	if reputation.ReputationScore != 45 || reputation.HamCount != 1 || reputation.TotalEmails != 2 {
		t.Errorf("取消标记后 = score %v / ham %d / total %d，期望 45/1/2",
			reputation.ReputationScore, reputation.HamCount, reputation.TotalEmails)
	}

	// 下限：连续标记垃圾必须夹在 0，不能变成负数
	for i := 0; i < 10; i++ {
		if err := manager.UpdateReputationByUserFeedback(ctx, sender, true); err != nil {
			t.Fatalf("第 %d 次标记垃圾失败: %v", i, err)
		}
	}
	reputation, _ = manager.GetOrCreateReputation(ctx, sender)
	if reputation.ReputationScore != 0 {
		t.Errorf("评分下限 = %v，期望 0", reputation.ReputationScore)
	}
	if reputation.TrustLevel != "blocked" {
		t.Errorf("0 分信任级别 = %q，期望 blocked", reputation.TrustLevel)
	}

	// 上限：连续取消标记必须夹在 100
	for i := 0; i < 30; i++ {
		if err := manager.UpdateReputationByUserFeedback(ctx, sender, false); err != nil {
			t.Fatalf("第 %d 次取消标记失败: %v", i, err)
		}
	}
	reputation, _ = manager.GetOrCreateReputation(ctx, sender)
	if reputation.ReputationScore != 100 {
		t.Errorf("评分上限 = %v，期望 100", reputation.ReputationScore)
	}
	if reputation.TrustLevel != "trusted" {
		t.Errorf("100 分信任级别 = %q，期望 trusted", reputation.TrustLevel)
	}
}

// TestUpdateReputationByDetectionBranches 覆盖检测结果分支：
// >=60 降 5 分并记 spam；>=40 降 2 分（不记 spam）；<40 升 1 分并记 ham。
func TestUpdateReputationByDetectionBranches(t *testing.T) {
	cases := []struct {
		name      string
		spamScore int
		wantScore float64
		wantSpam  int64
		wantHam   int64
		wantTotal int64
	}{
		{name: "确认垃圾", spamScore: 60, wantScore: 45, wantSpam: 1, wantHam: 0, wantTotal: 1},
		{name: "可疑邮件", spamScore: 40, wantScore: 48, wantSpam: 0, wantHam: 0, wantTotal: 1},
		{name: "正常邮件", spamScore: 0, wantScore: 51, wantSpam: 0, wantHam: 1, wantTotal: 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			manager, _ := newReputationTestEnv(t)
			ctx := context.Background()
			sender := "detect@example.com"

			if err := manager.UpdateReputationByDetection(ctx, sender, c.spamScore); err != nil {
				t.Fatalf("按检测结果更新信誉失败: %v", err)
			}
			reputation, err := manager.GetOrCreateReputation(ctx, sender)
			if err != nil {
				t.Fatalf("读取信誉失败: %v", err)
			}
			if reputation.ReputationScore != c.wantScore {
				t.Errorf("评分 = %v，期望 %v", reputation.ReputationScore, c.wantScore)
			}
			if reputation.SpamCount != c.wantSpam || reputation.HamCount != c.wantHam || reputation.TotalEmails != c.wantTotal {
				t.Errorf("计数 = spam %d / ham %d / total %d，期望 %d/%d/%d",
					reputation.SpamCount, reputation.HamCount, reputation.TotalEmails, c.wantSpam, c.wantHam, c.wantTotal)
			}
		})
	}
}

// TestReputationScoreThresholds 覆盖低/高信誉判定边界（<20 低，>80 高）。
func TestReputationScoreThresholds(t *testing.T) {
	manager, repo := newReputationTestEnv(t)
	ctx := context.Background()

	sender := "threshold@example.com"
	reputation, err := manager.GetOrCreateReputation(ctx, sender)
	if err != nil {
		t.Fatalf("创建信誉失败: %v", err)
	}

	low, err := manager.IsLowReputation(ctx, sender)
	if err != nil || low {
		t.Errorf("50 分被判为低信誉: low=%v err=%v", low, err)
	}
	high, err := manager.IsHighReputation(ctx, sender)
	if err != nil || high {
		t.Errorf("50 分被判为高信誉: high=%v err=%v", high, err)
	}

	// 边界：恰好 20 分不算低信誉，恰好 80 分不算高信誉
	reputation.ReputationScore = 20
	if err := repo.Update(ctx, reputation); err != nil {
		t.Fatalf("更新信誉失败: %v", err)
	}
	if low, _ := manager.IsLowReputation(ctx, sender); low {
		t.Error("20 分被判为低信誉（边界应为不低）")
	}

	reputation.ReputationScore = 80
	if err := repo.Update(ctx, reputation); err != nil {
		t.Fatalf("更新信誉失败: %v", err)
	}
	if high, _ := manager.IsHighReputation(ctx, sender); high {
		t.Error("80 分被判为高信誉（边界应为不高）")
	}

	reputation.ReputationScore = 19
	if err := repo.Update(ctx, reputation); err != nil {
		t.Fatalf("更新信誉失败: %v", err)
	}
	if low, _ := manager.IsLowReputation(ctx, sender); !low {
		t.Error("19 分未被判为低信誉")
	}

	reputation.ReputationScore = 81
	if err := repo.Update(ctx, reputation); err != nil {
		t.Fatalf("更新信誉失败: %v", err)
	}
	if high, _ := manager.IsHighReputation(ctx, sender); !high {
		t.Error("81 分未被判为高信誉")
	}
}

// TestAdjustScoreByReputation 覆盖信誉对垃圾评分的加权：
// 高信誉打折 20%，低信誉加价 20%，中性不动，且结果夹在 0-100。
func TestAdjustScoreByReputation(t *testing.T) {
	manager, repo := newReputationTestEnv(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		score     float64
		spamScore int
		want      int
	}{
		{name: "高信誉打折", score: 90, spamScore: 100, want: 80},
		{name: "低信誉加价", score: 10, spamScore: 100, want: 100},
		{name: "中性不动", score: 50, spamScore: 60, want: 60},
		{name: "低信誉加价后夹到 100", score: 0, spamScore: 95, want: 100},
		{name: "零分保持零", score: 50, spamScore: 0, want: 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sender := "adjust-" + c.name + "@example.com"
			reputation, err := manager.GetOrCreateReputation(ctx, sender)
			if err != nil {
				t.Fatalf("创建信誉失败: %v", err)
			}
			reputation.ReputationScore = c.score
			if err := repo.Update(ctx, reputation); err != nil {
				t.Fatalf("更新信誉失败: %v", err)
			}

			got, err := manager.AdjustScoreByReputation(ctx, sender, c.spamScore)
			if err != nil {
				t.Fatalf("调整评分失败: %v", err)
			}
			if got != c.want {
				t.Errorf("调整后评分 = %d，期望 %d", got, c.want)
			}
			if got < 0 || got > 100 {
				t.Errorf("调整后评分越界: %d", got)
			}
		})
	}
}

// TestGetReputationStatsComputesSpamRate 覆盖统计口径：
// 垃圾率 = spam / total，无邮件时为 0（不能除零）。
func TestGetReputationStatsComputesSpamRate(t *testing.T) {
	manager, _ := newReputationTestEnv(t)
	ctx := context.Background()
	sender := "stats@example.com"

	stats, err := manager.GetReputationStats(ctx, sender)
	if err != nil {
		t.Fatalf("读取统计失败: %v", err)
	}
	if stats.SpamRate != 0 || stats.TotalEmails != 0 {
		t.Errorf("空统计 = %+v，期望垃圾率 0 且无邮件", stats)
	}
	if stats.Domain != "example.com" {
		t.Errorf("统计域名 = %q", stats.Domain)
	}

	for i := 0; i < 2; i++ {
		if err := manager.UpdateReputationByUserFeedback(ctx, sender, true); err != nil {
			t.Fatalf("标记垃圾失败: %v", err)
		}
		if err := manager.UpdateReputationByUserFeedback(ctx, sender, false); err != nil {
			t.Fatalf("取消标记失败: %v", err)
		}
	}

	stats, err = manager.GetReputationStats(ctx, sender)
	if err != nil {
		t.Fatalf("读取统计失败: %v", err)
	}
	if stats.TotalEmails != 4 || stats.SpamCount != 2 || stats.HamCount != 2 {
		t.Errorf("统计计数 = total %d / spam %d / ham %d，期望 4/2/2", stats.TotalEmails, stats.SpamCount, stats.HamCount)
	}
	if stats.SpamRate != 50 {
		t.Errorf("垃圾率 = %v，期望 50", stats.SpamRate)
	}
}

// TestGetReputationScoreForUnknownSender 覆盖未知发件人：
// 返回默认 50 分并顺手建档，不能报错。
func TestGetReputationScoreForUnknownSender(t *testing.T) {
	manager, repo := newReputationTestEnv(t)
	ctx := context.Background()

	score, err := manager.GetReputationScore(ctx, "unknown@example.com")
	if err != nil {
		t.Fatalf("未知发件人取分失败: %v", err)
	}
	if score != 50 {
		t.Errorf("未知发件人评分 = %v，期望 50", score)
	}

	stored, err := repo.FindByEmail(ctx, "unknown@example.com")
	if err != nil {
		t.Fatalf("查询信誉失败: %v", err)
	}
	if stored == nil {
		t.Error("取分后没有建立信誉记录")
	}
}

// TestBatchUpdateReputationIsSafeNoop 覆盖批量任务占位实现：
// 当前是空实现，必须返回 nil 而不是 panic（定时任务会调用它）。
func TestBatchUpdateReputationIsSafeNoop(t *testing.T) {
	manager, _ := newReputationTestEnv(t)

	if err := manager.BatchUpdateReputation(context.Background()); err != nil {
		t.Errorf("批量更新信誉返回错误: %v", err)
	}
}

// TestCalculateTrustLevelBoundaries 覆盖信任级别分档边界。
func TestCalculateTrustLevelBoundaries(t *testing.T) {
	manager, _ := newReputationTestEnv(t)

	cases := []struct {
		score float64
		want  string
	}{
		{score: 100, want: "trusted"},
		{score: 80, want: "trusted"},
		{score: 79.9, want: "neutral"},
		{score: 40, want: "neutral"},
		{score: 39.9, want: "suspicious"},
		{score: 20, want: "suspicious"},
		{score: 19.9, want: "blocked"},
		{score: 0, want: "blocked"},
	}

	for _, c := range cases {
		if got := manager.calculateTrustLevel(c.score); got != c.want {
			t.Errorf("calculateTrustLevel(%v) = %q，期望 %q", c.score, got, c.want)
		}
	}
}
