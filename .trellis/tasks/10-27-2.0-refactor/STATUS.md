# FusionMail 2.0 重构 — 进度状态

> 生成时间：2026-10-27
> 执行方式：无人值守，按 Trellis 流程「拆解 → 生成 → 执行 → 验证 → 更新状态」循环
> 基线：`go build` / `go vet` / `go test ./... -race` 全绿；前端 `tsc` / `vitest` / `eslint` 全绿

---

## 一、起点：三份架构体检的结论

��行前先做了完整取证（后端 291 文件 / 87,820 行，前端 202 文件 / 41,294 行）：

| 维度 | 基线值 |
|------|--------|
| 后端测试覆盖率 | **13.0%**（service 7.8% / handler 5.3% / repository 7.9% / spam 3.7%） |
| 前端测试 | 8 文件 / 36 个 `it` / 触达面约 2.3% |
| CI 质量门禁 | **零**（只有 docker build/push/deploy） |
| `test/integration` | **整个包编译不过**（`openSQLiteMemoryDB` 从未定义） |
| SQL 迁移 | 39 个文件**从未被执行**；实测按序跑：24 成功 / 15 失败，**无法从零建库** |
| 加密密钥 | 静默 pad/truncate；无 key-id；无轮换路径 |

**核心命题**：2.0 不是「重写代码」，是「建立反馈回路 + 把已经做对的地方变成编译期强制」。
不做 big-bang 重写——在 2–3% 覆盖率下重写 13 万行等于蒙眼重写。

---

## 二、已完成（8 个阶段，9 次提交）

### ✅ P0 环境与基线取证

- 安装 Go 1.25.14 / PostgreSQL 16.15 / Redis 7.0.15（apt 源从不可解析的
  `mirrors.tencentyun.com` 切到 `mirrors.aliyun.com`）
- **修复 `test/integration` 编译失败**：补齐缺失的 `openSQLiteMemoryDB`，
  并修正 `cache=private` 导致 AutoMigrate 与查询落到不同 SQLite 连接的陷阱
- 收敛 SQLite 单连接 + `busy_timeout`，消除 `-race` 下的 `table locked` 抖动
- 建立可复现基线：`go test ./... -race` 全绿，覆盖率 13.0%

### ✅ P1 CI 质量门禁（提交 `2804435`）

- `.cnb.yml` 新增 `quality-gate`，位于**所有 build/deploy 之前**：
  `go vet` + `go test -race` + `tsc --noEmit` + `vitest` + `eslint`
- 新增 `scripts/ci-quality-gate.sh`，本地一键执行同一套门禁
- **门禁立刻抓到真实问题**：`pkg/goroutine/pool.go:293` vs `:310` 的数据竞争

### ✅ P2 P0 缺陷修复（��交 `2804435` + `93425ea`）

后端 8 项 + 前端 4 项，全部有验证：

| # | 缺陷 | 修复 |
|---|------|------|
| 1 | 公开 API 泄露 `dedupe_key`/`synced_at` | 新增 `response.PublicEmailItem` 逐字段映射，`Emails interface{}` 改强类型 |
| 2 | WebAPI 配置明文返回密码 | `sanitizeAuthData` 改**按字段名反射脱敏**（fail-closed），修掉 CloudMail.Password 与 Cloudflare.WebhookSecret 两处泄露 |
| 3 | 任意登录用户可读/下载/清空日志 | 新增 `middleware.RequireAdmin`，挂到 `/logs` 与 `/dev` |
| 4 | 日志下载路径穿越 | 白名单校验（backend/frontend），5 处未校验的 `filepath.Join` 全部收口 |
| 5 | goroutine 泄漏 + 数据竞争 | 无缓冲 channel 改带缓冲 + 结果经 channel 回传，不再改写共享 `result` |
| 6 | 限流降级计数器 race + 内存无界 | 加互斥锁 + 定期清理协程 + `Stop()` |
| 7 | 加密失败明文落库 / 解密失败当明文用 | 3 处 fail-open 改 fail-closed |
| 8 | 密钥静默 pad/truncate | 新增 `deriveKey`，长度不符直接报错；���带消除 `config` 与 `crypto` 的重复常量定义 |
| 9 | 前端 API 层 `<T = any>` 类型逃逸 | 改必填泛型，暴露并修掉 9 处解包不一致 |
| 10 | react-hot-toast 提示全部不显示 | 12 个文件迁移到 sonner，移除依赖 |
| 11 | 深色模式刷新闪白 | 统一主题存储键，`initTheme()` 读正确的键 |

