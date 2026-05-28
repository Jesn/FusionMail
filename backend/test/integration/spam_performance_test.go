package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"

	"gorm.io/gorm"
)

// setupPerformanceTestDB 创建性能测试数据库
func setupPerformanceTestDB(t *testing.T) *gorm.DB {
	db := openSQLiteMemoryDB(t)

	// 自动迁移
	err := db.AutoMigrate(
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

// createTestDetector 创建测试用的垃圾邮件检测器
func createTestDetector(db *gorm.DB) *spam.SpamDetector {
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

	return spam.NewSpamDetectorWithMonitor(
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
}

// TestDetectionLatency 测试检测延迟
func TestDetectionLatency(t *testing.T) {
	db := setupPerformanceTestDB(t)
	detector := createTestDetector(db)
	ctx := context.Background()

	// 创建测试邮件
	emails := []*model.Email{
		{
			ID:          1,
			Subject:     "Normal email subject",
			FromAddress: "sender1@example.com",
			TextBody:    "This is a normal email body.",
		},
		{
			ID:          2,
			Subject:     "Another normal email",
			FromAddress: "sender2@example.com",
			TextBody:    "Just checking in with you.",
		},
		{
			ID:          3,
			Subject:     "Meeting reminder",
			FromAddress: "sender3@example.com",
			TextBody:    "Don't forget about our meeting tomorrow.",
		},
	}

	t.Run("单次检测延迟", func(t *testing.T) {
		var totalLatency time.Duration
		iterations := 10

		for i := 0; i < iterations; i++ {
			email := emails[i%len(emails)]
			email.ID = int64(i + 100)

			start := time.Now()
			_, err := detector.Detect(ctx, email)
			latency := time.Since(start)

			if err != nil {
				t.Logf("Detection error (iteration %d): %v", i, err)
			}

			totalLatency += latency
		}

		avgLatency := totalLatency / time.Duration(iterations)
		t.Logf("平均检测延迟: %v", avgLatency)

		// 注意：在测试环境中，由于没有 Redis 缓存，延迟可能会更高
		// 生产环境中应该在 200ms 内完成
		if avgLatency > 500*time.Millisecond {
			t.Errorf("平均检测延迟过高: %v (目标 < 500ms)", avgLatency)
		}
	})

	t.Run("检测延迟分布", func(t *testing.T) {
		var latencies []time.Duration
		iterations := 20

		for i := 0; i < iterations; i++ {
			email := emails[i%len(emails)]
			email.ID = int64(i + 200)

			start := time.Now()
			detector.Detect(ctx, email)
			latencies = append(latencies, time.Since(start))
		}

		// 计算统计信息
		var total time.Duration
		var min, max time.Duration = latencies[0], latencies[0]

		for _, l := range latencies {
			total += l
			if l < min {
				min = l
			}
			if l > max {
				max = l
			}
		}

		avg := total / time.Duration(len(latencies))

		t.Logf("延迟统计:")
		t.Logf("  最小: %v", min)
		t.Logf("  最大: %v", max)
		t.Logf("  平均: %v", avg)
	})
}

// TestConcurrentDetection 测试并发检测性能
func TestConcurrentDetection(t *testing.T) {
	db := setupPerformanceTestDB(t)
	detector := createTestDetector(db)
	ctx := context.Background()

	t.Run("并发检测", func(t *testing.T) {
		concurrency := 10
		emailsPerGoroutine := 5

		var wg sync.WaitGroup
		results := make(chan time.Duration, concurrency*emailsPerGoroutine)
		errors := make(chan error, concurrency*emailsPerGoroutine)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < emailsPerGoroutine; j++ {
					email := &model.Email{
						ID:          int64(goroutineID*100 + j),
						Subject:     fmt.Sprintf("Test email %d-%d", goroutineID, j),
						FromAddress: fmt.Sprintf("sender%d@example.com", goroutineID),
						TextBody:    "This is a test email for concurrent detection.",
					}

					detectStart := time.Now()
					_, err := detector.Detect(ctx, email)
					detectLatency := time.Since(detectStart)

					if err != nil {
						errors <- err
					}
					results <- detectLatency
				}
			}(i)
		}

		wg.Wait()
		close(results)
		close(errors)

		totalTime := time.Since(start)
		totalEmails := concurrency * emailsPerGoroutine

		// 收集结果
		var totalLatency time.Duration
		var count int
		for latency := range results {
			totalLatency += latency
			count++
		}

		// 收集错误
		var errorCount int
		for range errors {
			errorCount++
		}

		avgLatency := totalLatency / time.Duration(count)
		throughput := float64(totalEmails) / totalTime.Seconds()

		t.Logf("并发检测结果:")
		t.Logf("  并发数: %d", concurrency)
		t.Logf("  总邮件数: %d", totalEmails)
		t.Logf("  总耗时: %v", totalTime)
		t.Logf("  平均延迟: %v", avgLatency)
		t.Logf("  吞吐量: %.2f 封/秒", throughput)
		t.Logf("  错误数: %d", errorCount)

		// 验证吞吐量
		if throughput < 10 {
			t.Errorf("吞吐量过低: %.2f 封/秒 (目标 > 10 封/秒)", throughput)
		}
	})
}

// TestPerformanceMonitorAccuracy 测试性能监控准确性
func TestPerformanceMonitorAccuracy(t *testing.T) {
	db := setupPerformanceTestDB(t)
	detector := createTestDetector(db)
	ctx := context.Background()

	t.Run("性能指标准确性", func(t *testing.T) {
		// 执行一些检测
		for i := 0; i < 10; i++ {
			email := &model.Email{
				ID:          int64(i + 1000),
				Subject:     fmt.Sprintf("Test email %d", i),
				FromAddress: "test@example.com",
				TextBody:    "Test content",
			}
			detector.Detect(ctx, email)
		}

		// 获取性能指标
		metrics := detector.GetPerformanceMetrics()

		t.Logf("性能指标:")
		t.Logf("  总检测数: %d", metrics.TotalDetections)
		t.Logf("  慢检测数: %d", metrics.SlowDetections)
		t.Logf("  慢检测率: %.2f%%", metrics.SlowRate)
		t.Logf("  平均延迟: %.2fms", metrics.AvgLatencyMs)
		t.Logf("  最大延迟: %.2fms", metrics.MaxLatencyMs)
		t.Logf("  垃圾邮件数: %d", metrics.SpamCount)
		t.Logf("  正常邮件数: %d", metrics.HamCount)

		// 验证指标
		if metrics.TotalDetections != 10 {
			t.Errorf("总检测数不正确: %d (期望 10)", metrics.TotalDetections)
		}
	})

	t.Run("健康状态检查", func(t *testing.T) {
		health := detector.GetPerformanceHealth()

		t.Logf("健康状态: %s", health.Status)
		t.Logf("详情: %v", health.Details)

		if health.Status == "" {
			t.Error("健康状态不应为空")
		}
	})
}

// TestCacheHitRate 测试缓存命中率（模拟）
func TestCacheHitRate(t *testing.T) {
	db := setupPerformanceTestDB(t)
	detector := createTestDetector(db)
	ctx := context.Background()

	t.Run("重复发件人检测", func(t *testing.T) {
		// 使用相同的发件人发送多封邮件
		sender := "repeated@example.com"

		var firstLatency, subsequentLatency time.Duration

		for i := 0; i < 5; i++ {
			email := &model.Email{
				ID:          int64(i + 2000),
				Subject:     fmt.Sprintf("Email %d from same sender", i),
				FromAddress: sender,
				TextBody:    "Test content",
			}

			start := time.Now()
			detector.Detect(ctx, email)
			latency := time.Since(start)

			if i == 0 {
				firstLatency = latency
			} else {
				subsequentLatency += latency
			}
		}

		avgSubsequent := subsequentLatency / 4

		t.Logf("首次检测延迟: %v", firstLatency)
		t.Logf("后续平均延迟: %v", avgSubsequent)

		// 注意：在没有 Redis 的测试环境中，缓存不会生效
		// 这里只是验证检测可以正常工作
	})
}

// BenchmarkDetection 基准测试
func BenchmarkDetection(b *testing.B) {
	db := openSQLiteMemoryDB(b)

	db.AutoMigrate(
		&model.Email{},
		&model.EmailList{},
		&model.SenderReputation{},
		&model.SpamRule{},
		&model.BayesianTraining{},
		&model.SpamDetectionLog{},
	)

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

	detector := spam.NewSpamDetector(
		whitelistChecker,
		preFilter,
		ruleEngine,
		reputationManager,
		nil,
		logRepo,
	)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		email := &model.Email{
			ID:          int64(i),
			Subject:     "Benchmark test email",
			FromAddress: "benchmark@example.com",
			TextBody:    "This is a benchmark test email.",
		}
		detector.Detect(ctx, email)
	}
}

// BenchmarkConcurrentDetection 并发基准测试
func BenchmarkConcurrentDetection(b *testing.B) {
	db := openSQLiteMemoryDB(b)

	db.AutoMigrate(
		&model.Email{},
		&model.EmailList{},
		&model.SenderReputation{},
		&model.SpamRule{},
		&model.BayesianTraining{},
		&model.SpamDetectionLog{},
	)

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

	detector := spam.NewSpamDetector(
		whitelistChecker,
		preFilter,
		ruleEngine,
		reputationManager,
		nil,
		logRepo,
	)

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			email := &model.Email{
				ID:          int64(i),
				Subject:     "Concurrent benchmark test email",
				FromAddress: "concurrent@example.com",
				TextBody:    "This is a concurrent benchmark test email.",
			}
			detector.Detect(ctx, email)
			i++
		}
	})
}
