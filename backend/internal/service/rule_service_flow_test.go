package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"

	"gorm.io/gorm"
)

// ruleFlowTestEnv 是规则引擎流程用例的接线：真实 PostgreSQL 仓库。
type ruleFlowTestEnv struct {
	svc          RuleService
	db           *gorm.DB
	ruleRepo     repository.RuleRepository
	emailRepo    repository.EmailRepository
	matchLogRepo repository.RuleMatchLogRepository
}

func newRuleFlowTestEnv(t *testing.T) *ruleFlowTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	ruleRepo := repository.NewRuleRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	matchLogRepo := repository.NewRuleMatchLogRepository(db)

	return &ruleFlowTestEnv{
		svc:          NewRuleService(ruleRepo, emailRepo, matchLogRepo),
		db:           db,
		ruleRepo:     ruleRepo,
		emailRepo:    emailRepo,
		matchLogRepo: matchLogRepo,
	}
}

// ruleFlowTestRule 构造一条规则。
func ruleFlowTestRule(name, matchMode string, conditions model.RuleConditions, actions model.RuleActions) *model.EmailRule {
	return &model.EmailRule{
		Name:       name,
		AccountUID: "acct-rule",
		Enabled:    true,
		MatchMode:  matchMode,
		Conditions: conditions,
		Actions:    actions,
	}
}

// seedRuleFlowEmail 写入一封邮件 fixture。
func seedRuleFlowEmail(t *testing.T, db *gorm.DB, providerID string) *model.Email {
	t.Helper()

	email := &model.Email{
		ProviderID:    providerID,
		AccountUID:    "acct-rule",
		MessageID:     providerID + "@example.com",
		Subject:       "发票报销通知",
		FromAddress:   "billing@example.com",
		FromName:      "Billing",
		ToAddress:     "me@example.com",
		TextBody:      "请查收本月发票",
		HasAttachment: true,
		SentAt:        time.Now(),
		ReceivedAt:    time.Now(),
	}
	if err := db.Create(email).Error; err != nil {
		t.Fatalf("写入邮件 fixture 失败: %v", err)
	}
	return email
}

// TestRuleCreateValidatesConfiguration 覆盖创建校验：
// 名称/账户/条件/动作缺一不可，字段与操作符必须在白名单内，正则必须可编译。
func TestRuleCreateValidatesConfiguration(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()

	validCond := model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}}
	validAction := model.RuleActions{{Type: "mark_read"}}

	missingAccount := ruleFlowTestRule("规则", "all", validCond, validAction)
	missingAccount.AccountUID = ""

	cases := []struct {
		name    string
		rule    *model.EmailRule
		wantErr string
	}{
		{name: "缺少名称", rule: ruleFlowTestRule("", "all", validCond, validAction), wantErr: "rule name is required"},
		{name: "缺少账户", rule: missingAccount, wantErr: "account UID is required"},
		{name: "缺少条件", rule: ruleFlowTestRule("规则", "all", nil, validAction), wantErr: "at least one condition is required"},
		{name: "缺少动作", rule: ruleFlowTestRule("规则", "all", validCond, nil), wantErr: "at least one action is required"},
		{
			name:    "非法条件字段",
			rule:    ruleFlowTestRule("规则", "all", model.RuleConditions{{Field: "magic", Operator: "contains", Value: "x"}}, validAction),
			wantErr: "invalid condition field",
		},
		{
			name:    "非法操作符",
			rule:    ruleFlowTestRule("规则", "all", model.RuleConditions{{Field: "subject", Operator: "sounds_like", Value: "x"}}, validAction),
			wantErr: "invalid condition operator",
		},
		{
			name:    "正则不可编译",
			rule:    ruleFlowTestRule("规则", "all", model.RuleConditions{{Field: "subject", Operator: "regex", Value: "([a-z"}}, validAction),
			wantErr: "invalid regex pattern",
		},
		{
			name:    "非法动作类型",
			rule:    ruleFlowTestRule("规则", "all", validCond, model.RuleActions{{Type: "explode"}}),
			wantErr: "invalid action type",
		},
		{
			name:    "需要参数的动作缺参数",
			rule:    ruleFlowTestRule("规则", "all", validCond, model.RuleActions{{Type: "add_label"}}),
			wantErr: "requires a value",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := env.svc.Create(ctx, c.rule)
			if err == nil {
				t.Fatal("非法规则应被拒绝")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %q，期望包含 %q", err.Error(), c.wantErr)
			}
		})
	}

	var count int64
	if err := env.db.Model(&model.EmailRule{}).Count(&count).Error; err != nil {
		t.Fatalf("统计规则失败: %v", err)
	}
	if count != 0 {
		t.Errorf("被拒绝的规则仍写入了 %d 行", count)
	}
}

