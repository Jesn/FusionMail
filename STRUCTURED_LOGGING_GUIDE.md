# FusionMail 结构化日志指南

## 📚 概述

FusionMail 使用结构化日志系统，提供清晰、可追踪、易于分析的日志输出。

---

## 🎯 日志特性

### 1. 结构化字段
- **时间戳**: RFC3339 格式
- **日志级别**: DEBUG, INFO, WARN, ERROR, FATAL
- **消息**: 简洁的日志消息
- **调用者**: 文件名和行号
- **字段**: 键值对形式的上下文信息

### 2. Request ID 追踪
- 每个请求都有唯一的 Request ID
- Request ID 贯穿整个请求链路
- 便于关联和追踪请求

### 3. 分级日志
- **DEBUG**: 调试信息（开发环境）
- **INFO**: 一般信息（正常操作）
- **WARN**: 警告信息（业务错误）
- **ERROR**: 错误信息（系统错误）
- **FATAL**: 致命错误（程序退出）

---

## 📝 日志格式

### 基本格式
```
[LEVEL] TIMESTAMP MESSAGE (CALLER) | field1=value1, field2=value2, ...
```

### 示例
```
[INFO] 2025-11-07T15:04:05+08:00 account created successfully (account_service.go:123) | request_id=abc-123, uid=user-001, email=test@example.com, provider=gmail

[WARN] 2025-11-07T15:04:06+08:00 business error occurred (response.go:45) | request_id=abc-124, error_code=2000, error_message=账户不存在, path=/api/v1/accounts/xxx, method=GET

[ERROR] 2025-11-07T15:04:07+08:00 system error occurred (response.go:67) | request_id=abc-125, error=database connection failed, path=/api/v1/accounts, method=POST
```

---

## 🔧 使用方法

### 1. 在 Service 层使用

```go
import "fusionmail/pkg/logger"

func (s *accountService) CreateAccount(ctx context.Context, req *dto.CreateAccountRequest) (*model.Account, error) {
    // 记录信息日志
    logger.Info("creating account",
        "operation", "create_account",
        "email", req.Email,
        "provider", req.Provider,
    )
    
    // 记录错误日志
    if err != nil {
        logger.Error("failed to create account",
            "operation", "create_account",
            "email", req.Email,
            "error", err.Error(),
        )
        return nil, err
    }
    
    return account, nil
}
```

### 2. 在 Handler 层使用（带 Request ID）

```go
import "fusionmail/pkg/logger"

func (h *Handler) CreateAccount(c *gin.Context) {
    // 创建带 Request ID 的日志记录器
    log := logger.WithRequestID(c)
    
    // 记录信息日志
    log.Info("processing create account request",
        "email", req.Email,
        "provider", req.Provider,
    )
    
    // 记录错误日志
    if err != nil {
        log.Error("failed to process request",
            "error", err.Error(),
            "path", c.Request.URL.Path,
        )
        dto.HandleServiceError(c, err)
        return
    }
}
```

### 3. 在中间件中使用

```go
import "fusionmail/pkg/logger"

func SomeMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        log := logger.WithRequestID(c)
        
        log.Info("middleware processing",
            "path", c.Request.URL.Path,
            "method", c.Request.Method,
        )
        
        c.Next()
    }
}
```

---

## 📊 日志级别使用指南

### DEBUG
**用途**: 详细的调试信息

**示例**:
```go
logger.Debug("processing email sync",
    "account_uid", accountUID,
    "email_count", len(emails),
    "sync_mode", "incremental",
)
```

### INFO
**用途**: 正常的业务操作

**示例**:
```go
logger.Info("account created successfully",
    "uid", account.UID,
    "email", account.Email,
    "provider", account.Provider,
)
```

### WARN
**用途**: 业务错误或警告

**示例**:
```go
log.Warn("business error occurred",
    "error_code", apiErr.Code,
    "error_message", apiErr.Message,
    "path", c.Request.URL.Path,
)
```

### ERROR
**用途**: 系统错误

**示例**:
```go
logger.Error("database connection failed",
    "operation", "create_account",
    "error", err.Error(),
    "retry_count", retryCount,
)
```

### FATAL
**用途**: 致命错误（程序无法继续运行）

**示例**:
```go
logger.Fatal("failed to initialize database",
    "error", err.Error(),
    "database_url", dbURL,
)
```

---

## 🎯 最佳实践

### 1. 使用有意义的消息
```go
// ✅ 好的
logger.Info("account created successfully")

// ❌ 不好的
logger.Info("success")
```

### 2. 提供足够的上下文
```go
// ✅ 好的
logger.Error("failed to create account",
    "operation", "create_account",
    "email", req.Email,
    "provider", req.Provider,
    "error", err.Error(),
)

// ❌ 不好的
logger.Error("error", "error", err.Error())
```

### 3. 使用一致的字段名
```go
// ✅ 好的 - 使用统一的字段名
logger.Info("processing request",
    "operation", "create_account",
    "email", req.Email,
)

// ❌ 不好的 - 字段名不一致
logger.Info("processing request",
    "op", "create_account",
    "user_email", req.Email,
)
```

