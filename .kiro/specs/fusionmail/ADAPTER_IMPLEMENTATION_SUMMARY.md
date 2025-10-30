# 邮箱协议适配器实现总结

## 实现日期
2025-10-30

## 实现概述

本次完成了 FusionMail 项目中三个核心邮箱协议适配器的实现：
1. **POP3 适配器** - 支持传统 POP3 协议
2. **Gmail API 适配器** - 支持 Google Gmail 官方 API
3. **Microsoft Graph 适配器** - 支持 Microsoft Outlook/Hotmail 官方 API

## 实现详情

### 1. POP3 适配器 (`backend/internal/adapter/pop3.go`)

#### 功能特性
- ✅ 基于 `github.com/knadh/go-pop3` 库实现
- ✅ 支持 TLS/SSL 加密连接
- ✅ 支持用户名密码认证
- ✅ 实现邮件列表拉取
- ✅ 实现邮件详情获取
- ✅ 实现连接测试
- ✅ 线程安全（使用 mutex 保护并发访问）

#### 技术限制
- ⚠️ **代理支持受限**：go-pop3 库不支持自定义 Dialer，无法实现 SOCKS5/HTTP 代理
- ⚠️ **不支持增量同步**：POP3 协议本身不支持基于时间的增量拉取，`since` 参数在客户端过滤

#### 使用场景
- 适用于 QQ 邮箱、163 邮箱等国内邮箱服务
- 适用于不支持 IMAP 的传统邮箱服务
- 推荐优先使用 IMAP 协议（功能更强大）

#### 关键实现
```go
// 连接并认证
opt := pop3.Opt{
    Host:       host,
    Port:       port,
    TLSEnabled: true,
}
client := pop3.New(opt)
conn, _ := client.NewConn()
conn.Auth(email, password)

// 拉取邮件
count, _, _ := conn.Stat()
msgBuffer, _ := conn.RetrRaw(msgNum)
```

---

### 2. Gmail API 适配器 (`backend/internal/adapter/gmail.go`)

#### 功能特性
- ✅ 基于 `google.golang.org/api/gmail/v1` 官方 SDK
- ✅ 支持 OAuth2 认证（Access Token + Refresh Token）
- ✅ 支持增量同步（基于 `after:` 查询语法）
- ✅ 支持代理配置（HTTP/SOCKS5）
- ✅ 自动解析邮件头、正文、附件
- ✅ 支持 HTML 和纯文本正文
- ✅ 保留源邮箱状态（已读/未读、标签）

#### 技术优势
- 🚀 **性能优秀**：官方 API 比 IMAP 更快更稳定
- 🔒 **安全性高**：使用 OAuth2，无需存储密码
- 📊 **功能丰富**：支持标签、会话、搜索等高级功能
- ⚡ **增量同步**：支持基于时间的高效增量拉取

#### 关键实现
```go
// OAuth2 认证
token := &oauth2.Token{
    AccessToken:  accessToken,
    RefreshToken: refreshToken,
}
oauth2Config := &oauth2.Config{...}
httpClient := oauth2Config.Client(ctx, token)
service, _ := gmail.NewService(ctx, option.WithHTTPClient(httpClient))

// 增量拉取邮件
query := fmt.Sprintf("in:inbox after:%d", since.Unix())
response, _ := service.Users.Messages.List("me").Q(query).Do()
```

#### 使用场景
- ✅ Gmail 邮箱（推荐首选）
- ✅ Google Workspace 企业邮箱
- ✅ 需要高性能和稳定性的场景

---

### 3. Microsoft Graph 适配器 (`backend/internal/adapter/graph.go`)

#### 功能特性
- ✅ 基于 Microsoft Graph API v1.0
- ✅ 支持 OAuth2 认证（Access Token + Refresh Token）
- ✅ 支持增量同步（基于 `$filter` 查询参数）
- ✅ 支持代理配置（HTTP/SOCKS5）
- ✅ 自动解析邮件头、正文、附件
- ✅ 支持 HTML 和纯文本正文
- ✅ 保留源邮箱状态（已读/未读、分类）

#### 技术优势
- 🚀 **性能优秀**：官方 API 比 IMAP 更快更稳定
- 🔒 **安全性高**：使用 OAuth2，无需存储密码
- 📊 **功能丰富**：支持分类、会话、搜索等高级功能
- ⚡ **增量同步**：支持基于时间的高效增量拉取

#### 关键实现
```go
// OAuth2 认证
token := &oauth2.Token{
    AccessToken:  accessToken,
    RefreshToken: refreshToken,
}
oauth2Config := &oauth2.Config{...}
httpClient := oauth2Config.Client(ctx, token)

// 增量拉取邮件
params := url.Values{}
params.Set("$filter", fmt.Sprintf("receivedDateTime ge %s", since.Format(time.RFC3339)))
params.Set("$top", "100")
requestURL := fmt.Sprintf("%s/me/messages?%s", baseURL, params.Encode())
resp, _ := httpClient.Get(requestURL)
```

#### 使用场景
- ✅ Outlook.com 邮箱（推荐首选）
- ✅ Hotmail 邮箱
- ✅ Microsoft 365 企业邮箱
- ✅ 需要高性能和稳定性的场景

---

## 依赖包管理

### 新增依赖
```bash
go get github.com/knadh/go-pop3@v1.0.0
go get golang.org/x/oauth2@v0.32.0
go get google.golang.org/api@v0.254.0
```

