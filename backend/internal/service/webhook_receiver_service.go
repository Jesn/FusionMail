package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/webhook"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
)

// 模块日志记录器
var webhookReceiverLog = logger.NewWithModule("WebhookReceiver")

// WebhookReceiverService Webhook 接收服务接口
// 处理外部邮件服务商推送的 webhook 请求
type WebhookReceiverService interface {
	// ProcessEmail 处理 webhook 推送的邮件
	// providerType: 服务商类型（如 cloudflare_temp_email）
	// email: 标准化后的邮件数据
	// webhookSecret: 请求中携带的 webhook secret（用于匹配账户）
	ProcessEmail(ctx context.Context, providerType string, email *webhook.NormalizedEmail, webhookSecret string) (*webhook.WebhookResult, error)

	// FindAccountByWebhookSecret 根据 webhook secret 查找账户
	FindAccountByWebhookSecret(ctx context.Context, providerType, webhookSecret string) (*model.EmailAccount, error)
}

// webhookReceiverService Webhook 接收服务实现
type webhookReceiverService struct {
	accountRepo   repository.AccountRepository
	emailRepo     repository.EmailRepository
	providerRepo  repository.ProviderRepository
	cryptoService *crypto.Service // 加密服务，用于解密凭证
	notifier      SyncNotifier
	logger        *logger.Logger
}

// NewWebhookReceiverService 创建 Webhook 接收服务实例
func NewWebhookReceiverService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	providerRepo repository.ProviderRepository,
	cryptoService *crypto.Service,
	logger *logger.Logger,
) WebhookReceiverService {
	return NewWebhookReceiverServiceWithNotifier(accountRepo, emailRepo, providerRepo, cryptoService, logger, NewSSESyncNotifier())
}

func NewWebhookReceiverServiceWithNotifier(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	providerRepo repository.ProviderRepository,
	cryptoService *crypto.Service,
	logger *logger.Logger,
	notifier SyncNotifier,
) WebhookReceiverService {
	return &webhookReceiverService{
		accountRepo:   accountRepo,
		emailRepo:     emailRepo,
		providerRepo:  providerRepo,
		cryptoService: cryptoService,
		notifier:      resolveSyncNotifier(notifier),
		logger:        logger,
	}
}