// TestEmailRuleCRUDLifecycle 覆盖规则生命周期与错误路径（含条件/动作的 JSON 往返）。
func TestEmailRuleCRUDLifecycle(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()

	rule := ruleFlowTestRule("发票归档", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "archive"}})
	if err := env.svc.Create(ctx, rule); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("创建后规则 ID 仍为 0")
	}

	got, err := env.svc.GetByID(ctx, rule.ID)
	if err != nil {
		t.Fatalf("查询规则失败: %v", err)
	}
	if got.Name != "发票归档" || len(got.Conditions) != 1 || got.Conditions[0].Value != "发票" {
		t.Errorf("规则条件未正确往返: %+v", got)
	}
	if len(got.Actions) != 1 || got.Actions[0].Type != "archive" {
		t.Errorf("规则动作未正确往返: %+v", got.Actions)
	}

	rules, err := env.svc.ListByAccount(ctx, "acct-rule")
	if err != nil {
		t.Fatalf("按账户查询规则失败: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("账户规则数 = %d，期望 1", len(rules))
	}

	rule.Name = "发票归档（改）"
	if err := env.svc.Update(ctx, rule); err != nil {
		t.Fatalf("更新规则失败: %v", err)
	}
	got, _ = env.svc.GetByID(ctx, rule.ID)
	if got.Name != "发票归档（改）" {
		t.Errorf("更新未生效: %+v", got)
	}

	if err := env.svc.Delete(ctx, rule.ID); err != nil {
		t.Fatalf("删除规则失败: %v", err)
	}
	if _, err := env.svc.GetByID(ctx, rule.ID); err == nil {
		t.Error("删除后仍能查到规则")
	}
	if err := env.svc.Delete(ctx, rule.ID); err == nil {
		t.Error("删除不存在的规则应报错")
	}
	if _, err := env.svc.GetByID(ctx, 999999); err == nil {
		t.Error("查询不存在的规则应报错")
	}
}

// TestMatchRuleModes 覆盖 all / any 两种匹配模式与边界。
func TestMatchRuleModes(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	email := seedRuleFlowEmail(t, env.db, "p-match")

	subjectHit := model.RuleCondition{Field: "subject", Operator: "contains", Value: "发票"}
	fromMiss := model.RuleCondition{Field: "from", Operator: "contains", Value: "nobody@example.com"}

	cases := []struct {
		name       string
		conditions model.RuleConditions
		matchMode  string
		want       bool
	}{
		{name: "all 全部命中", conditions: model.RuleConditions{subjectHit, subjectHit}, matchMode: "all", want: true},
		{name: "all 有未命中", conditions: model.RuleConditions{subjectHit, fromMiss}, matchMode: "all", want: false},
		{name: "any 任一命中", conditions: model.RuleConditions{subjectHit, fromMiss}, matchMode: "any", want: true},
		{name: "any 全部未命中", conditions: model.RuleConditions{fromMiss}, matchMode: "any", want: false},
		{name: "无条件不匹配", conditions: nil, matchMode: "all", want: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rule := ruleFlowTestRule("匹配", c.matchMode, c.conditions, model.RuleActions{{Type: "mark_read"}})
			matched, err := env.svc.TestRule(ctx, rule, email)
			if err != nil {
				t.Fatalf("匹配失败: %v", err)
			}
			if matched != c.want {
				t.Errorf("matched = %v，期望 %v", matched, c.want)
			}
		})
	}
}

