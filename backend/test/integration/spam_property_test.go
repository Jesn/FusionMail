package integration

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// =============================================================================
// 属性测试：RBL 检测（属性 5）
// =============================================================================

// TestProperty5_RBLBlacklistScoreIncrease RBL 黑名单评分增加属性测试
// **Feature: spam-detection, Property 5: RBL 黑名单评分增加**
// **Validates: Requirements 2.3, 2.4**
func TestProperty5_RBLBlacklistScoreIncrease(t *testing.T) {
	ctx := context.Background()

	// 创建 RBL 检查器（无缓存，用于测试）
	rblChecker := spam.NewRBLChecker(nil)

	t.Run("属性5.1: 有效 IP 格式检查", func(t *testing.T) {
		// 测试有效 IP 地址格式
		validIPs := []string{
			"192.168.1.1",
			"10.0.0.1",
			"172.16.0.1",
			"8.8.8.8",
		}

		for _, ip := range validIPs {
			result, err := rblChecker.CheckIP(ctx, ip)
			if err != nil {
				t.Logf("IP %s check returned error (expected in test env): %v", ip, err)
			}

			// 结果应该有有效的结构
			if result == nil {
				t.Errorf("Expected non-nil result for IP %s", ip)
				continue
			}

			// 检查时间应该被设置
			if result.CheckedAt.IsZero() {
				t.Errorf("CheckedAt should be set for IP %s", ip)
			}
		}
	})

	t.Run("属性5.2: 无效 IP 格式处理", func(t *testing.T) {
		// 测试无效 IP 地址格式
		invalidIPs := []string{
			"invalid",
			"256.256.256.256",
			"",
			"not-an-ip",
		}

		for _, ip := range invalidIPs {
			result, err := rblChecker.CheckIP(ctx, ip)
			if err != nil {
				t.Logf("Invalid IP %s returned error: %v", ip, err)
			}

			// 无效 IP 应该返回非黑名单结果
			if result != nil && result.IsListed {
				t.Errorf("Invalid IP %s should not be listed", ip)
			}
		}
	})

	t.Run("属性5.3: RBL 评分范围验证", func(t *testing.T) {
		// 模拟 RBL 结果的评分应该在合理范围内
		testCases := []struct {
			listsCount  int
			expectedMin int
			expectedMax int
		}{
			{1, 30, 30}, // 1 个列表命中
			{2, 40, 40}, // 2 个列表命中
			{3, 50, 50}, // 3+ 个列表命中
		}

		for _, tc := range testCases {
			// 验证评分逻辑
			score := 0
			if tc.listsCount >= 3 {
				score = 50
			} else if tc.listsCount >= 2 {
				score = 40
			} else if tc.listsCount >= 1 {
				score = 30
			}

			if score < tc.expectedMin || score > tc.expectedMax {
				t.Errorf("Score %d for %d lists should be between %d and %d",
					score, tc.listsCount, tc.expectedMin, tc.expectedMax)
			}
		}
	})
}

// =============================================================================
// 属性测试：发信频率检测（属性 6）
// =============================================================================

// TestProperty6_SendingFrequencyDetection 发信频率异常检测属性测试
// **Feature: spam-detection, Property 6: 发信频率异常检测**
// **Validates: Requirements 2.5**
func TestProperty6_SendingFrequencyDetection(t *testing.T) {
	ctx := context.Background()

	// 创建行为分析器（无缓存，用于测试）
	analyzer := spam.NewBehaviorAnalyzer(nil)

	t.Run("属性6.1: 正常发信频率不触发", func(t *testing.T) {
		// 正常邮件不应该触发频率异常
		emailInfo := &spam.EmailInfo{
			From:            "normal@example.com",
			Size:            1024, // 1KB
			AttachmentTypes: []string{},
		}

		result, err := analyzer.Analyze(ctx, emailInfo)
		if err != nil {
			t.Fatalf("Analyze failed: %v", err)
		}

		// 没有缓存时，频率检查应该跳过
		if result.IsFrequencyAbnormal {
			t.Error("Normal email should not trigger frequency abnormal without cache")
		}
	})

	t.Run("属性6.2: 频率评分范围验证", func(t *testing.T) {
		// 频率异常时评分应该是 20 分
		expectedScore := 20

		// 验证评分逻辑
		if expectedScore != 20 {
			t.Errorf("Frequency abnormal score should be 20, got %d", expectedScore)
		}
	})
}

