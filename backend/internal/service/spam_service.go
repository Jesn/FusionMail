package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"fusionmail/pkg/logger"
	"regexp"
	"strings"
	"time"
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

// MarkAsSpam 标记邮件为垃圾邮件
func (s *spamService) MarkAsSpam(ctx context.Context, emailIDs []int64) error {
	for _, emailID := range emailIDs {
		// 获取邮件信息
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil || email == nil {
			spamServiceLog.Warn("无法找到邮件: id=%d, err=%v", emailID, err)
			continue
		}

		// 更新邮件的垃圾邮件状态
		email.IsSpam = true
		email.UserMarkedSpam = true
		now := time.Now()
		email.UserMarkedAt = &now

		if err := s.emailRepo.Update(ctx, email); err != nil {
			spamServiceLog.Warn("标记邮件为垃圾邮件失败: id=%d, err=%v", emailID, err)
			continue
		}

		// 异步更新发件人信誉（降低）
		if s.reputationManager != nil {
			go s.updateReputationForSpam(context.Background(), email.FromAddress, true)
		}

		// 异步添加贝叶斯训练数据
		if s.bayesianClassifier != nil {
			go s.addBayesianTraining(context.Background(), email, true)
		}
	}

	return nil
}

// UnmarkAsSpam 取消垃圾邮件标记
func (s *spamService) UnmarkAsSpam(ctx context.Context, emailIDs []int64) error {
	for _, emailID := range emailIDs {
		// 获取邮件信息
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil || email == nil {
			spamServiceLog.Warn("无法找到邮件: id=%d, err=%v", emailID, err)
			continue
		}

		// 取消垃圾邮件标记
		email.IsSpam = false
		email.UserMarkedSpam = false
		email.UserMarkedAt = nil

		if err := s.emailRepo.Update(ctx, email); err != nil {
			spamServiceLog.Warn("取消邮件垃圾邮件标记失败: id=%d, err=%v", emailID, err)
			continue
		}

		// 异步更新发件人信誉（提高）
		if s.reputationManager != nil {
			go s.updateReputationForSpam(context.Background(), email.FromAddress, false)
		}

		// 异步添加贝叶斯训练数据
		if s.bayesianClassifier != nil {
			go s.addBayesianTraining(context.Background(), email, false)
		}
	}

	return nil
}

