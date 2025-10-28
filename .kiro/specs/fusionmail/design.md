# FusionMail 系统设计文档

## 文档信息

**项目名称**：FusionMail - 轻量级邮件接收聚合系统  
**文档版本**：v1.0  
**创建日期**：2025-10-27  
**最后更新**：2025-10-27  
**设计负责人**：架构团队

---

## 1. 设计概述

### 1.1 设计目标

FusionMail 系统设计遵循以下核心目标：

1. **高性能**：支持并发同步多个邮箱，快速响应用户请求
2. **高可靠**：确保邮件数据不丢失，同步任务自动重试
3. **易扩展**：模块化设计，便于添加新的邮箱协议和功能
4. **轻量化**：资源占用低，适合个人和小团队部署
5. **安全性**：保护用户凭证和邮件数据安全

### 1.2 技术栈选型

#### 后端技术栈

| 技术 | 版本 | 用途 | 选型理由 |
|-----|------|------|---------|
| **Go** | 1.21+ | 后端开发语言 | 高性能、并发友好、内存安全、编译型语言 |
| **Gin** | 1.9+ | Web 框架 | 轻量级、高性能、中间件丰富 |
| **GORM** | 1.25+ | ORM 框架 | 功能完善、支持多数据库、迁移方便 |
| **PostgreSQL** | 15+ | 主数据库 | 开源、功能强大、支持 JSON、全文搜索 |
| **Redis** | 7+ | 缓存 + 队列 | 高性能、支持多种数据结构、持久化 |
| **JWT** | - | 认证方案 | 无状态、跨域友好、标准化 |

#### 前端技术栈

| 技术 | 版本 | 用途 | 选型理由 |
|-----|------|------|---------|
| **React** | 19.2.0 | 前端框架 | 组件化、生态丰富、性能优秀 |
| **TypeScript** | 5.9.3 | 类型系统 | 类型安全、IDE 友好、减少错误 |
| **Vite** | 7.1.7 | 构建工具 | 快速启动、HMR、现代化 |
| **Tailwind CSS** | 4.1.14 | CSS 框架 | 实用优先、响应式、可定制 |
| **shadcn/ui** | latest | UI 组件库 | 高质量、可定制、无依赖锁定 |
| **React Router** | 6+ | 路由管理 | 声明式、嵌套路由、代码分割 |
| **Axios** | 1.6+ | HTTP 客户端 | 拦截器、取消请求、自动转换 |
| **Zustand** | 4+ | 状态管理 | 轻量级、简单易用、TypeScript 友好 |

---

## 2. 系统架构设计

### 2.1 整体架构

FusionMail 采用**分层架构 + 模块化设计**，单体应用部署，便于轻量化运行。

