package dto

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse 分页响应格式
type PaginatedResponse struct {
	Success bool        `json:"success"`
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
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error":   "验证失败",
		"details": errors,
	})
}

// TooManyRequestsResponse 429 错误响应
func TooManyRequestsResponse(c *gin.Context) {
	c.Header("Retry-After", "60")
	ErrorResponse(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
}
