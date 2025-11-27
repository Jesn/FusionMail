package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"
	"fusionmail/internal/sse"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/oauth2config"
)

// SyncService 邮件同步服务接口
type SyncService interface {
	// SyncAccount 同步指定账户的邮件
	SyncAccount(ctx context.Context, accountUID string) error

	// SyncAllAccounts 同步所有启用的账户
	SyncAllAccounts(ctx context.Context) error

	// StartScheduler 启动定时同步调度器
	StartScheduler(ctx context.Context) error

	// StopScheduler 停止定时同步调度器
	StopScheduler() error
}

// syncService 邮件同步服务实现
type syncService struct {
	accountRepo          repository.AccountRepository
	emailRepo            repository.EmailRepository
	syncLogRepo          repository.SyncLogRepository
	adapterFactory       *adapter.Factory
	cryptoService        *crypto.Service
	schedulerStop        chan struct{}
	oauth2ConfigProvider *oauth2config.Provider // 新增：OAuth2配置提供者
	spamDetector         SpamDetectorInterface  // 垃圾邮件检测器
}

// SpamDetectorInterface 垃圾邮件检测器接口
type SpamDetectorInterface interface {
	DetectSpamSimple(ctx context.Context, email *model.Email) (*spam.SpamSimpleResult, error)
}

// NewSyncService 创建邮件同步服务实例
func NewSyncService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	adapterFactory *adapter.Factory,
	oauth2ClientRepo repository.OAuth2ClientRepository,
	providerRepo repository.ProviderRepository,
	logger *logger.Logger,
	cryptoService *crypto.Service, // 添加加密服务参数（指针类型）
	spamDetector SpamDetectorInterface, // 垃圾邮件检测器（可选）
) SyncService {

	// 创建OAuth2配置提供者
	oauth2Provider := oauth2config.NewProvider(oauth2ClientRepo, providerRepo, cryptoService, logger)

	return &syncService{
		accountRepo:          accountRepo,
		emailRepo:            emailRepo,
		syncLogRepo:          syncLogRepo,
		adapterFactory:       adapterFactory,
		cryptoService:        cryptoService,
		oauth2ConfigProvider: oauth2Provider,
		spamDetector:         spamDetector,
	}
}

// SyncAccount 同步指定账户的邮件
func (s *syncService) SyncAccount(ctx context.Context, accountUID string) error {
	log.Printf("[DEBUG] SyncAccount called for UID: %s", accountUID)

	// 获取账户信息
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountUID)
	}

	// 检查账户状态
	if account.Status != "active" {
		return fmt.Errorf("account is not active (status: %s): %s", account.Status, accountUID)
	}

	// 检查是否启用同步
	if !account.SyncEnabled {
		return fmt.Errorf("sync is disabled for account: %s", accountUID)
	}

	// 创建同步日志
	syncLog := &model.SyncLog{
		AccountUID: accountUID,
		SyncType:   "manual",
		Status:     "running",
		StartedAt:  time.Now(),
	}

	if err := s.syncLogRepo.Create(ctx, syncLog); err != nil {
		log.Printf("Failed to create sync log: %v", err)
	}

	// 执行同步
	err = s.doSync(ctx, account, syncLog)

	// 更新同步日志
	if err != nil {
		syncLog.Status = "failed"
		syncLog.ErrorMessage = err.Error()
		// Sync failed
	} else {
		syncLog.Status = "success"
		// Sync completed successfully
	}

	completedAt := time.Now()
	syncLog.CompletedAt = &completedAt
	syncLog.DurationMs = time.Since(syncLog.StartedAt).Milliseconds()

	// 保存同步日志到数据库
	if updateErr := s.syncLogRepo.Update(ctx, syncLog); updateErr != nil {
		log.Printf("Failed to update sync log: %v", updateErr)
	}

	// 更新账户同步状态（只更新同步相关字段，避免覆盖其他字段如 consecutive_auth_failures）
	if updateErr := s.accountRepo.UpdateSyncStatus(ctx, accountUID, syncLog.Status, syncLog.ErrorMessage); updateErr != nil {
		log.Printf("Failed to update account sync status: %v", updateErr)
	}

	return err
}

