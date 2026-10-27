package service

import (
	"context"
	"fmt"

	"fusionmail/internal/adapter"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	emailpkg "fusionmail/pkg/email"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/oauth2config"
)

// SendService 邮件发送服务
// Requirements: 1.1, 1.4, 1.5
type SendService struct {
	factory              *adapter.SenderFactory
	accountRepo          repository.AccountRepository
	sentEmailRepo        repository.SentEmailRepository
	emailRepo            repository.EmailRepository
	cryptoService        *crypto.Service        // 新增：加密服务
	oauth2ConfigProvider *oauth2config.Provider // 新增：OAuth2配置提供者
	logger               *logger.Logger
}

// NewSendService 创建发送服务实例
func NewSendService(
	factory *adapter.SenderFactory,
	accountRepo repository.AccountRepository,
	sentEmailRepo repository.SentEmailRepository,
	emailRepo repository.EmailRepository,
	cryptoService *crypto.Service,
	oauth2ConfigProvider *oauth2config.Provider,
	log *logger.Logger,
) *SendService {
	return &SendService{
		factory:              factory,
		accountRepo:          accountRepo,
		sentEmailRepo:        sentEmailRepo,
		emailRepo:            emailRepo,
		cryptoService:        cryptoService,
		oauth2ConfigProvider: oauth2ConfigProvider,
		logger:               log,
	}
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	AccountUID  string                       // 发送账户 UID
	To          []string                     // 收件人列表
	Cc          []string                     // 抄送列表
	Bcc         []string                     // 密送列表
	Subject     string                       // 主题
	TextBody    string                       // 纯文本正文
	HTMLBody    string                       // HTML 正文
	Attachments []adapter.OutgoingAttachment // 附件列表
	ReplyTo     string                       // 回复地址
}

// SendEmailResponse 发送邮件响应
type SendEmailResponse struct {
	Success       bool   `json:"success"`                   // 是否成功
	MessageID     string `json:"message_id,omitempty"`      // 邮件 Message-ID
	ProviderMsgID string `json:"provider_msg_id,omitempty"` // 服务商消息 ID
	SenderType    string `json:"sender_type,omitempty"`     // 使用的发送器类型
	Error         string `json:"error,omitempty"`           // 错误信息
	SentEmailID   int64  `json:"sent_email_id,omitempty"`   // 已发送邮件记录 ID
}

// SendEmail 发送邮件
// Requirements: 1.1, 1.4, 1.5
func (s *SendService) SendEmail(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error) {
	// 验证请求
	if err := s.validateSendRequest(req); err != nil {
		return &SendEmailResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 获取账户信息（预加载 Provider 关联，用于获取 SMTP 配置）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, req.AccountUID)
	if err != nil {
		return &SendEmailResponse{
			Success: false,
			Error:   "获取账户信息失败",
		}, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return &SendEmailResponse{
			Success: false,
			Error:   "账户不存在",
		}, fmt.Errorf("account not found: %s", req.AccountUID)
	}

	// 解析账户凭证
	credentials, err := s.parseCredentials(ctx, account)
	if err != nil {
		s.logger.Error("解析凭证失败: account=%s, error=%v", account.UID, err)
		return &SendEmailResponse{
			Success: false,
			Error:   fmt.Sprintf("解析凭证失败: %v", err),
		}, err
	}

	// 获取发送器（传入凭证）
	sender, err := s.factory.GetSenderWithFallback(account, credentials)
	if err != nil {
		return &SendEmailResponse{
			Success: false,
			Error:   fmt.Sprintf("无法获取发送器: %v", err),
		}, err
	}

	// 构建待发送邮件
	outgoingEmail := &adapter.OutgoingEmail{
		From:        account.Email,
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     req.Subject,
		TextBody:    req.TextBody,
		HTMLBody:    req.HTMLBody,
		Attachments: req.Attachments,
		ReplyTo:     req.ReplyTo,
		AccountUID:  req.AccountUID,
	}

	// 发送邮件
	result, err := sender.Send(ctx, outgoingEmail)
	if err != nil {
		// 记录失败的发送
		s.saveSentEmail(ctx, account, outgoingEmail, result, nil, nil)
		return &SendEmailResponse{
			Success:    false,
			Error:      result.Error,
			SenderType: result.SenderType,
		}, err
	}

	// 保存已发送邮件记录
	sentEmail, saveErr := s.saveSentEmail(ctx, account, outgoingEmail, result, nil, nil)
	if saveErr != nil {
		s.logger.Error("保存已发送邮件记录失败: %v", saveErr)
	}

	var sentEmailID int64
	if sentEmail != nil {
		sentEmailID = sentEmail.ID
	}

	return &SendEmailResponse{
		Success:       true,
		MessageID:     result.MessageID,
		ProviderMsgID: result.ProviderMsgID,
		SenderType:    result.SenderType,
		SentEmailID:   sentEmailID,
	}, nil
}

// validateSendRequest 验证发送请求
func (s *SendService) validateSendRequest(req *SendEmailRequest) error {
	if req.AccountUID == "" {
		return fmt.Errorf("发送账户不能为空")
	}

	if len(req.To) == 0 {
		return fmt.Errorf("收件人不能为空")
	}

	// 验证收件人邮箱格式
	for _, addr := range req.To {
		if !emailpkg.IsValidEmail(addr) {
			return fmt.Errorf("收件人邮箱格式不正确: %s", addr)
		}
	}

	for _, addr := range req.Cc {
		if !emailpkg.IsValidEmail(addr) {
			return fmt.Errorf("抄送邮箱格式不正确: %s", addr)
		}
	}

	for _, addr := range req.Bcc {
		if !emailpkg.IsValidEmail(addr) {
			return fmt.Errorf("密送邮箱格式不正确: %s", addr)
		}
	}

	// 验证附件大小
	var totalSize int64
	for _, att := range req.Attachments {
		if int64(len(att.Content)) > adapter.MaxSingleAttachmentSize {
			return fmt.Errorf("附件 %s 大小超过限制（最大 25MB）", att.Filename)
		}
		totalSize += int64(len(att.Content))
	}

	if totalSize > adapter.MaxTotalAttachmentSize {
		return fmt.Errorf("附件总大小超过限制（最大 50MB）")
	}

	return nil
}
