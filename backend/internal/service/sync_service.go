package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
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
	accountRepo    repository.AccountRepository
	emailRepo      repository.EmailRepository
	syncLogRepo    repository.SyncLogRepository
	adapterFactory *adapter.Factory
	encryptor      crypto.Encryptor
	schedulerStop  chan struct{}
}

// NewSyncService 创建邮件同步服务实例
func NewSyncService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	adapterFactory *adapter.Factory,
) SyncService {
	encryptor, _ := crypto.NewEncryptor()
	return &syncService{
		accountRepo:    accountRepo,
		emailRepo:      emailRepo,
		syncLogRepo:    syncLogRepo,
		adapterFactory: adapterFactory,
		encryptor:      encryptor,
	}
}

// SyncAccount 同步指定账户的邮件
func (s *syncService) SyncAccount(ctx context.Context, accountUID string) error {
	// 获取账户信息
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountUID)
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
		log.Printf("Sync failed for account %s: %v", accountUID, err)
	} else {
		syncLog.Status = "success"
		log.Printf("Sync completed for account %s: %d new emails", accountUID, syncLog.EmailsNew)
	}

	completedAt := time.Now()
	syncLog.CompletedAt = &completedAt
	syncLog.DurationMs = int(time.Since(syncLog.StartedAt).Milliseconds())

	// 更新账户同步状态
	account.LastSyncAt = &completedAt
	account.LastSyncStatus = syncLog.Status
	account.LastSyncError = syncLog.ErrorMessage
	if err := s.accountRepo.Update(ctx, account); err != nil {
		log.Printf("Failed to update account sync status: %v", err)
	}

	return err
}

// doSync 执行实际的同步逻辑
func (s *syncService) doSync(ctx context.Context, account *model.Account, syncLog *model.SyncLog) error {
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

	// 创建适配器
	provider, err := s.adapterFactory.CreateProviderFromAccount(
		account.Provider,
		account.Protocol,
		credentials,
		proxy,
	)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// 连接到邮箱服务器
	if err := provider.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer provider.Disconnect()

	// 确定同步起始时间（增量同步）
	since := time.Time{}
	if account.LastSyncAt != nil {
		// 增量同步：从上次同步时间开始（减去 5 分钟缓冲，避免遗漏）
		since = account.LastSyncAt.Add(-5 * time.Minute)
		log.Printf("Incremental sync for account %s since %s", account.UID, since.Format(time.RFC3339))
	} else {
		// 首次同步：从 30 天前开始
		since = time.Now().AddDate(0, 0, -30)
		log.Printf("Initial sync for account %s since %s", account.UID, since.Format(time.RFC3339))
	}

	// 拉取邮件列表
	emails, err := provider.FetchEmails(ctx, since, 1000) // 限制每次最多 1000 封
	if err != nil {
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	syncLog.EmailsFetched = len(emails)

	// 处理邮件
	for _, email := range emails {
		if err := s.processEmail(ctx, account.UID, email, syncLog); err != nil {
			log.Printf("Failed to process email %s: %v", email.ProviderID, err)
			continue
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
		syncLog.EmailsUpdated++
	} else {
		// 新邮件，创建
		newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)
		if err := s.emailRepo.Create(ctx, newEmail); err != nil {
			return err
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

// SyncAllAccounts 同步所有启用的账户
func (s *syncService) SyncAllAccounts(ctx context.Context) error {
	// 获取所有启用同步的账户
	accounts, err := s.accountRepo.ListSyncEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sync enabled accounts: %w", err)
	}

	log.Printf("Starting sync for %d accounts", len(accounts))

	// 并发同步账户
	for _, account := range accounts {
		go func(accountUID string) {
			if err := s.SyncAccount(ctx, accountUID); err != nil {
				log.Printf("Failed to sync account %s: %v", accountUID, err)
			}
		}(account.UID)
	}

	return nil
}

// StartScheduler 启动定时同步调度器
func (s *syncService) StartScheduler(ctx context.Context) error {
	s.schedulerStop = make(chan struct{})

	go func() {
		ticker := time.NewTicker(5 * time.Minute) // 每 5 分钟检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.SyncAllAccounts(ctx); err != nil {
					log.Printf("Scheduled sync failed: %v", err)
				}
			case <-s.schedulerStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Println("Sync scheduler started")
	return nil
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
func (s *syncService) parseCredentials(account *model.Account) (*adapter.Credentials, error) {
	// 解密密码
	password, err := s.encryptor.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// 根据提供商设置默认 IMAP 配置
	credentials := &adapter.Credentials{
		Email:    account.Email,
		Password: password,
		AuthType: account.AuthType,
	}

	// 设置 IMAP 服务器配置
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
	default:
		return nil, fmt.Errorf("unsupported provider: %s", account.Provider)
	}

	return credentials, nil
}

// parseProxyConfig 解析代理配置
func (s *syncService) parseProxyConfig(account *model.Account) (*adapter.ProxyConfig, error) {
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
