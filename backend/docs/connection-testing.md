# 连接测试功能文档

## 概述

GraphQuickAdapter 提供了完善的连接测试功能，用于验证邮箱账户的连接状态和凭据有效性。该功能支持多种测试模式，提供详细的错误信息和性能指标。

## 核心功能

### 1. 基本连接测试

`TestConnection` 方法提供快速的连接验证：

```go
func (a *GraphQuickAdapter) TestConnection(ctx context.Context) error
```

**特性：**
- 10 秒超时保护
- 自动 token 刷新
- 详细错误分类
- 性能日志记录

**使用示例：**
```go
ctx := context.Background()
err := adapter.TestConnection(ctx)
if err != nil {
    log.Printf("连接测试失败: %v", err)
}
```

### 2. 详细连接测试

`TestConnectionWithDetails` 方法返回详细的连接信息：

```go
func (a *GraphQuickAdapter) TestConnectionWithDetails(ctx context.Context) (*ConnectionTestResult, error)
```

**返回信息：**
```go
type ConnectionTestResult struct {
    UserID         string `json:"id"`
    DisplayName    string `json:"displayName"`
    Mail           string `json:"mail"`
    UserPrincipal  string `json:"userPrincipalName"`
    ResponseTimeMs int64  `json:"response_time_ms"`
}
```

**使用示例：**
```go
result, err := adapter.TestConnectionWithDetails(ctx)
if err != nil {
    log.Printf("详细连接测试失败: %v", err)
    return
}

fmt.Printf("用户: %s (%s)\n", result.DisplayName, result.Mail)
fmt.Printf("响应时间: %d ms\n", result.ResponseTimeMs)
```

### 3. 凭据验证

`ValidateCredentials` 方法验证配置的有效性：

```go
func (a *GraphQuickAdapter) ValidateCredentials(ctx context.Context) error
```

**验证项目：**
- 必需字段检查（ClientID、RefreshToken）
- 凭据格式验证
- Token 刷新测试

**使用示例：**
```go
err := adapter.ValidateCredentials(ctx)
if err != nil {
    var connErr *ConnectionTestError
    if errors.As(err, &connErr) {
        fmt.Printf("验证失败 [%s]: %s\n", connErr.Type, connErr.Message)
    }
}
```

## 错误处理

### ConnectionTestError 类型

所有连接测试相关的错误都使用 `ConnectionTestError` 类型：

```go
type ConnectionTestError struct {
    Type    string `json:"type"`    // 错误类型
    Message string `json:"message"` // 用户友好消息
    Details string `json:"details"` // 技术详情
    Code    string `json:"code,omitempty"`    // API 错误码
    Status  int    `json:"status,omitempty"`  // HTTP 状态码
}
```

### 错误类型分类

#### 1. authentication - 认证错误
- **原因**：凭据无效、token 过期
- **解决方案**：检查 ClientID 和 RefreshToken

```go
// 示例错误
&ConnectionTestError{
    Type:    "authentication",
    Message: "访问令牌无效或已过期",
    Details: "The provided authorization grant is invalid",
    Code:    "invalid_grant",
    Status:  400,
}
```

#### 2. network - 网络错误
- **原因**：网络连接问题、超时
- **解决方案**：检查网络连接和代理设置

```go
// 示例错误
&ConnectionTestError{
    Type:    "network",
    Message: "网络请求失败",
    Details: "context deadline exceeded",
}
```

#### 3. api - API 错误
- **原因**：Microsoft Graph API 返回错误
- **解决方案**：根据具体错误码处理

```go
// 示例错误
&ConnectionTestError{
    Type:    "api",
    Message: "没有权限访问此资源",
    Details: "Insufficient privileges to complete the operation",
    Code:    "Forbidden",
    Status:  403,
}
```

#### 4. validation - 验证错误
- **原因**：配置参数缺失或格式错误
- **解决方案**：检查配置参数

```go
// 示例错误
&ConnectionTestError{
    Type:    "validation",
    Message: "客户端 ID 不能为空",
    Details: "ClientID 字段是必需的",
}
```

#### 5. parsing - 解析错误
- **原因**：API 响应格式异常
- **解决方案**：检查 API 版本兼容性

### 常见错误码处理

| 错误码 | 含义 | 解决方案 |
|--------|------|----------|
| `InvalidAuthenticationToken` | Token 无效 | 刷新 token 或重新获取 |
| `Forbidden` | 权限不足 | 检查应用权限配置 |
| `ThrottledRequest` | 请求过频 | 实施退避重试 |
| `ServiceNotAvailable` | 服务不可用 | 稍后重试 |
| `invalid_grant` | 刷新令牌无效 | 重新获取 refresh_token |
| `invalid_client` | 客户端 ID 无效 | 检查应用注册 |

## 性能考虑

### 超时设置

