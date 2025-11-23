# 邮箱提供商配置化与 OAuth2 多客户端管理 - 任务拆解文档

## 📋 文档信息

- **项目名称**: FusionMail 邮箱提供商配置化
- **文档版本**: v1.0
- **创建日期**: 2025-11-21
- **作者**: FusionMail Team
- **状态**: 待分配

---

## 1. 项目概览

### 1.1 开发周期

- **总工期**: 2-3 周
- **第一阶段**: 邮箱提供商配置化（5-7 天）
- **第二阶段**: OAuth2 多客户端配置（7-10 天）

### 1.2 团队分工

| 角色 | 职责 | 人员 |
|------|------|------|
| 后端开发 | 数据库、API、Service 层实现 | 待分配 |
| 前端开发 | UI 组件、状态管理、API 调用 | 待分配 |
| 测试 | 单元测试、集成测试、性能测试 | 待分配 |
| 产品 | 需求评审、验收 | 待分配 |

---

## 2. 阶段一：邮箱提供商配置化

### 2.1 任务列表

#### P0 任务（核心功能，必须完成）

##### T1-01: 数据库设计与迁移
**负责人**: 后端开发
**预估工时**: 0.5 天
**依赖**: 无

**任务详情**:
- [ ] 创建 `providers` 表的迁移文件
- [ ] 编写建表 SQL（包含所有字段、索引、约束）
- [ ] 编写初始化数据 SQL（6个默认提供商）
- [ ] 测试迁移脚本的执行和回滚
- [ ] 编写迁移文档

**交付物**:
- `backend/migrations/005_create_providers_table.sql`
- 迁移执行和回滚测试报告

**验收标准**:
- 迁移脚本可以成功执行
- 数据库包含所有预期字段和索引
- 初始化数据正确插入（6条记录）
- 回滚脚本可以清理所有变更

---

##### T1-02: Provider 数据模型
**负责人**: 后端开发
**预估工时**: 0.5 天
**依赖**: T1-01

**任务详情**:
- [ ] 创建 `model/provider.go`
- [ ] 定义 Provider 结构体（所有字段）
- [ ] 实现 TableName() 方法
- [ ] 实现 JSON 序列化/反序列化方法
  - [ ] GetSupportedProtocols()
  - [ ] SetSupportedProtocols()
- [ ] 添加字段验证逻辑
- [ ] 编写单元测试

**交付物**:
- `backend/internal/model/provider.go`
- `backend/internal/model/provider_test.go`

**验收标准**:
- 模型字段与数据库表结构一致
- JSON 序列化/反序列化正常
- 单元测试覆盖率 > 80%

---

##### T1-03: ProviderRepository 实现
**负责人**: 后端开发
**预估工时**: 1 天
**依赖**: T1-02

**任务详情**:
- [ ] 创建 `repository/provider.go`
- [ ] 定义 ProviderRepository 接口
  - [ ] Create, Update, Delete
  - [ ] FindAll, FindByName, FindEnabled
- [ ] 实现接口方法
- [ ] 添加错误处理
- [ ] 编写单元测试

**交付物**:
- `backend/internal/repository/provider.go`
- `backend/internal/repository/provider_test.go`

**验收标准**:
- 所有接口方法实现正确
- 数据库操作正常
- 单元测试覆盖率 > 80%
- 错误处理完善

---

##### T1-04: SystemService 修改
**负责人**: 后端开发
**预估工时**: 1.5 天
**依赖**: T1-03

**任务详情**:
- [ ] 修改 SystemService 结构体
  - [ ] 添加 providerRepo 字段
  - [ ] 添加 cache 字段
- [ ] 重写 GetSupportedProviders() 方法
  - [ ] 实现缓存逻辑（Redis，TTL 1小时）
  - [ ] 从数据库读取提供商配置
  - [ ] 实现降级策略（fallback）
- [ ] 实现 getFallbackProviders() 方法
- [ ] 实现缓存失效逻辑
- [ ] 编写单元测试
- [ ] 编写集成测试

**交付物**:
- 修改后的 `backend/internal/service/system_service.go`
- 单元测试和集成测试文件

**验收标准**:
- 数据库查询成功时返回数据库配置
- 数据库查询失败时返回硬编码配置（降级）
- 缓存逻辑正常工作
- 测试覆盖率 > 80%

