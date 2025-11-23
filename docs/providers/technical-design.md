# 邮箱提供商配置化与 OAuth2 多客户端管理 - 技术设计文档

## 📋 文档信息

- **项目名称**: FusionMail 邮箱提供商配置化
- **文档版本**: v1.0
- **创建日期**: 2025-11-21
- **作者**: FusionMail Team
- **状态**: 待评审

---

## 1. 系统架构

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend (React)                      │
├─────────────────────────────────────────────────────────────┤
│  • AccountForm: 添加账户表单                                  │
│  • OAuth2ClientSelector: OAuth2 配置选择器                    │
│  • ProviderManagement (后期): 提供商管理界面                  │
└─────────────────────────────────────────────────────────────┘
                              ▼ HTTP/REST API
┌─────────────────────────────────────────────────────────────┐
│                      API Layer (Gin)                         │
├─────────────────────────────────────────────────────────────┤
│  • SystemHandler: GET /api/v1/system/providers              │
│  • OAuth2ClientHandler:                                      │
│    - GET /api/v1/oauth2/clients/:provider                   │
│    - GET /api/v1/oauth2/clients/detail/:id                  │
│    - POST/PUT/DELETE /api/v1/oauth2/clients (管理接口)      │
└─────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Service Layer                            │
├─────────────────────────────────────────────────────────────┤
│  • SystemService:                                            │
│    - GetSupportedProviders(): 获取提供商列表                 │
│    - getFallbackProviders(): 降级配置                        │
│  • OAuth2ClientService:                                      │
│    - GetAvailableClients(): 获取可用客户端                   │
│    - GetDefaultClient(): 获取默认客户端                      │
│    - UseClient(): 记录使用并更新配额                         │
│    - QuotaManager: 配额管理逻辑                              │
└─────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Repository Layer                           │
├─────────────────────────────────────────────────────────────┤
│  • ProviderRepository: 提供商 CRUD                           │
│  • OAuth2ClientRepository: OAuth2 客户端 CRUD                │
└─────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Database (PostgreSQL)                      │
├─────────────────────────────────────────────────────────────┤
│  • providers: 邮箱提供商配置表                               │
│  • oauth2_clients: OAuth2 客户端配置表                       │
│  • accounts: 账户表（扩展 oauth2_client_id 字段）            │
└─────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Cache Layer (Redis)                      │
├─────────────────────────────────────────────────────────────┤
│  • providers:cache: 提供商配置缓存 (TTL: 1h)                │
│  • oauth2_clients:{provider}:cache: OAuth2 配置缓存 (TTL: 5m)│
│  • oauth2_quota:{client_id}:{date}: 配额计数                 │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 核心组件说明

#### 1.2.1 数据层
- **providers 表**: 存储邮箱提供商配置
- **oauth2_clients 表**: 存储 OAuth2 应用配置
- **accounts 表扩展**: 关联使用的 OAuth2 配置

#### 1.2.2 Repository 层
- 提供数据访问接口
- 封装数据库操作
- 实现事务管理

#### 1.2.3 Service 层
- 实现业务逻辑
- 配额管理和切换逻辑
- 缓存管理
- 降级策略

#### 1.2.4 API 层
- RESTful API 接口
- 请求验证
- 权限控制

#### 1.2.5 前端层
- UI 组件
- 状态管理
- API 调用

---

## 2. 数据库设计

### 2.1 providers 表（邮箱提供商配置）

```sql
CREATE TABLE providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,           -- 提供商唯一标识
    display_name VARCHAR(100) NOT NULL,         -- 显示名称
    supported_protocols TEXT NOT NULL,          -- JSON数组: ["oauth2","imap","pop3"]
    recommended_protocol VARCHAR(20) NOT NULL,  -- 推荐协议
    requires_oauth BOOLEAN DEFAULT FALSE,       -- 是否强制OAuth

    -- 服务器配置
    imap_host VARCHAR(255),                     -- IMAP服务器地址
    imap_port INTEGER DEFAULT 993,
    pop3_host VARCHAR(255),                     -- POP3服务器地址
    pop3_port INTEGER DEFAULT 995,
    smtp_host VARCHAR(255),                     -- SMTP服务器（预留）
    smtp_port INTEGER DEFAULT 587,

    -- 管理字段
    enabled BOOLEAN DEFAULT TRUE,               -- 是否启用
    sort_order INTEGER DEFAULT 0,               -- 排序顺序
    description TEXT,                           -- 描述信息

    -- 元数据
    metadata TEXT,                              -- JSON格式的额外配置
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_providers_name ON providers(name);
CREATE INDEX idx_providers_enabled ON providers(enabled);
CREATE INDEX idx_providers_sort_order ON providers(sort_order);
```

#### 字段说明

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| name | VARCHAR(50) | 提供商唯一标识 | `gmail`, `outlook`, `qq` |
| display_name | VARCHAR(100) | 用户界面显示名称 | `Gmail`, `Outlook / Hotmail` |
| supported_protocols | TEXT | 支持的协议（JSON数组） | `["oauth2","imap"]` |
| recommended_protocol | VARCHAR(20) | 推荐协议 | `oauth2`, `imap` |
| requires_oauth | BOOLEAN | 是否强制 OAuth | `true` (Gmail), `false` (QQ) |
| enabled | BOOLEAN | 是否启用 | `true`, `false` |
| sort_order | INTEGER | 排序顺序，越小越靠前 | `1`, `2`, `3` |

#### 初始数据

