# FusionMail 技术细节分析

---

## 1. 后端技术栈详解

### 1.1 Web 框架 (Gin)

**特点**:
- 轻量级、高性能
- 内置中间件系统
- 路由分组支持

**路由结构**:
```
/api/v1/
├── /health (无需认证)
├── /system/providers (无需认证)
├── /auth/ (速率限制)
│   ├── POST /login
│   ├── POST /logout
│   ├── GET /google/authorize
│   └── GET /google/callback
└── /protected (需要认证)
    ├── /accounts
    ├── /emails
    ├── /rules
    ├── /webhooks
    └── /sync
```

### 1.2 ORM (GORM)

**特点**:
- 自动迁移支持
- 关联关系管理
- 钩子函数支持

**模型关系**:
- Account 1:N Email
- Email 1:N EmailAttachment
- Email M:N EmailLabel
- Account 1:N EmailRule
- Webhook 1:N WebhookLog

### 1.3 数据库 (PostgreSQL)

**性能优化**:
- 复合唯一索引 (provider_id, account_uid)
- 全文搜索 GIN 索引
- 降序索引 (sent_at DESC)
- 外键约束和级联删除

**特性使用**:
- JSON 字段存储规则条件和动作
- TIMESTAMP 字段记录时间信息
- TEXT 字段存储大文本内容

### 1.4 缓存 (Redis)

**用途**:
- 速率限制计数
- 会话存储
- 任务队列 (未来)

**配置**:
- 最大内存: 512MB
- 淘汰策略: allkeys-lru
- 持久化: AOF (Append Only File)

---

## 2. 邮箱适配器深度分析

### 2.1 IMAP 适配器

**库**: github.com/emersion/go-imap/v2

**特点**:
- 完整的 IMAP 协议支持
- 支持增量同步 (SEARCH 命令)
- 支持文件夹管理
- 支持 TLS/SSL

**关键方法**:
```go
Connect() - 建立 TLS 连接并登录
FetchEmails() - 使用 SEARCH 和 FETCH 获取邮件
FetchEmailDetail() - 获取单封邮件详情
TestConnection() - 列出邮箱验证连接
```

### 2.2 POP3 适配器

**库**: github.com/knadh/go-pop3

**限制**:
- 不支持增量同步 (客户端过滤)
- 不支持代理 (库限制)
- 功能有限

**适用场景**:
- 国内邮箱 (QQ、163)
- 不支持 IMAP 的服务

### 2.3 Gmail API 适配器

**库**: google.golang.org/api

**特点**:
- 官方 API，性能好
- 支持 OAuth2 认证
- 支持标签和文件夹
- 有配额限制

**关键端点**:
- /users/me/messages/list - 邮件列表
- /users/me/messages/get - 邮件详情
- /users/me/messages/modify - 修改邮件

### 2.4 Microsoft Graph API 适配器

**特点**:
- 官方 API，性能好
- 支持 OAuth2 认证
- 支持分页查询
- 支持 OData 过滤

**关键端点**:
- /me/messages - 邮件列表
- /me/messages/{id} - 邮件详情
- /me/mailFolders/inbox/messages - 收件箱邮件

---

## 3. 认证与授权

### 3.1 JWT 认证

**配置**:
- 密钥: 环境变量 JWT_SECRET
- 过期时间: 24 小时 (可配置)
- 算法: HS256

**流程**:
1. 用户登录 (密码验证)
2. 生成 JWT token
3. 返回 token 和过期时间
4. 客户端存储 token
5. 后续请求在 Authorization 头中携带

### 3.2 OAuth2 认证

**支持的提供商**:
- Google (Gmail)
- Microsoft (Outlook)

**流程**:
1. 前端重定向到授权页面
2. 用户授权
3. 回调获取 authorization code
4. 后端用 code 交换 access token
5. 保存 token 用于后续 API 调用

**Token 刷新**:
- 使用 refresh token 获取新的 access token
- 自动刷新机制 (需要完善)

### 3.3 速率限制

**实现**:
- 基于 Redis 的滑动窗口算法
- 登录接口: 100 次/分钟
- 普通接口: 200 次/分钟

**响应头**:
- X-RateLimit-Limit
- X-RateLimit-Remaining
- X-RateLimit-Reset

---

## 4. 邮件同步引擎

### 4.1 同步流程

**步骤**:
1. 获取账户配置
2. 解密凭证
3. 创建适配器
4. 连接邮箱服务器
5. 确定同步起始时间
6. 拉取邮件列表
7. 去重检查
8. 保存或更新邮件
9. 应用规则
10. 记录同步日志

### 4.2 增量同步

**机制**:
- 首次同步: 从 7 天前开始
- 后续同步: 从上次同步时间 - 5 分钟开始
- 缓冲时间: 避免遗漏邮件

**去重**:
- 使用 (provider_id, account_uid) 复合唯一索引
- 检查邮件是否已存在
- 存在则更新，不存在则创建

### 4.3 定时调度

**实现**:
- 使用 Cron 定时任务
- 可配置同步频率 (1-60 分钟)
- 支持手动同步触发