// doSync 执行实际的同步逻辑
func (s *syncService) doSync(ctx context.Context, account *model.EmailAccount, syncLog *model.SyncLog) error {
	log.Printf("[DEBUG] Starting sync for account %s (email: %s, auth_type: %s)", account.UID, account.Email, account.AuthType)

	// 解析认证凭证
	credentials, err := s.parseCredentials(account)
	if err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	// 解析代理配置
	proxy, err := s.parseProxyConfig(account)
	if err != nil {
		return fmt.Errorf("failed to parse proxy config: %w", err)
	}

	// 创建适配器配置
	config := &adapter.Config{
		Provider:    account.Provider,
		Protocol:    account.Protocol,
		Credentials: credentials,
		Proxy:       proxy,
		Timeout:     0, // 使用默认超时
	}

	// 使用自动选择方法创建适配器（会智能判断是否使用短效适配器）
	provider, err := s.adapterFactory.CreateProviderAuto(config)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// 连接到邮箱服务器
	if err := provider.Connect(ctx); err != nil {
		// 处理连接错误（包括认证错误的特殊处理）
		return s.handleSyncError(ctx, account, fmt.Errorf("failed to connect: %w", err))
	}
	defer provider.Disconnect()

	// 确定同步起始时间（增量同步）
	since := time.Time{}
	if account.LastSyncAt != nil {
		// 增量同步：从上次同步时间开始（减去 5 分钟缓冲，避免遗漏）
		since = account.LastSyncAt.Add(-5 * time.Minute)
		// Incremental sync started
	} else {
		// 首次同步：从 7 天前开始（避免获取太多历史邮件）
		since = time.Now().AddDate(0, 0, -7)
		// Initial sync started
	}

	// 拉取邮件列表
	emails, err := provider.FetchEmails(ctx, since, 1000) // 限制每次最多 1000 封
	if err != nil {
		// 处理同步错误（包括认证错误的特殊处理）
		return s.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails: %w", err))
	}

	syncLog.EmailsFetched = int64(len(emails))

	// 处理邮件
	for _, email := range emails {
		if err := s.processEmail(ctx, account.UID, email, syncLog); err != nil {
			// Failed to process email
			continue
		}
	}

	// 如果本次同步有新增或更新的邮件，通过 SSE 通知前端刷新统计/列表缓存
	if syncLog.EmailsNew > 0 || syncLog.EmailsUpdated > 0 {
		sse.Broadcast("email_counts_maybe_changed", "{}")
	}

	// 同步成功，重置失败计数（仅对 quick 账号）
	if account.AuthType == "quick" && account.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, account.UID); resetErr != nil {
			log.Printf("[ERROR] Failed to reset failure counter for account %s: %v", account.UID, resetErr)
		} else {
			log.Printf("[DEBUG] Reset failure counter for quick account %s after successful sync", account.UID)
		}
	}

	return nil
}

// processEmail 处理单封邮件
func (s *syncService) processEmail(ctx context.Context, accountUID string, adapterEmail *adapter.Email, syncLog *model.SyncLog) error {
	// 检查邮件是否已存在
	existingEmail, err := s.emailRepo.FindByProviderID(ctx, adapterEmail.ProviderID, accountUID)
	if err != nil {
		return err
	}

	if existingEmail != nil {
		// 邮件已存在，更新
		s.updateEmailFromAdapter(existingEmail, adapterEmail, accountUID)
		if err := s.emailRepo.Update(ctx, existingEmail); err != nil {
			return err
		}
		// 应用规则到已存在邮件（更新后）
		if err := s.applyRulesForEmail(ctx, existingEmail); err != nil {
			log.Printf("[WARN] Failed to apply rules to existing email %d: %v", existingEmail.ID, err)
		}
		syncLog.EmailsUpdated++
	} else {
		// 新邮件，创建
		newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)

		// 垃圾邮件检测（仅对新邮件）
		if s.spamDetector != nil {
			spamResult, spamErr := s.spamDetector.DetectSpamSimple(ctx, newEmail)
			if spamErr != nil {
				log.Printf("[WARN] Spam detection failed for email %s: %v", newEmail.MessageID, spamErr)
			} else if spamResult != nil {
				newEmail.IsSpam = spamResult.IsSpam
				newEmail.SpamScore = float64(spamResult.Score)
				newEmail.SpamConfidence = spamResult.Confidence
				newEmail.SpamReason = spamResult.Reason
				newEmail.SpamDetectedBy = spamResult.DetectedBy
				if spamResult.IsSpam {
					now := time.Now()
					newEmail.SpamDetectedAt = &now
					log.Printf("[INFO] 检测到垃圾邮件: %s (评分: %d, 置信度: %.2f, 原因: %s)",
						newEmail.Subject, spamResult.Score, spamResult.Confidence, spamResult.Reason)
				}
			}
		}

		if err := s.emailRepo.Create(ctx, newEmail); err != nil {
			return err
		}
		// 应用规则到新邮件
		if err := s.applyRulesForEmail(ctx, newEmail); err != nil {
			log.Printf("[WARN] Failed to apply rules to new email %d: %v", newEmail.ID, err)
		}
		syncLog.EmailsNew++
	}

	return nil
}