```sql
INSERT INTO providers (name, display_name, supported_protocols, recommended_protocol,
                       requires_oauth, imap_host, imap_port, pop3_host, pop3_port,
                       sort_order, description) VALUES
-- Gmail
('gmail', 'Gmail', '["oauth2","imap"]', 'oauth2', true,
 'imap.gmail.com', 993, '', 0, 1, 'Google Gmail 邮箱服务'),

-- Outlook
('outlook', 'Outlook / Hotmail', '["oauth2","imap"]', 'oauth2', true,
 'outlook.office365.com', 993, '', 0, 2, 'Microsoft Outlook / Hotmail 邮箱服务'),

-- iCloud
('icloud', 'iCloud Mail', '["imap"]', 'imap', false,
 'imap.mail.me.com', 993, '', 0, 3, 'Apple iCloud 邮箱服务'),

-- QQ邮箱
('qq', 'QQ 邮箱', '["imap","pop3"]', 'imap', false,
 'imap.qq.com', 993, 'pop.qq.com', 995, 4, '腾讯 QQ 邮箱服务'),

-- 163邮箱
('163', '163 邮箱', '["imap","pop3"]', 'imap', false,
 'imap.163.com', 993, 'pop.163.com', 995, 5, '网易 163 邮箱服务'),

-- 通用邮箱
('generic', '通用邮箱 (IMAP/POP3)', '["imap","pop3"]', 'imap', false,
 '', 993, '', 995, 99, '支持标准 IMAP/POP3 协议的通用邮箱');
```

### 2.2 oauth2_clients 表（OAuth2 客户端配置）

```sql
CREATE TABLE oauth2_clients (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,                 -- 配置名称
    provider VARCHAR(50) NOT NULL,              -- 关联的提供商

    -- OAuth2 凭证
    client_id VARCHAR(500) NOT NULL,            -- Client ID
    encrypted_client_secret TEXT NOT NULL,      -- 加密的 Client Secret
    redirect_uri VARCHAR(500),                  -- 回调地址
    scopes TEXT,                                -- JSON数组: ["scope1","scope2"]

    -- 配置管理
    is_default BOOLEAN DEFAULT FALSE,           -- 是否为默认配置
    enabled BOOLEAN DEFAULT TRUE,               -- 是否启用
    priority INTEGER DEFAULT 0,                 -- 优先级（越小优先级越高）

    -- 配额管理
    daily_quota INTEGER,                        -- 每日配额限制（NULL=无限制）
    quota_used_today INTEGER DEFAULT 0,         -- 今日已使用配额
    quota_reset_at TIMESTAMP,                   -- 配额重置时间

    -- 统计信息
    total_authorizations INTEGER DEFAULT 0,     -- 总授权次数
    success_authorizations INTEGER DEFAULT 0,   -- 成功授权次数
    failed_authorizations INTEGER DEFAULT 0,    -- 失败授权次数
    last_used_at TIMESTAMP,                     -- 最后使用时间

    -- 其他配置
    description TEXT,                           -- 配置描述
    tags TEXT,                                  -- JSON数组: ["production","test"]
    metadata TEXT,                              -- JSON格式的额外配置

    -- 时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_provider FOREIGN KEY (provider)
        REFERENCES providers(name) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_oauth2_clients_provider ON oauth2_clients(provider);
CREATE INDEX idx_oauth2_clients_enabled ON oauth2_clients(enabled);
CREATE INDEX idx_oauth2_clients_is_default ON oauth2_clients(is_default);
CREATE INDEX idx_oauth2_clients_priority ON oauth2_clients(priority);

-- 唯一约束：每个提供商只能有一个默认配置
CREATE UNIQUE INDEX idx_oauth2_clients_default_per_provider
    ON oauth2_clients(provider, is_default)
    WHERE is_default = true;
```

#### 字段说明

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| name | VARCHAR(100) | 配置名称 | `Gmail 生产环境`, `Outlook 测试` |
| provider | VARCHAR(50) | 关联的提供商 | `gmail`, `outlook` |
| client_id | VARCHAR(500) | OAuth2 Client ID | `xxx.apps.googleusercontent.com` |
| encrypted_client_secret | TEXT | 加密的 Client Secret | 加密后的字符串 |
| is_default | BOOLEAN | 是否为默认配置 | `true`, `false` |
| priority | INTEGER | 优先级 | `1`, `2`, `3` |
| daily_quota | INTEGER | 每日配额限制 | `10000`, `NULL`（无限制） |
| quota_used_today | INTEGER | 今日已用配额 | `1234` |

### 2.3 accounts 表扩展

```sql
-- 添加 OAuth2 客户端配置关联字段
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS oauth2_client_id BIGINT;

-- 添加外键约束
ALTER TABLE accounts ADD CONSTRAINT fk_oauth2_client
    FOREIGN KEY (oauth2_client_id)
    REFERENCES oauth2_clients(id) ON DELETE SET NULL;

-- 添加索引
CREATE INDEX idx_accounts_oauth2_client_id ON accounts(oauth2_client_id);

-- 添加注释
COMMENT ON COLUMN accounts.oauth2_client_id IS '使用的 OAuth2 客户端配置 ID';
```

### 2.4 ER 图

```
┌─────────────────────┐
│     providers       │
├─────────────────────┤
│ id (PK)             │
│ name (UK)           │
│ display_name        │
│ supported_protocols │
│ recommended_protocol│
│ requires_oauth      │
│ imap_host           │
│ imap_port           │
│ ...                 │
└─────────────────────┘
         │
         │ 1:N (provider)
         ▼
┌─────────────────────┐
│  oauth2_clients     │
├─────────────────────┤
│ id (PK)             │
│ name                │
│ provider (FK)       │
│ client_id           │
│ encrypted_...       │
│ is_default          │
│ priority            │
│ daily_quota         │
│ ...                 │
└─────────────────────┘
         │
         │ 1:N (oauth2_client_id)
         ▼
┌─────────────────────┐
│     accounts        │
├─────────────────────┤
│ id (PK)             │
│ uid (UK)            │
│ email               │
│ provider            │
│ protocol            │
│ oauth2_client_id(FK)│
│ ...                 │
└─────────────────────┘
```

---

## 3. 后端实现

### 3.1 数据模型（Go）

#### 3.1.1 Provider 模型

