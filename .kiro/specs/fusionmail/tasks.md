# FusionMail 实现任务清单

## 任务概述

本文档将 FusionMail 的需求和设计转化为具体的实现任务。任务按照依赖关系和优先级组织，每个任务都包含明确的目标和验收标准。

**实现策略**：
- 优先实现 P0 核心功能（MVP）
- 前后端并行开发，定期集成
- 测试任务标记为可选（*），但建议实现核心功能测试
- 每个任务完成后进行代码审查

---

## 任务列表

### 阶段 1：项目初始化与基础设施

- [x] 1. 项目初始化
  - 创建项目目录结构
  - 初始化 Git 仓库
  - 配置 .gitignore
  - _需求：需求 14_

- [x] 1.1 后端项目初始化
  - 初始化 Go 模块（go mod init）
  - 创建基础目录结构（cmd、internal、pkg、config）
  - 配置 Go 编译环境
  - 添加基础依赖（Gin、GORM、Redis 客户端）
  - _需求：需求 14.1, 14.3_

- [x] 1.2 前端项目初始化
  - 基于 frontend_template 创建前端项目
  - 安装额外依赖（Zustand、@tanstack/react-virtual）
  - 配置环境变量（.env）
  - 调整 Vite 代理配置（端口改为 3333）
  - _需求：需求 13.1, 14.3_

- [x] 1.3 Docker 环境配置
  - 编写后端 Dockerfile
  - 编写前端 Dockerfile
  - 编写 docker-compose.yml（PostgreSQL、Redis、后端、前端）
  - 配置数据持久化 Volume
  - _需求：需求 14.1, 14.2, 14.5_

- [x] 1.4 数据库初始化
  - 创建数据库迁移脚本
  - 实现自动建表逻辑（GORM AutoMigrate）
  - 添加初始数据（如果需要）
  - _需求：需求 14.4_

---

### 阶段 2：核心数据模型与存储层

- [ ] 2. 数据模型实现
  - 实现所有核心数据表的 Go 结构体
  - 配置 GORM 模型关系
  - _需求：需求 3_

- [x] 2.1 用户与认证模型
  - 实现 User 模型（users 表）
  - 实现 APIKey 模型（api_keys 表）
  - 实现密码哈希和验证逻辑
  - _需求：需求 12.1, 12.2_

- [x] 2.2 邮箱账户模型
  - 实现 Account 模型（accounts 表）
  - 实现凭证加密存储逻辑（AES-256）
  - 实现代理配置结构
  - _需求：需求 1.5, 1.6_

- [x] 2.3 邮件核心模型
  - 实现 Email 模型（emails 表）
  - 实现 EmailAttachment 模型（email_attachments 表）
  - 实现 Provider ID + UID 唯一索引
  - 实现本地状态字段（is_read、is_starred 等）
  - _需求：需求 3.1, 3.3, 3.4, 3.5_

- [x] 2.4 标签与规则模型
  - 实现 EmailLabel 模型（email_labels 表）
  - 实现 EmailLabelRelation 模型（email_label_relations 表）
  - 实现 EmailRule 模型（email_rules 表）
  - _需求：需求 6_

- [x] 2.5 Webhook 与日志模型
  - 实现 Webhook 模型（webhooks 表）
  - 实现 WebhookLog 模型（webhook_logs 表）
  - 实现 SyncLog 模型（sync_logs 表）
  - _需求：需求 7, 需求 11_

- [x] 2.6 Repository 层实现
  - 实现 UserRepository（CRUD 操作）
  - 实现 AccountRepository（CRUD + 查询）
  - 实现 EmailRepository（CRUD + 搜索 + 分页）
  - 实现 RuleRepository、WebhookRepository
  - _需求：需求 3, 需求 6, 需求 7_

---

### 阶段 3：邮箱协议适配层

- [x] 3. 邮箱协议适配器
  - 实现统一的 MailProvider 接口
  - 实现适配器工厂模式
  - _需求：需求 1.1, 1.2, 1.3_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/adapter/adapter.go`, `backend/internal/adapter/factory.go`

- [x] 3.6 Gmail OAuth2 认证集成（新增 - P0 高优先级）
  - 实现 Google OAuth2 配置管理
  - 实现 OAuth2 授权流程处理器
  - 实现 Token 自动刷新服务
  - 实现授权状态管理
  - _需求：需求 1A_

- [x] 3.7 Microsoft Graph OAuth2 认证集成（新增 - P0 高优先级）
  - 实现 Microsoft OAuth2 配置管理
  - 实现 Microsoft OAuth2 授权流程处理器
  - 实现 Microsoft Graph 邮件适配器
  - 实现基本账户类型识别（个人/工作账户）
  - 实现 Microsoft Token 自动刷新服务
  - _需求：需求 1B_
  - **完成时间**: 2025-01-31
  - **实现文件**: `backend/internal/service/oauth2_service.go`, `backend/internal/handler/oauth2_handler.go`, `backend/internal/adapter/graph.go`, `frontend/src/components/auth/OAuth2AuthButton.tsx`, `frontend/src/services/oauth2Service.ts`

- [x] 3.1 IMAP 适配器实现
  - 实现 IMAP 连接和认证
  - 实现增量邮件拉取（SINCE 命令）
  - 实现邮件详情获取
  - 实现代理支持
  - 实现连接池管理
  - _需求：需求 1.1, 1.6, 2.2_

- [x] 3.2 Gmail API 适配器实现
  - 实现 OAuth2 认证流程
  - 实现 Gmail API 邮件拉取
  - 实现邮件详情获取
  - 实现代理支持
  - 【暂缓】实现 API 降级到 IMAP 逻辑（需要在服务层实现）
  - _需求：需求 1.2, 1.8_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/adapter/gmail.go`