// createEmailFromAdapter 从适配器邮件创建数据库邮件模型
func (s *syncService) createEmailFromAdapter(adapterEmail *adapter.Email, accountUID string) *model.Email {
	return &model.Email{
		ProviderID:       adapterEmail.ProviderID,
		AccountUID:       accountUID,
		MessageID:        adapterEmail.MessageID,
		Subject:          adapterEmail.Subject,
		FromAddress:      adapterEmail.FromAddress,
		FromName:         adapterEmail.FromName,
		ToAddresses:      s.joinAddresses(adapterEmail.ToAddresses),
		CcAddresses:      s.joinAddresses(adapterEmail.CcAddresses),
		BccAddresses:     s.joinAddresses(adapterEmail.BccAddresses),
		ReplyTo:          adapterEmail.ReplyTo,
		TextBody:         adapterEmail.TextBody,
		HTMLBody:         adapterEmail.HTMLBody,
		Snippet:          adapterEmail.Snippet,
		SourceIsRead:     adapterEmail.SourceIsRead,
		SourceLabels:     s.joinLabels(adapterEmail.SourceLabels),
		SourceFolder:     adapterEmail.SourceFolder,
		HasAttachments:   adapterEmail.HasAttachments,
		AttachmentsCount: adapterEmail.AttachmentsCount,
		SentAt:           adapterEmail.SentAt,
		ReceivedAt:       adapterEmail.ReceivedAt,
		SizeBytes:        adapterEmail.SizeBytes,
		ThreadID:         adapterEmail.ThreadID,
		InReplyTo:        adapterEmail.InReplyTo,
		References:       adapterEmail.References,
		SyncedAt:         time.Now(),
	}
}

// updateEmailFromAdapter 从适配器邮件更新数据库邮件模型
func (s *syncService) updateEmailFromAdapter(dbEmail *model.Email, adapterEmail *adapter.Email, accountUID string) {
	// 更新可能变化的字段
	dbEmail.Subject = adapterEmail.Subject
	dbEmail.TextBody = adapterEmail.TextBody
	dbEmail.HTMLBody = adapterEmail.HTMLBody
	dbEmail.Snippet = adapterEmail.Snippet
	dbEmail.SourceIsRead = adapterEmail.SourceIsRead
	dbEmail.SourceLabels = s.joinLabels(adapterEmail.SourceLabels)
	dbEmail.SourceFolder = adapterEmail.SourceFolder
	dbEmail.HasAttachments = adapterEmail.HasAttachments
	dbEmail.AttachmentsCount = adapterEmail.AttachmentsCount
	dbEmail.SizeBytes = adapterEmail.SizeBytes
	dbEmail.SyncedAt = time.Now()
}