// TestMatchConditionOperators 覆盖全部操作符与字段取值（含错误路径）。
func TestMatchConditionOperators(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	email := &model.Email{
		FromAddress:   "Billing@Example.com",
		ToAddress:     "me@example.com",
		Subject:       "Invoice 发票通知",
		TextBody:      "请查收本月发票",
		HasAttachment: true,
	}

	cases := []struct {
		name      string
		condition model.RuleCondition
		want      bool
		wantErr   bool
	}{
		{name: "from 命中", condition: model.RuleCondition{Field: "from", Operator: "contains", Value: "billing"}, want: true},
		{name: "to 命中", condition: model.RuleCondition{Field: "to", Operator: "contains", Value: "me@"}, want: true},
		{name: "subject 忽略大小写", condition: model.RuleCondition{Field: "subject", Operator: "contains", Value: "invoice"}, want: true},
		{name: "body 命中", condition: model.RuleCondition{Field: "body", Operator: "contains", Value: "发票"}, want: true},
		{name: "not_contains 命中", condition: model.RuleCondition{Field: "subject", Operator: "not_contains", Value: "中奖"}, want: true},
		{name: "equals 忽略大小写", condition: model.RuleCondition{Field: "from", Operator: "equals", Value: "billing@example.com"}, want: true},
		{name: "not_equals 命中", condition: model.RuleCondition{Field: "from", Operator: "not_equals", Value: "other@example.com"}, want: true},
		{name: "starts_with 命中", condition: model.RuleCondition{Field: "subject", Operator: "starts_with", Value: "invoice"}, want: true},
		{name: "ends_with 命中", condition: model.RuleCondition{Field: "subject", Operator: "ends_with", Value: "通知"}, want: true},
		{name: "regex 未命中", condition: model.RuleCondition{Field: "subject", Operator: "regex", Value: `\d+`}, want: false},
		{name: "regex 命中", condition: model.RuleCondition{Field: "body", Operator: "regex", Value: `本月`}, want: true},
		{name: "has_attachment 为真", condition: model.RuleCondition{Field: "has_attachment", Operator: "equals", Value: "true"}, want: true},
		{name: "has_attachment 为假", condition: model.RuleCondition{Field: "has_attachment", Operator: "equals", Value: "false"}, want: false},
		{name: "未知字段报错", condition: model.RuleCondition{Field: "magic", Operator: "contains", Value: "x"}, wantErr: true},
		{name: "未知操作符报错", condition: model.RuleCondition{Field: "subject", Operator: "sounds_like", Value: "x"}, wantErr: true},
		{name: "非法正则报错", condition: model.RuleCondition{Field: "subject", Operator: "regex", Value: "([a-z"}, wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rule := ruleFlowTestRule("匹配", "all", model.RuleConditions{c.condition}, model.RuleActions{{Type: "mark_read"}})
			matched, err := env.svc.TestRule(context.Background(), rule, email)
			if c.wantErr {
				if err == nil {
					t.Fatal("应返回 error")
				}
				return
			}
			if err != nil {
				t.Fatalf("匹配失败: %v", err)
			}
			if matched != c.want {
				t.Errorf("matched = %v，期望 %v", matched, c.want)
			}
		})
	}
}

