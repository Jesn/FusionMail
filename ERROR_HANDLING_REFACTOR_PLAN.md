# FusionMail 错误处理重构方案

## 📋 重构目标

将 FusionMail 的错误处理机制从当前的混乱状态重构为统一、规范、易维护的体系。

**核心目标**：
1. 建立统一的错误码体系
2. 统一 Handler 层错误响应格式
3. 规范 Service 层错误处理
4. 添加完善的错误日志
5. 提升错误的可追踪性和可维护性

---

## 🎯 重构策略

### 策略 1: 渐进式重构
- 不影响现有功能
- 逐步替换旧代码
- 保持向后兼容

### 策略 2: 分层实施
- 第一步：建立基础设施（错误码、错误类型）
- 第二步：重构 DTO 层（响应函数）
- 第三步：重构 Handler 层
- 第四步：重构 Service 层
- 第五步：添加中间件和日志

### 策略 3: 测试驱动
- 每个步骤都要测试
- 确保不破坏现有功能
- 逐步提升代码质量

---

## 📝 详细实施步骤

### 第一阶段：建立基础设施 (1-2 小时)

#### 步骤 1.1: 创建错误码定义文件

**文件**: `backend/internal/dto/error_code.go`

**任务**：
- 定义所有错误码常量
- 按模块分类（通用、认证、账户、同步、规则、Webhook）
- 每个错误码都有清晰的注释


**代码示例**：
```go
package dto

// ErrorCode 错误码类型
type ErrorCode int

const (
    // ========== 通用错误 (1000-1099) ==========
    ErrInvalidRequest     ErrorCode = 1000 // 请求参数无效
    ErrValidationFailed   ErrorCode = 1001 // 数据验证失败
    ErrResourceNotFound   ErrorCode = 1002 // 资源不存在
    ErrDuplicateResource  ErrorCode = 1003 // 资源已存在
    ErrOperationFailed    ErrorCode = 1004 // 操作失败
    
    // ========== 认证错误 (1100-1199) ==========
    ErrUnauthorized       ErrorCode = 1100 // 未授权
    ErrInvalidCredentials ErrorCode = 1101 // 凭证无效
    ErrTokenExpired       ErrorCode = 1102 // Token 过期
    ErrTokenInvalid       ErrorCode = 1103 // Token 无效
    ErrForbidden          ErrorCode = 1104 // 禁止访问
    
    // ========== 账户错误 (2000-2099) ==========
    ErrAccountNotFound    ErrorCode = 2000 // 账户不存在
    ErrAccountDisabled    ErrorCode = 2001 // 账户已禁用
    ErrAccountExists      ErrorCode = 2002 // 账户已存在
    ErrAccountInvalid     ErrorCode = 2003 // 账户配置无效
    
    // ========== 同步错误 (3000-3099) ==========
    ErrSyncFailed         ErrorCode = 3000 // 同步失败
    ErrConnectionFailed   ErrorCode = 3001 // 连接失败
    ErrAuthFailed         ErrorCode = 3002 // 认证失败
    ErrProviderError      ErrorCode = 3003 // 邮箱服务商错误
    
    // ========== 规则错误 (4000-4099) ==========
    ErrRuleInvalid        ErrorCode = 4000 // 规则配置无效
    ErrRuleNotFound       ErrorCode = 4001 // 规则不存在
    ErrRuleConflict       ErrorCode = 4002 // 规则冲突
    ErrRuleExecuteFailed  ErrorCode = 4003 // 规则执行失败
    
    // ========== Webhook 错误 (5000-5099) ==========
    ErrWebhookInvalid     ErrorCode = 5000 // Webhook 配置无效
    ErrWebhookNotFound    ErrorCode = 5001 // Webhook 不存在
    ErrWebhookFailed      ErrorCode = 5002 // Webhook 调用失败
    
    // ========== 邮件错误 (6000-6099) ==========
    ErrEmailNotFound      ErrorCode = 6000 // 邮件不存在
    ErrEmailInvalid       ErrorCode = 6001 // 邮件格式无效
    
    // ========== 系统错误 (9000-9999) ==========
    ErrInternalServer     ErrorCode = 9000 // 服务器内部错误
    ErrDatabaseError      ErrorCode = 9001 // 数据库错误
    ErrEncryptionError    ErrorCode = 9002 // 加密错误
    ErrCacheError         ErrorCode = 9003 // 缓存错误
)

// ErrorCodeMessage 错误码对应的默认消息
var ErrorCodeMessage = map[ErrorCode]string{
    ErrInvalidRequest:     "请求参数无效",
    ErrValidationFailed:   "数据验证失败",
    ErrResourceNotFound:   "资源不存在",
    ErrDuplicateResource:  "资源已存在",
    ErrOperationFailed:    "操作失败",
    
    ErrUnauthorized:       "未授权访问",
    ErrInvalidCredentials: "用户名或密码错误",
    ErrTokenExpired:       "登录已过期，请重新登录",
    ErrTokenInvalid:       "登录凭证无效",
    ErrForbidden:          "没有权限访问该资源",
    
    ErrAccountNotFound:    "账户不存在",
    ErrAccountDisabled:    "账户已被禁用",
    ErrAccountExists:      "账户已存在",
    ErrAccountInvalid:     "账户配置无效",
    
    ErrSyncFailed:         "邮件同步失败",
    ErrConnectionFailed:   "连接邮箱服务器失败",
    ErrAuthFailed:         "邮箱认证失败",
    ErrProviderError:      "邮箱服务商返回错误",
    
    ErrRuleInvalid:        "规则配置无效",
    ErrRuleNotFound:       "规则不存在",
    ErrRuleConflict:       "规则存在冲突",
    ErrRuleExecuteFailed:  "规则执行失败",
    
    ErrWebhookInvalid:     "Webhook 配置无效",
    ErrWebhookNotFound:    "Webhook 不存在",
    ErrWebhookFailed:      "Webhook 调用失败",
    
    ErrEmailNotFound:      "邮件不存在",
    ErrEmailInvalid:       "邮件格式无效",
    
    ErrInternalServer:     "服务器内部错误",
    ErrDatabaseError:      "数据库操作失败",
    ErrEncryptionError:    "数据加密失败",
    ErrCacheError:         "缓存操作失败",
}

// GetMessage 获取错误码对应的默认消息
func (e ErrorCode) GetMessage() string {
    if msg, ok := ErrorCodeMessage[e]; ok {
        return msg
    }
    return "未知错误"
}
```