// SyncAllAccounts 同步所有启用的账户（立即同步，不考虑同步间隔）
// 主要用于手动触发全量同步
func (s *syncService) SyncAllAccounts(ctx context.Context) error {
	// 获取所有启用同步的账户
	accounts, err := s.accountRepo.ListSyncEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sync enabled accounts: %w", err)
	}

	log.Printf("Starting manual sync for %d accounts", len(accounts))

	// 并发同步账户
	for _, account := range accounts {
		go func(accountUID string) {
			if err := s.SyncAccount(ctx, accountUID); err != nil {
				log.Printf("Manual sync failed for account %s: %v", accountUID, err)
			}
		}(account.UID)
	}

	return nil
}

// StartScheduler 启动定时同步调度器
// 使用统一调度器 + 时间判断方案，支持每个账户的个性化同步间隔
func (s *syncService) StartScheduler(ctx context.Context) error {
	s.schedulerStop = make(chan struct{})

	go func() {
		// 每分钟检查一次，判断哪些账户需要同步
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		log.Println("[Scheduler] Sync scheduler started (checking every 1 minute)")

		for {
			select {
			case <-ticker.C:
				s.checkAndSyncAccounts(ctx)
			case <-s.schedulerStop:
				log.Println("[Scheduler] Sync scheduler stopped")
				return
			case <-ctx.Done():
				log.Println("[Scheduler] Sync scheduler cancelled")
				return
			}
		}
	}()

	return nil
}

// checkAndSyncAccounts 检查并同步需要同步的账户
// 根据每个账户的 sync_interval 和 last_sync_at 判断是否需要同步
func (s *syncService) checkAndSyncAccounts(ctx context.Context) {
	// 获取所有启用同步的账户
	accounts, err := s.accountRepo.ListSyncEnabled(ctx)
	if err != nil {
		log.Printf("[Scheduler] Failed to list sync enabled accounts: %v", err)
		return
	}

	if len(accounts) == 0 {
		return
	}

	now := time.Now()
	syncCount := 0

	// 检查每个账户是否需要同步
	for _, account := range accounts {
		if s.shouldSync(account, now) {
			syncCount++
			log.Printf("[Scheduler] Triggering sync for account %s (email: %s, interval: %d min)",
				account.UID, account.Email, account.SyncInterval)

			// 异步同步账户
			go func(acc *model.EmailAccount) {
				if err := s.SyncAccount(ctx, acc.UID); err != nil {
					log.Printf("[Scheduler] Sync failed for account %s: %v", acc.UID, err)
				}
			}(account)
		}
	}

	if syncCount > 0 {
		log.Printf("[Scheduler] Triggered sync for %d/%d accounts", syncCount, len(accounts))
	}
}

// shouldSync 判断账户是否需要同步
// 根据账户的 last_sync_at 和 sync_interval 计算是否到达下次同步时间
func (s *syncService) shouldSync(account *model.EmailAccount, now time.Time) bool {
	// 首次同步（从未同步过）
	if account.LastSyncAt == nil {
		log.Printf("[Scheduler] Account %s needs first sync", account.UID)
		return true
	}

	// 计算下次同步时间
	syncInterval := time.Duration(account.SyncInterval) * time.Minute
	nextSyncTime := account.LastSyncAt.Add(syncInterval)

	// 判断是否到达或超过下次同步时间
	shouldSync := now.After(nextSyncTime) || now.Equal(nextSyncTime)

	if shouldSync {
		timeSinceLastSync := now.Sub(*account.LastSyncAt)
		log.Printf("[Scheduler] Account %s ready for sync (last: %s ago, interval: %d min)",
			account.UID,
			timeSinceLastSync.Round(time.Minute),
			account.SyncInterval)
	}

	return shouldSync
}

// StopScheduler 停止定时同步调度器
func (s *syncService) StopScheduler() error {
	if s.schedulerStop != nil {
		close(s.schedulerStop)
		s.schedulerStop = nil
		log.Println("Sync scheduler stopped")
	}
	return nil
}

// 辅助方法

