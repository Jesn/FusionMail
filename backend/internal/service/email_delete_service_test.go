package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"
	"fusionmail/pkg/crypto"

	"gorm.io/gorm"
)

// countingDeleteRepo 记录仓库调用次数，验证批量永久删除/清空回收站
// 的查询次数与 N 无关（N+1 回归测试）。
type countingDeleteRepo struct {
	repository.EmailRepository // 未实现的方法 panic，见 panic 兜底

	findByIDsCalls        int
	permDeleteByIDsCalls  int
	permDeleteAllCalls    int
	permDeleteSingleCalls int // 旧实现的逐封删除调用，新实现应为 0
}

func (r *countingDeleteRepo) FindByIDs(_ context.Context, ids []int64) ([]*model.Email, error) {
	r.findByIDsCalls++
	out := make([]*model.Email, 0, len(ids))
	for _, id := range ids {
		// 一半在回收站，一半不在 —— 验证服务层过滤
		out = append(out, &model.Email{ID: id, IsDeleted: id%2 == 0})
	}
	return out, nil
}

func (r *countingDeleteRepo) PermanentDeleteByIDs(_ context.Context, ids []int64) (int64, error) {
	r.permDeleteByIDsCalls++
	return int64(len(ids)), nil
}

func (r *countingDeleteRepo) PermanentDeleteAllTrash(context.Context) (int64, error) {
	r.permDeleteAllCalls++
	return 7, nil
}

func (r *countingDeleteRepo) PermanentDelete(context.Context, int64) error {
	r.permDeleteSingleCalls++
	return nil
}

// TestBatchPermanentDeleteEmailsConstantQueries 验证 1000 个 ID 的批量永久删除
// 仓库调用次数为常数 2（FindByIDs + PermanentDeleteByIDs），且过滤掉非回收站邮件。
func TestBatchPermanentDeleteEmailsConstantQueries(t *testing.T) {
	repo := &countingDeleteRepo{}
	svc := NewEmailDeleteService(repo, nil, nil, nil)

	ids := make([]int64, 1000)
	for i := range ids {
		ids[i] = int64(i + 1)
	}

	n, err := svc.BatchPermanentDeleteEmails(context.Background(), ids)
	if err != nil {
		t.Fatalf("批量永久删除失败: %v", err)
	}
	// 1000 个 ID 中偶数 ID 在回收站（id%2==0）→ 500
	if n != 500 {
		t.Fatalf("应删 500 封，got %d", n)
	}
	if repo.findByIDsCalls != 1 || repo.permDeleteByIDsCalls != 1 {
		t.Fatalf("仓库调用应恒为 2 次（1 FindByIDs + 1 批量删），got find=%d delete=%d",
			repo.findByIDsCalls, repo.permDeleteByIDsCalls)
	}
	if repo.permDeleteSingleCalls != 0 {
		t.Fatalf("不应再逐封删除（旧 N+1 路径），got %d 次", repo.permDeleteSingleCalls)
	}
}

// TestEmptyTrashSingleQuery 验证清空回收站只调 1 次仓库方法（原实现分页 + 逐封删）。
func TestEmptyTrashSingleQuery(t *testing.T) {
	repo := &countingDeleteRepo{}
	svc := NewEmailDeleteService(repo, nil, nil, nil)

	n, err := svc.EmptyTrash(context.Background())
	if err != nil {
		t.Fatalf("清空回收站失败: %v", err)
	}
	if n != 7 {
		t.Fatalf("应返回仓库删除行数 7，got %d", n)
	}
	if repo.permDeleteAllCalls != 1 || repo.permDeleteSingleCalls != 0 || repo.findByIDsCalls != 0 {
		t.Fatalf("应只调 1 次 PermanentDeleteAllTrash，got all=%d single=%d find=%d",
			repo.permDeleteAllCalls, repo.permDeleteSingleCalls, repo.findByIDsCalls)
	}
}

// stubDeleteAccountRepo 只实现删除链路会走到的账户方法（内嵌 nil 接口 = 作用域断言）。
type stubDeleteAccountRepo struct {
	repository.AccountRepository
	account *model.EmailAccount
	err     error
}

