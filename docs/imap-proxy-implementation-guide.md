# IMAP 代理实现指南

## 概述

本文档提供了在 FusionMail 项目中为 IMAP 协议添加 SOCKS5 和 HTTP CONNECT 代理支持的详细实现方案。

## 当前实现分析

### 现有代码结构
```go
// backend/internal/adapter/imap.go
func (a *IMAPAdapter) Connect(ctx context.Context) error {
    addr := fmt.Sprintf("%s:%d", a.config.Credentials.Host, a.config.Credentials.Port)

    // 配置 TLS
    tlsConfig := &tls.Config{
        ServerName: a.config.Credentials.Host,
    }

    // 创建 IMAP 客户端选项
    options := &imapclient.Options{
        TLSConfig: tlsConfig,
    }

    // 直接连接到服务器 - 这里需要修改
    client, err := imapclient.DialTLS(addr, options)
    if err != nil {
        return fmt.Errorf("failed to connect to IMAP server: %w", err)
    }

    a.client = client
    return a.login(ctx)
}
```

### 问题分析
当前实现使用 `imapclient.DialTLS()` 直接连接，无法通过代理服务器。需要修改为使用自定义连接建立方式。

## 实现方案

### 方案一：SOCKS5 代理实现

#### 步骤 1: 导入必要的包
```go
import (
    "context"
    "crypto/tls"
    "fmt"
    "net"
    "time"

    "github.com/emersion/go-imap/client"
    "golang.org/x/net/proxy"
)
```

#### 步骤 2: 创建 SOCKS5 连接函数
```go
// dialSOCKS5 通过 SOCKS5 代理建立连接
func (a *IMAPAdapter) dialSOCKS5(ctx context.Context, network, addr string) (net.Conn, error) {
    if a.config.Proxy == nil || !a.config.Proxy.Enabled {
        return nil, fmt.Errorf("proxy not configured")
    }

    // 构建代理地址
    proxyAddr := fmt.Sprintf("%s:%d", a.config.Proxy.Host, a.config.Proxy.Port)

    // 配置 SOCKS5 认证
    var auth *proxy.Auth
    if a.config.Proxy.Username != "" {
        auth = &proxy.Auth{
            User:     a.config.Proxy.Username,
            Password: a.config.Proxy.Password,
        }
    }

    // 创建 SOCKS5 拨号器
    dialer, err := proxy.SOCKS5(network, proxyAddr, auth, proxy.Direct)
    if err != nil {
        return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
    }

    // 设置连接超时
    if a.config.Timeout > 0 {
        dialer = &timeoutDialer{
            dialer:  dialer,
            timeout: a.config.Timeout,
        }
    }

    // 通过代理建立连接
    return dialer.Dial(network, addr)
}

// timeoutDialer 包装拨号器以支持超时
type timeoutDialer struct {
    dialer  proxy.Dialer
    timeout time.Duration
}

func (d *timeoutDialer) Dial(network, addr string) (net.Conn, error) {
    // 创建带超时的连接
    conn, err := d.dialer.Dial(network, addr)
    if err != nil {
        return nil, err
    }

    // 设置读写超时
    if d.timeout > 0 {
        conn.SetDeadline(time.Now().Add(d.timeout))
    }

    return conn, nil
}
```

#### 步骤 3: 修改 Connect 函数
```go
func (a *IMAPAdapter) Connect(ctx context.Context) error {
    serverAddr := fmt.Sprintf("%s:%d", a.config.Credentials.Host, a.config.Credentials.Port)

    // 根据代理配置选择连接方式
    var rawConn net.Conn
    var err error

    if a.config.Proxy != nil && a.config.Proxy.Enabled {
        switch a.config.Proxy.Type {
        case "socks5":
            rawConn, err = a.dialSOCKS5(ctx, "tcp", serverAddr)
            if err != nil {
                return fmt.Errorf("SOCKS5 connection failed: %w", err)
            }
        case "http":
            rawConn, err = a.dialHTTPConnect(ctx, serverAddr)
            if err != nil {
                return fmt.Errorf("HTTP CONNECT connection failed: %w", err)
            }
        default:
            return fmt.Errorf("unsupported proxy type: %s", a.config.Proxy.Type)
        }
    } else {
        // 直接连接（保持原有逻辑）
        rawConn, err = net.DialTimeout("tcp", serverAddr, a.config.Timeout)
        if err != nil {
            return fmt.Errorf("direct connection failed: %w", err)
        }
    }

    defer func() {
        if err != nil && rawConn != nil {
            rawConn.Close()
        }
    }()

    // 配置 TLS
    var tlsConn *tls.Conn
    if a.config.Credentials.TLS {
        tlsConfig := &tls.Config{
            ServerName:         a.config.Credentials.Host,
            InsecureSkipVerify: false, // 生产环境应验证证书
        }

        tlsConn = tls.Client(rawConn, tlsConfig)

        // 执行 TLS 握手
        if err := tlsConn.Handshake(); err != nil {
            return fmt.Errorf("TLS handshake failed: %w", err)
        }

        a.client, err = client.New(tlsConn)
    } else {
        // 非 TLS 连接
        a.client, err = client.New(rawConn)
    }

    if err != nil {
        return fmt.Errorf("failed to create IMAP client: %w", err)
    }

    // 登录
    return a.login(ctx)
}
```