#### 步骤 1.2: 创建 APIError 类型

**文件**: `backend/internal/dto/api_error.go`

**任务**：
- 定义 APIError 结构体
- 实现 error 接口
- 提供便捷的构造函数

**代码示例**：
```go
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
```


#### 步骤 1.3: 更新 Response 结构体

**文件**: `backend/internal/dto/response.go`

**任务**：
- 在 Response 中添加 Code 字段
- 保持向后兼容

**修改方案**：
```go
package dto

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// Response 统一响应格式
type Response struct {
    Success bool        `json:"success"`
    Code    int         `json:"code,omitempty"`    // 新增：错误码
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Message string      `json:"message,omitempty"`
}

// PaginatedResponse 分页响应格式
type PaginatedResponse struct {
    Success bool        `json:"success"`
    Code    int         `json:"code,omitempty"`    // 新增：错误码
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

// ErrorResponse 错误响应（保持向后兼容）
func ErrorResponse(c *gin.Context, statusCode int, message string) {
    c.JSON(statusCode, Response{
        Success: false,
        Error:   message,
    })
}

// ErrorResponseWithCode 带错误码的错误响应（新增）
func ErrorResponseWithCode(c *gin.Context, statusCode int, code ErrorCode, message string) {
    c.JSON(statusCode, Response{
        Success: false,
        Code:    int(code),
        Error:   message,
    })
}

// ErrorResponseWithDetails 带详情的错误响应（新增）
func ErrorResponseWithDetails(c *gin.Context, statusCode int, apiErr *APIError) {
    response := gin.H{
        "success": false,
        "code":    apiErr.Code,
        "error":   apiErr.Message,
    }
    
    if apiErr.Details != nil && len(apiErr.Details) > 0 {
        response["details"] = apiErr.Details
    }
    
    c.JSON(statusCode, response)
}

// HandleServiceError 处理 Service 层错误（新增）
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
    switch {
    case code >= 1100 && code < 1200:
        // 认证错误 -> 401
        return http.StatusUnauthorized
    case code >= 2000 && code < 3000:
        // 资源不存在 -> 404
        return http.StatusNotFound
    case code >= 1000 && code < 2000:
        // 请求错误 -> 400
        return http.StatusBadRequest
    case code >= 9000:
        // 系统错误 -> 500
        return http.StatusInternalServerError
    default:
        // 其他业务错误 -> 400
        return http.StatusBadRequest
    }
}

// BadRequestResponse 400 错误响应
func BadRequestResponse(c *gin.Context, message string) {
    ErrorResponse(c, http.StatusBadRequest, message)
}

// BadRequestResponseWithCode 400 错误响应（带错误码）
func BadRequestResponseWithCode(c *gin.Context, code ErrorCode, message string) {
    ErrorResponseWithCode(c, http.StatusBadRequest, code, message)
}

// UnauthorizedResponse 401 错误响应
func UnauthorizedResponse(c *gin.Context, message string) {
    if message == "" {
        message = "未授权访问"
    }
    ErrorResponse(c, http.StatusUnauthorized, message)
}

// UnauthorizedResponseWithCode 401 错误响应（带错误码）
func UnauthorizedResponseWithCode(c *gin.Context, code ErrorCode) {
    ErrorResponseWithCode(c, http.StatusUnauthorized, code, code.GetMessage())
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

// NotFoundResponseWithCode 404 错误响应（带错误码）
func NotFoundResponseWithCode(c *gin.Context, code ErrorCode) {
    ErrorResponseWithCode(c, http.StatusNotFound, code, code.GetMessage())
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
```


