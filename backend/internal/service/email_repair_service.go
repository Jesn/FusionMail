package service

import (
	"context"
	"strings"

	"fusionmail/internal/model"
)

// looksLikeRawMIME 粗略判断一段文本是否像原始 MIME 源文
func looksLikeRawMIME(s string) bool {
	if s == "" {
		return false
	}
	ls := strings.ToLower(s)
	if strings.Contains(ls, "content-transfer-encoding:") && strings.Contains(ls, "content-type:") {
		return true
	}
	if strings.Contains(ls, "mime-version:") && strings.Contains(ls, "content-type:") {
		return true
	}
	return false
}

// tryRepairEmailBody 即时从远端拉取并解析邮件详情，更新本地存储
func (s *emailService) tryRepairEmailBody(ctx context.Context, email *model.Email) (bool, error) {
	// 获取账号
	account, err := s.accountRepo.FindByUID(ctx, email.AccountUID)
	if err != nil || account == nil {
		return false, err
	}

	// 解析凭证
	credentials, err := s.credentialResolver.Resolve(account)
	if err != nil {
		return false, err
	}

	// 创建适配器并连接
	providerName := account.GetProviderName()
	protocol := account.GetProtocol()
	mailAdapter, err := s.adapterFactory.CreateProviderFromAccount(
		providerName,
		protocol,
		credentials,
		nil,
	)
	if err != nil {
		return false, err
	}
	if err := mailAdapter.Connect(ctx); err != nil {
		return false, err
	}
	defer mailAdapter.Disconnect()

	// 拉取详情
	detail, err := mailAdapter.FetchEmailDetail(ctx, email.ProviderID)
	if err != nil || detail == nil {
		return false, err
	}

	changed := false
	if detail.HTMLBody != "" && detail.HTMLBody != email.HTMLBody {
		email.HTMLBody = detail.HTMLBody
		changed = true
	}
	if detail.TextBody != "" && detail.TextBody != email.TextBody {
		email.TextBody = detail.TextBody
		changed = true
	}
	if detail.Snippet != "" && detail.Snippet != email.Snippet {
		email.Snippet = detail.Snippet
		changed = true
	}
	if detail.HasAttachments != email.HasAttachments || detail.AttachmentsCount != email.AttachmentsCount {
		email.HasAttachments = detail.HasAttachments
		email.AttachmentsCount = detail.AttachmentsCount
		changed = true
	}

	if !changed {
		return false, nil
	}
	if err := s.emailRepo.Update(ctx, email); err != nil {
		return false, err
	}
	return true, nil
}