// parseCredentials 解析认证凭证
func (s *syncService) parseCredentials(account *model.EmailAccount) (*adapter.Credentials, error) {
	// 解密凭证数据
	decryptedData, err := s.cryptoService.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 初始化凭证结构
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: account.AuthType,
	}

	// 根据认证类型处理凭证
	if account.AuthType == "oauth2" {
		// OAuth2 凭证是 JSON 格式
		var oauthCreds struct {
			Email        string    `json:"email"`
			AuthType     string    `json:"auth_type"`
			AccessToken  string    `json:"access_token"`
			RefreshToken string    `json:"refresh_token"`
			TokenExpiry  time.Time `json:"token_expiry"`
		}

		if err := json.Unmarshal(decryptedData, &oauthCreds); err != nil {
			return nil, fmt.Errorf("failed to parse OAuth2 credentials: %w", err)
		}

		credentials.AccessToken = oauthCreds.AccessToken
		credentials.RefreshToken = oauthCreds.RefreshToken
		credentials.TokenExpiry = oauthCreds.TokenExpiry

		// 为 OAuth2 提供商设置 ClientID 和 ClientSecret
		// 这些凭证用于刷新 access_token
		if account.Provider == "gmail" && account.Protocol == "gmail_api" {
			// Gmail API OAuth2 配置 - 从数据库获取（使用provider_type）
			oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2Config(context.Background(), int(model.ProviderTypeGmail))
			if err != nil {
				return nil, fmt.Errorf("failed to get Gmail OAuth2 config from database: %w", err)
			}
			credentials.ClientID = oauth2Config.ClientID
			credentials.ClientSecret = oauth2Config.ClientSecret
		} else if account.Provider == "outlook" && account.Protocol == "graph" {
			// Microsoft Graph API OAuth2 配置 - 从数据库获取（使用provider_type）
			oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2Config(context.Background(), int(model.ProviderTypeOutlook))
			if err != nil {
				return nil, fmt.Errorf("failed to get Outlook OAuth2 config from database: %w", err)
			}
			credentials.ClientID = oauth2Config.ClientID
			credentials.ClientSecret = oauth2Config.ClientSecret
		}
	} else if account.AuthType == "quick" {
		// 短效认证凭证是 JSON 格式
		var quickCreds struct {
			Email        string `json:"email"`
			AuthType     string `json:"auth_type"`
			RefreshToken string `json:"refresh_token"`
			ClientID     string `json:"client_id"`
		}

		if err := json.Unmarshal(decryptedData, &quickCreds); err != nil {
			return nil, fmt.Errorf("failed to parse quick credentials: %w", err)
		}

		credentials.RefreshToken = quickCreds.RefreshToken
		credentials.ClientID = quickCreds.ClientID
		// 短效适配器不需要 ClientSecret
	} else {
		// 密码认证，直接使用解密后的数据作为密码
		credentials.Password = string(decryptedData)
	}

	// 设置 IMAP 服务器配置
	// 如果用户手动配置了服务器地址，优先使用用户配置
	if account.IMAPHost != "" && account.IMAPPort != 0 {
		credentials.Host = account.IMAPHost
		credentials.Port = account.IMAPPort
		credentials.TLS = true // 默认开启 TLS，后续可根据 Encryption 字段调整
	} else {
		// 使用预设的服务器配置
		switch account.Provider {
		case "icloud":
			credentials.Host = "imap.mail.me.com"
			credentials.Port = 993
			credentials.TLS = true
		case "qq":
			credentials.Host = "imap.qq.com"
			credentials.Port = 993
			credentials.TLS = true
		case "163":
			credentials.Host = "imap.163.com"
			credentials.Port = 993
			credentials.TLS = true
		case "gmail":
			credentials.Host = "imap.gmail.com"
			credentials.Port = 993
			credentials.TLS = true
		case "outlook":
			credentials.Host = "outlook.office365.com"
			credentials.Port = 993
			credentials.TLS = true
		case "generic":
			// generic 必须配置服务器信息，如果上面没有配置（即 IMAPHost 为空），这里会报错
		default:
			return nil, fmt.Errorf("unsupported provider: %s", account.Provider)
		}
	}

	// 对于 generic 或手动配置的情况，进行额外检查和设置
	if account.Provider == "generic" || (account.IMAPHost != "" && account.IMAPPort != 0) {
		if account.Protocol == "imap" {
			// 已经在上面设置了，这里再次确认（如果是 generic 且没有手动配置，会在下面报错）
			if credentials.Host == "" {
				credentials.Host = account.IMAPHost
				credentials.Port = account.IMAPPort
			}
		} else if account.Protocol == "pop3" {
			credentials.Host = account.POP3Host
			credentials.Port = account.POP3Port
		}

		// 智能修复常见的配置错误
		if credentials.Host == "mail.linuxdo.org" {
			// Auto-fixing incorrect host configuration
			credentials.Host = "mail.linux.do"
		}

		// 设置加密方式
		switch account.Encryption {
		case "ssl":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true // 默认使用 SSL
		}

		// 验证必要的配置
		if credentials.Host == "" || credentials.Port == 0 {
			return nil, fmt.Errorf("provider requires host and port configuration")
		}
	}

	return credentials, nil
}