---

### 第二阶段：重构 Handler 层 (2-3 小时)

#### 步骤 2.1: 重构 account_handler.go

**修改策略**：
- 使用 `dto.HandleServiceError()` 统一处理错误
- 使用 `dto.SuccessResponse()` 返回成功响应
- 移除所有 `gin.H` 的使用

**修改示例**：

**修改前**：
```go
func (h *AccountHandler) GetByUID(c *gin.Context) {
    uid := c.Param("uid")
    
    account, err := h.accountService.GetByUID(c.Request.Context(), uid)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    account,
    })
}
```

**修改后**：
```go
func (h *AccountHandler) GetByUID(c *gin.Context) {
    uid := c.Param("uid")
    
    account, err := h.accountService.GetByUID(c.Request.Context(), uid)
    if err != nil {
        dto.HandleServiceError(c, err)
        return
    }
    
    dto.SuccessResponse(c, account)
}
```

**完整修改清单**：
- [ ] GetByUID
- [ ] Create
- [ ] List
- [ ] Update
- [ ] Delete
- [ ] TestConnection
- [ ] SyncAccount
- [ ] DisableAccount
- [ ] EnableAccount
- [ ] ClearSyncError
- [ ] BatchImport


#### 步骤 2.2: 重构 email_handler.go

**修改策略**：同 account_handler.go

**关键修改点**：
```go
// 修改前
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
        "success": false,
        "error":   err.Error(),
    })
    return
}

// 修改后
if err != nil {
    dto.HandleServiceError(c, err)
    return
}
```

**完整修改清单**：
- [ ] GetEmailList
- [ ] GetEmailByID
- [ ] SearchEmails
- [ ] MarkAsRead
- [ ] MarkAsUnread
- [ ] MarkAllAsRead
- [ ] ToggleStar
- [ ] ArchiveEmail
- [ ] DeleteEmail
- [ ] GetUnreadCount
- [ ] GetAccountStats

#### 步骤 2.3: 重构 rule_handler.go

**修改策略**：
- 已经使用 `response.Error()`，需要改为 `dto.HandleServiceError()`
- 统一使用 dto 包的函数

**修改示例**：
```go
// 修改前
response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())

// 修改后
dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
// 或者如果 Service 返回 APIError
dto.HandleServiceError(c, err)
```

**完整修改清单**：
- [ ] CreateRule
- [ ] GetRuleByID
- [ ] ListRules
- [ ] UpdateRule
- [ ] DeleteRule
- [ ] EnableRule
- [ ] DisableRule
- [ ] TestRule
- [ ] GetRuleExecutionLogs