> **重要更正**：任务描述里「`emailListService` 双重解包」的根因判断**不成立**。
> `api.ts` 同时导出默认的原始 axios 实例与具名解包版，该文件用的是默认导出，
> `response.data` 恰好只解一层，原代码正确。已实测证伪并只做约定统一。

### ✅ P3 死代码清理（部分，约 1,470 行）

- `pkg/goroutine/{pool,semaphore,api,errors}.go` —— 0 引用，且 pool 内含数据竞争
- `pkg/queue/redis_queue.go` —— 0 importer
- `pkg/event/bus.go` + `service/event_service.go` —— 事件驱动架构从未接线
- `service/attachment_service.go` + `handler/attachment_handler.go` —— 路由被注释
- `middleware/error_handler.go` + `response.go:ErrorHandler` —— 0 调用，含不安全的 `err.(error)` 断言

**未完成**：`spam/cache_manager.go`（721 行）与 `fallback_manager.go`（401 行）
虽 `NewCacheManager` 零调用，但 `CacheManager` 类型被 5 个 struct 穿线，
删除需改多个构造函数签名，风险收益比不佳，留作后续。

### ✅ P4 schema 单一真相（提交 `0880d45`）

**取证结论**：39 个 SQL 迁移按序跑在干净 PostgreSQL 上 **24 成功 / 15 失败**——
`001:97` 用 PostgreSQL 保留字 `references` 作列名导致语法错误，`033:127` 的 `DO $`
少了 `$$`，后续全部级联失败。**这套迁移链根本无法从零建库**，而线上 schema
实际由 AutoMigrate 推导。

**修复**：
1. 从「AutoMigrate 在干净 PG 上的真实产出」导出 `000001_baseline.up.sql`
   （909 行 / 24 表），语义校验：列 350/350、表 24/24、索引 98/98、约束 36/36、序列 22/22 **完全一致**
2. 接入 **golang-migrate**：`pkg/database/migrate.go`，版本记录写入 `schema_migrations`
3. `cmd/migrate` 改造：`up` 走版本化迁移（不再 AutoMigrate、不再无条件 seed）；
   新增 `down` / `seed` / `automigrate`（显式）
4. **收紧 AutoMigrate 边界**：release 模式一律拒绝，即使显式 `ENABLE_AUTO_MIGRATE=true`
5. **修 Dockerfile**：此前没拷贝 `migrations/`，容器内执行迁移必然失败
6. 重写迁移契约测试（原测试编码的是旧契约）

### ✅ P5 加密密钥生命周期（提交 `9566ae7`）

- 密文格式升级为 `v2:<keyID>:<base64>`，keyID = 密钥 SHA-256 前 4 字节
- **向后兼容**：无前缀的历史密文逐把密钥尝试解密，**升级不需要停机重加密**
- 支持 `ENCRYPTION_KEY_PREVIOUS`（逗号分隔），使轮换成为可能
- 未知 key-id 报明确错误并提示补救方式
- 新增 7 个测试：格式、历史密文兼容、**轮换**、未知 key-id、非法密钥拒绝、去重、环境变量解析

### ✅ P7 统一响应契约（提交 `97c5c94`）

- **修正 210 处与实现不符的 Swagger 注解**：handler 声称返回 `response.Response`，
  但**没有任何 handler 文件 import 该包**，实际返回 `dto.Response`（多一个 `Code` 字段）。
  已提交的 `docs/swagger.json`（283KB）描述的信封类型与运行时不符。
- 消除第三套响应写法：3 个 handler 用裸 `c.JSON` + `gin.H` 手工拼装（53 处），
  其中 **24 处直接把 `err.Error()` 返回前端**（泄露 SQL 片段与内部路径）。
  迁移到 `dto.HandleServiceError` 等助手，替换 37 处。

### ✅ P11 AI 接入点需求分析（文档已产出）

`research/ai-opportunity-analysis.md`（1,122 行，7 章 + 证据索引，全部带 `文件:行号`）

**六个关键发现**：
1. **两套互不相通的规则引擎，都有硬伤**：用户规则引擎的 `add_label`/`remove_label`/`webhook`
   是 TODO 未实现（`rule_service.go:367-369`）→ AI 生成"打标签"规则目前完全无效；
   垃圾规则引擎的行为由**规则名里的中文子串**驱动（`rule_engine.go:258-280`）
2. **LLM/AI 代码与配置位为零**，唯一可复用抽象是 `TranslationService`
3. **异步地基是空的**：`pkg/queue` 与 `pkg/event` 都是死代码（本次已删），
   清理后代码库里**不存在任何可用的异步任务执行原语** → 异步框架从"锦上添花"升级为硬前置
