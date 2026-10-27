package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// TestSyncExecutorWebAPIPersistsEmails 覆盖 WebAPI 同步主路径：
// 上游 2 封 → 落库 2 行 + 同步日志计数 + 账户最后同步时间与状态更新。
func TestSyncExecutorWebAPIPersistsEmails(t *testing.T) {
	account := testWebAPIAccount(t, `{"base_url":"http://127.0.0.1:1","jwt_token":"t"}`)
	provider := &fakeWebAPIProvider{
		emails: []*adapter.Email{
			webAPITestEmail("p-1", "第一封", time.Now().Add(-2*time.Hour)),
			webAPITestEmail("p-2", "第二封", time.Now().Add(-time.Hour)),
		},
	}
	env := newSyncExecutorTestEnv(t, account, provider)

	if err := env.executor.SyncAccount(context.Background(), account.UID); err != nil {
		t.Fatalf("WebAPI 同步失败: %v", err)
	}

	emails := loadEmails(t, env.db)
	if len(emails) != 2 {
		t.Fatalf("emails 行数 = %d，期望 2", len(emails))
	}
	for _, email := range emails {
		if email.AccountUID != account.UID {
			t.Errorf("邮件 %d 的 AccountUID = %q", email.ID, email.AccountUID)
		}
		if email.DedupeKey == "" {
			t.Errorf("邮件 %d 缺少去重标识", email.ID)
		}
		if email.ToAddresses == "" {
			t.Errorf("邮件 %d 缺少收件人 JSON", email.ID)
		}
	}

	logs := loadSyncLogs(t, env.db)
	if len(logs) != 1 {
		t.Fatalf("sync_logs 行数 = %d，期望 1", len(logs))
	}
	log := logs[0]
	if log.Status != "success" {
		t.Errorf("同步日志状态 = %q，期望 success（err=%q）", log.Status, log.ErrorMessage)
	}
	if log.EmailsNew != 2 || log.EmailsFetched != 2 {
		t.Errorf("同步日志计数 new=%d fetched=%d，期望 2/2", log.EmailsNew, log.EmailsFetched)
	}
	if !log.IsFirstSync {
		t.Error("首次同步日志应标记 IsFirstSync")
	}
	if log.CompletedAt == nil {
		t.Error("同步日志缺少完成时间")
	}

	if account.LastSyncAt == nil {
		t.Error("同步成功后未更新账户最后同步时间")
	}
	update, ok := env.accountRepo.lastStatusFor(account.UID)
	if !ok || update.Status != "success" {
		t.Errorf("账户同步状态 = %+v，期望 success", update)
	}
	if provider.connected {
		t.Error("同步结束后适配器未断开连接")
	}
	if len(provider.fetchedSince) != 1 {
		t.Fatalf("FetchEmails 调用 %d 次，期望 1 次", len(provider.fetchedSince))
	}
	if !provider.fetchedSince[0].IsZero() {
		t.Errorf("首次 WebAPI 同步应为全量（since 零值），实际 %v", provider.fetchedSince[0])
	}
}

// TestSyncExecutorWebAPIDedupesOnSecondRun 覆盖增量同步：
// 同一批邮件第二次同步应更新而不是重复插入，且 since 变成增量窗口。
func TestSyncExecutorWebAPIDedupesOnSecondRun(t *testing.T) {
	account := testWebAPIAccount(t, `{"base_url":"http://127.0.0.1:1","jwt_token":"t"}`)
	provider := &fakeWebAPIProvider{
		emails: []*adapter.Email{
			webAPITestEmail("p-1", "第一封", time.Now().Add(-2*time.Hour)),
			webAPITestEmail("p-2", "第二封", time.Now().Add(-time.Hour)),
		},
	}
	env := newSyncExecutorTestEnv(t, account, provider)

	if err := env.executor.SyncAccount(context.Background(), account.UID); err != nil {
		t.Fatalf("首次同步失败: %v", err)
	}
	if err := env.executor.SyncAccount(context.Background(), account.UID); err != nil {
		t.Fatalf("第二次同步失败: %v", err)
	}

	if emails := loadEmails(t, env.db); len(emails) != 2 {
		t.Fatalf("重复同步后 emails 行数 = %d，期望仍为 2（去重失效）", len(emails))
	}

	logs := loadSyncLogs(t, env.db)
	if len(logs) != 2 {
		t.Fatalf("sync_logs 行数 = %d，期望 2", len(logs))
	}
	second := logs[1]
	if second.EmailsNew != 0 || second.EmailsUpdated != 2 {
		t.Errorf("第二次同步计数 new=%d updated=%d，期望 0/2", second.EmailsNew, second.EmailsUpdated)
	}
	if second.IsFirstSync {
		t.Error("第二次同步不应标记为首次同步")
	}

	if len(provider.fetchedSince) != 2 {
		t.Fatalf("FetchEmails 调用 %d 次，期望 2 次", len(provider.fetchedSince))
	}
	if provider.fetchedSince[1].IsZero() {
		t.Error("第二次同步应使用增量时间窗口，实际是零值（退化成全量）")
	}
}

