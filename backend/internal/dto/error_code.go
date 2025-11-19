package dto

// ErrorCode 错误码类型
type ErrorCode int

const (
	// ========== 通用错误 (1000-1099) ==========
	ErrInvalidRequest    ErrorCode = 1000 // 请求参数无效
	ErrValidationFailed  ErrorCode = 1001 // 数据验证失败
	ErrResourceNotFound  ErrorCode = 1002 // 资源不存在
	ErrDuplicateResource ErrorCode = 1003 // 资源已存在
	ErrOperationFailed   ErrorCode = 1004 // 操作失败

	// ========== 认证错误 (1100-1199) ==========
	ErrUnauthorized       ErrorCode = 1100 // 未授权
	ErrInvalidCredentials ErrorCode = 1101 // 凭证无效
	ErrTokenExpired       ErrorCode = 1102 // Token 过期
	ErrTokenInvalid       ErrorCode = 1103 // Token 无效
	ErrForbidden          ErrorCode = 1104 // 禁止访问

	// ========== 账户错误 (2000-2099) ==========
	ErrAccountNotFound ErrorCode = 2000 // 账户不存在
	ErrAccountDisabled ErrorCode = 2001 // 账户已禁用
	ErrAccountExists   ErrorCode = 2002 // 账户已存在
	ErrAccountInvalid  ErrorCode = 2003 // 账户配置无效

	// ========== 同步错误 (3000-3099) ==========
	ErrSyncFailed       ErrorCode = 3000 // 同步失败
	ErrConnectionFailed ErrorCode = 3001 // 连接失败
	ErrAuthFailed       ErrorCode = 3002 // 认证失败
	ErrProviderError    ErrorCode = 3003 // 邮箱服务商错误

	// ========== 规则错误 (4000-4099) ==========
	ErrRuleInvalid       ErrorCode = 4000 // 规则配置无效
	ErrRuleNotFound      ErrorCode = 4001 // 规则不存在
	ErrRuleConflict      ErrorCode = 4002 // 规则冲突
	ErrRuleExecuteFailed ErrorCode = 4003 // 规则执行失败

	// ========== Webhook 错误 (5000-5099) ==========
	ErrWebhookInvalid  ErrorCode = 5000 // Webhook 配置无效
	ErrWebhookNotFound ErrorCode = 5001 // Webhook 不存在
	ErrWebhookFailed   ErrorCode = 5002 // Webhook 调用失败

	// ========== 邮件错误 (6000-6099) ==========
	ErrEmailNotFound ErrorCode = 6000 // 邮件不存在
	ErrEmailInvalid  ErrorCode = 6001 // 邮件格式无效

	// ========== 设置错误 (7000-7099) ==========
	ErrSettingNotFound      ErrorCode = 7000 // 配置不存在
	ErrSettingInvalid       ErrorCode = 7001 // 配置无效
	ErrSettingForbidden     ErrorCode = 7002 // 无权限访问配置
	ErrSettingDecryptFailed ErrorCode = 7003 // 配置解密失败
	ErrSettingEncryptFailed ErrorCode = 7004 // 配置加密失败

	// ========== 系统错误 (9000-9999) ==========
	ErrInternalServer  ErrorCode = 9000 // 服务器内部错误
	ErrDatabaseError   ErrorCode = 9001 // 数据库错误
	ErrEncryptionError ErrorCode = 9002 // 加密错误
	ErrCacheError      ErrorCode = 9003 // 缓存错误
)

// ErrorCodeMessage 错误码对应的默认消息
var ErrorCodeMessage = map[ErrorCode]string{
	ErrInvalidRequest:    "请求参数无效",
	ErrValidationFailed:  "数据验证失败",
	ErrResourceNotFound:  "资源不存在",
	ErrDuplicateResource: "资源已存在",
	ErrOperationFailed:   "操作失败",

	ErrUnauthorized:       "未授权访问",
	ErrInvalidCredentials: "用户名或密码错误",
	ErrTokenExpired:       "登录已过期，请重新登录",
	ErrTokenInvalid:       "登录凭证无效",
	ErrForbidden:          "没有权限访问该资源",

	ErrAccountNotFound: "账户不存在",
	ErrAccountDisabled: "账户已被禁用",
	ErrAccountExists:   "账户已存在",
	ErrAccountInvalid:  "账户配置无效",

	ErrSyncFailed:       "邮件同步失败",
	ErrConnectionFailed: "连接邮箱服务器失败",
	ErrAuthFailed:       "邮箱认证失败",
	ErrProviderError:    "邮箱服务商返回错误",

	ErrRuleInvalid:       "规则配置无效",
	ErrRuleNotFound:      "规则不存在",
	ErrRuleConflict:      "规则存在冲突",
	ErrRuleExecuteFailed: "规则执行失败",

	ErrWebhookInvalid:  "Webhook 配置无效",
	ErrWebhookNotFound: "Webhook 不存在",
	ErrWebhookFailed:   "Webhook 调用失败",

	ErrEmailNotFound: "邮件不存在",
	ErrEmailInvalid:  "邮件格式无效",

	// 设置相关错误
	ErrSettingNotFound:      "配置项不存在",
	ErrSettingInvalid:       "配置值格式不正确",
	ErrSettingForbidden:     "您没有权限访问此配置",
	ErrSettingDecryptFailed: "配置数据解密失败，请联系管理员",
	ErrSettingEncryptFailed: "配置数据加密失败，请重试",

	ErrInternalServer:  "服务器内部错误",
	ErrDatabaseError:   "数据库操作失败",
	ErrEncryptionError: "数据加密失败",
	ErrCacheError:      "缓存操作失败",
}

// GetMessage 获取错误码对应的默认消息
func (e ErrorCode) GetMessage() string {
	if msg, ok := ErrorCodeMessage[e]; ok {
		return msg
	}
	return "未知错误"
}
