package adapter

// GraphQuickAdapter 扩展查询：
// 按过滤条件获取、分页获取、按文件夹获取、邮件计数、移至垃圾箱。
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

// EmailFilter 邮件过滤条件
type EmailFilter struct {
	Since      time.Time // 开始时间
	Until      time.Time // 结束时间
	IsRead     *bool     // 已读状态
	HasAttach  *bool     // 是否有附件
	FromEmail  string    // 发件人邮箱
	Subject    string    // 主题关键词
	Folder     string    // 文件夹名称
	Categories []string  // 分类标签
}

// FetchEmailsWithFilter 使用过滤条件获取邮件
func (a *GraphQuickAdapter) FetchEmailsWithFilter(ctx context.Context, filter *EmailFilter, limit int) ([]*Email, error) {
	// Debug 级别：频繁调用的操作
	// a.logger.Info("开始使用过滤条件获取邮件", ...)

	// 构建查询参数
	params := url.Values{}
	params.Set("$orderby", "receivedDateTime DESC")

	// 设置获取数量限制
	fetchLimit := 100
	if limit > 0 && limit < 100 {
		fetchLimit = limit
	}
	params.Set("$top", fmt.Sprintf("%d", fetchLimit))

	// 构建过滤条件
	var filters []string

	if filter != nil {
		// 时间范围过滤
		if !filter.Since.IsZero() {
			filters = append(filters, fmt.Sprintf("receivedDateTime ge %s", filter.Since.Format(time.RFC3339)))
		}
		if !filter.Until.IsZero() {
			filters = append(filters, fmt.Sprintf("receivedDateTime le %s", filter.Until.Format(time.RFC3339)))
		}

		// 已读状态过滤
		if filter.IsRead != nil {
			filters = append(filters, fmt.Sprintf("isRead eq %t", *filter.IsRead))
		}

		// 附件过滤
		if filter.HasAttach != nil {
			filters = append(filters, fmt.Sprintf("hasAttachments eq %t", *filter.HasAttach))
		}

		// 发件人过滤
		if filter.FromEmail != "" {
			filters = append(filters, fmt.Sprintf("from/emailAddress/address eq '%s'", filter.FromEmail))
		}

		// 主题关键词过滤
		if filter.Subject != "" {
			filters = append(filters, fmt.Sprintf("contains(subject,'%s')", filter.Subject))
		}

		// 分类过滤
		if len(filter.Categories) > 0 {
			categoryFilters := make([]string, len(filter.Categories))
			for i, cat := range filter.Categories {
				categoryFilters[i] = fmt.Sprintf("categories/any(c:c eq '%s')", cat)
			}
			filters = append(filters, "("+strings.Join(categoryFilters, " or ")+")")
		}
	}

	if len(filters) > 0 {
		params.Set("$filter", strings.Join(filters, " and "))
	}

	// 选择需要的字段
	params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/me/messages?%s", a.baseURL, params.Encode())

	// 发送请求
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch filtered messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("获取过滤邮件失败",
			"email", a.config.Email,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode filtered response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		emails = append(emails, email)
	}

	// Debug 级别：频繁调用的操作
	// a.logger.Info("成功获取过滤邮件", ...)

	return emails, nil
}

// FetchEmailsWithPagination 分页获取邮件
func (a *GraphQuickAdapter) FetchEmailsWithPagination(ctx context.Context, pageSize int, nextPageToken string) (*EmailPage, error) {
	// Debug 级别：频繁调用的操作
	// a.logger.Info("开始分页获取邮件", ...)

	var requestURL string

	if nextPageToken != "" {
		// 使用下一页链接
		requestURL = nextPageToken
	} else {
		// 构建首页请求
		params := url.Values{}
		params.Set("$orderby", "receivedDateTime DESC")

		if pageSize <= 0 || pageSize > 100 {
			pageSize = 50 // 默认页大小
		}
		params.Set("$top", fmt.Sprintf("%d", pageSize))

		// 选择需要的字段
		params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

		requestURL = fmt.Sprintf("%s/me/messages?%s", a.baseURL, params.Encode())
	}

	// 发送请求
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch paginated messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("分页获取邮件失败",
			"email", a.config.Email,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode paginated response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		emails = append(emails, email)
	}

	// 构建分页结果
	page := &EmailPage{
		Emails:        emails,
		NextPageToken: messageList.NextLink,
		HasNextPage:   messageList.NextLink != "",
		PageSize:      len(emails),
	}

	// Debug 级别：频繁调用的操作
	// a.logger.Info("成功分页获取邮件", ...)

	return page, nil
}