**错误处理**:
- 连接失败: 记录错误，等待下次调度
- 同步失败: 记录错误日志，更新账户状态
- 认证失败: 标记账户为错误状态

---

## 5. 规则引擎实现

### 5.1 规则匹配

**匹配模式**:
- all: 所有条件都必须匹配
- any: 任意一个条件匹配即可

**条件字段**:
- from: 发件人地址
- to: 收件人地址
- subject: 邮件主题
- body: 邮件正文
- has_attachment: 是否有附件

**操作符**:
- contains: 包含 (不区分大小写)
- not_contains: 不包含
- equals: 等于 (不区分大小写)
- not_equals: 不等于
- starts_with: 开头
- ends_with: 结尾
- regex: 正则表达式

### 5.2 动作执行

**支持的动作**:
- mark_read: 标记已读
- mark_unread: 标记未读
- star: 星标
- unstar: 取消星标
- archive: 归档
- delete: 删除
- move_folder: 移动到文件夹
- add_label: 添加标签 (未完成)
- webhook: 触发 Webhook (未完成)

### 5.3 优先级处理

**排序**:
- 按 priority 字段升序排序
- 数字越小优先级越高

**停止处理**:
- stop_processing = true 时，匹配后不再处理后续规则
- 用于实现"删除规则"等终止性操作

---

## 6. 前端架构

### 6.1 状态管理 (Zustand)

**Store 结构**:
```
authStore - 认证状态
├── user
├── token
├── expiresAt
└── isAuthenticated

emailStore - 邮件状态
├── emails
├── selectedEmail
├── total
├── page
└── filter

accountStore - 账户状态
├── accounts
├── selectedAccount
└── loading

uiStore - UI 状态
├── sidebarCollapsed
├── theme
├── emailListView
└── isSyncing
```

### 6.2 API 服务层

**Axios 配置**:
- 基础 URL: /api/v1
- 超时: 30 秒
- 自动 token 刷新
- 错误拦截和处理

**服务模块**:
- authService - 认证
- emailService - 邮件
- accountService - 账户
- ruleService - 规则
- webhookService - Webhook
- systemService - 系统

### 6.3 组件架构

**组件分类**:
- UI 组件 (shadcn/ui) - 基础组件
- 布局组件 - Header、Sidebar、MainLayout
- 功能组件 - EmailList、EmailDetail、AccountForm
- 页面组件 - InboxPage、AccountsPage、RulesPage

**虚拟滚动**:
- 使用 @tanstack/react-virtual
- 支持大列表高效渲染
- 邮件列表优化

---

## 7. 安全性分析

### 7.1 凭证管理

**加密**:
- 使用 AES-256 加密凭证
- 加密密钥: 环境变量 ENCRYPTION_KEY
- 存储在数据库中的是加密后的凭证

**密码**:
- 使用 bcrypt 哈希密码
- 不存储明文密码
- 登录时验证哈希值

### 7.2 API 安全

**认证**:
- JWT token 验证
- Token 过期检查
- 无效 token 拒绝

**授权**:
- 基于账户 UID 的隔离
- 用户只能访问自己的数据
- 管理员权限检查

**速率限制**:
- 防止暴力攻击
- 防止 DDoS
- 基于 IP 和用户的限制

### 7.3 输入验证

**验证项**:
- 邮箱地址格式
- 规则条件和动作
- Webhook URL 格式
- 文件名安全清理

---

## 8. 性能考虑

### 8.1 数据库优化

**索引**:
- 复合唯一索引加速去重检查
- 全文搜索 GIN 索引
- 降序索引加速排序

**查询优化**:
- 分页查询
- 字段选择
- 关联加载

### 8.2 缓存策略

**当前**:
- Redis 用于速率限制
- 会话存储

**建议**:
- 缓存热点数据 (账户列表、规则列表)
- 缓存 API 响应
- 缓存 OAuth2 token

### 8.3 并发处理

**同步**:
- 使用 mutex 保护并发访问
- 分布式锁防止重复同步

**异步**:
- 后台同步任务
- 事件驱动架构

---

## 9. 扩展性分析

### 9.1 新邮箱提供商

**添加步骤**:
1. 创建新的 Adapter 实现 MailProvider 接口
2. 在 Factory 中注册
3. 添加提供商信息
4. 编写测试

### 9.2 新规则动作

**添加步骤**:
1. 在 RuleAction 中定义新类型
2. 在 validateAction 中添加验证
3. 在 executeAction 中实现逻辑
4. 编写测试

### 9.3 新 API 端点

**添加步骤**:
1. 在 Handler 中添加方法
2. 在 Router 中注册路由
3. 添加中间件 (认证、速率限制)
4. 编写测试

---

## 10. 已知限制

### 10.1 功能限制

- POP3 不支持增量同步
- 标签功能未完全实现
- Webhook 重试机制不完善
- 邮件发送功能未实现

### 10.2 性能限制

- 单次同步最多 1000 封邮件
- 没有批量操作优化
- 缺少缓存策略

### 10.3 安全限制

- OAuth2 token 刷新需要完善
- 短期账号自动禁用需要实现
- 缺少审计日志

---

**文档版本**: 1.0  
**最后更新**: 2025-11-05

