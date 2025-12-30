package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// WebAPISyncService WebAPI 邮件同步服务
// 负责从 WebAPI Provider 拉取邮件并分发到对应的 EmailAccount
type WebAPISyncService struct {
	accountRepo repository.AccountRepository
	emailRepo   repository.EmailRepository
	syncLogRepo repository.SyncLogRepository
	logger      *logger.Logger
}

// NewWebAPISyncService 创建 WebAPI 同步服务
func NewWebAPISyncService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
) *WebAPISyncService {
	return &WebAPISyncService{
		accountRepo: accountRepo,
		emailRepo:   emailRepo,
		syncLogRepo: syncLogRepo,
		logger:      logger.NewWithModule("WebAPISync"),
	}
}

// SyncProvider 同步 WebAPI Provider 的邮件
// 从 Provider 拉取邮件，根据 TargetAddress 分发到对应的 EmailAccount
func (s *WebAPISyncService) SyncProvider(ctx context.Context, provider webapi.WebAPIProvider, providerAccountUID string) (*webapi.SyncResult, error) {
	s.logger.Info("开始 WebAPI 同步: provider=%s, account=%s", provider.GetServiceName(), providerAccountUID)

	// 创建同步结果
	result := webapi.NewSyncResult()

	// 连接到 WebAPI 服务
	if err := provider.Connect(ctx); err != nil {
		return nil, fmt.Errorf("连接 WebAPI 服务失败: %w", err)
	}
	defer provider.Disconnect()

	// 获取同步检查点
	checkpoint := provider.GetSyncCheckpoint()
	since := checkpoint.LastSyncTime

	// 拉取邮件
	emails, err := provider.FetchEmails(ctx, since, 0)
	if err != nil {
		return nil, fmt.Errorf("拉取邮件失败: %w", err)
	}

	s.logger.Info("拉取到 %d 封邮件", len(emails))

	// 转换为 WebAPIEmail
	webEmails := make([]*webapi.WebAPIEmail, 0, len(emails))
	for _, email := range emails {
		// 提取目标地址
		targetAddr := webapi.ExtractTargetAddress(email)
		webEmail := webapi.NewWebAPIEmail(email, targetAddr)
		webEmails = append(webEmails, webEmail)
	}

	// 按目标地址分组
	grouped := s.groupByTargetAddress(webEmails)

	// 分发到各个 EmailAccount
	for targetAddr, groupEmails := range grouped {
		if err := s.dispatchToAccount(ctx, providerAccountUID, targetAddr, groupEmails, result); err != nil {
			s.logger.Error("分发邮件失败: target=%s, err=%v", targetAddr, err)
			result.ErrorCount++
			continue
		}
	}

	// 更新同步检查点
	newCheckpoint := &webapi.SyncCheckpoint{
		LastSyncTime: time.Now(),
		TotalSynced:  checkpoint.TotalSynced + int64(result.TotalCount),
		LastCount:    result.TotalCount,
	}
	if err := provider.UpdateSyncCheckpoint(newCheckpoint); err != nil {
		s.logger.Warn("更新同步检查点失败: %v", err)
	}

	s.logger.Info("WebAPI 同步完成: total=%d, errors=%d, skipped=%d",
		result.TotalCount, result.ErrorCount, result.SkippedCount)

	return result, nil
}

// groupByTargetAddress 按目标地址分组邮件
func (s *WebAPISyncService) groupByTargetAddress(emails []*webapi.WebAPIEmail) map[string][]*webapi.WebAPIEmail {
	grouped := make(map[string][]*webapi.WebAPIEmail)

	for _, email := range emails {
		targetAddr := strings.ToLower(email.TargetAddress)
		if targetAddr == "" {
			// 没有目标地址的邮件，使用特殊键
			targetAddr = "_unknown_"
		}
		grouped[targetAddr] = append(grouped[targetAddr], email)
	}

	return grouped
}

// dispatchToAccount 分发邮件到指定的 EmailAccount
func (s *WebAPISyncService) dispatchToAccount(
	ctx context.Context,
	providerAccountUID string,
	targetAddress string,
	emails []*webapi.WebAPIEmail,
	result *webapi.SyncResult,
) error {
	// 查找或创建对应的 EmailAccount
	account, err := s.findOrCreateAccount(ctx, providerAccountUID, targetAddress)
	if err != nil {
		return fmt.Errorf("查找/创建账户失败: %w", err)
	}

	if account == nil {
		// 账户不存在且无法创建（如 Admin 模式下未配置的邮箱）
		s.logger.Debug("跳过未配置的邮箱: %s", targetAddress)
		result.SkippedCount += len(emails)
		return nil
	}

	// 保存邮件到数据库
	for _, webEmail := range emails {
		if err := s.saveEmail(ctx, account.UID, webEmail); err != nil {
			s.logger.Warn("保存邮件失败: account=%s, err=%v", account.UID, err)
			result.ErrorCount++
			continue
		}
		result.AddEmail(webEmail)
	}

	return nil
}

