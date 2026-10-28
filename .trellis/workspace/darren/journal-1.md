# Journal - darren (Part 1)

> AI development session journal
> Started: 2026-06-18

---



## Session 1: MVP 收口：路由隔离、README 对齐、SentEmail 修复

**Date**: 2026-08-07
**Task**: MVP 收口：路由隔离、README 对齐、SentEmail 修复
**Branch**: `main`

### Summary

统一收件箱 MVP 收口任务：隔离生产调试路由（/oauth2-test、/debug/sse 仅 DEV 注册，/settings/legacy 生产重定向），按代码事实重写 README 能力矩阵与非目标，修复 SentEmail 模型缺失于 AutoMigrate 导致生产 sent_emails 表不存在的 500 错误。前端 36 测试通过，生产构建成功，已部署 Fly.io 并验证健康端点。

### Git Commits

| Hash | Message |
|------|---------|
| `64a10b5` | (see git log) |

### Status

[OK] **Completed**

---

## Session 2: 日志系统内存崩溃修复

**Date**: 2026-08-21
**Task**: 日志系统内存崩溃修复 (`.trellis/tasks/08-21-fix-log-memory-crash`)
**Branch**: `main`

### Summary

修复 `/logs` 页面打开 45MB 日志导致 Fly.io OOM 崩溃。`readAndFilterLogs` 原实现全量加载+解析+存储所有日志行到内存后分页，改为 ring buffer 流式方案（内存 O(page×page_size)）。同时修正日志保留策略设计缺陷：`CleanupRotatedAppLogs` 原来显式跳过主文件 `backend.log`，现增加主文件超阈值时强制 `Rotate()`；`DefaultAppLogMaxSizeMB` 从 50 降到 10。正则编译从 per-call 提升为包级变量。前端 stats+logs 并行加载改为串行。

### Key Decisions

- Ring buffer 流式方案而非 offset 双 pass 方案（实现更简单，内存可预测）
- `DefaultAppLogMaxSizeMB` 50→10（让主文件更早滚动）
- 预先存在的集成测试失败（`TestSpamDetectionIntegration`）与本次改动无关

### Status

[OK] **Completed**


## Session 3: 2.0 后续 A：后端核心流程测试补强（P19）
<!-- trellis-session: v=2 fp=0db72356417e4ec3 -->

**Date**: 2026-10-27
**Task**: 2.0 后续 A：后端核心流程测试补强（P19）
**Branch**: `main`

### Summary

补齐 service 层核心流程回归测试：internal/service 覆盖率 10.2% → 32.2%、internal/service/spam 10.3% → 36.5%（均达 ≥30% 目标）。新增 PG 测试夹具 internal/testutil/pgtest（每包独立库 fm_test_<pkg>、每测 TRUNCATE、PG 不可达即跳过，CI 保持绿）。用例暴露并修复 5 处生产缺陷，其中 4 处用「先还原缺陷 → 用例必须失败」验证过用例真能发现缺陷。

### Main Changes

- 新增 internal/testutil/pgtest 夹具：每包独立库 + TestMain 建库/AutoMigrate 一次 + 每测 TRUNCATE；种子表不清；PG 不可达 t.Skip（降级模式实测 125 SKIP / 102 PASS / 0 FAIL）
- 新增用例：发信（SendEmail/Reply/ReplyAll/Forward/凭据解析）、同步（Scheduler/Executor WebAPI/SyncProcessor）、垃圾判定（标记与批量删除/规则 CRUD/规则引擎/贝叶斯/信誉/黑白名单）、清理（回收站/垃圾/日志/EmailDeleteService）、配置服务、CredentialResolver
- 生产修复 1：SendService.factory 收窄为 mailSenderResolver 窄接口（P9 同模式），新增 sendVia 兜底 nil 结果
- 生产修复 2：BatchDeleteSpam 不存在的 ID 不再计入删除数
- 生产修复 3：spam.addToList 先规范化 target 再查重/写缓存键
- 生产修复 4：cleanupDeletedEmails 恒清理 0 行（deleted_at 与 List 作用域互斥）且误用软删除 → 改用 updated_at 判定 + PermanentDelete
- 生产修复 5：GetByCategory 缓存敏感配置时二次加密，命中缓存返回密文 → 与 refreshCache 一致，原样入缓存
- 规格同步：testing.md 增加 PG 夹具约定/Redis 依赖/窄接口 seam/GORM 三个坑/还原验证法/契约速查矩阵；index.md 更新包数与降级模式命令

### Git Commits

| Hash | Message |
|------|---------|
| `7a42b46` | test(service): 核心流程回归测试（发信/同步/垃圾判定/清理） |

### Testing

