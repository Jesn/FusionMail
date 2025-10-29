package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// RuleService 规则服务接口
type RuleService interface {
	// Create 创建规则
	Create(ctx context.Context, rule *model.EmailRule) error
	// Update 更新规则
	Update(ctx context.Context, rule *model.EmailRule) error
	// Delete 删除规则
	Delete(ctx context.Context, id int64) error
	// GetByID 根据 ID 获取规则
	GetByID(ctx context.Context, id int64) (*model.EmailRule, error)
	// ListByAccount 获取账户的规则列表
	ListByAccount(ctx context.Context, accountUID string) ([]*model.EmailRule, error)
	// ApplyRules 对邮件应用规则
	ApplyRules(ctx context.Context, email *model.Email) error
	// TestRule 测试规则是否匹配邮件
	TestRule(ctx context.Context, rule *model.EmailRule, email *model.Email) (bool, error)
}

type ruleService struct {
	ruleRepo  repository.RuleRepository
	emailRepo repository.EmailRepository
}

// NewRuleService 创建规则服务实例
func NewRuleService(
	ruleRepo repository.RuleRepository,
	emailRepo repository.EmailRepository,
) RuleService {
	return &ruleService{
		ruleRepo:  ruleRepo,
		emailRepo: emailRepo,
	}
}

// Create 创建规则
func (s *ruleService) Create(ctx context.Context, rule *model.EmailRule) error {
	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}
	return s.ruleRepo.Create(ctx, rule)
}

// Update 更新规则
func (s *ruleService) Update(ctx context.Context, rule *model.EmailRule) error {
	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}
	return s.ruleRepo.Update(ctx, rule)
}

// Delete 删除规则
func (s *ruleService) Delete(ctx context.Context, id int64) error {
	return s.ruleRepo.Delete(ctx, id)
}

// GetByID 根据 ID 获取规则
func (s *ruleService) GetByID(ctx context.Context, id int64) (*model.EmailRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("rule not found")
	}
	return rule, nil
}

// ListByAccount 获取账户的规则列表
func (s *ruleService) ListByAccount(ctx context.Context, accountUID string) ([]*model.EmailRule, error) {
	return s.ruleRepo.ListByAccount(ctx, accountUID)
}

// ApplyRules 对邮件应用规则
func (s *ruleService) ApplyRules(ctx context.Context, email *model.Email) error {
	rules, err := s.ruleRepo.ListByAccount(ctx, email.AccountUID)
	if err != nil {
		return fmt.Errorf("failed to get rules: %w", err)
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		matched, err := s.matchRule(rule, email)
		if err != nil {
			continue
		}

		if matched {
			if err := s.executeActions(ctx, rule, email); err != nil {
				continue
			}

			if err := s.ruleRepo.UpdateMatchCount(ctx, rule.ID); err != nil {
				// Log error but continue
			}

			if rule.StopProcessing {
				break
			}
		}
	}

	return nil
}

// TestRule 测试规则是否匹配邮件
func (s *ruleService) TestRule(ctx context.Context, rule *model.EmailRule, email *model.Email) (bool, error) {
	return s.matchRule(rule, email)
}

