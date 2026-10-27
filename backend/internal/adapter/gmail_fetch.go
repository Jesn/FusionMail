package adapter

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// FetchEmails 拉取邮件列表
func (a *GmailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	// 自动刷新 token（如果需要）
	if err := a.RefreshTokenIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	if a.service == nil {
		return nil, fmt.Errorf("not connected to Gmail API")
	}

	// 构建查询条件
	query := "in:inbox"
	if !since.IsZero() {
		// Gmail 使用 after: 语法进行时间过滤
		query += fmt.Sprintf(" after:%d", since.Unix())
	}

	// 设置最大结果数
	maxResults := int64(100)
	if limit > 0 && limit < 100 {
		maxResults = int64(limit)
	}

	// 调用 Gmail API 列出邮件
	listCall := a.service.Users.Messages.List("me").
		Q(query).
		MaxResults(maxResults)

	response, err := listCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	if len(response.Messages) == 0 {
		return []*Email{}, nil
	}

	// 获取邮件详情
	emails := make([]*Email, 0, len(response.Messages))
	for _, msg := range response.Messages {
		select {
		case <-ctx.Done():
			return emails, ctx.Err()
		default:
		}

		email, err := a.FetchEmailDetail(ctx, msg.Id)
		if err != nil {
			// 记录错误但继续处理其他邮件
			continue
		}

		emails = append(emails, email)
	}

	return emails, nil
}

// FetchEmailDetail 获取邮件详情
func (a *GmailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	// 自动刷新 token（如果需要）
	if err := a.RefreshTokenIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	if a.service == nil {
		return nil, fmt.Errorf("not connected to Gmail API")
	}

	// 获取邮件详情
	msg, err := a.service.Users.Messages.Get("me", providerID).
		Format("full").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// 解析邮件
	email := &Email{
		ProviderID:     msg.Id,
		ThreadID:       msg.ThreadId,
		SizeBytes:      msg.SizeEstimate,
		SourceLabels:   msg.LabelIds,
		HasAttachments: false,
	}

	// 解析邮件头
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "Message-ID":
			email.MessageID = header.Value
		case "Subject":
			email.Subject = header.Value
		case "From":
			email.FromAddress, email.FromName = parseEmailAddress(header.Value)
		case "To":
			email.ToAddresses = parseEmailAddresses(header.Value)
		case "Cc":
			email.CcAddresses = parseEmailAddresses(header.Value)
		case "Bcc":
			email.BccAddresses = parseEmailAddresses(header.Value)
		case "Reply-To":
			email.ReplyTo = header.Value
		case "In-Reply-To":
			email.InReplyTo = header.Value
		case "References":
			email.References = header.Value
		case "Date":
			if t, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
				email.SentAt = t
			}
		}
	}

	// 设置接收时间
	email.ReceivedAt = time.Unix(msg.InternalDate/1000, 0)
	if email.SentAt.IsZero() {
		email.SentAt = email.ReceivedAt
	}

	// 解析邮件正文和附件
	a.parseMessagePart(msg.Payload, email)

	// 生成摘要
	if email.Snippet == "" {
		email.Snippet = msg.Snippet
	}

	// 判断是否已读
	isRead := !contains(msg.LabelIds, "UNREAD")
	email.SourceIsRead = &isRead

	return email, nil
}

// parseMessagePart 解析邮件部分（递归处理多部分邮件）
func (a *GmailAdapter) parseMessagePart(part *gmail.MessagePart, email *Email) {
	// 处理邮件正文
	if part.MimeType == "text/plain" && part.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
		email.TextBody = string(data)
	} else if part.MimeType == "text/html" && part.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
		email.HTMLBody = string(data)
	}

	// 处理附件
	if part.Filename != "" {
		email.HasAttachments = true
		email.AttachmentsCount++

		attachment := Attachment{
			Filename:    part.Filename,
			ContentType: part.MimeType,
			SizeBytes:   part.Body.Size,
		}

		// 检查是否是内联附件
		for _, header := range part.Headers {
			if header.Name == "Content-ID" {
				attachment.IsInline = true
				attachment.ContentID = header.Value
				break
			}
		}

		email.Attachments = append(email.Attachments, attachment)
	}

	// 递归处理子部分
	for _, subPart := range part.Parts {
		a.parseMessagePart(subPart, email)
	}
}

