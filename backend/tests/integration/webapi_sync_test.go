//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/adapter/webapi/cloudflare"
	"fusionmail/internal/adapter/webapi/cloudmail"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 集成测试配置
// 运行前需要设置环境变量或创建 .webapi-test-config 文件
type testConfig struct {
	CloudflareBaseURL       string
	CloudflareAdminPassword string
	DatabaseURL             string
}

func loadTestConfig() *testConfig {
	return &testConfig{
		CloudflareBaseURL:       getEnvOrDefault("CLOUDFLARE_BASE_URL", ""),
		CloudflareAdminPassword: getEnvOrDefault("CLOUDFLARE_ADMIN_PASSWORD", ""),
		DatabaseURL:             getEnvOrDefault("DATABASE_URL", ""),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupTestDB 初始化测试数据库连接
func setupTestDB(t *testing.T, databaseURL string) *gorm.DB {
	if databaseURL == "" {
		t.Skip("跳过数据库测试：未配置 DATABASE_URL")
	}

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(databaseURL), gormConfig)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	return db
}

// cleanupTestData 清理测试数据
func cleanupTestData(t *testing.T, db *gorm.DB, accountUID string) {
	if accountUID == "" {
		return
	}

	// 删除测试创建的 EmailAccount
	if err := db.Where("uid = ?", accountUID).Delete(&model.EmailAccount{}).Error; err != nil {
		t.Logf("清理 EmailAccount 失败: %v", err)
	}
}

// ============================================
// Cloudflare Admin 模式集成测试
// ============================================

// TestCloudflareAdminMode_Integration 测试 Cloudflare Admin 模式集成
func TestCloudflareAdminMode_Integration(t *testing.T) {
	config := loadTestConfig()

	if config.CloudflareBaseURL == "" || config.CloudflareAdminPassword == "" {
		t.Skip("跳过集成测试：未配置 Cloudflare 测试环境变量")
	}

	t.Log("=== Cloudflare Admin 模式集成测试 ===")
	t.Logf("Base URL: %s", config.CloudflareBaseURL)

	// 创建 Cloudflare 适配器配置
	authData := &model.CloudflareTempEmailAuthData{
		BaseURL:       config.CloudflareBaseURL,
		AccessMode:    model.WebAPIAccessModeAdmin,
		AdminPassword: config.CloudflareAdminPassword,
	}

	// 创建适配器
	adapter, err := cloudflare.NewCloudflareTempEmailAdapter(authData)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试连接
	t.Run("测试连接", func(t *testing.T) {
		err := adapter.TestConnection(ctx)
		if err != nil {
			t.Fatalf("连接测试失败: %v", err)
		}
		t.Log("✓ 连接测试成功")
	})

	// 连接
	t.Run("建立连接", func(t *testing.T) {
		err := adapter.Connect(ctx)
		if err != nil {
			t.Fatalf("连接失败: %v", err)
		}
		if !adapter.IsConnected() {
			t.Fatal("连接后 IsConnected() 应返回 true")
		}
		t.Log("✓ 连接成功")
	})

	// 拉取邮件
	t.Run("拉取邮件列表", func(t *testing.T) {
		emails, err := adapter.FetchEmails(ctx, time.Time{}, 10)
		if err != nil {
			t.Fatalf("拉取邮件失败: %v", err)
		}

		t.Logf("✓ 拉取到 %d 封邮件", len(emails))

		// 打印邮件摘要
		for i, email := range emails {
			if i >= 3 {
				t.Logf("  ... 还有 %d 封邮件", len(emails)-3)
				break
			}
			t.Logf("  [%d] ID=%s, Subject=%s, To=%v",
				i+1, email.ProviderID, truncate(email.Subject, 30), email.ToAddresses)
		}
	})

	// 断开连接
	t.Run("断开连接", func(t *testing.T) {
		err := adapter.Disconnect()
		if err != nil {
			t.Fatalf("断开连接失败: %v", err)
		}
		if adapter.IsConnected() {
			t.Fatal("断开后 IsConnected() 应返回 false")
		}
		t.Log("✓ 断开连接成功")
	})
}

// TestCloudflareAdminMode_FetchWithPagination 测试分页拉取
func TestCloudflareAdminMode_FetchWithPagination(t *testing.T) {
	config := loadTestConfig()

	if config.CloudflareBaseURL == "" || config.CloudflareAdminPassword == "" {
		t.Skip("跳过集成测试：未配置 Cloudflare 测试环境变量")
	}

	t.Log("=== Cloudflare Admin 模式分页测试 ===")

	authData := &model.CloudflareTempEmailAuthData{
		BaseURL:       config.CloudflareBaseURL,
		AccessMode:    model.WebAPIAccessModeAdmin,
		AdminPassword: config.CloudflareAdminPassword,
	}

	adapter, err := cloudflare.NewCloudflareTempEmailAdapter(authData)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := adapter.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Disconnect()

	// 测试不同的 limit 值
	limits := []int{5, 10, 20}
	for _, limit := range limits {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			emails, err := adapter.FetchEmails(ctx, time.Time{}, limit)
			if err != nil {
				t.Fatalf("拉取邮件失败: %v", err)
			}
			t.Logf("✓ limit=%d, 实际拉取=%d", limit, len(emails))
		})
	}
}

