package service

import (
	"context"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"fusionmail/pkg/logger"
)

// 模块日志记录器
var spamServiceLog = logger.NewWithModule("SpamService")

// SpamService 垃圾邮件服务接口
type SpamService interface {
	// 垃圾邮件标记
	MarkAsSpam(ctx context.Context, emailIDs []int64) error
	UnmarkAsSpam(ctx context.Context, emailIDs []int64) error

	// 垃圾邮件管理
	BatchDeleteSpam(ctx context.Context, emailIDs []int64) (int64, error)
	EmptySpamFolder(ctx context.Context, accountUID string) (int64, error)

	// 垃圾邮件查询
	GetSpamEmails(ctx context.Context, accountUID string, page, pageSize int) ([]*model.Email, int64, error)
	GetSpamStats(ctx context.Context, accountUID string) (*SpamStats, error)

	// 贝叶斯分类器
	GetBayesianStatus(ctx context.Context, userUID string) (*spam.ModelStatus, error)
	TrainBayesianModel(ctx context.Context, userUID string) error
	ResetBayesianModel(ctx context.Context, userUID string) error
	GetBayesianTrainingStats(ctx context.Context, userUID string) (*spam.TrainingStats, error)

	// 规则管理
	GetRules(ctx context.Context, category string, page, pageSize int) ([]*model.SpamRule, int64, error)
	GetRuleByID(ctx context.Context, id int64) (*model.SpamRule, error)
	CreateRule(ctx context.Context, rule *model.SpamRule) error
	UpdateRule(ctx context.Context, rule *model.SpamRule) error
	DeleteRule(ctx context.Context, id int64) error
	ToggleRule(ctx context.Context, id int64) error
	TestRule(ctx context.Context, pattern, category, content string) (bool, []string, error)
	GetRuleStats(ctx context.Context) (*RuleStats, error)
}

// SpamStats 垃圾邮件统计
type SpamStats struct {
	TotalCount   int64 `json:"total_count"`   // 总垃圾邮件数
	UnreadCount  int64 `json:"unread_count"`  // 未读垃圾邮件数
	TodayCount   int64 `json:"today_count"`   // 今日垃圾邮件数
	WeekCount    int64 `json:"week_count"`    // 本周垃圾邮件数
	MonthCount   int64 `json:"month_count"`   // 本月垃圾邮件数
	BlockedCount int64 `json:"blocked_count"` // 拦截的垃圾邮件数
}

// RuleStats 规则统计
type RuleStats struct {
	TotalCount    int64 `json:"total_count"`    // 总规则数
	EnabledCount  int64 `json:"enabled_count"`  // 启用规则数
	DisabledCount int64 `json:"disabled_count"` // 禁用规则数
	BuiltinCount  int64 `json:"builtin_count"`  // 内置规则数
	CustomCount   int64 `json:"custom_count"`   // 自定义规则数
	TotalHits     int64 `json:"total_hits"`     // 总命中次数
}

// spamService 垃圾邮件服务实现
type spamService struct {
	emailRepo          repository.EmailRepository
	ruleRepo           repository.SpamRuleRepository
	reputationManager  *spam.ReputationManager
	bayesianClassifier *spam.BayesianClassifier
}

// NewSpamService 创建垃圾邮件服务
func NewSpamService(
	emailRepo repository.EmailRepository,
	ruleRepo repository.SpamRuleRepository,
	reputationManager *spam.ReputationManager,
	bayesianClassifier *spam.BayesianClassifier,
) SpamService {
	return &spamService{
		emailRepo:          emailRepo,
		ruleRepo:           ruleRepo,
		reputationManager:  reputationManager,
		bayesianClassifier: bayesianClassifier,
	}
}
