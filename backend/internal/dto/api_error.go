package dto

import "fmt"

// APIError 统一的 API 错误类型
type APIError struct {
	Code    ErrorCode              `json:"code"`              // 错误码
	Message string                 `json:"message"`           // 错误消息
	Details map[string]interface{} `json:"details,omitempty"` // 错误详情
}

// Error 实现 error 接口
func (e *APIError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("[%d] %s (details: %v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewAPIError 创建 API 错误（使用默认消息）
func NewAPIError(code ErrorCode) *APIError {
	return &APIError{
		Code:    code,
		Message: code.GetMessage(),
	}
}

// NewAPIErrorWithMessage 创建 API 错误（自定义消息）
func NewAPIErrorWithMessage(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// NewAPIErrorWithDetails 创建带详情的 API 错误
func NewAPIErrorWithDetails(code ErrorCode, message string, details map[string]interface{}) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WithDetails 添加错误详情
func (e *APIError) WithDetails(details map[string]interface{}) *APIError {
	e.Details = details
	return e
}

// WithDetail 添加单个错误详情
func (e *APIError) WithDetail(key string, value interface{}) *APIError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// IsAPIError 判断是否为 APIError
func IsAPIError(err error) bool {
	_, ok := err.(*APIError)
	return ok
}

// AsAPIError 将 error 转换为 APIError
func AsAPIError(err error) (*APIError, bool) {
	apiErr, ok := err.(*APIError)
	return apiErr, ok
}