```
┌─────────────────────────────────────────────────────────────┐
│                        用户层                                 │
│                   (Web Browser / Mobile)                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      表现层 (Frontend)                        │
│   React 19 + TypeScript + Vite + Tailwind + shadcn/ui      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓ HTTPS / WebSocket
┌─────────────────────────────────────────────────────────────┐
│                    API 网关层 (API Gateway)                   │
│         Gin Framework + JWT Auth + Rate Limiter             │
│         ├─ REST API Endpoints                                │
│         ├─ WebSocket (实时通知)                               │
│         └─ Middleware (认证、日志、CORS)                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                   业务逻辑层 (Business Logic)                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 账户管理服务  │  │ 邮件同步引擎  │  │ 规则引擎      │      │
│  │ Account Svc  │  │  Sync Engine │  │  Rule Engine │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 邮件管理服务  │  │ Webhook 服务  │  │ 搜索服务      │      │
│  │  Email Svc   │  │ Webhook Svc  │  │ Search Svc   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                 邮箱协议适配层 (Mail Adapters)                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Gmail API    │  │ MS Graph API │  │ IMAP Adapter │      │
│  │  Adapter     │  │   Adapter    │  │              │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐                                           │
│  │ POP3 Adapter │  (统一接口 MailProvider)                   │
│  └──────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                  数据访问层 (Data Access)                     │
│              GORM + Repository Pattern                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                  基础设施层 (Infrastructure)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ PostgreSQL   │  │    Redis     │  │  File Store  │      │
│  │  (主数据库)   │  │ (缓存+队列)   │  │  (附件存储)   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    外部服务 (External Services)               │
│  Gmail API / Microsoft Graph / IMAP/POP3 Servers            │
│  Webhook Endpoints / Third-party Automation Platforms       │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 架构分层说明

#### 2.2.1 表现层 (Frontend)

**职责**：
- 用户界面展示
- 用户交互处理
- 前端路由管理
- 状态管理

**技术实现**：
- React 19 组件化开发
- TypeScript 类型安全
- Tailwind CSS 样式管理
- shadcn/ui 组件库
- Zustand 状态管理
- React Router 路由

#### 2.2.2 API 网关层

**职责**：
- 统一入口管理
- 身份认证与授权
- 请求路由与转发
- 速率限制
- 日志记录
- CORS 处理

**技术实现**：
- Gin Web 框架
- JWT 认证中间件
- Redis 速率限制
- 结构化日志

#### 2.2.3 业务逻辑层

**核心服务模块**：

1. **账户管理服务 (Account Service)**
   - 邮箱账户 CRUD
   - 凭证加密存储
   - 连接测试
   - 代理配置管理

2. **邮件同步引擎 (Sync Engine)**
   - 定时同步调度
   - 并发同步控制
   - 增量同步逻辑
   - 错误重试机制

3. **邮件管理服务 (Email Service)**
   - 邮件查询与过滤
   - 本地状态管理（只读镜像）
   - 附件管理
   - 邮件搜索

4. **规则引擎 (Rule Engine)**
   - 规则匹配与执行
   - 自动分类与标签
   - 触发动作处理

5. **Webhook 服务 (Webhook Service)**
   - 事件触发
   - HTTP 请求发送
   - 失败重试
   - 日志记录

6. **搜索服务 (Search Service)**
   - 全文搜索
   - 高级筛选
   - 智能文件夹

#### 2.2.4 邮箱协议适配层

**设计模式**：适配器模式 + 工厂模式

**统一接口 (MailProvider)**：
```go
type MailProvider interface {
    Connect(config *AccountConfig) error
    FetchEmails(since time.Time) ([]*Email, error)
    GetEmailDetail(providerID string) (*EmailDetail, error)
    Disconnect() error
}
```

**适配器实现**：
- GmailAdapter：Gmail API
- GraphAdapter：Microsoft Graph API
- IMAPAdapter：通用 IMAP 协议
- POP3Adapter：POP3 协议

#### 2.2.5 数据访问层

**设计模式**：Repository Pattern

**职责**：
- 数据库操作封装
- 事务管理
- 查询优化
- 数据映射

#### 2.2.6 基础设施层

**组件**：
- PostgreSQL：主数据存储
- Redis：缓存 + 消息队列
- File Store：附件存储（本地或对象存储）

---

## 3. 数据模型设计

### 3.1 数据库选型

**主数据库**：PostgreSQL 15+

**选型理由**：
- 开源免费，社区活跃
- 支持 JSON 字段，灵活存储邮件元数据
- 支持全文搜索（tsvector）
- 事务支持完善
- 性能优秀

### 3.2 核心数据表设计

#### 3.2.1 邮箱账户表 (accounts)

```sql
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    uid VARCHAR(64) UNIQUE NOT NULL,              -- 账户唯一标识
    email VARCHAR(255) NOT NULL,                  -- 邮箱地址
    provider VARCHAR(50) NOT NULL,                -- 服务商类型 (gmail/outlook/imap/pop3)
    protocol VARCHAR(20) NOT NULL,                -- 协议类型 (gmail_api/graph/imap/pop3)
    
    -- 认证信息（加密存储）
    auth_type VARCHAR(20) NOT NULL,               -- 认证类型 (oauth2/password/app_password)
    encrypted_credentials TEXT NOT NULL,          -- 加密后的凭证 (JSON)
    
    -- 代理配置
    proxy_enabled BOOLEAN DEFAULT FALSE,
    proxy_type VARCHAR(20),                       -- http/socks5
    proxy_host VARCHAR(255),
    proxy_port INTEGER,
    proxy_username VARCHAR(255),
    encrypted_proxy_password TEXT,
    
    -- 同步配置
    sync_enabled BOOLEAN DEFAULT TRUE,
    sync_interval INTEGER DEFAULT 5,              -- 同步间隔（分钟）
    last_sync_at TIMESTAMP,
    last_sync_status VARCHAR(20),                 -- success/failed/running
    last_sync_error TEXT,
    
    -- 统计信息
    total_emails INTEGER DEFAULT 0,
    unread_count INTEGER DEFAULT 0,
    
    -- 元数据
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,                         -- 软删除
    
    INDEX idx_email (email),
    INDEX idx_provider (provider),
    INDEX idx_sync_enabled (sync_enabled)
);
```

#### 3.2.2 邮件主表 (emails)

```sql
CREATE TABLE emails (
    id BIGSERIAL PRIMARY KEY,
    
    -- 唯一标识（Provider ID + Account UID）
    provider_id VARCHAR(255) NOT NULL,            -- 邮箱服务商原生 ID
    account_uid VARCHAR(64) NOT NULL,             -- 所属账户 UID
    message_id VARCHAR(255),                      -- 邮件 Message-ID（辅助）
    
    -- 基本信息
    subject TEXT NOT NULL,
    from_address VARCHAR(255) NOT NULL,
    from_name VARCHAR(255),
    to_addresses TEXT,                            -- JSON 数组
    cc_addresses TEXT,                            -- JSON 数组
    bcc_addresses TEXT,                           -- JSON 数组
    reply_to VARCHAR(255),
    
    -- 邮件内容
    text_body TEXT,                               -- 纯文本正文
    html_body TEXT,                               -- HTML 正文
    snippet TEXT,                                 -- 摘要（前 200 字符）
    
    -- 本地状态（只读镜像模式）
    is_read BOOLEAN DEFAULT FALSE,                -- 本地已读状态
    is_starred BOOLEAN DEFAULT FALSE,             -- 本地星标状态
    is_archived BOOLEAN DEFAULT FALSE,            -- 本地归档状态
    is_deleted BOOLEAN DEFAULT FALSE,             -- 本地删除状态（软删除）
    local_labels TEXT,                            -- 本地标签（JSON 数组）
    
    -- 源邮箱状态（只读，不修改）
    source_is_read BOOLEAN,                       -- 源邮箱已读状态
    source_labels TEXT,                           -- 源邮箱标签（JSON 数组）
    source_folder VARCHAR(255),                   -- 源邮箱文件夹
    
    -- 附件信息
    has_attachments BOOLEAN DEFAULT FALSE,
    attachments_count INTEGER DEFAULT 0,
    
    -- 时间信息
    sent_at TIMESTAMP NOT NULL,                   -- 发送时间
    received_at TIMESTAMP NOT NULL,               -- 接收时间
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- 同步时间
    
    -- 元数据
    size_bytes BIGINT,                            -- 邮件大小
    thread_id VARCHAR(255),                       -- 会话 ID
    in_reply_to VARCHAR(255),                     -- 回复的邮件 ID
    references TEXT,                              -- 引用的邮件 ID 列表
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束：Provider ID + Account UID
    UNIQUE (provider_id, account_uid),
    
    -- 索引
    INDEX idx_account_uid (account_uid),
    INDEX idx_message_id (message_id),
    INDEX idx_from_address (from_address),
    INDEX idx_sent_at (sent_at DESC),
    INDEX idx_is_read (is_read),
    INDEX idx_is_starred (is_starred),
    INDEX idx_is_archived (is_archived),
    INDEX idx_is_deleted (is_deleted),
    
    -- 全文搜索索引
    INDEX idx_fulltext_search USING gin(
        to_tsvector('english', 
            coalesce(subject, '') || ' ' || 
            coalesce(from_name, '') || ' ' || 
            coalesce(text_body, '')
        )
    ),
    
    FOREIGN KEY (account_uid) REFERENCES accounts(uid) ON DELETE CASCADE
);
```

#### 3.2.3 邮件附件表 (email_attachments)

```sql
CREATE TABLE email_attachments (
    id BIGSERIAL PRIMARY KEY,
    email_id BIGINT NOT NULL,
    
    -- 附件信息
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100),
    size_bytes BIGINT NOT NULL,
    
    -- 存储信息
    storage_type VARCHAR(20) DEFAULT 'local',     -- local/s3/oss
    storage_path TEXT NOT NULL,                   -- 存储路径或 URL
    
    -- 元数据
    is_inline BOOLEAN DEFAULT FALSE,              -- 是否内联附件
    content_id VARCHAR(255),                      -- Content-ID（用于内联）
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_email_id (email_id),
    INDEX idx_filename (filename),
    
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);
```

#### 3.2.4 邮件标签表 (email_labels)

```sql
CREATE TABLE email_labels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(20),                            -- 标签颜色
    description TEXT,
    
    -- 统计信息
    email_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (name)
);
```

#### 3.2.5 邮件-标签关联表 (email_label_relations)

```sql
CREATE TABLE email_label_relations (
    email_id BIGINT NOT NULL,
    label_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (email_id, label_id),
    
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES email_labels(id) ON DELETE CASCADE
);
```

#### 3.2.6 邮件规则表 (email_rules)

```sql
CREATE TABLE email_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 规则配置
    enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,                   -- 优先级（数字越小越高）
    
    -- 触发条件（JSON）
    conditions TEXT NOT NULL,                     -- 条件配置
    /*
    示例：
    {
        "match_type": "all",  // all/any
        "rules": [
            {"field": "from", "operator": "contains", "value": "@example.com"},
            {"field": "subject", "operator": "contains", "value": "重要"},
            {"field": "has_attachments", "operator": "equals", "value": true}
        ]
    }
    */
    
    -- 执行动作（JSON）
    actions TEXT NOT NULL,                        -- 动作配置
    /*
    示例：
    [
        {"type": "add_label", "value": "工作"},
        {"type": "mark_read"},
        {"type": "archive"},
        {"type": "webhook", "webhook_id": 123}
    ]
    */
    
    -- 统计信息
    matched_count INTEGER DEFAULT 0,
    last_matched_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_enabled (enabled),
    INDEX idx_priority (priority)
);
```

#### 3.2.7 Webhook 配置表 (webhooks)

```sql
CREATE TABLE webhooks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Webhook 配置
    url TEXT NOT NULL,
    method VARCHAR(10) DEFAULT 'POST',
    headers TEXT,                                 -- 自定义 HTTP 头（JSON）
    enabled BOOLEAN DEFAULT TRUE,
    
    -- 触发条件
    events TEXT NOT NULL,                         -- 监听的事件类型（JSON 数组）
    /*
    ["email.received", "email.read", "email.archived", "email.deleted"]
    */
    
    filters TEXT,                                 -- 过滤条件（JSON）
    /*
    {
        "account_uids": ["xxx", "yyy"],
        "labels": ["重要"],
        "from_contains": "@example.com"
    }
    */
    
    -- 重试配置
    retry_enabled BOOLEAN DEFAULT TRUE,
    max_retries INTEGER DEFAULT 3,
    retry_intervals TEXT DEFAULT '[10, 30, 60]',  -- 重试间隔（秒）
    
    -- 统计信息
    total_calls INTEGER DEFAULT 0,
    success_calls INTEGER DEFAULT 0,
    failed_calls INTEGER DEFAULT 0,
    last_called_at TIMESTAMP,
    last_status VARCHAR(20),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_enabled (enabled)
);
```

#### 3.2.8 Webhook 调用日志表 (webhook_logs)

```sql
CREATE TABLE webhook_logs (
    id BIGSERIAL PRIMARY KEY,
    webhook_id BIGINT NOT NULL,
    
    -- 请求信息
    request_url TEXT NOT NULL,
    request_method VARCHAR(10),
    request_headers TEXT,
    request_body TEXT,
    
    -- 响应信息
    response_status INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    
    -- 结果
    success BOOLEAN,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_webhook_id (webhook_id),
    INDEX idx_created_at (created_at DESC),
    
    FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
);
```

#### 3.2.9 同步日志表 (sync_logs)

```sql
CREATE TABLE sync_logs (
    id BIGSERIAL PRIMARY KEY,
    account_uid VARCHAR(64) NOT NULL,
    
    -- 同步信息
    sync_type VARCHAR(20) NOT NULL,               -- scheduled/manual
    status VARCHAR(20) NOT NULL,                  -- running/success/failed
    
    -- 统计信息
    emails_fetched INTEGER DEFAULT 0,
    emails_new INTEGER DEFAULT 0,
    emails_updated INTEGER DEFAULT 0,
    
    -- 时间信息
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    
    -- 错误信息
    error_message TEXT,
    error_stack TEXT,
    
    INDEX idx_account_uid (account_uid),
    INDEX idx_status (status),
    INDEX idx_started_at (started_at DESC),
    
    FOREIGN KEY (account_uid) REFERENCES accounts(uid) ON DELETE CASCADE
);
```

#### 3.2.10 API 密钥表 (api_keys)

```sql
CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    key_hash VARCHAR(64) UNIQUE NOT NULL,         -- API Key 的 SHA-256 哈希
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 权限配置
    permissions TEXT,                             -- 权限列表（JSON 数组）
    /*
    ["emails:read", "emails:write", "accounts:read", "sync:trigger"]
    */
    
    -- 速率限制
    rate_limit INTEGER DEFAULT 100,               -- 每分钟请求数
    
    -- 状态
    enabled BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP,
    
    -- 统计信息
    total_requests INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    last_ip VARCHAR(45),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_key_hash (key_hash),
    INDEX idx_enabled (enabled)
);
```

#### 3.2.11 用户表 (users)

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,          -- bcrypt 哈希
    
    -- 个人信息
    display_name VARCHAR(255),
    avatar_url TEXT,
    
    -- 设置
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    theme VARCHAR(20) DEFAULT 'light',            -- light/dark
    
    -- 状态
    is_active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    
    -- 安全
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(45),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_username (username),
    INDEX idx_email (email)
);
```

