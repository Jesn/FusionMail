package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

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