---

##### T1-05: 前端 Provider 类型定义和服务
**负责人**: 前端开发
**预估工时**: 0.5 天
**依赖**: T1-04（API 可用）

**任务详情**:
- [ ] 创建 `types/provider.ts`
- [ ] 定义 Provider 接口
- [ ] 修改 `services/systemService.ts`
  - [ ] 更新 Provider 类型
  - [ ] 确保 API 调用兼容
- [ ] 修改 `hooks/useProviders.ts`
  - [ ] 确保类型兼容
  - [ ] 测试数据获取

**交付物**:
- `frontend/src/types/provider.ts`
- 修改后的 `frontend/src/services/systemService.ts`
- 修改后的 `frontend/src/hooks/useProviders.ts`

**验收标准**:
- 类型定义完整且准确
- API 调用正常
- 前端能成功获取提供商列表

---

##### T1-06: 集成测试与验证
**负责人**: 后端开发 + 前端开发
**预估工时**: 1 天
**依赖**: T1-04, T1-05

**任务详情**:
- [ ] 端到端测试
  - [ ] 启动后端服务
  - [ ] 启动前端应用
  - [ ] 测试添加账户流程
- [ ] 验证提供商列表显示正确
- [ ] 验证账户添加功能正常
- [ ] 性能测试
  - [ ] 测试 API 响应时间（< 100ms）
  - [ ] 测试并发请求（100+）
- [ ] 降级测试
  - [ ] 模拟数据库故障
  - [ ] 验证使用硬编码配置
- [ ] 编写测试报告

**交付物**:
- 集成测试脚本
- 性能测试报告
- 端到端测试报告

**验收标准**:
- 所有功能正常工作
- API 响应时间 < 100ms
- 降级策略正常工作
- 现有账户功能不受影响

---

#### P1 任务（重要功能，优先完成）

##### T1-07: 提供商管理 API（后期）
**负责人**: 后端开发
**预估工时**: 1 天
**依赖**: T1-04

**任务详情**:
- [ ] 创建 ProviderHandler
- [ ] 实现 CRUD 接口
  - [ ] POST /api/v1/providers
  - [ ] GET /api/v1/providers
  - [ ] PUT /api/v1/providers/:name
  - [ ] DELETE /api/v1/providers/:name
- [ ] 添加权限控制（管理员）
- [ ] 添加请求验证
- [ ] 清除缓存逻辑
- [ ] 编写 API 文档

**交付物**:
- `backend/internal/handler/provider_handler.go`
- API 文档

**验收标准**:
- 所有 API 正常工作
- 权限控制生效
- 配置变更后缓存失效

---

#### P2 任务（辅助功能，可延后）

##### T1-08: 提供商管理界面（后期）
**负责人**: 前端开发
**预估工时**: 2 天
**依赖**: T1-07

**任务详情**:
- [ ] 创建提供商管理页面
- [ ] 实现列表展示
- [ ] 实现添加表单
- [ ] 实现编辑表单
- [ ] 实现删除确认
- [ ] 实现启用/禁用开关
- [ ] 添加表单验证

**交付物**:
- 提供商管理页面组件

**验收标准**:
- 页面功能完整
- UI/UX 友好
- 表单验证完善

---

### 2.2 第一阶段时间表

```
Week 1: Day 1-2
├─ T1-01: 数据库设计与迁移 (0.5天) ✓
├─ T1-02: Provider 数据模型 (0.5天) ✓
└─ T1-03: ProviderRepository 实现 (1天) →

Week 1: Day 3-4
├─ T1-03: ProviderRepository 完成 ✓
└─ T1-04: SystemService 修改 (1.5天) →

Week 1: Day 5
├─ T1-04: SystemService 完成 ✓
└─ T1-05: 前端实现 (0.5天) ✓

Week 2: Day 1
└─ T1-06: 集成测试与验证 (1天) ✓

阶段一完成 ✓
```

---

## 3. 阶段二：OAuth2 多客户端配置

### 3.1 任务列表

#### P0 任务（核心功能，必须完成）

##### T2-01: 数据库设计与迁移
**负责人**: 后端开发
**预估工时**: 1 天
**依赖**: T1-06

