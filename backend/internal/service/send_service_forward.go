package service

import (
	"context"
	"fmt"

	"fusionmail/internal/adapter"
)

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
	result, err := s.sendVia(ctx, sender, outgoingEmail)
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