```go
// backend/internal/model/provider.go
package model

import (
    "encoding/json"
    "time"
)

type Provider struct {
    ID                  int64     `gorm:"primaryKey" json:"id"`
    Name                string    `gorm:"uniqueIndex;size:50;not null" json:"name"`
    DisplayName         string    `gorm:"size:100;not null" json:"display_name"`
    SupportedProtocols  string    `gorm:"type:text;not null" json:"-"`
    RecommendedProtocol string    `gorm:"size:20;not null" json:"recommended_protocol"`
    RequiresOAuth       bool      `gorm:"default:false" json:"requires_oauth"`

    // 服务器配置
    IMAPHost            string    `gorm:"size:255" json:"imap_host"`
    IMAPPort            int       `json:"imap_port"`
    POP3Host            string    `gorm:"size:255" json:"pop3_host"`
    POP3Port            int       `json:"pop3_port"`
    SMTPHost            string    `gorm:"size:255" json:"smtp_host"`
    SMTPPort            int       `json:"smtp_port"`

    // 管理字段
    Enabled             bool      `gorm:"default:true" json:"enabled"`
    SortOrder           int       `gorm:"default:0" json:"sort_order"`
    Description         string    `gorm:"type:text" json:"description"`

    Metadata            string    `gorm:"type:text" json:"metadata,omitempty"`
    CreatedAt           time.Time `json:"created_at"`
    UpdatedAt           time.Time `json:"updated_at"`
}

func (Provider) TableName() string {
    return "providers"
}

// GetSupportedProtocols 获取支持的协议列表
func (p *Provider) GetSupportedProtocols() []string {
    var protocols []string
    json.Unmarshal([]byte(p.SupportedProtocols), &protocols)
    return protocols
}

// SetSupportedProtocols 设置支持的协议列表
func (p *Provider) SetSupportedProtocols(protocols []string) error {
    data, err := json.Marshal(protocols)
    if err != nil {
        return err
    }
    p.SupportedProtocols = string(data)
    return nil
}
```

#### 3.1.2 OAuth2Client 模型

```go
// backend/internal/model/oauth2_client.go
package model

import (
    "time"
)

type OAuth2Client struct {
    ID                     int64      `gorm:"primaryKey" json:"id"`
    Name                   string     `gorm:"size:100;not null" json:"name"`
    Provider               string     `gorm:"size:50;not null;index" json:"provider"`
    ClientID               string     `gorm:"size:500;not null" json:"client_id"`
    EncryptedClientSecret  string     `gorm:"type:text;not null" json:"-"`
    RedirectURI            string     `gorm:"size:500" json:"redirect_uri"`
    Scopes                 string     `gorm:"type:text" json:"-"`

    IsDefault              bool       `gorm:"default:false;index" json:"is_default"`
    Enabled                bool       `gorm:"default:true;index" json:"enabled"`
    Priority               int        `gorm:"default:0;index" json:"priority"`

    DailyQuota             *int       `json:"daily_quota,omitempty"`
    QuotaUsedToday         int        `gorm:"default:0" json:"quota_used_today"`
    QuotaResetAt           *time.Time `json:"quota_reset_at,omitempty"`

    TotalAuthorizations    int        `gorm:"default:0" json:"total_authorizations"`
    SuccessAuthorizations  int        `gorm:"default:0" json:"success_authorizations"`
    FailedAuthorizations   int        `gorm:"default:0" json:"failed_authorizations"`
    LastUsedAt             *time.Time `json:"last_used_at,omitempty"`

    Description            string     `gorm:"type:text" json:"description"`
    Tags                   string     `gorm:"type:text" json:"-"`
    Metadata               string     `gorm:"type:text" json:"metadata,omitempty"`

    CreatedAt              time.Time  `json:"created_at"`
    UpdatedAt              time.Time  `json:"updated_at"`
}

func (OAuth2Client) TableName() string {
    return "oauth2_clients"
}

// IsQuotaExceeded 检查配额是否超限
func (c *OAuth2Client) IsQuotaExceeded() bool {
    if c.DailyQuota == nil {
        return false // 无配额限制
    }

    // 检查配额是否需要重置
    if c.QuotaResetAt != nil && time.Now().After(*c.QuotaResetAt) {
        return false // 配额已重置
    }

    return c.QuotaUsedToday >= *c.DailyQuota
}

// GetAvailableQuota 获取剩余配额
func (c *OAuth2Client) GetAvailableQuota() int {
    if c.DailyQuota == nil {
        return -1 // 无限制
    }

    remaining := *c.DailyQuota - c.QuotaUsedToday
    if remaining < 0 {
        return 0
    }
    return remaining
}

// GetScopes 获取权限范围列表
func (c *OAuth2Client) GetScopes() []string {
    var scopes []string
    json.Unmarshal([]byte(c.Scopes), &scopes)
    return scopes
}

// GetTags 获取标签列表
func (c *OAuth2Client) GetTags() []string {
    var tags []string
    json.Unmarshal([]byte(c.Tags), &tags)
    return tags
}
```

### 3.2 Repository 层

#### 3.2.1 ProviderRepository 接口

```go
// backend/internal/repository/provider.go
package repository

import (
    "context"
    "fusionmail/internal/model"
)

type ProviderRepository interface {
    // 基础 CRUD
    Create(ctx context.Context, provider *model.Provider) error
    Update(ctx context.Context, provider *model.Provider) error
    Delete(ctx context.Context, name string) error

    // 查询方法
    FindAll(ctx context.Context) ([]model.Provider, error)
    FindByName(ctx context.Context, name string) (*model.Provider, error)
    FindEnabled(ctx context.Context) ([]model.Provider, error)
}
```

#### 3.2.2 OAuth2ClientRepository 接口

