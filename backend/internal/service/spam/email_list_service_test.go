package spam

import (
	"context"
	"testing"

	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"
)

func newEmailListTestEnv(t *testing.T) (*EmailListService, *WhitelistChecker, repository.EmailListRepository) {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)
	repo := repository.NewEmailListRepository(db)
	checker := NewWhitelistChecker(repo, nil) // cache 为 nil：走数据库直查路径
	return NewEmailListService(repo, checker), checker, repo
}

// TestAddToListNormalizesTarget 覆盖写入规范化：
// 邮箱/域名都要转小写并识别出 target_type，否则后续匹配会因大小写不一致而失效。
func TestAddToListNormalizesTarget(t *testing.T) {
	svc, _, _ := newEmailListTestEnv(t)
	ctx := context.Background()

	entry, err := svc.AddToWhitelist(ctx, "user-1", "  Friend@Example.COM  ", "同事")
	if err != nil {
		t.Fatalf("添加白名单失败: %v", err)
	}
	if entry.Target != "friend@example.com" {
		t.Errorf("Target = %q，期望小写去空格", entry.Target)
	}
	if entry.TargetType != "email" {
		t.Errorf("TargetType = %q，期望 email", entry.TargetType)
	}
	if entry.Type != "whitelist" || entry.UserUID != "user-1" || entry.Reason != "同事" {
		t.Errorf("条目字段错误: %+v", entry)
	}

	domainEntry, err := svc.AddToBlacklist(ctx, "user-1", "Spam.Example.COM", "广告域名")
	if err != nil {
		t.Fatalf("添加黑名单失败: %v", err)
	}
	if domainEntry.TargetType != "domain" || domainEntry.Target != "spam.example.com" {
		t.Errorf("域名条目 = %+v，期望 domain/spam.example.com", domainEntry)
	}
}

// TestAddToListRejectsInvalidTarget 覆盖入参校验：空值、缺 TLD、缺本地部分都要拒绝。
func TestAddToListRejectsInvalidTarget(t *testing.T) {
	svc, _, _ := newEmailListTestEnv(t)
	ctx := context.Background()

	for _, target := range []string{"", "   ", "not-an-email", "@example.com", "example", "a@b"} {
		t.Run("target="+target, func(t *testing.T) {
			if _, err := svc.AddToWhitelist(ctx, "user-1", target, ""); err == nil {
				t.Errorf("非法目标 %q 应被拒绝", target)
			}
			if _, err := svc.AddToBlacklist(ctx, "user-1", target, ""); err == nil {
				t.Errorf("非法目标 %q 应被拒绝（黑名单）", target)
			}
		})
	}
}

