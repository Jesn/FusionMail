# 2.0 后续 C：swagger 注解质量补全

## Goal

把 handler 层 swagger 注解从"能生成"提升到"生成得准"：修复文档与代码的漂移，
统一路径前缀，并把前端手写类型收敛到生成物。

## 背景

- P6 已落地类型代码生成（`79a7370`，7,850 行生成物，删 9 个手写 interface）
- 漂移报告：`.trellis/tasks/10-27-2.0-p6-boundary/research/frontend-type-drift-report.md`

## 复核更正（2026-10-29）

本 PRD 初版（写于 2026-10-27 20:10）是照 HANDOFF 的陈旧描述写的，**未复核代码**。
P16（`0f7fa1d`，2026-10-27 **17:26**，早于本 PRD 2 小时 45 分）已完成其大部分内容。
逐条核对结果：

| 初版声称 | 实测（2026-10-29） | 结论 |
|---|---|---|
| 20+ 端点注解为裸 `dto.Response`（生成 `data: unknown`） | **0 处**。90 处 `@Success ... dto.Response` 全部已带 `{data=response.Xxx}` 覆盖 | 已被 P16 解决 |
| swagger.json 中 `data: {}` 端点数 20+ → 0 | 仅剩 **2 处**，且位于基类定义 `dto.Response` / `dto.PaginatedResponse` 的 `properties.data`，**不是端点**（端点全部走 `{data=...}` 覆盖，不引用基类 `data`） | 已被 P16 解决 |
| `api.generated.ts` 中 unknown 字段显著减少 | 311 处 `unknown` **全部**是 `[name: string]: unknown`（map 类型的 index signature），与响应类型缺失无关 | 指标本身选错了 |
| swag 不产出 required | **产出**：110 个 definition 中 25 个带 `required`；dto 包有 18 个 `binding:"...required..."` 字段 | 已被 P16 解决 |

P16 验收记录原文：「全库裸 `@Success {object} dto.Response` 清零（26 处 200 + 6 处非 200 同款）」
「swag 0 解析错误；definitions 72→110；paths 111 不变；无 unknown 残留」。

**因此本任务重新界定为下列三项真实剩余工作。**

## Requirements

### R1 修复孤儿注解导致的文档回归

`internal/handler/oauth2_handler.go` 经 `41de287`（handler 大文件拆分）后只剩 29 行，
文件末尾挂着一整块 `/auth/google/authorize` 的 swagger 注解，但函数体已搬到
`oauth2_google.go` 且**没有带注解**。后果：跑规范命令
`swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal`
会**丢掉**该路径（111 → 110），即提交的 `docs/swagger.json` 与代码不同步。

- 把注解块搬到 `oauth2_google.go` 的真实 `GoogleAuthorize` 上
- 补上真实 handler 读取但注解缺失的 `group_id` 查询参数
- 清理 `oauth2_handler.go` 的孤儿注释

### R2 统一 swagger 路径前缀

110 个路径中 90 个带 `/api/v1`、20 个不带，且 swagger.json **无 `basePath`**，
故那 20 个在文档里指向不存在的路径。实际路由由 `router.Group("/api/v1")` 挂载。

需补前缀的 20 个（均无带前缀的重复注解，非重复项）：

```
/auth/2fa/{backup-codes,disable,setup,status,validate,verify}
/auth/{change-password,login,me}
/auth/google/{callback,reauthorize/{account_uid},refresh/{account_uid},revoke/{account_uid}}
/auth/microsoft/{authorize,callback,reauthorize/{account_uid},refresh/{account_uid},revoke/{account_uid}}
/emails
/rules/{id}   (仅 PUT；同路径的 GET/DELETE 已带前缀)
```

### R3 前端手写类型收敛到生成物

`frontend/src/types/*.ts`（不含生成物）有 106 处手写 `export interface`。
逐个与 `components['schemas']` 比对，**字段等价**的改为别名：

```ts
export type Foo = Schemas['response.Foo'];
```

字段不等价的**保留手写**并写明理由。

#### 执行中发现的真实阻塞：响应 DTO 无 `required`

首轮比对结论是 **0 处可收敛**。根因不是字段不等价，而是：

> `swag` 只从 `binding:"required"` 推导 `required`，而 `binding` 是**请求校验**概念，
> 只存在于请求 DTO 上。因此 25 个带 `required` 的 definition **全部是请求 DTO**，
> **110 个响应 schema 一个都没有** —— 生成类型里所有响应字段都是可选的。

这正是 PRD 说的「类型生成覆盖率天花板」。已核实 `validate:"required"` 能让 swag
对响应 DTO 产出 `required`，且**零运行时影响**（全项目无 `validate` tag、
未配置 go-playground validator、gin 用默认 `binding` tag）。

故 R3 拆成两步：**先给响应 DTO 补 `required` 语义，再收敛类型**。

#### 补 `required` 的范围与规则

规则：**无 `omitempty` 的字段 = 必存**（有 `omitempty` 的字段可能不出现，不标）。

| 结构体 | 标注字段数 |
|---|---|
| `model.AdapterResponse` | 8 |
| `model.ProviderResponse` | 20（5 个 omitempty 跳过） |
| `model.ProviderAdapterResponse` | 3 |
| `model.OAuth2ClientResponse` | 14（`provider_name` 跳过） |
| `model.RuleCondition` / `model.RuleAction` | 3 / 2 |
| `response.OAuth2ClientListResponse` | 5 |
| `response.AttachmentUploadResponse` | 4 |
| `service.EmailListItem` / `service.EmailListResponse` | 23 / 5 |
| **合计** | **87** |

带 `required` 的 definition：**25 → 34**。`paths` 110、`definitions` 110 均不变。

## 契约红线