```go
// backend/internal/repository/oauth2_client.go
package repository

import (
    "context"
    "fusionmail/internal/model"
)

type OAuth2ClientRepository interface {
    // 基础 CRUD
    Create(ctx context.Context, client *model.OAuth2Client) error
    Update(ctx context.Context, client *model.OAuth2Client) error
    Delete(ctx context.Context, id int64) error
    FindByID(ctx context.Context, id int64) (*model.OAuth2Client, error)

    // 查询方法
    FindAll(ctx context.Context) ([]model.OAuth2Client, error)
    FindByProvider(ctx context.Context, provider string) ([]model.OAuth2Client, error)
    FindEnabledByProvider(ctx context.Context, provider string) ([]model.OAuth2Client, error)
    FindDefaultByProvider(ctx context.Context, provider string) (*model.OAuth2Client, error)

    // 配额管理
    IncrementUsage(ctx context.Context, id int64) error
    ResetDailyQuota(ctx context.Context, id int64) error

    // 统计更新
    RecordAuthorization(ctx context.Context, id int64, success bool) error

    // 获取可用客户端（智能选择）
    GetAvailableClient(ctx context.Context, provider string) (*model.OAuth2Client, error)
}
```

### 3.3 Service 层

#### 3.3.1 SystemService 修改

```go
// backend/internal/service/system_service.go

// SystemService 结构体添加字段
type SystemService struct {
    // ... 现有字段 ...
    providerRepo repository.ProviderRepository
    cache        *redis.Client
}

// GetSupportedProviders 获取支持的邮箱提供商列表（修改后）
func (s *SystemService) GetSupportedProviders(ctx context.Context) ([]ProviderInfo, error) {
    // 1. 尝试从缓存读取
    cacheKey := "providers:cache"
    cached, err := s.cache.Get(ctx, cacheKey).Result()
    if err == nil && cached != "" {
        var providers []ProviderInfo
        if json.Unmarshal([]byte(cached), &providers) == nil {
            return providers, nil
        }
    }

    // 2. 从数据库读取
    dbProviders, err := s.providerRepo.FindEnabled(ctx)
    if err != nil {
        s.logger.Error("从数据库获取提供商失败", "error", err)
        // 降级：使用硬编码配置
        return s.getFallbackProviders(), nil
    }

    // 3. 转换为 DTO
    var providers []ProviderInfo
    for _, p := range dbProviders {
        providers = append(providers, ProviderInfo{
            Name:                p.Name,
            DisplayName:         p.DisplayName,
            SupportedProtocols:  p.GetSupportedProtocols(),
            RecommendedProtocol: p.RecommendedProtocol,
            RequiresOAuth:       p.RequiresOAuth,
            IMAPHost:            p.IMAPHost,
            IMAPPort:            p.IMAPPort,
            POP3Host:            p.POP3Host,
            POP3Port:            p.POP3Port,
        })
    }

    // 4. 写入缓存（1小时）
    if data, err := json.Marshal(providers); err == nil {
        s.cache.Set(ctx, cacheKey, string(data), time.Hour)
    }

    return providers, nil
}

// getFallbackProviders 降级配置（保留现有的硬编码逻辑）
func (s *SystemService) getFallbackProviders() []ProviderInfo {
    factory := adapter.NewFactory()
    // ... 现有逻辑 ...
}
```

#### 3.3.2 OAuth2ClientService

```go
// backend/internal/service/oauth2_client_service.go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "fusionmail/internal/model"
    "fusionmail/internal/repository"
    "fusionmail/pkg/encryption"
    "fusionmail/pkg/logger"
    "time"
)

type OAuth2ClientService struct {
    repo      repository.OAuth2ClientRepository
    encryptor *encryption.Encryptor
    cache     *redis.Client
    logger    *logger.Logger
}

func NewOAuth2ClientService(
    repo repository.OAuth2ClientRepository,
    encryptor *encryption.Encryptor,
    cache *redis.Client,
    logger *logger.Logger,
) *OAuth2ClientService {
    return &OAuth2ClientService{
        repo:      repo,
        encryptor: encryptor,
        cache:     cache,
        logger:    logger,
    }
}

// GetAvailableClientsForProvider 获取提供商可用的配置列表
func (s *OAuth2ClientService) GetAvailableClientsForProvider(
    ctx context.Context,
    provider string,
) ([]OAuth2ClientDTO, error) {
    // 从缓存或数据库获取
    clients, err := s.repo.FindEnabledByProvider(ctx, provider)
    if err != nil {
        return nil, fmt.Errorf("获取 OAuth2 客户端配置失败: %w", err)
    }

    // 转换为 DTO
    var result []OAuth2ClientDTO
    for _, client := range clients {
        dto := s.convertToDTO(&client)
        result = append(result, dto)
    }

    return result, nil
}

// GetBestAvailableClient 智能选择最佳可用配置
func (s *OAuth2ClientService) GetBestAvailableClient(
    ctx context.Context,
    provider string,
) (*model.OAuth2Client, error) {
    // 1. 尝试获取默认配置
    client, err := s.repo.FindDefaultByProvider(ctx, provider)
    if err == nil && !client.IsQuotaExceeded() {
        return client, nil
    }

    // 2. 默认配置不可用，智能选择
    client, err = s.repo.GetAvailableClient(ctx, provider)
    if err != nil {
        return nil, fmt.Errorf("无可用的 OAuth2 配置: %w", err)
    }

    return client, nil
}

// UseClient 使用配置（增加配额计数）
func (s *OAuth2ClientService) UseClient(ctx context.Context, id int64) error {
    // 使用 Redis 原子操作增加配额
    today := time.Now().Format("2006-01-02")
    quotaKey := fmt.Sprintf("oauth2_quota:%d:%s", id, today)

    // 增加计数
    count, err := s.cache.Incr(ctx, quotaKey).Result()
    if err != nil {
        s.logger.Error("增加 OAuth2 配额失败", "client_id", id, "error", err)
    }

    // 设置过期时间（2天）
    if count == 1 {
        s.cache.Expire(ctx, quotaKey, 48*time.Hour)
    }

    // 异步更新数据库
    go func() {
        if err := s.repo.IncrementUsage(context.Background(), id); err != nil {
            s.logger.Error("更新数据库配额失败", "client_id", id, "error", err)
        }
    }()

    return nil
}

// RecordAuthorizationResult 记录授权结果
func (s *OAuth2ClientService) RecordAuthorizationResult(
    ctx context.Context,
    id int64,
    success bool,
) error {
    return s.repo.RecordAuthorization(ctx, id, success)
}

// convertToDTO 转换为 DTO
func (s *OAuth2ClientService) convertToDTO(client *model.OAuth2Client) OAuth2ClientDTO {
    quotaInfo := QuotaInfo{
        HasLimit: client.DailyQuota != nil,
    }

    if client.DailyQuota != nil {
        quotaInfo.Limit = *client.DailyQuota
        quotaInfo.Used = client.QuotaUsedToday
        quotaInfo.Available = client.GetAvailableQuota()
        quotaInfo.IsExceeded = client.IsQuotaExceeded()
    }

    return OAuth2ClientDTO{
        ID:          client.ID,
        Name:        client.Name,
        Provider:    client.Provider,
        ClientID:    client.ClientID,
        IsDefault:   client.IsDefault,
        Priority:    client.Priority,
        Description: client.Description,
        Tags:        client.GetTags(),
        QuotaInfo:   quotaInfo,
        IsAvailable: !client.IsQuotaExceeded(),
    }
}

// DTO 定义
type OAuth2ClientDTO struct {
    ID          int64      `json:"id"`
    Name        string     `json:"name"`
    Provider    string     `json:"provider"`
    ClientID    string     `json:"client_id"`
    IsDefault   bool       `json:"is_default"`
    Priority    int        `json:"priority"`
    Description string     `json:"description"`
    Tags        []string   `json:"tags"`
    QuotaInfo   QuotaInfo  `json:"quota_info"`
    IsAvailable bool       `json:"is_available"`
}

type QuotaInfo struct {
    HasLimit   bool `json:"has_limit"`
    Limit      int  `json:"limit,omitempty"`
    Used       int  `json:"used,omitempty"`
    Available  int  `json:"available,omitempty"`
    IsExceeded bool `json:"is_exceeded"`
}
```

