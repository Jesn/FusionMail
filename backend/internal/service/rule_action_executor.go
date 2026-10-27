package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fusionmail/internal/model"
)

const (
	// ruleWebhookTimeout 单次 webhook 出站请求的超时。
	// 规则在同步路径上逐封邮件执行，无超时的出站请求会拖垮整个同步。
	ruleWebhookTimeout = 10 * time.Second
	// ruleWebhookMaxResponseBytes 读取响应体的上限。
	// 不读会阻止连接复用；不限制则异常端点可用超大响应耗尽内存。
	ruleWebhookMaxResponseBytes = 4 << 10
)

// parseEmailLabels 解析 Email.Labels 列。
//
// 该列的存储格式是 JSON 字符串数组，由同步路径统一写入：
//   - sync_processor.go 的 joinLabels（内部同样是 json.Marshal）
//   - sync_webapi.go 的 json.Marshal(email.SourceLabels)
//
// 空值或非法 JSON 一律视为「无标签」而不报错：规则动作不应因单封邮件的
// 脏数据中断整条同步链路。
func parseEmailLabels(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var labels []string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return nil
	}
	return labels
}

// encodeEmailLabels 把标签列表编码回 Email.Labels 的存储格式。
// 空列表编码为空字符串，与同步路径对「无标签」的表示保持一致。
func encodeEmailLabels(labels []string) (string, error) {
	if len(labels) == 0 {
		return "", nil
	}
	data, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// addLabel 实现 add_label 动作：把 action.Value 追加为邮件标签。
//
// 幂等：标签已存在时直接返回，不产生多余的写库
func (s *ruleService) addLabel(ctx context.Context, email *model.Email, label string) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return fmt.Errorf("add_label: empty label")
	}

	labels := parseEmailLabels(email.Labels)
	for _, existing := range labels {
		if existing == label {
			return nil
		}
	}

	encoded, err := encodeEmailLabels(append(labels, label))
	if err != nil {
		return fmt.Errorf("add_label: encode labels: %w", err)
	}
	email.Labels = encoded
	return s.emailRepo.Update(ctx, email)
}

// removeLabel 实现 remove_label 动作：从邮件标签中移除 action.Value。
//
// 幂等：标签不存在时直接返回，不产生多余的写库。
func (s *ruleService) removeLabel(ctx context.Context, email *model.Email, label string) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return fmt.Errorf("remove_label: empty label")
	}

	labels := parseEmailLabels(email.Labels)
	kept := make([]string, 0, len(labels))
	removed := false
	for _, existing := range labels {
		if existing == label {
			removed = true
			continue
		}
		kept = append(kept, existing)
	}
	if !removed {
		return nil
	}

	encoded, err := encodeEmailLabels(kept)
	if err != nil {
		return fmt.Errorf("remove_label: encode labels: %w", err)
	}
	email.Labels = encoded
	return s.emailRepo.Update(ctx, email)
}

// triggerWebhook 实现 webhook / trigger_webhook 动作：
// 向 action.Value 指定的 URL POST 一份邮件摘要。
//
// 前端 RuleForm 把该动作的 value 提示为「输入 Webhook URL」，
// 因此 value 是完整 URL，而不是 webhooks 表里的 ID 引用。
func (s *ruleService) triggerWebhook(ctx context.Context, email *model.Email, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("webhook: empty url")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("webhook: invalid url: %w", err)
	}
	// 只允许 http/https：阻止 file://、gopher:// 等协议被规则配置带进请求路径。
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("webhook: unsupported scheme %q (only http/https allowed)", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("webhook: missing host")
	}

	payload := map[string]any{
		"event":        "rule.matched",
		"email_id":     email.ID,
		"account_uid":  email.AccountUID,
		"message_id":   email.MessageID,
		"subject":      email.Subject,
		"from":         email.FromAddress,
		"from_name":    email.FromName,
		"to":           email.ToAddress,
		"folder":       email.Folder,
		"is_spam":      email.IsSpam,
		"received_at":  email.ReceivedAt,
		"labels":       parseEmailLabels(email.Labels),
		"triggered_at": time.Now(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "FusionMail-RuleEngine/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// Redacted() 抹掉 URL 里的 userinfo，避免凭据进日志。
		return fmt.Errorf("webhook: post %s: %w", parsed.Redacted(), err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, ruleWebhookMaxResponseBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: %s returned status %d", parsed.Redacted(), resp.StatusCode)
	}
	return nil
}