// ProcessEmail 处理 webhook 推送的邮件
// 核心逻辑：
// 1. 根据 webhookSecret 找到配置了该 secret 的主账户
// 2. 根据邮件的 to 地址查找或创建子账户
// 3. 存储邮件到对应的子账户
func (s *webhookReceiverService) ProcessEmail(ctx context.Context, providerType string, email *webhook.NormalizedEmail, webhookSecret string) (*webhook.WebhookResult, error) {
	webhookReceiverLog.Info("处理 webhook 邮件: provider=%s, to=%s, subject=%s",
		providerType, email.To, webhook.TruncateString(email.Subject, 50))

	// 1. 根据 webhook secret 找到主账户
	masterAccount, err := s.FindAccountByWebhookSecret(ctx, providerType, webhookSecret)
	if err != nil {
		webhookReceiverLog.Error("根据 webhook secret 查找账户失败: err=%v", err)
		return nil, webhook.NewInvalidSecretError()
	}

	// 2. 根据 to 地址查找或创建子账户
	targetAccount, err := s.findOrCreateAccountByEmail(ctx, masterAccount, email.To)
	if err != nil {
		webhookReceiverLog.Error("查找或创建账户失败: to=%s, err=%v", email.To, err)
		return nil, webhook.NewStorageError("find or create account failed", err)
	}

	// 3. 检查重复邮件
	exists, err := s.checkDuplicate(ctx, targetAccount.UID, email.ProviderID)
	if err != nil {
		webhookReceiverLog.Error("检查重复邮件失败: account_uid=%s, provider_id=%s, err=%v",
			targetAccount.UID, email.ProviderID, err)
		return nil, webhook.NewStorageError("check duplicate failed", err)
	}
	if exists {
		webhookReceiverLog.Debug("邮件已存在，跳过: account_uid=%s, provider_id=%s",
			targetAccount.UID, email.ProviderID)
		return &webhook.WebhookResult{
			Success:    true,
			Message:    "Email already exists",
			Duplicate:  true,
			AccountUID: targetAccount.UID,
		}, nil
	}

	// 4. 创建邮件记录
	emailModel, err := s.createEmailModel(targetAccount, email)
	if err != nil {
		webhookReceiverLog.Error("创建邮件模型失败: err=%v", err)
		return nil, webhook.NewStorageError("create email model failed", err)
	}

	// 5. 存储邮件
	if err := s.emailRepo.Create(ctx, emailModel); err != nil {
		webhookReceiverLog.Error("存储邮件失败: account_uid=%s, err=%v", targetAccount.UID, err)
		return nil, webhook.NewStorageError("save email failed", err)
	}

	// 6. 更新账户统计
	if err := s.accountRepo.IncrementEmailCount(ctx, targetAccount.UID, 1); err != nil {
		webhookReceiverLog.Warn("更新邮件计数失败: account_uid=%s, err=%v", targetAccount.UID, err)
		// 不影响主流程
	}

	// 7. 更新账户的最后同步时间
	now := time.Now()
	targetAccount.LastSyncAt = &now
	targetAccount.LastSyncStatus = "success"
	if err := s.accountRepo.Update(ctx, targetAccount); err != nil {
		webhookReceiverLog.Warn("更新账户最后同步时间失败: account_uid=%s, err=%v", targetAccount.UID, err)
		// 不影响主流程
	}

	// Webhook 接收成功，重置失败计数（所有账号类型）
	if targetAccount.ConsecutiveAuthFailures > 0 {
		if resetErr := s.accountRepo.ResetConsecutiveFailures(ctx, targetAccount.UID); resetErr != nil {
			webhookReceiverLog.Error("重置失败计数失败: account=%s, err=%v", targetAccount.UID, resetErr)
		} else {
			webhookReceiverLog.Info("已重置失败计数: account=%s, 原失败次数=%d", targetAccount.UID, targetAccount.ConsecutiveAuthFailures)
		}
	}

	// 同时更新主账户的最后同步时间（如果是子账户）
	if targetAccount.ParentAccountUID != nil && *targetAccount.ParentAccountUID != "" {
		if err := s.updateParentAccountSyncTime(ctx, *targetAccount.ParentAccountUID); err != nil {
			webhookReceiverLog.Warn("更新主账户最后同步时间失败: parent_uid=%s, err=%v", *targetAccount.ParentAccountUID, err)
		}
	}

	webhookReceiverLog.Info("邮件处理成功: email_id=%d, account_uid=%s, provider_id=%s",
		emailModel.ID, targetAccount.UID, email.ProviderID)

	// 8. 通知前端有新邮件到达
	NotifyEmailCountsMaybeChanged(s.notifier, nil)
	webhookReceiverLog.Debug("已发送通知: email_counts_maybe_changed")

	return &webhook.WebhookResult{
		Success:    true,
		Message:    "Email processed successfully",
		EmailID:    emailModel.ID,
		AccountUID: targetAccount.UID,
	}, nil
}

// updateParentAccountSyncTime 更新主账户的最后同步时间
func (s *webhookReceiverService) updateParentAccountSyncTime(ctx context.Context, parentUID string) error {
	parentAccount, err := s.accountRepo.FindByUID(ctx, parentUID)
	if err != nil {
		return err
	}
	if parentAccount == nil {
		return nil
	}

	now := time.Now()
	parentAccount.LastSyncAt = &now
	parentAccount.LastSyncStatus = "success"
	return s.accountRepo.Update(ctx, parentAccount)
}