### 3.4 API 层

#### 3.4.1 路由注册

```go
// backend/internal/router/router.go

func SetupRouter(/* ... */) *gin.Engine {
    // ... 现有代码 ...

    // OAuth2 客户端配置 API
    oauth2Group := v1.Group("/oauth2")
    {
        oauth2Group.GET("/clients/:provider", oauth2ClientHandler.GetAvailableClients)
        oauth2Group.GET("/clients/detail/:id", oauth2ClientHandler.GetClientByID)

        // 管理接口（需要管理员权限）
        oauth2Admin := oauth2Group.Group("/admin")
        oauth2Admin.Use(middleware.RequireAdmin())
        {
            oauth2Admin.POST("/clients", oauth2ClientHandler.CreateClient)
            oauth2Admin.PUT("/clients/:id", oauth2ClientHandler.UpdateClient)
            oauth2Admin.DELETE("/clients/:id", oauth2ClientHandler.DeleteClient)
        }
    }

    // 提供商管理 API（需要管理员权限）
    providerGroup := v1.Group("/providers")
    providerGroup.Use(middleware.RequireAdmin())
    {
        providerGroup.GET("", providerHandler.List)
        providerGroup.POST("", providerHandler.Create)
        providerGroup.PUT("/:name", providerHandler.Update)
        providerGroup.DELETE("/:name", providerHandler.Delete)
    }

    return router
}
```

#### 3.4.2 Handler 实现

```go
// backend/internal/handler/oauth2_client_handler.go
package handler

import (
    "fusionmail/internal/dto"
    "fusionmail/internal/service"
    "github.com/gin-gonic/gin"
    "strconv"
)

type OAuth2ClientHandler struct {
    service *service.OAuth2ClientService
}

func NewOAuth2ClientHandler(service *service.OAuth2ClientService) *OAuth2ClientHandler {
    return &OAuth2ClientHandler{service: service}
}

// GetAvailableClients 获取指定提供商的可用配置
func (h *OAuth2ClientHandler) GetAvailableClients(c *gin.Context) {
    provider := c.Param("provider")

    clients, err := h.service.GetAvailableClientsForProvider(c.Request.Context(), provider)
    if err != nil {
        dto.HandleServiceError(c, err)
        return
    }

    dto.SuccessResponse(c, clients)
}

// GetClientByID 获取指定配置详情
func (h *OAuth2ClientHandler) GetClientByID(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        dto.BadRequestResponse(c, "无效的客户端 ID")
        return
    }

    client, err := h.service.GetClientByID(c.Request.Context(), id)
    if err != nil {
        dto.HandleServiceError(c, err)
        return
    }

    dto.SuccessResponse(c, client)
}

// CreateClient 创建配置
func (h *OAuth2ClientHandler) CreateClient(c *gin.Context) {
    var req service.CreateOAuth2ClientRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        dto.BadRequestResponse(c, "请求参数错误")
        return
    }

    if err := h.service.CreateClient(c.Request.Context(), &req); err != nil {
        dto.HandleServiceError(c, err)
        return
    }

    dto.SuccessResponse(c, nil)
}
```

---

## 4. 前端实现

### 4.1 类型定义

```typescript
// frontend/src/types/provider.ts

export interface Provider {
  name: string;
  display_name: string;
  supported_protocols: string[];
  recommended_protocol: string;
  requires_oauth: boolean;
  imap_host?: string;
  imap_port?: number;
  pop3_host?: string;
  pop3_port?: number;
}

export interface OAuth2Client {
  id: number;
  name: string;
  provider: string;
  client_id: string;
  is_default: boolean;
  priority: number;
  description: string;
  tags: string[];
  quota_info: QuotaInfo;
  is_available: boolean;
}

export interface QuotaInfo {
  has_limit: boolean;
  limit?: number;
  used?: number;
  available?: number;
  is_exceeded: boolean;
}
```

### 4.2 API 服务