### 3.3 数据关系图

```
users (1) ──────────── (N) accounts
                           │
                           │ (1)
                           │
                           ↓
                        (N) emails
                           │
                           ├─── (N) email_attachments
                           │
                           └─── (N) email_label_relations ──── (N) email_labels

email_rules (N) ──────────── (N) webhooks

accounts (1) ──────────── (N) sync_logs

webhooks (1) ──────────── (N) webhook_logs
```

---

## 4. 核心模块设计

### 4.1 邮箱协议适配层设计

#### 4.1.1 统一接口定义

```go
package adapter

import (
    "context"
    "time"
)

// MailProvider 邮箱服务提供商统一接口
type MailProvider interface {
    // Connect 建立连接
    Connect(ctx context.Context, config *AccountConfig) error
    
    // FetchEmails 拉取邮件（增量同步）
    FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error)
    
    // GetEmailDetail 获取邮件详情
    GetEmailDetail(ctx context.Context, providerID string) (*EmailDetail, error)
    
    // TestConnection 测试连接
    TestConnection(ctx context.Context) error
    
    // Disconnect 断开连接
    Disconnect() error
    
    // GetProviderType 获取提供商类型
    GetProviderType() ProviderType
}

// AccountConfig 账户配置
type AccountConfig struct {
    Provider    ProviderType
    Email       string
    AuthType    AuthType
    Credentials map[string]string
    ProxyConfig *ProxyConfig
}

// ProxyConfig 代理配置
type ProxyConfig struct {
    Enabled  bool
    Type     string // http/socks5
    Host     string
    Port     int
    Username string
    Password string
}

// Email 邮件基本信息
type Email struct {
    ProviderID   string
    MessageID    string
    Subject      string
    From         *EmailAddress
    To           []*EmailAddress
    Cc           []*EmailAddress
    Bcc          []*EmailAddress
    SentAt       time.Time
    ReceivedAt   time.Time
    HasAttachments bool
    Snippet      string
    // ... 其他字段
}

// EmailDetail 邮件详情
type EmailDetail struct {
    *Email
    TextBody    string
    HTMLBody    string
    Attachments []*Attachment
    Headers     map[string]string
}

// ProviderType 提供商类型
type ProviderType string

const (
    ProviderGmail     ProviderType = "gmail"
    ProviderOutlook   ProviderType = "outlook"
    ProviderIMAP      ProviderType = "imap"
    ProviderPOP3      ProviderType = "pop3"
)

// AuthType 认证类型
type AuthType string

const (
    AuthOAuth2       AuthType = "oauth2"
    AuthPassword     AuthType = "password"
    AuthAppPassword  AuthType = "app_password"
)
```

#### 4.1.2 适配器工厂

```go
package adapter

// ProviderFactory 适配器工厂
type ProviderFactory struct {
    adapters map[ProviderType]func(*AccountConfig) (MailProvider, error)
}

// NewProviderFactory 创建工厂实例
func NewProviderFactory() *ProviderFactory {
    factory := &ProviderFactory{
        adapters: make(map[ProviderType]func(*AccountConfig) (MailProvider, error)),
    }
    
    // 注册适配器
    factory.Register(ProviderGmail, NewGmailAdapter)
    factory.Register(ProviderOutlook, NewGraphAdapter)
    factory.Register(ProviderIMAP, NewIMAPAdapter)
    factory.Register(ProviderPOP3, NewPOP3Adapter)
    
    return factory
}

// Register 注册适配器
func (f *ProviderFactory) Register(
    providerType ProviderType,
    constructor func(*AccountConfig) (MailProvider, error),
) {
    f.adapters[providerType] = constructor
}

// Create 创建适配器实例
func (f *ProviderFactory) Create(config *AccountConfig) (MailProvider, error) {
    constructor, exists := f.adapters[config.Provider]
    if !exists {
        return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
    }
    return constructor(config)
}
```

#### 4.1.3 Gmail API 适配器

```go
package adapter

import (
    "context"
    "golang.org/x/oauth2"
    "google.golang.org/api/gmail/v1"
)

type GmailAdapter struct {
    config  *AccountConfig
    service *gmail.Service
    client  *http.Client
}

func NewGmailAdapter(config *AccountConfig) (MailProvider, error) {
    return &GmailAdapter{config: config}, nil
}

func (a *GmailAdapter) Connect(ctx context.Context, config *AccountConfig) error {
    // OAuth2 配置
    oauth2Config := &oauth2.Config{
        ClientID:     config.Credentials["client_id"],
        ClientSecret: config.Credentials["client_secret"],
        Endpoint:     google.Endpoint,
        Scopes:       []string{gmail.GmailReadonlyScope},
    }
    
    // 创建 token
    token := &oauth2.Token{
        AccessToken:  config.Credentials["access_token"],
        RefreshToken: config.Credentials["refresh_token"],
    }
    
    // 创建 HTTP 客户端（支持代理）
    a.client = oauth2Config.Client(ctx, token)
    if config.ProxyConfig != nil && config.ProxyConfig.Enabled {
        a.client.Transport = createProxyTransport(config.ProxyConfig)
    }
    
    // 创建 Gmail 服务
    service, err := gmail.NewService(ctx, option.WithHTTPClient(a.client))
    if err != nil {
        return err
    }
    a.service = service
    
    return nil
}

func (a *GmailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
    // 构建查询条件
    query := fmt.Sprintf("after:%d", since.Unix())
    
    // 调用 Gmail API
    call := a.service.Users.Messages.List("me").Q(query).MaxResults(int64(limit))
    response, err := call.Do()
    if err != nil {
        return nil, err
    }
    
    // 转换为统一格式
    emails := make([]*Email, 0, len(response.Messages))
    for _, msg := range response.Messages {
        email, err := a.convertGmailMessage(ctx, msg.Id)
        if err != nil {
            continue // 跳过错误的邮件
        }
        emails = append(emails, email)
    }
    
    return emails, nil
}

// ... 其他方法实现
```

### 4.2 邮件同步引擎设计

#### 4.2.1 同步架构

```
┌─────────────────────────────────────────────────────────┐
│                    Sync Scheduler                        │
│              (定时调度器 - Cron Job)                      │
└─────────────────────────────────────────────────────────┘
                          │
                          ↓ 创建同步任务
┌─────────────────────────────────────────────────────────┐
│                    Redis Task Queue                      │
│                  (任务队列 - List)                        │
└─────────────────────────────────────────────────────────┘
                          │
                          ↓ 消费任务
┌─────────────────────────────────────────────────────────┐
│                    Sync Worker Pool                      │
│              (工作池 - Goroutine Pool)                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ Worker 1 │  │ Worker 2 │  │ Worker N │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
                          │
                          ↓ 执行同步
┌─────────────────────────────────────────────────────────┐
│                  Mail Provider Adapter                   │
│              (邮箱协议适配器)                             │
└─────────────────────────────────────────────────────────┘
                          │
                          ↓ 存储邮件
┌─────────────────────────────────────────────────────────┐
│                    Email Repository                      │
│                  (邮件数据仓库)                           │
└─────────────────────────────────────────────────────────┘
                          │
                          ↓ 触发事件
┌─────────────────────────────────────────────────────────┐
│                    Event Bus                             │
│              (事件总线 - Redis Pub/Sub)                   │
└─────────────────────────────────────────────────────────┘
                          │
                          ├─→ Rule Engine (规则引擎)
                          └─→ Webhook Service (Webhook 服务)
```

#### 4.2.2 同步流程