- 注解只描述现有行为，**不得顺手改任何响应形状**
- `swagger.json` 变更必须逐字段核对；`definitions` 的增删要逐个说明
- 路径前缀变更只影响 `paths` 键，不影响 `components['schemas']`
  （已核实：前端仅 4 个文件消费生成物，且都取 `components['schemas']`，无人用 `paths` 键）

## Acceptance Criteria

- [x] `swag init` 重新生成后与提交版的差异**仅为预期变更**：21 处路径前缀统一 +
      2 处 `group_id` 参数新增；**操作数 128 → 128（零丢失）**，`definitions` 110 → 110 零改动
- [x] swagger.json 中无前缀路径数：21 → 0；`paths` 总数 111 → 110
      （110 而非 111：`/rules/{id}` 的 PUT 并入已存在的 `/api/v1/rules/{id}`，
      故 21 个无前缀键被 20 个带前缀键取代）
- [x] `definitions` 数量不变（110），无字段增删
- [x] 至少 5 处手写 interface/type 收敛为生成物别名（实际 **6 处**，清单见验收记录）
- [x] `go build ./... && go vet ./... && go test ./... -race` —— 除 `test/integration` 的既有失败外全绿（已在独立 worktree 对原始 HEAD 复现）
- [x] 前端 `tsc --noEmit && vitest run && vite build` 全绿（tsc 0 错 / 11 files 67 tests / build 6.12s）
- [x] 新增回归测试 `internal/handler/swagger_sync_test.go`：断言每条 `@Router` 注解
      都出现在 `docs/swagger.json` 里（已做变异验证：注入孤儿注解 → FAIL，恢复 → PASS）

## 验收记录（2026-10-29）

### 漂移实测（提交版 vs 代码可生成版，操作级）

| 操作 | 提交版 | 重新生成（修复前） | 根因 |
|---|---|---|---|
| `GET /auth/google/authorize` | 完整 | **整条消失** | `oauth2_handler.go` 末尾孤儿注解（含 `@Router`） |
| `GET /auth/microsoft/authorize` | 完整 | **内容被清空**（summary/params/responses 全丢） | `oauth2_google.go` 末尾孤儿注解（**无** `@Router`）+ `@Router` 单独留在 `oauth2_microsoft.go:14` |

两处均源于 `41de287`（handler 大文件拆分）。Microsoft 那处还**丢了 `@Failure 400/500` 两行**
（拆分时被截断），已按 `git show 41de287^` 的原始块还原。

> 教训：初版 PRD 只对比了 swagger 的**路径键**，漏了**操作内容**的漂移。
> 只比键会漏掉「路径还在但操作被清空」这类问题。

### 契约核对结果

```
操作数       128 -> 128   （零丢失）
paths        111 -> 110   （21 个无前缀键 -> 20 个带前缀键）
definitions  110 -> 110   （零增删、零字段变更）
内容差异操作 2 处，均为有意新增 group_id 参数
```

### R3 收敛记录（6 处）

| # | 手写类型 | 文件 | 收敛为 |
|---|---|---|---|
| 1 | `AttachmentUploadResponse` | `email.ts` | `Schemas['response.AttachmentUploadResponse']` |
| 2 | `Adapter` | `adapter.ts` | `Schemas['model.AdapterResponse']` |
| 3 | `ProviderAdapter` | `adapter.ts` | `Schemas['model.ProviderAdapterResponse']` |
| 4 | `OAuth2Client` | `oauth2.ts` | `Schemas['model.OAuth2ClientResponse']` |
| 5 | `OAuth2ClientListResponse` | `oauth2.ts` | `Schemas['response.OAuth2ClientListResponse']` |
| 6 | `Provider` | `provider.ts` | `Schemas['model.ProviderResponse']` |

计数口径（排除 `api.generated.ts`）：

```
结构 interface    106 -> 100   (-6)
生成物别名    5 -> 11    (+6)
```

#### 保留手写的类型及理由

| 类型 | 理由 |
|---|---|
| `index.ts` `RuleCondition` / `RuleAction` | 手写用**字面量联合**表达后端枚举，比生成的裸 `string` 精确。收敛会**降低**类型安全，不做。另：两者联合值与后端文档不一致，属既有契约漂移，另开任务。 |
| `index.ts` `EmailListResponse` | 元素类型不同：手写 `emails: Email[]`（领域型，字段更富）vs 生成 `emails: service.EmailListItem[]`。收敛会改变应用领域模型。 |
| `email.ts` `SendEmailRequest` | 字段不同：手写多 `in_reply_to`/`references`；后端多 `reply_to`（本次未改，属既有漂移）。 |
| `email.ts` `UpdateSMTPConfigRequest` | 前端**本地形态**：`services/smtpService.ts:47` 已将 `smtp_enabled` 映射为线上的 `enabled`，线上契约正确。 |

#### 测试结果

```
后端  go build ./...                     OK
       go vet ./...                       OK
       go test ./... -race                除 test/integration 外全 ok
       test/integration                   FAIL: TestSpamDetectionIntegration
                                          （sql: database is closed —— 已在独立 worktree
                                           对原始 HEAD 63d4535 复现，确认是既有失败）
前端  npx tsc --noEmit                   PASS
       npx vitest run                     11 files / 67 tests passed
       npx vite build                     built in 6.12s
```


## Notes

- swag 工具：`/home/agentuser/go/bin/swag`（v1.16.4；CI 提示 v1.16.6）
- 生成物：`backend/docs/docs.go`（豁免大文件治理，生成物）
- 前端生成物：`frontend/src/types/api.generated.ts`（同豁免）
- CI 的 `scripts/ci-quality-gate.sh` 只跑 `swag init` **不 diff**，故漂移长期不可见 ——
  AC 最后一条即为补这个门禁
