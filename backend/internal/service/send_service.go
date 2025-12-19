package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
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

// ReplyRequest 回复邮件请求
type ReplyRequest struct {
	TextBody    string                       // 纯文本正文
	HTMLBody    string                       // HTML 正文
	Attachments []adapter.OutgoingAttachment // 附件列表
}

// Reply 回复邮件
// Requirements: 5.1, 5.4
func (s *SendService) Reply(ctx context.Context, emailID int64, accountUID string, req *ReplyRequest) (*SendEmailResponse, error) {
	// 获取原邮件
	originalEmail, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original email: %w", err)
	}
	if originalEmail == nil {
		return nil, fmt.Errorf("original email not found")
	}

	// 构建回复邮件请求
	sendReq := &SendEmailRequest{
		AccountUID:  accountUID,
		To:          []string{originalEmail.FromAddress},
		Subject:     s.buildReplySubject(originalEmail.Subject),
		TextBody:    s.buildReplyBody(req.TextBody, originalEmail),
		HTMLBody:    s.buildReplyHTMLBody(req.HTMLBody, originalEmail),
		Attachments: req.Attachments,
	}

	// 获取账户信息（预加载 Provider 关联，用于获取 SMTP 配置）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil || account == nil {
		return nil, fmt.Errorf("account not found")
	}

	// 解析账户凭证
	credentials, err := s.parseCredentials(ctx, account)
	if err != nil {
		s.logger.Error("解析凭证失败: account=%s, error=%v", account.UID, err)
		return nil, fmt.Errorf("解析凭证失败: %w", err)
	}

	// 获取发送器（传入凭证）
	sender, err := s.factory.GetSenderWithFallback(account, credentials)
	if err != nil {
		return nil, err
	}

	// 构建待发送邮件（包含回复头）
	outgoingEmail := &adapter.OutgoingEmail{
		From:        account.Email,
		To:          sendReq.To,
		Subject:     sendReq.Subject,
		TextBody:    sendReq.TextBody,
		HTMLBody:    sendReq.HTMLBody,
		Attachments: sendReq.Attachments,
		InReplyTo:   originalEmail.MessageID,
		References:  s.buildReferences(originalEmail),
		AccountUID:  accountUID,
	}

	// 发送邮件
	result, err := sender.Send(ctx, outgoingEmail)
	if err != nil {
		s.saveSentEmail(ctx, account, outgoingEmail, result, &emailID, nil)
		return &SendEmailResponse{
			Success:    false,
			Error:      result.Error,
			SenderType: result.SenderType,
		}, err
	}

	// 保存已发送邮件记录
	sentEmail, _ := s.saveSentEmail(ctx, account, outgoingEmail, result, &emailID, nil)

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

// ReplyAll 全部回复
// Requirements: 5.2, 5.4
func (s *SendService) ReplyAll(ctx context.Context, emailID int64, accountUID string, req *ReplyRequest) (*SendEmailResponse, error) {
	// 获取原邮件
	originalEmail, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original email: %w", err)
	}
	if originalEmail == nil {
		return nil, fmt.Errorf("original email not found")
	}

	// 获取账户信息（预加载 Provider 关联，用于获取 SMTP 配置）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil || account == nil {
		return nil, fmt.Errorf("account not found")
	}

	// 构建收件人列表（原发件人 + 原收件人，排除自己）
	recipients := s.buildReplyAllRecipients(originalEmail, account.Email)

	// 解析账户凭证
	credentials, err := s.parseCredentials(ctx, account)
	if err != nil {
		s.logger.Error("解析凭证失败: account=%s, error=%v", account.UID, err)
		return nil, fmt.Errorf("解析凭证失败: %w", err)
	}

	// 获取发送器（传入凭证）
	sender, err := s.factory.GetSenderWithFallback(account, credentials)
	if err != nil {
		return nil, err
	}

	// 构建待发送邮件
	outgoingEmail := &adapter.OutgoingEmail{
		From:        account.Email,
		To:          recipients.To,
		Cc:          recipients.Cc,
		Subject:     s.buildReplySubject(originalEmail.Subject),
		TextBody:    s.buildReplyBody(req.TextBody, originalEmail),
		HTMLBody:    s.buildReplyHTMLBody(req.HTMLBody, originalEmail),
		Attachments: req.Attachments,
		InReplyTo:   originalEmail.MessageID,
		References:  s.buildReferences(originalEmail),
		AccountUID:  accountUID,
	}

	// 发送邮件
	result, err := sender.Send(ctx, outgoingEmail)
	if err != nil {
		s.saveSentEmail(ctx, account, outgoingEmail, result, &emailID, nil)
		return &SendEmailResponse{
			Success:    false,
			Error:      result.Error,
			SenderType: result.SenderType,
		}, err
	}

	// 保存已发送邮件记录
	sentEmail, _ := s.saveSentEmail(ctx, account, outgoingEmail, result, &emailID, nil)

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

