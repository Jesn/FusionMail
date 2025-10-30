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
  - 调整 Vite 代理配置（端口改为 8080）
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

- [ ] 3. 邮箱协议适配器
  - 实现统一的 MailProvider 接口
  - 实现适配器工厂模式
  - _需求：需求 1.1, 1.2, 1.3_

- [x] 3.1 IMAP 适配器实现
  - 实现 IMAP 连接和认证
  - 实现增量邮件拉取（SINCE 命令）
  - 实现邮件详情获取
  - 实现代理支持
  - 实现连接池管理
  - _需求：需求 1.1, 1.6, 2.2_

- [ ] 3.2 Gmail API 适配器实现
  - 实现 OAuth2 认证流程
  - 实现 Gmail API 邮件拉取
  - 实现邮件详情获取
  - 实现代理支持
  - 实现 API 降级到 IMAP 逻辑
  - _需求：需求 1.2, 1.8_

- [ ] 3.3 Microsoft Graph 适配器实现
  - 实现 OAuth2 认证流程
  - 实现 Graph API 邮件拉取
  - 实现邮件详情获取
  - 实现代理支持
  - 实现 API 降级到 IMAP 逻辑
  - _需求：需求 1.3, 1.8_

- [ ] 3.4 POP3 适配器实现
  - 实现 POP3 连接和认证
  - 实现邮件拉取
  - 实现代理支持
  - 添加 POP3 警告提示逻辑
  - _需求：需求 1.1_

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

- [ ] 5. 附件存储实现
  - 实现统一的 StorageProvider 接口
  - 实现存储工厂模式
  - _需求：需求 3.7, 3.8_

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

- [ ] 6.4 Webhook 服务
  - 实现 Webhook CRUD 操作
  - 实现事件触发逻辑
  - 实现 HTTP 请求发送
  - 实现失败重试（3 次，间隔 10s/30s/60s）
  - 实现 Webhook 日志记录
  - _需求：需求 7_

- [ ] 6.5 事件总线
  - 实现 Redis Pub/Sub 事件总线
  - 实现邮件事件发布（received、read、archived、deleted）
  - 实现事件订阅和处理
  - 连接规则引擎和 Webhook 服务
  - _需求：需求 6, 需求 7_

---

### 阶段 7：API 接口层

- [ ] 7. API 基础设施
  - 配置 Gin 框架
  - 实现中间件（CORS、日志、错误处理）
  - 实现统一响应格式
  - _需求：需求 8_

- [x] 7.1 认证与授权
  - 实现 JWT 认证中间件
  - 实现 API Key 认证中间件
  - 实现登录/登出接口
  - 实现 Token 刷新接口
  - _需求：需求 12.2, 12.3_

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

- [ ] 7.7 Webhook 管理 API
  - GET /api/v1/webhooks（获取 Webhook 列表）
  - POST /api/v1/webhooks（创建 Webhook）
  - GET /api/v1/webhooks/:id（获取 Webhook 详情）
  - PUT /api/v1/webhooks/:id（更新 Webhook）
  - DELETE /api/v1/webhooks/:id（删除 Webhook）
  - POST /api/v1/webhooks/:id/test（测试 Webhook）
  - GET /api/v1/webhooks/:id/logs（获取调用日志）
  - _需求：需求 7_

- [ ] 7.8 系统管理 API
  - GET /api/v1/system/health（健康检查）
  - GET /api/v1/system/stats（系统统计）
  - GET /api/v1/sync/status（同步状态）
  - GET /api/v1/sync/logs（同步日志）
  - _需求：需求 11_

- [ ]* 7.9 API 文档
  - 编写 OpenAPI/Swagger 文档
  - 配置 Swagger UI
  - 提供 API 使用示例
  - _需求：需求 9.2_

---

### 阶段 8：前端核心功能

- [ ] 8. 前端基础设施
  - 配置 Zustand 状态管理
  - 实现 API 服务层封装
  - 配置路由
  - _需求：需求 13_

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

- [ ] 8.7 搜索功能
  - 实现 Search 页面
  - 实现搜索输入框和快捷键（/）
  - 实现高级搜索表单
  - 实现搜索结果显示（高亮关键词）
  - 实现智能文件夹保存
  - _需求：需求 5, 13.5_

- [ ] 8.8 规则配置页面
  - 实现 Rules 页面
  - 实现 RuleList 组件
  - 实现 RuleCard 组件
  - 实现 RuleForm 对话框（创建/编辑规则）
  - 实现规则启用/禁用切换
  - _需求：需求 6_

- [ ] 8.9 Webhook 配置页面
  - 实现 Webhooks 页面
  - 实现 WebhookList 组件
  - 实现 WebhookForm 对话框
  - 实现 Webhook 测试功能
  - 实现调用日志查看
  - _需求：需求 7_

- [ ] 8.10 系统设置页面
  - 实现 Settings 页面
  - 实现主题切换（深色/浅色）
  - 实现同步频率配置
  - 实现通知设置
  - _需求：需求 13.2, 13.6_

- [ ] 8.11 同步状态显示
  - 实现同步状态指示器
  - 实现同步进度显示
  - 实现同步历史查看
  - _需求：需求 13.4_

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

## 任务优先级说明

### P0 任务（MVP 核心功能）
必须完成的任务，对应需求文档中的 P0 需求：
- 阶段 1：项目初始化
- 阶段 2：数据模型
- 阶段 3：邮箱协议适配器（至少 IMAP 和 Gmail API）
- 阶段 4：邮件同步引擎
- 阶段 5：本地存储（默认）
- 阶段 6：核心业务逻辑
- 阶段 7：核心 API 接口
- 阶段 8：核心前端页面（收件箱、账户管理）
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
3. **第 5-6 周**：阶段 5-7（存储层、业务逻辑、API）
4. **第 7-8 周**：阶段 8（前端核心功能）
5. **第 9 周**：阶段 9-10（安全优化、监控）
6. **第 10 周**：阶段 11（集成测试、部署）

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

**文档版本**：v1.0  
**创建日期**：2025-10-28  
**最后更新**：2025-10-28