// ============================================
// 邮件分发逻辑集成测试
// ============================================

// TestWebAPIEmailDistribution_Integration 测试邮件分发逻辑
func TestWebAPIEmailDistribution_Integration(t *testing.T) {
	config := loadTestConfig()

	if config.CloudflareBaseURL == "" || config.CloudflareAdminPassword == "" {
		t.Skip("跳过集成测试：未配置 Cloudflare 测试环境变量")
	}

	t.Log("=== 邮件分发逻辑集成测试 ===")

	// 创建适配器
	authData := &model.CloudflareTempEmailAuthData{
		BaseURL:       config.CloudflareBaseURL,
		AccessMode:    model.WebAPIAccessModeAdmin,
		AdminPassword: config.CloudflareAdminPassword,
	}

	adapter, err := cloudflare.NewCloudflareTempEmailAdapter(authData)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 连接并拉取邮件
	if err := adapter.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Disconnect()

	emails, err := adapter.FetchEmails(ctx, time.Time{}, 20)
	if err != nil {
		t.Fatalf("拉取邮件失败: %v", err)
	}

	if len(emails) == 0 {
		t.Log("⚠ 没有邮件可供测试分发逻辑")
		return
	}

	// 转换为 WebAPIEmail 并按目标地址分组
	t.Run("按目标地址分组", func(t *testing.T) {
		grouped := make(map[string]int)

		for _, email := range emails {
			targetAddr := webapi.ExtractTargetAddress(email)
			if targetAddr == "" {
				targetAddr = "_unknown_"
			}
			grouped[targetAddr]++
		}

		t.Logf("✓ 邮件按 %d 个目标地址分组:", len(grouped))
		for addr, count := range grouped {
			t.Logf("  %s: %d 封", addr, count)
		}
	})

	// 验证邮件字段完整性
	t.Run("验证邮件字段", func(t *testing.T) {
		for i, email := range emails {
			if i >= 5 {
				break
			}

			// 检查必要字段
			if email.ProviderID == "" {
				t.Errorf("邮件 %d: ProviderID 为空", i)
			}
			if email.Subject == "" {
				t.Logf("邮件 %d: Subject 为空（可能正常）", i)
			}
			if email.FromAddress == "" {
				t.Errorf("邮件 %d: FromAddress 为空", i)
			}
			if len(email.ToAddresses) == 0 {
				t.Errorf("邮件 %d: ToAddresses 为空", i)
			}
		}
		t.Log("✓ 邮件字段验证完成")
	})
}

// ============================================
// 数据库集成测试
// ============================================

// TestWebAPISyncService_DatabaseIntegration 测试完整的同步流程（含数据库）
func TestWebAPISyncService_DatabaseIntegration(t *testing.T) {
	config := loadTestConfig()

	if config.CloudflareBaseURL == "" || config.CloudflareAdminPassword == "" {
		t.Skip("跳过集成测试：未配置 Cloudflare 测试环境变量")
	}

	db := setupTestDB(t, config.DatabaseURL)

	t.Log("=== WebAPI 同步服务数据库集成测试 ===")

	// 初始化日志
	log := logger.NewWithModule("WebAPISyncTest")

	// 初始化仓库
	accountRepo := repository.NewAccountRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)

	// 创建同步服务
	syncService := service.NewWebAPISyncService(accountRepo, emailRepo, syncLogRepo)

	// 创建 Cloudflare 适配器
	authData := &model.CloudflareTempEmailAuthData{
		BaseURL:       config.CloudflareBaseURL,
		AccessMode:    model.WebAPIAccessModeAdmin,
		AdminPassword: config.CloudflareAdminPassword,
	}

	adapter, err := cloudflare.NewCloudflareTempEmailAdapter(authData)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 测试同步
	t.Run("执行同步", func(t *testing.T) {
		// 使用一个测试账户 UID
		testAccountUID := "test-webapi-sync-" + time.Now().Format("20060102150405")

		result, err := syncService.SyncProvider(ctx, adapter, testAccountUID)
		if err != nil {
			t.Fatalf("同步失败: %v", err)
		}

		log.Info("同步完成: total=%d, errors=%d", result.TotalCount, result.ErrorCount)

		t.Logf("✓ 同步完成:")
		t.Logf("  - 总邮件数: %d", result.TotalCount)
		t.Logf("  - 错误数: %d", result.ErrorCount)
		t.Logf("  - 跳过数: %d", result.SkippedCount)
	})
}