```typescript
// frontend/src/services/oauth2ClientService.ts

import { api } from './api';
import type { OAuth2Client } from '../types/provider';

export const oauth2ClientService = {
  /**
   * 获取指定提供商的可用 OAuth2 客户端配置
   */
  async getAvailableClients(provider: string): Promise<OAuth2Client[]> {
    const response = await api.get<{
      success: boolean;
      data: OAuth2Client[];
    }>(`/oauth2/clients/${provider}`);

    if (response.success && response.data) {
      return response.data;
    }

    throw new Error('获取 OAuth2 客户端配置失败');
  },

  /**
   * 获取指定配置的详细信息
   */
  async getClientById(id: number): Promise<OAuth2Client> {
    const response = await api.get<{
      success: boolean;
      data: OAuth2Client;
    }>(`/oauth2/clients/detail/${id}`);

    if (response.success && response.data) {
      return response.data;
    }

    throw new Error('获取配置详情失败');
  },
};
```

### 4.3 组件实现

#### 4.3.1 OAuth2ClientSelector 组件

```tsx
// frontend/src/components/account/OAuth2ClientSelector.tsx

import { useState, useEffect } from 'react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Label } from '../ui/label';
import { Alert, AlertDescription } from '../ui/alert';
import { Badge } from '../ui/badge';
import { oauth2ClientService } from '../../services/oauth2ClientService';
import type { OAuth2Client } from '../../types/provider';

interface OAuth2ClientSelectorProps {
  provider: string; // 'gmail' or 'outlook'
  value: number | null;
  onChange: (clientId: number | null) => void;
}

export const OAuth2ClientSelector = ({ provider, value, onChange }: OAuth2ClientSelectorProps) => {
  const [clients, setClients] = useState<OAuth2Client[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadClients();
  }, [provider]);

  const loadClients = async () => {
    setLoading(true);
    try {
      const data = await oauth2ClientService.getAvailableClients(provider);
      setClients(data);

      // 自动选择默认配置
      if (!value) {
        const defaultClient = data.find(c => c.is_default && c.is_available);
        if (defaultClient) {
          onChange(defaultClient.id);
        } else {
          const firstAvailable = data.find(c => c.is_available);
          if (firstAvailable) {
            onChange(firstAvailable.id);
          }
        }
      }
    } catch (error) {
      console.error('加载 OAuth2 配置失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const selectedClient = clients.find(c => c.id === value);

  // 如果只有一个配置，不显示选择器
  if (clients.length <= 1) {
    return null;
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="oauth2_client">OAuth2 应用配置</Label>
      <Select
        value={value?.toString() || ''}
        onValueChange={(v) => onChange(parseInt(v))}
        disabled={loading}
      >
        <SelectTrigger>
          <SelectValue placeholder="选择配置" />
        </SelectTrigger>
        <SelectContent>
          {clients.map((client) => (
            <SelectItem
              key={client.id}
              value={client.id.toString()}
              disabled={!client.is_available}
            >
              <div className="flex items-center justify-between w-full">
                <span className="flex items-center gap-2">
                  {client.name}
                  {client.is_default && (
                    <Badge variant="secondary">默认</Badge>
                  )}
                  {!client.is_available && (
                    <Badge variant="destructive">配额已满</Badge>
                  )}
                </span>
                {client.quota_info.has_limit && (
                  <span className="text-xs text-muted-foreground ml-2">
                    {client.quota_info.available}/{client.quota_info.limit}
                  </span>
                )}
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* 显示选中配置的详细信息 */}
      {selectedClient && (
        <Alert>
          <AlertDescription className="space-y-1">
            {selectedClient.description && (
              <p className="text-sm">{selectedClient.description}</p>
            )}
            {selectedClient.quota_info.has_limit && (
              <p className="text-xs text-muted-foreground">
                今日配额: {selectedClient.quota_info.used}/{selectedClient.quota_info.limit}
                {selectedClient.quota_info.is_exceeded && (
                  <span className="text-red-600 ml-1">(已超限)</span>
                )}
              </p>
            )}
            {selectedClient.tags.length > 0 && (
              <div className="flex gap-1 mt-1">
                {selectedClient.tags.map(tag => (
                  <Badge key={tag} variant="outline" className="text-xs">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
};
```

#### 4.3.2 AccountForm 集成

```tsx
// frontend/src/components/account/AccountForm.tsx

import { OAuth2ClientSelector } from './OAuth2ClientSelector';

export const AccountForm = ({ /* ... */ }) => {
  const [selectedOAuth2ClientId, setSelectedOAuth2ClientId] = useState<number | null>(null);

  // ... 其他代码 ...

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        {/* ... 其他表单字段 ... */}

        {/* OAuth2 认证区域 */}
        {!isEditMode && formData.protocol === 'oauth2' && (
          <div className="space-y-4 p-4 border rounded-lg bg-blue-50 dark:bg-blue-900/20">
            {/* OAuth2 客户端选择器 */}
            <OAuth2ClientSelector
              provider={formData.provider}
              value={selectedOAuth2ClientId}
              onChange={setSelectedOAuth2ClientId}
            />

            <div>
              <h4 className="font-medium text-sm text-gray-900 dark:text-white">
                OAuth2 安全认证
              </h4>
              <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                点击下方按钮，通过官方授权页面安全登录
              </p>
            </div>

            <OAuth2AuthButton
              provider={formData.provider === 'gmail' ? 'google' : 'microsoft'}
              clientId={selectedOAuth2ClientId}
              onSuccess={() => {
                onClose();
                window.location.reload();
              }}
              onError={(error) => {
                console.error('OAuth2 error:', error);
              }}
            />
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};
```

---

## 5. 数据迁移

### 5.1 迁移脚本

