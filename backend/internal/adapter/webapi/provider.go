package webapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
)

// ============================================
// WebAPIProvider 接口定义
// ============================================

// WebAPIProvider Web API 邮箱服务提供商接口
// 继承 MailProvider 接口，并添加 WebAPI 特有的方法
// 设计原则：接口极简，认证和模式处理内聚到各 Adapter 实现中
type WebAPIProvider interface {
	adapter.MailProvider

	// GetServiceName 获取服务名称
	// 返回服务的唯一标识，如 "cloudflare_temp_email", "cloud_mail"
	GetServiceName() string

	// GetSyncCheckpoint 获取同步检查点
	// 用于增量同步，返回上次同步的位置信息
	GetSyncCheckpoint() *SyncCheckpoint

	// UpdateSyncCheckpoint 更新同步检查点
	// 同步完成后调用，保存当前同步位置
	UpdateSyncCheckpoint(checkpoint *SyncCheckpoint) error
}

// ============================================
// 同步检查点
// ============================================

// SyncCheckpoint 同步检查点，用于增量同步
type SyncCheckpoint struct {
	// 时间戳检查点
	LastSyncTime time.Time `json:"last_sync_time"` // 上次同步时间

	// ID 检查点（用于 ID 分页）
	LastEmailID string `json:"last_email_id,omitempty"` // 上次同步的最后一封邮件 ID

	// 游标检查点（用于游标分页）
	Cursor string `json:"cursor,omitempty"` // 分页游标

	// 统计信息
	TotalSynced int64 `json:"total_synced"` // 累计同步邮件数
	LastCount   int   `json:"last_count"`   // 上次同步数量

	// 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"` // 额外元数据
}

// NewSyncCheckpoint 创建新的同步检查点
func NewSyncCheckpoint() *SyncCheckpoint {
	return &SyncCheckpoint{
		LastSyncTime: time.Time{},
		Metadata:     make(map[string]interface{}),
	}
}

// IsEmpty 检查检查点是否为空（首次同步）
func (c *SyncCheckpoint) IsEmpty() bool {
	return c.LastSyncTime.IsZero() && c.LastEmailID == "" && c.Cursor == ""
}

// Update 更新检查点
func (c *SyncCheckpoint) Update(syncTime time.Time, count int) {
	c.LastSyncTime = syncTime
	c.LastCount = count
	c.TotalSynced += int64(count)
}

// ============================================
// WebAPI 邮件结构（带目标地址）
// ============================================

// WebAPIEmail WebAPI 返回的邮件结构
// 包含 TargetAddress 字段，用于 Admin 模式下的邮件分发
type WebAPIEmail struct {
	*adapter.Email

	// 目标邮箱地址（用于 Admin 模式分发）
	// Single 模式：从配置中获取
	// Admin 模式：从 API 响应中解析
	TargetAddress string `json:"target_address"`

	// 原始数据（用于调试）
	RawData map[string]interface{} `json:"raw_data,omitempty"`
}

// NewWebAPIEmail 创建 WebAPIEmail
func NewWebAPIEmail(email *adapter.Email, targetAddress string) *WebAPIEmail {
	return &WebAPIEmail{
		Email:         email,
		TargetAddress: targetAddress,
	}
}

// ToEmail 转换为标准 Email（丢弃 TargetAddress）
func (e *WebAPIEmail) ToEmail() *adapter.Email {
	return e.Email
}

// ============================================
// 错误类型定义
// ============================================

// WebAPIError WebAPI 错误类型
type WebAPIError struct {
	Code       string // 错误代码
	Message    string // 错误消息
	StatusCode int    // HTTP 状态码
	Retryable  bool   // 是否可重试
	Cause      error  // 原始错误
}

// Error 实现 error 接口
func (e *WebAPIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap
func (e *WebAPIError) Unwrap() error {
	return e.Cause
}

// 预定义错误代码
const (
	ErrCodeAuthFailed       = "AUTH_FAILED"       // 认证失败
	ErrCodeConnectionFailed = "CONNECTION_FAILED" // 连接失败
	ErrCodeRateLimited      = "RATE_LIMITED"      // 速率限制
	ErrCodeInvalidResponse  = "INVALID_RESPONSE"  // 响应格式无效
	ErrCodeNotFound         = "NOT_FOUND"         // 资源不存在
	ErrCodeServerError      = "SERVER_ERROR"      // 服务器错误
	ErrCodeTimeout          = "TIMEOUT"           // 请求超时
	ErrCodeConfigError      = "CONFIG_ERROR"      // 配置错误
	ErrCodeParseError       = "PARSE_ERROR"       // 解析错误
)

// 预定义错误
var (
	ErrAuthFailed       = &WebAPIError{Code: ErrCodeAuthFailed, Message: "认证失败", Retryable: false}
	ErrConnectionFailed = &WebAPIError{Code: ErrCodeConnectionFailed, Message: "连接失败", Retryable: true}
	ErrRateLimited      = &WebAPIError{Code: ErrCodeRateLimited, Message: "请求过于频繁", Retryable: true}
	ErrInvalidResponse  = &WebAPIError{Code: ErrCodeInvalidResponse, Message: "响应格式无效", Retryable: false}
	ErrNotFound         = &WebAPIError{Code: ErrCodeNotFound, Message: "资源不存在", Retryable: false}
	ErrServerError      = &WebAPIError{Code: ErrCodeServerError, Message: "服务器错误", Retryable: true}
	ErrTimeout          = &WebAPIError{Code: ErrCodeTimeout, Message: "请求超时", Retryable: true}
	ErrConfigError      = &WebAPIError{Code: ErrCodeConfigError, Message: "配置错误", Retryable: false}
	ErrParseError       = &WebAPIError{Code: ErrCodeParseError, Message: "解析错误", Retryable: false}
)

// NewWebAPIError 创建 WebAPI 错误
func NewWebAPIError(code, message string, statusCode int, retryable bool, cause error) *WebAPIError {
	return &WebAPIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Retryable:  retryable,
		Cause:      cause,
	}
}