#### 步骤 2.4: 重构 webhook_handler.go

**修改策略**：
- 已经使用 `dto.BadRequestResponse()`，保持不变
- 其他错误处理改为 `dto.HandleServiceError()`

**完整修改清单**：
- [ ] CreateWebhook
- [ ] GetWebhookByID
- [ ] ListWebhooks
- [ ] UpdateWebhook
- [ ] DeleteWebhook
- [ ] EnableWebhook
- [ ] DisableWebhook
- [ ] TestWebhook
- [ ] GetWebhookLogs

#### 步骤 2.5: 重构其他 Handler

**文件清单**：
- [ ] auth.go
- [ ] oauth2_handler.go
- [ ] system_handler.go
- [ ] attachment_handler.go


---

### 第三阶段：重构 Service 层 (3-4 小时)

#### 步骤 3.1: 重构 account_service.go

**修改策略**：
- 业务错误返回 `*dto.APIError`
- 系统错误继续使用 `fmt.Errorf` 包装
- 添加详细的错误信息

**修改示例**：

**修改前**：
```go
func (s *accountService) GetByUID(ctx context.Context, uid string) (*model.Account, error) {
    account, err := s.accountRepo.FindByUID(ctx, uid)
    if err != nil {
        return nil, fmt.Errorf("failed to get account: %w", err)
    }
    if account == nil {
        return nil, fmt.Errorf("account not found: %s", uid)
    }
    return account, nil
}
```

**修改后**：
```go
func (s *accountService) GetByUID(ctx context.Context, uid string) (*model.Account, error) {
    account, err := s.accountRepo.FindByUID(ctx, uid)
    if err != nil {
        // 数据库错误 - 系统错误
        logger.Error("database error when finding account",
            "uid", uid,
            "error", err.Error(),
        )
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    if account == nil {
        // 业务错误 - 返回 APIError
        return nil, dto.NewAPIError(dto.ErrAccountNotFound)
    }
    
    return account, nil
}
```

**关键修改点**：

1. **Create 方法**：
```go
func (s *accountService) Create(ctx context.Context, req *CreateAccountRequest) (*model.Account, error) {
    // 检查邮箱是否已存在
    existing, _ := s.accountRepo.FindByEmail(ctx, req.Email)
    if existing != nil {
        return nil, dto.NewAPIErrorWithMessage(
            dto.ErrAccountExists,
            fmt.Sprintf("邮箱账户 %s 已存在", req.Email),
        )
    }
    
    // 加密凭证
    encryptedCreds, err := s.encryptCredentials(req)
    if err != nil {
        logger.Error("failed to encrypt credentials", "error", err)
        return nil, fmt.Errorf("encryption error: %w", err)
    }
    
    // 创建账户
    account := &model.Account{
        UID:                  uuid.New().String(),
        Email:                req.Email,
        Provider:             req.Provider,
        EncryptedCredentials: encryptedCreds,
        // ... 其他字段
    }
    
    if err := s.accountRepo.Create(ctx, account); err != nil {
        logger.Error("failed to create account in database",
            "email", req.Email,
            "error", err,
        )
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    logger.Info("account created successfully", "uid", account.UID, "email", account.Email)
    return account, nil
}
```

2. **TestConnection 方法**：
```go
func (s *accountService) TestConnection(ctx context.Context, uid string) error {
    account, err := s.GetByUID(ctx, uid)
    if err != nil {
        return err // 已经是 APIError 或系统错误
    }
    
    // 创建适配器
    adapter, err := s.adapterFactory.CreateAdapter(account)
    if err != nil {
        return dto.NewAPIErrorWithMessage(
            dto.ErrAccountInvalid,
            "账户配置无效: "+err.Error(),
        )
    }
    
    // 测试连接
    if err := adapter.Connect(ctx); err != nil {
        return dto.NewAPIErrorWithMessage(
            dto.ErrConnectionFailed,
            "连接失败: "+err.Error(),
        )
    }
    
    return nil
}
```

**完整修改清单**：
- [ ] Create
- [ ] GetByUID
- [ ] List
- [ ] Update
- [ ] Delete
- [ ] TestConnection
- [ ] SetStatus
- [ ] DisableAccount
- [ ] EnableAccount
- [ ] ClearSyncError


