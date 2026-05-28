package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RuleEngine 规则引擎
type RuleEngine struct {
	ruleRepo     repository.SpamRuleRepository
	redisClient  *redis.Client
	surblChecker *SURBLChecker
	cacheManager *CacheManager
	cacheTTL     time.Duration
}

// RuleEngineResult 规则引擎检测结果
type RuleEngineResult struct {
	Score       int           // 总评分
	HitRules    []*HitRule    // 命中的规则
	CheckedTime time.Duration // 检测耗时
	SURBLResult *SURBLResult  // SURBL 检测结果
}

// HitRule 命中的规则
type HitRule struct {
	RuleID      int64  // 规则 ID
	RuleName    string // 规则名称
	Category    string // 规则类别
	Score       int    // 评分
	MatchedText string // 匹配的文本
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine(
	ruleRepo repository.SpamRuleRepository,
	redisClient *redis.Client,
	surblChecker *SURBLChecker,
) *RuleEngine {
	return &RuleEngine{
		ruleRepo:     ruleRepo,
		redisClient:  redisClient,
		surblChecker: surblChecker,
		cacheTTL:     10 * time.Minute, // 规则缓存 10 分钟
	}
}

// NewRuleEngineWithCache 创建带缓存管理器的规则引擎
func NewRuleEngineWithCache(
	ruleRepo repository.SpamRuleRepository,
	redisClient *redis.Client,
	surblChecker *SURBLChecker,
	cacheManager *CacheManager,
) *RuleEngine {
	return &RuleEngine{
		ruleRepo:     ruleRepo,
		redisClient:  redisClient,
		surblChecker: surblChecker,
		cacheManager: cacheManager,
		cacheTTL:     10 * time.Minute,
	}
}

// Check 执行规则检测
func (r *RuleEngine) Check(ctx context.Context, email *model.Email) (*RuleEngineResult, error) {
	startTime := time.Now()

	result := &RuleEngineResult{
		Score:    0,
		HitRules: make([]*HitRule, 0),
	}

	// 1. 加载启用的规则
	rules, err := r.loadRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	// 2. 执行规则匹配
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		matched, matchedText := r.matchRule(rule, email)
		if matched {
			result.HitRules = append(result.HitRules, &HitRule{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Category:    rule.Category,
				Score:       rule.Score,
				MatchedText: matchedText,
			})
			result.Score += rule.Score

			// 异步更新规则命中次数
			go r.incrementHitCount(context.Background(), rule.ID)
		}
	}

	// 3. 执行 SURBL URL 黑名单检查
	if r.surblChecker != nil {
		surblResult, err := r.surblChecker.CheckWithFallback(ctx, email.Subject, email.TextBody)
		if err == nil && surblResult != nil {
			result.SURBLResult = surblResult
			result.Score += surblResult.Score

			// 记录 SURBL 命中的 URL
			for _, url := range surblResult.ListedURLs {
				result.HitRules = append(result.HitRules, &HitRule{
					RuleID:      0, // SURBL 不是数据库规则
					RuleName:    "SURBL URL 黑名单",
					Category:    "url",
					Score:       30,
					MatchedText: url,
				})
			}
		}
	}

	result.CheckedTime = time.Since(startTime)
	return result, nil
}

// loadRules 加载规则（带缓存）
func (r *RuleEngine) loadRules(ctx context.Context) ([]*model.SpamRule, error) {
	// 1. 尝试从缓存获取
	if r.cacheManager != nil {
		if cached, ok := r.cacheManager.GetRules(ctx); ok {
			// 将缓存的规则转换为 model.SpamRule
			rules := make([]*model.SpamRule, len(cached.Rules))
			for i, cr := range cached.Rules {
				rules[i] = &model.SpamRule{
					ID:       cr.ID,
					Name:     cr.Name,
					Category: cr.Category,
					Pattern:  cr.Pattern,
					Score:    cr.Score,
					Enabled:  cr.Enabled,
				}
			}
			return rules, nil
		}
	}

	// 2. 从数据库加载
	rules, err := r.ruleRepo.FindEnabled(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 缓存规则
	if r.cacheManager != nil && len(rules) > 0 {
		cachedRules := &CachedRules{
			Rules:    make([]CachedRule, len(rules)),
			CachedAt: time.Now(),
		}
		for i, rule := range rules {
			cachedRules.Rules[i] = CachedRule{
				ID:       rule.ID,
				Name:     rule.Name,
				Category: rule.Category,
				Pattern:  rule.Pattern,
				Score:    rule.Score,
				Enabled:  rule.Enabled,
			}
		}
		r.cacheManager.SetRules(ctx, cachedRules)
	}

	return rules, nil
}

// matchRule 匹配单个规则
func (r *RuleEngine) matchRule(rule *model.SpamRule, email *model.Email) (bool, string) {
	switch rule.Category {
	case "keyword":
		return r.matchKeyword(rule, email)
	case "pattern":
		return r.matchPattern(rule, email)
	case "header":
		return r.matchHeader(rule, email)
	case "content":
		return r.matchContent(rule, email)
	case "url":
		return r.matchURL(rule, email)
	case "attachment":
		return r.matchAttachment(rule, email)
	default:
		return false, ""
	}
}

// matchKeyword 匹配关键词
func (r *RuleEngine) matchKeyword(rule *model.SpamRule, email *model.Email) (bool, string) {
	keyword := strings.ToLower(rule.Pattern)

	// 检查主题
	if strings.Contains(strings.ToLower(email.Subject), keyword) {
		return true, fmt.Sprintf("主题包含: %s", keyword)
	}

	// 检查正文
	if strings.Contains(strings.ToLower(email.TextBody), keyword) {
		return true, fmt.Sprintf("正文包含: %s", keyword)
	}

	return false, ""
}

// matchPattern 匹配正则表达式
func (r *RuleEngine) matchPattern(rule *model.SpamRule, email *model.Email) (bool, string) {
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return false, ""
	}

	// 检查主题
	if re.MatchString(email.Subject) {
		return true, fmt.Sprintf("主题匹配模式: %s", rule.Pattern)
	}

	// 检查正文
	if re.MatchString(email.TextBody) {
		return true, fmt.Sprintf("正文匹配模式: %s", rule.Pattern)
	}

	return false, ""
}

