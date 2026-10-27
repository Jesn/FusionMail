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