// findOrCreateAccount 查找或创建 EmailAccount
// 对于 Admin 模式，只查找已存在的账户，不自动创建
func (s *WebAPISyncService) findOrCreateAccount(ctx context.Context, providerAccountUID string, targetAddress string) (*model.EmailAccount, error) {
	if targetAddress == "" || targetAddress == "_unknown_" {
		return nil, nil
	}

	// 先查找是否已存在对应的 EmailAccount
	account, err := s.accountRepo.FindByEmail(ctx, targetAddress)
	if err != nil {
		return nil, err
	}

	if account != nil {
		return account, nil
	}

	// 账户不存在
	// 对于 Admin 模式，不自动创建账户，需要用户手动配置
	// 这里返回 nil 表示跳过该邮箱的邮件
	s.logger.Debug("邮箱账户不存在，跳过: %s", targetAddress)
	return nil, nil
}

// saveEmail 保存邮件到数据库
func (s *WebAPISyncService) saveEmail(ctx context.Context, accountUID string, webEmail *webapi.WebAPIEmail) error {
	adapterEmail := webEmail.ToEmail()

	// 检查邮件是否已存在（通过 ProviderID 或 MessageID）
	existingEmail, err := s.emailRepo.FindByProviderID(ctx, adapterEmail.ProviderID, accountUID)
	if err != nil {
		return err
	}

	if existingEmail != nil {
		// 邮件已存在，更新
		s.updateEmailFromAdapter(existingEmail, adapterEmail, accountUID)
		return s.emailRepo.Update(ctx, existingEmail)
	}

	// 新邮件，创建
	newEmail := s.createEmailFromAdapter(adapterEmail, accountUID)
	return s.emailRepo.Create(ctx, newEmail)
}

// createEmailFromAdapter 从适配器邮件创建数据库邮件模型
func (s *WebAPISyncService) createEmailFromAdapter(adapterEmail *adapter.Email, accountUID string) *model.Email {
	return &model.Email{
		ProviderID:       adapterEmail.ProviderID,
		AccountUID:       accountUID,
		MessageID:        adapterEmail.MessageID,
		Subject:          adapterEmail.Subject,
		FromAddress:      adapterEmail.FromAddress,
		FromName:         adapterEmail.FromName,
		ToAddresses:      strings.Join(adapterEmail.ToAddresses, ","),
		CcAddresses:      strings.Join(adapterEmail.CcAddresses, ","),
		BccAddresses:     strings.Join(adapterEmail.BccAddresses, ""),
		ReplyTo:          adapterEmail.ReplyTo,
		TextBody:         adapterEmail.TextBody,
		HTMLBody:         adapterEmail.HTMLBody,
		Snippet:          adapterEmail.Snippet,
		HasAttachments:   adapterEmail.HasAttachments,
		AttachmentsCount: adapterEmail.AttachmentsCount,
		SentAt:           adapterEmail.SentAt,
		ReceivedAt:       adapterEmail.ReceivedAt,
		SizeBytes:        adapterEmail.SizeBytes,
		ThreadID:         adapterEmail.ThreadID,
		InReplyTo:        adapterEmail.InReplyTo,
		References:       adapterEmail.References,
		IsRead:           false, // 新邮件默认未读
	}
}

// updateEmailFromAdapter 从适配器邮件更新数据库邮件模型
func (s *WebAPISyncService) updateEmailFromAdapter(email *model.Email, adapterEmail *adapter.Email, accountUID string) {
	// 更新可能变化的字段
	email.Subject = adapterEmail.Subject
	email.TextBody = adapterEmail.TextBody
	email.HTMLBody = adapterEmail.HTMLBody
	email.Snippet = adapterEmail.Snippet
	email.HasAttachments = adapterEmail.HasAttachments
	email.AttachmentsCount = adapterEmail.AttachmentsCount

	// 更新源邮箱状态（如果有）
	if adapterEmail.SourceIsRead != nil {
		// 注意：这里不直接覆盖本地已读状态，只记录源状态
		// 本地已读状态由用户操作控制
	}
}

// SyncWebAPIEmails 同步 WebAPI 邮件的便捷方法
// 接收已经拉取好的 WebAPIEmail 列表，进行分发和存储
func (s *WebAPISyncService) SyncWebAPIEmails(ctx context.Context, providerAccountUID string, emails []*webapi.WebAPIEmail) (*webapi.SyncResult, error) {
	result := webapi.NewSyncResult()

	// 按目标地址分组
	grouped := s.groupByTargetAddress(emails)

	// 分发到各个 EmailAccount
	for targetAddr, groupEmails := range grouped {
		if err := s.dispatchToAccount(ctx, providerAccountUID, targetAddr, groupEmails, result); err != nil {
			s.logger.Error("分发邮件失败: target=%s, err=%v", targetAddr, err)
			result.ErrorCount++
			continue
		}
	}

	return result, nil
}

// GetOrCreateAccountForEmail 获取或创建邮箱账户
// 用于 Single 模式，确保目标邮箱账户存在
func (s *WebAPISyncService) GetOrCreateAccountForEmail(ctx context.Context, email string, providerID int64, adapterID int64) (*model.EmailAccount, error) {
	// 先查找是否已存在
	account, err := s.accountRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if account != nil {
		return account, nil
	}

	// 创建新账户
	newAccount := &model.EmailAccount{
		Email:       email,
		ProviderID:  providerID,
		AdapterID:   adapterID,
		Status:      "active",
		SyncEnabled: true,
	}

	if err := s.accountRepo.Create(ctx, newAccount); err != nil {
		return nil, fmt.Errorf("创建账户失败: %w", err)
	}

	s.logger.Info("创建新邮箱账户: email=%s", email)
	return newAccount, nil
}
