package service

import (
	"context"
	"strings"
	"testing"

	"fusionmail/internal/model"
)

func testSpamRule(name, category, pattern string) *model.SpamRule {
	return &model.SpamRule{
		Name:     name,
		Category: category,
		Pattern:  pattern,
		Score:    10,
		Enabled:  true,
	}
}

// TestCreateRuleValidatesAndForcesNonBuiltin 覆盖创建校验：
// 非法正则/空模式/未知类别必须拒绝；外部传入的 IsBuiltin 必须被强制改回 false
// （否则用户可以伪造一条「内置」规则来绕过删除保护）。
func TestCreateRuleValidatesAndForcesNonBuiltin(t *testing.T) {
	env := newSpamTestEnv(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		rule    *model.SpamRule
		wantErr string
	}{
		{
			name:    "空模式",
			rule:    testSpamRule("空", "keyword", ""),
			wantErr: "规则模式不能为空",
		},
		{
			name:    "非法正则",
			rule:    testSpamRule("坏正则", "pattern", "([a-z"),
			wantErr: "无效的正则表达式",
		},
		{
			name:    "未知类别",
			rule:    testSpamRule("未知", "magic", "x"),
			wantErr: "未知的规则类别",
		},
		{
			name:    "斜杠包裹的非法正则",
			rule:    testSpamRule("坏斜杠正则", "content", "/([a-z/"),
			wantErr: "无效的正则表达式",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := env.svc.CreateRule(ctx, c.rule)
			if err == nil {
				t.Fatalf("非法规则应被拒绝")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %q，期望包含 %q", err.Error(), c.wantErr)
			}
		})
	}

	forged := testSpamRule("伪造内置", "keyword", "发票")
	forged.IsBuiltin = true
	if err := env.svc.CreateRule(ctx, forged); err != nil {
		t.Fatalf("合法规则创建失败: %v", err)
	}
	if forged.IsBuiltin {
		t.Error("CreateRule 未强制清除 IsBuiltin")
	}
}

// TestRuleCRUDLifecycle 覆盖规则生命周期：创建 → 查询 → 修改 → 切换 → 删除。
func TestRuleCRUDLifecycle(t *testing.T) {
	env := newSpamTestEnv(t)
	ctx := context.Background()

	rule := testSpamRule("发票关键词", "keyword", "发票")
	if err := env.svc.CreateRule(ctx, rule); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("创建后规则 ID 仍为 0")
	}

	got, err := env.svc.GetRuleByID(ctx, rule.ID)
	if err != nil {
		t.Fatalf("查询规则失败: %v", err)
	}
	if got == nil || got.Pattern != "发票" {
		t.Fatalf("查询到的规则不匹配: %+v", got)
	}

	rule.Pattern = "发票|报销"
	rule.Description = "更新后的描述"
	if err := env.svc.UpdateRule(ctx, rule); err != nil {
		t.Fatalf("更新规则失败: %v", err)
	}
	got, _ = env.svc.GetRuleByID(ctx, rule.ID)
	if got.Pattern != "发票|报销" || got.Description != "更新后的描述" {
		t.Errorf("更新未生效: %+v", got)
	}

	if err := env.svc.ToggleRule(ctx, rule.ID); err != nil {
		t.Fatalf("切换规则失败: %v", err)
	}
	got, _ = env.svc.GetRuleByID(ctx, rule.ID)
	if got.Enabled {
		t.Error("ToggleRule 未把启用的规则关掉")
	}

	if err := env.svc.DeleteRule(ctx, rule.ID); err != nil {
		t.Fatalf("删除规则失败: %v", err)
	}
	got, _ = env.svc.GetRuleByID(ctx, rule.ID)
	if got != nil {
		t.Errorf("删除后仍能查到规则: %+v", got)
	}
}

