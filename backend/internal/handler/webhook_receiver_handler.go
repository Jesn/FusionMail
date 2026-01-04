package handler

import (
	"net/http"
	"time"

	"fusionmail/internal/service"
	"fusionmail/internal/webhook"
	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 模块日志记录器
var webhookReceiverLog = logger.NewWithModule("WebhookReceiver")

// WebhookReceiverHandler 通用 Webhook 接收处理器
// 处理来自各种邮件服务商的 webhook 推送请求
type WebhookReceiverHandler struct {
	// registry 适配器注册表
	registry *webhook.AdapterRegistry

	// service Webhook 接收服务
	service service.WebhookReceiverService

	// logger 日志记录器
	logger *logger.Logger
}

// NewWebhookReceiverHandler 创建 Webhook 接收处理器实例
func NewWebhookReceiverHandler(
	registry *webhook.AdapterRegistry,
	service service.WebhookReceiverService,
	logger *logger.Logger,
) *WebhookReceiverHandler {
	return &WebhookReceiverHandler{
		registry: registry,
		service:  service,
		logger:   logger,
	}
}

// HandleWebhook 通用 Webhook 处理入口
// POST /api/v1/webhook/receive/:provider_type
// 根据 provider_type 选择对应的适配器处理请求
func (h *WebhookReceiverHandler) HandleWebhook(c *gin.Context) {
	startTime := time.Now()
	requestID := h.generateRequestID()

	// 从 URL 参数获取 provider_type
	providerType := c.Param("provider_type")

	// 记录请求日志（脱敏处理）
	webhookReceiverLog.Info("[%s] 收到 webhook 请求: provider=%s, remote_addr=%s",
		requestID, providerType, c.ClientIP())

	// 1. 获取对应的适配器
	adapter, ok := h.registry.Get(providerType)
	if !ok {
		webhookReceiverLog.Warn("[%s] 不支持的 provider 类型: %s", requestID, providerType)
		h.respondError(c, webhook.NewUnsupportedProviderError(providerType))
		return
	}

	// 2. 解析 payload（先解析以获取收件人地址）
	email, err := adapter.ParsePayload(c)
	if err != nil {
		webhookReceiverLog.Error("[%s] 解析 payload 失败: provider=%s, err=%v",
			requestID, providerType, err)
		h.respondError(c, webhook.NewInvalidPayloadError(err.Error()))
		return
	}

	// 记录邮件基本信息（脱敏）
	webhookReceiverLog.Debug("[%s] 解析邮件: to=%s, subject=%s",
		requestID, h.maskEmail(email.To), webhook.TruncateString(email.Subject, 30))

	// 3. 获取 webhook secret（从账户配置中获取）
	secret, err := h.service.GetWebhookSecret(c.Request.Context(), providerType, email.To)
	if err != nil {
		// 如果找不到账户，记录警告但继续处理（可能是新账户）
		webhookReceiverLog.Warn("[%s] 获取 webhook secret 失败: to=%s, err=%v",
			requestID, h.maskEmail(email.To), err)
	}

	// 4. 验证请求签名/Secret
	// 需要重新绑定 body，因为 ParsePayload 已经消费了 body
	// 注意：某些适配器可能在 ParsePayload 中已经验证了签名
	if secret != "" {
		// 从 header 中获取签名进行验证
		headerSecret := c.GetHeader(adapter.GetSignatureHeader())
		if headerSecret != secret {
			webhookReceiverLog.Warn("[%s] Secret 验证失败: provider=%s, to=%s",
				requestID, providerType, h.maskEmail(email.To))
			h.respondError(c, webhook.NewInvalidSecretError())
			return
		}
	}

	// 5. 调用服务处理邮件
	result, err := h.service.ProcessEmail(c.Request.Context(), providerType, email)
	if err != nil {
		webhookReceiverLog.Error("[%s] 处理邮件失败: provider=%s, to=%s, err=%v",
			requestID, providerType, h.maskEmail(email.To), err)
		h.respondError(c, err)
		return
	}

	// 6. 记录处理结果
	duration := time.Since(startTime)
	if result.Duplicate {
		webhookReceiverLog.Info("[%s] 邮件已存在（重复）: provider=%s, account_uid=%s, duration=%v",
			requestID, providerType, result.AccountUID, duration)
	} else {
		webhookReceiverLog.Info("[%s] 邮件处理成功: provider=%s, email_id=%d, account_uid=%s, duration=%v",
			requestID, providerType, result.EmailID, result.AccountUID, duration)
	}

	// 7. 返回成功响应
	h.respondSuccess(c, result)
}

// GetSupportedProviders 获取支持的 provider 类型列表
// GET /api/v1/webhook/receive/providers
func (h *WebhookReceiverHandler) GetSupportedProviders(c *gin.Context) {
	// 获取所有已注册适配器的详细信息
	infos := h.registry.ListInfo()

	// 构建响应
	providers := make([]map[string]interface{}, 0, len(infos))
	for _, info := range infos {
		providers = append(providers, map[string]interface{}{
			"type":             info.ProviderType,
			"signature_header": info.SignatureHeader,
			"webhook_url":      "/api/v1/webhook/receive/" + info.ProviderType,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"providers": providers,
		"count":     len(providers),
	})
}

// GetWebhookInfo 获取 Webhook 配置信息
// GET /api/v1/webhook/receive/info/:provider_type
func (h *WebhookReceiverHandler) GetWebhookInfo(c *gin.Context) {
	providerType := c.Param("provider_type")

	// 检查适配器是否存在
	adapter, ok := h.registry.Get(providerType)
	if !ok {
		h.respondError(c, webhook.NewUnsupportedProviderError(providerType))
		return
	}

	// 返回配置信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"info": map[string]interface{}{
			"provider_type":    adapter.GetProviderType(),
			"signature_header": adapter.GetSignatureHeader(),
			"webhook_url":      "/api/v1/webhook/receive/" + providerType,
			"description":      h.getProviderDescription(providerType),
		},
	})
}

