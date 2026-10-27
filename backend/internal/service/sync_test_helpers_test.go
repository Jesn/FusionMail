package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"
	"fusionmail/pkg/crypto"
	"gorm.io/gorm"
)

// fakeAccountSyncRunner 记录调度器发起的同步请求。
// SyncScheduler 只依赖这个窄接口（P9 拆分产物），所以调度用例不需要真实适配器。
type fakeAccountSyncRunner struct {
	mu      sync.Mutex
	calls   []string
	errs    map[string]error
	started chan string
}

func newFakeAccountSyncRunner() *fakeAccountSyncRunner {
	return &fakeAccountSyncRunner{
		errs:    make(map[string]error),
		started: make(chan string, 64),
	}
}

func (f *fakeAccountSyncRunner) SyncAccount(_ context.Context, uid string) error {
	f.mu.Lock()
	f.calls = append(f.calls, uid)
	err := f.errs[uid]
	f.mu.Unlock()

	select {
	case f.started <- uid:
	default:
	}
	return err
}

func (f *fakeAccountSyncRunner) calledUIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// waitForCalls 等待调度器把同步请求发出来。
// SyncAllAccounts / checkAndSyncAccounts 都是「起 goroutine 后立刻返回」，
// 断言必须等信号，不能 sleep 猜时间。
func (f *fakeAccountSyncRunner) waitForCalls(t *testing.T, n int) {
	t.Helper()

	deadline := time.After(3 * time.Second)
	for len(f.calledUIDs()) < n {
		select {
		case <-f.started:
		case <-deadline:
			t.Fatalf("等待 %d 次同步调用超时，实际 %d 次: %v", n, len(f.calledUIDs()), f.calledUIDs())
		}
	}
}

type syncStatusUpdate struct {
	UID     string
	Status  string
	Message string
}

// stubSyncAccountRepo 只实现同步链路会走到的账户方法（内嵌 nil 接口 = 作用域断言）。
type stubSyncAccountRepo struct {
	repository.AccountRepository
	accounts        []*model.EmailAccount
	listErr         error
	byUID           map[string]*model.EmailAccount
	statusUpdates   []syncStatusUpdate
	updatedAccounts []*model.EmailAccount
	resetFailures   []string
}

func newStubSyncAccountRepo(accounts ...*model.EmailAccount) *stubSyncAccountRepo {
	repo := &stubSyncAccountRepo{
		accounts: accounts,
		byUID:    make(map[string]*model.EmailAccount, len(accounts)),
	}
	for _, account := range accounts {
		repo.byUID[account.UID] = account
	}
	return repo
}

func (r *stubSyncAccountRepo) ListSyncEnabled(context.Context) ([]*model.EmailAccount, error) {
	return r.accounts, r.listErr
}

func (r *stubSyncAccountRepo) FindByUIDWithRelations(_ context.Context, uid string) (*model.EmailAccount, error) {
	return r.byUID[uid], nil
}

func (r *stubSyncAccountRepo) UpdateSyncStatus(_ context.Context, uid, status, message string) error {
	r.statusUpdates = append(r.statusUpdates, syncStatusUpdate{UID: uid, Status: status, Message: message})
	return nil
}

func (r *stubSyncAccountRepo) Update(_ context.Context, account *model.EmailAccount) error {
	r.updatedAccounts = append(r.updatedAccounts, account)
	return nil
}

func (r *stubSyncAccountRepo) ResetConsecutiveFailures(_ context.Context, uid string) error {
	r.resetFailures = append(r.resetFailures, uid)
	return nil
}

// IncrementConsecutiveFailures 只在认证类错误路径被调用，这里给个最小实现，
// 避免用例因为「替身少实现一个方法」而 panic。
func (r *stubSyncAccountRepo) IncrementConsecutiveFailures(context.Context, string) (int, error) {
	return 1, nil
}

// HealWebhookChildPollingFlags 由 StartScheduler 的启动自愈调用。
func (r *stubSyncAccountRepo) HealWebhookChildPollingFlags(context.Context) (int64, error) {
	return 0, nil
}

func (r *stubSyncAccountRepo) lastStatusFor(uid string) (syncStatusUpdate, bool) {
	for i := len(r.statusUpdates) - 1; i >= 0; i-- {
		if r.statusUpdates[i].UID == uid {
			return r.statusUpdates[i], true
		}
	}
	return syncStatusUpdate{}, false
}

// testWebAPIServiceType 是只存在于测试里的 WebAPI 服务类型，
// 避免和真实注册的 cloud_mail / cloudflare_temp_email 抢注册表。
const testWebAPIServiceType = "test_webapi_p19"

// testWebAPIProviderHolder 让每个用例决定注册表返回哪个替身适配器。
var testWebAPIProviderHolder struct {
	mu       sync.Mutex
	provider webapi.WebAPIProvider
	err      error
}

func init() {
	webapi.RegisterAdapter(testWebAPIServiceType, func(string) (webapi.WebAPIProvider, error) {
		testWebAPIProviderHolder.mu.Lock()
		defer testWebAPIProviderHolder.mu.Unlock()
		if testWebAPIProviderHolder.err != nil {
			return nil, testWebAPIProviderHolder.err
		}
		return testWebAPIProviderHolder.provider, nil
	})
}

func setTestWebAPIProvider(provider webapi.WebAPIProvider, err error) {
	testWebAPIProviderHolder.mu.Lock()
	defer testWebAPIProviderHolder.mu.Unlock()
	testWebAPIProviderHolder.provider = provider
	testWebAPIProviderHolder.err = err
}

