package service

import (
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"fusionmail/internal/testutil/pgtest"

	"gorm.io/gorm"
)

// spamTestEnv 是垃圾判定用例的接线：真实 PostgreSQL 仓库；
// 信誉/贝叶斯组件由 newSpamTestEnvWithManagers 决定是否挂载。
type spamTestEnv struct {
	svc            *spamService
	db             *gorm.DB
	emailRepo      repository.EmailRepository
	ruleRepo       repository.SpamRuleRepository
	reputationRepo repository.SenderReputationRepository
	trainingRepo   repository.BayesianTrainingRepository
}

// newSpamTestEnv 构造「只走同步路径」的接线：信誉/贝叶斯组件留空。
// MarkAsSpam/UnmarkAsSpam 会把信誉与训练副作用丢进 goroutine，
// 用例里若挂上真实组件，异步写入会跨测试污染下一个用例（Truncate 之后才落库）。
// 副作用本身由 TestUserFeedbackUpdatesReputationAndTraining 同步验证。
func newSpamTestEnv(t *testing.T) *spamTestEnv {
	t.Helper()
	return newSpamTestEnvWithManagers(t, false)
}

// newSpamSideEffectEnv 构造带真实信誉/贝叶斯组件的接线（仅副作用用例使用）。
func newSpamSideEffectEnv(t *testing.T) *spamTestEnv {
	t.Helper()
	return newSpamTestEnvWithManagers(t, true)
}

func newSpamTestEnvWithManagers(t *testing.T, withManagers bool) *spamTestEnv {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)

	emailRepo := repository.NewEmailRepository(db)
	ruleRepo := repository.NewSpamRuleRepository(db)
	reputationRepo := repository.NewSenderReputationRepository(db)
	trainingRepo := repository.NewBayesianTrainingRepository(db)

	var svc *spamService
	if withManagers {
		svc = NewSpamService(
			emailRepo,
			ruleRepo,
			spam.NewReputationManager(reputationRepo, nil), // redisClient 为 nil：缓存失效分支自带 nil 保护
			spam.NewBayesianClassifier(trainingRepo),
		).(*spamService)
	} else {
		svc = NewSpamService(emailRepo, ruleRepo, nil, nil).(*spamService)
	}

	return &spamTestEnv{
		svc:            svc,
		db:             db,
		emailRepo:      emailRepo,
		ruleRepo:       ruleRepo,
		reputationRepo: reputationRepo,
		trainingRepo:   trainingRepo,
	}
}

// seedSpamTestEmail 写入一封邮件 fixture。
func seedSpamTestEmail(t *testing.T, db *gorm.DB, accountUID, providerID string, isSpam bool) *model.Email {
	t.Helper()

	email := &model.Email{
		ProviderID:  providerID,
		AccountUID:  accountUID,
		MessageID:   providerID + "@example.com",
		Subject:     "测试邮件 " + providerID,
		FromAddress: "sender@example.com",
		ToAddress:   "me@example.com",
		TextBody:    "正文",
		IsSpam:      isSpam,
		SentAt:      time.Now(),
		ReceivedAt:  time.Now(),
	}
	if err := db.Create(email).Error; err != nil {
		t.Fatalf("写入邮件 fixture 失败: %v", err)
	}
	return email
}

// reloadEmail 从数据库重新读取邮件（避免断言到内存里被就地修改的对象）。
func reloadEmail(t *testing.T, env *spamTestEnv, id int64) *model.Email {
	t.Helper()

	email, err := env.emailRepo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("读取邮件失败: %v", err)
	}
	if email == nil {
		t.Fatalf("邮件 %d 不存在", id)
	}
	return email
}

// seedSpamRule 直接经仓库写入规则（绕过 CreateRule 的「强制非内置」逻辑）。
func seedSpamRule(t *testing.T, repo repository.SpamRuleRepository, rule *model.SpamRule) *model.SpamRule {
	t.Helper()

	if err := repo.Create(t.Context(), rule); err != nil {
		t.Fatalf("写入规则 fixture 失败: %v", err)
	}
	return rule
}