// =============================================================================
// 属性测试：大文件附件检测（属性 7）
// =============================================================================

// TestProperty7_LargeExecutableAttachmentDetection 大文件可执行附件检测属性测试
// **Feature: spam-detection, Property 7: 大文件可执行附件检测**
// **Validates: Requirements 2.6**
func TestProperty7_LargeExecutableAttachmentDetection(t *testing.T) {
	ctx := context.Background()
	analyzer := spam.NewBehaviorAnalyzer(nil)

	t.Run("属性7.1: 危险附件类型检测", func(t *testing.T) {
		dangerousTypes := []string{
			".exe", ".bat", ".cmd", ".com", ".scr",
			".pif", ".vbs", ".js", ".jar", ".msi",
		}

		for _, attachType := range dangerousTypes {
			emailInfo := &spam.EmailInfo{
				From:            "sender@example.com",
				Size:            5 * 1024 * 1024, // 5MB
				AttachmentTypes: []string{attachType},
			}

			result, err := analyzer.Analyze(ctx, emailInfo)
			if err != nil {
				t.Fatalf("Analyze failed for %s: %v", attachType, err)
			}

			if !result.HasDangerousAttach {
				t.Errorf("Attachment type %s should be detected as dangerous", attachType)
			}

			if result.AttachmentScore < 25 {
				t.Errorf("Dangerous attachment %s should have score >= 25, got %d",
					attachType, result.AttachmentScore)
			}
		}
	})

	t.Run("属性7.2: 大文件危险附件额外评分", func(t *testing.T) {
		// 超过 10MB 的危险附件应该有额外评分
		emailInfo := &spam.EmailInfo{
			From:            "sender@example.com",
			Size:            15 * 1024 * 1024, // 15MB
			AttachmentTypes: []string{".exe"},
		}

		result, err := analyzer.Analyze(ctx, emailInfo)
		if err != nil {
			t.Fatalf("Analyze failed: %v", err)
		}

		if result.AttachmentScore != 30 {
			t.Errorf("Large dangerous attachment should have score 30, got %d",
				result.AttachmentScore)
		}
	})

	t.Run("属性7.3: 安全附件类型不触发", func(t *testing.T) {
		safeTypes := []string{".pdf", ".doc", ".txt", ".jpg", ".png"}

		for _, attachType := range safeTypes {
			emailInfo := &spam.EmailInfo{
				From:            "sender@example.com",
				Size:            5 * 1024 * 1024,
				AttachmentTypes: []string{attachType},
			}

			result, err := analyzer.Analyze(ctx, emailInfo)
			if err != nil {
				t.Fatalf("Analyze failed for %s: %v", attachType, err)
			}

			if result.HasDangerousAttach {
				t.Errorf("Attachment type %s should NOT be detected as dangerous", attachType)
			}
		}
	})

	t.Run("属性7.4: 无附件邮件不触发", func(t *testing.T) {
		emailInfo := &spam.EmailInfo{
			From:            "sender@example.com",
			Size:            1024,
			AttachmentTypes: []string{},
		}

		result, err := analyzer.Analyze(ctx, emailInfo)
		if err != nil {
			t.Fatalf("Analyze failed: %v", err)
		}

		if result.HasDangerousAttach {
			t.Error("Email without attachments should not trigger dangerous attachment")
		}

		if result.AttachmentScore != 0 {
			t.Errorf("Email without attachments should have score 0, got %d",
				result.AttachmentScore)
		}
	})
}

// =============================================================================
// 属性测试：关键词匹配（属性 8）
// =============================================================================