// validateRule 验证规则
func (s *ruleService) validateRule(rule *model.EmailRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.AccountUID == "" {
		return fmt.Errorf("account UID is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("at least one condition is required")
	}
	if len(rule.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	for _, cond := range rule.Conditions {
		if err := s.validateCondition(&cond); err != nil {
			return err
		}
	}

	for _, action := range rule.Actions {
		if err := s.validateAction(&action); err != nil {
			return err
		}
	}

	return nil
}

// validateCondition 验证条件
func (s *ruleService) validateCondition(cond *model.RuleCondition) error {
	validFields := map[string]bool{
		"from":           true,
		"to":             true,
		"subject":        true,
		"body":           true,
		"has_attachment": true,
	}

	if !validFields[cond.Field] {
		return fmt.Errorf("invalid condition field: %s", cond.Field)
	}

	validOperators := map[string]bool{
		"contains":     true,
		"not_contains": true,
		"equals":       true,
		"not_equals":   true,
		"starts_with":  true,
		"ends_with":    true,
		"regex":        true,
	}

	if !validOperators[cond.Operator] {
		return fmt.Errorf("invalid condition operator: %s", cond.Operator)
	}

	if cond.Operator == "regex" {
		if _, err := regexp.Compile(cond.Value); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	return nil
}

// validateAction 验证动作
func (s *ruleService) validateAction(action *model.RuleAction) error {
	validTypes := map[string]bool{
		"add_label":    true,
		"remove_label": true,
		"mark_read":    true,
		"mark_unread":  true,
		"star":         true,
		"unstar":       true,
		"archive":      true,
		"delete":       true,
		"move_folder":  true,
		"webhook":      true,
	}

	if !validTypes[action.Type] {
		return fmt.Errorf("invalid action type: %s", action.Type)
	}

	requiresValue := map[string]bool{
		"add_label":    true,
		"remove_label": true,
		"move_folder":  true,
		"webhook":      true,
	}

	if requiresValue[action.Type] && action.Value == "" {
		return fmt.Errorf("action %s requires a value", action.Type)
	}

	return nil
}

// matchRule 检查规则是否匹配邮件
func (s *ruleService) matchRule(rule *model.EmailRule, email *model.Email) (bool, error) {
	if len(rule.Conditions) == 0 {
		return false, nil
	}

	if rule.MatchMode == "all" {
		for _, cond := range rule.Conditions {
			matched, err := s.matchCondition(&cond, email)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil
	} else {
		for _, cond := range rule.Conditions {
			matched, err := s.matchCondition(&cond, email)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}
}

// matchCondition 检查单个条件是否匹配
func (s *ruleService) matchCondition(cond *model.RuleCondition, email *model.Email) (bool, error) {
	var fieldValue string
	switch cond.Field {
	case "from":
		fieldValue = email.FromAddress
	case "to":
		fieldValue = email.ToAddress
	case "subject":
		fieldValue = email.Subject
	case "body":
		fieldValue = email.TextBody
	case "has_attachment":
		if email.HasAttachment {
			fieldValue = "true"
		} else {
			fieldValue = "false"
		}
	default:
		return false, fmt.Errorf("unknown field: %s", cond.Field)
	}

	switch cond.Operator {
	case "contains":
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(cond.Value)), nil
	case "not_contains":
		return !strings.Contains(strings.ToLower(fieldValue), strings.ToLower(cond.Value)), nil
	case "equals":
		return strings.EqualFold(fieldValue, cond.Value), nil
	case "not_equals":
		return !strings.EqualFold(fieldValue, cond.Value), nil
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(fieldValue), strings.ToLower(cond.Value)), nil
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(fieldValue), strings.ToLower(cond.Value)), nil
	case "regex":
		re, err := regexp.Compile(cond.Value)
		if err != nil {
			return false, fmt.Errorf("invalid regex: %w", err)
		}
		return re.MatchString(fieldValue), nil
	default:
		return false, fmt.Errorf("unknown operator: %s", cond.Operator)
	}
}

// executeActions 执行规则动作
func (s *ruleService) executeActions(ctx context.Context, rule *model.EmailRule, email *model.Email) error {
	for _, action := range rule.Actions {
		if err := s.executeAction(ctx, &action, email); err != nil {
			// Continue with other actions even if one fails
			continue
		}
	}
	return nil
}

// executeAction 执行单个动作
func (s *ruleService) executeAction(ctx context.Context, action *model.RuleAction, email *model.Email) error {
	switch action.Type {
	case "mark_read":
		email.IsRead = true
		return s.emailRepo.Update(ctx, email)
	case "mark_unread":
		email.IsRead = false
		return s.emailRepo.Update(ctx, email)
	case "star":
		email.IsStarred = true
		return s.emailRepo.Update(ctx, email)
	case "unstar":
		email.IsStarred = false
		return s.emailRepo.Update(ctx, email)
	case "archive":
		email.IsArchived = true
		return s.emailRepo.Update(ctx, email)
	case "delete":
		email.IsDeleted = true
		return s.emailRepo.Update(ctx, email)
	case "move_folder":
		email.Folder = action.Value
		return s.emailRepo.Update(ctx, email)
	case "add_label", "remove_label", "webhook":
		// TODO: Implement these actions
		return nil
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}
