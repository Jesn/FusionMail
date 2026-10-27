package service

import (
	"context"
	"encoding/json"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

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