// TestApplyRulesExecutesActionsAndLogsMatches 覆盖规则应用主路径：
// 命中后动作落库、命中计数递增、命中记录写入（回答「这封邮件为什么被自动处理」）。
func TestApplyRulesExecutesActionsAndLogsMatches(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	email := seedRuleFlowEmail(t, env.db, "p-apply")

	rule := ruleFlowTestRule("命中并处理", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "mark_read"}, {Type: "star"}, {Type: "add_label", Value: "财务"}})
	if err := env.svc.Create(ctx, rule); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	if err := env.svc.ApplyRules(ctx, email); err != nil {
		t.Fatalf("应用规则失败: %v", err)
	}

	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if !stored.IsRead || !stored.IsStarred {
		t.Errorf("动作未落库: is_read=%v is_starred=%v", stored.IsRead, stored.IsStarred)
	}
	if labels := parseEmailLabels(stored.Labels); len(labels) != 1 || labels[0] != "财务" {
		t.Errorf("标签动作未落库: %v", labels)
	}

	reloaded, err := env.ruleRepo.GetByID(ctx, rule.ID)
	if err != nil {
		t.Fatalf("读取规则失败: %v", err)
	}
	if reloaded.MatchedCount != 1 {
		t.Errorf("MatchedCount = %d，期望 1", reloaded.MatchedCount)
	}

	logs, err := env.svc.GetEmailRuleMatches(ctx, email.ID)
	if err != nil {
		t.Fatalf("查询命中记录失败: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("命中记录数 = %d，期望 1", len(logs))
	}
	if logs[0].RuleID != rule.ID || logs[0].RuleName != "命中并处理" {
		t.Errorf("命中记录字段错误: %+v", logs[0])
	}
	var actions model.RuleActions
	if err := json.Unmarshal([]byte(logs[0].Actions), &actions); err != nil {
		t.Fatalf("命中记录动作不是合法 JSON: %q", logs[0].Actions)
	}
	if len(actions) != 3 {
		t.Errorf("命中记录动作数 = %d，期望 3", len(actions))
	}
}

// TestApplyRulesSkipsDisabledAndUnmatched 覆盖跳过语义：
// 未启用或未命中的规则不执行动作、不写命中记录。
func TestApplyRulesSkipsDisabledAndUnmatched(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	email := seedRuleFlowEmail(t, env.db, "p-skip")

	disabled := ruleFlowTestRule("已禁用", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "mark_read"}})
	if err := env.svc.Create(ctx, disabled); err != nil {
		t.Fatalf("创建禁用��则失败: %v", err)
	}
	// Enabled 列带 `default:true`，GORM 会把 false 当成「未设置」而写回默认值 true，
	// 因此这里必须显式落库，否则测的不是「已禁用」的规则。
	if err := env.db.Model(&model.EmailRule{}).Where("id = ?", disabled.ID).
		Update("enabled", false).Error; err != nil {
		t.Fatalf("禁用规则失败: %v", err)
	}

	unmatched := ruleFlowTestRule("不命中", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "中奖"}},
		model.RuleActions{{Type: "star"}})
	if err := env.svc.Create(ctx, unmatched); err != nil {
		t.Fatalf("创建不命中规则失败: %v", err)
	}

	if err := env.svc.ApplyRules(ctx, email); err != nil {
		t.Fatalf("应用规则失败: %v", err)
	}

	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if stored.IsRead || stored.IsStarred {
		t.Errorf("未命中的规则被执行了: is_read=%v is_starred=%v", stored.IsRead, stored.IsStarred)
	}

	logs, err := env.svc.GetEmailRuleMatches(ctx, email.ID)
	if err != nil {
		t.Fatalf("查询命中记录失败: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("未命中却写了 %d 条命中记录", len(logs))
	}
}

// TestApplyRulesStopProcessing 覆盖 StopProcessing：
// 命中后必须停止后续规则，否则后面的规则会覆盖前面的处理结果。
func TestApplyRulesStopProcessing(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	email := seedRuleFlowEmail(t, env.db, "p-stop")

	first := ruleFlowTestRule("先执行并停止", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "star"}})
	first.Priority = 1
	first.StopProcessing = true
	if err := env.svc.Create(ctx, first); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	second := ruleFlowTestRule("后续规则", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "mark_read"}})
	second.Priority = 2
	if err := env.svc.Create(ctx, second); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	if err := env.svc.ApplyRules(ctx, email); err != nil {
		t.Fatalf("应用规则失败: %v", err)
	}

	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if !stored.IsStarred {
		t.Error("第一条规则的动作未执行")
	}
	if stored.IsRead {
		t.Error("StopProcessing 未生效：后续规则仍被执行")
	}
}