### 4. 避免记录敏感信息
```go
// ✅ 好的 - 不记录密码
logger.Info("account created",
    "email", req.Email,
)

// ❌ 不好的 - 记录了密码
logger.Info("account created",
    "email", req.Email,
    "password", req.Password, // 不要这样做！
)
```

### 5. 在 Handler 层使用 Request ID
```go
// ✅ 好的 - 使用 WithRequestID
log := logger.WithRequestID(c)
log.Info("processing request")

// ❌ 不好的 - 没有 Request ID
logger.Info("processing request")
```

---

## 🔍 日志追踪示例

### 场景：创建账户的完整日志链路

```
1. 请求到达
[INFO] 2025-11-07T15:04:05+08:00 processing create account request (account_handler.go:45) | request_id=abc-123, email=test@gmail.com, provider=gmail

2. Service 层处理
[INFO] 2025-11-07T15:04:05+08:00 creating account (account_service.go:78) | operation=create_account, email=test@gmail.com, provider=gmail

3. 加密凭证
[DEBUG] 2025-11-07T15:04:05+08:00 encrypting credentials (account_service.go:95) | operation=create_account, email=test@gmail.com

4. 保存到数据库
[INFO] 2025-11-07T15:04:05+08:00 account created successfully (account_service.go:123) | operation=create_account, uid=user-001, email=test@gmail.com, provider=gmail

5. 返回响应
[INFO] 2025-11-07T15:04:05+08:00 request completed (logger.go:67) | request_id=abc-123, status=201, duration=123ms
```

### 场景：错误处理的日志链路

```
1. 请求到达
[INFO] 2025-11-07T15:04:06+08:00 processing get account request (account_handler.go:89) | request_id=abc-124, uid=non-existent

2. Service 层查询
[WARN] 2025-11-07T15:04:06+08:00 account not found (account_service.go:156) | operation=get_account, uid=non-existent

3. 返回业务错误
[WARN] 2025-11-07T15:04:06+08:00 business error occurred (response.go:45) | request_id=abc-124, error_code=2000, error_message=账户不存在, path=/api/v1/accounts/non-existent, method=GET

4. 请求完成
[INFO] 2025-11-07T15:04:06+08:00 request completed (logger.go:67) | request_id=abc-124, status=404, duration=45ms
```

---

## 🧪 测试日志

### 运行日志测试脚本
```bash
# 启动服务
cd backend && ./server

# 在另一个终端运行测试
./test_structured_logging.sh
```

### 手动测试
```bash
# 发送带 Request ID 的请求
curl -H "X-Request-ID: test-001" http://localhost:3333/api/v1/accounts

# 查看服务器日志，应该看到：
# [INFO] ... | request_id=test-001, ...
```

---

## 📈 日志分析

### 使用 grep 过滤日志

```bash
# 查找特定 Request ID 的所有日志
grep "request_id=abc-123" server.log

# 查找所有错误日志
grep "\[ERROR\]" server.log

# 查找特定操作的日志
grep "operation=create_account" server.log

# 查找特定用户的日志
grep "email=test@gmail.com" server.log
```

### 使用 jq 分析日志（如果使用 JSON 格式）

```bash
# 统计各级别日志数量
cat server.log | jq -r '.level' | sort | uniq -c

# 查找响应时间最长的请求
cat server.log | jq 'select(.duration != null) | {request_id, duration}' | sort -k2 -n

# 查找所有错误
cat server.log | jq 'select(.level == "ERROR")'
```

---

## 🚀 未来优化

### 短期优化
1. **JSON 格式输出**
   - 便于机器解析
   - 支持日志聚合工具

2. **日志轮转**
   - 按大小或时间轮转
   - 自动压缩旧日志

### 长期优化
3. **集成日志聚合服务**
   - ELK Stack (Elasticsearch, Logstash, Kibana)
   - Loki + Grafana
   - Datadog / New Relic

4. **分布式追踪**
   - OpenTelemetry 集成
   - Jaeger / Zipkin
   - 跨服务追踪

---

## 📝 常见问题

### Q: 如何修改日志级别？
A: 在 main.go 中设置：
```go
logger.SetLevel(logger.DEBUG) // 开发环境
logger.SetLevel(logger.INFO)  // 生产环境
```

### Q: 如何在日志中隐藏敏感信息？
A: 不要记录密码、Token 等敏感信息：
```go
// ✅ 好的
logger.Info("user authenticated", "email", user.Email)

// ❌ 不好的
logger.Info("user authenticated", "password", user.Password)
```

### Q: 如何追踪跨多个服务的请求？
A: 使用 Request ID 并在服务间传递：
```go
// 在 HTTP 请求头中传递
req.Header.Set("X-Request-ID", requestID)
```

### Q: 日志太多怎么办？
A: 
1. 提高日志级别（INFO → WARN → ERROR）
2. 使用日志采样（只记录部分请求）
3. 使用日志聚合工具过滤

---

**完成日期**: 2025-11-07  
**版本**: v1.0  
**状态**: ✅ 完成