// parseProxyConfig 解析代理配置
func (s *syncService) parseProxyConfig(account *model.EmailAccount) (*adapter.ProxyConfig, error) {
	if !account.ProxyEnabled {
		return nil, nil
	}

	return &adapter.ProxyConfig{
		Enabled:  account.ProxyEnabled,
		Type:     account.ProxyType,
		Host:     account.ProxyHost,
		Port:     account.ProxyPort,
		Username: account.ProxyUsername,
		// Password: decrypt(account.EncryptedProxyPassword),
	}, nil
}

// joinAddresses 将地址列表转换为 JSON 字符串
func (s *syncService) joinAddresses(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	data, _ := json.Marshal(addresses)
	return string(data)
}

// joinLabels 将标签列表转换为 JSON 字符串
func (s *syncService) joinLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	data, _ := json.Marshal(labels)
	return string(data)
}

// isAuthError 判断错误是否为认证错误
// 认证错误包括：HTTP 401、token 过期、invalid_grant 等
func (s *syncService) isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// 检查 HTTP 401 状态码
	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
		return true
	}

	// 检查 token 过期相关错误
	authErrorPatterns := []string{
		"token expired",
		"token has been expired",
		"invalid_grant",
		"authentication failed",
		"authenticate failed",
		"invalid credentials",
		"access denied",
		"auth",
	}

	for _, pattern := range authErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// handleSyncError 处理同步错误
// 对于 quick 类型账号的认证错误，进行失败计数和自动禁用处理
func (s *syncService) handleSyncError(ctx context.Context, account *model.EmailAccount, err error) error {
	// 仅对 quick 类型账号进行特殊处理
	if account.AuthType != "quick" {
		return err
	}

	// 判断是否为认证错误
	if !s.isAuthError(err) {
		log.Printf("[DEBUG] Quick account %s sync failed with non-auth error: %v", account.UID, err)
		return err
	}

	// 增加失败计数
	failureCount, incErr := s.accountRepo.IncrementConsecutiveFailures(ctx, account.UID)
	if incErr != nil {
		log.Printf("[ERROR] Failed to increment failure counter for account %s: %v", account.UID, incErr)
		return err
	}

	log.Printf("[WARN] Quick account %s (email: %s) auth failure count: %d/3 - Error: %v",
		account.UID, account.Email, failureCount, err)

	// 检查是否达到自动禁用阈值
	if failureCount >= 3 {
		disableErr := s.accountRepo.AutoDisableAccount(
			ctx,
			account.UID,
			"auto_disabled_auth_failure",
		)

		if disableErr != nil {
			log.Printf("[ERROR] Failed to auto-disable account %s: %v", account.UID, disableErr)
		} else {
			log.Printf("[INFO] Auto-disabled quick account %s (email: %s) after %d consecutive auth failures",
				account.UID, account.Email, failureCount)
		}
	}

	return err
}

// applyRulesForEmail 在同步阶段对单封邮件应用规则
func (s *syncService) applyRulesForEmail(ctx context.Context, email *model.Email) error {
	// 临时构建 ruleService（避免改动更大范围的依赖注入）
	ruleRepo := repository.NewRuleRepository(database.GetDB())
	rs := NewRuleService(ruleRepo, s.emailRepo)
	return rs.ApplyRules(ctx, email)
}