```go
package sync

type SyncEngine struct {
    workerPool    *WorkerPool
    taskQueue     *TaskQueue
    providerFactory *adapter.ProviderFactory
    emailRepo     *repository.EmailRepository
    eventBus      *event.Bus
}

// SyncTask 同步任务
type SyncTask struct {
    AccountUID  string
    SyncType    string // scheduled/manual
    Priority    int
    CreatedAt   time.Time
}

// ExecuteSync 执行同步
func (e *SyncEngine) ExecuteSync(ctx context.Context, task *SyncTask) error {
    // 1. 获取账户配置
    account, err := e.getAccount(task.AccountUID)
    if err != nil {
        return err
    }
    
    // 2. 创建同步日志
    syncLog := e.createSyncLog(account, task.SyncType)
    defer e.completeSyncLog(syncLog)
    
    // 3. 创建邮箱适配器
    provider, err := e.providerFactory.Create(account.ToConfig())
    if err != nil {
        return err
    }
    defer provider.Disconnect()
    
    // 4. 连接邮箱
    if err := provider.Connect(ctx, account.ToConfig()); err != nil {
        return err
    }
    
    // 5. 增量拉取邮件
    since := account.LastSyncAt
    if since.IsZero() {
        since = time.Now().AddDate(0, -1, 0) // 默认拉取最近 1 个月
    }
    
    emails, err := provider.FetchEmails(ctx, since, 100)
    if err != nil {
        return err
    }
    
    // 6. 处理邮件
    for _, email := range emails {
        if err := e.processEmail(ctx, account, email); err != nil {
            log.Errorf("Failed to process email: %v", err)
            continue
        }
    }
    
    // 7. 更新同步状态
    account.LastSyncAt = time.Now()
    account.LastSyncStatus = "success"
    e.updateAccount(account)
    
    return nil
}

// processEmail 处理单封邮件
func (e *SyncEngine) processEmail(ctx context.Context, account *Account, email *Email) error {
    // 1. 去重检查
    exists, err := e.emailRepo.ExistsByProviderID(email.ProviderID, account.UID)
    if err != nil {
        return err
    }
    if exists {
        return nil // 邮件已存在，跳过
    }
    
    // 2. 保存邮件
    savedEmail, err := e.emailRepo.Create(email)
    if err != nil {
        return err
    }
    
    // 3. 发布邮件接收事件
    e.eventBus.Publish("email.received", &EmailReceivedEvent{
        Email:      savedEmail,
        AccountUID: account.UID,
    })
    
    return nil
}
```

### 4.3 RESTful API 设计

#### 4.3.1 API 规范

**基础 URL**：`https://api.fusionmail.com/v1`

**认证方式**：
- JWT Token（用户登录）
- API Key（第三方集成）

**请求头**：
```
Authorization: Bearer <token>
或
X-API-Key: <api_key>
Content-Type: application/json
```

**响应格式**：
```json
{
    "success": true,
    "data": {},
    "error": null,
    "meta": {
        "timestamp": "2025-10-27T10:00:00Z",
        "request_id": "uuid"
    }
}
```

#### 4.3.2 API 端点列表

##### 认证相关

| 方法 | 端点 | 描述 |
|-----|------|------|
| POST | `/auth/login` | 用户登录 |
| POST | `/auth/logout` | 用户登出 |
| POST | `/auth/refresh` | 刷新 Token |
| GET | `/auth/me` | 获取当前用户信息 |

##### 邮箱账户管理

| 方法 | 端点 | 描述 |
|-----|------|------|
| GET | `/accounts` | 获取账户列表 |
| POST | `/accounts` | 添加邮箱账户 |
| GET | `/accounts/:uid` | 获取账户详情 |
| PUT | `/accounts/:uid` | 更新账户配置 |
| DELETE | `/accounts/:uid` | 删除账户 |
| POST | `/accounts/:uid/test` | 测试账户连接 |
| POST | `/accounts/:uid/sync` | 手动触发同步 |

##### 邮件管理

| 方法 | 端点 | 描述 |
|-----|------|------|
| GET | `/emails` | 获取邮件列表 |
| GET | `/emails/:id` | 获取邮件详情 |
| PATCH | `/emails/:id/read` | 标记已读/未读 |
| PATCH | `/emails/:id/star` | 添加/取消星标 |
| PATCH | `/emails/:id/archive` | 归档/取消归档 |
| DELETE | `/emails/:id` | 删除邮件 |
| POST | `/emails/search` | 搜索邮件 |
| POST | `/emails/batch` | 批量操作 |

##### 标签管理

| 方法 | 端点 | 描述 |
|-----|------|------|
| GET | `/labels` | 获取标签列表 |
| POST | `/labels` | 创建标签 |
| PUT | `/labels/:id` | 更新标签 |
| DELETE | `/labels/:id` | 删除标签 |
| POST | `/emails/:id/labels` | 为邮件添加标签 |
| DELETE | `/emails/:id/labels/:labelId` | 移除邮件标签 |

##### 规则管理

| 方法 | 端点 | 描述 |
|-----|------|------|
| GET | `/rules` | 获取规则列表 |
| POST | `/rules` | 创建规则 |
| GET | `/rules/:id` | 获取规则详情 |
| PUT | `/rules/:id` | 更新规则 |
| DELETE | `/rules/:id` | 删除规则 |
| PATCH | `/rules/:id/toggle` | 启用/禁用规则 |

##### Webhook 管理

| 方法 | 端点 | 描述 |
|-----|------|------|
| GET | `/webhooks` | 获取 Webhook 列表 |
| POST | `/webhooks` | 创建 Webhook |
| GET | `/webhooks/:id` | 获取 Webhook 详情 |
| PUT | `/webhooks/:id` | 更新 Webhook |
| DELETE | `/webhooks/:id` | 删除 Webhook |
| POST | `/webhooks/:id/test` | 测试 Webhook |
| GET | `/webhooks/:id/logs` | 获取调用日志 |

##### 系统管理

| 方法 | 端点 | 描述 |
|-----|------|------|
| GET | `/system/health` | 健康检查 |
| GET | `/system/stats` | 系统统计 |
| GET | `/system/logs` | 系统日志 |
| GET | `/sync/status` | 同步状态 |
| GET | `/sync/logs` | 同步日志 |

#### 4.3.3 API 示例

**获取邮件列表**

```http
GET /api/v1/emails?page=1&limit=20&account_uid=xxx&is_read=false
Authorization: Bearer <token>
```

响应：
```json
{
    "success": true,
    "data": {
        "emails": [
            {
                "id": 123,
                "provider_id": "msg_xxx",
                "account_uid": "acc_xxx",
                "subject": "Welcome to FusionMail",
                "from": {
                    "address": "hello@fusionmail.com",
                    "name": "FusionMail Team"
                },
                "snippet": "Thank you for using FusionMail...",
                "is_read": false,
                "is_starred": false,
                "has_attachments": true,
                "sent_at": "2025-10-27T10:00:00Z",
                "labels": ["重要"]
            }
        ],
        "pagination": {
            "page": 1,
            "limit": 20,
            "total": 150,
            "total_pages": 8
        }
    }
}
```

**添加邮箱账户**

```http
POST /api/v1/accounts
Authorization: Bearer <token>
Content-Type: application/json

{
    "email": "user@gmail.com",
    "provider": "gmail",
    "auth_type": "oauth2",
    "credentials": {
        "access_token": "xxx",
        "refresh_token": "yyy",
        "client_id": "zzz",
        "client_secret": "aaa"
    },
    "proxy_config": {
        "enabled": true,
        "type": "http",
        "host": "proxy.example.com",
        "port": 8080
    },
    "sync_interval": 5
}
```

响应：
```json
{
    "success": true,
    "data": {
        "uid": "acc_xxx",
        "email": "user@gmail.com",
        "provider": "gmail",
        "protocol": "gmail_api",
        "sync_enabled": true,
        "sync_interval": 5,
        "created_at": "2025-10-27T10:00:00Z"
    }
}
```

---

## 5. 前端架构设计

### 5.1 前端技术栈

- **React 19.2.0**：组件化开发
- **TypeScript 5.9.3**：类型安全
- **Vite 7.1.7**：快速构建
- **Tailwind CSS 4.1.14**：样式管理
- **shadcn/ui**：UI 组件库
- **React Router 6+**：路由管理
- **Axios**：HTTP 客户端
- **Zustand 4+**：状态管理

### 5.2 项目结构