- [x] 3.3 Microsoft Graph 适配器实现
  - 实现 OAuth2 认证流程
  - 实现 Graph API 邮件拉取
  - 实现邮件详情获取
  - 实现代理支持
  - 【暂缓】实现 API 降级到 IMAP 逻辑（需要在服务层实现）
  - _需求：需求 1.3, 1.8_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/adapter/graph.go`

- [x] 3.4 POP3 适配器实现
  - 实现 POP3 连接和认证
  - 实现邮件拉取
  - 【限制】代理支持（go-pop3 库不支持自定义 Dialer）
  - 添加 POP3 警告提示逻辑（在 UI 层实现）
  - _需求：需求 1.1_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/adapter/pop3.go`

- [ ]* 3.5 适配器单元测试
  - 编写 IMAP 适配器测试
  - 编写 Gmail API 适配器测试
  - 编写 Graph 适配器测试
  - 使用 Mock 服务器测试
  - _需求：需求 1.7_

---

### 阶段 4：邮件同步引擎

- [x] 4. 邮件同步引擎
  - 实现同步任务调度器
  - 实现 Worker Pool 并发控制
  - _需求：需求 2_

- [x] 4.1 同步任务队列
  - 实现 Redis 任务队列
  - 实现任务入队和出队逻辑
  - 实现任务优先级管理
  - 实现分布式锁（防止重复同步）
  - _需求：需求 2.3_

- [x] 4.2 定时同步调度
  - 实现 Cron 定时任务
  - 实现可配置的同步频率（1-60 分钟）
  - 实现同步任务创建逻辑
  - _需求：需求 2.1_

- [x] 4.3 同步执行逻辑
  - 实现同步 Worker 逻辑
  - 实现增量同步机制
  - 实现邮件去重判断（Provider ID + UID）
  - 实现邮件存储逻辑
  - 实现同步日志记录
  - _需求：需求 2.2, 2.5, 3.4_

- [x] 4.4 错误处理与重试
  - 实现指数退避重试算法
  - 实现最大重试次数限制（3 次）
  - 实现错误日志记录
  - 实现同步状态更新
  - _需求：需求 2.4, 2.5_

- [x] 4.5 手动同步触发
  - 实现手动同步 API 端点
  - 实现立即同步逻辑
  - 实现同步状态查询
  - _需求：需求 2.6_

- [ ]* 4.6 同步引擎测试
  - 编写同步流程集成测试
  - 测试并发同步场景
  - 测试错误重试逻辑
  - _需求：需求 2_

---

### 阶段 5：附件存储层

- [x] 5. 附件存储实现
  - 实现统一的 StorageProvider 接口
  - 实现存储工厂模式
  - _需求：需求 3.7, 3.8_
  - **完成时间**: 已完成（早期实现）
  - **实现文件**: `backend/pkg/storage/storage.go`, `backend/pkg/storage/factory.go`

- [x] 5.1 本地存储实现（默认）
  - 实现本地文件存储逻辑
  - 实现文件上传和下载
  - 实现文件删除
  - 实现 URL 生成
  - 配置默认存储路径
  - _需求：需求 3.7_

- [ ] 5.2 S3 存储实现（可选）
  - 实现 AWS S3 客户端
  - 实现文件上传到 S3
  - 实现预签名 URL 生成
  - 实现文件删除
  - _需求：需求 3.8, 3.10_

- [ ] 5.3 OSS 存储实现（可选）
  - 实现阿里云 OSS 客户端
  - 实现文件上传到 OSS
  - 实现预签名 URL 生成
  - 实现文件删除
  - _需求：需求 3.8, 3.10_

- [x] 5.4 附件处理逻辑
  - 实现附件下载和保存
  - 实现附件元数据存储
  - 实现附件关联到邮件
  - _需求：需求 3.1, 4.8_

---

### 阶段 6：业务逻辑层

- [x] 6. 账户管理服务
  - 实现账户 CRUD 操作
  - 实现凭证加密和解密
  - 实现连接测试逻辑
  - _需求：需求 1_

