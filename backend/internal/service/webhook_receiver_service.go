package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/webhook"
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
	ProcessEmail(ctx context.Context, providerType string, email *webhook.NormalizedEmail) (*webhook.WebhookResult, error)

	// GetWebhookSecret 获取指定账户的 webhook secret
	GetWebhookSecret(ctx context.Context, providerType, toAddress string) (string, error)

	// ValidateWebhookEnabled 验证账户是否启用了 webhook 模式
	ValidateWebhookEnabled(ctx context.Context, providerType, toAddress string) error
}

// webhookReceiverService Webhook 接收服务实现
type webhookReceiverService struct {
	accountRepo  repository.AccountRepository
	emailRepo    repository.EmailRepository
	providerRepo repository.ProviderRepository
	logger       *logger.Logger
}

// NewWebhookReceiverService 创建 Webhook 接收服务实例
func NewWebhookReceiverService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	providerRepo repository.ProviderRepository,
	logger *logger.Logger,
) WebhookReceiverService {
	return &webhookReceiverService{
		accountRepo:  accountRepo,
		emailRepo:    emailRepo,
		providerRepo: providerRepo,
		logger:       logger,
	}
}

// ProcessEmail 处理 webhook 推送的邮件
func (s *webhookReceiverService) ProcessEmail(ctx context.Context, providerType string, email *webhook.NormalizedEmail) (*webhook.WebhookResult, error) {
	webhookReceiverLog.Info("处理 webhook 邮件: provider=%s, to=%s, subject=%s",
		providerType, email.To, webhook.TruncateString(email.Subject, 50))

	// 1. 根据收件人地址查找账户
	account, err := s.findAccountByEmail(ctx, email.To)
	if err != nil {
		webhookReceiverLog.Error("查找账户失败: to=%s, err=%v", email.To, err)
		return nil, webhook.NewAccountNotFoundError(email.To)
	}

	// 2. 验证账户是否启用了 webhook 模式
	if err := s.validateAccountWebhookMode(account); err != nil {
		webhookReceiverLog.Warn("账户未启用 webhook 模式: uid=%s, email=%s", account.UID, account.Email)
		// 不返回错误，仍然处理邮件（兼容性考虑）
	}

	// 3. 检查重复邮件
	exists, err := s.checkDuplicate(ctx, account.UID, email.ProviderID)
	if err != nil {
		webhookReceiverLog.Error("检查重复邮件失败: account_uid=%s, provider_id=%s, err=%v",
			account.UID, email.ProviderID, err)
		return nil, webhook.NewStorageError("check duplicate failed", err)
	}
	if exists {
		webhookReceiverLog.Debug("邮件已存在，跳过: account_uid=%s, provider_id=%s",
			account.UID, email.ProviderID)
		return &webhook.WebhookResult{
			Success:    true,
			Message:    "Email already exists",
			Duplicate:  true,
			AccountUID: account.UID,
		}, nil
	}

	// 4. 创建邮件记录
	emailModel, err := s.createEmailModel(account, email)
	if err != nil {
		webhookReceiverLog.Error("创建邮件模型失败: err=%v", err)
		return nil, webhook.NewStorageError("create email model failed", err)
	}

	// 5. 存储邮件
	if err := s.emailRepo.Create(ctx, emailModel); err != nil {
		webhookReceiverLog.Error("存储邮件失败: account_uid=%s, err=%v", account.UID, err)
		return nil, webhook.NewStorageError("save email failed", err)
	}

	// 6. 更新账户统计
	if err := s.accountRepo.IncrementEmailCount(ctx, account.UID, 1); err != nil {
		webhookReceiverLog.Warn("更新邮件计数失败: account_uid=%s, err=%v", account.UID, err)
		// 不影响主流程
	}

	webhookReceiverLog.Info("邮件处理成功: email_id=%d, account_uid=%s, provider_id=%s",
		emailModel.ID, account.UID, email.ProviderID)

	return &webhook.WebhookResult{
		Success:    true,
		Message:    "Email processed successfully",
		EmailID:    emailModel.ID,
		AccountUID: account.UID,
	}, nil
}

// GetWebhookSecret 获取指定账户的 webhook secret
func (s *webhookReceiverService) GetWebhookSecret(ctx context.Context, providerType, toAddress string) (string, error) {
	// 根据收件人地址查找账户
	account, err := s.findAccountByEmail(ctx, toAddress)
	if err != nil {
		return "", err
	}

	// 从账户的 auth_data 中提取 webhook_secret
	secret, err := s.extractWebhookSecret(account)
	if err != nil {
		return "", err
	}

	return secret, nil
}

// ValidateWebhookEnabled 验证账户是否启用了 webhook 模式
func (s *webhookReceiverService) ValidateWebhookEnabled(ctx context.Context, providerType, toAddress string) error {
	account, err := s.findAccountByEmail(ctx, toAddress)
	if err != nil {
		return err
	}

	return s.validateAccountWebhookMode(account)
}

// findAccountByEmail 根据邮箱地址查找账户
func (s *webhookReceiverService) findAccountByEmail(ctx context.Context, email string) (*model.EmailAccount, error) {
	// 标准化邮箱地址
	email = webhook.NormalizeEmailAddress(email)

	// 查找账户
	account, err := s.accountRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return nil, webhook.ErrAccountNotFound
	}

	return account, nil
}

// validateAccountWebhookMode 验证账户是否启用了 webhook 模式
func (s *webhookReceiverService) validateAccountWebhookMode(account *model.EmailAccount) error {
	// 解析 EncryptedCredentials（实际上存储的是 JSON 格式的认证数据）
	// 注意：这里假设 EncryptedCredentials 已经解密或者是明文存储的配置
	// 实际使用时可能需要先解密
	var authData map[string]interface{}
	if account.EncryptedCredentials != "" {
		if err := json.Unmarshal([]byte(account.EncryptedCredentials), &authData); err != nil {
			// 如果解析失败，可能是加密的数据，跳过验证
			return nil
		}
	}

	// 检查 sync_mode
	syncMode, _ := authData["sync_mode"].(string)
	if syncMode != "webhook" {
		return webhook.ErrWebhookDisabled
	}

	return nil
}

// extractWebhookSecret 从账户配置中提取 webhook secret
func (s *webhookReceiverService) extractWebhookSecret(account *model.EmailAccount) (string, error) {
	if account.EncryptedCredentials == "" {
		return "", nil
	}

	var authData map[string]interface{}
	if err := json.Unmarshal([]byte(account.EncryptedCredentials), &authData); err != nil {
		// 如果解析失败，可能是加密的数据，返回空
		return "", nil
	}

	secret, _ := authData["webhook_secret"].(string)
	return secret, nil
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