**任务详情**:
- [ ] 创建 `oauth2_clients` 表的迁移文件
- [ ] 编写建表 SQL
  - [ ] 所有字段、索引、约束
  - [ ] 外键关联到 providers 表
  - [ ] 唯一约束（每个 provider 只有一个默认配置）
- [ ] 扩展 `accounts` 表
  - [ ] 添加 oauth2_client_id 字段
  - [ ] 添加外键约束
- [ ] 测试迁移脚本
- [ ] 编写迁移文档

**交付物**:
- `backend/migrations/006_create_oauth2_clients_table.sql`
- 迁移执行测试报告

**验收标准**:
- 迁移脚本成功执行
- 所有约束和索引正确创建
- accounts 表扩展成功

---

##### T2-02: OAuth2Client 数据模型
**负责人**: 后端开发
**预估工时**: 0.5 天
**依赖**: T2-01

**任务详情**:
- [ ] 创建 `model/oauth2_client.go`
- [ ] 定义 OAuth2Client 结构体
- [ ] 实现 TableName() 方法
- [ ] 实现辅助方法
  - [ ] IsQuotaExceeded()
  - [ ] GetAvailableQuota()
  - [ ] GetScopes()
  - [ ] GetTags()
- [ ] 编写单元测试

**交付物**:
- `backend/internal/model/oauth2_client.go`
- `backend/internal/model/oauth2_client_test.go`

**验收标准**:
- 模型字段完整
- 辅助方法逻辑正确
- 单元测试覆盖率 > 80%

---

##### T2-03: OAuth2ClientRepository 实现
**负责人**: 后端开发
**预估工时**: 1.5 天
**依赖**: T2-02

**任务详情**:
- [ ] 创建 `repository/oauth2_client.go`
- [ ] 定义接口
  - [ ] 基础 CRUD
  - [ ] 查询方法（按提供商、默认配置等）
  - [ ] 配额管理方法
  - [ ] 统计更新方法
  - [ ] GetAvailableClient（智能选择）
- [ ] 实现所有接口方法
- [ ] 编写单元测试

**交付物**:
- `backend/internal/repository/oauth2_client.go`
- `backend/internal/repository/oauth2_client_test.go`

**验收标准**:
- 所有接口方法实现正确
- 智能选择算法正确（优先级、配额）
- 单元测试覆盖率 > 80%

---

##### T2-04: OAuth2ClientService 实现
**负责人**: 后端开发
**预估工时**: 2 天
**依赖**: T2-03

**任务详情**:
- [ ] 创建 `service/oauth2_client_service.go`
- [ ] 实现 Service 结构体
- [ ] 实现核心方法
  - [ ] GetAvailableClientsForProvider
  - [ ] GetBestAvailableClient
  - [ ] GetClientByID
  - [ ] UseClient（配额管理）
  - [ ] RecordAuthorizationResult
  - [ ] CreateClient（加密 Client Secret）
- [ ] 实现 Redis 配额计数
- [ ] 实现缓存逻辑
- [ ] 定义 DTO
- [ ] 编写单元测试
- [ ] 编写集成测试

**交付物**:
- `backend/internal/service/oauth2_client_service.go`
- 单元测试和集成测试文件

**验收标准**:
- 所有方法实现正确
- 配额管理逻辑正确
- 缓存正常工作
- Client Secret 正确加密
- 测试覆盖率 > 80%

---

##### T2-05: OAuth2ClientHandler 实现
**负责人**: 后端开发
**预估工时**: 1 天
**依赖**: T2-04

**任务详情**:
- [ ] 创建 `handler/oauth2_client_handler.go`
- [ ] 实现 Handler 结构体
- [ ] 实现 API 接口
  - [ ] GET /api/v1/oauth2/clients/:provider
  - [ ] GET /api/v1/oauth2/clients/detail/:id
  - [ ] POST /api/v1/oauth2/clients（管理）
  - [ ] PUT /api/v1/oauth2/clients/:id（管理）
  - [ ] DELETE /api/v1/oauth2/clients/:id（管理）
- [ ] 添加权限控制
- [ ] 添加请求验证
- [ ] 注册路由
- [ ] 编写 API 文档

**交付物**:
- `backend/internal/handler/oauth2_client_handler.go`
- API 路由注册代码
- API 文档