// TestSyncExecutorWebAPIFetchFailureMarksSyncLogFailed 覆盖上游故障路径：
// 拉取失败必须留下 failed 日志与账户失败状态（成功但无变化的日志会被删掉，
// 失败日志必须保留，否则故障无迹可查）。
func TestSyncExecutorWebAPIFetchFailureMarksSyncLogFailed(t *testing.T) {
	account := testWebAPIAccount(t, `{"base_url":"http://127.0.0.1:1","jwt_token":"t"}`)
	provider := &fakeWebAPIProvider{fetchErr: errors.New("上游 500")}
	env := newSyncExecutorTestEnv(t, account, provider)

	err := env.executor.SyncAccount(context.Background(), account.UID)
	if err == nil {
		t.Fatal("上游拉取失败时应返回 error")
	}
	if !strings.Contains(err.Error(), "拉取邮件失败") {
		t.Errorf("error = %q，期望点名拉取失败", err.Error())
	}

	if emails := loadEmails(t, env.db); len(emails) != 0 {
		t.Errorf("拉取失败却落库了 %d 封邮件", len(emails))
	}

	logs := loadSyncLogs(t, env.db)
	if len(logs) != 1 {
		t.Fatalf("sync_logs 行数 = %d，期望 1", len(logs))
	}
	if logs[0].Status != "failed" {
		t.Errorf("同步日志状态 = %q，期望 failed", logs[0].Status)
	}
	if !strings.Contains(logs[0].ErrorMessage, "上游 500") {
		t.Errorf("同步日志错误信息 = %q，未包含上游原因", logs[0].ErrorMessage)
	}

	update, ok := env.accountRepo.lastStatusFor(account.UID)
	if !ok || update.Status != "failed" {
		t.Errorf("账户同步状态 = %+v，期望 failed", update)
	}
}

// TestSyncExecutorWebAPIDecryptFailureFailsClosed 覆盖凭据解密失败：
// fail-closed —— 不得把密文当明文继续同步（换密钥/数据损坏后必须暴露）。
func TestSyncExecutorWebAPIDecryptFailureFailsClosed(t *testing.T) {
	account := testWebAPIAccount(t, `{"base_url":"http://127.0.0.1:1","jwt_token":"t"}`)
	account.EncryptedCredentials = "this-is-not-a-valid-ciphertext"

	provider := &fakeWebAPIProvider{}
	env := newSyncExecutorTestEnv(t, account, provider)

	err := env.executor.SyncAccount(context.Background(), account.UID)
	if err == nil {
		t.Fatal("凭据解密失败时应返回 error")
	}
	if !strings.Contains(err.Error(), "解密") {
		t.Errorf("error = %q，期望点名解密失败", err.Error())
	}
	if provider.connected {
		t.Error("解密失败却连接了上游适配器")
	}
	if emails := loadEmails(t, env.db); len(emails) != 0 {
		t.Errorf("解密失败却落库了 %d 封邮件", len(emails))
	}
}

// TestSyncExecutorWebAPIWebhookOnlyConfigSkips 覆盖脏数据兜底：
// 只有 webhook_secret、没有 base_url 的账户不得进入轮询适配器创建。
func TestSyncExecutorWebAPIWebhookOnlyConfigSkips(t *testing.T) {
	account := testWebAPIAccount(t, `{"webhook_secret":"s"}`)
	provider := &fakeWebAPIProvider{}
	env := newSyncExecutorTestEnv(t, account, provider)

	if err := env.executor.SyncAccount(context.Background(), account.UID); err != nil {
		t.Fatalf("webhook-only 账户应静默跳过，实际报错: %v", err)
	}
	if provider.connected {
		t.Error("webhook-only 账户不应连接轮询适配器")
	}
	if emails := loadEmails(t, env.db); len(emails) != 0 {
		t.Errorf("webhook-only 账户落库了 %d 封邮件", len(emails))
	}

	logs := loadSyncLogs(t, env.db)
	if len(logs) != 1 || logs[0].Status != "success" || logs[0].EmailsNew != 0 {
		t.Errorf("webhook-only 跳过应留下 success 且 0 新邮件的日志，实际 %+v", logs)
	}
}