```
frontend/
├── public/                 # 静态资源
├── src/
│   ├── api/               # API 调用封装
│   │   ├── client.ts      # Axios 实例配置
│   │   ├── accounts.ts    # 账户相关 API
│   │   ├── emails.ts      # 邮件相关 API
│   │   ├── rules.ts       # 规则相关 API
│   │   └── webhooks.ts    # Webhook 相关 API
│   │
│   ├── components/        # 可复用组件
│   │   ├── ui/           # shadcn/ui 组件
│   │   ├── layout/       # 布局组件
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── MainLayout.tsx
│   │   ├── email/        # 邮件相关组件
│   │   │   ├── EmailList.tsx
│   │   │   ├── EmailItem.tsx
│   │   │   ├── EmailDetail.tsx
│   │   │   └── EmailToolbar.tsx
│   │   └── account/      # 账户相关组件
│   │       ├── AccountList.tsx
│   │       ├── AccountForm.tsx
│   │       └── AccountCard.tsx
│   │
│   ├── pages/            # 页面组件
│   │   ├── Login.tsx
│   │   ├── Dashboard.tsx
│   │   ├── Inbox.tsx
│   │   ├── EmailDetail.tsx
│   │   ├── Accounts.tsx
│   │   ├── Rules.tsx
│   │   ├── Webhooks.tsx
│   │   ├── Search.tsx
│   │   └── Settings.tsx
│   │
│   ├── stores/           # Zustand 状态管理
│   │   ├── authStore.ts
│   │   ├── emailStore.ts
│   │   ├── accountStore.ts
│   │   └── uiStore.ts
│   │
│   ├── hooks/            # 自定义 Hooks
│   │   ├── useAuth.ts
│   │   ├── useEmails.ts
│   │   ├── useAccounts.ts
│   │   └── useDebounce.ts
│   │
│   ├── types/            # TypeScript 类型定义
│   │   ├── email.ts
│   │   ├── account.ts
│   │   ├── rule.ts
│   │   └── api.ts
│   │
│   ├── utils/            # 工具函数
│   │   ├── format.ts     # 格式化函数
│   │   ├── validation.ts # 验证函数
│   │   └── storage.ts    # 本地存储
│   │
│   ├── router/           # 路由配置
│   │   └── index.tsx
│   │
│   ├── App.tsx           # 根组件
│   ├── main.tsx          # 入口文件
│   └── index.css         # 全局样式
│
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

### 5.3 核心页面设计

#### 5.3.1 统一收件箱页面 (Inbox)

**功能**：
- 显示所有邮箱的聚合邮件列表
- 支持筛选（按账户、已读/未读、星标、标签）
- 支持排序（按时间、发件人）
- 支持批量操作
- 虚拟滚动加载大量邮件
- 实时同步状态显示

**组件结构**：
```tsx
<InboxPage>
  <EmailToolbar>
    <FilterDropdown />
    <SortDropdown />
    <BatchActions />
    <SyncStatus />
  </EmailToolbar>
  
  <EmailList>
    <VirtualScroll>
      {emails.map(email => (
        <EmailItem key={email.id} email={email} />
      ))}
    </VirtualScroll>
  </EmailList>
  
  <EmailDetail email={selectedEmail} />
</InboxPage>
```

#### 5.3.2 邮箱账户管理页面 (Accounts)

**功能**：
- 显示已添加的邮箱账户列表
- 添加新邮箱账户（支持多种协议）
- 编辑账户配置（代理、同步频率）
- 测试账户连接
- 手动触发同步
- 查看同步历史

**组件结构**：
```tsx
<AccountsPage>
  <AccountToolbar>
    <AddAccountButton />
    <RefreshButton />
  </AccountToolbar>
  
  <AccountList>
    {accounts.map(account => (
      <AccountCard 
        key={account.uid}
        account={account}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onSync={handleSync}
      />
    ))}
  </AccountList>
  
  <AccountFormDialog 
    open={isDialogOpen}
    account={selectedAccount}
    onSave={handleSave}
  />
</AccountsPage>
```

#### 5.3.3 规则配置页面 (Rules)

**功能**：
- 显示邮件规则列表
- 创建/编辑规则
- 启用/禁用规则
- 调整规则优先级
- 查看规则执行日志

**组件结构**：
```tsx
<RulesPage>
  <RuleToolbar>
    <CreateRuleButton />
  </RuleToolbar>
  
  <RuleList>
    {rules.map(rule => (
      <RuleCard
        key={rule.id}
        rule={rule}
        onToggle={handleToggle}
        onEdit={handleEdit}
        onDelete={handleDelete}
      />
    ))}
  </RuleList>
  
  <RuleFormDialog
    open={isDialogOpen}
    rule={selectedRule}
    onSave={handleSave}
  />
</RulesPage>
```

### 5.4 状态管理设计

#### 5.4.1 认证状态 (authStore)

```typescript
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      
      login: async (email, password) => {
        const response = await api.auth.login({ email, password });
        set({
          user: response.user,
          token: response.token,
          isAuthenticated: true,
        });
      },
      
      logout: () => {
        set({ user: null, token: null, isAuthenticated: false });
      },
      
      refreshToken: async () => {
        const response = await api.auth.refresh();
        set({ token: response.token });
      },
    }),
    {
      name: 'auth-storage',
    }
  )
);
```

#### 5.4.2 邮件状态 (emailStore)

```typescript
import { create } from 'zustand';

interface EmailState {
  emails: Email[];
  selectedEmail: Email | null;
  filters: EmailFilters;
  pagination: Pagination;
  loading: boolean;
  
  fetchEmails: (params?: FetchParams) => Promise<void>;
  selectEmail: (id: number) => void;
  markAsRead: (id: number) => Promise<void>;
  toggleStar: (id: number) => Promise<void>;
  archiveEmail: (id: number) => Promise<void>;
  deleteEmail: (id: number) => Promise<void>;
  setFilters: (filters: Partial<EmailFilters>) => void;
}

export const useEmailStore = create<EmailState>((set, get) => ({
  emails: [],
  selectedEmail: null,
  filters: {},
  pagination: { page: 1, limit: 20, total: 0 },
  loading: false,
  
  fetchEmails: async (params) => {
    set({ loading: true });
    try {
      const response = await api.emails.list({
        ...get().filters,
        ...get().pagination,
        ...params,
      });
      set({
        emails: response.data.emails,
        pagination: response.data.pagination,
      });
    } finally {
      set({ loading: false });
    }
  },
  
  selectEmail: (id) => {
    const email = get().emails.find(e => e.id === id);
    set({ selectedEmail: email || null });
  },
  
  markAsRead: async (id) => {
    await api.emails.markAsRead(id);
    set({
      emails: get().emails.map(e =>
        e.id === id ? { ...e, is_read: true } : e
      ),
    });
  },
  
  // ... 其他方法
}));
```

---

## 6. 安全设计

### 6.1 认证与授权

#### 6.1.1 JWT 认证

**Token 结构**：
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "user_id": 123,
    "username": "user@example.com",
    "exp": 1698400000,
    "iat": 1698396400
  }
}
```

**Token 管理**：
- Access Token 有效期：15 分钟
- Refresh Token 有效期：7 天
- Token 存储：LocalStorage（前端）+ Redis（后端黑名单）

#### 6.1.2 API Key 认证

**生成规则**：
```
API Key = prefix_randomString(32)
例如：fm_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

**存储方式**：
- 数据库存储 SHA-256 哈希值
- 用户只能在创建时看到完整 Key

### 6.2 数据加密

#### 6.2.1 敏感数据加密

**加密算法**：AES-256-GCM

**加密字段**：
- 邮箱账户密码
- OAuth2 Token
- 代理密码
- API Key

**实现示例**：
```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
)

type Encryptor struct {
    key []byte
}

func NewEncryptor(key string) *Encryptor {
    return &Encryptor{key: []byte(key)}
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := gcm.NonceSize()
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}
```

#### 6.2.2 传输加密

- 强制使用 HTTPS
- TLS 1.2+
- 证书自动续期（Let's Encrypt）

### 6.3 安全防护

#### 6.3.1 速率限制

**实现方式**：Redis + Token Bucket 算法

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
)

type RateLimiter struct {
    redis *redis.Client
}

func (r *RateLimiter) Limit(limit int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := "rate_limit:" + c.ClientIP()
        
        count, err := r.redis.Incr(c.Request.Context(), key).Result()
        if err != nil {
            c.AbortWithStatus(500)
            return
        }
        
        if count == 1 {
            r.redis.Expire(c.Request.Context(), key, window)
        }
        
        if count > int64(limit) {
            c.AbortWithStatusJSON(429, gin.H{
                "error": "Too many requests",
            })
            return
        }
        
        c.Next()
    }
}
```

**限制策略**：
- 登录接口：5 次/分钟
- API 接口：100 次/分钟（可配置）
- Webhook 调用：10 次/秒

#### 6.3.2 输入验证

**验证规则**：
- 邮箱格式验证
- URL 格式验证
- SQL 注入防护（使用 ORM 参数化查询）
- XSS 防护（HTML 转义）
- CSRF 防护（Token 验证）

#### 6.3.3 CORS 配置

```go
package middleware

import (
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
    config := cors.DefaultConfig()
    config.AllowOrigins = []string{
        "https://fusionmail.com",
        "https://app.fusionmail.com",
    }
    config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
    config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-API-Key"}
    config.ExposeHeaders = []string{"Content-Length"}
    config.AllowCredentials = true
    config.MaxAge = 12 * time.Hour
    
    return cors.New(config)
}
```

---

## 7. 性能优化设计

### 7.1 缓存策略

#### 7.1.1 Redis 缓存

**缓存内容**：
- 邮件列表（按筛选条件）
- 未读邮件数量
- 账户信息
- 用户会话

**缓存策略**：
```go
// 邮件列表缓存
key: "emails:list:{account_uid}:{filters_hash}"
ttl: 5 minutes

// 未读数量缓存
key: "emails:unread:{account_uid}"
ttl: 1 minute

// 账户信息缓存
key: "account:{uid}"
ttl: 30 minutes
```

**缓存更新**：
- 新邮件到达时清除相关缓存
- 邮件状态变更时更新缓存
- 使用 Redis Pub/Sub 通知缓存失效

#### 7.1.2 浏览器缓存