// ForwardRequest 转发邮件请求
type ForwardRequest struct {
	To          []string                     // 收件人列表
	Cc          []string                     // 抄送列表
	TextBody    string                       // 附加的纯文本正文
	HTMLBody    string                       // 附加的 HTML 正文
	Attachments []adapter.OutgoingAttachment // 额外附件
}

// Forward 转发邮件
// Requirements: 5.3, 5.5
func (s *SendService) Forward(ctx context.Context, emailID int64, accountUID string, req *ForwardRequest) (*SendEmailResponse, error) {
	// 获取原邮件
	originalEmail, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original email: %w", err)
	}
	if originalEmail == nil {
		return nil, fmt.Errorf("original email not found")
	}

	// 获取账户信息（预加载 Provider 关联，用于获取 SMTP 配置）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil || account == nil {
		return nil, fmt.Errorf("account not found")
	}

	// 解析账户凭证
	credentials, err := s.parseCredentials(ctx, account)
	if err != nil {
		s.logger.Error("解析凭证失败: account=%s, error=%v", account.UID, err)
		return nil, fmt.Errorf("解析凭证失败: %w", err)
	}

	// 获取发送器（传入凭证）
	sender, err := s.factory.GetSenderWithFallback(account, credentials)
	if err != nil {
		return nil, err
	}

	// 构建转发邮件
	outgoingEmail := &adapter.OutgoingEmail{
		From:        account.Email,
		To:          req.To,
		Cc:          req.Cc,
		Subject:     s.buildForwardSubject(originalEmail.Subject),
		TextBody:    s.buildForwardBody(req.TextBody, originalEmail),
		HTMLBody:    s.buildForwardHTMLBody(req.HTMLBody, originalEmail),
		Attachments: req.Attachments, // TODO: 包含原邮件附件
		AccountUID:  accountUID,
	}

	// 发送邮件
	result, err := sender.Send(ctx, outgoingEmail)
	if err != nil {
		s.saveSentEmail(ctx, account, outgoingEmail, result, nil, &emailID)
		return &SendEmailResponse{
			Success:    false,
			Error:      result.Error,
			SenderType: result.SenderType,
		}, err
	}

	// 保存已发送邮件记录
	sentEmail, _ := s.saveSentEmail(ctx, account, outgoingEmail, result, nil, &emailID)

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