### 依赖说明
- `github.com/knadh/go-pop3`: 轻量级 POP3 客户端库
- `golang.org/x/oauth2`: Google 官方 OAuth2 库
- `google.golang.org/api`: Google API 官方 Go SDK（包含 Gmail API）
- Microsoft Graph API 使用标准 HTTP 客户端，无需额外依赖

---

## 统一接口设计

所有适配器都实现了 `MailProvider` 接口：

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

### 接口优势
- ✅ **统一抽象**：上层服务无需关心具体协议实现
- ✅ **易于扩展**：新增协议只需实现接口
- ✅ **便于测试**：可以轻松 Mock 适配器进行单元测试
- ✅ **灵活切换**：可以动态选择最优协议

---

## 适配器工厂

通过 `Factory` 模式创建适配器实例：

```go
factory := adapter.NewFactory()

// 创建 Gmail 适配器
gmailAdapter, _ := factory.CreateProvider(&adapter.Config{
    Provider: "gmail",
    Protocol: "gmail_api",
    Credentials: &adapter.Credentials{
        AccessToken:  "...",
        RefreshToken: "...",
    },
})

// 创建 Outlook 适配器
outlookAdapter, _ := factory.CreateProvider(&adapter.Config{
    Provider: "outlook",
    Protocol: "graph",
    Credentials: &adapter.Credentials{
        AccessToken:  "...",
        RefreshToken: "...",
    },
})

// 创建 POP3 适配器
pop3Adapter, _ := factory.CreateProvider(&adapter.Config{
    Provider: "qq",
    Protocol: "pop3",
    Credentials: &adapter.Credentials{
        Email:    "user@qq.com",
        Password: "...",
        Host:     "pop.qq.com",
        Port:     995,
        TLS:      true,
    },
})
```

---

## 协议选择建议

### 推荐优先级

| 邮箱服务商 | 推荐协议 | 备选协议 | 说明 |
|-----------|---------|---------|------|
| Gmail | Gmail API | IMAP | API 性能更好，功能更强 |
| Outlook/Hotmail | Graph API | IMAP | API 性能更好，功能更强 |
| iCloud | IMAP | - | 仅支持 IMAP |
| QQ 邮箱 | IMAP | POP3 | IMAP 功能更强 |
| 163 邮箱 | IMAP | POP3 | IMAP 功能更强 |
| 其他邮箱 | IMAP | POP3 | 通用协议 |

### 协议对比

| 特性 | Gmail API | Graph API | IMAP | POP3 |
|-----|----------|-----------|------|------|
| 增量同步 | ✅ | ✅ | ✅ | ❌ |
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| 功能丰富度 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| 通用性 | ❌ | ❌ | ✅ | ✅ |
| 代理支持 | ✅ | ✅ | ✅ | ⚠️ |
| 认证方式 | OAuth2 | OAuth2 | 密码/OAuth2 | 密码 |

---

## 测试建议

### 单元测试
```bash
# 测试 POP3 适配器
go test -v ./internal/adapter -run TestPOP3Adapter

# 测试 Gmail 适配器
go test -v ./internal/adapter -run TestGmailAdapter

# 测试 Graph 适配器
go test -v ./internal/adapter -run TestGraphAdapter
```

### 集成测试
建议使用真实邮箱账户进行集成测试：
1. 创建测试邮箱账户
2. 配置 OAuth2 凭证（Gmail/Outlook）
3. 运行集成测试脚本
4. 验证邮件拉取、解析、状态同步等功能

---

## 后续优化建议

### 短期优化
1. **POP3 代理支持**：考虑使用其他 POP3 库或自己实现客户端
2. **错误重试机制**：实现指数退避重试算法
3. **连接池管理**：复用 HTTP 连接，提高性能
4. **日志记录**：添加详细的调试日志

### 中期优化
1. **API 降级逻辑**：Gmail API/Graph API 失败时自动降级到 IMAP
2. **Token 自动刷新**：实现 OAuth2 Token 自动刷新机制
3. **批量操作**：支持批量拉取邮件，提高效率
4. **增量同步优化**：使用 IMAP IDLE 或 Webhook 实现实时推送

### 长期优化
1. **Exchange 协议支持**：支持企业级 Exchange 服务器
2. **JMAP 协议支持**：支持新一代邮件协议 JMAP
3. **邮件发送功能**：实现 SMTP 发送邮件
4. **双向状态同步**：支持本地状态回写到源邮箱

---

## 相关文档

- [技术栈文档](./tech.md)
- [项目结构文档](./structure.md)
- [任务清单](./tasks.md)
- [代码规范](../../steering/code-conventions.md)

---

## 总结

本次实现完成了 FusionMail 项目的三个核心邮箱协议适配器，为项目提供了：

✅ **多协议支持**：覆盖主流邮箱服务商（Gmail、Outlook、QQ、163 等）  
✅ **统一接口**：上层服务无需关心具体协议实现  
✅ **高性能**：优先使用官方 API，性能优于传统协议  
✅ **安全性**：支持 OAuth2 认证，无需存储密码  
✅ **可扩展**：易于添加新的协议支持  

下一步工作：
1. 实现适配器单元测试（任务 3.5）
2. 在同步引擎中集成适配器（任务 4.3）
3. 实现 API 降级逻辑（任务 3.2/3.3）
4. 完善错误处理和日志记录

**实现者**: Kiro AI Assistant  
**审核状态**: 待审核  
**版本**: v1.0