// EmailPage 分页邮件结果
type EmailPage struct {
	Emails        []*Email `json:"emails"`
	NextPageToken string   `json:"next_page_token"`
	HasNextPage   bool     `json:"has_next_page"`
	PageSize      int      `json:"page_size"`
}

// FetchEmailsByFolder 按文件夹获取邮件
func (a *GraphQuickAdapter) FetchEmailsByFolder(ctx context.Context, folderName string, limit int) ([]*Email, error) {
	// Debug 级别：频繁调用的操作
	// a.logger.Info("开始按文件夹获取邮件", ...)

	// 首先获取文件夹 ID
	folderID, err := a.getFolderID(ctx, folderName)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder ID: %w", err)
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("$orderby", "receivedDateTime DESC")

	fetchLimit := 100
	if limit > 0 && limit < 100 {
		fetchLimit = limit
	}
	params.Set("$top", fmt.Sprintf("%d", fetchLimit))

	// 选择需要的字段
	params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/me/mailFolders/%s/messages?%s", a.baseURL, folderID, params.Encode())

	// 发送请求
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch folder messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("按文件夹获取邮件失败",
			"email", a.config.Email,
			"folder", folderName,
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var messageList GraphMessageList
	if err := json.NewDecoder(resp.Body).Decode(&messageList); err != nil {
		return nil, fmt.Errorf("failed to decode folder response: %w", err)
	}

	// 转换为 Email 对象
	emails := make([]*Email, 0, len(messageList.Value))
	for _, msg := range messageList.Value {
		email := a.convertGraphMessageToEmail(&msg)
		email.SourceFolder = folderName // 设置源文件夹
		emails = append(emails, email)
	}

	// Debug 级别：频繁调用的操作
	// a.logger.Info("成功按文件夹获取邮件", ...)

	return emails, nil
}

// getFolderID 获取文件夹 ID
func (a *GraphQuickAdapter) getFolderID(ctx context.Context, folderName string) (string, error) {
	// 常见文件夹的映射
	wellKnownFolders := map[string]string{
		"inbox":   "inbox",
		"sent":    "sentitems",
		"drafts":  "drafts",
		"deleted": "deleteditems",
		"junk":    "junkemail",
		"outbox":  "outbox",
		"archive": "archive",
	}

	// 检查是否是已知文件夹
	if folderID, exists := wellKnownFolders[strings.ToLower(folderName)]; exists {
		return folderID, nil
	}

	// 搜索自定义文件夹
	requestURL := fmt.Sprintf("%s/me/mailFolders?$filter=displayName eq '%s'", a.baseURL, folderName)

	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to search folder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to find folder: %s", folderName)
	}

	var folderList struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&folderList); err != nil {
		return "", fmt.Errorf("failed to decode folder response: %w", err)
	}

	if len(folderList.Value) == 0 {
		return "", fmt.Errorf("folder not found: %s", folderName)
	}

	return folderList.Value[0].ID, nil
}

// GetEmailCount 获取邮件总数
func (a *GraphQuickAdapter) GetEmailCount(ctx context.Context) (int, error) {
	// Debug 级别：频繁调用的操作
	// a.logger.Info("开始获取邮件总数", ...)

	requestURL := fmt.Sprintf("%s/me/messages/$count", a.baseURL)

	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get email count: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read count response: %w", err)
	}

	count := 0
	if _, err := fmt.Sscanf(string(body), "%d", &count); err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	// Debug 级别：频繁调用的操作
	// a.logger.Info("成功获取邮件总数", ...)
	return count, nil
}

// MoveToTrash 将邮件移至垃圾箱（Deleted Items）
func (a *GraphQuickAdapter) MoveToTrash(ctx context.Context, providerID string) error {
	a.logger.Info("开始移动邮件到垃圾箱", "email", a.config.Email, "messageID", providerID)

	// 构建 DELETE 请求
	requestURL := fmt.Sprintf("%s/me/messages/%s", a.baseURL, providerID)

	resp, err := a.makeAuthenticatedRequest(ctx, "DELETE", requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	defer resp.Body.Close()

	// 处理响应
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		a.logger.Info("成功移动邮件到垃圾箱", "email", a.config.Email, "messageID", providerID)
		return nil
	}

	// 处理 404：邮件不存在，视为幂等成功
	if resp.StatusCode == http.StatusNotFound {
		a.logger.Warn("邮件不存在（已被删除）", "email", a.config.Email, "messageID", providerID)
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("Graph API returned status %d: %s", resp.StatusCode, string(body))
}