```sql
-- backend/migrations/005_create_providers_table.sql
-- 创建邮箱提供商配置表
CREATE TABLE IF NOT EXISTS providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    supported_protocols TEXT NOT NULL,
    recommended_protocol VARCHAR(20) NOT NULL,
    requires_oauth BOOLEAN DEFAULT FALSE,
    imap_host VARCHAR(255),
    imap_port INTEGER DEFAULT 993,
    pop3_host VARCHAR(255),
    pop3_port INTEGER DEFAULT 995,
    smtp_host VARCHAR(255),
    smtp_port INTEGER DEFAULT 587,
    enabled BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,
    description TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_providers_name ON providers(name);
CREATE INDEX idx_providers_enabled ON providers(enabled);
CREATE INDEX idx_providers_sort_order ON providers(sort_order);

-- 初始化数据（包含现有的所有提供商配置）
INSERT INTO providers (name, display_name, supported_protocols, recommended_protocol, requires_oauth, imap_host, imap_port, pop3_host, pop3_port, sort_order, description) VALUES
('gmail', 'Gmail', '["oauth2","imap"]', 'oauth2', true, 'imap.gmail.com', 993, '', 0, 1, 'Google Gmail 邮箱服务'),
('outlook', 'Outlook / Hotmail', '["oauth2","imap"]', 'oauth2', true, 'outlook.office365.com', 993, '', 0, 2, 'Microsoft Outlook / Hotmail 邮箱服务'),
('icloud', 'iCloud Mail', '["imap"]', 'imap', false, 'imap.mail.me.com', 993, '', 0, 3, 'Apple iCloud 邮箱服务'),
('qq', 'QQ 邮箱', '["imap","pop3"]', 'imap', false, 'imap.qq.com', 993, 'pop.qq.com', 995, 4, '腾讯 QQ 邮箱服务'),
('163', '163 邮箱', '["imap","pop3"]', 'imap', false, 'imap.163.com', 993, 'pop.163.com', 995, 5, '网易 163 邮箱服务'),
('generic', '通用邮箱 (IMAP/POP3)', '["imap","pop3"]', 'imap', false, '', 993, '', 995, 99, '支持标准 IMAP/POP3 协议的通用邮箱');
```

```sql
-- backend/migrations/006_create_oauth2_clients_table.sql
-- 创建 OAuth2 客户端配置表
CREATE TABLE IF NOT EXISTS oauth2_clients (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    client_id VARCHAR(500) NOT NULL,
    encrypted_client_secret TEXT NOT NULL,
    redirect_uri VARCHAR(500),
    scopes TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    daily_quota INTEGER,
    quota_used_today INTEGER DEFAULT 0,
    quota_reset_at TIMESTAMP,
    total_authorizations INTEGER DEFAULT 0,
    success_authorizations INTEGER DEFAULT 0,
    failed_authorizations INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    description TEXT,
    tags TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_provider FOREIGN KEY (provider) REFERENCES providers(name) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_oauth2_clients_provider ON oauth2_clients(provider);
CREATE INDEX idx_oauth2_clients_enabled ON oauth2_clients(enabled);
CREATE INDEX idx_oauth2_clients_is_default ON oauth2_clients(is_default);
CREATE INDEX idx_oauth2_clients_priority ON oauth2_clients(priority);

-- 唯一约束：每个提供商只能有一个默认配置
CREATE UNIQUE INDEX idx_oauth2_clients_default_per_provider
    ON oauth2_clients(provider, is_default)
    WHERE is_default = true;

-- 扩展 accounts 表
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS oauth2_client_id BIGINT;
ALTER TABLE accounts ADD CONSTRAINT fk_oauth2_client
    FOREIGN KEY (oauth2_client_id) REFERENCES oauth2_clients(id) ON DELETE SET NULL;

CREATE INDEX idx_accounts_oauth2_client_id ON accounts(oauth2_client_id);

COMMENT ON COLUMN accounts.oauth2_client_id IS '使用的 OAuth2 客户端配置 ID';
```

### 5.2 迁移执行

```bash
# 运行迁移
cd backend
go run cmd/migrate/main.go up

# 或者使用 SQL 客户端直接执行
psql -U postgres -d fusionmail -f migrations/005_create_providers_table.sql
psql -U postgres -d fusionmail -f migrations/006_create_oauth2_clients_table.sql
```

---

## 6. 配额管理机制

### 6.1 配额计数

#### 6.1.1 Redis 计数器

```go
// 配额 Key 格式: oauth2_quota:{client_id}:{date}
// 例如: oauth2_quota:1:2025-11-21

func (s *OAuth2ClientService) UseClient(ctx context.Context, id int64) error {
    today := time.Now().Format("2006-01-02")
    quotaKey := fmt.Sprintf("oauth2_quota:%d:%s", id, today)

    // 原子增加
    count, err := s.cache.Incr(ctx, quotaKey).Result()
    if err != nil {
        return err
    }

    // 设置过期时间（2天，防止数据堆积）
    if count == 1 {
        s.cache.Expire(ctx, quotaKey, 48*time.Hour)
    }

    // 异步同步到数据库
    go s.syncQuotaToDB(id, int(count))

    return nil
}
```

#### 6.1.2 配额重置

```go
// 定时任务：每日凌晨重置配额
func (s *OAuth2ClientService) ResetDailyQuotas(ctx context.Context) error {
    // 重置所有配置的 quota_used_today
    return s.repo.ResetAllDailyQuotas(ctx)
}

// 使用 cron 调度
// 0 0 * * * - 每天凌晨执行
```

### 6.2 智能选择算法

```go
// GetAvailableClient 智能选择可用配置
func (r *oauth2ClientRepository) GetAvailableClient(
    ctx context.Context,
    provider string,
) (*model.OAuth2Client, error) {
    var client model.OAuth2Client

    // 查询条件：
    // 1. 提供商匹配
    // 2. 已启用
    // 3. 配额未超限（无配额限制 OR 已用 < 限制 OR 配额已过期）
    // 排序规则：
    // 1. 优先级（priority ASC）
    // 2. 最少使用（last_used_at ASC）
    err := r.db.WithContext(ctx).
        Where("provider = ? AND enabled = ?", provider, true).
        Where(`
            daily_quota IS NULL
            OR quota_used_today < daily_quota
            OR quota_reset_at < NOW()
        `).
        Order("priority ASC, last_used_at ASC NULLS FIRST").
        First(&client).Error

    if err != nil {
        return nil, err
    }

    return &client, nil
}
```