// FindAccountByWebhookSecret 根据 webhook secret 查找账户
// 遍历所有启用了 webhook 模式的账户，找到匹配 secret 的账户
func (s *webhookReceiverService) FindAccountByWebhookSecret(ctx context.Context, providerType, webhookSecret string) (*model.EmailAccount, error) {
	if webhookSecret == "" {
		return nil, webhook.ErrInvalidSecret
	}

	// 获取所有账户（这里可以优化为只查询 webhook 模式的账户）
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// 遍历查找匹配 secret 的账户
	for _, account := range accounts {
		secret := s.extractWebhookSecret(account)
		if secret != "" && secret == webhookSecret {
			// 验证是否为 webhook 模式
			if s.isWebhookMode(account) {
				return account, nil
			}
		}
	}

	return nil, webhook.ErrAccountNotFound
}

// findOrCreateAccountByEmail 根据邮箱地址查找或创建账户
// 如果邮箱地址对应的账户已存在，直接返回
// 如果不存在，创建一个子账户（关联到主账户）
func (s *webhookReceiverService) findOrCreateAccountByEmail(ctx context.Context, masterAccount *model.EmailAccount, email string) (*model.EmailAccount, error) {
	// 标准化邮箱地址
	email = webhook.NormalizeEmailAddress(email)

	// 1. 先尝试精确匹配邮箱地址
	account, err := s.accountRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account != nil {
		return account, nil
	}

	// 2. 不存在，创建子账户
	webhookReceiverLog.Info("创建子邮箱账户: master_uid=%s, email=%s", masterAccount.UID, email)

	childAccount, err := s.createChildAccount(ctx, masterAccount, email)
	if err != nil {
		return nil, fmt.Errorf("create child account failed: %w", err)
	}

	return childAccount, nil
}

// createChildAccount 创建子邮箱账户
func (s *webhookReceiverService) createChildAccount(ctx context.Context, parent *model.EmailAccount, email string) (*model.EmailAccount, error) {
	// 生成唯一 UID
	uid := fmt.Sprintf("webhook_%d", time.Now().UnixNano())

	// 继承父账户的配置
	var authData map[string]interface{}
	if parent.EncryptedCredentials != "" {
		if err := json.Unmarshal([]byte(parent.EncryptedCredentials), &authData); err != nil {
			authData = make(map[string]interface{})
		}
	} else {
		authData = make(map[string]interface{})
	}

	// 设置子账户的邮箱地址
	authData["email"] = email
	// 保留 webhook_secret 和 sync_mode

	authDataJSON, err := json.Marshal(authData)
	if err != nil {
		return nil, err
	}

	// 创建子账户
	childAccount := &model.EmailAccount{
		UID:                  uid,
		Email:                email,
		ProviderID:           parent.ProviderID,
		AdapterID:            parent.AdapterID,
		EncryptedCredentials: string(authDataJSON),
		Status:               "active",
		SyncEnabled:          false, // Webhook 模式不需要轮询同步
		SyncInterval:         parent.SyncInterval,
		GroupID:              parent.GroupID,
		ParentAccountUID:     &parent.UID, // 关联父账户
	}

	if err := s.accountRepo.Create(ctx, childAccount); err != nil {
		return nil, err
	}

	webhookReceiverLog.Info("子邮箱账户创建成功: uid=%s, email=%s, parent_uid=%s",
		childAccount.UID, email, parent.UID)

	return childAccount, nil
}

// isWebhookMode 检查账户是否为 webhook 模式
// 优先使用数据库字段 SyncModeField，如果为空则从解密后的凭证中读取
func (s *webhookReceiverService) isWebhookMode(account *model.EmailAccount) bool {
	// 优先使用数据库字段
	if account.SyncModeField != "" {
		return account.SyncModeField == "webhook"
	}

	// 回退到从凭证中读取
	if account.EncryptedCredentials == "" {
		return false
	}

	// 解密凭证数据
	decryptedData := account.EncryptedCredentials
	if s.cryptoService != nil {
		decrypted, err := s.cryptoService.Decrypt(account.EncryptedCredentials)
		if err != nil {
			webhookReceiverLog.Debug("解密凭证失败: account_uid=%s, err=%v", account.UID, err)
			return false
		}
		decryptedData = string(decrypted)
	}

	var authData map[string]interface{}
	if err := json.Unmarshal([]byte(decryptedData), &authData); err != nil {
		return false
	}

	syncMode, _ := authData["sync_mode"].(string)
	return syncMode == "webhook"
}