// TestAddToListIsIdempotent 覆盖重复添加：同一条目应返回已有记录而不是写第二行。
func TestAddToListIsIdempotent(t *testing.T) {
	svc, _, repo := newEmailListTestEnv(t)
	ctx := context.Background()

	first, err := svc.AddToWhitelist(ctx, "user-1", "dup@example.com", "第一次")
	if err != nil {
		t.Fatalf("首次添加失败: %v", err)
	}
	second, err := svc.AddToWhitelist(ctx, "user-1", "DUP@example.com", "第二次")
	if err != nil {
		t.Fatalf("重复添加失败: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("重复添加创建了新条目: %d != %d", first.ID, second.ID)
	}

	_, total, err := repo.List(ctx, "user-1", "whitelist", 0, 10)
	if err != nil {
		t.Fatalf("列出白名单失败: %v", err)
	}
	if total != 1 {
		t.Errorf("白名单条目数 = %d，期望 1", total)
	}
}

// TestRemoveFromListChecksOwnershipAndType 覆盖删除的权限与类型校验：
// 别人的条目、类型不匹配、不存在的条目都不能删。
func TestRemoveFromListChecksOwnershipAndType(t *testing.T) {
	svc, _, repo := newEmailListTestEnv(t)
	ctx := context.Background()

	entry, err := svc.AddToWhitelist(ctx, "user-1", "mine@example.com", "")
	if err != nil {
		t.Fatalf("添加白名单失败: %v", err)
	}

	if err := svc.RemoveFromWhitelist(ctx, entry.ID, "user-2"); err == nil {
		t.Error("删除他人条目应被拒绝")
	}
	if err := svc.RemoveFromBlacklist(ctx, entry.ID, "user-1"); err == nil {
		t.Error("类型不匹配的删除应被拒绝")
	}
	if err := svc.RemoveFromWhitelist(ctx, 999999, "user-1"); err == nil {
		t.Error("删除不存在的条目应报错")
	}

	// 失败的删除不能把条目删掉
	if _, total, _ := repo.List(ctx, "user-1", "whitelist", 0, 10); total != 1 {
		t.Fatalf("失败的删除影响了数据: total=%d", total)
	}

	if err := svc.RemoveFromWhitelist(ctx, entry.ID, "user-1"); err != nil {
		t.Fatalf("正常删除失败: %v", err)
	}
	if _, total, _ := repo.List(ctx, "user-1", "whitelist", 0, 10); total != 0 {
		t.Errorf("删除后仍有 %d 条", total)
	}
}

// TestGetWhitelistAndBlacklistScopedByUser 覆盖列表查询的账户与类型隔离。
func TestGetWhitelistAndBlacklistScopedByUser(t *testing.T) {
	svc, _, _ := newEmailListTestEnv(t)
	ctx := context.Background()

	if _, err := svc.AddToWhitelist(ctx, "user-1", "w1@example.com", ""); err != nil {
		t.Fatalf("添加白名单失败: %v", err)
	}
	if _, err := svc.AddToWhitelist(ctx, "user-1", "w2@example.com", ""); err != nil {
		t.Fatalf("添加白名单失败: %v", err)
	}
	if _, err := svc.AddToBlacklist(ctx, "user-1", "b1@example.com", ""); err != nil {
		t.Fatalf("添加黑名单失败: %v", err)
	}
	if _, err := svc.AddToWhitelist(ctx, "user-2", "other@example.com", ""); err != nil {
		t.Fatalf("添加白名单失败: %v", err)
	}

	whitelist, total, err := svc.GetWhitelist(ctx, "user-1", 0, 10)
	if err != nil {
		t.Fatalf("查询白名单失败: %v", err)
	}
	if total != 2 || len(whitelist) != 2 {
		t.Errorf("user-1 白名单 total=%d len=%d，期望 2/2", total, len(whitelist))
	}
	for _, entry := range whitelist {
		if entry.Type != "whitelist" || entry.UserUID != "user-1" {
			t.Errorf("白名单混入其他数据: %+v", entry)
		}
	}

	blacklist, total, err := svc.GetBlacklist(ctx, "user-1", 0, 10)
	if err != nil {
		t.Fatalf("查询黑名单失败: %v", err)
	}
	if total != 1 || len(blacklist) != 1 {
		t.Errorf("user-1 黑名单 total=%d len=%d，期望 1/1", total, len(blacklist))
	}
}

// TestWhitelistCheckerMatchesEmailAndDomain 覆盖白/黑名单匹配语义：
// 精确邮箱命中，同域名其他地址也应命中（域名条目），无关发件人不命中。
func TestWhitelistCheckerMatchesEmailAndDomain(t *testing.T) {
	svc, checker, _ := newEmailListTestEnv(t)
	ctx := context.Background()

	if _, err := svc.AddToWhitelist(ctx, "user-1", "boss@corp.example.com", ""); err != nil {
		t.Fatalf("添加白名单失败: %v", err)
	}
	if _, err := svc.AddToWhitelist(ctx, "user-1", "partner.example.com", ""); err != nil {
		t.Fatalf("添加域名白名单失败: %v", err)
	}
	if _, err := svc.AddToBlacklist(ctx, "user-1", "spammer@bad.example.com", ""); err != nil {
		t.Fatalf("添加黑名单失败: %v", err)
	}

	cases := []struct {
		name          string
		sender        string
		wantWhitelist bool
		wantBlacklist bool
	}{
		{name: "精确邮箱命中白名单", sender: "boss@corp.example.com", wantWhitelist: true},
		{name: "域名命中白名单", sender: "anyone@partner.example.com", wantWhitelist: true},
		{name: "无关发件人不命中", sender: "stranger@other.example.com"},
		{name: "精确邮箱命中黑名单", sender: "spammer@bad.example.com", wantBlacklist: true},
		{name: "同域名其他地址不命中黑名单", sender: "other@bad.example.com"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			whitelisted, err := checker.CheckWhitelist(ctx, "user-1", c.sender)
			if err != nil {
				t.Fatalf("白名单检查失败: %v", err)
			}
			if whitelisted != c.wantWhitelist {
				t.Errorf("CheckWhitelist(%q) = %v，期望 %v", c.sender, whitelisted, c.wantWhitelist)
			}

			blacklisted, err := checker.CheckBlacklist(ctx, "user-1", c.sender)
			if err != nil {
				t.Fatalf("黑名单检查失败: %v", err)
			}
			if blacklisted != c.wantBlacklist {
				t.Errorf("CheckBlacklist(%q) = %v，期望 %v", c.sender, blacklisted, c.wantBlacklist)
			}
		})
	}

	// 其他用户不受影响
	if whitelisted, err := checker.CheckWhitelist(ctx, "user-2", "boss@corp.example.com"); err != nil || whitelisted {
		t.Errorf("白名单跨用户泄漏: whitelisted=%v err=%v", whitelisted, err)
	}
}

// TestBuiltinRulesInitializeIsIdempotent 覆盖内置规则初始化：
// 首次写入全部内置规则，第二次必须跳过（不能重复插入）。
func TestBuiltinRulesInitializeIsIdempotent(t *testing.T) {
	db := pgtest.DB(t)
	pgtest.Truncate(t)
	repo := repository.NewSpamRuleRepository(db)
	initializer := NewBuiltinRulesInitializer(repo)
	ctx := context.Background()

	builtin := initializer.getBuiltinRules()
	if len(builtin) == 0 {
		t.Fatal("内置规则列表为空")
	}
	for i, rule := range builtin {
		if rule.Pattern == "" {
			t.Errorf("第 %d 条内置规则模式为空", i)
		}
		if !rule.IsBuiltin {
			t.Errorf("第 %d 条内置规则未标记 IsBuiltin", i)
		}
		if !rule.Enabled {
			t.Errorf("第 %d 条内置规则默认未启用", i)
		}
		if rule.Name == "" || rule.Category == "" {
			t.Errorf("第 %d 条内置规则缺少名称或类别: %+v", i, rule)
		}
	}

	if err := initializer.Initialize(ctx); err != nil {
		t.Fatalf("初始化内置规则失败: %v", err)
	}
	count, err := repo.CountBuiltin(ctx)
	if err != nil {
		t.Fatalf("统计内置规则失败: %v", err)
	}
	if count != int64(len(builtin)) {
		t.Fatalf("内置规则数 = %d，期望 %d", count, len(builtin))
	}

	if err := initializer.Initialize(ctx); err != nil {
		t.Fatalf("重复初始化失败: %v", err)
	}
	again, err := repo.CountBuiltin(ctx)
	if err != nil {
		t.Fatalf("统计内置规则失败: %v", err)
	}
	if again != count {
		t.Errorf("重复初始化后规则数 = %d，期望仍为 %d", again, count)
	}
}
