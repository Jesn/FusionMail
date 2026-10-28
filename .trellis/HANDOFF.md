# FusionMail 剩余任务交接

> **给新会话**：先读本文件，再按 §1 挑一个任务开工。
> 生成时间：2026-10-28（P19「后端核心流程测试补强」完成并归档后）。
> 维护者：darren。本文件是跨任务交接，不属于任何单个 Trellis 任务。

---

## 0. 一句话现状

2.0 重构主线（`10-27-2.0-refactor`，22 个子任务）已完成 20 个（p0–p19），
**只剩 p20、p21**；另有一批 6–8 月的历史任务，其工作其实已被 2.0 重构落地，
但任务状态没更新（见 §3）。

---

## 1. 立即可做（2.0 主线剩余 2 项）

### P20 前端缓存语义统一 ← 建议先做

- 任务目录：`.trellis/tasks/10-27-2.0-p20-cache-unification/`（PRD 已写好）
- 状态：`planning`
- **开工前必须先问用户**：哪些��据要跨刷新持久化？三选一——
  1. 全部内存化（刷新即拉取，settingsCache 也废弃）→ 最简单
  2. 仅设置类持久化（现状语义）→ 统一为「内存 TTL + 设置白名单 localStorage」
  3. 引入统一缓存层（如 TanStack Query 的 staleTime/gcTime）→ 语义最清晰，改动最大
- 现状 4 套语义并存：

  | 缓存 | 机制 | TTL/失效 |
  |---|---|---|
  | accountStore | 内存 store | TTL 10min |
  | groupStore | 内存 store | TTL 5min |
  | emailCacheStore | getOrSet 惰性 | 无 TTL（引用失效） |
  | settingsCache | localStorage | 跨刷新持久化 |

- 涉及文件：`frontend/src/stores/*`、`frontend/src/hooks/useSettings.ts`（563 行，若动它需重新评估行数）
- 验收：用户决策记录进 PRD；全仓 grep 复核每类数据只有一条读写路径；缓存失效挂在对应 mutation 上（不能只靠 TTL）；`tsc` / `vitest`（基线 5 文件 24 测试）/ `vite build` 全绿；手工冒烟账户/分组/设置改动后刷新一致
- 纯前端任务，不需要 PG/Redis

### P21 swagger 注解质量补全

- 任务目录：`.trellis/tasks/10-27-2.0-p21-swagger-annotations/`（PRD 已写好）
- 状态：`planning`
- 目标：把 handler 注解从「能生成」提升到「生成得准」
  - 20+ 个裸 `@Success ... dto.Response` → 具体响应类型（`dto.XxxResponse` 或内联结构）
  - 文档化「在用但未注解」的端点
  - 请求 DTO 的 required 标注（swag 依据 `binding` 生成，需确认版本行为）
- 工具：`swag` 在 `/home/agentuser/go/bin/swag`
- **契约红线**：注解只描述现有行为，不得顺手改任何响应形状；swagger.json 变更逐字段核对
- 验收：swagger.json 中 `data: {}`（unknown）端点 20+ → 0；前端 `api.generated.ts` unknown 字段显著减少（前后数字记录）；后端 `build/vet/test -race` 全绿；前端 `tsc/vitest/build` 全绿；至少接掉 5 处手写 interface/any
- 漂移报告参考：`.trellis/tasks/10-27-2.0-p6-boundary/research/frontend-type-drift-report.md`

---

## 2. 环境与约束（新会话必读，否则会卡住）

### 沙箱坑（会直接卡住构建）

本机 Landlock 沙箱禁止写 workspace 之外的路径，`go build/test` 默认的
`~/.cache/go-build` 会 permission denied。**必须重定向 GOCACHE**：

```bash
cd /opt/apps/coding/FusionMail/backend
export GOCACHE=/opt/apps/coding/FusionMail/.trellis/.runtime/gocache \
       ENCRYPTION_KEY="test-2fa-secret-key-32-bytes-ok!" \
       JWT_SECRET="12345678901234567890123456789012"
```

（`.trellis/.runtime/` 已 gitignore；里面有一个 5.4G 的构建缓存副本，可随时删。）

### 本机可用 / 不可用

- 可用：PostgreSQL `127.0.0.1:5432`（`fusionmail` / `fusionmail_test_pw`，superuser）、Redis `127.0.0.1:6379`
- **不可用**：`docker`（所以无法本地验证「CI 里起 PG 服务」的改法）
- CI（`.cnb.yml`）**没有 PG/Redis**：DB 用例靠 `internal/testutil/pgtest` 整包跳过，
  必须保持「跳过而不是失败」

### 提交前验证（全跑，不许跳）

```bash
cd backend
go build ./... && go vet ./... && go test ./... -race -count=1          # 23 个包
DB_HOST=127.0.0.1 DB_PORT=1 TEST_REDIS_ADDR=127.0.0.1:1 go test ./... -count=1   # 降级模式，必须 0 FAIL

cd ../frontend
npx tsc --noEmit && npx vitest run && npx vite build                    # 基线 5 文件 24 测试
```