// TestWebAPIProviderService_Integration 测试 Provider 服务集成
func TestWebAPIProviderService_Integration(t *testing.T) {
	config := loadTestConfig()

	if config.CloudflareBaseURL == "" || config.CloudflareAdminPassword == "" {
		t.Skip("跳过集成测试：未配置 Cloudflare 测试环境变量")
	}

	db := setupTestDB(t, config.DatabaseURL)

	t.Log("=== WebAPI Provider 服务集成测试 ===")

	// 初始化仓库
	accountRepo := repository.NewAccountRepository(db)
	providerRepo := repository.NewProviderRepository(db)
	adapterRepo := repository.NewAdapterRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)

	// 创建加密服务（使用测试密钥）
	cryptoSvc := createTestCryptoService(t)

	// 创建 Provider 服务
	providerService := service.NewWebAPIProviderService(
		accountRepo,
		providerRepo,
		adapterRepo,
		emailRepo,
		syncLogRepo,
		cryptoSvc,
	)

	var testAccountUID string

	// 测试创建 Provider
	t.Run("创建 WebAPI Provider", func(t *testing.T) {
		authData := &model.CloudflareTempEmailAuthData{
			BaseURL:       config.CloudflareBaseURL,
			AccessMode:    model.WebAPIAccessModeAdmin,
			AdminPassword: config.CloudflareAdminPassword,
		}

		authDataJSON, err := json.Marshal(authData)
		if err != nil {
			t.Fatalf("序列化认证数据失败: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		account, err := providerService.Create(ctx, "Integration Test Provider", model.WebAPIServiceTypeCloudflareTempEmail, string(authDataJSON))
		if err != nil {
			t.Fatalf("创建 Provider 失败: %v", err)
		}

		testAccountUID = account.UID
		t.Logf("✓ 创建 Provider 成功: UID=%s", testAccountUID)
	})

	defer cleanupTestData(t, db, testAccountUID)

	// 测试连接
	t.Run("测试连接", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := providerService.TestConnectionByUID(ctx, testAccountUID)
		if err != nil {
			t.Fatalf("测试连接失败: %v", err)
		}
		if !result.Success {
			t.Fatalf("连接测试未成功: %s", result.Error)
		}
		t.Logf("✓ 测试连接成功: %s", result.Message)
	})

	// 测试获取详情
	t.Run("获取 Provider 详情", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		account, err := providerService.GetByUID(ctx, testAccountUID)
		if err != nil {
			t.Fatalf("获取 Provider 失败: %v", err)
		}

		t.Logf("✓ 获取 Provider 详情成功: Email=%s", account.Email)
	})

	// 测试触发同步
	t.Run("触发同步", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		err := providerService.TriggerSync(ctx, testAccountUID)
		if err != nil {
			t.Fatalf("触发同步失败: %v", err)
		}

		// 等待异步同步完成
		time.Sleep(5 * time.Second)

		// 获取同步状态
		status, err := providerService.GetSyncStatus(ctx, testAccountUID)
		if err != nil {
			t.Logf("获取同步状态失败: %v", err)
		} else {
			t.Logf("✓ 同步已触发: status=%s, emailCount=%d", status.Status, status.EmailCount)
		}
	})

	// 测试删除
	t.Run("删除 Provider", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := providerService.Delete(ctx, testAccountUID)
		if err != nil {
			t.Fatalf("删除 Provider 失败: %v", err)
		}

		// 验证删除
		_, err = providerService.GetByUID(ctx, testAccountUID)
		if err == nil {
			t.Error("Provider 应该已被删除")
		}
		t.Log("✓ 删除 Provider 成功")

		// 清空 UID，避免重复清理
		testAccountUID = ""
	})
}

// ============================================
// Cloud Mail 集成测试
// ============================================