// TestBuiltinRuleProtection 覆盖内置规则保护：
// 内置规则不能删除，也只能改启用状态（不能改模式）。
func TestBuiltinRuleProtection(t *testing.T) {
	env := newSpamTestEnv(t)
	ctx := context.Background()

	builtin := seedSpamRule(t, env.ruleRepo, &model.SpamRule{
		Name:      "内置规则",
		Category:  "keyword",
		Pattern:   "原始模式",
		Score:     20,
		Enabled:   true,
		IsBuiltin: true,
	})

	if err := env.svc.DeleteRule(ctx, builtin.ID); err == nil {
		t.Fatal("内置规则不应允许删除")
	} else if !strings.Contains(err.Error(), "内置规则不能删除") {
		t.Errorf("error = %q，期望提示内置规则不可删", err.Error())
	}

	// UpdateRule 对内置规则只接受 Enabled
	builtin.Pattern = "被篡改的模式"
	builtin.Enabled = false
	if err := env.svc.UpdateRule(ctx, builtin); err != nil {
		t.Fatalf("更新内置规则启用状态失败: %v", err)
	}
	got, _ := env.svc.GetRuleByID(ctx, builtin.ID)
	if got.Pattern != "原始模式" {
		t.Errorf("内置规则的 Pattern 被改动为 %q", got.Pattern)
	}
	if got.Enabled {
		t.Error("内置规则的 Enabled 未按请求关闭")
	}
}

// TestRuleOperationsOnMissingRule 覆盖错误路径：对不存在的规则执行更新/删除/切换。
func TestRuleOperationsOnMissingRule(t *testing.T) {
	env := newSpamTestEnv(t)
	ctx := context.Background()

	missing := testSpamRule("不存在", "keyword", "x")
	missing.ID = 999999

	if err := env.svc.UpdateRule(ctx, missing); err == nil {
		t.Error("更新不存在的规则应报错")
	}
	if err := env.svc.DeleteRule(ctx, missing.ID); err == nil {
		t.Error("删除不存在的规则应报错")
	}
	if err := env.svc.ToggleRule(ctx, missing.ID); err == nil {
		t.Error("切换不存在的规则应报错")
	}
	if rule, err := env.svc.GetRuleByID(ctx, missing.ID); err != nil || rule != nil {
		t.Errorf("查询不存在的规则 = %+v, err=%v，期望 nil,nil", rule, err)
	}
}

// TestGetRulesPaginationAndCategoryFilter 覆盖列表的类别过滤与分页边界。
func TestGetRulesPaginationAndCategoryFilter(t *testing.T) {
	env := newSpamTestEnv(t)
	ctx := context.Background()

	seedSpamRule(t, env.ruleRepo, testSpamRule("关键词1", "keyword", "a"))
	seedSpamRule(t, env.ruleRepo, testSpamRule("关键词2", "keyword", "b"))
	seedSpamRule(t, env.ruleRepo, testSpamRule("正则1", "pattern", "c.*"))

	rules, total, err := env.svc.GetRules(ctx, "", 1, 2)
	if err != nil {
		t.Fatalf("查询规则列表失败: %v", err)
	}
	if total != 3 || len(rules) != 2 {
		t.Fatalf("分页 total=%d len=%d，期望 3/2", total, len(rules))
	}

	rules, total, err = env.svc.GetRules(ctx, "pattern", 1, 10)
	if err != nil {
		t.Fatalf("按类别查询失败: %v", err)
	}
	if total != 1 || len(rules) != 1 || rules[0].Category != "pattern" {
		t.Fatalf("类别过滤结果 total=%d rules=%+v", total, rules)
	}

	// 边界：不存在的类别 → 空结果而不是报错
	rules, total, err = env.svc.GetRules(ctx, "attachment", 1, 10)
	if err != nil {
		t.Fatalf("空类别查询不应报错: %v", err)
	}
	if total != 0 || len(rules) != 0 {
		t.Errorf("空类别查询结果 total=%d len=%d", total, len(rules))
	}
}