#### 步骤 3.2: 重构 email_service.go

**修改策略**：同 account_service.go

**关键修改点**：

```go
func (s *emailService) GetEmailByID(ctx context.Context, id int64) (*model.Email, error) {
    email, err := s.emailRepo.FindByID(ctx, id)
    if err != nil {
        logger.Error("database error when finding email", "id", id, "error", err)
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    if email == nil {
        return nil, dto.NewAPIError(dto.ErrEmailNotFound)
    }
    
    return email, nil
}

func (s *emailService) MarkAsRead(ctx context.Context, ids []int64) error {
    if len(ids) == 0 {
        return dto.NewAPIErrorWithMessage(
            dto.ErrInvalidRequest,
            "邮件 ID 列表不能为空",
        )
    }
    
    if err := s.emailRepo.BatchUpdateReadStatus(ctx, ids, true); err != nil {
        logger.Error("failed to mark emails as read", "ids", ids, "error", err)
        return fmt.Errorf("database error: %w", err)
    }
    
    return nil
}
```

**完整修改清单**：
- [ ] GetEmailList
- [ ] GetEmailByID
- [ ] SearchEmails
- [ ] MarkAsRead
- [ ] MarkAsUnread
- [ ] MarkAllAsRead
- [ ] ToggleStar
- [ ] ArchiveEmail
- [ ] DeleteEmail
- [ ] GetUnreadCount
- [ ] GetAccountStats

#### 步骤 3.3: 重构 rule_service.go

**修改策略**：同上

**关键修改点**：

```go
func (s *ruleService) Create(ctx context.Context, rule *model.EmailRule) error {
    // 验证规则配置
    if err := s.validateRule(rule); err != nil {
        return dto.NewAPIErrorWithMessage(
            dto.ErrRuleInvalid,
            "规则配置无效: "+err.Error(),
        )
    }
    
    // 检查规则名称是否重复
    existing, _ := s.ruleRepo.FindByName(ctx, rule.Name)
    if existing != nil {
        return dto.NewAPIErrorWithMessage(
            dto.ErrDuplicateResource,
            fmt.Sprintf("规则名称 '%s' 已存在", rule.Name),
        )
    }
    
    if err := s.ruleRepo.Create(ctx, rule); err != nil {
        logger.Error("failed to create rule", "name", rule.Name, "error", err)
        return fmt.Errorf("database error: %w", err)
    }
    
    return nil
}

func (s *ruleService) GetByID(ctx context.Context, id int64) (*model.EmailRule, error) {
    rule, err := s.ruleRepo.FindByID(ctx, id)
    if err != nil {
        logger.Error("database error when finding rule", "id", id, "error", err)
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    if rule == nil {
        return nil, dto.NewAPIError(dto.ErrRuleNotFound)
    }
    
    return rule, nil
}
```

**完整修改清单**：
- [ ] Create
- [ ] GetByID
- [ ] List
- [ ] Update
- [ ] Delete
- [ ] Enable
- [ ] Disable
- [ ] ExecuteRule
- [ ] ValidateRule

#### 步骤 3.4: 重构 webhook_service.go

**修改策略**：同上

**完整修改清单**：
- [ ] Create
- [ ] GetByID
- [ ] List
- [ ] Update
- [ ] Delete
- [ ] Enable
- [ ] Disable
- [ ] TriggerWebhook
- [ ] ValidateWebhook

#### 步骤 3.5: 重构其他 Service

**文件清单**：
- [ ] auth_service.go
- [ ] oauth2_service.go
- [ ] sync_service.go
- [ ] sync_manager.go
- [ ] attachment_service.go
- [ ] event_service.go
- [ ] system_service.go


---

### 第四阶段：添加错误处理中间件 (1 小时)

#### 步骤 4.1: 创建错误处理中间件

**文件**: `backend/internal/middleware/error_handler.go`