// TestCloudMailAdapter_Integration 测试 Cloud Mail 适配器集成
func TestCloudMailAdapter_Integration(t *testing.T) {
	baseURL := os.Getenv("CLOUDMAIL_BASE_URL")
	jwtToken := os.Getenv("CLOUDMAIL_JWT_TOKEN")

	if baseURL == "" || jwtToken == "" {
		t.Skip("跳过集成测试：未配置 Cloud Mail 测试环境变量 (CLOUDMAIL_BASE_URL, CLOUDMAIL_JWT_TOKEN)")
	}

	t.Log("=== Cloud Mail 适配器集成测试 ===")
	t.Logf("Base URL: %s", baseURL)

	// 创建 Cloud Mail 适配器配置（不指定账户，自动从 API 获取）
	authData := &model.CloudMailAuthData{
		BaseURL:  baseURL,
		JWTToken: jwtToken,
		Accounts: nil, // 自动从 API 获取账户列表
	}

	// 创建适配器
	adapter, err := cloudmail.NewCloudMailAdapter(authData)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试连接
	t.Run("测试连接", func(t *testing.T) {
		err := adapter.TestConnection(ctx)
		if err != nil {
			t.Fatalf("连接测试失败: %v", err)
		}
		t.Log("✓ 连接测试成功")
	})

	// 连接（会自动获取账户列表）
	t.Run("建立连接", func(t *testing.T) {
		err := adapter.Connect(ctx)
		if err != nil {
			t.Fatalf("连接失败: %v", err)
		}
		if !adapter.IsConnected() {
			t.Fatal("连接后 IsConnected() 应返回 true")
		}

		accounts := adapter.GetAccounts()
		t.Logf("✓ 连接成功，获取到 %d 个账户:", len(accounts))
		for i, acc := range accounts {
			t.Logf("  [%d] ID=%d, Email=%s, Name=%s", i+1, acc.AccountID, acc.Email, acc.Name)
		}
	})

	// 拉取邮件
	t.Run("拉取邮件列表", func(t *testing.T) {
		emails, err := adapter.FetchEmails(ctx, time.Time{}, 10)
		if err != nil {
			t.Fatalf("拉取邮件失败: %v", err)
		}

		t.Logf("✓ 拉取到 %d 封邮件", len(emails))

		// 打印邮件摘要
		for i, email := range emails {
			if i >= 5 {
				t.Logf("  ... 还有 %d 封邮件", len(emails)-5)
				break
			}
			t.Logf("  [%d] ID=%s, Subject=%s, From=%s, To=%v",
				i+1, email.ProviderID, truncate(email.Subject, 30), email.FromAddress, email.ToAddresses)
		}
	})

	// 断开连接
	t.Run("断开连接", func(t *testing.T) {
		err := adapter.Disconnect()
		if err != nil {
			t.Fatalf("断开连接失败: %v", err)
		}
		if adapter.IsConnected() {
			t.Fatal("断开后 IsConnected() 应返回 false")
		}
		t.Log("✓ 断开连接成功")
	})
}

// TestCloudMailAdapter_WithSpecificAccount 测试指定账户的邮件拉取
func TestCloudMailAdapter_WithSpecificAccount(t *testing.T) {
	baseURL := os.Getenv("CLOUDMAIL_BASE_URL")
	jwtToken := os.Getenv("CLOUDMAIL_JWT_TOKEN")
	accountID := os.Getenv("CLOUDMAIL_ACCOUNT_ID")
	accountEmail := os.Getenv("CLOUDMAIL_ACCOUNT_EMAIL")

	if baseURL == "" || jwtToken == "" {
		t.Skip("跳过集成测试：未配置 Cloud Mail 测试环境变量")
	}

	if accountID == "" || accountEmail == "" {
		t.Skip("跳过指定账户测试：未配置 CLOUDMAIL_ACCOUNT_ID 和 CLOUDMAIL_ACCOUNT_EMAIL")
	}

	t.Log("=== Cloud Mail 指定账户测试 ===")

	// 解析账户 ID
	accID := 0
	fmt.Sscanf(accountID, "%d", &accID)

	// 创建配置（指定账户）
	authData := &model.CloudMailAuthData{
		BaseURL:  baseURL,
		JWTToken: jwtToken,
		Accounts: []model.CloudMailAccount{
			{
				AccountID: accID,
				Email:     accountEmail,
			},
		},
	}

	adapter, err := cloudmail.NewCloudMailAdapter(authData)
	if err != nil {
		t.Fatalf("创建适配器失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := adapter.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Disconnect()

	// 拉取邮件
	emails, err := adapter.FetchEmails(ctx, time.Time{}, 20)
	if err != nil {
		t.Fatalf("拉取邮件失败: %v", err)
	}

	t.Logf("✓ 账户 %s 拉取到 %d 封邮件", accountEmail, len(emails))

	// 验证邮件目标地址
	for i, email := range emails {
		if i >= 3 {
			break
		}
		if len(email.ToAddresses) > 0 && email.ToAddresses[0] != accountEmail {
			t.Logf("⚠ 邮件 %d 的目标地址不匹配: expected=%s, got=%s",
				i, accountEmail, email.ToAddresses[0])
		}
	}
}

// ============================================
// 工具函数
// ============================================

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// createTestCryptoService 创建测试用加密服务
func createTestCryptoService(t *testing.T) *crypto.Service {
	// 使用测试密钥（32 字节）
	testKey := "test-encryption-key-32-bytes!!"
	svc, err := crypto.NewService(testKey)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}
	return svc
}