- [x] 6.1 邮件管理服务
  - 实现邮件查询（列表、详情）
  - 实现邮件筛选和排序
  - 实现本地状态更新（标记、星标、归档、删除）
  - 实现批量操作
  - _需求：需求 4_
  - **完成时间**: 2025-10-29
  - **实现文件**: `backend/internal/service/email_service.go`

- [x] 6.2 邮件搜索服务
  - 实现全文搜索（PostgreSQL tsvector）
  - 实现高级筛选
  - 【暂时先不处理】实现智能文件夹（保存搜索条件）
  - 【暂时先不处理】实现搜索结果高亮
  - _需求：需求 5_
  - **完成时间**: 2025-10-29
  - **实现文件**: `backend/internal/service/email_service.go`（搜索功能已集成）

- [x] 6.3 规则引擎服务
  - 实现规则 CRUD 操作
  - 实现规则匹配逻辑
  - 实现规则执行（标记已读、归档、删除、星标）
  - 【暂时先不处理】添加标签动作（待标签功能实现）
  - 【暂时先不处理】触发 Webhook 动作（待 Webhook 功能实现）
  - 实现规则优先级排序
  - 实现规则日志记录（执行统计）
  - _需求：需求 6_
  - **完成时间**: 2025-10-29
  - **实现文件**: `backend/internal/service/rule_service.go`

- [x] 6.4 Webhook 服务
  - 实现 Webhook CRUD 操作
  - 实现事件触发逻辑
  - 实现 HTTP 请求发送
  - 实现失败重试（3 次，间隔 10s/30s/60s）
  - 实现 Webhook 日志记录
  - _需求：需求 7_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/service/webhook_service.go`, `backend/internal/repository/webhook.go`

- [x] 6.5 事件总线
  - 实现 Redis Pub/Sub 事件总线
  - 实现邮件事件发布（received、read、archived、deleted）
  - 实现事件订阅和处理
  - 连接规则引擎和 Webhook 服务
  - _需求：需求 6, 需求 7_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/pkg/event/bus.go`, `backend/internal/service/event_service.go`

---

### 阶段 7：API 接口层

- [x] 7. API 基础设施
  - 配置 Gin 框架
  - 实现中间件（CORS、日志、错误处理）
  - 实现统一响应格式
  - _需求：需求 8_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/router/router.go`, `backend/internal/middleware/`, `backend/internal/dto/response.go`

- [x] 7.1 认证与授权
  - 实现 JWT 认证中间件
  - 实现 API Key 认证中间件
  - 实现登录/登出接口
  - 实现 Token 刷新接口
  - _需求：需求 12.2, 12.3_

- [x] 7.1A Gmail OAuth2 API 端点（新增 - P0 高优先级）
  - GET /auth/google/authorize（生成授权 URL）
  - POST /auth/google/callback（处理授权回调）
  - POST /auth/google/refresh（刷新 Token）
  - POST /auth/google/revoke（撤销授权）
  - _需求：需求 1A_

- [ ] 7.1B Microsoft OAuth2 API 端点（新增 - P0 高优先级）
  - GET /auth/microsoft/authorize（生成授权 URL）
  - POST /auth/microsoft/callback（处理授权回调）
  - POST /auth/microsoft/refresh（刷新 Token）
  - POST /auth/microsoft/revoke（撤销授权）
  - _需求：需求 1B_

- [x] 7.2 速率限制
  - 实现 Redis 速率限制中间件
  - 配置不同接口的限流策略
  - 实现 429 错误响应
  - _需求：需求 8.7_

- [x] 7.3 账户管理 API
  - POST /api/v1/accounts（添加账户）
  - GET /api/v1/accounts（获取账户列表）
  - GET /api/v1/accounts/:uid（获取账户详情）
  - PUT /api/v1/accounts/:uid（更新账户）
  - DELETE /api/v1/accounts/:uid（删除账户）
  - POST /api/v1/accounts/:uid/test（测试连接）
  - POST /api/v1/accounts/:uid/sync（手动同步）
  - _需求：需求 1, 需求 8.5_

- [x] 7.4 邮件管理 API
  - GET /api/v1/emails（获取邮件列表，支持分页）
  - GET /api/v1/emails/:id（获取邮件详情）
  - POST /api/v1/emails/mark-read（批量标记已读）
  - POST /api/v1/emails/mark-unread（批量标记未读）
  - POST /api/v1/emails/:id/toggle-star（切换星标）
  - POST /api/v1/emails/:id/archive（归档邮件）
  - DELETE /api/v1/emails/:id（删除邮件）
  - GET /api/v1/emails/unread-count（获取未读数）
  - GET /api/v1/emails/stats/:account_uid（获取账户统计）
  - _需求：需求 4, 需求 8.3, 需求 8.4_
  - **完成时间**: 2025-10-29
  - **实现文件**: `backend/internal/handler/email_handler.go`

- [x] 7.5 搜索 API
  - GET /api/v1/emails/search（搜索邮件）
  - 【暂时先不处理】GET /api/v1/labels（获取标签列表）
  - 【暂时先不处理】POST /api/v1/labels（创建标签）
  - 【暂时先不处理】POST /api/v1/emails/:id/labels（添加标签）
  - _需求：需求 5, 需求 8.3_
  - **完成时间**: 2025-10-29
  - **实现文件**: `backend/internal/handler/email_handler.go`（搜索端点已实现）

- [x] 7.6 规则管理 API
  - GET /api/v1/rules（获取规则列表）
  - POST /api/v1/rules（创建规则）
  - GET /api/v1/rules/:id（获取规则详情）
  - PUT /api/v1/rules/:id（更新规则）
  - DELETE /api/v1/rules/:id（删除规则）
  - POST /api/v1/rules/:id/toggle（启用/禁用规则）
  - POST /api/v1/rules/apply/:account_uid（对账户应用规则）
  - _需求：需求 6_
  - **完成时间**: 2025-10-29
  - **实现文件**: `backend/internal/handler/rule_handler.go`

- [x] 7.7 Webhook 管理 API
  - GET /api/v1/webhooks（获取 Webhook 列表）
  - POST /api/v1/webhooks（创建 Webhook）
  - GET /api/v1/webhooks/:id（获取 Webhook 详情）
  - PUT /api/v1/webhooks/:id（更新 Webhook）
  - DELETE /api/v1/webhooks/:id（删除 Webhook）
  - POST /api/v1/webhooks/:id/test（测试 Webhook）
  - GET /api/v1/webhooks/:id/logs（获取调用日志）
  - _需求：需求 7_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/handler/webhook_handler.go`

