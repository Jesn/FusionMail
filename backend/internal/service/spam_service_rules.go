package service

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"regexp"
	"strings"
)

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
