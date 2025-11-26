# 邮件协议代理支持需求文档

## 概述

本文档描述了 FusionMail 项目中各邮件协议对 HTTP 代理和 SOCKS5 代理的支持需求。由于企业网络环境的特殊性，许多组织需要通过代理服务器访问外部邮件服务，因此代理支持是邮件服务集成的重要功能。

## 当前代理支持状态

### 已实现代理支持的协议

| 协议 | 适配器 | HTTP代理 | SOCKS5代理 | 状态 |
|------|--------|----------|------------|------|
| Gmail API | `GmailAdapter` | ✅ 支持 | ❌ 不支持 | 已实现 |
| Microsoft Graph | `GraphAdapter` | ✅ 支持 | ❌ 不支持 | 已实现 |
| POP3 | `POP3Adapter` | ❌ 不支持 | ✅ 支持 | 已实现 |

### 待实现代理支持的协议

| 协议 | 适配器 | HTTP代理 | SOCKS5代理 | 状态 |
|------|--------|----------|------------|------|
| IMAP | `IMAPAdapter` | ⚠️ 待实现 | ✅ 待实现 | **需求** |

## 代理配置结构

当前已定义的代理配置结构如下（`backend/internal/adapter/adapter.go`）：

```go
type ProxyConfig struct {
    Enabled  bool   // 是否启用代理
    Type     string // 代理类型：http/socks5
    Host     string // 代理服务器地址
    Port     int    // 代理端口
    Username string // 代理用户名（可选）
    Password string // 代理密码（可选）
}
```

## 技术实现需求

### 1. IMAP 协议代理支持

#### 需求背景
IMAP 协议本身不直接支持代理，但可以通过网络层代理实现：
- **SOCKS5**: 完全支持，实现简单
- **HTTP CONNECT**: 有限支持，需要代理服务器支持 CONNECT 方法

#### 实现方案

##### SOCKS5 代理实现
```go
// 1. 创建 SOCKS5 拨号器
auth := &proxy.Auth{}
if config.Proxy.Username != "" {
    auth.User = config.Proxy.Username
    auth.Password = config.Proxy.Password
}
dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)

// 2. 通过代理建立 TCP 连接
rawConn, err := dialer.Dial("tcp", serverAddr)

// 3. 升级为 TLS 连接（如果需要）
tlsConn := tls.Client(rawConn, tlsConfig)

// 4. 创建 IMAP 客户端
client := imapclient.New(tlsConn)
```

##### HTTP CONNECT 代理实现
```go
// 1. 连接到 HTTP 代理
proxyConn, err := net.Dial("tcp", proxyAddr)

// 2. 发送 CONNECT 请求
fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", serverAddr, serverAddr)

// 3. 验证响应
resp, _ := http.ReadResponse(bufio.NewReader(proxyConn), nil)
if resp.StatusCode != 200 {
    return errors.New("proxy refused connection")
}

// 4. 升级为 TLS（如果需要）
tlsConn := tls.Client(proxyConn, tlsConfig)

// 5. 创建 IMAP 客户端
client := imapclient.New(tlsConn)
```

#### 代码修改位置
- **文件**: `backend/internal/adapter/imap.go`
- **函数**: `Connect(ctx context.Context) error`
- **修改点**: 替换 `imapclient.DialTLS()` 调用，使用自定义连接建立逻辑

### 2. Gmail API 代理支持增强

#### 当前状态
- 仅支持 HTTP/HTTPS 代理
- 不支持 SOCKS5 代理

#### 需求
- 添加 SOCKS5 代理支持
- 统一代理处理逻辑

#### 实现方案
```go
// 对于 SOCKS5 代理，创建自定义拨号器
if config.Proxy.Type == "socks5" {
    dialer, _ := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
    transport := &http.Transport{
        Dial: dialer.Dial,
    }
    httpClient.Transport = transport
}
```

### 3. Microsoft Graph API 代理支持增强

#### 当前状态
- 仅支持 HTTP/HTTPS 代理
- 不支持 SOCKS5 代理

