package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"

	"gorm.io/gorm"
)

// schedulerTestEnv 把调度器用例需要的接线一次性给全。
type schedulerTestEnv struct {
	scheduler   *SyncScheduler
	accountRepo *stubSyncAccountRepo
	runner      *fakeAccountSyncRunner
	db          *gorm.DB
}

func newSchedulerTestEnv(t *testing.T, accounts ...*model.EmailAccount) *schedulerTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	accountRepo := newStubSyncAccountRepo(accounts...)
	runner := newFakeAccountSyncRunner()
	scheduler := NewSyncScheduler(
		accountRepo,
		repository.NewSyncLogRepository(db),
		nil, // syncLock：不依赖 Redis
		NewSyncProgressRegistry(),
		runner,
	)
	return &schedulerTestEnv{scheduler: scheduler, accountRepo: accountRepo, runner: runner, db: db}
}

func syncTestAccount(uid string) *model.EmailAccount {
	return &model.EmailAccount{
		UID:          uid,
		Email:        uid + "@example.com",
		Status:       "active",
		SyncEnabled:  true,
		SyncInterval: 2,
	}
}

// TestSyncAllAccountsTriggersRunnerForEveryAccount 覆盖手动全量同步：
// 每个启用账户都必须被触发一次。
func TestSyncAllAccountsTriggersRunnerForEveryAccount(t *testing.T) {
	env := newSchedulerTestEnv(t, syncTestAccount("a-1"), syncTestAccount("a-2"), syncTestAccount("a-3"))

	if err := env.scheduler.SyncAllAccounts(context.Background()); err != nil {
		t.Fatalf("SyncAllAccounts 失败: %v", err)
	}
	env.runner.waitForCalls(t, 3)

	got := env.runner.calledUIDs()
	if len(got) != 3 {
		t.Fatalf("触发次数 = %d，期望 3: %v", len(got), got)
	}
	seen := map[string]int{}
	for _, uid := range got {
		seen[uid]++
	}
	for _, uid := range []string{"a-1", "a-2", "a-3"} {
		if seen[uid] != 1 {
			t.Errorf("账户 %s 触发 %d 次，期望 1 次", uid, seen[uid])
		}
	}
}

// TestSyncAllAccountsReturnsErrorWhenListFails 覆盖错误路径：
// 账户列表查询失败必须返回错误（而不是静默当成 0 个账户）。
func TestSyncAllAccountsReturnsErrorWhenListFails(t *testing.T) {
	env := newSchedulerTestEnv(t)
	env.accountRepo.listErr = errors.New("db is down")

	err := env.scheduler.SyncAllAccounts(context.Background())
	if err == nil {
		t.Fatal("账户列表查询失败时应返回 error")
	}
	if len(env.runner.calledUIDs()) != 0 {
		t.Error("列表查询失败却触发了同步")
	}
}

// TestSyncAllAccountsWithoutAccountsIsNoop 覆盖边界：没有账户时是空操作。
func TestSyncAllAccountsWithoutAccountsIsNoop(t *testing.T) {
	env := newSchedulerTestEnv(t)

	if err := env.scheduler.SyncAllAccounts(context.Background()); err != nil {
		t.Fatalf("空账户列表不应报错: %v", err)
	}
	// SyncAllAccounts 起 goroutine 后立即返回，给后台一个调度窗口再断言
	time.Sleep(50 * time.Millisecond)
	if n := len(env.runner.calledUIDs()); n != 0 {
		t.Errorf("空账户列表触发了 %d 次同步", n)
	}
}

// TestCheckAndSyncAccountsOnlySyncsDueAccounts 覆盖定时调度的核心判断：
// 只有「从未同步」「已到间隔」的账户被触发；webhook 模式账户跳过。
func TestCheckAndSyncAccountsOnlySyncsDueAccounts(t *testing.T) {
	now := time.Now()

	neverSynced := syncTestAccount("due-never")
	neverSynced.LastSyncAt = nil

	due := syncTestAccount("due-interval")
	lastSync := now.Add(-10 * time.Minute)
	due.LastSyncAt = &lastSync

	notDue := syncTestAccount("not-due")
	recent := now.Add(-30 * time.Second)
	notDue.LastSyncAt = &recent

	webhook := syncTestAccount("webhook-mode")
	webhook.SyncModeField = model.SyncModeWebhook

	env := newSchedulerTestEnv(t, neverSynced, due, notDue, webhook)

	env.scheduler.checkAndSyncAccounts(context.Background())
	env.runner.waitForCalls(t, 2)

	got := env.runner.calledUIDs()
	seen := map[string]bool{}
	for _, uid := range got {
		seen[uid] = true
	}
	if !seen["due-never"] {
		t.Error("从未同步的账户没有被触发")
	}
	if !seen["due-interval"] {
		t.Error("已到同步间隔的账户没有被触发")
	}
	if seen["not-due"] {
		t.Error("未到同步间隔的账户被误触发")
	}
	if seen["webhook-mode"] {
		t.Error("webhook 模式账户不应参与轮询同步")
	}
}