func (r *stubDeleteAccountRepo) FindByUID(context.Context, string) (*model.EmailAccount, error) {
	return r.account, r.err
}

// fakeSoftDeleter 是服务器软删除替身：按脚本依次返回错误。
type fakeSoftDeleter struct {
	adapter.SoftDeleter
	errs  []error
	calls []string
}

func (f *fakeSoftDeleter) MoveToTrash(_ context.Context, providerID string) error {
	f.calls = append(f.calls, providerID)
	if len(f.errs) == 0 {
		return nil
	}
	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}

// deleteTestEnv 是删除链路用例的接线（真实 PostgreSQL 仓库 + 替身账户仓库）。
type deleteTestEnv struct {
	svc         *EmailDeleteService
	db          *gorm.DB
	emailRepo   repository.EmailRepository
	accountRepo *stubDeleteAccountRepo
	crypto      *crypto.Service
}

func newDeleteTestEnv(t *testing.T) *deleteTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	cryptoService, err := crypto.NewService(testEncryptionKey)
	if err != nil {
		t.Fatalf("构造 crypto service 失败: %v", err)
	}

	emailRepo := repository.NewEmailRepository(db)
	accountRepo := &stubDeleteAccountRepo{}

	svc := NewEmailDeleteService(
		emailRepo,
		accountRepo,
		adapter.NewFactory(),
		NewCredentialResolver(cryptoService, nil),
	)

	return &deleteTestEnv{svc: svc, db: db, emailRepo: emailRepo, accountRepo: accountRepo, crypto: cryptoService}
}

// seedDeleteEmail 写入一封邮件 fixture。
func seedDeleteEmail(t *testing.T, db *gorm.DB, providerID string, isDeleted bool) *model.Email {
	t.Helper()

	email := &model.Email{
		ProviderID:  providerID,
		AccountUID:  "acct-delete",
		MessageID:   providerID + "@example.com",
		Subject:     "删除用例 " + providerID,
		FromAddress: "sender@example.com",
		ToAddress:   "me@example.com",
		TextBody:    "正文",
		IsDeleted:   isDeleted,
		SentAt:      time.Now(),
		ReceivedAt:  time.Now(),
	}
	if err := db.Create(email).Error; err != nil {
		t.Fatalf("写入邮件 fixture 失败: %v", err)
	}
	return email
}

// TestDeleteEmailMarksTrashAndRejectsMissing 覆盖单封删除：
// 标记 is_deleted 而不是物理删除（要能从回收站��复），不存在的邮件报错。
func TestDeleteEmailMarksTrashAndRejectsMissing(t *testing.T) {
	env := newDeleteTestEnv(t)
	ctx := context.Background()

	email := seedDeleteEmail(t, env.db, "p-del", false)
	if err := env.svc.DeleteEmail(ctx, email.ID); err != nil {
		t.Fatalf("删除邮件失败: %v", err)
	}

	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("删除后读不到邮件: %v", err)
	}
	if !stored.IsDeleted {
		t.Error("邮件未被标记为已删除")
	}
	if stored.DeletedAt.Valid {
		t.Error("软删除写入了 deleted_at，回收站列��（按 is_deleted 过滤）将看不到它")
	}

	if err := env.svc.DeleteEmail(ctx, 999999); err == nil {
		t.Error("删除不存在的邮件应报错")
	}
}