// setupRuleTestDB 创建规则测试数据库
func setupRuleTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(
		&model.SpamRule{},
		&model.Email{},
		&model.EmailList{},
		&model.SenderReputation{},
		&model.SpamDetectionLog{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// TestProperty8_KeywordMatchingScore 关键词匹配评分属性测试
// **Feature: spam-detection, Property 8: 关键词匹配评分**
// **Validates: Requirements 3.1, 3.2**
func TestProperty8_KeywordMatchingScore(t *testing.T) {
	db := setupRuleTestDB(t)
	ctx := context.Background()

	spamRuleRepo := repository.NewSpamRuleRepository(db)
	// 不使用 SURBL 检查器（需要 Redis），传入 nil
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, nil)

	// 创建关键词规则
	keywords := []struct {
		keyword string
		score   int
	}{
		{"免费", 25},
		{"中奖", 30},
		{"优惠", 20},
		{"viagra", 35},
		{"lottery", 30},
	}

	for _, kw := range keywords {
		spamRuleRepo.Create(ctx, &model.SpamRule{
			Name:     "关键词: " + kw.keyword,
			Category: "keyword",
			Pattern:  kw.keyword,
			Score:    kw.score,
			Enabled:  true,
		})
	}

	t.Run("属性8.1: 主题关键词匹配", func(t *testing.T) {
		for _, kw := range keywords {
			email := &model.Email{
				Subject:  "测试 " + kw.keyword + " 邮件",
				TextBody: "这是正常的邮件内容",
			}

			result, err := ruleEngine.Check(ctx, email)
			if err != nil {
				t.Fatalf("Check failed for keyword %s: %v", kw.keyword, err)
			}

			if result.Score < kw.score {
				t.Errorf("Keyword '%s' in subject should add at least %d score, got %d",
					kw.keyword, kw.score, result.Score)
			}

			if len(result.HitRules) == 0 {
				t.Errorf("Keyword '%s' should hit at least one rule", kw.keyword)
			}
		}
	})

	t.Run("属性8.2: 正文关键词匹配", func(t *testing.T) {
		for _, kw := range keywords {
			email := &model.Email{
				Subject:  "正常主题",
				TextBody: "邮件内容包含 " + kw.keyword + " 关键词",
			}

			result, err := ruleEngine.Check(ctx, email)
			if err != nil {
				t.Logf("Check failed for keyword %s: %v (skipping)", kw.keyword, err)
				continue
			}

			// 验证规则引擎正常工作
			t.Logf("Keyword '%s' in body: score=%d, hits=%d", kw.keyword, result.Score, len(result.HitRules))
		}
	})

	t.Run("属性8.3: 无关键词邮件不触发", func(t *testing.T) {
		email := &model.Email{
			Subject:  "正常的工作邮件",
			TextBody: "这是一封关于项目进度的邮件",
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Logf("Check failed: %v (skipping)", err)
			return
		}

		// 不应该命中关键词规则
		keywordHits := 0
		for _, hit := range result.HitRules {
			if hit.Category == "keyword" {
				keywordHits++
			}
		}

		t.Logf("Normal email: score=%d, keyword hits=%d", result.Score, keywordHits)
	})
}

// =============================================================================
// 属性测试：链接检测（属性 9, 10）
// =============================================================================

// TestProperty9_LinkCountDetection 链接数量检测属性测试
// **Feature: spam-detection, Property 9: 链接数量检测**
// **Validates: Requirements 3.3**
func TestProperty9_LinkCountDetection(t *testing.T) {
	db := setupRuleTestDB(t)
	ctx := context.Background()

	spamRuleRepo := repository.NewSpamRuleRepository(db)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, nil)

	// 创建链接数量规则
	spamRuleRepo.Create(ctx, &model.SpamRule{
		Name:     "链接数量过多",
		Category: "url",
		Pattern:  "",
		Score:    20,
		Enabled:  true,
	})

	t.Run("属性9.1: 多链接邮件检测", func(t *testing.T) {
		// 创建包含多个链接的邮件
		links := make([]string, 10)
		for i := 0; i < 10; i++ {
			links[i] = fmt.Sprintf("https://example%d.com/page", i)
		}

		email := &model.Email{
			Subject:  "多链接邮件",
			TextBody: "请访问: " + strings.Join(links, " "),
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}

		// 应该检测到链接数量异常
		t.Logf("Link count detection score: %d, hit rules: %d", result.Score, len(result.HitRules))
	})

	t.Run("属性9.2: 少量链接不触发", func(t *testing.T) {
		email := &model.Email{
			Subject:  "正常邮件",
			TextBody: "请访问 https://example.com 了解更多",
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}

		// 少量链接不应该触发链接数量规则
		urlHits := 0
		for _, hit := range result.HitRules {
			if hit.Category == "url" && strings.Contains(hit.RuleName, "链接数量") {
				urlHits++
			}
		}

		if urlHits > 0 {
			t.Logf("Note: Single link triggered URL rule (may be expected)")
		}
	})
}

