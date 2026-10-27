package service

import (
	"context"
	"os"
	"testing"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"

	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	pgtest.Init("service")
	os.Exit(m.Run())
}

// testEncryptionKey 必须恰好 32 字节（见 .trellis/spec/backend/testing.md）。
const testEncryptionKey = "test-2fa-secret-key-32-bytes-ok!"

// fakeMailSender 记录真正交给发送器的邮件，并按用例返回预设结果。
// 用它替代真实 SMTP：发信链路的回归点是「构建了什么 + 落库了什么」，
// 不是「能不能连上 25 端口」。
type fakeMailSender struct {
	sent       []*adapter.OutgoingEmail
	result     *adapter.SendResult
	err        error
	senderType string
}

func (f *fakeMailSender) Send(_ context.Context, email *adapter.OutgoingEmail) (*adapter.SendResult, error) {
	f.sent = append(f.sent, email)
	return f.result, f.err
}

func (f *fakeMailSender) TestConnection(context.Context) error { return nil }

func (f *fakeMailSender) GetSenderType() string {
	if f.senderType != "" {
		return f.senderType
	}
	return adapter.SenderTypeSMTP
}

// stubSenderResolver 是 SendService 唯一的生产依赖替身（见 design.md D3），
// 顺带记录它拿到的凭证，用于断言凭据解密链路。
type stubSenderResolver struct {
	sender  adapter.MailSender
	err     error
	account *model.EmailAccount
	creds   *adapter.AccountCredentials
}

func (r *stubSenderResolver) GetSenderWithFallback(
	account *model.EmailAccount,
	creds *adapter.AccountCredentials,
) (adapter.MailSender, error) {
	r.account, r.creds = account, creds
	return r.sender, r.err
}

// stubSendAccountRepo 只实现发信链路真正会走到的账户查询。
// 内嵌 nil 接口：一旦测试路径碰到别的方法就会 panic —— 这是有意的
// 「作用域断言」，避免用例悄悄依赖无关行为。
type stubSendAccountRepo struct {
	repository.AccountRepository
	account *model.EmailAccount
	err     error
}

func (r *stubSendAccountRepo) FindByUIDWithRelations(context.Context, string) (*model.EmailAccount, error) {
	return r.account, r.err
}

// sendTestEnv 是一次发信用例的全部接线。
type sendTestEnv struct {
	svc      *SendService
	db       *gorm.DB
	crypto   *crypto.Service
	resolver *stubSenderResolver
	sender   *fakeMailSender
}

// newSendTestEnv 准备真实 PostgreSQL 上的发信环境：真实 sent_email / email
// 仓库（断言落库），账户查询与发送器用替身。
func newSendTestEnv(t *testing.T, account *model.EmailAccount) *sendTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	cryptoService, err := crypto.NewService(testEncryptionKey)
	if err != nil {
		t.Fatalf("构造 crypto service 失败: %v", err)
	}

	sender := &fakeMailSender{}
	resolver := &stubSenderResolver{sender: sender}

	svc := NewSendService(
		resolver,
		&stubSendAccountRepo{account: account},
		repository.NewSentEmailRepository(db),
		repository.NewEmailRepository(db),
		cryptoService,
		nil, // oauth2ConfigProvider：仅 OAuth2 缺 client_id 的兜底路径需要，本文件不覆盖
		logger.NewWithModule("SendTest"),
	)

	return &sendTestEnv{svc: svc, db: db, crypto: cryptoService, resolver: resolver, sender: sender}
}

// testSendAccount 返回一个 SMTP 账户 fixture（不落库，账户查询是替身）。
func testSendAccount() *model.EmailAccount {
	return &model.EmailAccount{
		UID:    "acct-send-1",
		Email:  "sender@example.com",
		Status: "active",
	}
}

// loadSentEmails 读出全部已发送记录，按 ID 升序。
func loadSentEmails(t *testing.T, db *gorm.DB) []model.SentEmail {
	t.Helper()

	var rows []model.SentEmail
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("读取 sent_emails 失败: %v", err)
	}
	return rows
}
