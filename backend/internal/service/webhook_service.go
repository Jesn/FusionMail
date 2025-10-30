package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// WebhookService Webhook 服务接口
type WebhookService interface {
	// CRUD 操作
	Create(ctx context.Context, webhook *model.Webhook) error
	GetByID(ctx context.Context, id int64) (*model.Webhook, error)
	Update(ctx context.Context, webhook *model.Webhook) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*model.Webhook, int64, error)

	// 状态管理
	Enable(ctx context.Context, id int64) error
	Disable(ctx context.Context, id int64) error

	// 事件触发
	TriggerEvent(ctx context.Context, event *WebhookEvent) error

	// 测试
	TestWebhook(ctx context.Context, id int64, testPayload map[string]interface{}) error
}

// WebhookEvent Webhook 事件
type WebhookEvent struct {
	Type      string                 `json:"type"`      // 事件类型
	Timestamp time.Time              `json:"timestamp"` // 事件时间
	Data      map[string]interface{} `json:"data"`      // 事件数据
}

// webhookService Webhook 服务实现
type webhookService struct {
	webhookRepo    repository.WebhookRepository
	webhookLogRepo repository.WebhookLogRepository
	httpClient     *http.Client
	logger         *logger.Logger
}

// NewWebhookService 创建 Webhook 服务实例
func NewWebhookService(
	webhookRepo repository.WebhookRepository,
	webhookLogRepo repository.WebhookLogRepository,
	logger *logger.Logger,
) WebhookService {
	return &webhookService{
		webhookRepo:    webhookRepo,
		webhookLogRepo: webhookLogRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Create 创建 Webhook
func (s *webhookService) Create(ctx context.Context, webhook *model.Webhook) error {
	// 验证 URL
	if webhook.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	// 验证 HTTP 方法
	if webhook.Method == "" {
		webhook.Method = "POST"
	}
	webhook.Method = strings.ToUpper(webhook.Method)
	if webhook.Method != "POST" && webhook.Method != "PUT" && webhook.Method != "PATCH" {
		return fmt.Errorf("unsupported HTTP method: %s", webhook.Method)
	}

	// 验证事件类型
	if webhook.Events == "" {
		return fmt.Errorf("webhook events are required")
	}

	// 设置默认值
	if webhook.MaxRetries == 0 {
		webhook.MaxRetries = 3
	}
	if webhook.RetryIntervals == "" {
		webhook.RetryIntervals = "[10, 30, 60]"
	}

	return s.webhookRepo.Create(ctx, webhook)
}

// GetByID 根据 ID 获取 Webhook
func (s *webhookService) GetByID(ctx context.Context, id int64) (*model.Webhook, error) {
	webhook, err := s.webhookRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if webhook == nil {
		return nil, fmt.Errorf("webhook not found")
	}
	return webhook, nil
}

// Update 更新 Webhook
func (s *webhookService) Update(ctx context.Context, webhook *model.Webhook) error {
	// 验证 Webhook 是否存在
	existing, err := s.webhookRepo.FindByID(ctx, webhook.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("webhook not found")
	}

	// 验证 URL
	if webhook.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	// 验证 HTTP 方法
	if webhook.Method == "" {
		webhook.Method = "POST"
	}
	webhook.Method = strings.ToUpper(webhook.Method)

	return s.webhookRepo.Update(ctx, webhook)
}

// Delete 删除 Webhook
func (s *webhookService) Delete(ctx context.Context, id int64) error {
	// 验证 Webhook 是否存在
	webhook, err := s.webhookRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if webhook == nil {
		return fmt.Errorf("webhook not found")
	}

	return s.webhookRepo.Delete(ctx, id)
}

// List 获取 Webhook 列表
func (s *webhookService) List(ctx context.Context, offset, limit int) ([]*model.Webhook, int64, error) {
	return s.webhookRepo.List(ctx, offset, limit)
}

// Enable 启用 Webhook
func (s *webhookService) Enable(ctx context.Context, id int64) error {
	return s.webhookRepo.Enable(ctx, id)
}

// Disable 禁用 Webhook
func (s *webhookService) Disable(ctx context.Context, id int64) error {
	return s.webhookRepo.Disable(ctx, id)
}

// TriggerEvent 触发事件
func (s *webhookService) TriggerEvent(ctx context.Context, event *WebhookEvent) error {
	// 获取启用的 Webhook 列表
	webhooks, err := s.webhookRepo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to get enabled webhooks: %w", err)
	}

	// 为每个匹配的 Webhook 发送请求
	for _, webhook := range webhooks {
		if s.shouldTriggerWebhook(webhook, event) {
			go s.sendWebhookRequest(context.Background(), webhook, event)
		}
	}

	return nil
}

// shouldTriggerWebhook 判断是否应该触发 Webhook
func (s *webhookService) shouldTriggerWebhook(webhook *model.Webhook, event *WebhookEvent) bool {
	// 解析监听的事件类型
	var events []string
	if err := json.Unmarshal([]byte(webhook.Events), &events); err != nil {
		s.logger.Error("Failed to parse webhook events", "webhook_id", webhook.ID, "error", err)
		return false
	}

	// 检查事件类型是否匹配
	for _, eventType := range events {
		if eventType == event.Type || eventType == "*" {
			return true
		}
	}

	return false
}

// sendWebhookRequest 发送 Webhook 请求
func (s *webhookService) sendWebhookRequest(ctx context.Context, webhook *model.Webhook, event *WebhookEvent) {
	startTime := time.Now()

	// 构建请求体
	payload := map[string]interface{}{
		"webhook_id": webhook.ID,
		"event":      event,
		"timestamp":  time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal webhook payload", "webhook_id", webhook.ID, "error", err)
		return
	}

	// 发送请求（带重试）
	success := s.sendWithRetry(ctx, webhook, payloadBytes, 0)

	// 更新统计信息
	if err := s.webhookRepo.UpdateCallStats(ctx, webhook.ID, success); err != nil {
		s.logger.Error("Failed to update webhook stats", "webhook_id", webhook.ID, "error", err)
	}

	duration := time.Since(startTime)
	s.logger.Info("Webhook request completed",
		"webhook_id", webhook.ID,
		"success", success,
		"duration", duration)
}

// sendWithRetry 发送请求（带重试）
func (s *webhookService) sendWithRetry(ctx context.Context, webhook *model.Webhook, payload []byte, retryCount int) bool {
	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		s.logWebhookCall(ctx, webhook, payload, 0, "", fmt.Sprintf("Failed to create request: %v", err), retryCount, false)
		return false
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "FusionMail-Webhook/1.0")

	// 添加自定义头
	if webhook.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(webhook.Headers), &headers); err == nil {
			for key, value := range headers {
				req.Header.Set(key, value)
			}
		}
	}

	// 发送请求
	startTime := time.Now()
	resp, err := s.httpClient.Do(req)
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		s.logWebhookCall(ctx, webhook, payload, 0, "", fmt.Sprintf("Request failed: %v", err), retryCount, false, responseTime)

		// 重试逻辑
		if webhook.RetryEnabled && retryCount < webhook.MaxRetries {
			s.scheduleRetry(ctx, webhook, payload, retryCount+1)
		}
		return false
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, _ := io.ReadAll(resp.Body)
	responseBodyStr := string(responseBody)

	// 判断是否成功（2xx 状态码）
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	// 记录日志
	s.logWebhookCall(ctx, webhook, payload, resp.StatusCode, responseBodyStr, "", retryCount, success, responseTime)

	// 如果失败且需要重试
	if !success && webhook.RetryEnabled && retryCount < webhook.MaxRetries {
		s.scheduleRetry(ctx, webhook, payload, retryCount+1)
	}

	return success
}