// saveSentEmail 保存已发送邮件记录
func (s *SendService) saveSentEmail(
	ctx context.Context,
	account *model.EmailAccount,
	email *adapter.OutgoingEmail,
	result *adapter.SendResult,
	replyToID *int64,
	forwardFromID *int64,
) (*model.SentEmail, error) {
	// 序列化收件人列表
	toJSON, _ := json.Marshal(email.To)
	ccJSON, _ := json.Marshal(email.Cc)
	bccJSON, _ := json.Marshal(email.Bcc)

	// 序列化附件信息
	var attachmentInfo string
	if len(email.Attachments) > 0 {
		attInfos := make([]model.SentEmailAttachmentInfo, len(email.Attachments))
		for i, att := range email.Attachments {
			attInfos[i] = model.SentEmailAttachmentInfo{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				SizeBytes:   int64(len(att.Content)),
			}
		}
		attJSON, _ := json.Marshal(attInfos)
		attachmentInfo = string(attJSON)
	}

	status := model.SentEmailStatusSent
	if !result.Success {
		status = model.SentEmailStatusFailed
	}

	sentEmail := &model.SentEmail{
		AccountUID:      account.UID,
		ProviderMsgID:   result.ProviderMsgID,
		MessageID:       result.MessageID,
		Subject:         email.Subject,
		FromAddress:     account.Email,
		ToAddresses:     string(toJSON),
		CcAddresses:     string(ccJSON),
		BccAddresses:    string(bccJSON),
		TextBody:        email.TextBody,
		HTMLBody:        email.HTMLBody,
		HasAttachments:  len(email.Attachments) > 0,
		AttachmentCount: len(email.Attachments),
		AttachmentInfo:  attachmentInfo,
		ReplyToEmailID:  replyToID,
		ForwardFromID:   forwardFromID,
		InReplyTo:       email.InReplyTo,
		References:      email.References,
		Status:          status,
		ErrorMessage:    result.Error,
		SentAt:          time.Now(),
		SenderType:      result.SenderType,
	}

	if err := s.sentEmailRepo.Create(ctx, sentEmail); err != nil {
		return nil, err
	}

	return sentEmail, nil
}

// buildReplySubject 构建回复主题
func (s *SendService) buildReplySubject(originalSubject string) string {
	if strings.HasPrefix(strings.ToLower(originalSubject), "re:") {
		return originalSubject
	}
	return "Re: " + originalSubject
}

// buildForwardSubject 构建转发主题
// Requirements: 5.5
func (s *SendService) buildForwardSubject(originalSubject string) string {
	if strings.HasPrefix(strings.ToLower(originalSubject), "fwd:") {
		return originalSubject
	}
	return "Fwd: " + originalSubject
}

// buildReferences 构建 References 头
func (s *SendService) buildReferences(originalEmail *model.Email) string {
	refs := originalEmail.References
	if refs != "" {
		refs += " "
	}
	if originalEmail.MessageID != "" {
		refs += "<" + originalEmail.MessageID + ">"
	}
	return refs
}

// buildReplyBody 构建回复正文
func (s *SendService) buildReplyBody(newBody string, originalEmail *model.Email) string {
	return fmt.Sprintf("%s\n\n-------- 原始邮件 --------\n发件人: %s\n日期: %s\n主题: %s\n\n%s",
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.Subject,
		originalEmail.TextBody,
	)
}

// buildReplyHTMLBody 构建回复 HTML 正文
func (s *SendService) buildReplyHTMLBody(newBody string, originalEmail *model.Email) string {
	if newBody == "" && originalEmail.HTMLBody == "" {
		return ""
	}

	originalContent := originalEmail.HTMLBody
	if originalContent == "" {
		originalContent = "<pre>" + originalEmail.TextBody + "</pre>"
	}

	return fmt.Sprintf(`%s
<br><br>
<div style="border-left: 2px solid #ccc; padding-left: 10px; margin-left: 10px;">
<p><strong>发件人:</strong> %s<br>
<strong>日期:</strong> %s<br>
<strong>主题:</strong> %s</p>
%s
</div>`,
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.Subject,
		originalContent,
	)
}

// buildForwardBody 构建转发正文
func (s *SendService) buildForwardBody(newBody string, originalEmail *model.Email) string {
	return fmt.Sprintf("%s\n\n-------- 转发邮件 --------\n发件人: %s\n日期: %s\n收件人: %s\n主题: %s\n\n%s",
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.ToAddress,
		originalEmail.Subject,
		originalEmail.TextBody,
	)
}