// WrapError 包装错误
func WrapError(code, message string, cause error) *WebAPIError {
	return &WebAPIError{
		Code:      code,
		Message:   message,
		Retryable: false,
		Cause:     cause,
	}
}

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	var webErr *WebAPIError
	if errors.As(err, &webErr) {
		return webErr.Retryable
	}
	return false
}

// IsAuthError 检查是否为认证错误
func IsAuthError(err error) bool {
	var webErr *WebAPIError
	if errors.As(err, &webErr) {
		return webErr.Code == ErrCodeAuthFailed
	}
	return false
}

// ============================================
// 基础适配器实现
// ============================================

// BaseWebAPIAdapter WebAPI 适配器基类
// 提供通用功能，具体适配器继承此类
type BaseWebAPIAdapter struct {
	serviceName string
	checkpoint  *SyncCheckpoint
	connected   bool
}

// NewBaseWebAPIAdapter 创建基础适配器
func NewBaseWebAPIAdapter(serviceName string) *BaseWebAPIAdapter {
	return &BaseWebAPIAdapter{
		serviceName: serviceName,
		checkpoint:  NewSyncCheckpoint(),
		connected:   false,
	}
}

// GetServiceName 获取服务名称
func (b *BaseWebAPIAdapter) GetServiceName() string {
	return b.serviceName
}

// GetSyncCheckpoint 获取同步检查点
func (b *BaseWebAPIAdapter) GetSyncCheckpoint() *SyncCheckpoint {
	return b.checkpoint
}

// UpdateSyncCheckpoint 更新同步检查点
func (b *BaseWebAPIAdapter) UpdateSyncCheckpoint(checkpoint *SyncCheckpoint) error {
	if checkpoint == nil {
		return errors.New("checkpoint 不能为空")
	}
	b.checkpoint = checkpoint
	return nil
}

// GetProviderType 获取提供商类型
func (b *BaseWebAPIAdapter) GetProviderType() string {
	return b.serviceName
}

// GetProtocol 获取协议类型
func (b *BaseWebAPIAdapter) GetProtocol() string {
	return "webapi"
}

// IsConnected 检查是否已连接
func (b *BaseWebAPIAdapter) IsConnected() bool {
	return b.connected
}

// SetConnected 设置连接状态
func (b *BaseWebAPIAdapter) SetConnected(connected bool) {
	b.connected = connected
}

// ============================================
// HTTP 客户端配置
// ============================================

// HTTPClientConfig HTTP 客户端配置
type HTTPClientConfig struct {
	Timeout         time.Duration // 请求超时时间
	MaxRetries      int           // 最大重试次数
	RetryDelay      time.Duration // 重试延迟
	UserAgent       string        // User-Agent
	FollowRedirects bool          // 是否跟随重定向
}

// DefaultHTTPClientConfig 默认 HTTP 客户端配置
func DefaultHTTPClientConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      time.Second,
		UserAgent:       "FusionMail/1.0",
		FollowRedirects: true,
	}
}

// ============================================
// 同步结果
// ============================================

// SyncResult 同步结果
type SyncResult struct {
	Emails       []*WebAPIEmail // 同步的邮件列表
	TotalCount   int            // 本次同步数量
	HasMore      bool           // 是否还有更多
	NextCursor   string         // 下一页游标
	SyncTime     time.Time      // 同步时间
	ErrorCount   int            // 错误数量
	SkippedCount int            // 跳过数量
}

// NewSyncResult 创建同步结果
func NewSyncResult() *SyncResult {
	return &SyncResult{
		Emails:   make([]*WebAPIEmail, 0),
		SyncTime: time.Now(),
	}
}

// AddEmail 添加邮件
func (r *SyncResult) AddEmail(email *WebAPIEmail) {
	r.Emails = append(r.Emails, email)
	r.TotalCount++
}

// AddEmails 批量添加邮件
func (r *SyncResult) AddEmails(emails []*WebAPIEmail) {
	r.Emails = append(r.Emails, emails...)
	r.TotalCount += len(emails)
}

// ============================================
// 上下文键
// ============================================

type contextKey string

const (
	// ContextKeyRequestID 请求 ID
	ContextKeyRequestID contextKey = "request_id"
	// ContextKeyAccountUID 账户 UID
	ContextKeyAccountUID contextKey = "account_uid"
	// ContextKeyProviderID Provider ID
	ContextKeyProviderID contextKey = "provider_id"
)

// WithRequestID 设置请求 ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// GetRequestID 获取请求 ID
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(ContextKeyRequestID); v != nil {
		return v.(string)
	}
	return ""
}

// WithAccountUID 设置账户 UID
func WithAccountUID(ctx context.Context, accountUID string) context.Context {
	return context.WithValue(ctx, ContextKeyAccountUID, accountUID)
}

// GetAccountUID 获取账户 UID
func GetAccountUID(ctx context.Context) string {
	if v := ctx.Value(ContextKeyAccountUID); v != nil {
		return v.(string)
	}
	return ""
}