- [x] 7.8 系统管理 API
  - GET /api/v1/system/health（健康检查）
  - GET /api/v1/system/stats（系统统计）
  - GET /api/v1/sync/status（同步状态）
  - GET /api/v1/sync/logs（同步日志）
  - _需求：需求 11_
  - **完成时间**: 2025-10-30
  - **实现文件**: `backend/internal/handler/system_handler.go`, `backend/internal/service/system_service.go`

- [ ]* 7.9 API 文档
  - 编写 OpenAPI/Swagger 文档
  - 配置 Swagger UI
  - 提供 API 使用示例
  - _需求：需求 9.2_

---

### 阶段 8：前端核心功能

- [x] 8. 前端基础设施
  - 配置 Zustand 状态管理
  - 实现 API 服务层封装
  - 配置路由
  - _需求：需求 13_
  - **完成时间**: 已完成（子任务全部完成）
  - **实现文件**: `frontend/src/stores/`, `frontend/src/services/`, `frontend/src/components/layout/`

- [x] 8.1 状态管理实现
  - 实现 authStore（认证状态）✅
  - 实现 emailStore（邮件状态）✅
  - 实现 accountStore（账户状态）✅
  - 实现 uiStore（UI 状态）✅
  - _需求：需求 13_
  - **完成时间**: 2025-10-29
  - **实现文件**: `frontend/src/stores/`

- [x] 8.2 API 服务层
  - 实现 emailService（邮件 API 调用）✅
  - 实现 accountService（账户 API 调用）✅
  - 实现 ruleService（规则 API 调用）✅
  - 【暂时先不处理】实现 webhookService（Webhook API 调用）
  - 【暂时先不处理】实现 syncService（同步 API 调用）
  - _需求：需求 8_
  - **完成时间**: 2025-10-29
  - **实现文件**: `frontend/src/services/`

- [x] 8.3 布局组件
  - 实现 MainLayout（主布局）✅
  - 实现 Header（头部导航）✅
  - 实现 Sidebar（侧边栏）✅
  - 实现响应式布局✅
  - _需求：需求 13.1_
  - **完成时间**: 2025-10-29
  - **实现文件**: `frontend/src/components/layout/`

- [x] 8.4 统一收件箱页面
  - 实现 Inbox 页面组件✅
  - 实现 EmailList 组件（虚拟滚动）✅
  - 实现 EmailItem 组件✅
  - 实现 EmailToolbar（筛选、排序、批量操作）✅
  - 实现邮件列表加载和分页✅
  - _需求：需求 4.1, 4.2, 4.3, 4.7, 13.3_
  - **完成时间**: 2025-10-29
  - **实现文件**: `frontend/src/pages/InboxPage.tsx`, `frontend/src/components/email/`

- [x] 8.5 邮件详情页面
  - 实现 EmailDetail 页面✅
  - 实现邮件正文显示（HTML/纯文本）✅
  - 实现附件列表和预览✅
  - 实现邮件操作按钮（标记、星标、归档、删除）✅
  - 显示"仅本地"提示✅
  - _需求：需求 4.4, 4.5, 4.6, 4.8, 4.9_
  - **完成时间**: 2025-10-29
  - **实现文件**: `frontend/src/pages/EmailDetailPage.tsx`, `frontend/src/components/email/EmailDetail.tsx`

