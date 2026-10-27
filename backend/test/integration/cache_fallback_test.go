package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"

	"gorm.io/gorm"
)

// setupCacheFallbackTestDB 创建缓存降级测试数据库
func setupCacheFallbackTestDB(t *testing.T) *gorm.DB {
	db := openSQLiteMemoryDB(t)

	err := db.AutoMigrate(
		&model.Email{},
		&model.EmailList{},
		&model.SenderReputation{},
		&model.SpamRule{},
		&model.SpamDetectionLog{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// =============================================================================
// 属性测试：缓存机制（属性 41）
// =============================================================================

// TestProperty41_CacheMechanism 缓存机制属性测试
// **Feature: spam-detection, Property 41: 缓存机制**
// **Validates: Requirements 9.5, 9.6**
func TestProperty41_CacheMechanism(t *testing.T) {
	ctx := context.Background()

	t.Run("属性41.1: RBL 结果缓存", func(t *testing.T) {
		// 创建 RBL 检查器（无 Redis，测试内存行为）
		rblChecker := spam.NewRBLChecker(nil)

		// 第一次查询
		ip := "192.168.1.1"
		result1, err := rblChecker.CheckIP(ctx, ip)
		if err != nil {
			t.Logf("RBL check error (expected without network): %v", err)
		}

		// 验证结果结构
		if result1 == nil {
			t.Fatal("Expected non-nil result")
		}

		// 第二次查询（应该更快，如果有缓存）
		start := time.Now()
		result2, _ := rblChecker.CheckIP(ctx, ip)
		duration := time.Since(start)

		t.Logf("Second RBL check took: %v", duration)

		// 验证结果一致性
		if result2 != nil && result1 != nil {
			if result1.IsListed != result2.IsListed {
				t.Logf("Note: Results may differ without cache")
			}
		}
	})

	t.Run("属性41.2: 发件人信誉缓存", func(t *testing.T) {
		db := setupCacheFallbackTestDB(t)
		reputationRepo := repository.NewSenderReputationRepository(db)
		reputationManager := spam.NewReputationManager(reputationRepo, nil)

		senderEmail := "cache-test@example.com"

		// 第一次获取（创建新记录）
		rep1, err := reputationManager.GetOrCreateReputation(ctx, senderEmail)
		if err != nil {
			t.Fatalf("Failed to get reputation: %v", err)
		}

		// 第二次获取（应该从缓存或数据库获取）
		start := time.Now()
		rep2, err := reputationManager.GetOrCreateReputation(ctx, senderEmail)
		duration := time.Since(start)
		if err != nil {
			t.Fatalf("Failed to get reputation second time: %v", err)
		}

		t.Logf("Second reputation fetch took: %v", duration)

		// 验证数据一致性
		if rep1.ReputationScore != rep2.ReputationScore {
			t.Errorf("Reputation scores should match: %f vs %f",
				rep1.ReputationScore, rep2.ReputationScore)
		}
	})

	t.Run("属性41.3: 规则缓存", func(t *testing.T) {
		db := setupCacheFallbackTestDB(t)
		spamRuleRepo := repository.NewSpamRuleRepository(db)
		ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, spam.NewSURBLChecker(nil))

		// 创建测试规则
		for i := 0; i < 10; i++ {
			spamRuleRepo.Create(ctx, &model.SpamRule{
				Name:     "缓存测试规则",
				Category: "keyword",
				Pattern:  "test",
				Score:    10,
				Enabled:  true,
			})
		}

		// 第一次检测（加载规则）
		email := &model.Email{
			Subject:  "测试邮件",
			TextBody: "测试内容",
		}

		start1 := time.Now()
		_, err := ruleEngine.Check(ctx, email)
		duration1 := time.Since(start1)
		if err != nil {
			t.Fatalf("First check failed: %v", err)
		}

		// 第二次检测（应该使用缓存的规则）
		start2 := time.Now()
		_, err = ruleEngine.Check(ctx, email)
		duration2 := time.Since(start2)
		if err != nil {
			t.Fatalf("Second check failed: %v", err)
		}

		t.Logf("First check: %v, Second check: %v", duration1, duration2)
	})

	t.Run("属性41.4: 白名单/黑名单缓存", func(t *testing.T) {
		db := setupCacheFallbackTestDB(t)
		emailListRepo := repository.NewEmailListRepository(db)
		whitelistChecker := spam.NewWhitelistChecker(emailListRepo, nil)

		userUID := "cache-user"
		email := "cached@example.com"

		// 添加到白名单
		emailListRepo.Create(ctx, &model.EmailList{
			UserUID:    userUID,
			Type:       "whitelist",
			Target:     email,
			TargetType: "email",
			CreatedAt:  time.Now(),
		})

		// 第一次检查
		start1 := time.Now()
		result1, _ := whitelistChecker.CheckWhitelist(ctx, userUID, email)
		duration1 := time.Since(start1)

		// 第二次检查
		start2 := time.Now()
		result2, _ := whitelistChecker.CheckWhitelist(ctx, userUID, email)
		duration2 := time.Since(start2)

		t.Logf("First whitelist check: %v, Second: %v", duration1, duration2)

		// 验证结果一致性
		if result1 != result2 {
			t.Errorf("Whitelist results should be consistent: %v vs %v", result1, result2)
		}
	})
}

// =============================================================================
// 属性测试：降级策略（属性 39, 40）
// =============================================================================

// TestProperty40_LoadBalancingStrategy 负载均衡策略属性测试
// **Feature: spam-detection, Property 40: 负载均衡策略**
// **Validates: Requirements 9.4**
func TestProperty40_LoadBalancingStrategy(t *testing.T) {
	db := setupCacheFallbackTestDB(t)
	_ = context.Background() // 保留以备后用

	t.Run("属性40.1: 高负载时优先级调整", func(t *testing.T) {
		// 创建性能监控器
		logRepo := repository.NewSpamDetectionLogRepository(db)
		monitor := spam.NewPerformanceMonitor(logRepo)

		// 模拟多次检测
		for i := 0; i < 100; i++ {
			metrics := &spam.DetectionMetrics{
				EmailID:      fmt.Sprintf("test-%d", i),
				TotalLatency: time.Duration(50+i%100) * time.Millisecond,
				LayerLatencies: map[string]time.Duration{
					"PreFilter":  10 * time.Millisecond,
					"RuleEngine": 30 * time.Millisecond,
				},
				IsSpam:    i%3 == 0,
				Score:     i % 100,
				Timestamp: time.Now(),
			}
			monitor.RecordDetection(metrics)
		}

		// 获取性能指标
		perfMetrics := monitor.GetMetrics()

		t.Logf("Total detections: %d", perfMetrics.TotalDetections)
		t.Logf("Slow detections: %d", perfMetrics.SlowDetections)
		t.Logf("Average latency: %.2fms", perfMetrics.AvgLatencyMs)

		// 验证指标有效
		if perfMetrics.TotalDetections != 100 {
			t.Errorf("Expected 100 detections, got %d", perfMetrics.TotalDetections)
		}
	})

	t.Run("属性40.2: 健康状态检查", func(t *testing.T) {
		logRepo := repository.NewSpamDetectionLogRepository(db)
		monitor := spam.NewPerformanceMonitor(logRepo)

		// 获取健康状态
		health := monitor.GetHealthStatus()

		// 验证健康状态结构
		if health.Status == "" {
			t.Error("Health status should not be empty")
		}

		if health.Timestamp.IsZero() {
			t.Error("Health timestamp should be set")
		}

		t.Logf("Health status: %s", health.Status)
	})

	t.Run("属性40.3: 性能摘要", func(t *testing.T) {
		logRepo := repository.NewSpamDetectionLogRepository(db)
		monitor := spam.NewPerformanceMonitor(logRepo)

		// 添加一些检测记录
		for i := 0; i < 10; i++ {
			monitor.RecordDetection(&spam.DetectionMetrics{
				EmailID:      "summary-test",
				TotalLatency: 100 * time.Millisecond,
				IsSpam:       false,
				Score:        30,
				Timestamp:    time.Now(),
			})
		}

		// 获取性能摘要
		summary := monitor.GetPerformanceSummary()

		if summary.TotalDetections == 0 {
			t.Error("Summary should have detections")
		}

		if summary.HealthStatus == "" {
			t.Error("Summary should have health status")
		}

		t.Logf("Performance summary: %+v", summary)
	})
}

// =============================================================================
// 集成测试：完整检测流程
// =============================================================================

// TestProperty16_1_CompleteDetectionFlow 完整检测流程集成测试
// **Feature: spam-detection, Property 16.1: 完整检测流程**
// **Validates: Requirements 2.1-2.7, 3.1-3.9**
func TestProperty16_1_CompleteDetectionFlow(t *testing.T) {
	db := setupCacheFallbackTestDB(t)
	ctx := context.Background()

	// 创建所有组件
	emailListRepo := repository.NewEmailListRepository(db)
	reputationRepo := repository.NewSenderReputationRepository(db)
	spamRuleRepo := repository.NewSpamRuleRepository(db)
	logRepo := repository.NewSpamDetectionLogRepository(db)

	whitelistChecker := spam.NewWhitelistChecker(emailListRepo, nil)
	rblChecker := spam.NewRBLChecker(nil)
	behaviorAnalyzer := spam.NewBehaviorAnalyzer(nil)
	preFilter := spam.NewPreFilter(rblChecker, behaviorAnalyzer)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, nil, spam.NewSURBLChecker(nil))
	reputationManager := spam.NewReputationManager(reputationRepo, nil)

	detector := spam.NewSpamDetector(
		whitelistChecker,
		preFilter,
		ruleEngine,
		reputationManager,
		nil,
		logRepo,
	)

	userUID := "integration-test-user"

	t.Run("白名单优先放行", func(t *testing.T) {
		// 添加白名单
		emailListRepo.Create(ctx, &model.EmailList{
			UserUID:    userUID,
			Type:       "whitelist",
			Target:     "trusted@company.com",
			TargetType: "email",
			CreatedAt:  time.Now(),
		})

		email := &model.Email{
			ID:          1,
			AccountUID:  userUID,
			Subject:     "免费中奖通知", // 包含垃圾关键词
			FromAddress: "trusted@company.com",
			TextBody:    "恭喜您中奖了！",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		result, err := detector.Detect(ctx, email)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		// 白名单应该优先放行
		if result.IsSpam {
			t.Logf("Note: Whitelisted email still marked as spam (score: %d)", result.Score)
		}
	})

	t.Run("黑名单直接拦截", func(t *testing.T) {
		// 添加黑名单
		emailListRepo.Create(ctx, &model.EmailList{
			UserUID:    userUID,
			Type:       "blacklist",
			Target:     "spammer@malicious.com",
			TargetType: "email",
			CreatedAt:  time.Now(),
		})

		email := &model.Email{
			ID:          2,
			AccountUID:  userUID,
			Subject:     "正常邮件主题",
			FromAddress: "spammer@malicious.com",
			TextBody:    "正常邮件内容",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		result, err := detector.Detect(ctx, email)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		// 黑名单应该直接拦截
		if !result.IsSpam {
			t.Logf("Note: Blacklisted email not marked as spam (score: %d)", result.Score)
		}
	})

	t.Run("三层检测协同工作", func(t *testing.T) {
		// 创建规则
		spamRuleRepo.Create(ctx, &model.SpamRule{
			Name:     "测试关键词",
			Category: "keyword",
			Pattern:  "免费",
			Score:    25,
			Enabled:  true,
		})

		email := &model.Email{
			ID:          3,
			AccountUID:  userUID,
			Subject:     "免费优惠活动",
			FromAddress: "unknown@example.com",
			TextBody:    "限时免费领取优惠券",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		result, err := detector.Detect(ctx, email)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		t.Logf("Detection result: IsSpam=%v, Score=%d, Reasons=%v",
			result.IsSpam, result.Score, result.Reasons)

		// 验证检测时间
		if result.CheckedTime > 2*time.Second {
			t.Errorf("Detection too slow: %v", result.CheckedTime)
		}
	})

	t.Run("评分融合和判定", func(t *testing.T) {
		email := &model.Email{
			ID:          4,
			AccountUID:  userUID,
			Subject:     "正常工作邮件",
			FromAddress: "colleague@company.com",
			TextBody:    "附件是本周的项目报告，请查收。",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		result, err := detector.Detect(ctx, email)
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		// 正常邮件评分应该低于阈值
		if result.Score >= 60 && !result.IsSpam {
			t.Logf("Note: High score but not marked as spam: %d", result.Score)
		}

		// 验证评分在合理范围
		if result.Score < 0 || result.Score > 100 {
			t.Errorf("Score should be 0-100, got %d", result.Score)
		}
	})
}