---

## 7. 缓存策略

### 7.1 缓存层次

```
┌─────────────────────────────────────────┐
│         应用层（Service）                │
├─────────────────────────────────────────┤
│  1. 内存缓存（Process Cache）            │
│     - 提供商列表（本地变量）             │
│     - TTL: 进程生命周期                  │
├─────────────────────────────────────────┤
│  2. Redis 缓存                           │
│     - providers:cache (1 hour)          │
│     - oauth2_clients:{provider} (5 min) │
│     - oauth2_quota:{id}:{date} (2 day)  │
├─────────────────────────────────────────┤
│  3. 数据库（PostgreSQL）                 │
│     - providers 表                       │
│     - oauth2_clients 表                  │
└─────────────────────────────────────────┘
```

### 7.2 缓存失效策略

#### 7.2.1 主动失效

```go
// 配置更新时清除缓存
func (s *SystemService) UpdateProvider(ctx context.Context, provider *model.Provider) error {
    // 更新数据库
    if err := s.providerRepo.Update(ctx, provider); err != nil {
        return err
    }

    // 清除缓存
    s.cache.Del(ctx, "providers:cache")

    return nil
}
```

#### 7.2.2 被动失效

- TTL 到期自动失效
- Redis 内存不足时 LRU 淘汰

---

## 8. 安全性设计

### 8.1 敏感信息加密

```go
// Client Secret 使用 AES-256-GCM 加密
type Encryptor struct {
    key []byte // 32 bytes for AES-256
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
    // 加密逻辑（使用现有的 encryption package）
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
    // 解密逻辑
}
```

### 8.2 权限控制

```go
// 管理接口需要管理员权限
func RequireAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        user := c.MustGet("user").(*model.User)
        if !user.IsAdmin {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "需要管理员权限",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 8.3 API 响应过滤

```go
// DTO 不包含敏感字段
type OAuth2ClientDTO struct {
    ID          int64  `json:"id"`
    Name        string `json:"name"`
    ClientID    string `json:"client_id"`
    // ClientSecret 不返回
    // ...
}
```

---

## 9. 监控与日志

### 9.1 关键日志

```go
// 配额使用
s.logger.Info("使用 OAuth2 配置",
    "client_id", clientID,
    "provider", provider,
    "quota_used", quotaUsed,
    "quota_limit", quotaLimit)

// 配额超限
s.logger.Warn("OAuth2 配置配额超限，切换到备用配置",
    "client_id", clientID,
    "provider", provider,
    "fallback_client_id", fallbackID)

// 授权失败
s.logger.Error("OAuth2 授权失败",
    "client_id", clientID,
    "provider", provider,
    "error", err)
```

### 9.2 监控指标

- OAuth2 配置使用次数
- 配额使用率
- 授权成功率
- 配置切换频率
- API 响应时间

---

## 10. 测试策略

### 10.1 单元测试

```go
// backend/internal/service/oauth2_client_service_test.go

func TestGetBestAvailableClient(t *testing.T) {
    // 测试默认配置可用
    // 测试默认配置配额超限，切换到备用配置
    // 测试所有配置都超限
}

func TestQuotaManagement(t *testing.T) {
    // 测试配额增加
    // 测试配额重置
    // 测试配额超限判断
}
```

### 10.2 集成测试

```go
func TestProviderFlow(t *testing.T) {
    // 1. 创建提供商
    // 2. 创建 OAuth2 配置
    // 3. 用户添加账户
    // 4. 验证使用正确的配置
}
```

### 10.3 性能测试

```bash
# 使用 Apache Bench 测试 API 性能
ab -n 1000 -c 100 http://localhost:8080/api/v1/system/providers

# 预期响应时间 < 100ms
```

---

## 11. 部署方案

### 11.1 部署步骤

1. **数据库迁移**
   ```bash
   # 执行迁移脚本
   go run cmd/migrate/main.go up
   ```

2. **配置 OAuth2 应用**
   ```bash
   # 使用管理接口或直接插入数据库
   INSERT INTO oauth2_clients (...) VALUES (...);
   ```

3. **重启服务**
   ```bash
   # 重启后端服务
   systemctl restart fusionmail-backend
   ```

4. **验证功能**
   ```bash
   # 测试 API
   curl http://localhost:8080/api/v1/system/providers
   curl http://localhost:8080/api/v1/oauth2/clients/gmail
   ```

### 11.2 回滚方案

1. **降级策略**：代码中保留硬编码配置作为 fallback
2. **数据库回滚**：删除新增的表和字段
3. **代码回滚**：恢复到之前的版本

---

## 12. 后续优化

### 12.1 短期优化（1个月）

- 添加配置管理界面
- 完善监控告警
- 优化缓存策略

### 12.2 长期优化（3-6个月）

- 智能配额分配算法
- 多租户配置隔离
- 配置模板功能

---

## 13. 附录

### 13.1 API 接口文档

详见 [API 文档](./api-documentation.md)

### 13.2 数据库 Schema

详见 [数据库设计](./database-schema.md)

### 13.3 配置示例

```json
// Gmail OAuth2 配置示例
{
  "name": "Gmail 生产环境",
  "provider": "gmail",
  "client_id": "xxx.apps.googleusercontent.com",
  "client_secret": "GOCSPX-xxx",
  "redirect_uri": "http://localhost:8080/api/v1/oauth2/callback",
  "scopes": [
    "https://www.googleapis.com/auth/gmail.readonly",
    "https://www.googleapis.com/auth/gmail.modify"
  ],
  "is_default": true,
  "enabled": true,
  "priority": 1,
  "daily_quota": 10000,
  "description": "生产环境使用的 Gmail OAuth2 配置",
  "tags": ["production", "primary"]
}
```

---

**文档状态**: 待评审
**下一步行动**: 团队评审后开始开发