- [x] 8.6 邮箱账户管理页面
  - 实现 Accounts 页面✅
  - 实现 AccountCard 组件✅
  - 实现 AccountForm 对话框（添加账户）✅
  - 实现连接测试功能✅
  - 实现手动同步按钮✅
  - _需求：需求 1_
  - **完成时间**: 2025-10-29
  - **实现文件**: `frontend/src/pages/AccountsPage.tsx`, `frontend/src/components/account/`

- [x] 8.6A Gmail OAuth2 前端集成（新增 - P0 高优先级）
  - 实现 Gmail OAuth2 授权组件
  - 实现授权状态显示和管理
  - 集成到账户创建表单
  - 实现重新授权功能
  - 实现授权错误处理和重试
  - _需求：需求 1A_

- [ ] 8.6B Microsoft OAuth2 前端集成（新增 - P0 高优先级）
  - 实现 Microsoft OAuth2 授权组件
  - 实现基本账户类型显示（个人/工作账户）
  - 实现授权状态显示和管理
  - 集成到账户创建表单
  - 实现重新授权功能
  - 实现基本错误处理和重试
  - _需求：需求 1B_

- [x] 8.7 搜索功能
  - 实现 Search 页面
  - 实现搜索输入框和快捷键（/）
  - 实现高级搜索表单
  - 实现搜索结果显示（高亮关键词）
  - 实现智能文件夹保存
  - _需求：需求 5, 13.5_
  - **完成时间**: 2025-10-30
  - **实现文件**: `frontend/src/pages/SearchPage.tsx`, `frontend/src/hooks/useSearch.ts`

- [x] 8.8 规则配置页面
  - 实现 Rules 页面
  - 实现 RuleList 组件
  - 实现 RuleCard 组件
  - 实现 RuleForm 对话框（创建/编辑规则）
  - 实现规则启用/禁用切换
  - _需求：需求 6_
  - **完成时间**: 2025-10-30
  - **实现文件**: `frontend/src/pages/RulesPage.tsx`, `frontend/src/components/rule/`, `frontend/src/hooks/useRules.ts`

- [x] 8.9 Webhook 配置页面
  - 实现 Webhooks 页面
  - 实现 WebhookList 组件
  - 实现 WebhookForm 对话框
  - 实现 Webhook 测试功能
  - 实现调用日志查看
  - _需求：需求 7_
  - **完成时间**: 2025-10-30
  - **实现文件**: `frontend/src/pages/WebhooksPage.tsx`, `frontend/src/components/webhook/`, `frontend/src/hooks/useWebhooks.ts`, `frontend/src/services/webhookService.ts`

- [x] 8.10 系统设置页面
  - 实现 Settings 页面
  - 实现主题切换（深色/浅色）
  - 实现同步频率配置
  - 实现通知设置
  - _需求：需求 13.2, 13.6_
  - **完成时间**: 2025-10-30
  - **实现文件**: `frontend/src/pages/SettingsPage.tsx`, `frontend/src/hooks/useTheme.ts`

- [x] 8.11 同步状态显示
  - 实现同步状态指示器
  - 实现同步进度显示
  - 实现同步历史查看
  - _需求：需求 13.4_
  - **完成时间**: 2025-10-30
  - **实现文件**: `frontend/src/hooks/useSync.ts`, `frontend/src/components/sync/`, `frontend/src/components/ui/progress.tsx`, `frontend/src/components/ui/tooltip.tsx`

---

### 阶段 9：安全与性能优化

- [ ] 9. 安全加固
  - 实现 AES-256 加密工具
  - 实现密码哈希（bcrypt）
  - 配置 HTTPS
  - _需求：需求 12_

- [ ] 9.1 数据加密
  - 实现凭证加密存储
  - 实现 Token 加密存储
  - 实现代理密码加密
  - _需求：需求 12.1_

- [ ] 9.2 输入验证
  - 实现邮箱格式验证
  - 实现 URL 格式验证
  - 实现 SQL 注入防护
  - 实现 XSS 防护
  - _需求：需求 12_

- [ ] 9.3 性能优化
  - 实现 Redis 缓存（邮件列表、未读数）
  - 实现数据库索引优化
  - 实现连接池配置
  - 实现查询优化
  - _需求：需求 3.2_

- [ ] 9.4 前端性能优化
  - 实现虚拟滚动（@tanstack/react-virtual）
  - 实现代码分割（React.lazy）
  - 实现图片懒加载
  - 优化打包体积
  - _需求：需求 13.3_

---

### 阶段 10：监控与日志

- [ ] 10. 日志系统
  - 配置结构化日志（Zap）
  - 实现日志分级（DEBUG、INFO、WARN、ERROR）
  - 实现日志轮转
  - _需求：需求 11.3, 11.4_

- [ ] 10.1 监控指标
  - 实现 Prometheus 指标采集
  - 实现健康检查接口
  - 实现系统统计接口
  - _需求：需求 11.1, 11.2_