// TestSyncExecutorRejectsAccountNotEligibleForSync 覆盖前置校验边界：
// 非 active、未启用同步、webhook 模式的账户都不应触发上游拉取。
func TestSyncExecutorRejectsAccountNotEligibleForSync(t *testing.T) {
	cases := []struct {
		name        string
		mutate      func(*model.EmailAccount)
		wantErr     string
		wantSyncLog bool
	}{
		{
			name:        "账户被禁用",
			mutate:      func(a *model.EmailAccount) { a.Status = "disabled" },
			wantErr:     "is not active",
			wantSyncLog: false,
		},
		{
			name:        "未启用同步",
			mutate:      func(a *model.EmailAccount) { a.SyncEnabled = false },
			wantErr:     "sync is disabled",
			wantSyncLog: false,
		},
		{
			name:        "webhook 模式账户",
			mutate:      func(a *model.EmailAccount) { a.SyncModeField = model.SyncModeWebhook },
			wantErr:     "",
			wantSyncLog: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			account := testWebAPIAccount(t, `{"base_url":"http://127.0.0.1:1","jwt_token":"t"}`)
			c.mutate(account)

			provider := &fakeWebAPIProvider{emails: []*adapter.Email{webAPITestEmail("p-1", "不应被拉取", time.Now())}}
			env := newSyncExecutorTestEnv(t, account, provider)

			err := env.executor.SyncAccount(context.Background(), account.UID)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("应静默跳过，实际报错: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("error = %v，期望包含 %q", err, c.wantErr)
			}

			if len(provider.fetchedSince) != 0 {
				t.Error("不合格账户仍触发了上游拉取")
			}
			if emails := loadEmails(t, env.db); len(emails) != 0 {
				t.Errorf("不合格账户落库了 %d 封邮件", len(emails))
			}
			logs := loadSyncLogs(t, env.db)
			if c.wantSyncLog && len(logs) == 0 {
				t.Error("期望留下同步日志，实际没有")
			}
			if !c.wantSyncLog && len(logs) != 0 {
				t.Errorf("不应留下同步日志，实际 %d 条", len(logs))
			}
		})
	}
}

// TestSyncExecutorWebAPIAdapterCreationFailure 覆盖适配器创建失败路径。
func TestSyncExecutorWebAPIAdapterCreationFailure(t *testing.T) {
	account := testWebAPIAccount(t, `{"base_url":"http://127.0.0.1:1","jwt_token":"t"}`)
	env := newSyncExecutorTestEnv(t, account, &fakeWebAPIProvider{})

	setTestWebAPIProvider(nil, errors.New("不支持的服务类型"))
	t.Cleanup(func() { setTestWebAPIProvider(nil, nil) })

	err := env.executor.SyncAccount(context.Background(), account.UID)
	if err == nil {
		t.Fatal("适配器创建失败时应返回 error")
	}
	if !strings.Contains(err.Error(), "创建 WebAPI 适配器失败") {
		t.Errorf("error = %q，期望点名适配器创建失败", err.Error())
	}
	if emails := loadEmails(t, env.db); len(emails) != 0 {
		t.Errorf("适配器创建失败却落库了 %d 封邮件", len(emails))
	}
}

// TestSyncExecutorWebAPIEmailFieldMapping 覆盖上游字段到落库模型的映射：
// 收件人/抄送/标签必须序列化成 JSON 数��，而不是丢字段。
func TestSyncExecutorWebAPIEmailFieldMapping(t *testing.T) {
	account := testWebAPIAccount(t, `{"base_url":"http://127.0.0.1:1","jwt_token":"t"}`)
	email := webAPITestEmail("p-map", "映射检查", time.Now())
	email.CcAddresses = []string{"cc@example.com"}
	email.BccAddresses = []string{"bcc@example.com"}
	email.SourceLabels = []string{"inbox", "important"}
	email.HasAttachments = true
	email.AttachmentsCount = 2

	provider := &fakeWebAPIProvider{emails: []*adapter.Email{email}}
	env := newSyncExecutorTestEnv(t, account, provider)

	if err := env.executor.SyncAccount(context.Background(), account.UID); err != nil {
		t.Fatalf("同步失败: %v", err)
	}

	rows := loadEmails(t, env.db)
	if len(rows) != 1 {
		t.Fatalf("emails 行数 = %d，期望 1", len(rows))
	}
	row := rows[0]
	if row.Subject != "映射检查" || row.MessageID != "p-map@example.com" {
		t.Errorf("主题/MessageID 映射错误: %q/%q", row.Subject, row.MessageID)
	}
	if !row.HasAttachments || row.AttachmentsCount != 2 {
		t.Errorf("附件字段映射错误: %v/%d", row.HasAttachments, row.AttachmentsCount)
	}

	var to, cc, bcc, labels []string
	for _, c := range []struct {
		raw  string
		dest *[]string
		name string
	}{
		{row.ToAddresses, &to, "to_addresses"},
		{row.CcAddresses, &cc, "cc_addresses"},
		{row.BccAddresses, &bcc, "bcc_addresses"},
		{row.Labels, &labels, "labels"},
	} {
		if c.raw == "" {
			t.Errorf("%s 为空，字段被丢弃", c.name)
			continue
		}
		if err := json.Unmarshal([]byte(c.raw), c.dest); err != nil {
			t.Errorf("%s = %q 不是合法 JSON 数组: %v", c.name, c.raw, err)
		}
	}
	if len(to) != 1 || to[0] != "webapi@example.com" {
		t.Errorf("to_addresses = %v", to)
	}
	if len(cc) != 1 || cc[0] != "cc@example.com" {
		t.Errorf("cc_addresses = %v", cc)
	}
	if len(bcc) != 1 || bcc[0] != "bcc@example.com" {
		t.Errorf("bcc_addresses = %v", bcc)
	}
	if len(labels) != 2 {
		t.Errorf("labels = %v，期望 2 个标签", labels)
	}
}