// TestShouldSyncBoundaries 覆盖间隔计算的边界（纯函数，无依赖）。
func TestShouldSyncBoundaries(t *testing.T) {
	env := newSchedulerTestEnv(t)
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		account *model.EmailAccount
		want    bool
	}{
		{
			name: "从未同步过",
			account: func() *model.EmailAccount {
				a := syncTestAccount("never")
				a.LastSyncAt = nil
				return a
			}(),
			want: true,
		},
		{
			name: "恰好到达间隔",
			account: func() *model.EmailAccount {
				a := syncTestAccount("exact")
				at := now.Add(-2 * time.Minute)
				a.LastSyncAt = &at
				return a
			}(),
			want: true,
		},
		{
			name: "间隔未到",
			account: func() *model.EmailAccount {
				a := syncTestAccount("early")
				at := now.Add(-time.Minute)
				a.LastSyncAt = &at
				return a
			}(),
			want: false,
		},
		{
			name: "间隔为 0 时按默认 2 分钟",
			account: func() *model.EmailAccount {
				a := syncTestAccount("zero-interval")
				a.SyncInterval = 0
				at := now.Add(-90 * time.Second)
				a.LastSyncAt = &at
				return a
			}(),
			want: false,
		},
		{
			name: "webhook 模式恒不同步",
			account: func() *model.EmailAccount {
				a := syncTestAccount("webhook")
				a.SyncModeField = model.SyncModeWebhook
				a.LastSyncAt = nil
				return a
			}(),
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := env.scheduler.shouldSync(c.account, now); got != c.want {
				t.Errorf("shouldSync = %v，期望 %v", got, c.want)
			}
		})
	}
}

// TestCleanupStaleSyncLogsMarksStaleAsFailed 覆盖卡住任务清理：
// 超过 maxAge 仍 running 的日志标 failed，账户同步状态同步更新，新任务不受影响。
func TestCleanupStaleSyncLogsMarksStaleAsFailed(t *testing.T) {
	env := newSchedulerTestEnv(t, syncTestAccount("stuck-account"))

	stale := &model.SyncLog{
		AccountUID: "stuck-account",
		SyncType:   "manual",
		Status:     "running",
		StartedAt:  time.Now().Add(-2 * time.Hour),
	}
	fresh := &model.SyncLog{
		AccountUID: "fresh-account",
		SyncType:   "manual",
		Status:     "running",
		StartedAt:  time.Now(),
	}
	for _, log := range []*model.SyncLog{stale, fresh} {
		if err := env.scheduler.syncLogRepo.Create(context.Background(), log); err != nil {
			t.Fatalf("写入同步日志失败: %v", err)
		}
	}

	cleaned, err := env.scheduler.CleanupStaleSyncLogs(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("清理卡住任务失败: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("清理数量 = %d，期望 1", cleaned)
	}

	logs := loadSyncLogs(t, env.db)
	byUID := map[string]model.SyncLog{}
	for _, l := range logs {
		byUID[l.AccountUID] = l
	}
	if got := byUID["stuck-account"]; got.Status != "failed" || got.ErrorMessage == "" || got.CompletedAt == nil {
		t.Errorf("卡住日志未被正确标记: status=%q err=%q completedAt=%v", got.Status, got.ErrorMessage, got.CompletedAt)
	}
	if got := byUID["fresh-account"]; got.Status != "running" {
		t.Errorf("新任务被误清理: status=%q", got.Status)
	}

	update, ok := env.accountRepo.lastStatusFor("stuck-account")
	if !ok {
		t.Fatal("清理后没有更新账户同步状态")
	}
	if update.Status != "failed" || update.Message == "" {
		t.Errorf("账户同步状态 = %+v，期望 failed 且带原因", update)
	}
}

// TestCleanupStaleSyncLogsWithoutStaleLogs 覆盖边界：没有卡住任务时返回 0 且不报错。
func TestCleanupStaleSyncLogsWithoutStaleLogs(t *testing.T) {
	env := newSchedulerTestEnv(t)

	cleaned, err := env.scheduler.CleanupStaleSyncLogs(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("无卡住任务时不应报错: %v", err)
	}
	if cleaned != 0 {
		t.Errorf("清理数量 = %d，期望 0", cleaned)
	}
}

// TestStopSchedulerIsIdempotent 覆盖停止调度器的边界：
// 未启动时停止应无副作用，重复停止不得 panic（close 已关闭 channel）。
func TestStopSchedulerIsIdempotent(t *testing.T) {
	env := newSchedulerTestEnv(t)

	if err := env.scheduler.StopScheduler(); err != nil {
		t.Fatalf("未启动时停止调度器失败: %v", err)
	}

	if err := env.scheduler.StartScheduler(context.Background()); err != nil {
		t.Fatalf("启动调度器失败: %v", err)
	}
	if err := env.scheduler.StopScheduler(); err != nil {
		t.Fatalf("停止调度器失败: %v", err)
	}
	if err := env.scheduler.StopScheduler(); err != nil {
		t.Fatalf("重复停止调度器失败: %v", err)
	}
}