4. **中文全文搜索实际是失效的**：`email.go:304` 用 `to_tsvector('english')` 对中文必然
   匹配失败，总是降级到 `ILIKE` 全表扫，且 GIN 索引在 `migrations/001:110` 是被注释掉的
5. **后端从不 sanitize 正文**（`sync_processor.go:248-249`），前端 sanitize 对后端 AI 无效；
   且 `spam_detector_safety.go:20-37` 是 fail-open（不能照抄）
6. **`webhook` 动作是双刃剑**：实现它必须同时引入 URL 白名单，否则 AI 生成规则 +
   正文注入 = SSRF 通道

**排序按「注入危害面」从低到高**（而非单纯价值）：
一期 25 人日（LLM 地基/摘要/翻译增强/只读分类）→ 二期 34（异步框架/自动打标/规则生成/运维归因）
→ 三期 53（语义搜索/垃圾检测/RAG/智能回复），加共享地基 12，**总计 ≈112 人日**。

**两点待你决策**：
1. `pkg/queue` 去留 —— 本次 P3 已作为死代码删除。若二期要做异步，建议**重写**而非复活
   （原实现 BLPop 弹出即丢、无 ack/DLQ、锁无 owner 校验）
2. **默认 provider 必须是本地模型** —— 现有翻译功能已欠隐私债
   （`translation_service.go:95-97` token 拼在 URL query + 正文明文外发无提示无脱敏），
   AI 接入会放大 10 倍

---

### ✅ P6 分层边界强制（提交 `f82539f`）

用 `swag init` 重新生成 API 文档后精确统计，**5 个带 gorm tag 的持久化模型
出现在 API 契约里**，暴露 `created_at` / `updated_at` / `deleted_at`：

| 模型 | 进入路径 |
|------|----------|
| `model.EmailRule` | `response.TestRuleResponse.Rule *model.EmailRule`（"DTO"只是包了一层） |
| `model.SentEmail` | `service.ListSentEmailsResponse.Emails []*model.SentEmail` |
| `model.Provider` | `provider_handler` 直接把请求体绑定到 `model.Provider` |
| `model.ProviderAdapter` | 被 `model.Provider` 的关联字段级联带出 |
| `model.Adapter` | 被 `model.ProviderAdapter.Adapter` 级联带出 |

修复：新增 `response.RuleSummary` / `response.SentEmailResponse` /
`request.ProviderRequest`，逐字段显式映射。**泄露数 5 → 0**，契约定义数 75 → 72。

**守卫测试**（`internal/dto/response/boundary_test.go`）：
- `TestNoGORMModelLeaksIntoAPIContract` —— 解析 swagger.json，断言任何带 gorm tag
  的模型都不得出现在 API 契约里
- `TestAPIContractHasNoGhostEnvelopeType` —— 断言不存在幽灵类型 `response.Response`

**已验证守卫真的会拦**：临时注入 `model.Adapter` → 测试 FAIL 并精确报出模型名。
CI 里已加入 `swag init` 前置步骤，否则守卫测试会因缺文档而跳过。

### 🔄 P10 业务逻辑重构（进行中，2 项已落地）

**① 中文全文搜索失效修复（提交 `6860013`）**

`Search` 先用 `to_tsvector('english', ...)` 做全文搜索，但 **'english' 配置对中文
整段只产生一个 token** —— 实测 `plainto_tsquery('english','财报')` 命中数为 **0**。
所以中文查询必然匹配不到，每次白跑一次全表 tsvector 计算再降级到 ILIKE，
而 `migrations/001:110` 的 GIN 索引是被注释掉的，降级路径连索引都没有。

修复：新增迁移 `000002_email_search_trgm` 启用 `pg_trgm`（PG13+ trusted extension，
数据库 owner 即可创建）并为四个可搜索列建 `GIN(gin_trgm_ops)` 索引；
应用层新增 `isASCIIQuery`，含非 ASCII 的查询直接跳过无用的英文全文搜索。

验证（强制关闭 seqscan 排除小表干扰）：
```
2 字中文 '%财报%'    → Bitmap Index Scan on idx_emails_subject_trgm  ✅
4 字中文 '%季度财报%' → Bitmap Index Scan on idx_emails_subject_trgm  ✅
英文   '%quarterly%' → Bitmap Index Scan on idx_emails_subject_trgm  ✅
```
功能实测：插入真实中文邮件后，中文查询命中「季度财报已发布 / 张三」✅

**② 主列表复合索引（提交 `d5199cd`）**

主列表固定按 `(account_uid, sent_at DESC)` 取数，但表上只有单列索引，
实测计划是 `Seq Scan + Sort`。补 `idx_emails_account_sent_at` 后：
```
Limit -> Index Scan using idx_emails_account_sent_at
```
**Sort 节点消失**。同时移除一个与 AutoMigrate 重复的全文索引，
并**首次实测了迁移的 down 路径**（`down -steps=1` → `up` 回到版本 2）。