**验收标准**:
- 所有 API 正常工作
- 权限控制生效
- 请求验证完善

---

##### T2-06: 修改 OAuth2 授权流程
**负责人**: 后端开发
**预估工时**: 1.5 天
**依赖**: T2-05

**任务详情**:
- [ ] 修改 OAuth2Service
  - [ ] Authorize 方法支持 client_id 参数
  - [ ] 如果未指定，使用默认配置
  - [ ] 从 oauth2_client 读取凭证
- [ ] 修改 Callback 方法
  - [ ] 记录使用的 oauth2_client_id
  - [ ] 调用 UseClient 增加配额
  - [ ] 记录授权结果
- [ ] 更新 Account 模型
  - [ ] 保存 oauth2_client_id
- [ ] 编写集成测试

**交付物**:
- 修改后的 OAuth2Service
- 集成测试文件

**验收标准**:
- OAuth2 授权流程正常
- client_id 参数正确处理
- 配额正确增加
- 账户正确关联配置

---

##### T2-07: 前端类型定义和服务
**负责人**: 前端开发
**预估工时**: 0.5 天
**依赖**: T2-05

**任务详情**:
- [ ] 创建 `types/provider.ts` 的 OAuth2Client 接口
- [ ] 创建 `services/oauth2ClientService.ts`
  - [ ] getAvailableClients
  - [ ] getClientById
- [ ] 编写类型测试

**交付物**:
- 类型定义文件
- API 服务文件

**验收标准**:
- 类型定义完整
- API 调用正常

---

##### T2-08: OAuth2ClientSelector 组件
**负责人**: 前端开发
**预估工时**: 1.5 天
**依赖**: T2-07

**任务详情**:
- [ ] 创建 `components/account/OAuth2ClientSelector.tsx`
- [ ] 实现组件功能
  - [ ] 加载可用配置列表
  - [ ] 显示配置选择器（下拉框）
  - [ ] 显示配额信息
  - [ ] 显示配置详情
  - [ ] 自动选择默认配置
- [ ] 添加样式
- [ ] 处理加载和错误状态
- [ ] 编写组件测试

**交付物**:
- OAuth2ClientSelector 组件
- 组件单元测试

**验收标准**:
- 组件功能完整
- UI/UX 友好
- 自动选择逻辑正确
- 配额显示正确

---

##### T2-09: 集成 OAuth2ClientSelector 到 AccountForm
**负责人**: 前端开发
**预估工时**: 1 天
**依赖**: T2-08

**任务详情**:
- [ ] 修改 AccountForm 组件
  - [ ] 添加 selectedOAuth2ClientId 状态
  - [ ] 集成 OAuth2ClientSelector 组件
  - [ ] 仅在 OAuth2 协议时显示
  - [ ] 单个配置时隐藏选择器
- [ ] 修改 OAuth2AuthButton 组件
  - [ ] 添加 clientId 参数
  - [ ] 传递 clientId 到授权 URL
- [ ] 测试整个流程

**交付物**:
- 修改后的 AccountForm
- 修改后的 OAuth2AuthButton
- 集成测试

**验收标准**:
- OAuth2 配置选择器正常显示
- 用户可以选择配置
- 授权流程使用选中的配置
- 单个配置时不显示选择器

---

##### T2-10: 集成测试与验证
**负责人**: 后端开发 + 前端开发
**预估工时**: 2 天
**依赖**: T2-06, T2-09

**任务详情**:
- [ ] 端到端测试
  - [ ] 配置多个 OAuth2 应用
  - [ ] 测试用户添加账户流程
  - [ ] 验证使用正确的配置
  - [ ] 验证配额增加
- [ ] 配额管理测试
  - [ ] 测试配额超限切换
  - [ ] 测试配额重置
  - [ ] 测试智能选择
- [ ] 性能测试
  - [ ] 并发授权测试
  - [ ] API 响应时间
- [ ] 降级测试
- [ ] 编写测试报告

**交付物**:
- 端到端测试脚本
- 配额管理测试报告
- 性能测试报告

**验收标准**:
- 所有功能正常工作
- 配额管理正确
- 智能选择正确
- 性能符合要求

---

#### P1 任务（重要功能，优先完成）

##### T2-11: 配额重置定时任务
**负责人**: 后端开发
**预估工时**: 0.5 天
**依赖**: T2-04

