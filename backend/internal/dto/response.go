package dto

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式
type Response struct {
	Success bool        `json:"success"`
	Code    int         `json:"code,omitempty"` // 错误码
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse 分页响应格式
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code,omitempty"` // 错误码
	Data    interface{} `json:"data"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（带消息）
func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// PaginatedSuccessResponse 分页成功响应
func PaginatedSuccessResponse(c *gin.Context, data interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error:   message,
	})
}

// BadRequestResponse 400 错误响应
func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

// UnauthorizedResponse 401 错误响应
func UnauthorizedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "未授权访问"
	}
	ErrorResponse(c, http.StatusUnauthorized, message)
}

// ForbiddenResponse 403 错误响应
func ForbiddenResponse(c *gin.Context, message string) {
	if message == "" {
		message = "禁止访问"
	}
	ErrorResponse(c, http.StatusForbidden, message)
}

// NotFoundResponse 404 错误响应
func NotFoundResponse(c *gin.Context, message string) {
	if message == "" {
		message = "资源不存在"
	}
	ErrorResponse(c, http.StatusNotFound, message)
}

// InternalServerErrorResponse 500 错误响应
func InternalServerErrorResponse(c *gin.Context, message string) {
	if message == "" {
		message = "服务器内部错误"
	}
	ErrorResponse(c, http.StatusInternalServerError, message)
}

// ValidationErrorResponse 验证错误响应
func ValidationErrorResponse(c *gin.Context, errors map[string]string) {
	apiErr := NewAPIErrorWithDetails(
		ErrValidationFailed,
		"数据验证失败",
		map[string]interface{}{"fields": errors},
	)
	ErrorResponseWithDetails(c, http.StatusBadRequest, apiErr)
}

// TooManyRequestsResponse 429 错误响应
func TooManyRequestsResponse(c *gin.Context) {
	c.Header("Retry-After", "60")
	ErrorResponse(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
}

// ErrorResponseWithCode 带错误码的错误响应
func ErrorResponseWithCode(c *gin.Context, statusCode int, code ErrorCode, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Code:    int(code),
		Error:   message,
	})
}

// ErrorResponseWithDetails 带详情的错误响应
func ErrorResponseWithDetails(c *gin.Context, statusCode int, apiErr *APIError) {
	response := gin.H{
		"success": false,
		"code":    apiErr.Code,
		"error":   apiErr.Message,
	}

	if len(apiErr.Details) > 0 {
		response["details"] = apiErr.Details
	}

	c.JSON(statusCode, response)
}

// HandleServiceError 处理 Service 层错误
func HandleServiceError(c *gin.Context, err error) {
	if apiErr, ok := AsAPIError(err); ok {
		// 业务错误 - 使用错误码映射到 HTTP 状态码
		statusCode := GetHTTPStatusFromErrorCode(apiErr.Code)
		ErrorResponseWithDetails(c, statusCode, apiErr)
	} else {
		// 系统错误 - 返回 500
		InternalServerErrorResponse(c, "服务器内部错误")
	}
}

// GetHTTPStatusFromErrorCode 根据错误码获取 HTTP 状态码
func GetHTTPStatusFromErrorCode(code ErrorCode) int {
	switch code {
	// 特殊处理：账户已存在 -> 409 Conflict
	case ErrAccountExists:
		return http.StatusConflict
	// 特殊处理：资源已存在 -> 409 Conflict
	case ErrDuplicateResource:
		return http.StatusConflict
	default:
		// 按范围映射
		switch {
		case code >= 1100 && code < 1200:
			// 认证错误 -> 401
			return http.StatusUnauthorized
		case code == ErrAccountNotFound || code == ErrRuleNotFound || code == ErrWebhookNotFound || code == ErrEmailNotFound || code == ErrSettingNotFound || code == ErrResourceNotFound:
			// 资源不存在 -> 404
			return http.StatusNotFound
		case code >= 1000 && code < 2000:
			// 请求错误 -> 400
			return http.StatusBadRequest
		case code >= 2000 && code < 3000:
			// 账户相关错误（除了 NotFound）-> 400
			return http.StatusBadRequest
		case code >= 9000:
			// 系统错误 -> 500
			return http.StatusInternalServerError
		default:
			// 其他业务错误 -> 400
			return http.StatusBadRequest
		}
	}
}

// BadRequestResponseWithCode 400 错误响应（带错误码）
func BadRequestResponseWithCode(c *gin.Context, code ErrorCode, message string) {
	ErrorResponseWithCode(c, http.StatusBadRequest, code, message)
}

// UnauthorizedResponseWithCode 401 错误响应（带错误码）
func UnauthorizedResponseWithCode(c *gin.Context, code ErrorCode) {
	ErrorResponseWithCode(c, http.StatusUnauthorized, code, code.GetMessage())
}

// NotFoundResponseWithCode 404 错误响应（带错误码）
func NotFoundResponseWithCode(c *gin.Context, code ErrorCode) {
	ErrorResponseWithCode(c, http.StatusNotFound, code, code.GetMessage())
}

// NotImplementedResponse 501 错误响应（功能未实现）
func NotImplementedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "功能尚未实现"
	}
	ErrorResponse(c, http.StatusNotImplemented, message)
}