**静态资源**：
- JS/CSS 文件：强缓存 1 年
- 图片资源：强缓存 30 天
- HTML 文件：协商缓存

**Service Worker**：
- 离线缓存邮件列表
- 预加载常用资源

### 7.2 数据库优化

#### 7.2.1 索引设计

**核心索引**：
```sql
-- 邮件表
CREATE INDEX idx_emails_composite ON emails(account_uid, sent_at DESC, is_deleted);
CREATE INDEX idx_emails_search ON emails USING gin(to_tsvector('english', subject || ' ' || text_body));

-- 账户表
CREATE INDEX idx_accounts_sync ON accounts(sync_enabled, last_sync_at);

-- 同步日志表
CREATE INDEX idx_sync_logs_composite ON sync_logs(account_uid, started_at DESC);
```

#### 7.2.2 查询优化

**分页查询**：
```sql
-- 使用游标分页（性能更好）
SELECT * FROM emails
WHERE account_uid = $1
  AND id < $2  -- 游标
  AND is_deleted = false
ORDER BY id DESC
LIMIT 20;
```

**统计查询**：
```sql
-- 使用物化视图
CREATE MATERIALIZED VIEW email_stats AS
SELECT 
    account_uid,
    COUNT(*) as total_count,
    COUNT(*) FILTER (WHERE is_read = false) as unread_count,
    COUNT(*) FILTER (WHERE is_starred = true) as starred_count
FROM emails
WHERE is_deleted = false
GROUP BY account_uid;

-- 定期刷新
REFRESH MATERIALIZED VIEW CONCURRENTLY email_stats;
```

### 7.3 前端性能优化

#### 7.3.1 虚拟滚动

```tsx
import { useVirtualizer } from '@tanstack/react-virtual';

function EmailList({ emails }: { emails: Email[] }) {
  const parentRef = useRef<HTMLDivElement>(null);
  
  const virtualizer = useVirtualizer({
    count: emails.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 80, // 每项高度
    overscan: 5, // 预渲染数量
  });
  
  return (
    <div ref={parentRef} className="h-screen overflow-auto">
      <div style={{ height: `${virtualizer.getTotalSize()}px` }}>
        {virtualizer.getVirtualItems().map((virtualItem) => (
          <div
            key={virtualItem.key}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualItem.size}px`,
              transform: `translateY(${virtualItem.start}px)`,
            }}
          >
            <EmailItem email={emails[virtualItem.index]} />
          </div>
        ))}
      </div>
    </div>
  );
}
```

#### 7.3.2 代码分割

```tsx
import { lazy, Suspense } from 'react';

// 路由级别代码分割
const Inbox = lazy(() => import('./pages/Inbox'));
const Accounts = lazy(() => import('./pages/Accounts'));
const Rules = lazy(() => import('./pages/Rules'));

function App() {
  return (
    <Suspense fallback={<Loading />}>
      <Routes>
        <Route path="/inbox" element={<Inbox />} />
        <Route path="/accounts" element={<Accounts />} />
        <Route path="/rules" element={<Rules />} />
      </Routes>
    </Suspense>
  );
}
```

#### 7.3.3 图片优化

- 使用 WebP 格式
- 懒加载（Intersection Observer）
- 响应式图片（srcset）
- CDN 加速

---

## 8. 部署架构设计

### 8.1 Docker 容器化

#### 8.1.1 Dockerfile

**后端 Dockerfile**：
```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o fusionmail-server ./cmd/server

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/fusionmail-server .
COPY --from=builder /app/config ./config

EXPOSE 8080
CMD ["./fusionmail-server"]
```

**前端 Dockerfile**：
```dockerfile
# 构建阶段
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# 运行阶段
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

#### 8.1.2 Docker Compose

```yaml
version: '3.8'

services:
  # PostgreSQL 数据库
  postgres:
    image: postgres:15-alpine
    container_name: fusionmail-postgres
    environment:
      POSTGRES_DB: fusionmail
      POSTGRES_USER: fusionmail
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fusionmail"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: fusionmail-redis
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # 后端服务
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: fusionmail-backend
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: fusionmail
      DB_USER: fusionmail
      DB_PASSWORD: ${DB_PASSWORD}
      REDIS_HOST: redis
      REDIS_PORT: 6379
      JWT_SECRET: ${JWT_SECRET}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  # 前端服务
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: fusionmail-frontend
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### 8.2 环境配置

#### 8.2.1 配置文件

**config/config.yaml**：
```yaml
server:
  port: 8080
  mode: production  # development/production

database:
  host: ${DB_HOST}
  port: ${DB_PORT}
  name: ${DB_NAME}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  max_open_conns: 100
  max_idle_conns: 10

redis:
  host: ${REDIS_HOST}
  port: ${REDIS_PORT}
  password: ${REDIS_PASSWORD}
  db: 0

security:
  jwt_secret: ${JWT_SECRET}
  encryption_key: ${ENCRYPTION_KEY}
  token_expire: 15m
  refresh_expire: 168h

sync:
  worker_count: 5
  max_concurrent: 10
  default_interval: 5m

logging:
  level: info  # debug/info/warn/error
  format: json
  output: stdout
```

---

## 9. 监控与日志

### 9.1 日志设计

#### 9.1.1 结构化日志

```go
package logger

import (
    "go.uber.org/zap"
)

type Logger struct {
    *zap.Logger
}

func NewLogger() *Logger {
    config := zap.NewProductionConfig()
    config.OutputPaths = []string{"stdout", "/var/log/fusionmail/app.log"}
    
    logger, _ := config.Build()
    return &Logger{logger}
}

// 使用示例
logger.Info("Email synced",
    zap.String("account_uid", "acc_xxx"),
    zap.Int("count", 10),
    zap.Duration("duration", time.Second*5),
)
```

#### 9.1.2 日志级别

- **DEBUG**：详细的调试信息
- **INFO**：一般信息（同步完成、API 调用）
- **WARN**：警告信息（重试、降级）
- **ERROR**：错误信息（同步失败、API 错误）

### 9.2 监控指标

#### 9.2.1 Prometheus 指标

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    // 邮件同步指标
    EmailsSynced = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "fusionmail_emails_synced_total",
            Help: "Total number of emails synced",
        },
        []string{"account_uid", "provider"},
    )
    
    // 同步耗时
    SyncDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "fusionmail_sync_duration_seconds",
            Help: "Sync duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"account_uid"},
    )
    
    // API 请求数
    APIRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "fusionmail_api_requests_total",
            Help: "Total number of API requests",
        },
        []string{"method", "endpoint", "status"},
    )
)
```

#### 9.2.2 健康检查

```go
package health

type HealthChecker struct {
    db    *gorm.DB
    redis *redis.Client
}

func (h *HealthChecker) Check() *HealthStatus {
    status := &HealthStatus{
        Status: "healthy",
        Checks: make(map[string]string),
    }
    
    // 检查数据库
    if err := h.db.Exec("SELECT 1").Error; err != nil {
        status.Status = "unhealthy"
        status.Checks["database"] = "failed"
    } else {
        status.Checks["database"] = "ok"
    }
    
    // 检查 Redis
    if err := h.redis.Ping(context.Background()).Err(); err != nil {
        status.Status = "unhealthy"
        status.Checks["redis"] = "failed"
    } else {
        status.Checks["redis"] = "ok"
    }
    
    return status
}
```

---

## 10. 总结

FusionMail 系统设计遵循以下核心原则：

1. **模块化设计**：各模块职责清晰，便于维护和扩展
2. **高性能**：使用缓存、索引、虚拟滚动等优化手段
3. **高可靠**：错误重试、健康检查、日志监控
4. **安全性**：数据加密、认证授权、速率限制
5. **轻量化**：单体部署，资源占用低

**下一步**：
1. 创建项目骨架
2. 实现核心模块
3. 编写单元测试
4. 部署测试环境
5. 性能测试与优化

---

**文档版本**：v1.0  
**创建日期**：2025-10-27  
**最后更新**：2025-10-27


---

## 11. 扩展性设计

### 11.1 附件存储扩展

#### 11.1.1 存储抽象层设计

为了支持多种存储方式（本地、S3、OSS 等），设计统一的存储接口：

```go
package storage

import (
    "context"
    "io"
)

// StorageProvider 存储提供商接口
type StorageProvider interface {
    // Upload 上传文件
    Upload(ctx context.Context, key string, reader io.Reader, size int64) (*UploadResult, error)
    
    // Download 下载文件
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    
    // Delete 删除文件
    Delete(ctx context.Context, key string) error
    
    // GetURL 获取访问 URL（支持预签名）
    GetURL(ctx context.Context, key string, expires time.Duration) (string, error)
    
    // Exists 检查文件是否存在
    Exists(ctx context.Context, key string) (bool, error)
    
    // GetType 获取存储类型
    GetType() StorageType
}

// StorageType 存储类型
type StorageType string

const (
    StorageLocal StorageType = "local"
    StorageS3    StorageType = "s3"
    StorageOSS   StorageType = "oss"
    StorageCOS   StorageType = "cos"  // 腾讯云 COS
    StorageOBS   StorageType = "obs"  // 华为云 OBS
)

// UploadResult 上传结果
type UploadResult struct {
    Key         string
    URL         string
    Size        int64
    ContentType string
    ETag        string
}

// StorageConfig 存储配置
type StorageConfig struct {
    Type       StorageType
    Endpoint   string
    Bucket     string
    Region     string
    AccessKey  string
    SecretKey  string
    BasePath   string
    PublicURL  string
}
```

