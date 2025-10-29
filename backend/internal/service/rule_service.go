package service

import (
	"context"
	"encoding/json"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"log"
	"strings"
	"time"
)

// RuleService 规则引擎服务接口
type RuleService interface {
	// 规则管理
	CreateRule(ctx context.Context, rule *model.Rule) error
	GetRuleByID(ctx context.Context, id int64) (*model.Rule, error)
	ListRules(ctx context.Context, accountUID string) ([]*model.Rule, error)
	UpdateRule(ctx context.Context, rule *model.Rule) error
	DeleteRule(ctx context.Context, id int64) error
	ToggleRule(ctx context.Context, id int64) error

	// 规则执行
	ApplyRulesToEmail(ctx context.Context, email *model.Email) error
	ApplyRulesToAccount(ctx context.Context, accountUID string) error
}

// ruleService 规则引擎服务实现
type ruleService struct {
	ruleRepo    repository.RuleRepository
	emailRepo   repository.EmailRepository
	webhookRepo repository.WebhookRepository
}

// NewRuleService 创建规则引擎服务实例
func NewRuleService(
	ruleRepo repository.RuleRepository,
	emailRepo repository.EmailRepository,
	webhookRepo repository.WebhookRepository,
) RuleService {
	return &ruleService{
		ruleRepo:    ruleRepo,
		emailRepo:   emailRepo,
		webhookRepo: webhookRepo,
	}
}

// CreateRule 创建规则
func (s *ruleService) CreateRule(ctx context.Context, rule *model.Rule) error {
	// 验证规则
	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	return s.ruleRepo.Create(ctx, rule)
}

// GetRuleByID 根据 ID 获取规则
func (s *ruleService) GetRuleByID(ctx context.Context, id int64) (*model.Rule, error) {
	rule, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("rule not found")
	}
	return rule, nil
}

// ListRules 获取规则列表
func (s *ruleService) ListRules(ctx context.Context, accountUID string) ([]*model.Rule, error) {
	return s.ruleRepo.ListByAccount(ctx, accountUID)
}

// UpdateRule 更新规则
func (s *ruleService) UpdateRule(ctx context.Context, rule *model.Rule) error {
	// 验证规则
	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	return s.ruleRepo.Update(ctx, rule)
}

// DeleteRule 删除规则
func (s *ruleService) DeleteRule(ctx context.Context, id int64) error {
	return s.ruleRepo.Delete(ctx, id)
}

// ToggleRule 切换规则启用状态
func (s *ruleService) ToggleRule(ctx context.Context, id int64) error {
	rule, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		return fmt.Errorf("rule not found")
	}

	rule.Enabled = !rule.Enabled
	return s.ruleRepo.Update(ctx, rule)
}

// ApplyRulesToEmail 对单封邮件应用规则
func (s *ruleService) ApplyRulesToEmail(ctx context.Context, email *model.Email) error {
	// 获取该账户的所有启用规则
	rules, err := s.ruleRepo.ListByAccount(ctx, email.AccountUID)
	if err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	// 按优先级排序并应用规则
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 检查规则条件是否匹配
		if s.matchRule(rule, email) {
			log.Printf("Rule %d matched for email %d", rule.ID, email.ID)

			// 执行规则动作
			if err := s.executeRuleActions(ctx, rule, email); err != nil {
				log.Printf("Failed to execute rule %d actions: %v", rule.ID, err)
				continue
			}

			// 更新规则执行统计
			rule.ExecutionCount++
			rule.LastExecutedAt = timePtr(time.Now())
			if err := s.ruleRepo.Update(ctx, rule); err != nil {
				log.Printf("Failed to update rule stats: %v", err)
			}

			// 如果规则设置为停止后续规则，则退出
			if rule.StopProcessing {
				break
			}
		}
	}

	return nil
}

// ApplyRulesToAccount 对账户的所有邮件应用规则
func (s *ruleService) ApplyRulesToAccount(ctx context.Context, accountUID string) error {
	// 获取账户的所有未删除邮件
	filter := &repository.EmailFilter{
		AccountUID: accountUID,
	}
	falseVal := false
	filter.IsDeleted = &falseVal

	emails, _, err := s.emailRepo.List(ctx, filter, 0, 10000) // 限制最多处理 10000 封
	if err != nil {
		return fmt.Errorf("failed to list emails: %w", err)
	}

	log.Printf("Applying rules to %d emails in account %s", len(emails), accountUID)

	// 对每封邮件应用规则
	for _, email := range emails {
		if err := s.ApplyRulesToEmail(ctx, email); err != nil {
			log.Printf("Failed to apply rules to email %d: %v", email.ID, err)
		}
	}

	return nil
}

