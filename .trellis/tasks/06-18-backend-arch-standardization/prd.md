# Backend Architecture Standardization

## Goal

将 FusionMail 后端从早期快速演进形成的“能跑但依赖和职责逐渐混杂”的结构，调整为适合中型项目长期维护的模块化、结构化架构。

本任务不追求一次性重写系统，而是按 P0 → P3 分阶段治理：先修复真实行为风险，再统一重复业务规则，再收敛依赖图和文件结构，最后整理接口契约和清理历史遗留。

## Background

当前后端是 Go + Gin + GORM + PostgreSQL + Redis 的中型服务，代码规模约 66K 行。核心问题集中在：

- `backend/cmd/server/main.go` 约 692 行，混合配置、数据库、依赖组装、路由、静态文件、服务生命周期。
- `backend/internal/service/sync_service.go` 约 1828 行，混合同步调度、锁、凭证解析、邮件处理、错误策略、WebAPI 同步、SSE 通知。
- 凭证解析和账号协议配置逻辑散落在 `sync_service.go`、`email_service.go`、`sync_manager.go` 等多个位置。
- `sync_service.go` 的批处理统计存在累计值重复累加风险。
- `database.GetDB()` 被 service 内部直接使用，绕过正常依赖注入。
- API 层和 service 层直接暴露 GORM model，部分内部字段进入 Swagger/API 契约。
- `pkg/database/database.go` 混合基础设施和业务种子数据。
- 部分 repository 接口过大，调用方依赖范围不清晰。

## Scope

### Included

- 修复同步批处理计数行为风险。
- 抽取统一凭证解析与账号协议配置解析能力。
- 清理 service 内部直接 `database.GetDB()` 的依赖绕过路径。
- 收敛 service 层对 SSE 的直接调用。
- 拆分 `main.go`、`sync_service.go`、`database.go`、`router.go` 等超大/职责混杂文件。
- 为核心 API 引入 response DTO，逐步停止直接返回 GORM model。
- 解耦 handler/service 对 repository 查询结构的直接依赖。
- 清理死代码、重复测试目录、命名歧义和低风险边界问题。

### Excluded

- 不重写核心同步协议实现。
- 不引入 wire/fx 等依赖注入框架，除非后续证据证明手动容器无法满足维护需求。
- 不改变现有数据库 schema，除非某个阶段明确需要迁移且单独评审。
- 不改变前端交互行为，除非 API 契约治理阶段明确声明兼容方案。
- 不在父任务内直接执行所有代码实现；具体实现由阶段子任务承接。

## Delivery Structure

父任务负责全局约束、阶段依赖和验收标准。具体实施拆为 4 个阶段子任务：

1. `06-18-p0-sync-credential-di-fix`：行为修正 + 凭证解析 + DI 绕过修复。
2. `06-18-p1-structure-split`：入口、同步服务、数据库种子、路由和大文件拆分。
3. `06-18-p2-api-contract`：API 契约、DTO、filter 解耦和 repository 接口治理。
4. `06-18-p3-cleanup`：死代码、测试目录、常量依赖、命名消歧。

## Requirements

- 每个阶段必须能独立 review，不能把所有重构混在一个不可审查的大变更中。
- 每个阶段必须保持现有业务行为兼容，除非 PRD 明确标注行为修正。
- 优先小步迁移，避免一次性大规模移动导致回滚困难。
- 新增抽象必须服务于已识别的重复逻辑或边界问题，不做纯形式化抽象。
- 单文件目标：核心业务文件尽量控制在 500 行以内；复杂协议适配器允许例外，但需要明确职责边界。
- service 层不得新增对 `database.GetDB()` 的直接依赖。
- 新增 HTTP API 不得直接返回 GORM model。
- 每阶段完成后至少运行后端最小验证：`go test ./...`。如果耗时或环境阻塞，需要记录阻塞原因。

## Acceptance Criteria

- [ ] 已创建父任务和 P0/P1/P2/P3 四个阶段子任务。
- [ ] 父任务包含全局 `prd.md`、`design.md`、`implement.md`。
- [ ] 每个阶段子任务包含明确的 `prd.md`，说明目标、范围、非目标、验收标准。
- [ ] 阶段依赖顺序清晰：P0 → P1 → P2 → P3。
- [ ] DI 方案明确：默认采用手动 `buildContainer()`，暂不引入 wire/fx。
- [ ] 每个阶段都有验证要求和回滚边界。
- [ ] 任务文档不要求读者依赖聊天上下文即可理解执行目标。

## Risks

- 大规模文件拆分容易产生无意义 diff，必须避免格式化 churn。
- API DTO 化可能影响前端依赖，需要优先做兼容字段映射和逐步迁移。
- 同步服务是高风险核心路径，P0/P1 必须保持行为证据和测试覆盖。
- 旧任务 `00-bootstrap-guidelines` 仍处于 in_progress，本任务不直接处理它，避免混淆 Trellis 生命周期。