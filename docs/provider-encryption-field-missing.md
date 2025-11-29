# Provider 缺少加密方式字段问题

## 🐛 问题描述

前端邮箱提供商配置页面（`/providers`）缺少加密方式（Encryption）配置字段，导致：

1. 无法为提供商预设默认的加密方式
2. 用户添加账户时可能使用错误的加密方式
3. 139 邮箱等需要特定加密配置的服务商无法正确配置

## 📊 当前状态

### Provider 模型（后端）

**文件**: `backend/internal/model/provider.go`

**缺少字段**:
```go
// ❌ 缺少
Encryption string `gorm:"size:20" json:"encryption"` // 加密方式 (ssl/starttls/none)
```

### EmailAccount 模型（后端）

**文件**: `backend/internal/model/email_account.go`

**已有字段**:
```go
// ✅ 已有
Encryption string `gorm:"size:20" json:"encryption"` // 加密方式 (ssl/starttls/none)
```

### 前端类型定义

**文件**: `frontend/src/types/provider.ts`

**缺少字段**:
```typescript
// ❌ 缺少
encryption?: string; // 加密方式
```

### 前端表单

**文件**: `frontend/src/pages/ProvidersPage.tsx`

**缺少字段**: 创建和编辑表单中没有加密方式选择器

## ✅ 解决方案

### 1. 更新后端 Provider 模型

**文件**: `backend/internal/model/provider.go`

```go
type Provider struct {
	// ... 现有字段 ...
	
	// 服务器配置
	IMAPHost string `gorm:"size:255" json:"imap_host"`
	IMAPPort int `json:"imap_port"`
	POP3Host string `gorm:"size:255" json:"pop3_host"`
	POP3Port int `json:"pop3_port"`
	SMTPHost string `gorm:"size:255" json:"smtp_host"`
	SMTPPort int `json:"smtp_port"`
	
	// 新增：加密配置
	IMAPEncryption string `gorm:"size:20;default:'ssl'" json:"imap_encryption"` // IMAP 加密方式 (ssl/starttls/none)
	POP3Encryption string `gorm:"size:20;default:'ssl'" json:"pop3_encryption"` // POP3 加密方式 (ssl/starttls/none)
	SMTPEncryption string `gorm:"size:20;default:'ssl'" json:"smtp_encryption"` // SMTP 加密方式 (ssl/starttls/none)
	
	// ... 其他字段 ...
}
```

### 2. 创建数据库迁移

**文件**: `backend/migrations/006_add_encryption_to_providers.sql`

```sql
-- 添加加密方式字段到 providers 表
ALTER TABLE providers 
ADD COLUMN imap_encryption VARCHAR(20) DEFAULT 'ssl',
ADD COLUMN pop3_encryption VARCHAR(20) DEFAULT 'ssl',
ADD COLUMN smtp_encryption VARCHAR(20) DEFAULT 'ssl';

-- 更新现有记录的加密方式
-- 993/995/465 端口默认使用 SSL
UPDATE providers 
SET imap_encryption = 'ssl' 
WHERE imap_port = 993;

UPDATE providers 
SET pop3_encryption = 'ssl' 
WHERE pop3_port = 995;

UPDATE providers 
SET smtp_encryption = 'ssl' 
WHERE smtp_port = 465;

-- 143/110/587 端口默认使用 STARTTLS
UPDATE providers 
SET imap_encryption = 'starttls' 
WHERE imap_port = 143;

UPDATE providers 
SET pop3_encryption = 'starttls' 
WHERE pop3_port = 110;

UPDATE providers 
SET smtp_encryption = 'starttls' 
WHERE smtp_port = 587;

-- 为 139 邮箱设置正确的加密方式
UPDATE providers 
SET imap_encryption = 'ssl',
    pop3_encryption = 'ssl'
WHERE name = '139';
```

### 3. 更新前端类型定义

**文件**: `frontend/src/types/provider.ts`

```typescript
export interface Provider {
  // ... 现有字段 ...
  imap_host: string;
  imap_port: number;
  imap_encryption?: string; // 新增
  pop3_host?: string;
  pop3_port?: number;
  pop3_encryption?: string; // 新增
  smtp_host?: string;
  smtp_port?: number;
  smtp_encryption?: string; // 新增
  // ... 其他字段 ...
}

export interface ProviderCreateRequest {
  // ... 现有字段 ...
  imap_host?: string;
  imap_port?: number;
  imap_encryption?: string; // 新增
  pop3_host?: string;
  pop3_port?: number;
  pop3_encryption?: string; // 新增
  smtp_host?: string;
  smtp_port?: number;
  smtp_encryption?: string; // 新增
  // ... 其他字段 ...
}

export interface ProviderUpdateRequest {
  // ... 现有字段 ...
  imap_encryption?: string; // 新增
  pop3_encryption?: string; // 新增
  smtp_encryption?: string; // 新增
  // ... 其他字段 ...
}
```