// respondSuccess 返回成功响应
func (h *WebhookReceiverHandler) respondSuccess(c *gin.Context, result *webhook.WebhookResult) {
	c.JSON(http.StatusOK, gin.H{
		"success":     result.Success,
		"message":     result.Message,
		"email_id":    result.EmailID,
		"duplicate":   result.Duplicate,
		"account_uid": result.AccountUID,
	})
}

// respondError 返回错误响应
func (h *WebhookReceiverHandler) respondError(c *gin.Context, err error) {
	statusCode := webhook.GetHTTPStatusCode(err)
	errorCode := webhook.GetWebhookErrorCode(err)

	// 获取错误消息
	message := "Internal server error"
	if webhookErr, ok := err.(*webhook.WebhookError); ok {
		message = webhookErr.Message
	} else if err != nil {
		message = err.Error()
	}

	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   errorCode,
		"message": message,
	})
}

// generateRequestID 生成请求追踪 ID
func (h *WebhookReceiverHandler) generateRequestID() string {
	return uuid.New().String()[:8]
}

// maskEmail 邮箱地址脱敏
// 例如: user@example.com -> u***@example.com
func (h *WebhookReceiverHandler) maskEmail(email string) string {
	if email == "" {
		return ""
	}

	// 查找 @ 符号位置
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}

	if atIndex <= 0 {
		return email
	}

	// 保留第一个字符，其余用 *** 替换
	localPart := email[:atIndex]
	domain := email[atIndex:]

	if len(localPart) <= 1 {
		return localPart + "***" + domain
	}

	return string(localPart[0]) + "***" + domain
}

// getProviderDescription 获取 provider 描述
func (h *WebhookReceiverHandler) getProviderDescription(providerType string) string {
	descriptions := map[string]string{
		"cloudflare_temp_email": "Cloudflare Temp Email webhook 接收器，支持实时邮件推送",
		"mailgun":               "Mailgun webhook 接收器，支持 HMAC-SHA256 签名验证",
		"sendgrid":              "SendGrid webhook 接收器，支持 ECDSA 签名验证",
		"postmark":              "Postmark webhook 接收器，支持 Token 验证",
	}

	if desc, ok := descriptions[providerType]; ok {
		return desc
	}
	return "通用邮件 webhook 接收器"
}