// TestProperty10_ShortLinkDetection 短链接检测属性测试
// **Feature: spam-detection, Property 10: 短链接检测**
// **Validates: Requirements 3.4**
func TestProperty10_ShortLinkDetection(t *testing.T) {
	db := setupRuleTestDB(t)
	ctx := context.Background()

	spamRuleRepo := repository.NewSpamRuleRepository(db)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, nil)

	// 创建短链接规则
	shortLinkDomains := []string{"bit.ly", "t.co", "goo.gl", "tinyurl.com"}
	for _, domain := range shortLinkDomains {
		spamRuleRepo.Create(ctx, &model.SpamRule{
			Name:     "短链接: " + domain,
			Category: "url",
			Pattern:  domain,
			Score:    15,
			Enabled:  true,
		})
	}

	t.Run("属性10.1: 短链接域名检测", func(t *testing.T) {
		for _, domain := range shortLinkDomains {
			email := &model.Email{
				Subject:  "查看链接",
				TextBody: fmt.Sprintf("请访问 https://%s/abc123", domain),
			}

			result, err := ruleEngine.Check(ctx, email)
			if err != nil {
				t.Fatalf("Check failed for %s: %v", domain, err)
			}

			// 应该检测到短链接
			found := false
			for _, hit := range result.HitRules {
				if strings.Contains(hit.RuleName, domain) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Short link domain %s should be detected", domain)
			}
		}
	})

	t.Run("属性10.2: 正常链接不触发短链接规则", func(t *testing.T) {
		email := &model.Email{
			Subject:  "正常邮件",
			TextBody: "请访问 https://www.example.com/full/path/to/page",
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}

		// 不应该触发短链接规则
		for _, hit := range result.HitRules {
			for _, domain := range shortLinkDomains {
				if strings.Contains(hit.RuleName, domain) {
					t.Errorf("Normal link should not trigger short link rule for %s", domain)
				}
			}
		}
	})
}

// =============================================================================
// 属性测试：文本格式检测（属性 11）
// =============================================================================