#### 需求
- 添加 SOCKS5 代理支持
- 与 Gmail API 保持一致的处理逻辑

### 4. POP3 代理支持增强

#### 当前状态
- 仅支持 SOCKS5 代理
- 不支持 HTTP 代理

#### 需求
- 添加 HTTP CONNECT 代理支持
- 统一代理配置处理

## 代理类型优先级

### 推荐优先级
1. **SOCKS5**: 首选方案，协议无关，性能优秀
2. **HTTP CONNECT**: 备选方案，需要代理服务器配合

### 选择依据
- **SOCKS5**: 对应用层协议透明，无数据包修改，性能更好
- **HTTP CONNECT**: 可能被代理服务器限制，实现相对复杂

## 配置管理需求

### 用户界面需求
1. **代理配置界面**
   - 代理类型选择（HTTP/SOCKS5）
   - 代理服务器地址和端口输入
   - 认证信息输入（用户名/密码）
   - 代理测试功能

2. **账户配置集成**
   - 每个邮件账户可独立配置代理
   - 支持使用系统代理设置
   - 代理配置保存和验证

### API 需求
1. **代理配置 API**
   - 代理配置 CRUD 操作
   - 代理连接测试
   - 代理状态检查

2. **账户代理绑定**
   - 账户与代理配置的关联
   - 代理配置的启用/禁用

## 测试需求

### 单元测试
1. **代理连接测试**
   - SOCKS5 代理连接测试
   - HTTP CONNECT 代理连接测试
   - 代理认证测试

2. **协议功能测试**
   - 各协议通过代理的基本功能验证
   - 代理切换测试
   - 错误处理测试

### 集成测试
1. **真实代理环境测试**
   - 常见代理软件测试（Squid、Shadowsocks、V2Ray等）
   - 企业代理环境测试

2. **性能测试**
   - 代理连接性能
   - 数据传输性能
   - 并发连接测试

## 安全考虑

### 代理认证安全
1. 代理密码加密存储
2. 支持代理认证失败重试
3. 代理连接超时处理

### 数据传输安全
1. 代理隧道加密（TLS）
2. 端到端加密保持
3. 代理服务器证书验证

## 错误处理

### 代理连接错误
- 代理服务器不可达
- 代理认证失败
- 代理拒绝连接

### 协议特定错误
- IMAP 服务器连接失败
- TLS 握手失败
- 协议协商失败

## 实施计划

### 第一阶段：IMAP SOCKS5 代理支持
1. 实现 IMAP SOCKS5 代理连接
2. 添加代理配置验证
3. 单元测试和集成测试

### 第二阶段：IMAP HTTP CONNECT 代理支持
1. 实现 IMAP HTTP CONNECT 代理连接
2. 完善错误处理机制
3. 性能优化和测试

### 第三阶段：其他协议代理增强
1. Gmail API SOCKS5 支持
2. Microsoft Graph SOCKS5 支持
3. POP3 HTTP CONNECT 支持

### 第四阶段：用户界面和 API
1. 代理配置管理界面
2. 代理配置 API
3. 用户使用文档

## 相关文件

- `backend/internal/adapter/adapter.go` - 代理配置结构定义
- `backend/internal/adapter/imap.go` - IMAP 适配器（待修改）
- `backend/internal/adapter/gmail.go` - Gmail 适配器（参考实现）
- `backend/internal/adapter/graph.go` - Graph 适配器（参考实现）
- `backend/internal/adapter/pop3.go` - POP3 适配器（参考实现）

## 参考资源

- [SOCKS5 协议规范](https://tools.ietf.org/html/rfc1928)
- [HTTP CONNECT 方法](https://tools.ietf.org/html/rfc7231#section-4.3.6)
- [go-imap 文档](https://github.com/emersion/go-imap)
- [golang.org/x/net/proxy](https://pkg.go.dev/golang.org/x/net/proxy)

---

**创建日期**: 2025-11-26
**最后更新**: 2025-11-26
**文档状态**: 需求分析完成，待实施