// buildForwardHTMLBody 构建转发 HTML 正文
func (s *SendService) buildForwardHTMLBody(newBody string, originalEmail *model.Email) string {
	if newBody == "" && originalEmail.HTMLBody == "" {
		return ""
	}

	originalContent := originalEmail.HTMLBody
	if originalContent == "" {
		originalContent = "<pre>" + originalEmail.TextBody + "</pre>"
	}

	return fmt.Sprintf(`%s
<br><br>
<div style="border: 1px solid #ccc; padding: 10px; margin: 10px 0;">
<p><strong>---------- 转发邮件 ----------</strong><br>
<strong>发件人:</strong> %s<br>
<strong>日期:</strong> %s<br>
<strong>收件人:</strong> %s<br>
<strong>主题:</strong> %s</p>
<hr>
%s
</div>`,
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.ToAddress,
		originalEmail.Subject,
		originalContent,
	)
}

// ReplyAllRecipients 全部回复收件人
type ReplyAllRecipients struct {
	To []string
	Cc []string
}

// buildReplyAllRecipients 构建全部回复收件人列表
func (s *SendService) buildReplyAllRecipients(originalEmail *model.Email, myEmail string) ReplyAllRecipients {
	result := ReplyAllRecipients{
		To: []string{},
		Cc: []string{},
	}

	myEmailLower := strings.ToLower(myEmail)

	// 原发件人作为主要收件人
	if strings.ToLower(originalEmail.FromAddress) != myEmailLower {
		result.To = append(result.To, originalEmail.FromAddress)
	}

	// 解析原收件人列表
	var toAddresses []string
	if originalEmail.ToAddresses != "" {
		json.Unmarshal([]byte(originalEmail.ToAddresses), &toAddresses)
	}

	// 原收件人（排除自己）加入抄送
	for _, addr := range toAddresses {
		if strings.ToLower(addr) != myEmailLower {
			result.Cc = append(result.Cc, addr)
		}
	}

	// 解析原抄送列表
	var ccAddresses []string
	if originalEmail.CcAddresses != "" {
		json.Unmarshal([]byte(originalEmail.CcAddresses), &ccAddresses)
	}

	// 原抄送（排除自己）也加入抄送
	for _, addr := range ccAddresses {
		if strings.ToLower(addr) != myEmailLower {
			result.Cc = append(result.Cc, addr)
		}
	}

	return result
}

// parseCredentials 解析账户凭证
// 从加密的凭证数据中提取 OAuth2 或密码凭证
func (s *SendService) parseCredentials(ctx context.Context, account *model.EmailAccount) (*adapter.AccountCredentials, error) {
	// 如果没有加密凭证，返回 nil（SMTP 发送可能不需要凭证）
	if account.EncryptedCredentials == "" {
		return nil, nil
	}

	// 解密凭证数据
	decryptedData, err := s.cryptoService.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// 根据认证类型处理凭证
	authType := account.GetAuthType()
	if authType == "oauth2" {
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

		credentials := &adapter.AccountCredentials{
			AccessToken:  oauthCreds.AccessToken,
			RefreshToken: oauthCreds.RefreshToken,
			TokenExpiry:  oauthCreds.TokenExpiry,
		}

		// 获取 OAuth2 配置（ClientID 和 ClientSecret）
		if account.ProviderID > 0 {
			oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2ConfigByProviderID(ctx, account.ProviderID)
			if err != nil {
				s.logger.Warn("获取 OAuth2 配置失败: provider_id=%d, error=%v", account.ProviderID, err)
			} else if oauth2Config != nil {
				credentials.ClientID = oauth2Config.ClientID
				credentials.ClientSecret = oauth2Config.ClientSecret
			}
		} else {
			providerName := account.GetProviderName()
			if providerName != "" {
				oauth2Config, err := s.oauth2ConfigProvider.GetOAuth2ConfigByName(ctx, providerName)
				if err != nil {
					s.logger.Warn("获取 OAuth2 配置失败: provider=%s, error=%v", providerName, err)
				} else if oauth2Config != nil {
					credentials.ClientID = oauth2Config.ClientID
					credentials.ClientSecret = oauth2Config.ClientSecret
				}
			}
		}

		return credentials, nil
	}

	// 密码认证：凭证就是密码本身
	// 对于 SMTP 发送，密码已经存储在 EncryptedSMTPPassword 中
	// 这里返回 nil，让 SenderFactory 使用 SMTP 配置
	return nil, nil
}