#### 11.1.2 本地存储实现

```go
package storage

import (
    "context"
    "io"
    "os"
    "path/filepath"
)

type LocalStorage struct {
    basePath  string
    publicURL string
}

func NewLocalStorage(config *StorageConfig) (*LocalStorage, error) {
    // 确保目录存在
    if err := os.MkdirAll(config.BasePath, 0755); err != nil {
        return nil, err
    }
    
    return &LocalStorage{
        basePath:  config.BasePath,
        publicURL: config.PublicURL,
    }, nil
}

func (s *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64) (*UploadResult, error) {
    filePath := filepath.Join(s.basePath, key)
    
    // 创建目录
    if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
        return nil, err
    }
    
    // 创建文件
    file, err := os.Create(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    // 写入数据
    written, err := io.Copy(file, reader)
    if err != nil {
        return nil, err
    }
    
    return &UploadResult{
        Key:  key,
        URL:  s.publicURL + "/" + key,
        Size: written,
    }, nil
}

func (s *LocalStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
    filePath := filepath.Join(s.basePath, key)
    return os.Open(filePath)
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
    filePath := filepath.Join(s.basePath, key)
    return os.Remove(filePath)
}

func (s *LocalStorage) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
    return s.publicURL + "/" + key, nil
}

func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
    filePath := filepath.Join(s.basePath, key)
    _, err := os.Stat(filePath)
    if os.IsNotExist(err) {
        return false, nil
    }
    return err == nil, err
}

func (s *LocalStorage) GetType() StorageType {
    return StorageLocal
}
```

#### 11.1.3 S3 存储实现

```go
package storage

import (
    "context"
    "io"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
    client *s3.Client
    bucket string
    region string
}

func NewS3Storage(cfg *StorageConfig) (*S3Storage, error) {
    // 加载 AWS 配置
    awsConfig, err := config.LoadDefaultConfig(context.Background(),
        config.WithRegion(cfg.Region),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
            cfg.AccessKey,
            cfg.SecretKey,
            "",
        )),
    )
    if err != nil {
        return nil, err
    }
    
    // 创建 S3 客户端
    client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
        if cfg.Endpoint != "" {
            o.BaseEndpoint = aws.String(cfg.Endpoint)
        }
    })
    
    return &S3Storage{
        client: client,
        bucket: cfg.Bucket,
        region: cfg.Region,
    }, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64) (*UploadResult, error) {
    // 上传到 S3
    output, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:   reader,
    })
    if err != nil {
        return nil, err
    }
    
    // 获取 URL
    url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
    
    return &UploadResult{
        Key:  key,
        URL:  url,
        Size: size,
        ETag: aws.ToString(output.ETag),
    }, nil
}

func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
    output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, err
    }
    return output.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
    _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    return err
}

func (s *S3Storage) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
    // 生成预签名 URL
    presignClient := s3.NewPresignClient(s.client)
    request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    }, func(opts *s3.PresignOptions) {
        opts.Expires = expires
    })
    if err != nil {
        return "", err
    }
    return request.URL, nil
}

func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
    _, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        // 检查是否是 NotFound 错误
        return false, nil
    }
    return true, nil
}

func (s *S3Storage) GetType() StorageType {
    return StorageS3
}
```

#### 11.1.4 阿里云 OSS 存储实现

```go
package storage

import (
    "context"
    "io"
    "time"
    
    "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSStorage struct {
    client *oss.Client
    bucket *oss.Bucket
}

func NewOSSStorage(cfg *StorageConfig) (*OSSStorage, error) {
    // 创建 OSS 客户端
    client, err := oss.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey)
    if err != nil {
        return nil, err
    }
    
    // 获取 Bucket
    bucket, err := client.Bucket(cfg.Bucket)
    if err != nil {
        return nil, err
    }
    
    return &OSSStorage{
        client: client,
        bucket: bucket,
    }, nil
}

func (s *OSSStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64) (*UploadResult, error) {
    // 上传到 OSS
    err := s.bucket.PutObject(key, reader)
    if err != nil {
        return nil, err
    }
    
    // 获取 URL
    url := s.bucket.GetConfig().Endpoint + "/" + key
    
    return &UploadResult{
        Key:  key,
        URL:  url,
        Size: size,
    }, nil
}

func (s *OSSStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
    return s.bucket.GetObject(key)
}

func (s *OSSStorage) Delete(ctx context.Context, key string) error {
    return s.bucket.DeleteObject(key)
}

func (s *OSSStorage) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
    // 生成预签名 URL
    return s.bucket.SignURL(key, oss.HTTPGet, int64(expires.Seconds()))
}

func (s *OSSStorage) Exists(ctx context.Context, key string) (bool, error) {
    return s.bucket.IsObjectExist(key)
}

func (s *OSSStorage) GetType() StorageType {
    return StorageOSS
}
```

#### 11.1.5 存储工厂

```go
package storage

type StorageFactory struct {
    providers map[StorageType]func(*StorageConfig) (StorageProvider, error)
}

func NewStorageFactory() *StorageFactory {
    factory := &StorageFactory{
        providers: make(map[StorageType]func(*StorageConfig) (StorageProvider, error)),
    }
    
    // 注册存储提供商
    factory.Register(StorageLocal, func(cfg *StorageConfig) (StorageProvider, error) {
        return NewLocalStorage(cfg)
    })
    factory.Register(StorageS3, func(cfg *StorageConfig) (StorageProvider, error) {
        return NewS3Storage(cfg)
    })
    factory.Register(StorageOSS, func(cfg *StorageConfig) (StorageProvider, error) {
        return NewOSSStorage(cfg)
    })
    
    return factory
}

func (f *StorageFactory) Register(
    storageType StorageType,
    constructor func(*StorageConfig) (StorageProvider, error),
) {
    f.providers[storageType] = constructor
}

func (f *StorageFactory) Create(config *StorageConfig) (StorageProvider, error) {
    constructor, exists := f.providers[config.Type]
    if !exists {
        return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
    }
    return constructor(config)
}
```

#### 11.1.6 配置示例

```yaml
# config/config.yaml
storage:
  # 存储类型：local/s3/oss/cos/obs
  # 默认：local（本地存储）
  type: local
  
  # 本地存储配置（默认）
  local:
    base_path: /var/fusionmail/attachments
    public_url: https://fusionmail.com/attachments
  
  # S3 存储配置（可选）
  s3:
    endpoint: https://s3.amazonaws.com
    bucket: fusionmail-attachments
    region: us-east-1
    access_key: ${AWS_ACCESS_KEY}
    secret_key: ${AWS_SECRET_KEY}
  
  # 阿里云 OSS 配置（可选）
  oss:
    endpoint: https://oss-cn-hangzhou.aliyuncs.com
    bucket: fusionmail-attachments
    access_key: ${OSS_ACCESS_KEY}
    secret_key: ${OSS_SECRET_KEY}
```

**默认配置说明**：
- 系统默认使用**本地存储**（`type: local`）
- 本地存储适合个人和小团队使用，无需额外配置
- 对象存储（S3/OSS）为可选配置，适合大规模部署或需要高可用性的场景
- 可以在运行时通过配置文件切换存储类型，无需修改代码

### 11.2 邮件发送扩展点设计

虽然当前版本暂缓邮件发送功能，但在架构上预留扩展点，便于未来实现。

#### 11.2.1 邮件发送接口设计

```go
package sender

import (
    "context"
)

// MailSender 邮件发送接口
type MailSender interface {
    // Send 发送邮件
    Send(ctx context.Context, message *OutgoingMessage) (*SendResult, error)
    
    // SendBatch 批量发送
    SendBatch(ctx context.Context, messages []*OutgoingMessage) ([]*SendResult, error)
    
    // GetSenderType 获取发送器类型
    GetSenderType() SenderType
}

// OutgoingMessage 待发送邮件
type OutgoingMessage struct {
    AccountUID  string           // 发送账户
    From        *EmailAddress    // 发件人
    To          []*EmailAddress  // 收件人
    Cc          []*EmailAddress  // 抄送
    Bcc         []*EmailAddress  // 密送
    Subject     string           // 主题
    TextBody    string           // 纯文本正文
    HTMLBody    string           // HTML 正文
    Attachments []*Attachment    // 附件
    InReplyTo   string           // 回复的邮件 ID
    References  []string         // 引用的邮件 ID 列表
    Headers     map[string]string // 自定义头部
}

// SendResult 发送结果
type SendResult struct {
    MessageID   string
    Status      SendStatus
    Error       error
    SentAt      time.Time
}

// SendStatus 发送状态
type SendStatus string

const (
    SendStatusSuccess SendStatus = "success"
    SendStatusFailed  SendStatus = "failed"
    SendStatusQueued  SendStatus = "queued"
)

// SenderType 发送器类型
type SenderType string

const (
    SenderGmail   SenderType = "gmail"
    SenderGraph   SenderType = "graph"
    SenderSMTP    SenderType = "smtp"
)
```