- [ ] 10.2 告警通知
  - 实现邮箱连接失败告警
  - 实现同步失败告警
  - 实现告警通知发送（邮件/Webhook）
  - _需求：需求 11.6_

---

### 阶段 11：集成测试与部署

- [ ] 11. 前后端集成
  - 配置 CORS
  - 联调所有 API 接口
  - 测试完整业务流程
  - _需求：需求 8_

- [ ] 11.1 Docker 部署
  - 构建后端 Docker 镜像
  - 构建前端 Docker 镜像
  - 测试 Docker Compose 部署
  - 验证数据持久化
  - _需求：需求 14.1, 14.2, 14.5_

- [ ] 11.2 环境配置
  - 编写配置文件模板
  - 编写环境变量说明
  - 编写部署文档
  - _需求：需求 14.3, 14.6, 14.7_

- [ ]* 11.3 端到端测试
  - 测试账户添加流程
  - 测试邮件同步流程
  - 测试规则执行流程
  - 测试 Webhook 触发流程
  - _需求：所有需求_

- [ ] 11.4 文档编写
  - 编写用户使用文档
  - 编写 API 文档
  - 编写部署文档
  - 编写开发文档
  - _需求：需求 9.2, 14.7_

---

## Gmail OAuth2 集成实施计划（新增）

基于对当前项目的深入分析，Gmail OAuth2 接入是完全可行的，且是 Gmail 集成的关键缺失功能。以下是详细的实施计划：

### 实施阶段

#### 第一阶段：基础配置和后端 OAuth2 流程（1-2 周）

**任务 3.6.1：Google Cloud Console 配置**
- 创建 Google Cloud 项目
- 启用 Gmail API
- 创建 OAuth2 客户端 ID（Web 应用类型）
- 配置授权回调 URL：`https://your-domain.com/auth/google/callback`
- 设置必要的 API 权限和测试用户

**任务 3.6.2：后端 OAuth2 配置管理**
- 添加环境变量配置（GOOGLE_CLIENT_ID、GOOGLE_CLIENT_SECRET、GOOGLE_REDIRECT_URL）
- 实现 GoogleOAuth2Config 结构体
- 集成到现有配置系统

**任务 3.6.3：OAuth2 处理器实现**
- 实现 OAuth2Handler 核心逻辑
- 实现 PKCE (Proof Key for Code Exchange) 安全机制
- 实现 State 参数管理（防止 CSRF 攻击）
- 实现授权 URL 生成和回调处理

**任务 3.6.4：Token 生命周期管理**
- 实现 TokenRefreshService 自动刷新服务
- 实现 Token 过期检查和自动续期
- 实现刷新失败时的重新授权机制

#### 第二阶段：API 端点和前端集成（1-1.5 周）

**任务 7.1A.1：后端 OAuth2 API 端点**
- GET /auth/google/authorize - 生成授权 URL
- POST /auth/google/callback - 处理授权回调
- POST /auth/google/refresh - 手动刷新 Token
- POST /auth/google/revoke - 撤销授权

**任务 8.6A.1：前端 OAuth2 组件开发**
- 实现 GmailOAuth2 授权组件
- 实现授权状态显示组件
- 实现错误处理和重试机制

**任务 8.6A.2：账户创建流程集成**
- 在 AccountForm 中添加 OAuth2 选项
- 集成授权流程到账户创建向导
- 实现授权成功后的自动账户创建

#### 第三阶段：测试和优化（0.5-1 周）

**任务 3.6.5：集成测试**
- 端到端 OAuth2 流程测试
- Token 刷新机制测试
- 错误场景测试（网络异常、授权失败等）

**任务 8.6A.3：用户体验优化**
- 实现清晰的授权步骤指引
- 实现实时状态反馈
- 实现友好的错误提示

### 技术实现要点

#### 安全性考虑
- 使用 PKCE 增强 OAuth2 安全性
- State 参数防止 CSRF 攻击
- Token 加密存储
- 定期 Token 轮换

#### 用户体验设计
- 清晰的授权步骤指引
- 实时状态反馈
- 友好的错误提示
- 一键重新授权功能

#### 错误处理策略
- Token 过期自动刷新
- 刷新失败降级到重新授权
- 网络异常重试机制
- API 限流处理

### 成功标准

1. ✅ 用户可以通过 Google 账号授权添加 Gmail 账户
2. ✅ OAuth2 Token 自动刷新机制正常工作
3. ✅ 邮件同步功能与 OAuth2 认证完全集成
4. ✅ 授权失败时有清晰的错误提示和重试机制
5. ✅ 支持多个 Gmail 账户同时管理

### 实施优势

- **技术可行性高**：基于现有架构，实现难度不大
- **用户需求迫切**：OAuth2 是 Gmail 集成的必要条件
- **投入产出比高**：相对较小的开发投入，显著提升产品价值
- **风险可控**：不会影响现有功能，可以渐进式推进

---

## Microsoft Graph OAuth2 集成实施计划（新增）