**代码实现**：
```go
package middleware

import (
    "fusionmail/internal/dto"
    "fusionmail/pkg/logger"
    "runtime/debug"
    
    "github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware 统一错误处理中间件
func ErrorHandlerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                // 记录 panic 信息和堆栈
                logger.Error("panic recovered",
                    "error", err,
                    "stack", string(debug.Stack()),
                    "path", c.Request.URL.Path,
                    "method", c.Request.Method,
                )
                
                // 返回 500 错误
                dto.InternalServerErrorResponse(c, "服务器内部错误")
                c.Abort()
            }
        }()
        
        c.Next()
        
        // 处理错误
        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            
            // 记录错误日志
            logger.Error("request error",
                "path", c.Request.URL.Path,
                "method", c.Request.Method,
                "error", err.Error(),
            )
            
            // 返回错误响应
            dto.HandleServiceError(c, err)
            c.Abort()
        }
    }
}

// RequestLoggerMiddleware 请求日志中间件（增强版）
func RequestLoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 记录请求开始
        logger.Info("request started",
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "ip", c.ClientIP(),
        )
        
        c.Next()
        
        // 记录请求结束
        statusCode := c.Writer.Status()
        if statusCode >= 400 {
            logger.Warn("request completed with error",
                "method", c.Request.Method,
                "path", c.Request.URL.Path,
                "status", statusCode,
            )
        } else {
            logger.Info("request completed",
                "method", c.Request.Method,
                "path", c.Request.URL.Path,
                "status", statusCode,
            )
        }
    }
}
```

#### 步骤 4.2: 更新路由配置

**文件**: `backend/internal/router/router.go`

**修改方案**：
```go
func SetupRouter(/* ... */) *gin.Engine {
    r := gin.New()
    
    // 全局中间件
    r.Use(gin.Recovery())                          // Gin 默认的 Recovery
    r.Use(middleware.ErrorHandlerMiddleware())     // 新增：统一错误处理
    r.Use(middleware.RequestLoggerMiddleware())    // 新增：请求日志
    r.Use(middleware.CORSMiddleware())
    r.Use(middleware.RateLimitMiddleware())
    
    // ... 路由配置
    
    return r
}
```


---

### 第五阶段：完善错误日志 (1 小时)

#### 步骤 5.1: 增强日志记录

**在关键位置添加日志**：

1. **Service 层**：
```go
// 成功操作
logger.Info("operation completed", "operation", "create_account", "uid", account.UID)

// 业务错误
logger.Warn("business error", "operation", "get_account", "uid", uid, "error", "account not found")

// 系统错误
logger.Error("system error", "operation", "create_account", "error", err.Error())
```

2. **Repository 层**：
```go
// 数据库操作失败
logger.Error("database operation failed",
    "operation", "create",
    "table", "accounts",
    "error", err.Error(),
)
```

3. **Adapter 层**：
```go
// 连接失败
logger.Error("failed to connect to email provider",
    "provider", account.Provider,
    "email", account.Email,
    "error", err.Error(),
)

// 同步失败
logger.Error("failed to sync emails",
    "account_uid", account.UID,
    "error", err.Error(),
)
```

#### 步骤 5.2: 添加错误追踪 ID

**可选：为每个请求生成唯一 ID**

