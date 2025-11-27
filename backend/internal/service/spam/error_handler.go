package spam

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// 定义垃圾邮件检测相关的错误类型
var (
	// 外部服务错误
	ErrRBLServiceUnavailable   = errors.New("RBL service is unavailable")
	ErrSURBLServiceUnavailable = errors.New("SURBL service is unavailable")
	ErrRBLTimeout              = errors.New("RBL query timeout")
	ErrSURBLTimeout            = errors.New("SURBL query timeout")
	ErrNetworkError            = errors.New("network connection error")

	// 数据验证错误
	ErrInvalidEmailFormat  = errors.New("invalid email address format")
	ErrInvalidDomainFormat = errors.New("invalid domain format")
	ErrInvalidRulePattern  = errors.New("invalid rule pattern syntax")
	ErrInvalidScoreRange   = errors.New("score must be between 0 and 100")
	ErrEmptyInput          = errors.New("input cannot be empty")

	// 系统错误
	ErrDatabaseConnection      = errors.New("database connection failed")
	ErrCacheServiceUnavailable = errors.New("cache service is unavailable")
	ErrInsufficientMemory      = errors.New("insufficient memory")
	ErrModelNotTrained         = errors.New("bayesian model not trained")

	// 业务逻辑错误
	ErrWhitelistEntryExists    = errors.New("whitelist entry already exists")
	ErrBlacklistEntryExists    = errors.New("blacklist entry already exists")
	ErrRuleNotFound            = errors.New("spam rule not found")
	ErrBuiltinRuleCannotDelete = errors.New("builtin rule cannot be deleted")
	ErrReputationNotFound      = errors.New("sender reputation not found")
)