Microsoft Graph OAuth2 接入与 Gmail OAuth2 同等重要，支持 Outlook/Hotmail 用户的安全认证需求。

### 实施阶段

#### 第一阶段：Azure AD 配置和后端 OAuth2 流程（1-2 周）

**任务 3.7.1：Azure AD 应用配置**
- 在 Azure Portal 创建应用注册
- 配置支持的账户类型：主要支持个人 Microsoft 账户
- 设置授权回调 URL：`https://your-domain.com/auth/microsoft/callback`
- 配置 API 权限：Mail.ReadWrite、Mail.Send、User.Read、offline_access
- 设置基本测试配置

**任务 3.7.2：后端 Microsoft OAuth2 配置管理**
- 添加环境变量配置（MICROSOFT_CLIENT_ID、MICROSOFT_CLIENT_SECRET、MICROSOFT_REDIRECT_URL）
- 实现 MicrosoftOAuth2Config 结构体
- 配置基本端点（common endpoint）
- 集成到现有配置系统

**任务 3.7.3：Microsoft OAuth2 处理器实现**
- 实现 MicrosoftOAuth2Handler 核心逻辑
- 实现 PKCE (Proof Key for Code Exchange) 安全机制
- 实现基本账户类型识别
- 实现授权 URL 生成和回调处理
- 实现基本错误处理和用户友好提示

**任务 3.7.4：Microsoft Graph 邮件适配器**
- 实现 GraphAdapter 完整功能
- 支持 Microsoft Graph API v1.0 邮件访问
- 实现邮件列表获取和详情解析
- 处理 Microsoft Graph 的分页和限流
- 支持多部分邮件和附件处理

**任务 3.7.5：Microsoft Token 生命周期管理**
- 扩展 TokenRefreshService 支持 Microsoft Token
- 实现 Microsoft Token 过期检查和自动续期
- 实现刷新失败时的重新授权机制
- 实现基本的 Token 管理逻辑

#### 第二阶段：API 端点和前端集成（1-1.5 周）

**任务 7.1B.1：后端 Microsoft OAuth2 API 端点**
- GET /auth/microsoft/authorize - 生成授权 URL
- POST /auth/microsoft/callback - 处理授权回调
- POST /auth/microsoft/refresh - 手动刷新 Token
- POST /auth/microsoft/revoke - 撤销授权
- 支持账户类型检测和显示

**任务 8.6B.1：前端 Microsoft OAuth2 组件开发**
- 实现 MicrosoftOAuth2 授权组件
- 实现基本账户类型显示（个人/工作账户）
- 实现授权状态显示组件
- 实现基本错误处理和重试机制

**任务 8.6B.2：账户创建流程集成**
- 在 AccountForm 中添加 Microsoft OAuth2 选项
- 集成授权流程到账户创建向导
- 实现授权成功后的自动账户创建
- 实现基本账户类型识别和显示

#### 第三阶段：测试和优化（0.5-1 周）

**任务 3.7.6：集成测试**
- 端到端 Microsoft OAuth2 流程测试
- 个人账户测试（主要场景）
- Token 刷新机制测试
- Microsoft Graph API 调用测试
- 基本错误场景测试（网络异常、授权失败等）

**任务 8.6B.3：用户体验优化**
- 实现清晰的授权步骤指引
- 实现基本账户类型显示
- 实现实时状态反馈
- 实现友好的错误提示

### Microsoft Graph OAuth2 技术要点

#### 与 Gmail OAuth2 的差异

1. **账户类型支持**：
   - 主要支持个人账户（@hotmail.com、@outlook.com、@live.com）
   - 基本支持工作账户（简化处理）
   - 使用 "common" 端点

2. **权限范围差异**：
   - Microsoft: `Mail.ReadWrite`, `Mail.Send`, `User.Read`, `offline_access`
   - Gmail: `gmail.readonly`, `gmail.send`, `gmail.modify`

3. **API 结构差异**：
   - Microsoft Graph 使用 RESTful API 结构
   - 支持 OData 查询语法
   - 分页机制与 Gmail API 不同

4. **错误处理**：
   - Microsoft Graph 特有的错误码和错误结构
   - 基本的错误提示和用户引导

#### 安全性考虑

- 使用 PKCE 增强 OAuth2 安全性
- State 参数防止 CSRF 攻击
- Token 加密存储
- 基本的安全验证机制

#### 用户体验设计

- 清晰显示账户类型（个人/工作账户）
- 提供基本的账户类型识别
- 友好的授权流程
- 简洁的错误提示和处理

### Microsoft Graph OAuth2 成功标准

1. ✅ 用户可以通过 Microsoft 账号授权添加 Outlook/Hotmail 账户
2. ✅ 支持个人账户（主要）和基本的工作账户识别
3. ✅ Microsoft OAuth2 Token 自动刷新机制正常工作
4. ✅ 邮件同步功能与 Microsoft Graph OAuth2 认证完全集成
5. ✅ 授权失败时有清晰的错误提示和重试机制
6. ✅ 支持多个 Microsoft 账户同时管理
7. ✅ 基本的账户类型识别和显示

