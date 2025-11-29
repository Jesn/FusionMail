# HTTP/HTTPS 部署指南

## 问题说明

### 症状
在局域网通过 HTTP 访问 FusionMail 时，浏览器报错：
```
ERR_SSL_PROTOCOL_ERROR
```

HTML 页面可以加载，但 CSS/JS 等静态资源无法加载。

### 根本原因

FusionMail 的 CSP（Content Security Policy）中间件默认包含 `upgrade-insecure-requests` 指令，该指令会告诉浏览器自动将所有 HTTP 请求升级到 HTTPS。

在纯 HTTP 环境下，这会导致：
- ✅ 第一次 HTML 请求成功（HTTP）
- ❌ 页面内的资源请求被升级到 HTTPS 失败

### 解决方案

**v1.0.1 版本已修复**：CSP 中间件现在会自动检测访问协议，仅在 HTTPS 环境下启用 `upgrade-insecure-requests`。

---

## 部署场景

### 场景一：纯 HTTP 部署（开发/内网环境）

**适用场景**：
- 本地开发环境
- 内网/局域网部署
- 无需 HTTPS 的场景

**配置**：
```yaml
# docker-compose.yml
services:
  app:
    ports:
      - "3333:3333"  # 或其他端口
    environment:
      COOKIE_SECURE: false  # 可选，Cookie 不强制 HTTPS
```

**访问**：
```
http://192.168.x.x:3333
```

**说明**：
- CSP 自动检测为 HTTP，不会启用 `upgrade-insecure-requests`
- 所有资源正常通过 HTTP 加载

---

### 场景二：Nginx 反向代理 + HTTPS

**适用场景**：
- 生产环境
- 需要 HTTPS 的场景
- 使用自签名证书或 Let's Encrypt

**架构**：
```
浏览器 (HTTPS:443) → Nginx → FusionMail (HTTP:3333)
```

**Nginx 配置示例**：
```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://fusionmail:3333;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;  # 重要！告诉后端使用的是 HTTPS
    }

    # SSE 端点特殊配置
    location /api/v1/events {
        proxy_pass http://fusionmail:3333;
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_read_timeout 90s;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**说明**：
- Nginx 设置 `X-Forwarded-Proto: https` 头
- FusionMail 检测到该头，自动启用 `upgrade-insecure-requests`
- 浏览器和 Nginx 之间使用 HTTPS
- Nginx 和 FusionMail 之间使用 HTTP（内网安全）

---

### 场景三：直接 HTTPS 部署

**适用场景**：
- 不使用反向代理
- 直接暴露 FusionMail 到公网

**配置**：
需要修改 FusionMail 代码，添加 TLS 配置（当前版本不支持）。

**推荐**：使用场景二（Nginx 反向代理）更灵活和安全。

---

## 常见问题

### Q1: 为什么本地 127.0.0.1 正常，但局域网 IP 不行？

**A**: 浏览器对 localhost/127.0.0.1 有特殊处理，不会强制升级到 HTTPS。但对其他 IP 地址，如果之前访问过 HTTPS 站点，可能会记住 HSTS 策略。

### Q2: 清除了浏览器缓存还是不行？

**A**: 问题不在浏览器缓存，而在代码的 CSP 策略。升级到 v1.0.1+ 版本即可解决。

### Q3: 如何生成自签名证书？

**A**: 使用 OpenSSL：
```bash
# 生成私钥
openssl genrsa -out key.pem 2048

# 生成证书（有效期 365 天）
openssl req -new -x509 -key key.pem -out cert.pem -days 365 \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=FusionMail/CN=192.168.2.200"
```

### Q4: 自签名证书浏览器提示不安全？

**A**: 这是正常的。可以：
1. 点击"高级" → "继续访问"（临时）
2. 将证书添加到系统信任列表（永久）
3. 使用 Let's Encrypt 获取免费的受信任证书（需要域名）

### Q5: 如何判断当前是 HTTP 还是 HTTPS？

**A**: 查看响应头中的 `Content-Security-Policy`：
- 包含 `upgrade-insecure-requests` → HTTPS 环境
- 不包含 → HTTP 环境

---

## 版本历史

### v1.0.1 (2025-11-29)
- ✅ 修复：CSP 中间件根据访问协议动态启用 `upgrade-insecure-requests`
- ✅ 支持：HTTP 和 HTTPS 环境自动适配
- ✅ 支持：Nginx 反向代理场景（通过 `X-Forwarded-Proto` 头检测）

### v1.0.0
- ❌ 问题：CSP 始终启用 `upgrade-insecure-requests`，导致 HTTP 环境资源加载失败

---

## 推荐配置

### 开发环境
- 使用 HTTP
- 端口：3333 或 4444
- 访问：`http://localhost:3333`

### 测试环境（局域网）
- 使用 HTTP
- 配置 hosts 文件使用域名
- 访问：`http://fusionmail.local:3333`

### 生产环境
- 使用 Nginx + HTTPS
- 配置 Let's Encrypt 自动续期
- 访问：`https://your-domain.com`

---

## 技术细节

### CSP upgrade-insecure-requests 指令

**作用**：告诉浏览器自动将页面内的所有 HTTP 请求升级到 HTTPS。

**适用场景**：
- ✅ HTTPS 站点，但页面内有 HTTP 资源
- ✅ 逐步迁移到 HTTPS 的过渡期

**不适用场景**：
- ❌ 纯 HTTP 站点（会导致资源加载失败）

### 检测逻辑

```go
// 检查是否为 HTTPS 环境
if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
    cspDirectives = append(cspDirectives, "upgrade-insecure-requests")
}
```

**检测条件**：
1. `c.Request.TLS != nil`：直接 HTTPS 连接
2. `X-Forwarded-Proto == "https"`：通过 HTTPS 反向代理

---

## 相关文档

- [CSP 规范](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
- [upgrade-insecure-requests](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/upgrade-insecure-requests)
- [Nginx 反向代理配置](https://nginx.org/en/docs/http/ngx_http_proxy_module.html)
