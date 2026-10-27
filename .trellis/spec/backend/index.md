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
   （22 个包），前端 `npx tsc --noEmit && npx vitest run`。

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
