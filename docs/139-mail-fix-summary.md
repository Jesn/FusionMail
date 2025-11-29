# 139 邮箱 TLS 问题修复总结

## 📋 问题描述

139 邮箱（中国移动）连接失败，错误信息：`remote error: tls: handshake failure`

## 🔍 根本原因

1. **TLS 配置不兼容** - 139 邮箱服务器需要特定的 `CipherSuites` 配置
2. **Provider 缺少加密字段** - 无法为不同邮箱服务商预设正确的加密配置
3. **139 邮箱未在 Provider 列表** - 用户无法直接选择 139 邮箱

## ✅ 已完成的修复

### 1. 修复 IMAP 适配器 TLS 配置

**文件**: `backend/internal/adapter/imap.go`

为 139 邮箱添加了兼容的 `CipherSuites` 配置：

```go
if strings.Contains(host, "139.com") {
    tlsConfig.InsecureSkipVerify = true
    tlsConfig.MinVersion = tls.VersionTLS10
    tlsConfig.MaxVersion = tls.VersionTLS12
    tlsConfig.CipherSuites = []uint16{
        tls.TLS_RSA_WITH_AES_128_CBC_SHA,
        tls.TLS_RSA_WITH_AES_256_CBC_SHA,
        // ... 更多兼容的加密套件
    }
}
```

### 2. 添加中国邮箱服务商到 Provider 列表

**文件**: `backend/migrations/014_add_139_provider.sql`

新增的邮箱服务商：
- 139 邮箱（中国移动）- imap.139.com:993
- 126 邮箱（网易）- imap.126.com:993
- 189 邮箱（中国电信）- imap.189.cn:993

### 3. 为 Provider 模型添加加密字段

**文件**: `backend/internal/model/provider.go`

新增字段：
- `imap_encryption` - IMAP 加密方式 (ssl/starttls/none)
- `pop3_encryption` - POP3 加密方式 (ssl/starttls/none)
- `smtp_encryption` - SMTP 加密方式 (ssl/starttls/none)

**数据库迁移**: `backend/migrations/015_add_encryption_to_providers.sql`

### 4. 更新前端类型和表单

**更新的文件**:
- `frontend/src/types/provider.ts` - 添加加密字段类型
- `frontend/src/services/systemService.ts` - 更新 Provider 接口
- `frontend/src/pages/ProvidersPage.tsx` - 添加加密方式选择器
- `frontend/src/components/account/AccountForm.tsx` - 从 Provider 继承加密配置
- `frontend/src/hooks/useProviders.ts` - 添加 139/126/189 邮箱域名映射

## 📊 当前 Provider 列表

| 名称 | 显示名称 | IMAP 服务器 | 加密方式 |
|-----|---------|------------|---------|
| gmail | Gmail | imap.gmail.com:993 | SSL |
| outlook | Outlook / Hotmail | outlook.office365.com:993 | SSL |
| icloud | iCloud Mail | imap.mail.me.com:993 | SSL |
| qq | QQ 邮箱 | imap.qq.com:993 | SSL |
| 163 | 163 邮箱 | imap.163.com:993 | SSL |
| 139 | 139 邮箱 (中国移动) | imap.139.com:993 | SSL |
| 126 | 126 邮箱 (网易) | imap.126.com:993 | SSL |
| 189 | 189 邮箱 (中国电信) | imap.189.cn:993 | SSL |
| generic | 通用邮箱 | - | SSL |

## 🎯 使用方法

### 添加 139 邮箱账户

1. 访问 `/accounts` 页面
2. 点击"添加账户"
3. 输入邮箱地址（如 `xxx@139.com`）
4. 系统自动识别为 139 邮箱并填充配置
5. 输入授权码（非登录密码）
6. 点击添加

### 管理 Provider 配置

1. 访问 `/providers` 页面
2. 可以编辑现有 Provider 的加密配置
3. 可以新增自定义 Provider

## 🔧 技术细节

### 139 邮箱连接成功的关键配置

```go
tlsConfig := &tls.Config{
    ServerName:         "imap.139.com",
    InsecureSkipVerify: true,             // 跳过证书验证
    MinVersion:         tls.VersionTLS10, // 支持 TLS 1.0
    MaxVersion:         tls.VersionTLS12, // 最高 TLS 1.2
    CipherSuites: []uint16{               // 关键：指定兼容的加密套件
        tls.TLS_RSA_WITH_AES_128_CBC_SHA,
        tls.TLS_RSA_WITH_AES_256_CBC_SHA,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        // ...
    },
}
```

### 加密方式说明

| 加密方式 | 端口 | 说明 |
|---------|------|------|
| SSL/TLS | 993 (IMAP), 995 (POP3) | 直接建立加密连接（推荐） |
| STARTTLS | 143 (IMAP), 110 (POP3) | 先明文连接，再升级为加密 |
| 无加密 | 143 (IMAP), 110 (POP3) | 不推荐，仅用于测试 |

## 📝 相关文件

- `backend/internal/adapter/imap.go` - IMAP 适配器
- `backend/internal/model/provider.go` - Provider 模型
- `backend/migrations/014_add_139_provider.sql` - 添加中国邮箱服务商
- `backend/migrations/015_add_encryption_to_providers.sql` - 添加加密字段
- `frontend/src/pages/ProvidersPage.tsx` - Provider 管理页面
- `frontend/src/components/account/AccountForm.tsx` - 账户表单
- `docs/139-mail-integration-analysis.md` - 详细分析报告

---

**修复时间**: 2024-11-29  
**状态**: ✅ 已完成