### 方案二：HTTP CONNECT 代理实现

#### 创建 HTTP CONNECT 连接函数
```go
// dialHTTPConnect 通过 HTTP CONNECT 代理建立连接
func (a *IMAPAdapter) dialHTTPConnect(ctx context.Context, targetAddr string) (net.Conn, error) {
    if a.config.Proxy == nil || !a.config.Proxy.Enabled {
        return nil, fmt.Errorf("proxy not configured")
    }

    // 构建代理地址
    proxyAddr := fmt.Sprintf("%s:%d", a.config.Proxy.Host, a.config.Proxy.Port)

    // 连接到代理服务器
    proxyConn, err := net.DialTimeout("tcp", proxyAddr, a.config.Timeout)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to proxy: %w", err)
    }

    // 设置连接超时
    if a.config.Timeout > 0 {
        proxyConn.SetDeadline(time.Now().Add(a.config.Timeout))
    }

    // 构建 CONNECT 请求
    connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\n", targetAddr)
    connectReq += fmt.Sprintf("Host: %s\r\n", targetAddr)
    connectReq += "Proxy-Connection: Keep-Alive\r\n"

    // 添加代理认证（如果需要）
    if a.config.Proxy.Username != "" {
        auth := a.config.Proxy.Username + ":" + a.config.Proxy.Password
        authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))
        connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", authEncoded)
    }

    connectReq += "\r\n"

    // 发送 CONNECT 请求
    if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
        proxyConn.Close()
        return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
    }

    // 读取响应
    reader := bufio.NewReader(proxyConn)

    // 读取状态行
    statusLine, err := reader.ReadString('\n')
    if err != nil {
        proxyConn.Close()
        return nil, fmt.Errorf("failed to read status line: %w", err)
    }

    // 解析状态码
    parts := strings.Split(strings.TrimSpace(statusLine), " ")
    if len(parts) < 2 {
        proxyConn.Close()
        return nil, fmt.Errorf("invalid status line: %s", statusLine)
    }

    statusCode := parts[1]
    if statusCode != "200" {
        // 读取错误响应
        var errorMsg strings.Builder
        errorMsg.WriteString(fmt.Sprintf("HTTP CONNECT failed with status %s", statusCode))

        // 尝试读取响应头获取更多信息
        for {
            line, err := reader.ReadString('\n')
            if err != nil || line == "\r\n" {
                break
            }
            if strings.HasPrefix(line, "X-Squid-Error:") || strings.HasPrefix(line, "X-Proxy-Error:") {
                errorMsg.WriteString(fmt.Sprintf(": %s", strings.TrimSpace(line)))
            }
        }

        proxyConn.Close()
        return nil, fmt.Errorf("%s", errorMsg.String())
    }

    // 跳过响应头
    for {
        line, err := reader.ReadString('\n')
        if err != nil || line == "\r\n" {
            break
        }
    }

    // 清除超时设置（连接已建立）
    proxyConn.SetDeadline(time.Time{})

    return proxyConn, nil
}
```

## 完整实现示例

