package adapter

// GraphQuickAdapter 邮件获取与解析：
// 收件箱邮件列表、邮件详情、附件获取、Graph 消息到 Email 的转换。
// 核心流程必须与 backend/micro.py 保持一致。
// 自 graph_quick.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchEmails 拉取邮件列表
// 核心实现：必须与 backend/micro.py 的 print_inbox 函数保持一致
// 使用 /me/mailFolders/inbox/messages 端点获取收件箱邮件
func (a *GraphQuickAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	// Debug 级别：频繁调用的操作，生产环境不输出
	// a.logger.Info("开始获取邮件列表", ...)

	// 构建查询参数
	params := url.Values{}
	params.Set("$orderby", "receivedDateTime DESC")

	// 设置获取数量限制
	fetchLimit := 100 // 默认每次最多获取 100 封
	if limit > 0 && limit < 100 {
		fetchLimit = limit
	}
	params.Set("$top", fmt.Sprintf("%d", fetchLimit))

	// 添加时间过滤
	var filters []string
	if !since.IsZero() {
		filters = append(filters, fmt.Sprintf("receivedDateTime ge %s", since.Format(time.RFC3339)))
	}

	// 可以添加其他过滤条件
	if len(filters) > 0 {
		params.Set("$filter", strings.Join(filters, " and "))
	}

	// 选择需要的字段以优化性能
	params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

	// 构建请求 URL
	// 重要：必须使用 /me/mailFolders/inbox/messages 端点（与 micro.py 一致）
	// 不能使用 /me/messages 端点
	requestURL := fmt.Sprintf("%s/me/mailFolders/inbox/messages?%s", a.baseURL, params.Encode())

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("获取邮件列表失败",
			"email", a.config.Email,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		emails = append(emails, email)
	}

	// Debug 级别：频繁调用的操作，生产环境不输出
	// a.logger.Info("成功获取邮件列表", ...)

	return emails, nil
}

// FetchEmailDetail 获取邮件详情
// FetchEmailDetail 获取邮件详情
func (a *GraphQuickAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	// Debug 级别：频繁调用的操作，生产环境不输出
	// a.logger.Info("开始获取邮件详情", ...)

	// 验证 providerID
	if providerID == "" {
		return nil, fmt.Errorf("provider ID cannot be empty")
	}

	// 构建请求 URL，包含完整的字段选择
	params := url.Values{}
	params.Set("$select", "id,subject,body,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification,inReplyTo,references")

	requestURL := fmt.Sprintf("%s/me/messages/%s?%s", a.baseURL, providerID, params.Encode())

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("message not found: %s", providerID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("获取邮件详情失败",
			"email", a.config.Email,
			"provider_id", providerID,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var msg GraphMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为 Email 对象
	email := a.convertGraphMessageToEmail(&msg)

	// 获取附件信息
	if msg.HasAttachments {
		// Debug 级别：频繁调用的操作
		// a.logger.Info("获取邮件附件", ...)

		attachments, err := a.fetchAttachments(ctx, providerID)
		if err != nil {
			a.logger.Warn("获取附件失败",
				"email", a.config.Email,
				"provider_id", providerID,
				"error", err)
			// 不因为附件获取失败而失败整个请求
		} else {
			email.Attachments = attachments
			email.AttachmentsCount = len(attachments)
		}
	}

	// Debug 级别：频繁调用的操作
	// a.logger.Info("成功获取邮件详情", ...)

	return email, nil
}

// fetchAttachments 获取附件列表
// fetchAttachments 获取附件列表
func (a *GraphQuickAdapter) fetchAttachments(ctx context.Context, messageID string) ([]Attachment, error) {
	requestURL := fmt.Sprintf("%s/me/messages/%s/attachments", a.baseURL, messageID)

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attachments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	var attachmentList GraphAttachmentList
	if err := json.NewDecoder(resp.Body).Decode(&attachmentList); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	// 转换为 Attachment 对象
	attachments := make([]Attachment, 0, len(attachmentList.Value))
	for _, att := range attachmentList.Value {
		attachments = append(attachments, Attachment{
			Filename:    att.Name,
			ContentType: att.ContentType,
			SizeBytes:   att.Size,
			IsInline:    att.IsInline,
			ContentID:   att.ContentID,
		})
	}

	return attachments, nil
}

// convertGraphMessageToEmail 转换 Graph 消息为 Email 对象
func (a *GraphQuickAdapter) convertGraphMessageToEmail(msg *GraphMessage) *Email {
	email := &Email{
		ProviderID:     msg.ID,
		MessageID:      msg.InternetMessageID,
		Subject:        msg.Subject,
		Snippet:        msg.BodyPreview,
		ThreadID:       msg.ConversationID,
		HasAttachments: msg.HasAttachments,
		SourceLabels:   msg.Categories,
		SourceIsRead:   &msg.IsRead,
	}

	// 解析发件人
	email.FromAddress = msg.From.EmailAddress.Address
	email.FromName = msg.From.EmailAddress.Name

	// 解析收件人
	email.ToAddresses = make([]string, len(msg.ToRecipients))
	for i, recipient := range msg.ToRecipients {
		email.ToAddresses[i] = recipient.EmailAddress.Address
	}

	// 解析抄送
	email.CcAddresses = make([]string, len(msg.CcRecipients))
	for i, recipient := range msg.CcRecipients {
		email.CcAddresses[i] = recipient.EmailAddress.Address
	}

	// 解析密送
	email.BccAddresses = make([]string, len(msg.BccRecipients))
	for i, recipient := range msg.BccRecipients {
		email.BccAddresses[i] = recipient.EmailAddress.Address
	}

	// 解析回复地址
	if len(msg.ReplyTo) > 0 {
		email.ReplyTo = msg.ReplyTo[0].EmailAddress.Address
	}

	// 解析时间
	if sentTime, err := time.Parse(time.RFC3339, msg.SentDateTime); err == nil {
		email.SentAt = sentTime
	}
	if receivedTime, err := time.Parse(time.RFC3339, msg.ReceivedDateTime); err == nil {
		email.ReceivedAt = receivedTime
	}

	// 解析邮件正文
	if msg.Body.ContentType == "html" {
		email.HTMLBody = msg.Body.Content
	} else {
		email.TextBody = msg.Body.Content
	}

	return email
}
