package cloudmail

// CloudMailAdapter 邮件拉取与转换：
// 多账户邮件列表拉取、邮件格式转换、附件 MIME 推断、邮件详情获取。
// 自 adapter.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
)

// FetchEmails 拉取所有账户的邮件
func (a *CloudMailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	if len(a.accounts) == 0 {
		a.log.Warn("没有可用的账户")
		return []*adapter.Email{}, nil
	}

	allEmails := make([]*adapter.Email, 0)

	// 遍历所有账户拉取邮件
	for _, account := range a.accounts {
		emails, err := a.fetchEmailsForAccount(ctx, account, since, limit)
		if err != nil {
			a.log.Warn("拉取账户邮件失败: account=%s, err=%v", account.Email, err)
			continue
		}
		allEmails = append(allEmails, emails...)
	}

	a.log.Info("Cloud Mail 拉取完成: total=%d, accounts=%d", len(allEmails), len(a.accounts))
	return allEmails, nil
}

// fetchEmailsForAccount 拉取单个账户的邮件
func (a *CloudMailAdapter) fetchEmailsForAccount(ctx context.Context, account CloudMailAccount, since time.Time, limit int) ([]*adapter.Email, error) {
	a.log.Info("拉取账户邮件: account=%s (id=%d), since=%v, limit=%d", account.Email, account.AccountID, since, limit)

	// 构建请求 URL
	// API: /api/email/list?accountId=xxx&emailId=0&timeSort=0&size=30&type=0
	size := limit
	if size <= 0 {
		size = 50
	}
	url := fmt.Sprintf("%s/api/email/list?accountId=%d&emailId=0&timeSort=0&size=%d&type=0",
		a.config.BaseURL, account.AccountID, size)

	a.log.Info("请求 URL: %s", url)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置请求头（模拟浏览器请求）
	a.setCommonHeaders(req)

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	a.log.Info("响应状态码: %d", resp.StatusCode)

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, webapi.ErrAuthFailed
	}
	if resp.StatusCode != http.StatusOK {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
	}

	a.log.Info("响应体长度: %d bytes", len(body))

	// 解析 JSON
	var response EmailListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		a.log.Error("解析 JSON 失败: %v, body=%s", err, string(body[:min(500, len(body))]))
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	a.log.Info("API 响应: code=%d, message=%s, list_count=%d, total=%d",
		response.Code, response.Message, len(response.Data.List), response.Data.Total)

	if response.Code != 200 {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("API 返回错误: %s", response.Message), response.Code, true, nil)
	}

	// 转换邮件
	emails := make([]*adapter.Email, 0, len(response.Data.List))
	for _, item := range response.Data.List {
		email := a.convertToEmail(item, account.Email)

		// 过滤时间
		if !since.IsZero() && email.ReceivedAt.Before(since) {
			a.log.Debug("跳过旧邮件: id=%d, time=%v, since=%v", item.EmailID, email.ReceivedAt, since)
			continue
		}

		emails = append(emails, email)
	}

	a.log.Info("账户邮件拉取完成: account=%s, count=%d (原始=%d)", account.Email, len(emails), len(response.Data.List))
	return emails, nil
}

// convertToEmail 转换为标准邮件格式
func (a *CloudMailAdapter) convertToEmail(item EmailItem, targetAddress string) *adapter.Email {
	email := &adapter.Email{
		ProviderID:  strconv.Itoa(item.EmailID),
		MessageID:   item.MessageID,
		Subject:     item.Subject,
		FromAddress: item.SendEmail,
		FromName:    item.Name,
		ToAddresses: []string{targetAddress},
		TextBody:    item.Text,
		HTMLBody:    item.Content,
	}

	// 解析收件人（如果有）
	if item.ToEmail != "" {
		email.ToAddresses = []string{item.ToEmail}
	}

	// 解析时间 (格式: "2025-11-17 05:28:25")
	if item.CreateTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", item.CreateTime); err == nil {
			email.ReceivedAt = t
			email.SentAt = t
		}
	}

	// 生成摘要
	email.Snippet = webapi.GenerateSnippet(email, 200)

	// 附件信息：元数据映射（FilePath 存入 ExternalRef 留作按需拉取）
	email.HasAttachments = len(item.AttList) > 0
	email.AttachmentsCount = len(item.AttList)
	for _, att := range item.AttList {
		email.Attachments = append(email.Attachments, adapter.Attachment{
			Filename:    att.FileName,
			ContentType: mimeByExtension(att.FileName),
			SizeBytes:   att.FileSize,
			ExternalRef: att.FilePath,
		})
	}

	return email
}

// mimeByExtension 按扩展名猜 MIME 类型（未知扩展名返回空串）。
func mimeByExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if m, ok := mimeByExt[ext]; ok {
		return m
	}
	return ""
}

var mimeByExt = map[string]string{
	".pdf": "application/pdf", ".zip": "application/zip", ".rar": "application/vnd.rar",
	".7z": "application/x-7z-compressed", ".gz": "application/gzip", ".tar": "application/x-tar",
	".doc": "application/msword", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls": "application/vnd.ms-excel", ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".svg": "image/svg+xml", ".txt": "text/plain", ".csv": "text/csv",
	".mp3": "audio/mpeg", ".mp4": "video/mp4", ".exe": "application/x-msdownload",
}

// FetchEmailDetail 获取邮件详情
func (a *CloudMailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*adapter.Email, error) {
	if !a.IsConnected() {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "未连接到服务", nil)
	}

	// 构建请求 URL
	url := fmt.Sprintf("%s/api/email/detail?emailId=%s", a.config.BaseURL, providerID)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "创建请求失败", err)
	}

	// 设置请求头（模拟浏览器请求）
	a.setCommonHeaders(req)

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeConnectionFailed, "请求失败", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, webapi.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("服务器返回错误: %d", resp.StatusCode), resp.StatusCode, true, nil)
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "读取响应失败", err)
	}

	var response EmailDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	if response.Code != 200 {
		return nil, webapi.NewWebAPIError(webapi.ErrCodeServerError, fmt.Sprintf("API 返回错误: %s", response.Message), response.Code, true, nil)
	}

	return a.convertToEmail(response.Data, response.Data.ToEmail), nil
}