// TestLabelActionsAreIdempotent 覆盖标签动作在真实落库下的幂等性：
// 重复添加不产生重复标签，移除不存在的标签不影响现有标签。
func TestLabelActionsAreIdempotent(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	email := seedRuleFlowEmail(t, env.db, "p-label")

	// ListByAccount 的排序是 `priority ASC, created_at DESC`：同优先级时后建的规则先执行。
	// 这里必须给出显式优先级，否则「移除」会先于「添加」执行，断言失去意义。
	add := ruleFlowTestRule("打标签", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "add_label", Value: "财务"}})
	add.Priority = 1
	if err := env.svc.Create(ctx, add); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := env.svc.ApplyRules(ctx, email); err != nil {
			t.Fatalf("第 %d 次应用规则失败: %v", i+1, err)
		}
	}
	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if labels := parseEmailLabels(stored.Labels); len(labels) != 1 {
		t.Errorf("重复添加产生了 %d 个标签: %v", len(labels), labels)
	}

	removeMissing := ruleFlowTestRule("移除不存在的标签", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "remove_label", Value: "不存在"}})
	removeMissing.Priority = 2
	if err := env.svc.Create(ctx, removeMissing); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}
	if err := env.svc.ApplyRules(ctx, email); err != nil {
		t.Fatalf("应用规则失败: %v", err)
	}
	stored, _ = env.emailRepo.FindByID(ctx, email.ID)
	if labels := parseEmailLabels(stored.Labels); len(labels) != 1 || labels[0] != "财务" {
		t.Errorf("移除不存在的标签影响了现有标签: %v", labels)
	}

	remove := ruleFlowTestRule("移除标���", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "remove_label", Value: "财务"}})
	remove.Priority = 3
	if err := env.svc.Create(ctx, remove); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}
	if err := env.svc.ApplyRules(ctx, email); err != nil {
		t.Fatalf("应用规则失败: %v", err)
	}
	stored, _ = env.emailRepo.FindByID(ctx, email.ID)
	if labels := parseEmailLabels(stored.Labels); len(labels) != 0 {
		t.Errorf("标签未被移除: %v", labels)
	}
}

