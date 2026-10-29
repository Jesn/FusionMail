# Backend Development Guidelines

> FusionMail 后端（Go 1.25 + Gin + GORM + PostgreSQL）的可执行契约。
> 由 2.0 重构会话沉淀，每条规则都有守卫测试或取证依据。

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Model Layer](./model-layer.md) | gorm 模型边界：json tag 规则、JSON 持久化值对象、守卫测试 | Filled |
| [Service Layer](./service-layer.md) | 上帝对象绞杀模式（按用例拆分）、组合接口转发、DI 约定 | Filled |
| [Testing](./testing.md) | 测试约定：SQLite 内存库、mock 仓库、race 验证、守卫测试清单 | Filled |

---

## 硬性规则速查（违反即 CI 失败）

1. **gorm 表模型不得携带 json tag** —— API 线格式只允许定义在 `internal/dto/`。
   守卫：`internal/model/json_tag_guard_test.go`（三重测试）+
   `internal/dto/response/boundary_test.go`（swagger 泄露检测）。
   例外类型必须登记进 `jsonTagAllowlist` 并写明理由。
2. **响应体必须走 DTO** —— handler 禁止直接返回 `model.*`。
   守卫：`TestNoGORMModelLeaksIntoAPIContract`（解析 swagger.json）。
3. **纯 JSON 配置结构放 `internal/dto/webapiconfig`**，不放 model 包。
   这些类型的 json tag 是持久化格式（`auth_data`/`metadata` 列），不可删。
4. **每个阶段完成后必须真的 `task.py finish` + 归档** —— 本项目最大病史就是
   "活干完了任务不关"。
5. **提交前完整验证**：`go build ./... && go vet ./... && go test ./... -race -count=1`
   （23 个包），前端 `npx tsc --noEmit && npx vitest run`。
   无 PG/Redis 环境（等价 CI）另跑一次，必须 0 FAIL（DB 用例整包跳过）：
   `DB_HOST=127.0.0.1 DB_PORT=1 TEST_REDIS_ADDR=127.0.0.1:1 go test ./... -count=1`。

---

## 改动 API 契约前的强制检查

```bash
# 1. 改动前备份
cp backend/docs/swagger.json /tmp/swagger_before.json

# 2. 改动后重新生成并对比
cd backend && PATH="$PATH:~/go/bin" swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
diff /tmp/swagger_before.json docs/swagger.json
```

diff 非空 = 契约变更，必须在前端同步（`frontend/src/types/index.ts` + 对应 service），
且在提交信息中显式声明。

### 响应 DTO 的必填字段必须标 `validate:"required"`

`swag` **只从 `binding:"required"` 推导 `required`**，而 `binding` 是请求校验概念，
只出现在请求 DTO 上。后果：

> **响应 DTO 不标 required → 生成的 TS 类型里所有响应字段都是可选（`?:`）→
> `openapi-typescript` 的生成物无法替代前端手写类型。**

这是「类型生成覆盖率天花板」的真正根因（2026-10-29 实测：110 个响应 schema 中
带 `required` 的为 0）。

**写法**：给响应 DTO 加 `validate:"required"`（**不要用 `binding`**）：

```go
type AdapterResponse struct {
	ID          int64     `json:"id" validate:"required"`
	DisplayName string    `json:"display_name" validate:"required"`
	Adapter     *Adapter  `json:"adapter,omitempty"`   // 有 omitempty 就不标
}
```

**规则**：

| 字段情况 | 是否标 required |
|---|---|
| 无 `omitempty`（总会序列化） | **标** |
| 有 `omitempty`（可能不出现） | **不标** |
| 指针但无 `omitempty`（会序列化为 `null`） | 标（生成 `T \| null`） |

**为什么用 `validate` 而不是 `binding`**：

- 本项目的 `validate` tag **零运行时作用** —— 未配置 go-playground validator、
  gin 用默认 `binding` tag、全项目无其它 `validate` 使用。它纯粹是给 swag 的信号。
- `binding:"required"` 有真实校验副作用：`RuleCondition`/`RuleAction` 同时出现在
  请求体（`request.CreateRuleRequest`）里，标 `binding` 会让校验行为改变。
  且 bool 字段的 `binding:"required"` 对 `false` 会误判为缺失。

**改完必须重新生成前端类型**：`cd frontend && npm run generate:api-types`。
