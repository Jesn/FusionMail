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