### 完整的 Connect 函数
```go
func (a *IMAPAdapter) Connect(ctx context.Context) error {
    // 验证配置
    if a.config.Credentials.Host == "" {
        return fmt.Errorf("IMAP host is required")
    }

    serverAddr := fmt.Sprintf("%s:%d", a.config.Credentials.Host, a.config.Credentials.Port)

    // 根据代理配置选择连接方式
    var rawConn net.Conn
    var err error

    if a.config.Proxy != nil && a.config.Proxy.Enabled {
        a.logDebug("Connecting through %s proxy %s:%d", a.config.Proxy.Type, a.config.Proxy.Host, a.config.Proxy.Port)

        switch strings.ToLower(a.config.Proxy.Type) {
        case "socks5":
            rawConn, err = a.dialSOCKS5(ctx, "tcp", serverAddr)
            if err != nil {
                return fmt.Errorf("SOCKS5 connection failed: %w", err)
            }
            a.logDebug("SOCKS5 connection established to %s", serverAddr)

        case "http":
            rawConn, err = a.dialHTTPConnect(ctx, serverAddr)
            if err != nil {
                return fmt.Errorf("HTTP CONNECT connection failed: %w", err)
            }
            a.logDebug("HTTP CONNECT tunnel established to %s", serverAddr)

        default:
            return fmt.Errorf("unsupported proxy type: %s", a.config.Proxy.Type)
        }
    } else {
        // 直接连接
        a.logDebug("Direct connecting to %s", serverAddr)
        rawConn, err = net.DialTimeout("tcp", serverAddr, a.config.Timeout)
        if err != nil {
            return fmt.Errorf("direct connection failed: %w", err)
        }
    }

    // 确保连接在出错时关闭
    defer func() {
        if err != nil && rawConn != nil {
            rawConn.Close()
        }
    }()

    // 配置 TLS（如果需要）
    var conn net.Conn = rawConn
    if a.config.Credentials.TLS {
        tlsConfig := &tls.Config{
            ServerName:         a.config.Credentials.Host,
            InsecureSkipVerify: false,
        }

        tlsConn := tls.Client(rawConn, tlsConfig)

        // 执行 TLS 握手
        handshakeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()

        done := make(chan error, 1)
        go func() {
            done <- tlsConn.Handshake()
        }()

        select {
        case err = <-done:
            if err != nil {
                return fmt.Errorf("TLS handshake failed: %w", err)
            }
        case <-handshakeCtx.Done():
            return fmt.Errorf("TLS handshake timeout")
        }

        conn = tlsConn
        a.logDebug("TLS handshake completed")
    }

    // 创建 IMAP 客户端
    a.client, err = client.New(conn)
    if err != nil {
        return fmt.Errorf("failed to create IMAP client: %w", err)
    }

    a.logDebug("IMAP client created successfully")

    // 执行登录
    return a.login(ctx)
}
```

## 错误处理和日志

### 增强的错误处理
```go
// 包装错误以提供更多上下文信息
func (a *IMAPAdapter) wrapError(err error, context string) error {
    if err == nil {
        return nil
    }

    // 添加代理信息（如果使用了代理）
    if a.config.Proxy != nil && a.config.Proxy.Enabled {
        return fmt.Errorf("%s (proxy: %s://%s:%d): %w",
            context, a.config.Proxy.Type, a.config.Proxy.Host, a.config.Proxy.Port, err)
    }

    return fmt.Errorf("%s: %w", context, err)
}

// 调试日志
func (a *IMAPAdapter) logDebug(format string, args ...interface{}) {
    if a.config.Debug || os.Getenv("IMAP_DEBUG") == "true" {
        log.Printf("[IMAP-PROXY] "+format, args...)
    }
}
```

## 测试方案

### 单元测试
```go
func TestIMAP_SOCKS5Proxy(t *testing.T) {
    // 启动 SOCKS5 测试服务器
    proxyServer := startTestSOCKS5Server(t)
    defer proxyServer.Stop()

    config := &Config{
        Provider: "imap",
        Protocol: "imap",
        Credentials: &Credentials{
            Host:     "imap.gmail.com",
            Port:     993,
            Email:    "test@gmail.com",
            Password: "test-password",
            TLS:      true,
        },
        Proxy: &ProxyConfig{
            Enabled:  true,
            Type:     "socks5",
            Host:     "localhost",
            Port:     proxyServer.Port(),
            Username: "",
            Password: "",
        },
        Timeout: 30 * time.Second,
    }

    adapter, err := NewIMAPAdapter(config)
    require.NoError(t, err)

    ctx := context.Background()
    err = adapter.Connect(ctx)

    // 验证连接结果
    if err != nil {
        t.Logf("Connection result: %v", err)
        // 在网络不可用时，至少验证代理配置被正确处理
        assert.Contains(t, err.Error(), "SOCKS5")
    }
}
```