// scheduleRetry 安排重试
func (s *webhookService) scheduleRetry(ctx context.Context, webhook *model.Webhook, payload []byte, retryCount int) {
	// 解析重试间隔
	var intervals []int
	if err := json.Unmarshal([]byte(webhook.RetryIntervals), &intervals); err != nil {
		intervals = []int{10, 30, 60} // 默认间隔
	}

	// 获取重试间隔
	var interval int
	if retryCount-1 < len(intervals) {
		interval = intervals[retryCount-1]
	} else {
		interval = intervals[len(intervals)-1] // 使用最后一个间隔
	}

	// 延迟重试
	go func() {
		time.Sleep(time.Duration(interval) * time.Second)
		s.sendWithRetry(context.Background(), webhook, payload, retryCount)
	}()

	s.logger.Info("Scheduled webhook retry",
		"webhook_id", webhook.ID,
		"retry_count", retryCount,
		"delay_seconds", interval)
}

// logWebhookCall 记录 Webhook 调用日志
func (s *webhookService) logWebhookCall(ctx context.Context, webhook *model.Webhook, requestBody []byte,
	responseStatus int, responseBody, errorMessage string, retryCount int, success bool, responseTimeMs ...int) {

	// 构建请求头字符串
	var requestHeaders string
	if webhook.Headers != "" {
		requestHeaders = webhook.Headers
	}

	// 创建日志记录
	log := &model.WebhookLog{
		WebhookID:      webhook.ID,
		RequestURL:     webhook.URL,
		RequestMethod:  webhook.Method,
		RequestHeaders: requestHeaders,
		RequestBody:    string(requestBody),
		ResponseStatus: responseStatus,
		ResponseBody:   responseBody,
		Success:        success,
		ErrorMessage:   errorMessage,
		RetryCount:     retryCount,
	}

	// 设置响应时间
	if len(responseTimeMs) > 0 {
		log.ResponseTimeMs = responseTimeMs[0]
	}

	// 保存日志
	if err := s.webhookLogRepo.Create(ctx, log); err != nil {
		s.logger.Error("Failed to save webhook log", "webhook_id", webhook.ID, "error", err)
	}
}

// TestWebhook 测试 Webhook
func (s *webhookService) TestWebhook(ctx context.Context, id int64, testPayload map[string]interface{}) error {
	// 获取 Webhook
	webhook, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 构建测试事件
	event := &WebhookEvent{
		Type:      "test",
		Timestamp: time.Now(),
		Data:      testPayload,
	}

	// 发送测试请求
	payload := map[string]interface{}{
		"webhook_id": webhook.ID,
		"event":      event,
		"timestamp":  time.Now().Unix(),
		"test":       true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal test payload: %w", err)
	}

	// 发送请求（不重试）
	success := s.sendWithRetry(ctx, webhook, payloadBytes, 0)
	if !success {
		return fmt.Errorf("webhook test failed")
	}

	return nil
}