### 4. 更新前端表单

**文件**: `frontend/src/pages/ProvidersPage.tsx`

在 IMAP 配置部分添加加密方式选择器：

```tsx
<div className="grid grid-cols-3 gap-4">
  <div className="space-y-2">
    <Label>IMAP 服务器</Label>
    <Input
      placeholder="imap.example.com"
      value={createForm.imap_host}
      onChange={(e) =>
        setCreateForm({ ...createForm, imap_host: e.target.value })
      }
    />
  </div>
  <div className="space-y-2">
    <Label>IMAP 端口</Label>
    <Input
      type="number"
      placeholder="993"
      value={createForm.imap_port}
      onChange={(e) =>
        setCreateForm({
          ...createForm,
          imap_port: parseInt(e.target.value) || 993,
        })
      }
    />
  </div>
  {/* 新增：IMAP 加密方式 */}
  <div className="space-y-2">
    <Label>IMAP 加密</Label>
    <Select
      value={createForm.imap_encryption || 'ssl'}
      onValueChange={(value) =>
        setCreateForm({ ...createForm, imap_encryption: value })
      }
    >
      <SelectTrigger>
        <SelectValue placeholder="选择加密方式" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="ssl">SSL/TLS (推荐)</SelectItem>
        <SelectItem value="starttls">STARTTLS</SelectItem>
        <SelectItem value="none">无加密 (不推荐)</SelectItem>
      </SelectContent>
    </Select>
  </div>
</div>
```

### 5. 更新账户创建逻辑

**文件**: `backend/internal/service/account_service.go`

在创建账户时，从 Provider 配置中读取默认的加密方式：

```go
// 从 Provider 获取默认配置
if account.Protocol == "imap" {
    provider, err := s.providerRepo.GetByName(ctx, account.Provider)
    if err == nil && provider != nil {
        // 使用 Provider 的默认加密方式
        if account.Encryption == "" {
            account.Encryption = provider.IMAPEncryption
        }
        if account.IMAPHost == "" {
            account.IMAPHost = provider.IMAPHost
        }
        if account.IMAPPort == 0 {
            account.IMAPPort = provider.IMAPPort
        }
    }
}
```

## 📋 加密方式说明

### SSL/TLS（推荐）

- **适用端口**: IMAP 993, POP3 995, SMTP 465
- **连接方式**: 直接建立 TLS 加密连接
- **安全性**: 高
- **适用场景**: 所有现代邮件服务器

### STARTTLS

- **适用端口**: IMAP 143, POP3 110, SMTP 587
- **连接方式**: 先明文连接，再升级到 TLS
- **安全性**: 中（可能被降级攻击）
- **适用场景**: 某些旧服务器或特殊配置

### 无加密（不推荐）

- **适用端口**: IMAP 143, POP3 110, SMTP 25
- **连接方式**: 明文传输
- **安全性**: 低（密码和邮件内容可被窃听）
- **适用场景**: 仅用于测试或内网环境

## 🎯 预设配置示例

### 139 邮箱（中国移动）

```json
{
  "name": "139",
  "display_name": "139 邮箱",
  "imap_host": "imap.139.com",
  "imap_port": 993,
  "imap_encryption": "ssl",
  "pop3_host": "pop.139.com",
  "pop3_port": 995,
  "pop3_encryption": "ssl"
}
```

### Gmail

```json
{
  "name": "gmail",
  "display_name": "Google Gmail",
  "imap_host": "imap.gmail.com",
  "imap_port": 993,
  "imap_encryption": "ssl",
  "smtp_host": "smtp.gmail.com",
  "smtp_port": 465,
  "smtp_encryption": "ssl"
}
```

### QQ 邮箱

```json
{
  "name": "qq",
  "display_name": "QQ 邮箱",
  "imap_host": "imap.qq.com",
  "imap_port": 993,
  "imap_encryption": "ssl",
  "smtp_host": "smtp.qq.com",
  "smtp_port": 465,
  "smtp_encryption": "ssl"
}
```

## 🔄 实施步骤

1. ✅ 更新后端 Provider 模型
2. ✅ 创建数据库迁移脚本
3. ✅ 执行数据库迁移
4. ✅ 更新前端类型定义
5. ✅ 更新前端表单组件
6. ✅ 更新账户创建逻辑
7. ✅ 测试验证

## 📚 相关文档

- [139 邮箱故障排查](./139-mail-troubleshooting.md)
- [中国邮箱服务商配置指南](./china-mail-providers.md)

---

**创建时间**: 2024-11-29  
**维护者**: FusionMail Team  
**状态**: 待实施