**P10 剩余**：N+1（分组 2N / 子账户 N / 批量删除 2N）、offset 分页改 cursor、
`add_label`/`remove_label`/`webhook` 三个 TODO 动作实现。

## 三、未完成（需要后续会话）

| 阶段 | 内容 | 预估 |
|------|------|------|
| **P6 剩余** | model 层 517 个 `json:"` tag 尚未移除（后端侧已通过守卫测试阻止**新增**泄露，但存量 tag 仍在）；前端接入 `openapi-typescript` 从 `swagger.json` 生成类型，删掉 37 个手写 interface + 8 个重名类型 | 3–5 人日 |
| **P8** 消除全局状态与显式 DI | 消灭 `database.DB` / `redis.Client` 全局单例（`GetDB()` 被 14 处调用）；消灭 3 个 adapter 的 `init()` 隐式注册；消灭 `NewInitService()` 被 new 3 次；26 处 `context.Background()` 裸 goroutine 改 ctx + errgroup | 5–8 人日 |
| **P9** 上帝对象绞杀 | `syncService` 1,781 行/6 文件、`WebAPIProviderService` 1,634/6、`accountService` 1,093/5、`emailService` 985/3 —— 按**用例**（而非实体）拆分 | 8–12 人日 |
| **P10 剩余** | N+1（分组 2N / 子账户 N / 批量删除 2N）、API 全 offset 分页改 cursor、`add_label`/`remove_label`/`webhook` 三个 TODO 动作实现 | 8–12 人日 |
| **P3 剩余** | `spam/cache_manager.go` + `fallback_manager.go`（1,122 行） | 1 人日 |
| **P12** 前端架构收敛 | 3,221 行死代码、17 个 ≥500 行文件、4 套缓存语义并存、`AccountsPage.tsx` 1,471 行 | 8–12 人日 |

---

## 四、验证方式

```bash
# 后端（全部通过）
cd backend
go build ./... && go vet ./... && go test ./... -race -count=1

# 前端（全部通过）
cd frontend
npx tsc --noEmit && npx vitest run && npx eslint .

# 一键门禁
./scripts/ci-quality-gate.sh

# 迁移端到端验证
createdb fm_test && DB_NAME=fm_test go run ./cmd/migrate -action=up
psql fm_test -c "SELECT * FROM schema_migrations;"   # 应返回 1|f
```

---

## 五、本次提交清单

| 提交 | 内容 |
|------|------|
| `2804435` | fix(2.0-p0): 修复 7 项安全与正确性缺陷 + 接入 CI 质量门禁 |
| `93425ea` | fix(2.0-p0): 修复前端 3 项 P0 缺陷 + 消除 API 层类型逃逸 |
| `0880d45` | refactor(2.0-p4): schema 单一真相 —— 接入 golang-migrate 版本化迁移 |
| `9566ae7` | feat(2.0-p5): 加密密钥生命周期 —— 密文带 key-id，支持轮换 |
| `97c5c94` | fix(2.0-p7): 统一响应契约 —— 修正 210 处与实现不符的 Swagger 注解 |
| `b8ae624` | chore(trellis): 建立 2.0 任务树并清理僵尸任务 |
| `f82539f` | refactor(2.0-p6): 分层边界强制 —— 消除 5 个 GORM 模型泄露 + 加守卫测试 |
| `6860013` | fix(2.0-p10): 修复中文全文搜索失效 —— 加 trigram 索引 + 按字符集选路径 |
| `d5199cd` | perf(2.0-p10): 补邮件主列表复合索引 + 修正迁移中的重复索引 |

---

## 六、给下一次会话的建议起点

1. **先做 P6（分层边界强制）** —— 它是 P8/P9 的前置：只要 model 还带 `json` tag，
   任何结构重构都会继续产生契约泄露
2. **P9 按用例切，不按行数切** —— Go 没有 partial class，跨文件拆分只带来"文件级"错觉；
   `syncService` 有 15 个字段、30 个方法，必须按同步阶段（调度/执行/处理/错误策略）拆成独立类型
3. **每个阶段完成后必须真的 `task.py finish` + 归档** —— 这个项目最大的 Trellis 病症
   就是"活干完了任务不关"（执行前 20 个活跃任务只有 1 个归档）
4. **补 `.trellis/spec/backend/`** —— 88K 行后端代码至今零规范，
   `00-bootstrap-guidelines` 从 2026-06 至今仍是 `in_progress`