- **基本连接测试**：10 秒超时
- **详细连接测试**：15 秒超时
- **凭据验证**：使用默认超时（30 秒）

### 响应时间基准

- **正常情况**：< 2 秒
- **可接受范围**：< 5 秒
- **需要优化**：> 5 秒

### 并发安全

所有连接测试方法都是并发安全的，可以同时从多个协程调用。

## 使用指南

### 1. 快速验证

```go
// 创建适配器
adapter, err := NewGraphQuickAdapter(config)
if err != nil {
    return err
}

// 快速连接测试
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := adapter.TestConnection(ctx); err != nil {
    return fmt.Errorf("连接失败: %w", err)
}
```

### 2. 详细诊断

```go
// 执行详细测试
result, err := adapter.TestConnectionWithDetails(ctx)
if err != nil {
    var connErr *ConnectionTestError
    if errors.As(err, &connErr) {
        switch connErr.Type {
        case "authentication":
            // 处理认证错误
            log.Printf("认证失败: %s", connErr.Message)
        case "network":
            // 处理网络错误
            log.Printf("网络问题: %s", connErr.Message)
        case "api":
            // 处理 API 错误
            log.Printf("API 错误 [%s]: %s", connErr.Code, connErr.Message)
        }
    }
    return err
}

// 使用详细信息
log.Printf("连接成功: %s (%s), 响应时间: %dms", 
    result.DisplayName, result.Mail, result.ResponseTimeMs)
```

### 3. 批量验证

```go
// 批量测试多个账户
accounts := []*Config{config1, config2, config3}
results := make(chan error, len(accounts))

for _, config := range accounts {
    go func(cfg *Config) {
        adapter, err := NewGraphQuickAdapter(cfg)
        if err != nil {
            results <- err
            return
        }
        
        results <- adapter.TestConnection(ctx)
    }(config)
}

// 收集结果
for i := 0; i < len(accounts); i++ {
    if err := <-results; err != nil {
        log.Printf("账户 %d 测试失败: %v", i, err)
    }
}
```

## 测试工具

### 命令行测试工具

使用提供的测试脚本进行连接测试：

```bash
# 设置环境变量
export TEST_EMAIL=your@outlook.com
export TEST_CLIENT_ID=your_client_id
export TEST_REFRESH_TOKEN=your_refresh_token

# 运行测试
go run backend/scripts/test_connection.go
```

**测试内容：**
1. 凭据验证
2. 基本连接测试
3. 详细连接测试
4. Token 状态检查
5. 并发连接测试
6. 性能基准测试

### 单元测试

运行单元测试：

```bash
# 运行连接测试相关的单元测试
go test ./internal/adapter -run TestGraphQuickAdapter_TestConnection

# 运行所有连接相关测试
go test ./internal/adapter -run Connection

# 运行性能基准测试
go test ./internal/adapter -bench BenchmarkGraphQuickAdapter_TestConnection
```

## 监控和告警

### 关键指标

1. **连接成功率**：应保持在 99% 以上
2. **平均响应时间**：应小于 2 秒
3. **错误分布**：监控各类错误的比例

### 告警规则

```yaml
# 连接成功率告警
- alert: GraphQuickAdapterConnectionFailure
  expr: connection_success_rate < 0.95
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "GraphQuickAdapter 连接成功率过低"

# 响应时间告警
- alert: GraphQuickAdapterSlowResponse
  expr: connection_response_time_p95 > 5000
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "GraphQuickAdapter 响应时间过长"
```

## 故障排除

### 常见问题

#### 1. 连接超时

**症状**：`context deadline exceeded` 错误

**可能原因**：
- 网络连接问题
- 代理配置错误
- Microsoft 服务响应慢

**解决方案**：
- 检查网络连接
- 验证代理设置
- 增加超时时间
- 检查 Microsoft 服务状态

#### 2. 认证失败

**症状**：`invalid_grant` 或 `InvalidAuthenticationToken` 错误

**可能原因**：
- RefreshToken 过期
- ClientID 错误
- 应用权限不足

**解决方案**：
- 重新获取 RefreshToken
- 验证 ClientID 配置
- 检查应用权限设置

#### 3. 权限不足

**症状**：`Forbidden` 错误

**可能原因**：
- 应用权限配置不正确
- 用户权限不足
- 租户策略限制

**解决方案**：
- 检查应用权限配置
- 联系管理员授权
- 验证租户策略

### 调试技巧

1. **启用详细日志**：
```go
adapter.logger.Info("调试信息", "key", "value")
```

2. **检查 Token 状态**：
```go
tokenInfo := adapter.GetTokenInfo()
log.Printf("Token 信息: %+v", tokenInfo)
```

3. **使用测试工具**：
```bash
go run backend/scripts/test_connection.go
```

---

**注意**：连接测试功能是短效邮箱适配器的重要组成部分，确保在生产环境中正确配置监控和告警，以便及时发现和解决连接问题。