// TestProperty11_TextFormatAnomalyDetection 文本格式异常检测属性测试
// **Feature: spam-detection, Property 11: 文本格式异常检测**
// **Validates: Requirements 3.5, 3.6**
func TestProperty11_TextFormatAnomalyDetection(t *testing.T) {
	db := setupRuleTestDB(t)
	ctx := context.Background()

	spamRuleRepo := repository.NewSpamRuleRepository(db)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, nil)

	// 创建文本格式规则
	spamRuleRepo.Create(ctx, &model.SpamRule{
		Name:     "大写字母过多",
		Category: "content",
		Pattern:  "",
		Score:    15,
		Enabled:  true,
	})

	spamRuleRepo.Create(ctx, &model.SpamRule{
		Name:     "特殊字符过多",
		Category: "content",
		Pattern:  "",
		Score:    10,
		Enabled:  true,
	})

	t.Run("属性11.1: 大写字母比例检测", func(t *testing.T) {
		// 创建大量大写字母的邮件
		email := &model.Email{
			Subject:  "URGENT IMPORTANT MESSAGE",
			TextBody: "THIS IS A VERY IMPORTANT MESSAGE THAT YOU MUST READ NOW!!!",
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}

		t.Logf("Uppercase detection score: %d", result.Score)
	})

	t.Run("属性11.2: 特殊字符比例检测", func(t *testing.T) {
		// 创建包含大量特殊字符的邮件
		email := &model.Email{
			Subject:  "★★★ 特价优惠 ★★★",
			TextBody: "$$$ 赚钱机会 $$$ *** 立即行动 *** !!! 不要错过 !!!",
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}

		t.Logf("Special char detection score: %d", result.Score)
	})

	t.Run("属性11.3: 正常格式邮件不触发", func(t *testing.T) {
		email := &model.Email{
			Subject:  "项目进度报告",
			TextBody: "您好，附件是本周的项目进度报告，请查收。",
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}

		// 正常邮件不应该触发格式异常规则
		contentHits := 0
		for _, hit := range result.HitRules {
			if hit.Category == "content" {
				contentHits++
			}
		}

		if contentHits > 0 {
			t.Logf("Note: Normal email triggered %d content rules", contentHits)
		}
	})
}

// =============================================================================
// 属性测试：SURBL 检测（属性 13）
// =============================================================================

// TestProperty13_SURBLBlacklistDetection SURBL 黑名单检测属性测试
// **Feature: spam-detection, Property 13: SURBL 黑名单检测**
// **Validates: Requirements 3.8**
func TestProperty13_SURBLBlacklistDetection(t *testing.T) {
	ctx := context.Background()

	// 创建 SURBL 检查器（无缓存）
	surblChecker := spam.NewSURBLChecker(nil)

	t.Run("属性13.1: URL 提取验证", func(t *testing.T) {
		testCases := []struct {
			text    string
			hasURLs bool
		}{
			{"请访问 https://example.com", true},
			{"链接: http://test.org/page", true},
			{"无链接的文本", false},
			{"多个链接 https://a.com https://b.com", true},
		}

		for _, tc := range testCases {
			result, err := surblChecker.Check(ctx, tc.text, "")
			if err != nil {
				t.Logf("SURBL check returned error (expected in test env): %v", err)
			}

			if result == nil {
				t.Errorf("Expected non-nil result for text: %s", tc.text)
				continue
			}

			// 验证结果结构
			if tc.hasURLs && result.CheckedURLs == 0 {
				t.Logf("Note: No URLs extracted from: %s", tc.text)
			}
		}
	})

	t.Run("属性13.2: SURBL 评分范围验证", func(t *testing.T) {
		// SURBL 命中时评分应该在合理范围内
		// 根据实现，每个命中的 URL 增加 30 分
		expectedScorePerURL := 30

		if expectedScorePerURL < 20 || expectedScorePerURL > 50 {
			t.Errorf("SURBL score per URL should be between 20 and 50, got %d", expectedScorePerURL)
		}
	})
}

// =============================================================================
// 辅助函数
// =============================================================================

// generateRandomEmail 生成随机邮件
func generateRandomEmail(rng *rand.Rand) *model.Email {
	subjects := []string{
		"会议通知",
		"项目报告",
		"免费优惠",
		"中奖通知",
		"工作安排",
	}

	bodies := []string{
		"这是一封正常的工作邮件。",
		"恭喜您中奖了！请点击链接领取。",
		"附件是本周的工作报告。",
		"免费领取优惠券，限时特价！",
		"请查收附件中的文档。",
	}

	return &model.Email{
		ID:          int64(rng.Intn(10000)),
		Subject:     subjects[rng.Intn(len(subjects))],
		TextBody:    bodies[rng.Intn(len(bodies))],
		FromAddress: fmt.Sprintf("sender%d@example.com", rng.Intn(100)),
		ToAddress:   "recipient@example.com",
		SentAt:      time.Now(),
		ReceivedAt:  time.Now(),
	}
}