// SpamError 垃圾邮件检测错误
type SpamError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Cause     error                  `json:"-"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// Error 实现 error 接口
func (e *SpamError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Unwrap
func (e *SpamError) Unwrap() error {
	return e.Cause
}

// NewSpamError 创建新的垃圾邮件检测错误
func NewSpamError(code, message string, cause error) *SpamError {
	return &SpamError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// WithContext 添加上下文信息
func (e *SpamError) WithContext(key string, value interface{}) *SpamError {
	e.Context[key] = value
	return e
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	retryConfig *RetryConfig
	errorLog    *ErrorLog
	mu          sync.RWMutex
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           // 最大重试次数
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子
	RetryableErrors []error       // 可重试的错误类型
}

// ErrorLog 错误日志
type ErrorLog struct {
	mu      sync.RWMutex
	entries []*ErrorLogEntry
	maxSize int
}

// ErrorLogEntry 错误日志条目
type ErrorLogEntry struct {
	Error     *SpamError `json:"error"`
	Operation string     `json:"operation"`
	Timestamp time.Time  `json:"timestamp"`
	Retried   bool       `json:"retried"`
	Recovered bool       `json:"recovered"`
}

// DefaultRetryConfig 返回默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []error{
			ErrRBLTimeout,
			ErrSURBLTimeout,
			ErrNetworkError,
			ErrDatabaseConnection,
			ErrCacheServiceUnavailable,
		},
	}
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler(config *RetryConfig) *ErrorHandler {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &ErrorHandler{
		retryConfig: config,
		errorLog: &ErrorLog{
			entries: make([]*ErrorLogEntry, 0),
			maxSize: 1000,
		},
	}
}

// ==================== 重试机制 ====================

// RetryableOperation 可重试的操作函数类型
type RetryableOperation func(ctx context.Context) (interface{}, error)

// ExecuteWithRetry 带重试机制执行操作
func (h *ErrorHandler) ExecuteWithRetry(
	ctx context.Context,
	operation string,
	fn RetryableOperation,
) (interface{}, error) {
	var lastErr error
	delay := h.retryConfig.InitialDelay

	for attempt := 0; attempt <= h.retryConfig.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 执行操作
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// 检查是否是可重试的错误
		if !h.isRetryable(err) {
			h.logError(operation, err, false, false)
			return nil, err
		}

		// 如果不是最后一次尝试，等待后重试
		if attempt < h.retryConfig.MaxRetries {
			h.logError(operation, err, true, false)

			// 等待延迟时间
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			// 计算下一次延迟（指数退避）
			delay = time.Duration(float64(delay) * h.retryConfig.BackoffFactor)
			if delay > h.retryConfig.MaxDelay {
				delay = h.retryConfig.MaxDelay
			}
		}
	}

	// 所有重试都失败
	h.logError(operation, lastErr, false, false)
	return nil, fmt.Errorf("operation %s failed after %d retries: %w", operation, h.retryConfig.MaxRetries, lastErr)
}

// isRetryable 检查错误是否可重试
func (h *ErrorHandler) isRetryable(err error) bool {
	for _, retryableErr := range h.retryConfig.RetryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	// 检查是否是 SpamError 类型
	var spamErr *SpamError
	if errors.As(err, &spamErr) {
		// 根据错误代码判断是否可重试
		switch spamErr.Code {
		case "NETWORK_ERROR", "TIMEOUT", "SERVICE_UNAVAILABLE":
			return true
		}
	}

	return false
}

// ==================== 错误日志 ====================

// logError 记录错误日志
func (h *ErrorHandler) logError(operation string, err error, retried, recovered bool) {
	h.errorLog.mu.Lock()
	defer h.errorLog.mu.Unlock()

	// 转换为 SpamError
	var spamErr *SpamError
	if !errors.As(err, &spamErr) {
		spamErr = NewSpamError("UNKNOWN", err.Error(), err)
	}

	entry := &ErrorLogEntry{
		Error:     spamErr,
		Operation: operation,
		Timestamp: time.Now(),
		Retried:   retried,
		Recovered: recovered,
	}

	h.errorLog.entries = append(h.errorLog.entries, entry)

	// 限制日志大小
	if len(h.errorLog.entries) > h.errorLog.maxSize {
		h.errorLog.entries = h.errorLog.entries[len(h.errorLog.entries)-h.errorLog.maxSize:]
	}
}

// GetRecentErrors 获取最近的错误日志
func (h *ErrorHandler) GetRecentErrors(count int) []*ErrorLogEntry {
	h.errorLog.mu.RLock()
	defer h.errorLog.mu.RUnlock()

	if count > len(h.errorLog.entries) {
		count = len(h.errorLog.entries)
	}

	result := make([]*ErrorLogEntry, count)
	copy(result, h.errorLog.entries[len(h.errorLog.entries)-count:])
	return result
}

// GetErrorStats 获取错误统计信息
func (h *ErrorHandler) GetErrorStats() *ErrorStats {
	h.errorLog.mu.RLock()
	defer h.errorLog.mu.RUnlock()

	stats := &ErrorStats{
		TotalErrors:    int64(len(h.errorLog.entries)),
		ErrorsByCode:   make(map[string]int64),
		ErrorsByOp:     make(map[string]int64),
		RetriedCount:   0,
		RecoveredCount: 0,
	}

	for _, entry := range h.errorLog.entries {
		stats.ErrorsByCode[entry.Error.Code]++
		stats.ErrorsByOp[entry.Operation]++
		if entry.Retried {
			stats.RetriedCount++
		}
		if entry.Recovered {
			stats.RecoveredCount++
		}
	}

	return stats
}

// ErrorStats 错误统计信息
type ErrorStats struct {
	TotalErrors    int64            `json:"total_errors"`
	ErrorsByCode   map[string]int64 `json:"errors_by_code"`
	ErrorsByOp     map[string]int64 `json:"errors_by_operation"`
	RetriedCount   int64            `json:"retried_count"`
	RecoveredCount int64            `json:"recovered_count"`
}

// ClearErrorLog 清除错误日志
func (h *ErrorHandler) ClearErrorLog() {
	h.errorLog.mu.Lock()
	defer h.errorLog.mu.Unlock()

	h.errorLog.entries = make([]*ErrorLogEntry, 0)
}

// ==================== 错误恢复 ====================

// RecoverFromPanic 从 panic 恢复
func (h *ErrorHandler) RecoverFromPanic(operation string) {
	if r := recover(); r != nil {
		var err error
		switch v := r.(type) {
		case error:
			err = v
		case string:
			err = errors.New(v)
		default:
			err = fmt.Errorf("unknown panic: %v", v)
		}

		spamErr := NewSpamError("PANIC", "recovered from panic", err).
			WithContext("operation", operation)

		h.logError(operation, spamErr, false, true)
	}
}

// SafeExecute 安全执行操作（带 panic 恢复）
func (h *ErrorHandler) SafeExecute(
	ctx context.Context,
	operation string,
	fn RetryableOperation,
) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			switch v := r.(type) {
			case error:
				panicErr = v
			case string:
				panicErr = errors.New(v)
			default:
				panicErr = fmt.Errorf("unknown panic: %v", v)
			}

			err = NewSpamError("PANIC", "operation panicked", panicErr).
				WithContext("operation", operation)

			h.logError(operation, err, false, true)
		}
	}()

	return fn(ctx)
}

// ==================== 错误包装 ====================

// WrapExternalServiceError 包装外部服务错误
func WrapExternalServiceError(service string, err error) *SpamError {
	code := fmt.Sprintf("%s_ERROR", service)
	message := fmt.Sprintf("%s service error", service)
	return NewSpamError(code, message, err).
		WithContext("service", service)
}

// WrapValidationError 包装验证错误
func WrapValidationError(field string, err error) *SpamError {
	return NewSpamError("VALIDATION_ERROR", fmt.Sprintf("validation failed for %s", field), err).
		WithContext("field", field)
}

// WrapDatabaseError 包装数据库错误
func WrapDatabaseError(operation string, err error) *SpamError {
	return NewSpamError("DATABASE_ERROR", fmt.Sprintf("database %s failed", operation), err).
		WithContext("operation", operation)
}

// WrapCacheError 包装缓存错误
func WrapCacheError(operation string, err error) *SpamError {
	return NewSpamError("CACHE_ERROR", fmt.Sprintf("cache %s failed", operation), err).
		WithContext("operation", operation)
}

// ==================== 错误检查辅助函数 ====================

// IsExternalServiceError 检查是否是外部服务错误
func IsExternalServiceError(err error) bool {
	return errors.Is(err, ErrRBLServiceUnavailable) ||
		errors.Is(err, ErrSURBLServiceUnavailable) ||
		errors.Is(err, ErrRBLTimeout) ||
		errors.Is(err, ErrSURBLTimeout) ||
		errors.Is(err, ErrNetworkError)
}

// IsValidationError 检查是否是验证错误
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidEmailFormat) ||
		errors.Is(err, ErrInvalidDomainFormat) ||
		errors.Is(err, ErrInvalidRulePattern) ||
		errors.Is(err, ErrInvalidScoreRange) ||
		errors.Is(err, ErrEmptyInput)
}

// IsSystemError 检查是否是系统错误
func IsSystemError(err error) bool {
	return errors.Is(err, ErrDatabaseConnection) ||
		errors.Is(err, ErrCacheServiceUnavailable) ||
		errors.Is(err, ErrInsufficientMemory)
}

// IsRetryableError 检查错误是否可重试
func IsRetryableError(err error) bool {
	return IsExternalServiceError(err) || errors.Is(err, ErrDatabaseConnection)
}
