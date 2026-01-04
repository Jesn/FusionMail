package webhook

import (
	"errors"
	"fmt"
)

// 预定义的 Webhook 错误
var (
	// ErrInvalidSecret Secret/签名验证失败
	ErrInvalidSecret = errors.New("webhook: invalid or missing secret")

	// ErrInvalidPayload Payload 格式错误
	ErrInvalidPayload = errors.New("webhook: invalid payload format")

	// ErrMissingField 缺少必填字段
	ErrMissingField = errors.New("webhook: missing required field")

	// ErrUnsupportedProvider 不支持的 provider 类型
	ErrUnsupportedProvider = errors.New("webhook: unsupported provider type")

	// ErrAccountNotFound 找不到对应的邮箱账户
	ErrAccountNotFound = errors.New("webhook: email account not found")

	// ErrProviderNotFound 找不到对应的 Provider 配置
	ErrProviderNotFound = errors.New("webhook: provider not found")

	// ErrStorageFailed 邮件存储失败
	ErrStorageFailed = errors.New("webhook: failed to store email")

	// ErrDuplicateEmail 重复邮件（非错误，用于标识）
	ErrDuplicateEmail = errors.New("webhook: duplicate email")

	// ErrWebhookDisabled Webhook 功能未启用
	ErrWebhookDisabled = errors.New("webhook: webhook is disabled for this account")

	// ErrRateLimited 请求频率超限
	ErrRateLimited = errors.New("webhook: rate limit exceeded")

	// ErrPayloadTooLarge Payload 过大
	ErrPayloadTooLarge = errors.New("webhook: payload too large")

	// ErrTimeout 处理超时
	ErrTimeout = errors.New("webhook: processing timeout")
)

// WebhookError 自定义 Webhook 错误类型
// 包含错误码和详细信息，便于 API 响应
type WebhookError struct {
	// Code 错误码
	Code string `json:"code"`

	// Message 错误消息
	Message string `json:"message"`

	// Field 相关字段（用于字段验证错误）
	Field string `json:"field,omitempty"`

	// Cause 原始错误
	Cause error `json:"-"`
}