// fakeWebAPIProvider 是 WebAPI 适配器替身：只实现执行器真正调用的三个方法，
// 其余（GetConfig/TestConnection/检查点等）保持 nil 接口，误用即 panic。
type fakeWebAPIProvider struct {
	webapi.WebAPIProvider
	emails       []*adapter.Email
	fetchErr     error
	connectErr   error
	connected    bool
	fetchedSince []time.Time
	fetchLimits  []int
}

func (f *fakeWebAPIProvider) Connect(context.Context) error {
	if f.connectErr != nil {
		return f.connectErr
	}
	f.connected = true
	return nil
}

func (f *fakeWebAPIProvider) Disconnect() error {
	f.connected = false
	return nil
}

func (f *fakeWebAPIProvider) FetchEmails(_ context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	f.fetchedSince = append(f.fetchedSince, since)
	f.fetchLimits = append(f.fetchLimits, limit)
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.emails, nil
}

func (f *fakeWebAPIProvider) GetServiceName() string { return testWebAPIServiceType }

// syncExecutorTestEnv 是同步执行器用例的接线。
type syncExecutorTestEnv struct {
	executor    *SyncExecutor
	db          *gorm.DB
	crypto      *crypto.Service
	accountRepo *stubSyncAccountRepo
	emailRepo   repository.EmailRepository
	syncLogRepo repository.SyncLogRepository
	provider    *fakeWebAPIProvider
	account     *model.EmailAccount
}

// newSyncExecutorTestEnv 用真实 PostgreSQL 上的 email / sync_log 仓库 +
// 替身 WebAPI 适配器构造执行器。syncLock 传 nil：走内存锁路径，
// 让同步用例不依赖 Redis（Redis 锁路径由 sync_lock_test.go 单独覆盖）。
func newSyncExecutorTestEnv(t *testing.T, account *model.EmailAccount, provider *fakeWebAPIProvider) *syncExecutorTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	cryptoService, err := crypto.NewService(testEncryptionKey)
	if err != nil {
		t.Fatalf("构造 crypto service 失败: %v", err)
	}

	setTestWebAPIProvider(provider, nil)
	t.Cleanup(func() { setTestWebAPIProvider(nil, nil) })

	emailRepo := repository.NewEmailRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)
	accountRepo := newStubSyncAccountRepo(account)

	executor := NewSyncExecutor(
		accountRepo,
		emailRepo,
		syncLogRepo,
		adapter.NewFactory(),
		webapi.NewWebAPIAdapterFactory(),
		cryptoService,
		NewCredentialResolver(cryptoService, nil),
		NewNoopSyncNotifier(),
		NewDedupeKeyGenerator(),
		NewSyncProcessor(emailRepo, repository.NewDeletedEmailKeyRepository(db), NewDedupeKeyGenerator(), nil, nil),
		NewSyncErrorPolicy(accountRepo, syncLogRepo),
		NewSyncProgressRegistry(),
		nil,
	)

	return &syncExecutorTestEnv{
		executor:    executor,
		db:          db,
		crypto:      cryptoService,
		accountRepo: accountRepo,
		emailRepo:   emailRepo,
		syncLogRepo: syncLogRepo,
		provider:    provider,
		account:     account,
	}
}

// testWebAPIAccount 构造一个 WebAPI 协议账户（内存 fixture，不落库）。
// 加密只用密钥，不依赖 env，便于在构造 env 之前先备好账户。
func testWebAPIAccount(t *testing.T, authDataJSON string) *model.EmailAccount {
	t.Helper()

	cryptoService, err := crypto.NewService(testEncryptionKey)
	if err != nil {
		t.Fatalf("构造 crypto service 失败: %v", err)
	}
	encrypted, err := cryptoService.Encrypt([]byte(authDataJSON))
	if err != nil {
		t.Fatalf("加密 AuthData 失败: %v", err)
	}

	return &model.EmailAccount{
		UID:                  "acct-webapi-1",
		Email:                "webapi@example.com",
		Status:               "active",
		SyncEnabled:          true,
		SyncInterval:         2,
		EncryptedCredentials: encrypted,
		ProviderRef: &model.Provider{
			ID:       1,
			Name:     "测试 WebAPI 服务",
			Metadata: `{"service_type":"` + testWebAPIServiceType + `"}`,
		},
		AdapterRef: &model.Adapter{ID: 1, Name: "webapi"},
	}
}

// webAPITestEmail 构造一封上游返回的邮件。
func webAPITestEmail(providerID, subject string, sentAt time.Time) *adapter.Email {
	return &adapter.Email{
		ProviderID:  providerID,
		MessageID:   providerID + "@example.com",
		Subject:     subject,
		FromAddress: "sender@example.com",
		FromName:    "Sender",
		ToAddresses: []string{"webapi@example.com"},
		TextBody:    subject + " 正文",
		SentAt:      sentAt,
		ReceivedAt:  sentAt,
	}
}

// loadSyncLogs 读出全部同步日志，按 ID 升序。
func loadSyncLogs(t *testing.T, db *gorm.DB) []model.SyncLog {
	t.Helper()

	var logs []model.SyncLog
	if err := db.Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("读取 sync_logs 失败: %v", err)
	}
	return logs
}

// loadEmails 读出全部邮件，按 ID 升序。
func loadEmails(t *testing.T, db *gorm.DB) []model.Email {
	t.Helper()

	var emails []model.Email
	if err := db.Order("id asc").Find(&emails).Error; err != nil {
		t.Fatalf("读取 emails 失败: %v", err)
	}
	return emails
}