// BatchDeleteSpam 批量删除垃圾邮件
func (s *spamService) BatchDeleteSpam(ctx context.Context, emailIDs []int64) (int64, error) {
	deletedCount := int64(0)

	for _, emailID := range emailIDs {
		// 软删除邮件
		deleted := true
		if err := s.emailRepo.UpdateLocalStatus(ctx, emailID, nil, nil, nil, &deleted); err != nil {
			spamServiceLog.Warn("删除邮件失败: id=%d, err=%v", emailID, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// EmptySpamFolder 清空垃圾箱
func (s *spamService) EmptySpamFolder(ctx context.Context, accountUID string) (int64, error) {
	// 构建过滤条件：查询所有垃圾邮件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 查询所有垃圾邮件
	emails, _, err := s.emailRepo.List(ctx, filter, 0, 10000) // 限制最多 10000 封
	if err != nil {
		return 0, fmt.Errorf("failed to list spam emails: %w", err)
	}

	// 批量删除
	deletedCount := int64(0)
	for _, email := range emails {
		deleted := true
		if err := s.emailRepo.UpdateLocalStatus(ctx, email.ID, nil, nil, nil, &deleted); err != nil {
			spamServiceLog.Warn("删除邮件失败: id=%d, err=%v", email.ID, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// GetSpamEmails 获取垃圾邮件列表
func (s *spamService) GetSpamEmails(ctx context.Context, accountUID string, page, pageSize int) ([]*model.Email, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建过滤条件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 查询垃圾邮件列表
	emails, total, err := s.emailRepo.List(ctx, filter, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get spam emails: %w", err)
	}

	return emails, total, nil
}

// GetSpamStats 获取垃圾邮件统计
func (s *spamService) GetSpamStats(ctx context.Context, accountUID string) (*SpamStats, error) {
	stats := &SpamStats{}

	// 构建基础过滤条件
	isSpam := true
	isDeleted := false
	filter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
	}

	if accountUID != "" {
		filter.AccountUID = accountUID
	}

	// 统计总数
	total, err := s.emailRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count spam emails: %w", err)
	}
	stats.TotalCount = total

	// 统计未读数
	isRead := false
	unreadFilter := &repository.EmailFilter{
		IsSpam:    &isSpam,
		IsDeleted: &isDeleted,
		IsRead:    &isRead,
	}
	if accountUID != "" {
		unreadFilter.AccountUID = accountUID
	}
	unreadCount, err := s.emailRepo.Count(ctx, unreadFilter)
	if err == nil {
		stats.UnreadCount = unreadCount
	}

	// 其他统计暂时设为 0，后续实现
	stats.TodayCount = 0
	stats.WeekCount = 0
	stats.MonthCount = 0
	stats.BlockedCount = 0

	return stats, nil
}

// updateReputationForSpam 更新发件人信誉（用户反馈）
func (s *spamService) updateReputationForSpam(ctx context.Context, senderEmail string, isSpam bool) {
	if s.reputationManager == nil {
		return
	}
	if err := s.reputationManager.UpdateReputationByUserFeedback(ctx, senderEmail, isSpam); err != nil {
		spamServiceLog.Warn("更新发件人信誉失败: sender=%s, err=%v", senderEmail, err)
	}
}

// addBayesianTraining 添加贝叶斯训练数据
func (s *spamService) addBayesianTraining(ctx context.Context, email *model.Email, isSpam bool) {
	if s.bayesianClassifier == nil {
		return
	}

	// 使用贝叶斯分类器的方法添加训练数据
	if err := s.bayesianClassifier.AddTrainingData(ctx, email.AccountUID, email, isSpam); err != nil {
		spamServiceLog.Warn("添加贝叶斯训练数据失败: emailId=%d, err=%v", email.ID, err)
	}
}

// GetBayesianStatus 获取贝叶斯模型状态
func (s *spamService) GetBayesianStatus(ctx context.Context, userUID string) (*spam.ModelStatus, error) {
	if s.bayesianClassifier == nil {
		return nil, fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.GetModelStatus(ctx, userUID)
}

// TrainBayesianModel 手动训练贝叶斯模型
func (s *spamService) TrainBayesianModel(ctx context.Context, userUID string) error {
	if s.bayesianClassifier == nil {
		return fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.Train(ctx, userUID)
}

// ResetBayesianModel 重置贝叶斯模型
func (s *spamService) ResetBayesianModel(ctx context.Context, userUID string) error {
	if s.bayesianClassifier == nil {
		return fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.Reset(ctx, userUID)
}

// GetBayesianTrainingStats 获取贝叶斯训练统计
func (s *spamService) GetBayesianTrainingStats(ctx context.Context, userUID string) (*spam.TrainingStats, error) {
	if s.bayesianClassifier == nil {
		return nil, fmt.Errorf("贝叶斯分类器未初始化")
	}
	return s.bayesianClassifier.GetTrainingStats(ctx, userUID)
}

// GetRules 获取规则列表
func (s *spamService) GetRules(ctx context.Context, category string, page, pageSize int) ([]*model.SpamRule, int64, error) {
	if s.ruleRepo == nil {
		return nil, 0, fmt.Errorf("规则仓库未初始化")
	}

	offset := (page - 1) * pageSize
	if category != "" {
		return s.ruleRepo.ListByCategory(ctx, category, offset, pageSize)
	}
	return s.ruleRepo.List(ctx, offset, pageSize)
}

// GetRuleByID 根据 ID 获取规则
func (s *spamService) GetRuleByID(ctx context.Context, id int64) (*model.SpamRule, error) {
	if s.ruleRepo == nil {
		return nil, fmt.Errorf("规则仓库未初始化")
	}
	return s.ruleRepo.FindByID(ctx, id)
}

// CreateRule 创建规则
func (s *spamService) CreateRule(ctx context.Context, rule *model.SpamRule) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("规则仓库未初始化")
	}

	// 验证规则模式
	if err := validateRulePattern(rule.Category, rule.Pattern); err != nil {
		return fmt.Errorf("规则模式验证失败: %w", err)
	}

	// 自定义规则不能是内置规则
	rule.IsBuiltin = false

	return s.ruleRepo.Create(ctx, rule)
}

// UpdateRule 更新规则
func (s *spamService) UpdateRule(ctx context.Context, rule *model.SpamRule) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("规则仓库未初始化")
	}

	// 检查规则是否存在
	existing, err := s.ruleRepo.FindByID(ctx, rule.ID)
	if err != nil {
		return fmt.Errorf("查询规则失败: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("规则不存在")
	}

	// 内置规则只能修改启用状态
	if existing.IsBuiltin {
		existing.Enabled = rule.Enabled
		return s.ruleRepo.Update(ctx, existing)
	}

	// 验证规则模式
	if err := validateRulePattern(rule.Category, rule.Pattern); err != nil {
		return fmt.Errorf("规则模式验证失败: %w", err)
	}

	// 保持内置标记不变
	rule.IsBuiltin = existing.IsBuiltin

	return s.ruleRepo.Update(ctx, rule)
}

// DeleteRule 删除规则
func (s *spamService) DeleteRule(ctx context.Context, id int64) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("规则仓库未初始化")
	}

	// 检查规则是否存在
	existing, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("查询规则失败: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("规则不存在")
	}

	// 内置规则不能删除
	if existing.IsBuiltin {
		return fmt.Errorf("内置规则不能删除")
	}

	return s.ruleRepo.Delete(ctx, id)
}

// ToggleRule 切换规则启用状态
func (s *spamService) ToggleRule(ctx context.Context, id int64) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("规则仓库未初始化")
	}

	// 检查规则是否存在
	existing, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("查询规则失败: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("规则不存在")
	}

	return s.ruleRepo.ToggleEnabled(ctx, id)
}

// TestRule 测试规则
func (s *spamService) TestRule(ctx context.Context, pattern, category, content string) (bool, []string, error) {
	// 验证规则模式
	if err := validateRulePattern(category, pattern); err != nil {
		return false, nil, fmt.Errorf("规则模式验证失败: %w", err)
	}

	// 根据类别执行匹配
	matched, matches := matchPattern(category, pattern, content)
	return matched, matches, nil
}

// GetRuleStats 获取规则统计
func (s *spamService) GetRuleStats(ctx context.Context) (*RuleStats, error) {
	if s.ruleRepo == nil {
		return nil, fmt.Errorf("规则仓库未初始化")
	}

	// 获取所有规则
	rules, err := s.ruleRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取规则失败: %w", err)
	}

	stats := &RuleStats{}
	for _, rule := range rules {
		stats.TotalCount++
		if rule.Enabled {
			stats.EnabledCount++
		} else {
			stats.DisabledCount++
		}
		if rule.IsBuiltin {
			stats.BuiltinCount++
		} else {
			stats.CustomCount++
		}
		stats.TotalHits += rule.HitCount
	}

	return stats, nil
}

// validateRulePattern 验证规则模式
func validateRulePattern(category, pattern string) error {
	if pattern == "" {
		return fmt.Errorf("规则模式不能为空")
	}

	switch category {
	case "keyword":
		// 关键词规则，直接字符串匹配，无需特殊验证
		return nil
	case "pattern":
		// 正则表达式规则，验证正则语法
		_, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("无效的正则表达式: %w", err)
		}
		return nil
	case "header", "content", "url", "attachment":
		// 这些类别可以使用关键词或正则
		// 如果以 / 开头和结尾，视为正则表达式
		if len(pattern) > 2 && pattern[0] == '/' && pattern[len(pattern)-1] == '/' {
			_, err := regexp.Compile(pattern[1 : len(pattern)-1])
			if err != nil {
				return fmt.Errorf("无效的正则表达式: %w", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("未知的规则类别: %s", category)
	}
}

// matchPattern 执行模式匹配
func matchPattern(category, pattern, content string) (bool, []string) {
	var matches []string

	switch category {
	case "keyword":
		// 关键词匹配（不区分大小写）
		lowerContent := strings.ToLower(content)
		lowerPattern := strings.ToLower(pattern)
		if strings.Contains(lowerContent, lowerPattern) {
			matches = append(matches, pattern)
			return true, matches
		}
		return false, nil

	case "pattern":
		// 正则表达式匹配
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, nil
		}
		found := re.FindAllString(content, -1)
		if len(found) > 0 {
			return true, found
		}
		return false, nil

	case "header", "content", "url", "attachment":
		// 支持关键词或正则
		if len(pattern) > 2 && pattern[0] == '/' && pattern[len(pattern)-1] == '/' {
			// 正则表达式
			re, err := regexp.Compile(pattern[1 : len(pattern)-1])
			if err != nil {
				return false, nil
			}
			found := re.FindAllString(content, -1)
			if len(found) > 0 {
				return true, found
			}
			return false, nil
		}
		// 关键词匹配
		lowerContent := strings.ToLower(content)
		lowerPattern := strings.ToLower(pattern)
		if strings.Contains(lowerContent, lowerPattern) {
			matches = append(matches, pattern)
			return true, matches
		}
		return false, nil

	default:
		return false, nil
	}
}