// extractWebhookSecret 从账户配置中提取 webhook secret
// 需要先解密 EncryptedCredentials，然后从 JSON 中提取 webhook_secret 字段
func (s *webhookReceiverService) extractWebhookSecret(account *model.EmailAccount) string {
	if account.EncryptedCredentials == "" {
		return ""
	}

	// 解密凭证数据
	decryptedData := account.EncryptedCredentials
	if s.cryptoService != nil {
		decrypted, err := s.cryptoService.Decrypt(account.EncryptedCredentials)
		if err != nil {
			webhookReceiverLog.Debug("解密凭证失败，尝试使用原始数据: account_uid=%s, err=%v", account.UID, err)
			// 如果解密失败，可能是未加密的数据，继续尝试解析
		} else {
			decryptedData = string(decrypted)
		}
	}

	var authData map[string]interface{}
	if err := json.Unmarshal([]byte(decryptedData), &authData); err != nil {
		webhookReceiverLog.Debug("解析凭证 JSON 失败: account_uid=%s, err=%v", account.UID, err)
		return ""
	}

	secret, _ := authData["webhook_secret"].(string)
	return secret
}

// checkDuplicate 检查邮件是否已存在
func (s *webhookReceiverService) checkDuplicate(ctx context.Context, accountUID, providerID string) (bool, error) {
	if providerID == "" {
		return false, nil
	}

	existing, err := s.emailRepo.FindByProviderID(ctx, providerID, accountUID)
	if err != nil {
		return false, err
	}

	return existing != nil, nil
}

// createEmailModel 创建邮件模型
func (s *webhookReceiverService) createEmailModel(account *model.EmailAccount, email *webhook.NormalizedEmail) (*model.Email, error) {
	now := time.Now()

	// 处理发送时间
	sentAt := now
	if email.SentAt != nil {
		sentAt = *email.SentAt
	}

	// 处理接收时间
	receivedAt := now
	if email.ReceivedAt != nil {
		receivedAt = *email.ReceivedAt
	}

	// 序列化收件人地址
	toAddressesJSON := ""
	if len(email.ToAddresses) > 0 {
		if data, err := json.Marshal(email.ToAddresses); err == nil {
			toAddressesJSON = string(data)
		}
	}

	// 序列化抄送地址
	ccAddressesJSON := ""
	if len(email.CcAddresses) > 0 {
		if data, err := json.Marshal(email.CcAddresses); err == nil {
			ccAddressesJSON = string(data)
		}
	}

	// 生成摘要
	snippet := webhook.ExtractSnippet(email.TextBody, email.HtmlBody, 200)

	// 生成去重标识
	dedupeKey := ""
	if email.MessageID != "" {
		dedupeKey = "mid:" + email.MessageID
	} else if email.ProviderID != "" {
		dedupeKey = "pid:" + email.ProviderID
	}

	// 创建邮件模型
	emailModel := &model.Email{
		ProviderID:       email.ProviderID,
		AccountUID:       account.UID,
		MessageID:        email.MessageID,
		DedupeKey:        dedupeKey,
		Subject:          email.Subject,
		FromAddress:      email.FromAddress,
		FromName:         email.FromName,
		ToAddress:        email.To,
		ToAddresses:      toAddressesJSON,
		CcAddresses:      ccAddressesJSON,
		ReplyTo:          email.ReplyTo,
		TextBody:         email.TextBody,
		HTMLBody:         email.HtmlBody,
		Snippet:          snippet,
		IsRead:           false,
		IsStarred:        false,
		IsArchived:       false,
		IsDeleted:        false,
		SentAt:           sentAt,
		ReceivedAt:       receivedAt,
		SyncedAt:         now,
		InReplyTo:        email.InReplyTo,
		References:       email.References,
		HasAttachment:    email.HasAttachments(),
		HasAttachments:   email.HasAttachments(),
		AttachmentsCount: email.AttachmentsCount(),
	}

	return emailModel, nil
}
