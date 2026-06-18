# P2 API Contract

## Goal

治理后端 API 契约边界，逐步停止直接暴露 GORM model 和 repository 查询结构，让 handler、service、repository 三层职责更清晰。

P2 关注接口契约和类型边界，不主动进行大规模文件拆分。

## Background

当前部分接口存在跨层泄漏：

- handler 层直接构造 `repository.EmailFilter`。
- service 层方法直接返回 `*model.Email`、`*model.EmailAccount` 等 GORM model。
- Swagger 中可见 `gorm.DeletedAt` 等内部结构，说明 API 文档已经暴露存储层细节。
- repository 接口方法数量过多，调用方依赖范围不清晰。

## Scope

### Included

1. 为核心 Email / Account API 引入 response DTO。
2. `EmailService` 定义业务查询参数，handler 不再直接使用 `repository.EmailFilter`。
3. Repository 接口按消费方拆分，降低 mock 和依赖范围。
4. 调整 Swagger 标注，避免内部存储字段进入 API 契约。

### Excluded

- 不做全量 API DTO 化；优先核心高风险实体。
- 不改变前端字段名，除非明确标注兼容处理。
- 不改变数据库字段或模型存储结构。
- 不做路由重构；P1 已处理。

## Requirements

### R1: 核心响应必须由 DTO 承载

优先覆盖：

- 邮件详情：`EmailDetailResponse`
- 邮件列表：确认现有 `EmailListItem` 是否足够，必要时移动到 response DTO 包。
- 账户详情/列表：`AccountResponse`
- 同步状态/进度：确认是否返回稳定 response 类型。

DTO 不应包含：

- `gorm.DeletedAt`
- `DedupeKey`
- 内部同步游标
- 加密凭证
- 内部错误追踪字段，除非 API 明确需要

### R2: handler 不直接依赖 repository filter

- handler 将 HTTP query 解析为 request DTO 或 service query 参数。
- service 内部决定如何映射到 repository filter。
- 默认过滤策略（如不显示 deleted/spam）在 service 层集中表达，handler 只负责读取输入。

### R3: repository 接口按调用场景拆分

- 保留现有具体实现结构体。
- 通过小接口让 service 只依赖所需能力。
- 拆分接口不得造成大量重复实现。
- 优先拆 `AccountRepository` 和 `EmailRepository` 中最明显的读/写/同步/分组边界。

### R4: API 文档不暴露存储层细节

- 内部字段加 `swaggerignore` 或通过 DTO 避免出现在文档。
- 新增 response 类型需要保持 JSON 字段命名和前端兼容。

## Acceptance Criteria

- [ ] 邮件详情接口不直接返回 `*model.Email`。
- [ ] 账户核心接口不直接返回未裁剪的 GORM model。
- [ ] `handler/email_handler.go` 不再构造 `repository.EmailFilter`。
- [ ] `repository.EmailFilter` 不再作为 service 对外契约。
- [ ] Swagger 不再为核心接口暴露 `gorm.DeletedAt`。
- [ ] 拆分 repository 接口后调用方依赖范围更小，且不破坏现有实现。
- [ ] `go test ./...` 可运行。

## Validation

- `cd backend && go test ./...`
- 重新生成或检查 Swagger 文档，确认核心响应类型不泄漏 GORM 内部字段。
- 前端关键页面字段兼容性检查：邮件列表、邮件详情、账户列表。

## Notes

P2 必须在 P1 之后执行，避免 API 契约调整和结构拆分同时发生，降低 review 和回滚难度。