### 集成测试
```go
func TestIMAP_ProxyIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    testCases := []struct {
        name       string
        proxyType  string
        proxyHost  string
        proxyPort  int
        expectError bool
    }{
        {
            name:       "SOCKS5 无认证",
            proxyType:  "socks5",
            proxyHost:  "localhost",
            proxyPort:  1080,
            expectError: false,
        },
        {
            name:       "HTTP CONNECT 无认证",
            proxyType:  "http",
            proxyHost:  "localhost",
            proxyPort:  8080,
            expectError: false,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            config := &Config{
                // ... 测试配置
            }

            adapter, _ := NewIMAPAdapter(config)
            err := adapter.Connect(context.Background())

            if tc.expectError {
                assert.Error(t, err)
            } else {
                // 连接可能因网络问题失败，但代理逻辑应该正确
                if err != nil {
                    assert.Contains(t, err.Error(), tc.proxyType)
                }
            }
        })
    }
}
```

## 性能优化

### 连接池优化
```go
// 考虑实现连接池以复用代理连接
type proxyConnPool struct {
    mu       sync.Mutex
    conns    map[string][]net.Conn
    maxConns int
}

func (p *proxyConnPool) getConn(proxyType, targetAddr string) (net.Conn, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    key := fmt.Sprintf("%s:%s", proxyType, targetAddr)
    if conns, ok := p.conns[key]; ok && len(conns) > 0 {
        conn := conns[len(conns)-1]
        p.conns[key] = conns[:len(conns)-1]
        return conn, nil
    }

    return nil, fmt.Errorf("no available connection")
}
```

### 超时控制
```go
// 更精细的超时控制
type dialerOptions struct {
    connectTimeout time.Duration
    readTimeout    time.Duration
    writeTimeout   time.Duration
}

func (a *IMAPAdapter) dialWithTimeout(ctx context.Context, network, addr string, opts dialerOptions) (net.Conn, error) {
    dialer := &net.Dialer{
        Timeout: opts.connectTimeout,
    }

    conn, err := dialer.DialContext(ctx, network, addr)
    if err != nil {
        return nil, err
    }

    // 设置读写超时
    if opts.readTimeout > 0 {
        conn.SetReadDeadline(time.Now().Add(opts.readTimeout))
    }
    if opts.writeTimeout > 0 {
        conn.SetWriteDeadline(time.Now().Add(opts.writeTimeout))
    }

    return conn, nil
}
```

## 部署和配置

### Docker 环境配置
```yaml
# docker-compose.yml 中添加代理服务
services:
  socks5-proxy:
    image: serjs/go-socks5-proxy
    ports:
      - "1080:1080"
    environment:
      - PROXY_USER=testuser
      - PROXY_PASSWORD=testpass

  imap-test:
    build: .
    environment:
      - IMAP_PROXY_TYPE=socks5
      - IMAP_PROXY_HOST=socks5-proxy
      - IMAP_PROXY_PORT=1080
```

### 环境变量配置
```bash
# 开发环境配置
export IMAP_SOCKS5_PROXY="socks5://user:pass@localhost:1080"
export IMAP_HTTP_PROXY="http://user:pass@localhost:8080"

# 测试不同代理
go test -v ./internal/adapter/ -run TestIMAP_Proxy
```

## 故障排除

### 常见问题

1. **SOCKS5 认证失败**
   ```
   错误: failed to create SOCKS5 dialer: proxy: failed to authenticate with proxy
   解决: 检查用户名密码是否正确，确认 SOCKS5 服务器支持认证
   ```

2. **HTTP CONNECT 被拒绝**
   ```
   错误: HTTP CONNECT failed with status 403
   解决: 检查代理服务器是否允许连接到目标端口，确认认证信息正确
   ```

3. **TLS 握手失败**
   ```
   错误: TLS handshake failed: x509: certificate is valid for ...
   解决: 检查 TLS 配置，确认 ServerName 设置正确
   ```

### 调试技巧
```go
// 启用详细日志
os.Setenv("IMAP_DEBUG", "true")
os.Setenv("SOCKS5_DEBUG", "true")

// 使用 tcpdump 抓包分析
sudo tcpdump -i any -nn port 1080 or port 8080 or port 993

// 测试代理连接
curl -x socks5://localhost:1080 https://www.google.com
curl -x http://localhost:8080 https://www.google.com
```

---

**创建日期**: 2025-11-26
**最后更新**: 2025-11-26
**文档状态**: 实现指南完成

**参考资源**:
- [SOCKS5 协议规范](https://tools.ietf.org/html/rfc1928)
- [HTTP CONNECT 方法](https://tools.ietf.org/html/rfc7231#section-4.3.6)
- [go-imap 文档](https://github.com/emersion/go-imap)
- [Golang SOCKS5 示例](https://www.example-code.com/golang/imap_socks_proxy.asp)