// TestBatchDeleteEmailsValidatesAndDedupes 覆盖批量删除的入参校验与去重：
// 空列表/非法 ID 拒绝；重复 ID 只删一次；已在回收站的邮件不重复计数。
func TestBatchDeleteEmailsValidatesAndDedupes(t *testing.T) {
	env := newDeleteTestEnv(t)
	ctx := context.Background()

	if _, err := env.svc.BatchDeleteEmails(ctx, nil); err == nil {
		t.Error("空 ID 列表应报错")
	}
	if _, err := env.svc.BatchDeleteEmails(ctx, []int64{1, 0}); err == nil {
		t.Error("非法 ID 应报错")
	}

	first := seedDeleteEmail(t, env.db, "p-batch-1", false)
	second := seedDeleteEmail(t, env.db, "p-batch-2", false)

	deleted, err := env.svc.BatchDeleteEmails(ctx, []int64{first.ID, first.ID, second.ID})
	if err != nil {
		t.Fatalf("批量删除失败: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("删除计数 = %d，期望 2（重复 ID 只算一次）", deleted)
	}

	// 已在回收站的邮件再次提交不应重复计数
	deleted, err = env.svc.BatchDeleteEmails(ctx, []int64{first.ID, second.ID})
	if err != nil {
		t.Fatalf("重复批量删除失败: %v", err)
	}
	if deleted != 0 {
		t.Errorf("重复删除计数 = %d，期望 0", deleted)
	}

	// 全部 ID 都不存在时返回 0 而不是报错
	deleted, err = env.svc.BatchDeleteEmails(ctx, []int64{888888, 999999})
	if err != nil {
		t.Fatalf("不存在的 ID 不应报错: %v", err)
	}
	if deleted != 0 {
		t.Errorf("不存在的 ID 删除计数 = %d，期望 0", deleted)
	}
}

// TestRestoreEmailReturnsToInbox 覆盖恢复：
// 恢复时必须同时清掉 is_deleted 与 is_archived，否则邮件会回到归档而不是收件箱。
func TestRestoreEmailReturnsToInbox(t *testing.T) {
	env := newDeleteTestEnv(t)
	ctx := context.Background()

	email := seedDeleteEmail(t, env.db, "p-restore", true)
	if err := env.db.Model(&model.Email{}).Where("id = ?", email.ID).
		UpdateColumns(map[string]interface{}{"is_archived": true}).Error; err != nil {
		t.Fatalf("设置归档状态失败: %v", err)
	}

	if err := env.svc.RestoreEmail(ctx, email.ID); err != nil {
		t.Fatalf("恢复邮件失败: %v", err)
	}

	stored, err := env.emailRepo.FindByID(ctx, email.ID)
	if err != nil || stored == nil {
		t.Fatalf("恢复后读不到邮件: %v", err)
	}
	if stored.IsDeleted {
		t.Error("恢复后 IsDeleted 仍为 true")
	}
	if stored.IsArchived {
		t.Error("恢复后 IsArchived 仍为 true（邮件不会回到收件箱）")
	}

	if err := env.svc.RestoreEmail(ctx, 999999); err == nil {
		t.Error("��复不存在的邮件应报错")
	}
}

// TestPermanentDeleteEmailOnlyFromTrash 覆盖永久删除：
// 只允许删除回收站里的邮件，正常邮件必须拒绝且不能丢数据。
func TestPermanentDeleteEmailOnlyFromTrash(t *testing.T) {
	env := newDeleteTestEnv(t)
	ctx := context.Background()

	normal := seedDeleteEmail(t, env.db, "p-normal", false)
	if err := env.svc.PermanentDeleteEmail(ctx, normal.ID); err == nil {
		t.Fatal("永久删除正常邮件应被拒绝")
	}
	if !emailExists(t, env.db, normal.ID) {
		t.Error("被拒绝的永久删除却删掉了数据")
	}

	trash := seedDeleteEmail(t, env.db, "p-trash", true)
	if err := env.svc.PermanentDeleteEmail(ctx, trash.ID); err != nil {
		t.Fatalf("永久删除回收站邮件失败: %v", err)
	}
	if emailExists(t, env.db, trash.ID) {
		t.Error("永久删除后邮件仍在库中（未真正物理删除）")
	}

	if err := env.svc.PermanentDeleteEmail(ctx, 999999); err == nil {
		t.Error("永久删除不存在的邮件应报错")
	}
}

// TestBatchPermanentDeleteEmailsSkipsNonTrash 覆盖批量永久删除的真实落库语义：
// 只删回收站里的，正常邮件被跳过；返回行数���成功数。
func TestBatchPermanentDeleteEmailsSkipsNonTrash(t *testing.T) {
	env := newDeleteTestEnv(t)
	ctx := context.Background()

	trash := seedDeleteEmail(t, env.db, "p-batch-trash", true)
	normal := seedDeleteEmail(t, env.db, "p-batch-normal", false)

	deleted, err := env.svc.BatchPermanentDeleteEmails(ctx, []int64{trash.ID, normal.ID})
	if err != nil {
		t.Fatalf("批量永久删除失败: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("删除计数 = %d，期望 1", deleted)
	}
	if emailExists(t, env.db, trash.ID) {
		t.Error("回收站邮件未被物理删除")
	}
	if !emailExists(t, env.db, normal.ID) {
		t.Error("正常邮件被误删")
	}

	if deleted, err := env.svc.BatchPermanentDeleteEmails(ctx, nil); err != nil || deleted != 0 {
		t.Errorf("空列表应返回 0,nil，实际 %d,%v", deleted, err)
	}
	if deleted, err := env.svc.BatchPermanentDeleteEmails(ctx, []int64{normal.ID}); err != nil || deleted != 0 {
		t.Errorf("全是正常邮件时应返回 0,nil，实际 %d,%v", deleted, err)
	}
}

// TestEmptyTrashRemovesOnlyTrash 覆盖清空回收站：一��物理删除全部回收站邮件。
func TestEmptyTrashRemovesOnlyTrash(t *testing.T) {
	env := newDeleteTestEnv(t)
	ctx := context.Background()

	first := seedDeleteEmail(t, env.db, "p-empty-1", true)
	second := seedDeleteEmail(t, env.db, "p-empty-2", true)
	normal := seedDeleteEmail(t, env.db, "p-empty-normal", false)

	deleted, err := env.svc.EmptyTrash(ctx)
	if err != nil {
		t.Fatalf("清空回收站失败: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("清空计数 = %d，期望 2", deleted)
	}
	if emailExists(t, env.db, first.ID) || emailExists(t, env.db, second.ID) {
		t.Error("回收站邮件未被清空")
	}
	if !emailExists(t, env.db, normal.ID) {
		t.Error("正常邮件被清空回收站误删")
	}

	if deleted, err := env.svc.EmptyTrash(ctx); err != nil || deleted != 0 {
		t.Errorf("重复清空应返回 0,nil，实际 %d,%v", deleted, err)
	}
}

// TestTryServerSoftDeleteEarlyReturns 覆盖服务器软删除的前置分支：
// 账户查不到 / 未开启 soft 策�� / 凭证解析失败 / 适配器创建失败，都必须安静返回。
func TestTryServerSoftDeleteEarlyReturns(t *testing.T) {
	email := &model.Email{ID: 1, AccountUID: "acct-delete", ProviderID: "p-1"}

	cases := []struct {
		name    string
		account *model.EmailAccount
		err     error
	}{
		{name: "账户查��失败", err: errors.New("db down")},
		{name: "账户不存在", account: nil},
		{name: "未开启服务器软删除", account: &model.EmailAccount{UID: "acct-delete", ServerDeletePolicy: "off"}},
		{
			name: "凭证解析失败",
			account: &model.EmailAccount{
				UID:                  "acct-delete",
				ServerDeletePolicy:   "soft",
				EncryptedCredentials: "broken",
			},
		},
		{
			name: "适配器创建失败",
			account: &model.EmailAccount{
				UID:                "acct-delete",
				ServerDeletePolicy: "soft",
				AdapterRef:         &model.Adapter{Name: "gmail", AuthType: "oauth2"},
				ProviderRef:        &model.Provider{Name: "unknown-provider"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newDeleteTestEnv(t)
			env.accountRepo.account = c.account
			env.accountRepo.err = c.err

			if c.account != nil && c.account.EncryptedCredentials == "" && c.account.AdapterRef != nil {
				encrypted, err := env.crypto.Encrypt([]byte(`{"access_token":"at","client_id":"cid","client_secret":"cs"}`))
				if err != nil {
					t.Fatalf("加密凭据失败: %v", err)
				}
				c.account.EncryptedCredentials = encrypted
			}

			// 不应 panic，也不应留下任何副作用
			env.svc.tryServerSoftDelete(context.Background(), email)
		})
	}
}

// TestSoftDeleteWithRetrySucceedsFirstAttempt 覆盖首次即成功。
func TestSoftDeleteWithRetrySucceedsFirstAttempt(t *testing.T) {
	svc := NewEmailDeleteService(nil, nil, nil, nil)
	deleter := &fakeSoftDeleter{}

	if err := svc.softDeleteWithRetry(context.Background(), deleter, "p-1"); err != nil {
		t.Fatalf("软删除失败: %v", err)
	}
	if len(deleter.calls) != 1 {
		t.Errorf("调用次数 = %d，期望 1", len(deleter.calls))
	}
}

// TestSoftDeleteWithRetryTreats404AsSuccess 覆盖幂等语义：
// 服务器返回 404（邮件已不存在）视为成功，不应重试。
func TestSoftDeleteWithRetryTreats404AsSuccess(t *testing.T) {
	svc := NewEmailDeleteService(nil, nil, nil, nil)
	deleter := &fakeSoftDeleter{errs: []error{errors.New("404 not found")}}

	if err := svc.softDeleteWithRetry(context.Background(), deleter, "p-1"); err != nil {
		t.Fatalf("404 应视为成功，实际报错: %v", err)
	}
	if len(deleter.calls) != 1 {
		t.Errorf("404 后重试了 %d 次，期望不重试", len(deleter.calls)-1)
	}
}

// TestSoftDeleteWithRetryRetriesThenSucceeds 覆盖重试后成功。
func TestSoftDeleteWithRetryRetriesThenSucceeds(t *testing.T) {
	svc := NewEmailDeleteService(nil, nil, nil, nil)
	deleter := &fakeSoftDeleter{errs: []error{errors.New("临时故障")}}

	if err := svc.softDeleteWithRetry(context.Background(), deleter, "p-1"); err != nil {
		t.Fatalf("重试后应成功，实际报错: %v", err)
	}
	if len(deleter.calls) != 2 {
		t.Errorf("调用次数 = %d，期望 2", len(deleter.calls))
	}
}

// TestSoftDeleteWithRetryGivesUpAfterMaxAttempts 覆盖重试耗尽：
// 必须返回错误（调用方据此记录日志），且尝试次数受上限约束。
func TestSoftDeleteWithRetryGivesUpAfterMaxAttempts(t *testing.T) {
	svc := NewEmailDeleteService(nil, nil, nil, nil)
	deleter := &fakeSoftDeleter{errs: []error{
		errors.New("失败1"), errors.New("失败2"), errors.New("失败3"),
	}}

	// 退避是 1s + 2s，这里真的等满 3 次尝试：只有这样才能验证「耗尽后返回错误」，
	// 而不是被 context 取消提前中止（那个语义由下一条用例覆盖）。
	err := svc.softDeleteWithRetry(context.Background(), deleter, "p-1")
	if err == nil {
		t.Fatal("重试耗尽应返回错误")
	}
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("error = %q，期望提示重试耗尽", err.Error())
	}
	if len(deleter.calls) != 3 {
		t.Errorf("调用次数 = %d，期望 3（maxRetries）", len(deleter.calls))
	}
}

// TestSoftDeleteWithRetryStopsWhenContextExpires 覆盖退避中 context 到期：
// 必须立刻返回而不是等满重试次数。
func TestSoftDeleteWithRetryStopsWhenContextExpires(t *testing.T) {
	svc := NewEmailDeleteService(nil, nil, nil, nil)
	deleter := &fakeSoftDeleter{errs: []error{errors.New("失败1"), errors.New("失败2")}}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := svc.softDeleteWithRetry(ctx, deleter, "p-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v，期望 context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("context 到期后仍等待了 %v", elapsed)
	}
	if len(deleter.calls) != 1 {
		t.Errorf("调用次数 = %d，期望 1", len(deleter.calls))
	}
}

// TestSoftDeleteWithRetryHonorsContextCancel 覆盖取消：
// context 取消后必须立刻返回，不能继续退避等待。
func TestSoftDeleteWithRetryHonorsContextCancel(t *testing.T) {
	svc := NewEmailDeleteService(nil, nil, nil, nil)
	deleter := &fakeSoftDeleter{errs: []error{errors.New("失败1"), errors.New("失败2")}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := svc.softDeleteWithRetry(ctx, deleter, "p-1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v，期望 context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("取消后仍等待了 %v", elapsed)
	}
	if len(deleter.calls) != 1 {
		t.Errorf("取消后仍重试: 调用次数 = %d", len(deleter.calls))
	}
}