// matchRule 检查邮件是否匹配规则条件
func (s *ruleService) matchRule(rule *model.Rule, email *model.Email) bool {
	// 解析条件 JSON
	var conditions []map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Conditions), &conditions); err != nil {
		log.Printf("Failed to parse rule conditions: %v", err)
		return false
	}

	// 检查每个条件
	for _, condition := range conditions {
		field, _ := condition["field"].(string)
		operator, _ := condition["operator"].(string)
		value, _ := condition["value"].(string)

		if !s.matchCondition(field, operator, value, email) {
			return false
		}
	}

	return true
}

// matchCondition 检查单个条件是否匹配
func (s *ruleService) matchCondition(field, operator, value string, email *model.Email) bool {
	var fieldValue string

	// 获取字段值
	switch field {
	case "from_address":
		fieldValue = email.FromAddress
	case "from_name":
		fieldValue = email.FromName
	case "subject":
		fieldValue = email.Subject
	case "body":
		fieldValue = email.TextBody
	case "to_addresses":
		fieldValue = email.ToAddresses
	default:
		return false
	}

	// 应用操作符
	switch operator {
	case "contains":
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(value))
	case "not_contains":
		return !strings.Contains(strings.ToLower(fieldValue), strings.ToLower(value))
	case "equals":
		return strings.EqualFold(fieldValue, value)
	case "not_equals":
		return !strings.EqualFold(fieldValue, value)
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(fieldValue), strings.ToLower(value))
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(fieldValue), strings.ToLower(value))
	default:
		return false
	}
}

// executeRuleActions 执行规则动作
func (s *ruleService) executeRuleActions(ctx context.Context, rule *model.Rule, email *model.Email) error {
	// 解析动作 JSON
	var actions []map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Actions), &actions); err != nil {
		return fmt.Errorf("failed to parse rule actions: %w", err)
	}

	// 执行每个动作
	for _, action := range actions {
		actionType, _ := action["type"].(string)
		actionValue, _ := action["value"].(string)

		if err := s.executeAction(ctx, actionType, actionValue, email); err != nil {
			log.Printf("Failed to execute action %s: %v", actionType, err)
			continue
		}
	}

	return nil
}

// executeAction 执行单个动作
func (s *ruleService) executeAction(ctx context.Context, actionType, actionValue string, email *model.Email) error {
	switch actionType {
	case "mark_read":
		trueVal := true
		return s.emailRepo.UpdateLocalStatus(ctx, email.ID, &trueVal, nil, nil, nil)

	case "mark_unread":
		falseVal := false
		return s.emailRepo.UpdateLocalStatus(ctx, email.ID, &falseVal, nil, nil, nil)

	case "star":
		trueVal := true
		return s.emailRepo.UpdateLocalStatus(ctx, email.ID, nil, &trueVal, nil, nil)

	case "archive":
		trueVal := true
		return s.emailRepo.UpdateLocalStatus(ctx, email.ID, nil, nil, &trueVal, nil)

	case "delete":
		trueVal := true
		return s.emailRepo.UpdateLocalStatus(ctx, email.ID, nil, nil, nil, &trueVal)

	case "add_label":
		// TODO: 实现标签功能
		log.Printf("Add label action not yet implemented: %s", actionValue)
		return nil

	case "trigger_webhook":
		// TODO: 实现 Webhook 触发
		log.Printf("Trigger webhook action not yet implemented: %s", actionValue)
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", actionType)
	}
}

// validateRule 验证规则
func (s *ruleService) validateRule(rule *model.Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if rule.AccountUID == "" {
		return fmt.Errorf("account_uid is required")
	}

	// 验证条件 JSON
	var conditions []map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Conditions), &conditions); err != nil {
		return fmt.Errorf("invalid conditions JSON: %w", err)
	}

	if len(conditions) == 0 {
		return fmt.Errorf("at least one condition is required")
	}

	// 验证动作 JSON
	var actions []map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Actions), &actions); err != nil {
		return fmt.Errorf("invalid actions JSON: %w", err)
	}

	if len(actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	return nil
}

// 辅助函数
func timePtr(t time.Time) *time.Time {
	return &t
}