**任务详情**:
- [ ] 实现定时任务逻辑
- [ ] 使用 cron 表达式（每日凌晨）
- [ ] 重置所有配置的配额
- [ ] 添加日志
- [ ] 测试定时任务

**交付物**:
- 定时任务实现代码

**验收标准**:
- 定时任务正常执行
- 配额正确重置

---

#### P2 任务（辅助功能，可延后）

##### T2-12: OAuth2 配置管理界面
**负责人**: 前端开发
**预估工时**: 3 天
**依赖**: T2-05

**任务详情**:
- [ ] 创建配置管理页面
- [ ] 实现列表展示（按提供商分组）
- [ ] 实现添加表单
- [ ] 实现编辑表单
- [ ] 实现删除确认
- [ ] 显示统计信息
- [ ] 显示配额使用情况
- [ ] 添加表单验证

**交付物**:
- OAuth2 配置管理页面

**验收标准**:
- 页面功能完整
- UI/UX 友好
- 统计信息准确

---

### 3.2 第二阶段时间表

```
Week 2: Day 2-3
├─ T2-01: 数据库设计与迁移 (1天) ✓
└─ T2-02: OAuth2Client 数据模型 (0.5天) →

Week 2: Day 4-5
├─ T2-02: OAuth2Client 完成 ✓
└─ T2-03: OAuth2ClientRepository (1.5天) →

Week 3: Day 1-2
├─ T2-03: Repository 完成 ✓
└─ T2-04: OAuth2ClientService (2天) →

Week 3: Day 3
├─ T2-04: Service 完成 ✓
└─ T2-05: Handler 实现 (1天) ✓

Week 3: Day 4-5
├─ T2-06: OAuth2 授权流程修改 (1.5天) →
└─ T2-07: 前端类型定义 (0.5天，并行) →

Week 4: Day 1-2
├─ T2-06: 授权流程完成 ✓
├─ T2-08: OAuth2ClientSelector (1.5天) →
└─ T2-11: 定时任务 (0.5天，并行) ✓

Week 4: Day 3
└─ T2-09: 集成到 AccountForm (1天) ✓

Week 4: Day 4-5
└─ T2-10: 集成测试与验证 (2天) ✓

阶段二完成 ✓
```

---

## 4. 任务依赖关系图

```
阶段一：邮箱提供商配置化
┌──────────────────────────────────────────────────────┐
│                                                      │
│  T1-01 数据库设计                                     │
│    ↓                                                 │
│  T1-02 Provider 模型                                 │
│    ↓                                                 │
│  T1-03 ProviderRepository                           │
│    ↓                                                 │
│  T1-04 SystemService 修改                           │
│    ↓                                                 │
│  T1-05 前端实现                                      │
│    ↓                                                 │
│  T1-06 集成测试                                      │
│    ↓                                                 │
│  [可选] T1-07 管理 API → T1-08 管理界面              │
│                                                      │
└──────────────────────────────────────────────────────┘

阶段二：OAuth2 多客户端配置
┌──────────────────────────────────────────────────────┐
│                                                      │
│  T2-01 数据库设计                                     │
│    ↓                                                 │
│  T2-02 OAuth2Client 模型                            │
│    ↓                                                 │
│  T2-03 OAuth2ClientRepository                       │
│    ↓                                                 │
│  T2-04 OAuth2ClientService                          │
│    ↓                                                 │
│  T2-05 Handler API                                  │
│    ↓             ↓                                   │
│  T2-06 授权流程  T2-07 前端类型                      │
│    ↓             ↓                                   │
│        T2-08 OAuth2ClientSelector                   │
│              ↓                                       │
│        T2-09 集成到 AccountForm                      │
│              ↓                                       │
│        T2-10 集成测试                                │
│                                                      │
│  [并行] T2-11 定时任务                               │
│  [可选] T2-12 管理界面                               │
│                                                      │
└──────────────────────────────────────────────────────┘
```

---

## 5. 风险与应对

### 5.1 技术风险

