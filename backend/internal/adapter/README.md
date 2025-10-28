# 邮箱协议适配层

## 概述

适配层提供统一的接口来访问不同的邮箱服务，支持多种协议和提供商。

## 架构设计

```
┌─────────────────────────────────────────┐
│         MailProvider 接口                │
│  (统一的邮箱服务提供商接口)               │
└─────────────────────────────────────────┘
                    ▲
                    │
        ┌───────────┼───────────┐
        │           │           │
┌───────┴──────┐ ┌──┴────────┐ ┌┴──────────┐
│ IMAP Adapter │ │Gmail API  │ │Graph API  │
│              │ │Adapter    │ │Adapter    │
└──────────────┘ └───────────┘ └───────────┘
        │
┌───────┴──────┐
│ POP3 Adapter │
└──────────────┘
```

## 支持的协议

### 1. IMAP (Internet Message Access Protocol)
- **优点**：功能完整，支持增量同步，支持文件夹管理
- **缺点**：需要保持连接，性能相对较低
- **适用**：通用邮箱服务（iCloud、QQ、163 等）

### 2. POP3 (Post Office Protocol 3)
- **优点**：简单，兼容性好
- **缺点**：不支持增量同步，功能有限
- **适用**：简单的邮件接收场景
- **注意**：不推荐使用，仅作为备选方案

### 3. Gmail API
- **优点**：性能好，功能强大，官方支持
- **缺点**：需要 OAuth2 认证，有配额限制
- **适用**：Gmail 邮箱
- **降级**：API 不可用时自动降级到 IMAP

### 4. Microsoft Graph API
- **优点**：性能好，功能强大，官方支持
- **缺点**：需要 OAuth2 认证
- **适用**：Outlook/Hotmail 邮箱
- **降级**：API 不可用时自动降级到 IMAP

## 接口定义

### MailProvider 接口

```go
type MailProvider interface {
    Connect(ctx context.Context) error
    Disconnect() error
    FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error)
    FetchEmailDetail(ctx context.Context, providerID string) (*Email, error)
    GetProviderType() string
    GetProtocol() string
    TestConnection(ctx context.Context) error
}
```

### 核心方法

- **Connect**: 建立连接（IMAP/POP3 需要，API 可选）
- **Disconnect**: 断开连接
- **FetchEmails**: 增量拉取邮件列表
- **FetchEmailDetail**: 获取邮件完整内容
- **TestConnection**: 测试连接是否正常

## 工厂模式

使用工厂模式创建适配器实例：

```go
factory := adapter.NewFactory()

// 创建 IMAP 适配器
config := &adapter.Config{
    Provider: "icloud",
    Protocol: "imap",
    Credentials: &adapter.Credentials{
        Email:    "user@icloud.com",
        Password: "app-password",
        Host:     "imap.mail.me.com",
        Port:     993,
        TLS:      true,
    },
}

provider, err := factory.CreateProvider(config)
```

## 数据结构

### Email 结构

```go
type Email struct {
    ProviderID   string      // 服务商原生 ID
    MessageID    string      // Message-ID
    Subject      string      // 主题
    FromAddress  string      // 发件人
    ToAddresses  []string    // 收件人
    TextBody     string      // 纯文本正文
    HTMLBody     string      // HTML 正文
    Attachments  []Attachment // 附件
    SentAt       time.Time   // 发送时间
    // ... 更多字段
}
```

### Credentials 结构

```go
type Credentials struct {
    Email        string    // 邮箱地址
    AuthType     string    // 认证类型
    Password     string    // 密码
    AccessToken  string    // OAuth2 访问令牌
    RefreshToken string    // OAuth2 刷新令牌
    Host         string    // IMAP/POP3 服务器
    Port         int       // 端口
    // ... 更多字段
}
```

## 代理支持

所有适配器都支持 HTTP/SOCKS5 代理：

```go
config := &adapter.Config{
    // ... 其他配置
    Proxy: &adapter.ProxyConfig{
        Enabled:  true,
        Type:     "socks5",
        Host:     "127.0.0.1",
        Port:     1080,
        Username: "user",
        Password: "pass",
    },
}
```

## 错误处理

适配器使用标准的 Go error 处理：

- 连接错误：网络问题、认证失败
- 协议错误：服务器响应异常
- 超时错误：操作超时
- 配额错误：API 配额超限

## 实现状态

- [x] 接口定义
- [x] 工厂模式
- [ ] IMAP 适配器（待实现）
- [ ] POP3 适配器（待实现）
- [ ] Gmail API 适配器（待实现）
- [ ] Graph API 适配器（待实现）

## 使用示例

### 基本使用

```go
// 1. 创建工厂
factory := adapter.NewFactory()

// 2. 创建适配器
provider, err := factory.CreateProvider(config)
if err != nil {
    return err
}

// 3. 连接
ctx := context.Background()
if err := provider.Connect(ctx); err != nil {
    return err
}
defer provider.Disconnect()

// 4. 拉取邮件
since := time.Now().AddDate(0, 0, -7) // 最近 7 天
emails, err := provider.FetchEmails(ctx, since, 100)
if err != nil {
    return err
}

// 5. 处理邮件
for _, email := range emails {
    fmt.Printf("Subject: %s\n", email.Subject)
}
```

### 测试连接

```go
if err := provider.TestConnection(ctx); err != nil {
    log.Printf("Connection test failed: %v", err)
}
```

## 最佳实践

1. **使用 Context**：所有操作都应该传递 context，支持超时和取消
2. **连接池**：IMAP 适配器应该使用连接池管理连接
3. **错误重试**：网络错误应该自动重试
4. **增量同步**：使用 since 参数进行增量同步
5. **批量处理**：使用 limit 参数控制每次拉取的数量

## 性能优化

1. **并发控制**：限制并发连接数
2. **缓存**：缓存邮件列表和详情
3. **分页**：大量邮件分批拉取
4. **压缩**：启用传输压缩（如果支持）

## 安全考虑

1. **凭证加密**：敏感信息加密存储
2. **TLS/SSL**：强制使用加密连接
3. **令牌刷新**：OAuth2 令牌自动刷新
4. **超时设置**：防止长时间阻塞

## 未来扩展

- [ ] Exchange 协议支持
- [ ] IMAP IDLE 支持（实时推送）
- [ ] 邮件发送功能
- [ ] 联系人同步
- [ ] 日历同步
