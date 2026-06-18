# P1 Structure Split

## Goal

在 P0 修复关键行为和依赖绕过后，对后端主要超大文件和职责混杂模块进行结构拆分，使项目达到中型 Go 服务可维护的模块边界。

P1 关注代码组织和依赖组装，不主动改变业务行为。

## Background

当前后端存在多个过大的关键文件：

- `backend/cmd/server/main.go`：约 692 行，混合启动、依赖组装、路由、静态文件和生命周期。
- `backend/internal/service/sync_service.go`：约 1828 行，混合同步调度、执行、处理、错误策略、WebAPI 同步。
- `backend/pkg/database/database.go`：约 712 行，混合数据库连接和业务种子/修复逻辑。
- `backend/internal/router/router.go`：约 606 行，`SetupRouter` 参数过多，并保留已无实际用途的 `swaggerEnabled` 参数。
- 多个 service / adapter 文件超过 600 行，职责边界不清晰。

## Scope

### Included

1. 拆分 `main.go` 的依赖组装为显式 container。
2. 按同步状态机职责拆分 `sync_service.go`。
3. 从 `pkg/database` 迁出业务种子数据和修复逻辑。
4. 用 `RouterDeps` 收敛路由参数。
5. 拆分若干核心超大 service 文件，降低单文件复杂度。

### Excluded

- 不引入 wire/fx。
- 不改变 API response 契约；归 P2。
- 不清理 AdapterManager 等死代码；归 P3。
- 不改变数据库 schema。

## Requirements

### R1: 入口层职责分离

- `cmd/server/main.go` 保留启动流程主线：加载环境、配置、数据库、依赖容器、路由、HTTP 生命周期。
- 依赖构建移动到 `cmd/server/container.go` 或等价文件。
- 容器结构清晰表达 Repositories、Services、Handlers、Runtime resources。

### R2: 同步服务按职责拆分

目标文件结构：

- `sync_service.go`：接口、结构体、构造函数、核心入口。
- `sync_scheduler.go`：定时调度与同步触发判断。
- `sync_executor.go`：同步执行路径，包括 UID、batch、legacy。
- `sync_processor.go`：邮件处理、去重、规则应用。
- `sync_error_policy.go`：认证错误判断、失败计数、自动禁用/回收策略。
- `sync_webapi.go`：WebAPI 同步路径。

拆分后必须保持包内行为一致，避免改动函数语义。

### R3: 数据库基础设施和业务 seed 分离

- `pkg/database` 仅负责连接、连接池、GORM 配置、关闭、获取 DB。
- Provider、Adapter、Setting seed 移动到 `internal/seed` 或等价内部包。
- 启动和迁移命令仍能调用 seed 入口。

### R4: 路由依赖收敛

- `SetupRouter` 接收 `RouterDeps` 或等价结构体，避免 20+ 参数函数签名。
- 移除 `swaggerEnabled` 死参数。
- 各 `Register*Routes` 共享已构建的 auth/rate limit 中间件配置，减少重复初始化。

### R5: 单文件大小治理

优先处理超过 600 行且职责混杂的业务文件：

- `account_service.go`
- `email_service.go`
- `oauth2_service.go`
- `webapi_provider_service.go`
- `spam_detector.go`

目标不是机械按行数切割，而是按职责切分，避免产生没有业务意义的文件。

## Acceptance Criteria

- [ ] `main.go` 明显缩小，依赖组装不再集中在主函数中。
- [ ] `sync_service.go` 按职责拆分，单个同步相关文件职责清晰。
- [ ] `pkg/database/database.go` 不再包含大段 Provider/Setting seed 数据。
- [ ] `SetupRouter` 不再使用 20+ 参数签名。
- [ ] 拆分后 `go test ./...` 可运行。
- [ ] 没有引入无意义的跨包循环依赖。
- [ ] 没有大规模格式化无关文件。

## Validation

- `cd backend && go test ./...`
- `cd backend && go test ./cmd/server ./internal/service ./internal/router ./pkg/database`
- 搜索确认：`swaggerEnabled` 死参数已移除。
- 搜索确认：`pkg/database` 不再包含业务 provider/setting seed 大段配置。

## Notes

P1 应在 P0 合并后执行，避免同时修改同步行为和大规模文件结构。