已知抖动（非本次引入）：`test/integration` 偶发失败，日志为 `sql: database is closed`
（垃圾检测异步 goroutine 在测试关闭 DB 后继续写库）。重跑即过。

### 工作流约定

- 用 Trellis：`/trellis:continue` 或 `trellis-start`；复杂任务要 `prd.md` + `design.md` + `implement.md` 三件套才 `task.py start`
- **干完必须 `task.py finish` + `task.py archive`** —— 本项目最大病史就是「活干完了任务不关」
- 归档时若 `branch` 与 `base_branch` 都是 `main`，用
  `task.py archive <task> --skip-branch-validation`（本项目直接在 main 上干活，无 PR）
- 规格沉淀在 `.trellis/spec/backend/{index,model-layer,service-layer,testing}.md`；
  学到新约定/踩到坑就写进去，和代码同一个提交批次

---

## 3. 历史遗留任务：工作已完成，状态没更新（建议归档）

以下都是本会话实测证据（不是推测）：

| 任务 | 声称状态 | 实测证据 | 建议 |
|---|---|---|---|
| `08-14-public-email-apis` | in_progress | 8 个 `public/mail` 端点全部存在（send/detail/delete/clear/sent/search/receive/mark-read）；跨账户校验 `email.AccountUID != account.UID` → 404 已实现（`internal/handler/public_mail_ops.go`） | 冒烟一次后归档 |
| `00-bootstrap-guidelines` | in_progress | spec 三个文件（model-layer / service-layer / testing）均 `Filled` | 直接归档 |
| `06-18-p2-api-contract` | planning | handler 无 `repository.EmailFilter{}`；`docs/swagger.json` 无 `gorm.DeletedAt`；守卫测试 `TestNoGORMModelLeaksIntoAPIContract` 通过 | 直接归档 |
| `06-18-p3-cleanup` | planning | `internal/adapter/manager.go` 已删除；`backend/tests/` 已不存在；`internal/webhook` 已不存在。**残余**：`config/config.go:64` 仍是 `const DefaultEncryptionKey = crypto.DefaultEncryptionKey` | 清掉残余后归档 |
| `06-19-p0-observability-shutdown` | in_progress | `LOG_FORMAT=json` 已实现（`pkg/logger/logger.go:117`）；`/metrics` 已注册（`internal/router/router.go:130`）；`appCtx`/`appCancel` + `srv.Shutdown(30s)` + `tracingShutdown` 已实现（`cmd/server/main.go:417-442`）。**残余**：`internal/sse/hub.go` 无 `Close`，SSE 长连接靠 `srv.Shutdown` 等 30s 超时 | 补 SSE 主动关闭后归档 |
| `06-18-backend-arch-standardization` | active 2/4 | 4 个子任务中 p0/p1 已完成，p2/p3 见上（实际已完成） | 子任务归档后归档父任务 |

> 这 6 个任务合计只剩 2 个真实残余项（`config` → `crypto` 常量、SSE 主动关闭），
> 都是一小时内能收掉的活。建议新会话先花半小时清干净，让任务板只剩 p20/p21。

---

### ⚠️ 2.0 主线收尾时的归档顺序（实测坑）

`task.py archive <父任务>` **不会递归归档子任务**，只会清掉子任务的 `parent` 字段
（把它们变成孤立顶层任务，见 `.trellis/scripts/common/task_store.py:1363` 附近）。
所以 p20/p21 完成后，除了归档父任务 `10-27-2.0-refactor`，还要**逐个归档**
已完成但仍在任务板上的子任务（p0–p18，共 19 个；p19 已归档）：

```bash
cd /opt/apps/coding/FusionMail
for t in $(ls .trellis/tasks | grep '^10-27-2\.0-p'); do
  python3 ./.trellis/scripts/task.py archive "$t" --skip-branch-validation
done
```

（`archive` 会把 `in_progress` 自动置为 `completed`；已是 `completed` 的直接归档。
父任务本身要等子任务都归档完再归档，否则子任务会被"脱钩"成顶层任务。）

---

## 4. 已归档（无需再管）

- `10-27-2.0-p19-core-flow-tests` —— 本会话完成（覆盖率 service 32.2% / spam 36.5%）
- 2.0 系列 `p0`–`p18` 全部 `completed`
- `08-07-inbox-optimization`、`08-21-fix-log-memory-crash` —— 本会话归档（纯记账）

---

## 5. 新会话开场提示词（���直接粘贴）

```
读 .trellis/HANDOFF.md，然后开始做 P20（前端缓存语义统一）。
先按 PRD 里的前置决策问我持久化口径，再动手。
```

或者先���历史遗留：

```
读 .trellis/HANDOFF.md，按 §3 把 6 个历史遗留任务收尾归档
（先补 config→crypto 常量和 SSE 主动关闭两个残余项），再开始 P21。
```