// Error 实现 error 接口
func (e *WebhookError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("webhook error [%s]: %s (field: %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("webhook error [%s]: %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *WebhookError) Unwrap() error {
	return e.Cause
}

// 错误码常量
const (
	ErrCodeInvalidSecret       = "WEBHOOK_INVALID_SECRET"
	ErrCodeInvalidPayload      = "WEBHOOK_INVALID_PAYLOAD"
	ErrCodeMissingField        = "WEBHOOK_MISSING_FIELD"
	ErrCodeUnsupportedProvider = "WEBHOOK_UNSUPPORTED_PROVIDER"
	ErrCodeAccountNotFound     = "WEBHOOK_ACCOUNT_NOT_FOUND"
	ErrCodeProviderNotFound    = "WEBHOOK_PROVIDER_NOT_FOUND"
	ErrCodeStorageError        = "WEBHOOK_STORAGE_ERROR"
	ErrCodeDuplicate           = "WEBHOOK_DUPLICATE"
	ErrCodeDisabled            = "WEBHOOK_DISABLED"
	ErrCodeRateLimited         = "WEBHOOK_RATE_LIMITED"
	ErrCodePayloadTooLarge     = "WEBHOOK_PAYLOAD_TOO_LARGE"
	ErrCodeTimeout             = "WEBHOOK_TIMEOUT"
	ErrCodeInternalError       = "WEBHOOK_INTERNAL_ERROR"
)

// NewWebhookError 创建新的 Webhook 错误
func NewWebhookError(code, message string) *WebhookError {
	return &WebhookError{
		Code:    code,
		Message: message,
	}
}

// NewWebhookErrorWithField 创建带字段信息的 Webhook 错误
func NewWebhookErrorWithField(code, message, field string) *WebhookError {
	return &WebhookError{
		Code:    code,
		Message: message,
		Field:   field,
	}
}

// NewWebhookErrorWithCause 创建带原始错误的 Webhook 错误
func NewWebhookErrorWithCause(code, message string, cause error) *WebhookError {
	return &WebhookError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewInvalidSecretError 创建 Secret 验证失败错误
func NewInvalidSecretError() *WebhookError {
	return &WebhookError{
		Code:    ErrCodeInvalidSecret,
		Message: "Invalid or missing webhook secret",
		Cause:   ErrInvalidSecret,
	}
}

// NewInvalidPayloadError 创建 Payload 格式错误
func NewInvalidPayloadError(detail string) *WebhookError {
	return &WebhookError{
		Code:    ErrCodeInvalidPayload,
		Message: fmt.Sprintf("Invalid payload format: %s", detail),
		Cause:   ErrInvalidPayload,
	}
}

// NewMissingFieldError 创建缺少必填字段错误
func NewMissingFieldError(field string) *WebhookError {
	return &WebhookError{
		Code:    ErrCodeMissingField,
		Message: fmt.Sprintf("Missing required field: %s", field),
		Field:   field,
		Cause:   ErrMissingField,
	}
}

// NewUnsupportedProviderError 创建不支持的 provider 类型错误
func NewUnsupportedProviderError(providerType string) *WebhookError {
	return &WebhookError{
		Code:    ErrCodeUnsupportedProvider,
		Message: fmt.Sprintf("Unsupported provider type: %s", providerType),
		Cause:   ErrUnsupportedProvider,
	}
}

// NewAccountNotFoundError 创建账户未找到错误
func NewAccountNotFoundError(email string) *WebhookError {
	return &WebhookError{
		Code:    ErrCodeAccountNotFound,
		Message: fmt.Sprintf("Email account not found for: %s", email),
		Cause:   ErrAccountNotFound,
	}
}

// NewStorageError 创建存储失败错误
func NewStorageError(detail string, cause error) *WebhookError {
	return &WebhookError{
		Code:    ErrCodeStorageError,
		Message: fmt.Sprintf("Failed to store email: %s", detail),
		Cause:   cause,
	}
}

// NewDuplicateError 创建重复邮件错误（非真正错误）
func NewDuplicateError(providerID string) *WebhookError {
	return &WebhookError{
		Code:    ErrCodeDuplicate,
		Message: fmt.Sprintf("Email already exists: %s", providerID),
		Cause:   ErrDuplicateEmail,
	}
}

// IsInvalidSecret 检查是否为 Secret 验证失败错误
func IsInvalidSecret(err error) bool {
	return errors.Is(err, ErrInvalidSecret)
}

// IsInvalidPayload 检查是否为 Payload 格式错误
func IsInvalidPayload(err error) bool {
	return errors.Is(err, ErrInvalidPayload)
}

// IsMissingField 检查是否为缺少必填字段错误
func IsMissingField(err error) bool {
	return errors.Is(err, ErrMissingField)
}

// IsUnsupportedProvider 检查是否为不支持的 provider 类型错误
func IsUnsupportedProvider(err error) bool {
	return errors.Is(err, ErrUnsupportedProvider)
}

// IsAccountNotFound 检查是否为账户未找到错误
func IsAccountNotFound(err error) bool {
	return errors.Is(err, ErrAccountNotFound)
}

// IsDuplicateEmail 检查是否为重复邮件
func IsDuplicateEmail(err error) bool {
	return errors.Is(err, ErrDuplicateEmail)
}

// GetWebhookErrorCode 从错误中提取错误码
// 如果不是 WebhookError 类型，返回通用内部错误码
func GetWebhookErrorCode(err error) string {
	var webhookErr *WebhookError
	if errors.As(err, &webhookErr) {
		return webhookErr.Code
	}
	return ErrCodeInternalError
}

// GetHTTPStatusCode 根据错误类型返回对应的 HTTP 状态码
func GetHTTPStatusCode(err error) int {
	if err == nil {
		return 200
	}

	switch {
	case errors.Is(err, ErrInvalidSecret):
		return 401
	case errors.Is(err, ErrInvalidPayload), errors.Is(err, ErrMissingField):
		return 400
	case errors.Is(err, ErrUnsupportedProvider):
		return 400
	case errors.Is(err, ErrAccountNotFound), errors.Is(err, ErrProviderNotFound):
		return 404
	case errors.Is(err, ErrWebhookDisabled):
		return 403
	case errors.Is(err, ErrRateLimited):
		return 429
	case errors.Is(err, ErrPayloadTooLarge):
		return 413
	case errors.Is(err, ErrTimeout):
		return 504
	case errors.Is(err, ErrDuplicateEmail):
		return 200 // 重复邮件返回 200（幂等性）
	default:
		return 500
	}
}