// TestExecuteActionCoversAllTypes 覆盖全部动作类型对邮件字段的真实影响。
func TestExecuteActionCoversAllTypes(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	svc := env.svc.(*ruleService)

	cases := []struct {
		name   string
		action model.RuleAction
		check  func(t *testing.T, email *model.Email)
	}{
		{name: "mark_unread", action: model.RuleAction{Type: "mark_unread"}, check: func(t *testing.T, e *model.Email) {
			if e.IsRead {
				t.Error("mark_unread 未生效")
			}
		}},
		{name: "unstar", action: model.RuleAction{Type: "unstar"}, check: func(t *testing.T, e *model.Email) {
			if e.IsStarred {
				t.Error("unstar 未生效")
			}
		}},
		{name: "archive", action: model.RuleAction{Type: "archive"}, check: func(t *testing.T, e *model.Email) {
			if !e.IsArchived {
				t.Error("archive 未生效")
			}
		}},
		{name: "delete", action: model.RuleAction{Type: "delete"}, check: func(t *testing.T, e *model.Email) {
			if !e.IsDeleted {
				t.Error("delete 未生效")
			}
		}},
		{name: "move_folder", action: model.RuleAction{Type: "move_folder", Value: "财务"}, check: func(t *testing.T, e *model.Email) {
			if e.Folder != "财务" {
				t.Errorf("move_folder 未生效: %q", e.Folder)
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			email := seedRuleFlowEmail(t, env.db, "p-action-"+c.name)
			// 先置反，确保断言的是动作的效果而不是默认值
			email.IsRead = true
			email.IsStarred = true
			if err := env.emailRepo.Update(ctx, email); err != nil {
				t.Fatalf("准备邮件状态失败: %v", err)
			}

			if err := svc.executeAction(ctx, &c.action, email); err != nil {
				t.Fatalf("执行动作失败: %v", err)
			}

			stored, err := env.emailRepo.FindByID(ctx, email.ID)
			if err != nil || stored == nil {
				t.Fatalf("读取邮件失败: %v", err)
			}
			c.check(t, stored)
		})
	}

	email := seedRuleFlowEmail(t, env.db, "p-action-unknown")
	if err := svc.executeAction(ctx, &model.RuleAction{Type: "explode"}, email); err == nil {
		t.Error("未知动作类型应报错")
	}
}

// TestExecuteActionsContinuesAfterFailure 覆盖动作容错：
// 单个动作失败不阻断后续动作（但要留痕）。
func TestExecuteActionsContinuesAfterFailure(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	email := seedRuleFlowEmail(t, env.db, "p-action-fail")

	rule := ruleFlowTestRule("含失败动作", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{
			{Type: "add_label"}, // 缺参数 → 失败
			{Type: "star"},      // 仍应执行
		})

	if err := env.svc.(*ruleService).executeActions(ctx, rule, email); err != nil {
		t.Fatalf("动作失败不应让整体报错: %v", err)
	}

	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if !stored.IsStarred {
		t.Error("前一个动作失败后，后续动作未执行")
	}
}

// TestApplyRulesToEmail 覆盖按 ID 应用规则：邮件不存在时报错。
func TestApplyRulesToEmail(t *testing.T) {
	env := newRuleFlowTestEnv(t)
	ctx := context.Background()
	email := seedRuleFlowEmail(t, env.db, "p-by-id")

	rule := ruleFlowTestRule("按 ID 应用", "all",
		model.RuleConditions{{Field: "subject", Operator: "contains", Value: "发票"}},
		model.RuleActions{{Type: "star"}})
	if err := env.svc.Create(ctx, rule); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	if err := env.svc.ApplyRulesToEmail(ctx, email.ID, "acct-rule"); err != nil {
		t.Fatalf("按 ID 应用规则失败: %v", err)
	}
	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if !stored.IsStarred {
		t.Error("按 ID 应用规则未生效")
	}

	if err := env.svc.ApplyRulesToEmail(ctx, 999999, "acct-rule"); err == nil {
		t.Error("邮件不存在时应报错")
	}
}

// TestGetEmailRuleMatchesWithoutRepo 覆盖依赖缺省：没有命中记录仓库时返回空列表。
func TestGetEmailRuleMatchesWithoutRepo(t *testing.T) {
	svc := NewRuleService(nil, nil, nil)

	matches, err := svc.GetEmailRuleMatches(context.Background(), 1)
	if err != nil {
		t.Fatalf("缺少仓库时不应报错: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("应返回空列表，实际 %d 条", len(matches))
	}
}

// TestActionsToJSON 覆盖动作序列化：空动作返回 "[]"，非空返回合法 JSON。
func TestActionsToJSON(t *testing.T) {
	if got := actionsToJSON(nil); got != "[]" {
		t.Errorf("空动作 = %q，期望 []", got)
	}

	got := actionsToJSON(model.RuleActions{{Type: "star"}, {Type: "add_label", Value: "财务"}})
	var actions model.RuleActions
	if err := json.Unmarshal([]byte(got), &actions); err != nil {
		t.Fatalf("序列化结果不是合法 JSON: %q", got)
	}
	if len(actions) != 2 || actions[1].Value != "财务" {
		t.Errorf("序列化结果错误: %+v", actions)
	}
}
