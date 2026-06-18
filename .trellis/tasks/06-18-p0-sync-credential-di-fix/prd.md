# P0 Sync Credential DI Fix

## Goal

优先修复后端架构优化中的高风险基础问题：同步批处理计数风险、凭证解析重复、service 内部绕过依赖注入、SSE 广播散落。

P0 的目标不是大规模重构，而是先把已经影响行为一致性和后续拆分安全性的关键问题处理掉，为 P1 结构拆分提供稳定基础。

## Background

当前同步相关代码存在以下风险：

- `backend/internal/service/sync_service.go` 中 `processBatchEmails` 返回累计值，`doSyncWithBatch` 外层继续累加，可能导致多批同步统计被放大。
- OAuth2 / quick / password 凭证解析、IMAP/POP3 配置、主机修复逻辑在多个文件重复实现。
- `sync_service.go` 中 `applyRulesForEmail` 通过 `database.GetDB()` 临时创建 `RuleService`，绕过构造函数依赖注入。
- `sync_service.go`、`webhook_receiver_service.go`、`progress_tracker.go` 等 service 直接调用 `sse.Broadcast`，业务层绑定传输实现。

## Scope

### Included

1. 修复批处理同步统计语义。
2. 抽取统一 `CredentialResolver`，集中处理账号凭证和协议配置。
3. 将规则应用器通过构造函数注入 `syncService`，去掉 `applyRulesForEmail` 内部临时构建。
4. 引入轻量通知接口，减少 service 对 `sse.Broadcast` 的直接依赖。
5. 为关键行为补充或调整测试。

### Excluded

- 不拆分整个 `sync_service.go` 文件；完整拆分归 P1。
- 不调整 API response DTO；归 P2。
- 不迁移数据库 schema。
- 不引入事件总线框架。
- 不引入 DI 框架。

## Requirements

### R1: 批处理同步计数必须按增量统计

- `processBatchEmails` 返回值必须表达“本批新增/更新/失败数量”，不能返回全局累计值。
- `doSyncWithBatch` 的 `totalNew`、`totalUpdated`、`totalFailed` 必须等于所有批次增量之和。
- `syncLog.EmailsNew`、`syncLog.EmailsUpdated` 最终值必须与实际处理结果一致。

### R2: 凭证解析必须有单一入口

- 新增 `CredentialResolver` 或同等命名的集中解析组件。
- 统一处理：
  - password 凭证；
  - OAuth2 凭证；
  - quick 凭证；
  - IMAP/POP3 host、port、encryption；
  - `mail.linuxdo.org` 到 `mail.linux.do` 的兼容修复。
- `sync_service.go`、`email_service.go`、`sync_manager.go` 不再各自复制解析逻辑。

### R3: 规则应用不得绕过依赖注入

- `syncService` 通过构造函数接收 `RuleService` 或更小的 `RuleApplier` 接口。
- `applyRulesForEmail` 内部不得调用 `database.GetDB()`。
- `NewSyncManager` 或后续容器构建处负责创建并注入规则服务。

### R4: service 层通知能力需要收敛

- 定义轻量通知接口，例如 `SyncNotifier`。
- 同步和 webhook 相关 service 通过接口发送“邮件计数可能变化”等事件。
- `progress_tracker.go` 可以保留独立进度推送，但如果改动成本可控，也应通过同一通知接口。
- handler 层可暂时保留直接 `sse.Broadcast`，因为 HTTP 层直接响应前端事件属于可接受边界。

## Acceptance Criteria

- [ ] 多批同步计数不会重复累加。
- [ ] 有测试覆盖至少两批邮件同步时的新增/更新计数。
- [ ] `sync_service.go` 中 `applyRulesForEmail` 不再创建 repository/service。
- [ ] service 层不新增 `database.GetDB()` 调用。
- [ ] 凭证解析重复逻辑减少，核心账号凭证解析从统一组件进入。
- [ ] 现有同步路径、连接测试、邮件删除/修复路径仍能通过编译。
- [ ] `go test ./...` 可运行；如失败，需要标注是否为既有失败。

## Validation

- `cd backend && go test ./...`
- 如可行，增加针对 `processBatchEmails` 或同步处理器的单元测试。
- 手动复核：搜索 `database.GetDB()`、`sse.Broadcast`、凭证解析重复片段的剩余使用点。

## Notes

P0 是后续结构化拆分的前置阶段。不要在本阶段进行大量文件搬迁，避免行为修复和结构 diff 混在一起。