// TestTestRuleMatching 覆盖规则试跑：关键词忽略大小写、正则返回命中片段、
// 非法模式拒绝、无命中返回 false。
func TestTestRuleMatching(t *testing.T) {
	env := newSpamTestEnv(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		category  string
		pattern   string
		content   string
		wantMatch bool
		wantErr   string
	}{
		{
			name:      "关键词忽略大小写",
			category:  "keyword",
			pattern:   "发票",
			content:   "请查收发票附件",
			wantMatch: true,
		},
		{
			name:      "关键词未命中",
			category:  "keyword",
			pattern:   "中奖",
			content:   "请查收发票附件",
			wantMatch: false,
		},
		{
			name:      "正则命中",
			category:  "pattern",
			pattern:   `\d{6}`,
			content:   "验证码 123456 请勿泄露",
			wantMatch: true,
		},
		{
			name:      "斜杠包裹正则命中",
			category:  "content",
			pattern:   `/https?:\/\/[^\s]+/`,
			content:   "点击 http://example.com/claim 领取",
			wantMatch: true,
		},
		{
			name:     "非法正则",
			category: "pattern",
			pattern:  "([a-z",
			content:  "任意内容",
			wantErr:  "无效的正则表达式",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matched, matches, err := env.svc.TestRule(ctx, c.pattern, c.category, c.content)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error = %v，期望包含 %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("试跑规则失败: %v", err)
			}
			if matched != c.wantMatch {
				t.Errorf("matched = %v，期望 %v", matched, c.wantMatch)
			}
			if c.wantMatch && len(matches) == 0 {
				t.Error("命中但未返回命中片段")
			}
			if !c.wantMatch && len(matches) != 0 {
				t.Errorf("未命中却返回片段: %v", matches)
			}
		})
	}
}

// TestGetRuleStatsAggregates 覆盖规则统计：总数/启用/禁用/内置/自定义/命中次数。
func TestGetRuleStatsAggregates(t *testing.T) {
	env := newSpamTestEnv(t)
	ctx := context.Background()

	enabledBuiltin := seedSpamRule(t, env.ruleRepo, &model.SpamRule{
		Name: "内置启用", Category: "keyword", Pattern: "a", Enabled: true, IsBuiltin: true, HitCount: 3,
	})
	if enabledBuiltin.ID == 0 {
		t.Fatal("规则 fixture 未拿到主键")
	}
	seedSpamRule(t, env.ruleRepo, &model.SpamRule{
		Name: "内置禁用", Category: "keyword", Pattern: "b", Enabled: false, IsBuiltin: true, HitCount: 2,
	})
	seedSpamRule(t, env.ruleRepo, &model.SpamRule{
		Name: "自定义启用", Category: "keyword", Pattern: "c", Enabled: true, IsBuiltin: false, HitCount: 1,
	})

	stats, err := env.svc.GetRuleStats(ctx)
	if err != nil {
		t.Fatalf("统计规则失败: %v", err)
	}
	if stats.TotalCount != 3 || stats.EnabledCount != 2 || stats.DisabledCount != 1 {
		t.Errorf("计数错误: total=%d enabled=%d disabled=%d", stats.TotalCount, stats.EnabledCount, stats.DisabledCount)
	}
	if stats.BuiltinCount != 2 || stats.CustomCount != 1 {
		t.Errorf("内置/自定义计数错误: builtin=%d custom=%d", stats.BuiltinCount, stats.CustomCount)
	}
	if stats.TotalHits != 6 {
		t.Errorf("TotalHits = %d，期望 6", stats.TotalHits)
	}

	// 内部一致性：启用+禁用=总数，内置+自定义=总数
	if stats.EnabledCount+stats.DisabledCount != stats.TotalCount {
		t.Error("启用+禁用 != 总数")
	}
	if stats.BuiltinCount+stats.CustomCount != stats.TotalCount {
		t.Error("内置+自定义 != 总数")
	}
}

// TestNilRuleRepoFailsLoudly 覆盖依赖缺失：规则仓库为 nil 时必须显式报错，
// 不能静默返回空列表让界面显示「没有规则」。
func TestNilRuleRepoFailsLoudly(t *testing.T) {
	svc := &spamService{}
	ctx := context.Background()

	if _, _, err := svc.GetRules(ctx, "", 1, 10); err == nil {
		t.Error("GetRules 在仓库缺失时应报错")
	}
	if err := svc.CreateRule(ctx, testSpamRule("x", "keyword", "x")); err == nil {
		t.Error("CreateRule 在仓库缺失时应报错")
	}
	if _, err := svc.GetRuleStats(ctx); err == nil {
		t.Error("GetRuleStats 在仓库缺失时应报错")
	}
}