- [OK] go build ./... && go vet ./... 通过
- [OK] go test ./... -race -count=1：23 个包全绿
- [OK] 降级模式（DB_PORT=1 + TEST_REDIS_ADDR=127.0.0.1:1）：0 FAIL
- [OK] 覆盖率：internal/service 32.2%（基线 10.2%）、internal/service/spam 36.5%（基线 10.3%）

### Status

[OK] **Completed**

### Next Steps

- P20 前端缓存语义统一、P21 swagger 注解补全（.trellis/tasks 下已建好）
- 后续任务：CI 内起 PG/Redis 服务，让 DB 用例在 CI 真跑（本机无 docker，未盲改 .cnb.yml）
- 未覆盖：IMAP/POP3 拉取路径、AccountService CRUD 补强（design.md §5 已声明）


## Session 4: 剩余任务整理与交接（HANDOFF）
<!-- trellis-session: v=2 fp=f1d57c67dc0ab20c -->

**Date**: 2026-10-28
**Task**: 剩余任务整理与交接（HANDOFF）
**Branch**: `main`

### Summary

把任务板上「剩下的活」整理成一份零上下文可接手的交接文档 .trellis/HANDOFF.md：2.0 主线剩余 p20（前端缓存语义统一）/p21（swagger 注解补全）；6 个历史遗留任务经实测确认工作已完成但状态没更新（08-14-public-email-apis、00-bootstrap-guidelines、06-18-p2-api-contract、06-18-p3-cleanup、06-19-p0-observability-shutdown、06-18-backend-arch-standardization），其中只剩 2 个真实残余项（config→crypto 常量、SSE 主动关闭）。同时归档 2 个已完成未归档任务，任务板 42 → 40。

### Main Changes

- 新增 .trellis/HANDOFF.md：剩余任务清单（含每项的 PRD 要点、前置决策、涉及文件、验收标准）+ 环境约束 + 新会话开场提示词
- 实测核对 6 个历史遗留任务：8 个 public/mail 端点与跨账户 404 校验已实现、spec 三文件已 Filled、handler 无 repository.EmailFilter{}、swagger 无 gorm.DeletedAt、adapter/manager.go 与 tests/ 与 internal/webhook 均已消失、LOG_FORMAT=json + /metrics + 30s 优雅关闭均已实现
- 记录两个真实残余项：config/config.go:64 仍依赖 crypto.DefaultEncryptionKey；internal/sse/hub.go 无 Close（SSE 长连接靠 srv.Shutdown 等 30s 超时）
- 记录实测坑��task.py archive <父任务> 不递归归档子任务，只清掉子任务 parent 字段（task_store.py:1363 附近），附批量归档 p0-p18 的命令
- 归档 08-07-inbox-optimization、08-21-fix-log-memory-crash（均 completed 但未归档）

### Git Commits

| Hash | Message |
|------|---------|
| `175b1da` | docs(trellis): 新增剩余任务交接文档 HANDOFF.md |
| `05f4652` | docs(trellis): HANDOFF 补充 2.0 子任务归档顺序说明 |

### Testing

- [OK] 核对方式为源码/产物实测（grep + 读 swagger.json + 读 cmd/server/main.go），非推测
- [OK] 整理后工作区干净，任务板仅剩 00-bootstrap-guidelines、06-18 系列 3 项、08-14-public-email-apis、p20、p21

### Status

[OK] **Completed**

### Next Steps

- 新会话：读 .trellis/HANDOFF.md，按 §1 做 P20（先问持久化口径）
- 或先按 §3 收尾 6 个历史遗留任务（补 2 个残余项后归档），再开始 P21


## Session 5: P20 前端缓存语义统一
<!-- trellis-session: v=2 fp=cc53f5631d344def -->

**Date**: 2026-10-29
**Task**: P20 前端缓存语义统一
**Branch**: `main`

### Summary

把 5 套缓存语义收敛到 cachePolicy 单一策略源与 cacheInvalidation 单一失效入口；修复 mutation 给 TTL 续命、敏感设置整分类落盘、SettingsPage 旁路持久化等 5 个缺陷

### Main Changes

- utils/cachePolicy.ts 扩为唯一策略源；新增 utils/cacheInvalidation.ts；settingsCache 改按 key 白名单；accountStore/groupStore 修缓存时钟语义

### Git Commits

| Hash | Message |
|------|---------|
| `2959e51` | refactor(frontend): 统一缓存语义为单一策略源与失效入口 |

### Testing

- [OK] tsc PASS；vitest 11 文件 67 测试全绿（基线 5 文件 24）；vite build PASS

### Status

[OK] **Completed**

### Next Steps

- P21 swagger 注解质量补全；可选：删除 DEV-only 的 pages/SettingsPage.tsx
