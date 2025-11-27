package integration

import (
	"context"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSpamTestDB 创建垃圾邮件检测测试数据库
func setupSpamTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(
		&model.Email{},
		&model.EmailList{},
		&model.SenderReputation{},
		&model.SpamRule{},
		&model.BayesianTraining{},
		&model.SpamDetectionLog{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// TestSpamDetectionIntegration 垃圾邮件检测集成测试
func TestSpamDetectionIntegration(t *testing.T) {
	db := setupSpamTestDB(t)
	ctx := context.Background()

	// 创建仓库
	emailListRepo := repository.NewEmailListRepository(db)
	reputationRepo := repository.NewSenderReputationRepository(db)
	spamRuleRepo := repository.NewSpamRuleRepository(db)
	logRepo := repository.NewSpamDetectionLogRepository(db)

	// 创建组件
	whitelistChecker := spam.NewWhitelistChecker(emailListRepo, nil)
	rblChecker := spam.NewRBLChecker(nil)
	behaviorAnalyzer := spam.NewBehaviorAnalyzer(nil)
	preFilter := spam.NewPreFilter(rblChecker, behaviorAnalyzer)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, nil)
	reputationManager := spam.NewReputationManager(reputationRepo, nil)

	// 创建垃圾邮件检测器
	detector := spam.NewSpamDetector(
		whitelistChecker,
		preFilter,
		ruleEngine,
		reputationManager,
		nil,
		logRepo,
	)

	t.Run("正常邮件检测", func(t *testing.T) {
		email := &model.Email{
			ID:          1,
			ProviderID:  "test-email-001",
			AccountUID:  "test-account",
			Subject:     "Hello, how are you?",
			FromAddress: "friend@example.com",
			ToAddress:   "user@example.com",
			TextBody:    "Just wanted to say hi.",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		result, err := detector.Detect(ctx, email)
		if err != nil {
			t.Fatalf("Failed to detect spam: %v", err)
		}

		if result.IsSpam {
			t.Errorf("Normal email should not be spam, score: %d", result.Score)
		}
	})

	t.Run("检测性能测试", func(t *testing.T) {
		email := &model.Email{
			ID:          3,
			ProviderID:  "test-email-003",
			AccountUID:  "test-account",
			Subject:     "Performance test email",
			FromAddress: "test@example.com",
			ToAddress:   "user@example.com",
			TextBody:    "This is a test email.",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		result, err := detector.Detect(ctx, email)
		if err != nil {
			t.Fatalf("Failed to detect spam: %v", err)
		}

		// 检测应该在 200ms 内完成
		if result.CheckedTime > 200*time.Millisecond {
			t.Errorf("Detection should complete within 200ms, took: %v", result.CheckedTime)
		}
	})
}

// TestWhitelistBlacklistIntegration 白名单/黑名单集成测试
func TestWhitelistBlacklistIntegration(t *testing.T) {
	db := setupSpamTestDB(t)
	ctx := context.Background()

	emailListRepo := repository.NewEmailListRepository(db)
	whitelistChecker := spam.NewWhitelistChecker(emailListRepo, nil)

	userUID := "test-user-001"

	t.Run("白名单优先放行", func(t *testing.T) {
		whitelist := &model.EmailList{
			UserUID:    userUID,
			Type:       "whitelist",
			Target:     "trusted@example.com",
			TargetType: "email",
			Reason:     "Trusted sender",
		}
		err := emailListRepo.Create(ctx, whitelist)
		if err != nil {
			t.Fatalf("Failed to create whitelist: %v", err)
		}

		isWhitelisted, err := whitelistChecker.CheckWhitelist(ctx, userUID, "trusted@example.com")
		if err != nil {
			t.Fatalf("Failed to check whitelist: %v", err)
		}

		if !isWhitelisted {
			t.Error("Email should be whitelisted")
		}
	})

	t.Run("黑名单直接拦截", func(t *testing.T) {
		blacklist := &model.EmailList{
			UserUID:    userUID,
			Type:       "blacklist",
			Target:     "spam@malicious.com",
			TargetType: "email",
			Reason:     "Known spammer",
		}
		err := emailListRepo.Create(ctx, blacklist)
		if err != nil {
			t.Fatalf("Failed to create blacklist: %v", err)
		}

		isBlacklisted, err := whitelistChecker.CheckBlacklist(ctx, userUID, "spam@malicious.com")
		if err != nil {
			t.Fatalf("Failed to check blacklist: %v", err)
		}

		if !isBlacklisted {
			t.Error("Email should be blacklisted")
		}
	})

	t.Run("域名白名单", func(t *testing.T) {
		domainWhitelist := &model.EmailList{
			UserUID:    userUID,
			Type:       "whitelist",
			Target:     "company.com",
			TargetType: "domain",
			Reason:     "Company domain",
		}
		err := emailListRepo.Create(ctx, domainWhitelist)
		if err != nil {
			t.Fatalf("Failed to create domain whitelist: %v", err)
		}

		isWhitelisted, err := whitelistChecker.CheckWhitelist(ctx, userUID, "anyone@company.com")
		if err != nil {
			t.Fatalf("Failed to check domain whitelist: %v", err)
		}

		if !isWhitelisted {
			t.Error("Email from whitelisted domain should be whitelisted")
		}
	})
}

// TestReputationSystemIntegration 发件人信誉系统集成测试
func TestReputationSystemIntegration(t *testing.T) {
	db := setupSpamTestDB(t)
	ctx := context.Background()

	reputationRepo := repository.NewSenderReputationRepository(db)
	reputationManager := spam.NewReputationManager(reputationRepo, nil)

	t.Run("新发件人初始信誉", func(t *testing.T) {
		senderEmail := "newsender@example.com"

		reputation, err := reputationManager.GetOrCreateReputation(ctx, senderEmail)
		if err != nil {
			t.Fatalf("Failed to get reputation: %v", err)
		}

		if reputation.ReputationScore != 50 {
			t.Errorf("New sender should have initial score 50, got: %f", reputation.ReputationScore)
		}
	})

	t.Run("用户反馈调整信誉", func(t *testing.T) {
		senderEmail := "feedback@example.com"

		_, err := reputationManager.GetOrCreateReputation(ctx, senderEmail)
		if err != nil {
			t.Fatalf("Failed to create reputation: %v", err)
		}

		err = reputationManager.UpdateReputationByUserFeedback(ctx, senderEmail, true)
		if err != nil {
			t.Fatalf("Failed to update reputation: %v", err)
		}

		reputation, _ := reputationManager.GetOrCreateReputation(ctx, senderEmail)
		if reputation.ReputationScore >= 50 {
			t.Errorf("Reputation should decrease after spam feedback, got: %f", reputation.ReputationScore)
		}
	})

	t.Run("信誉影响评分", func(t *testing.T) {
		lowRepSender := "lowrep@example.com"
		reputation := &model.SenderReputation{
			Email:           lowRepSender,
			Domain:          "example.com",
			ReputationScore: 15,
			TrustLevel:      "suspicious",
			RBLStatus:       "unknown",
		}
		err := reputationRepo.Create(ctx, reputation)
		if err != nil {
			t.Fatalf("Failed to create low reputation: %v", err)
		}

		originalScore := 30
		adjustedScore, err := reputationManager.AdjustScoreByReputation(ctx, lowRepSender, originalScore)
		if err != nil {
			t.Fatalf("Failed to adjust score: %v", err)
		}

		if adjustedScore <= originalScore {
			t.Errorf("Low reputation should increase spam score, original: %d, adjusted: %d", originalScore, adjustedScore)
		}
	})
}

// TestRuleEngineIntegrationSpam 规则引擎垃圾邮件检测集成测试
func TestRuleEngineIntegrationSpam(t *testing.T) {
	db := setupSpamTestDB(t)
	ctx := context.Background()

	spamRuleRepo := repository.NewSpamRuleRepository(db)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, nil)

	t.Run("关键词匹配规则", func(t *testing.T) {
		rule := &model.SpamRule{
			Name:        "高风险关键词",
			Description: "检测高风险垃圾邮件关键词",
			Category:    "pattern", // 使用 pattern 类型支持正则表达式
			Pattern:     "免费|中奖|优惠",
			Score:       25,
			Enabled:     true,
			IsBuiltin:   false,
		}
		err := spamRuleRepo.Create(ctx, rule)
		if err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}

		email := &model.Email{
			Subject:  "免费领取大奖",
			TextBody: "恭喜您中奖了！",
		}

		result, err := ruleEngine.Check(ctx, email)
		if err != nil {
			t.Fatalf("Failed to check rules: %v", err)
		}

		if result.Score == 0 {
			t.Error("Rule should match and add score")
		}

		if len(result.HitRules) == 0 {
			t.Error("Should have hit rules")
		}
	})
}

// TestPerformanceMonitorIntegration 性能监控集成测试
func TestPerformanceMonitorIntegration(t *testing.T) {
	db := setupSpamTestDB(t)

	logRepo := repository.NewSpamDetectionLogRepository(db)
	monitor := spam.NewPerformanceMonitor(logRepo)

	t.Run("记录检测性能", func(t *testing.T) {
		metrics := &spam.DetectionMetrics{
			EmailID:      "test-001",
			TotalLatency: 50 * time.Millisecond,
			LayerLatencies: map[string]time.Duration{
				"PreFilter":  10 * time.Millisecond,
				"RuleEngine": 30 * time.Millisecond,
				"Reputation": 10 * time.Millisecond,
			},
			IsSpam:    false,
			Score:     25,
			Timestamp: time.Now(),
		}

		monitor.RecordDetection(metrics)

		perfMetrics := monitor.GetMetrics()

		if perfMetrics.TotalDetections != 1 {
			t.Errorf("Total detections should be 1, got: %d", perfMetrics.TotalDetections)
		}

		if perfMetrics.SlowDetections != 0 {
			t.Errorf("Slow detections should be 0, got: %d", perfMetrics.SlowDetections)
		}
	})

	t.Run("慢检测记录", func(t *testing.T) {
		slowMetrics := &spam.DetectionMetrics{
			EmailID:      "test-002",
			TotalLatency: 300 * time.Millisecond,
			LayerLatencies: map[string]time.Duration{
				"PreFilter":  100 * time.Millisecond,
				"RuleEngine": 150 * time.Millisecond,
				"Reputation": 50 * time.Millisecond,
			},
			IsSpam:    true,
			Score:     75,
			Timestamp: time.Now(),
		}

		monitor.RecordDetection(slowMetrics)

		perfMetrics := monitor.GetMetrics()

		if perfMetrics.SlowDetections == 0 {
			t.Error("Should have recorded slow detection")
		}
	})

	t.Run("健康状态检查", func(t *testing.T) {
		health := monitor.GetHealthStatus()

		if health.Status == "" {
			t.Error("Health status should not be empty")
		}

		if health.Timestamp.IsZero() {
			t.Error("Health timestamp should not be zero")
		}
	})

	t.Run("性能摘要", func(t *testing.T) {
		summary := monitor.GetPerformanceSummary()

		if summary.TotalDetections == 0 {
			t.Error("Summary should have total detections")
		}

		if summary.HealthStatus == "" {
			t.Error("Summary should have health status")
		}
	})
}

// TestEndToEndSpamDetection 端到端垃圾邮件检测测试
func TestEndToEndSpamDetection(t *testing.T) {
	db := setupSpamTestDB(t)
	ctx := context.Background()

	emailListRepo := repository.NewEmailListRepository(db)
	reputationRepo := repository.NewSenderReputationRepository(db)
	spamRuleRepo := repository.NewSpamRuleRepository(db)
	logRepo := repository.NewSpamDetectionLogRepository(db)

	whitelistChecker := spam.NewWhitelistChecker(emailListRepo, nil)
	rblChecker := spam.NewRBLChecker(nil)
	behaviorAnalyzer := spam.NewBehaviorAnalyzer(nil)
	preFilter := spam.NewPreFilter(rblChecker, behaviorAnalyzer)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, nil)
	reputationManager := spam.NewReputationManager(reputationRepo, nil)
	performanceMonitor := spam.NewPerformanceMonitor(logRepo)

	detector := spam.NewSpamDetectorWithMonitor(
		whitelistChecker,
		preFilter,
		ruleEngine,
		reputationManager,
		nil,
		logRepo,
		nil,
		nil,
		nil,
		performanceMonitor,
	)

	userUID := "e2e-test-user"

	t.Run("完整检测流程 - 正常邮件", func(t *testing.T) {
		email := &model.Email{
			ID:          100,
			ProviderID:  "e2e-normal-001",
			AccountUID:  userUID,
			Subject:     "Meeting tomorrow at 10am",
			FromAddress: "colleague@company.com",
			ToAddress:   "user@company.com",
			TextBody:    "Hi, let's meet tomorrow.",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		result, err := detector.Detect(ctx, email)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		if result.IsSpam {
			t.Errorf("Normal email should not be spam, score: %d", result.Score)
		}

		// 注意：在测试环境中，DNS 查询可能会很慢，所以放宽性能要求
		// 生产环境中应该在 200ms 内完成
		if result.CheckedTime > 2*time.Second {
			t.Errorf("Detection too slow: %v", result.CheckedTime)
		}
	})

	t.Run("性能监控验证", func(t *testing.T) {
		metrics := detector.GetPerformanceMetrics()

		if metrics.TotalDetections == 0 {
			t.Error("Should have recorded detections")
		}

		health := detector.GetPerformanceHealth()
		if health.Status == "" {
			t.Error("Should have health status")
		}
	})
}