// matchHeader 匹配邮件头
func (r *RuleEngine) matchHeader(rule *model.SpamRule, email *model.Email) (bool, string) {
	pattern := strings.ToLower(rule.Pattern)

	// 检查发件人地址
	if strings.Contains(strings.ToLower(email.FromAddress), pattern) {
		return true, fmt.Sprintf("发件人地址包含: %s", pattern)
	}

	return false, ""
}

// matchContent 匹配内容特征
func (r *RuleEngine) matchContent(rule *model.SpamRule, email *model.Email) (bool, string) {
	text := email.TextBody
	if text == "" {
		return false, ""
	}

	// 根据规则名称判断检测类型
	if strings.Contains(rule.Name, "大写字母") {
		ratio := r.calculateUppercaseRatio(text)
		threshold := 0.3 // 默认阈值 30%
		if ratio > threshold {
			return true, fmt.Sprintf("大写字母比例: %.1f%%", ratio*100)
		}
	}

	if strings.Contains(rule.Name, "特殊字符") {
		ratio := r.calculateSpecialCharRatio(text)
		threshold := 0.2 // 默认阈值 20%
		if ratio > threshold {
			return true, fmt.Sprintf("特殊字符比例: %.1f%%", ratio*100)
		}
	}

	if strings.Contains(rule.Name, "HTML 标签") {
		count := r.countHTMLTags(email.HTMLBody)
		threshold := 100 // 默认阈值 100 个标签
		if count > threshold {
			return true, fmt.Sprintf("HTML 标签数量: %d", count)
		}
	}

	return false, ""
}

// matchURL 匹配 URL 特征
func (r *RuleEngine) matchURL(rule *model.SpamRule, email *model.Email) (bool, string) {
	text := email.Subject + " " + email.TextBody

	// 链接数量检测
	if strings.Contains(rule.Name, "链接数量") {
		count := r.countURLs(text)
		threshold := 5 // 默认阈值 5 个链接
		if count > threshold {
			return true, fmt.Sprintf("链接数量: %d", count)
		}
	}

	// 短链接检测
	if strings.Contains(rule.Name, "短链接") {
		domain := rule.Pattern
		if strings.Contains(text, domain) {
			return true, fmt.Sprintf("包含短链接域名: %s", domain)
		}
	}

	return false, ""
}

// matchAttachment 匹配附件特征
func (r *RuleEngine) matchAttachment(rule *model.SpamRule, email *model.Email) (bool, string) {
	if !email.HasAttachment {
		return false, ""
	}

	// 可执行附件检测
	if strings.Contains(rule.Name, "可执行附件") {
		// 这里简化处理，实际应该查询附件表
		// 假设附件信息存储在某个字段中
		// ext := rule.Pattern
		return false, ""
	}

	// 图片附件过多检测
	if strings.Contains(rule.Name, "图片附件") {
		// 这里简化处理，实际应该查询附件表统计图片数量
		return false, ""
	}

	return false, ""
}

// calculateUppercaseRatio 计算大写字母比例
func (r *RuleEngine) calculateUppercaseRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	uppercaseCount := 0
	letterCount := 0

	for _, char := range text {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') {
			letterCount++
			if char >= 'A' && char <= 'Z' {
				uppercaseCount++
			}
		}
	}

	if letterCount == 0 {
		return 0
	}

	return float64(uppercaseCount) / float64(letterCount)
}

// calculateSpecialCharRatio 计算特殊字符比例
func (r *RuleEngine) calculateSpecialCharRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	specialCount := 0

	for _, char := range text {
		// 特殊字符：非字母、数字、空格、常见标点
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == ' ' || char == '.' || char == ',' ||
			char == '!' || char == '?' || char == '\n' || char == '\r') {
			specialCount++
		}
	}

	return float64(specialCount) / float64(len(text))
}

// countHTMLTags 统计 HTML 标签数量
func (r *RuleEngine) countHTMLTags(html string) int {
	re := regexp.MustCompile(`<[^>]+>`)
	matches := re.FindAllString(html, -1)
	return len(matches)
}

// countURLs 统计 URL 数量
func (r *RuleEngine) countURLs(text string) int {
	re := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	matches := re.FindAllString(text, -1)
	return len(matches)
}

// incrementHitCount 增加规则命中次数
func (r *RuleEngine) incrementHitCount(ctx context.Context, ruleID int64) {
	if err := r.ruleRepo.IncrementHitCount(ctx, ruleID); err != nil {
		// 记录错误但不影响主流程
		fmt.Printf("警告: 更新规则命中次数失败 [规则ID: %d]: %v\n", ruleID, err)
	}
}

// InvalidateCache 使缓存失效
func (r *RuleEngine) InvalidateCache(ctx context.Context) error {
	// 优先使用新的缓存管理器
	if r.cacheManager != nil {
		return r.cacheManager.InvalidateRulesCache(ctx)
	}

	// 兼容旧的缓存方式
	cacheKey := "spam:rules:enabled"
	return r.redisClient.Del(ctx, cacheKey).Err()
}