#### 11.2.2 SMTP 发送器实现（示例）

```go
package sender

import (
    "context"
    "crypto/tls"
    "net/smtp"
)

type SMTPSender struct {
    host     string
    port     int
    username string
    password string
    useTLS   bool
}

func NewSMTPSender(config *SMTPConfig) *SMTPSender {
    return &SMTPSender{
        host:     config.Host,
        port:     config.Port,
        username: config.Username,
        password: config.Password,
        useTLS:   config.UseTLS,
    }
}

func (s *SMTPSender) Send(ctx context.Context, message *OutgoingMessage) (*SendResult, error) {
    // 构建邮件内容
    emailContent := s.buildEmailContent(message)
    
    // 连接 SMTP 服务器
    addr := fmt.Sprintf("%s:%d", s.host, s.port)
    
    var client *smtp.Client
    var err error
    
    if s.useTLS {
        // TLS 连接
        tlsConfig := &tls.Config{
            ServerName: s.host,
        }
        conn, err := tls.Dial("tcp", addr, tlsConfig)
        if err != nil {
            return nil, err
        }
        client, err = smtp.NewClient(conn, s.host)
    } else {
        // 普通连接
        client, err = smtp.Dial(addr)
    }
    
    if err != nil {
        return &SendResult{
            Status: SendStatusFailed,
            Error:  err,
        }, err
    }
    defer client.Close()
    
    // 认证
    auth := smtp.PlainAuth("", s.username, s.password, s.host)
    if err := client.Auth(auth); err != nil {
        return &SendResult{
            Status: SendStatusFailed,
            Error:  err,
        }, err
    }
    
    // 设置发件人
    if err := client.Mail(message.From.Address); err != nil {
        return &SendResult{
            Status: SendStatusFailed,
            Error:  err,
        }, err
    }
    
    // 设置收件人
    for _, to := range message.To {
        if err := client.Rcpt(to.Address); err != nil {
            return &SendResult{
                Status: SendStatusFailed,
                Error:  err,
            }, err
        }
    }
    
    // 发送邮件内容
    writer, err := client.Data()
    if err != nil {
        return &SendResult{
            Status: SendStatusFailed,
            Error:  err,
        }, err
    }
    
    _, err = writer.Write([]byte(emailContent))
    if err != nil {
        return &SendResult{
            Status: SendStatusFailed,
            Error:  err,
        }, err
    }
    
    err = writer.Close()
    if err != nil {
        return &SendResult{
            Status: SendStatusFailed,
            Error:  err,
        }, err
    }
    
    return &SendResult{
        Status: SendStatusSuccess,
        SentAt: time.Now(),
    }, nil
}

func (s *SMTPSender) buildEmailContent(message *OutgoingMessage) string {
    // 构建符合 RFC 5322 的邮件内容
    // 包括头部、正文、附件等
    // 这里简化处理，实际应使用专业的邮件构建库
    return fmt.Sprintf(
        "From: %s\r\n"+
        "To: %s\r\n"+
        "Subject: %s\r\n"+
        "Content-Type: text/plain; charset=UTF-8\r\n"+
        "\r\n"+
        "%s",
        message.From.Address,
        message.To[0].Address,
        message.Subject,
        message.TextBody,
    )
}

func (s *SMTPSender) GetSenderType() SenderType {
    return SenderSMTP
}
```

#### 11.2.3 发送器工厂

```go
package sender

type SenderFactory struct {
    senders map[SenderType]func(interface{}) (MailSender, error)
}

func NewSenderFactory() *SenderFactory {
    factory := &SenderFactory{
        senders: make(map[SenderType]func(interface{}) (MailSender, error)),
    }
    
    // 注册发送器
    factory.Register(SenderSMTP, func(config interface{}) (MailSender, error) {
        return NewSMTPSender(config.(*SMTPConfig)), nil
    })
    
    // 未来可以添加更多发送器
    // factory.Register(SenderGmail, NewGmailSender)
    // factory.Register(SenderGraph, NewGraphSender)
    
    return factory
}

func (f *SenderFactory) Register(
    senderType SenderType,
    constructor func(interface{}) (MailSender, error),
) {
    f.senders[senderType] = constructor
}

func (f *SenderFactory) Create(senderType SenderType, config interface{}) (MailSender, error) {
    constructor, exists := f.senders[senderType]
    if !exists {
        return nil, fmt.Errorf("unsupported sender type: %s", senderType)
    }
    return constructor(config)
}
```

#### 11.2.4 邮件发送服务

```go
package service

type EmailSendService struct {
    senderFactory *sender.SenderFactory
    accountRepo   *repository.AccountRepository
    emailRepo     *repository.EmailRepository
}

// SendEmail 发送邮件（未来实现）
func (s *EmailSendService) SendEmail(ctx context.Context, message *sender.OutgoingMessage) (*sender.SendResult, error) {
    // 1. 获取账户配置
    account, err := s.accountRepo.GetByUID(message.AccountUID)
    if err != nil {
        return nil, err
    }
    
    // 2. 创建发送器
    mailSender, err := s.senderFactory.Create(
        sender.SenderType(account.Protocol),
        account.GetSenderConfig(),
    )
    if err != nil {
        return nil, err
    }
    
    // 3. 发送邮件
    result, err := mailSender.Send(ctx, message)
    if err != nil {
        return result, err
    }
    
    // 4. 保存到已发送文件夹（可选）
    if result.Status == sender.SendStatusSuccess {
        s.saveSentEmail(ctx, message, result)
    }
    
    return result, nil
}

// saveSentEmail 保存已发送邮件
func (s *EmailSendService) saveSentEmail(ctx context.Context, message *sender.OutgoingMessage, result *sender.SendResult) error {
    // 将发送的邮件保存到数据库
    // 标记为已发送状态
    // 这样用户可以在"已发送"文件夹中查看
    return nil
}
```

#### 11.2.5 数据表扩展

为支持邮件发送，需要在 `emails` 表中添加字段：

```sql
ALTER TABLE emails ADD COLUMN email_type VARCHAR(20) DEFAULT 'received';
-- email_type: received/sent/draft

ALTER TABLE emails ADD COLUMN draft_data TEXT;
-- 草稿数据（JSON 格式）

CREATE TABLE sent_emails (
    id BIGSERIAL PRIMARY KEY,
    account_uid VARCHAR(64) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    
    -- 发送信息
    to_addresses TEXT NOT NULL,
    cc_addresses TEXT,
    bcc_addresses TEXT,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    
    -- 发送状态
    status VARCHAR(20) NOT NULL,  -- success/failed/queued
    sent_at TIMESTAMP,
    error_message TEXT,
    
    -- 关联信息
    in_reply_to VARCHAR(255),
    thread_id VARCHAR(255),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_account_uid (account_uid),
    INDEX idx_sent_at (sent_at DESC),
    
    FOREIGN KEY (account_uid) REFERENCES accounts(uid) ON DELETE CASCADE
);
```

#### 11.2.6 API 端点预留

```go
// 未来的邮件发送 API 端点
// POST /api/v1/emails/send
// POST /api/v1/emails/reply/:id
// POST /api/v1/emails/forward/:id
// POST /api/v1/drafts
// PUT /api/v1/drafts/:id
// DELETE /api/v1/drafts/:id
```

### 11.3 扩展性总结

通过以上设计，FusionMail 在架构上具备良好的扩展性：

1. **存储扩展**：
   - 统一的 `StorageProvider` 接口
   - 支持本地、S3、OSS 等多种存储
   - 通过工厂模式轻松添加新的存储类型

2. **邮件发送扩展**：
   - 预留 `MailSender` 接口
   - 支持 SMTP、Gmail API、Microsoft Graph 等
   - 数据表已考虑发送邮件的存储需求

3. **协议扩展**：
   - `MailProvider` 接口支持添加新的邮箱协议
   - 适配器模式便于扩展

4. **功能扩展**：
   - 模块化设计，各模块职责清晰
   - 事件驱动架构，便于添加新的业务逻辑

---

**文档版本**：v1.1  
**创建日期**：2025-10-27  
**最后更新**：2025-10-27  
**变更记录**：
- v1.1 (2025-10-27): 添加附件存储扩展设计（支持 S3/OSS）和邮件发送扩展点设计
- v1.0 (2025-10-27): 初始版本