**文件**: `backend/internal/middleware/request_id.go`

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// RequestIDMiddleware 为每个请求生成唯一 ID
func RequestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)
        
        c.Next()
    }
}
```

**在日志中使用**：
```go
requestID, _ := c.Get("request_id")
logger.Error("request error",
    "request_id", requestID,
    "error", err.Error(),
)
```

---

## 🧪 测试计划

### 单元测试

#### 测试 APIError
```go
func TestAPIError(t *testing.T) {
    // 测试创建错误
    err := dto.NewAPIError(dto.ErrAccountNotFound)
    assert.Equal(t, dto.ErrAccountNotFound, err.Code)
    assert.Equal(t, "账户不存在", err.Message)
    
    // 测试带详情的错误
    err = dto.NewAPIErrorWithDetails(
        dto.ErrValidationFailed,
        "验证失败",
        map[string]interface{}{"email": "invalid format"},
    )
    assert.NotNil(t, err.Details)
}
```

#### 测试错误响应
```go
func TestHandleServiceError(t *testing.T) {
    // 测试 APIError
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    
    err := dto.NewAPIError(dto.ErrAccountNotFound)
    dto.HandleServiceError(c, err)
    
    assert.Equal(t, http.StatusNotFound, w.Code)
    
    // 测试系统错误
    w = httptest.NewRecorder()
    c, _ = gin.CreateTestContext(w)
    
    err = fmt.Errorf("database error")
    dto.HandleServiceError(c, err)
    
    assert.Equal(t, http.StatusInternalServerError, w.Code)
}
```

### 集成测试

#### 测试完整的错误流程
```go
func TestAccountNotFoundError(t *testing.T) {
    // 发送请求
    req := httptest.NewRequest("GET", "/api/v1/accounts/non-existent", nil)
    w := httptest.NewRecorder()
    
    router.ServeHTTP(w, req)
    
    // 验证响应
    assert.Equal(t, http.StatusNotFound, w.Code)
    
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    
    assert.False(t, response["success"].(bool))
    assert.Equal(t, float64(dto.ErrAccountNotFound), response["code"].(float64))
    assert.Contains(t, response["error"].(string), "账户不存在")
}
```

---

## 📊 重构进度追踪

### 第一阶段：基础设施 ✅
- [ ] 创建 error_code.go
- [ ] 创建 api_error.go
- [ ] 更新 response.go

### 第二阶段：Handler 层 ⏳
- [ ] account_handler.go (11 个方法)
- [ ] email_handler.go (11 个方法)
- [ ] rule_handler.go (9 个方法)
- [ ] webhook_handler.go (9 个方法)
- [ ] auth.go
- [ ] oauth2_handler.go
- [ ] system_handler.go
- [ ] attachment_handler.go

### 第三阶段：Service 层 ⏳
- [ ] account_service.go (10 个方法)
- [ ] email_service.go (11 个方法)
- [ ] rule_service.go (9 个方法)
- [ ] webhook_service.go (8 个方法)
- [ ] auth_service.go
- [ ] oauth2_service.go
- [ ] sync_service.go
- [ ] sync_manager.go
- [ ] attachment_service.go
- [ ] event_service.go
- [ ] system_service.go

### 第四阶段：中间件 ⏳
- [ ] 创建 error_handler.go
- [ ] 更新 router.go

### 第五阶段：日志 ⏳
- [ ] 添加 Service 层日志
- [ ] 添加 Repository 层日志
- [ ] 添加 Adapter 层日志
- [ ] 添加 Request ID 中间件（可选）

### 第六阶段：测试 ⏳
- [ ] 单元测试
- [ ] 集成测试
- [ ] 手动测试

---

## 🎯 预期成果

### 改进前
```json
{
  "success": false,
  "error": "failed to get account: account not found: abc123"
}
```

### 改进后
```json
{
  "success": false,
  "code": 2000,
  "error": "账户不存在"
}
```

### 带详情的错误
```json
{
  "success": false,
  "code": 1001,
  "error": "数据验证失败",
  "details": {
    "fields": {
      "email": "邮箱格式无效",
      "password": "密码长度不能少于 8 位"
    }
  }
}
```

---

## 📝 注意事项

### 1. 向后兼容
- 保留旧的响应函数（如 `ErrorResponse`）
- 新增带错误码的函数（如 `ErrorResponseWithCode`）
- 逐步迁移，不影响现有功能

### 2. 错误信息安全
- 系统错误不暴露详细信息给前端
- 数据库错误统一返回"服务器内部错误"
- 敏感信息不记录到日志

### 3. 性能考虑
- 错误日志不要过于频繁
- 避免在循环中记录日志
- 使用结构化日志提升性能

### 4. 团队协作
- 统一错误码定义
- 文档化所有错误码
- Code Review 确保规范执行

---

## 🚀 开始实施

### 立即开始
```bash
# 1. 创建新文件
cd backend/internal/dto
touch error_code.go api_error.go

# 2. 编写代码
# 按照本文档的代码示例实现

# 3. 测试
go test ./internal/dto/...

# 4. 逐步重构 Handler
# 从 account_handler.go 开始

# 5. 提交代码
git add .
git commit -m "feat: 统一错误处理机制 - 第一阶段"
```

### 预计时间
- 第一阶段：1-2 小时
- 第二阶段：2-3 小时
- 第三阶段：3-4 小时
- 第四阶段：1 小时
- 第五阶段：1 小时
- 测试和调试：2-3 小时

**总计：10-15 小时**

---

**文档版本**: v1.0  
**创建时间**: 2025-11-07  
**最后更新**: 2025-11-07