// 辅助函数

// parseEmailAddress 解析邮件地址
func parseEmailAddress(addr string) (email, name string) {
	// 格式：Name <email@example.com> 或 email@example.com
	if strings.Contains(addr, "<") && strings.Contains(addr, ">") {
		parts := strings.Split(addr, "<")
		name = strings.TrimSpace(parts[0])
		email = strings.TrimSpace(strings.Trim(parts[1], ">"))
	} else {
		email = strings.TrimSpace(addr)
	}
	return
}

// parseEmailAddresses 解析多个邮件地址
func parseEmailAddresses(addrs string) []string {
	if addrs == "" {
		return nil
	}

	parts := strings.Split(addrs, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		email, _ := parseEmailAddress(part)
		if email != "" {
			result = append(result, email)
		}
	}

	return result
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ============================================================================
// BatchFetcher 接口实现 - 支持分批拉取邮件
// Requirements: 3.1
// ============================================================================

// FetchEmailsBatch 分批拉取邮件
// Requirements: 3.1 - 使用 pageToken 实现分页
func (a *GmailAdapter) FetchEmailsBatch(ctx context.Context, since time.Time, batchSize int, cursor string) ([]*Email, string, bool, error) {
	// 自动刷新 token（如果需要）
	if err := a.RefreshTokenIfNeeded(ctx); err != nil {
		return nil, "", false, fmt.Errorf("token refresh failed: %w", err)
	}

	if a.service == nil {
		return nil, "", false, fmt.Errorf("not connected to Gmail API")
	}

	// 构建查询条件
	query := "in:inbox"
	if !since.IsZero() {
		query += fmt.Sprintf(" after:%d", since.Unix())
	}

	// 设置批次大小
	maxResults := int64(batchSize)
	if maxResults <= 0 {
		maxResults = 100
	}
	if maxResults > 500 {
		maxResults = 500 // Gmail API 最大限制
	}

	// 调用 Gmail API 列出邮件
	listCall := a.service.Users.Messages.List("me").
		Q(query).
		MaxResults(maxResults)

	// 如果有游标（pageToken），使用它
	if cursor != "" {
		listCall = listCall.PageToken(cursor)
	}

	response, err := listCall.Do()
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to list messages: %w", err)
	}

	if len(response.Messages) == 0 {
		return []*Email{}, "", false, nil
	}

	// 获取邮件详情
	emails := make([]*Email, 0, len(response.Messages))
	for _, msg := range response.Messages {
		select {
		case <-ctx.Done():
			return emails, response.NextPageToken, response.NextPageToken != "", ctx.Err()
		default:
		}

		email, err := a.FetchEmailDetail(ctx, msg.Id)
		if err != nil {
			// 记录错误但继续处理其他邮件
			continue
		}

		emails = append(emails, email)
	}

	// 判断是否还有更多
	hasMore := response.NextPageToken != ""

	return emails, response.NextPageToken, hasMore, nil
}

// GetEstimatedCount 获取预估邮件数量
// Requirements: 2.1 - 提供预估总数
func (a *GmailAdapter) GetEstimatedCount(ctx context.Context, since time.Time) (int, error) {
	// 自动刷新 token（如果需要）
	if err := a.RefreshTokenIfNeeded(ctx); err != nil {
		return 0, fmt.Errorf("token refresh failed: %w", err)
	}

	if a.service == nil {
		return 0, fmt.Errorf("not connected to Gmail API")
	}

	// 构建查询条件
	query := "in:inbox"
	if !since.IsZero() {
		query += fmt.Sprintf(" after:%d", since.Unix())
	}

	// Gmail API 不直接提供计数，需要遍历获取
	// 使用小批量快速估算
	count := 0
	pageToken := ""

	for {
		listCall := a.service.Users.Messages.List("me").
			Q(query).
			MaxResults(500) // 使用最大值快速遍历

		if pageToken != "" {
			listCall = listCall.PageToken(pageToken)
		}

		response, err := listCall.Do()
		if err != nil {
			return count, fmt.Errorf("failed to list messages: %w", err)
		}

		count += len(response.Messages)

		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken

		// 检查 context 是否取消
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		default:
		}
	}

	return count, nil
}