| 风险 | 影响 | 概率 | 应对措施 | 负责人 |
|------|------|------|----------|--------|
| 数据迁移失败 | 高 | 低 | 充分测试；准备回滚方案 | 后端开发 |
| 配额计数不准确 | 高 | 中 | 使用 Redis 原子操作；定期校验 | 后端开发 |
| 缓存一致性问题 | 中 | 低 | 配置变更时主动失效缓存 | 后端开发 |
| OAuth2 切换逻辑复杂 | 中 | 中 | 详细设计；充分测试边界情况 | 后端开发 |
| 前端性能问题 | 低 | 低 | 优化组件渲染；使用缓存 | 前端开发 |

### 5.2 进度风险

| 风险 | 影响 | 概率 | 应对措施 | 负责人 |
|------|------|------|----------|--------|
| 任务预估不准 | 中 | 中 | 每日 standup；及时调整 | 团队 |
| 人员变动 | 高 | 低 | 文档完善；代码 review | 团队 |
| 需求变更 | 中 | 低 | 评审需求；控制范围 | 产品 |
| 测试发现重大问题 | 高 | 低 | 单元测试；持续集成 | 测试 |

---

## 6. 测试计划

### 6.1 单元测试

| 模块 | 测试内容 | 覆盖率目标 | 负责人 |
|------|----------|-----------|--------|
| Model | 字段验证、辅助方法 | > 80% | 后端开发 |
| Repository | CRUD 操作、查询方法 | > 80% | 后端开发 |
| Service | 业务逻辑、缓存、配额 | > 80% | 后端开发 |
| Handler | 请求验证、权限控制 | > 70% | 后端开发 |
| 前端组件 | UI 交互、状态管理 | > 70% | 前端开发 |

### 6.2 集成测试

| 测试场景 | 预期结果 | 负责人 |
|----------|----------|--------|
| 添加账户流程 | 使用数据库配置，账户创建成功 | 全栈 |
| OAuth2 授权流程 | 使用选中的配置，授权成功 | 全栈 |
| 配额超限切换 | 自动切换到备用配置 | 后端开发 |
| 数据库故障降级 | 使用硬编码配置 | 后端开发 |

### 6.3 性能测试

| 测试项 | 目标 | 工具 | 负责人 |
|--------|------|------|--------|
| API 响应时间 | < 100ms | Apache Bench | 后端开发 |
| 并发请求 | 100+ 并发无错误 | Apache Bench | 后端开发 |
| 缓存命中率 | > 90% | Redis Monitor | 后端开发 |

---

## 7. 交付清单

### 7.1 代码交付

#### 后端
- [ ] 数据迁移文件（2个）
- [ ] Model 文件（2个）
- [ ] Repository 文件（2个）
- [ ] Service 文件（修改 1 个，新增 1 个）
- [ ] Handler 文件（2个）
- [ ] 路由注册代码
- [ ] 单元测试文件（多个）
- [ ] 集成测试文件（多个）

#### 前端
- [ ] 类型定义文件
- [ ] API 服务文件
- [ ] OAuth2ClientSelector 组件
- [ ] 修改后的 AccountForm 组件
- [ ] 修改后的 OAuth2AuthButton 组件
- [ ] 组件测试文件

### 7.2 文档交付

- [ ] 需求文档
- [ ] 技术设计文档
- [ ] 任务拆解文档（本文档）
- [ ] API 接口文档
- [ ] 数据库迁移指南
- [ ] 管理员操作手册
- [ ] 测试报告

### 7.3 其他交付

- [ ] 数据迁移脚本
- [ ] 定时任务配置
- [ ] 监控配置
- [ ] 部署指南

---

## 8. 验收标准

### 8.1 功能验收

- [ ] 提供商配置存储在数据库中
- [ ] 前端可以从数据库获取提供商列表
- [ ] 支持多个 OAuth2 配置
- [ ] 用户可以选择使用哪个配置
- [ ] 配额管理正常工作
- [ ] 配额超限时自动切换
- [ ] 现有功能不受影响

### 8.2 性能验收

- [ ] API 响应时间 < 100ms
- [ ] 支持 100+ 并发请求
- [ ] 缓存命中率 > 90%

### 8.3 质量验收

- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
- [ ] 性能测试通过
- [ ] 代码 review 通过
- [ ] 文档完善

### 8.4 安全验收

- [ ] Client Secret 加密存储
- [ ] 管理接口有权限控制
- [ ] API 不返回敏感信息
- [ ] SQL 注入防护