### Microsoft Graph OAuth2 实施优势

- **市场需求大**：Outlook/Hotmail 用户群体庞大
- **轻量化实现**：专注个人用户和小团队需求
- **技术架构一致**：复用 Gmail OAuth2 的基础设施
- **用户体验统一**：与 Gmail OAuth2 保持一致的操作流程

---

## 任务优先级说明

### P0 任务（MVP 核心功能）
必须完成的任务，对应需求文档中的 P0 需求：
- 阶段 1：项目初始化
- 阶段 2：数据模型
- 阶段 3：邮箱协议适配器（至少 IMAP 和 Gmail API）
- **阶段 3.6：Gmail OAuth2 认证集成（新增 - 高优先级）**
- **阶段 3.7：Microsoft Graph OAuth2 认证集成（新增 - 高优先级）**
- 阶段 4：邮件同步引擎
- 阶段 5：本地存储（默认）
- 阶段 6：核心业务逻辑
- 阶段 7：核心 API 接口
- **阶段 7.1A：Gmail OAuth2 API 端点（新增 - 高优先级）**
- **阶段 7.1B：Microsoft OAuth2 API 端点（新增 - 高优先级）**
- 阶段 8：核心前端页面（收件箱、账户管理）
- **阶段 8.6A：Gmail OAuth2 前端集成（新增 - 高优先级）**
- **阶段 8.6B：Microsoft OAuth2 前端集成（新增 - 高优先级）**
- 阶段 9：基础安全
- 阶段 11：部署

### P1 任务（重要功能）
对应需求文档中的 P1 需求：
- 邮件搜索
- 规则引擎
- Webhook 集成
- 完整的 API 接口
- 监控与日志

### P2 任务（增强功能）
对应需求文档中的 P2 需求：
- 对象存储（S3/OSS）
- 代理健康检查
- UI 增强功能

### 可选任务（标记为 *）
测试相关任务，建议实现但不强制：
- 单元测试
- 集成测试
- 端到端测试

---

## 实施建议

### 开发顺序
1. **第 1-2 周**：阶段 1-2（项目初始化、数据模型）
2. **第 3-4 周**：阶段 3-4（协议适配器、同步引擎）
3. **第 4.5-5.5 周**：**阶段 3.6 + 7.1A（Gmail OAuth2 后端集成）** - 新增高优先级
4. **第 5.5-6 周**：**阶段 3.7 + 7.1B（Microsoft Graph OAuth2 后端集成）** - 新增高优先级（简化版）
5. **第 6-7 周**：阶段 5-7（存储层、业务逻辑、API）
6. **第 7-8 周**：阶段 8（前端核心功能）
7. **第 8-8.5 周**：**阶段 8.6A（Gmail OAuth2 前端集成）** - 新增高优先级
8. **第 8.5-9 周**：**阶段 8.6B（Microsoft OAuth2 前端集成）** - 新增高优先级（简化版）
9. **第 9.5 周**：阶段 9-10（安全优化、监控）
10. **第 10 周**：阶段 11（集成测试、部署）

**注意**：
- Gmail 和 Microsoft Graph OAuth2 集成都被标记为 P0 高优先级任务
- Microsoft Graph OAuth2 采用简化实现，专注个人用户和小团队需求
- 建议采用序列开发策略：先完成 Gmail OAuth2，再开发 Microsoft Graph OAuth2
- 总开发时间保持 10 周，通过简化企业级功能节省 1 周时间
- 两个 OAuth2 实现可以复用基础架构，降低开发成本

### 并行开发
- 前端和后端可以并行开发
- 定义好 API 接口后，前端可以使用 Mock 数据开发
- 每周进行一次前后端集成测试

### 代码审查
- 每个阶段完成后进行代码审查
- 重点审查安全性、性能、代码质量

### 测试策略
- 核心功能必须有单元测试
- 关键流程必须有集成测试
- 部署前必须进行端到端测试

---

**文档版本**：v1.1  
**创建日期**：2025-10-28  
**最后更新**：2025-01-31  
**变更记录**：
- v1.1 (2025-01-31): 添加 Gmail OAuth2 认证集成的完整实施计划，包括需求、设计和任务分解
- v1.0 (2025-10-28): 初始版本

---

## 文档版本信息

**文档版本**：v1.3  
**创建日期**：2025-10-28  
**最后更新**：2025-01-31  
**变更记录**：
- v1.3 (2025-01-31): 简化 Microsoft Graph OAuth2 实施计划，移除复杂企业级功能，专注轻量化实现
- v1.2 (2025-01-31): 添加 Microsoft Graph OAuth2 认证集成的完整实施计划，包括需求、设计和任务分解
- v1.1 (2025-01-31): 添加 Gmail OAuth2 认证集成的完整实施计划，包括需求、设计和任务分解
- v1.0 (2025-10-28): 初始版本