---

## 9. 里程碑

### M1: 阶段一完成（Week 2 Day 1）
- [ ] 提供商配置化功能完成
- [ ] 集成测试通过
- [ ] 文档完成

### M2: 阶段二核心功能完成（Week 3 Day 5）
- [ ] OAuth2 多客户端配置完成
- [ ] 配额管理功能完成
- [ ] 前端集成完成

### M3: 项目完成（Week 4 Day 5）
- [ ] 所有功能验收通过
- [ ] 性能测试通过
- [ ] 文档完整
- [ ] 准备上线

---

## 10. 开发规范

### 10.1 代码规范

- **Go 代码**：遵循 Go 官方代码规范，使用 `gofmt` 格式化
- **TypeScript 代码**：遵循 ESLint 配置，使用 Prettier 格式化
- **命名规范**：
  - Go：驼峰命名（PascalCase 导出，camelCase 私有）
  - TypeScript：驼峰命名（camelCase）
  - 数据库：蛇形命名（snake_case）

### 10.2 Git 规范

- **分支命名**：
  - 功能分支：`feature/provider-config`
  - 修复分支：`fix/oauth2-quota-bug`
- **Commit 规范**：
  - `feat: 添加 Provider 数据模型`
  - `fix: 修复配额计数不准确问题`
  - `docs: 更新 API 文档`
  - `test: 添加 OAuth2ClientService 单元测试`
  - `refactor: 重构配额管理逻辑`

### 10.3 Code Review

- 所有代码必须经过 review 才能合并
- Review 关注点：
  - 功能正确性
  - 代码质量
  - 测试覆盖
  - 文档完善
  - 性能考虑
  - 安全性

---

## 11. 沟通计划

### 11.1 日常沟通

- **Daily Standup**：每天 10:00，15 分钟
  - 昨天完成的工作
  - 今天计划的工作
  - 遇到的问题和阻碍

### 11.2 周会

- **Weekly Review**：每周五 15:00，1 小时
  - 本周进度回顾
  - 下周计划
  - 风险评估
  - 经验总结

### 11.3 里程碑评审

- **M1 评审**：Week 2 Day 1 下午
- **M2 评审**：Week 3 Day 5 下午
- **M3 评审（最终验收）**：Week 4 Day 5 下午

---

## 12. 工具和环境

### 12.1 开发工具

- **IDE**：
  - Go：VSCode + Go 插件 / GoLand
  - TypeScript：VSCode + Volar
- **数据库管理**：
  - PgAdmin / DBeaver / psql
- **API 测试**：
  - Postman / curl / HTTPie
- **版本控制**：
  - Git + GitHub

### 12.2 测试工具

- **单元测试**：
  - Go：`testing` 包 + `testify`
  - TypeScript：Jest / Vitest
- **集成测试**：
  - Go：`httptest`
  - E2E：Playwright (可选)
- **性能测试**：
  - Apache Bench (ab)
  - wrk

### 12.3 监控工具

- **日志**：结构化日志（slog）
- **指标**：Prometheus (可选)
- **追踪**：分布式追踪（可选）

---

## 13. 附录

### 13.1 关键技术栈

| 层次 | 技术 | 版本 |
|------|------|------|
| 后端语言 | Go | 1.21+ |
| Web 框架 | Gin | Latest |
| ORM | GORM | Latest |
| 数据库 | PostgreSQL | 14+ |
| 缓存 | Redis | 6+ |
| 前端框架 | React | 18+ |
| 语言 | TypeScript | 5+ |
| UI 库 | shadcn/ui | Latest |

### 13.2 参考资料

- [需求文档](./requirements.md)
- [技术设计文档](./technical-design.md)
- [FusionMail 架构文档](../architecture.md)
- [Go 代码规范](https://go.dev/doc/effective_go)
- [TypeScript 风格指南](https://google.github.io/styleguide/tsguide.html)

---

## 14. 变更记录

| 版本 | 日期 | 作者 | 变更内容 |
|------|------|------|----------|
| v1.0 | 2025-11-21 | FusionMail Team | 初始版本 |

---

**文档状态**: 待分配任务
**下一步行动**:
1. 团队评审任务拆解
2. 确认任务负责人
3. 确认开发时间表
4. 开始第一阶段开发
