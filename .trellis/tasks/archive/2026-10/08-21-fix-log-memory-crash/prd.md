# 日志系统内存崩溃修复

## Goal

修复 `/logs` 页面打开 45MB 日志文件导致 Fly.io 机器 OOM 崩溃的问题，同时修正日志保留策略无法清理主文件的设计缺陷。

## Background

线上环境 `fusionmail.fly.dev` 打开 `/logs` 页面时，后端 `GET /api/v1/logs` 会将整个 `backend.log` 文件读入内存、逐行解析为结构体、全量反转后再分页。45MB 文件（约 22.5 万行）导致内存峰值超过 200MB，叠加应用基线内存后触发 Fly.io 机器 OOM Kill。

同时，用户在设置页配置的"同步日志保留天数"清理的是数据库 `sync_logs` 表，与文件日志 `backend.log` 无关；即使正确配置了"应用运行日志保留天数"，主文件也因 `CleanupRotatedAppLogs` 显式跳过而无法被清理，只能靠 lumberjack 在 50MB 时被动滚动。

## Requirements

### R1: 日志读取内存安全（后端）

- `GET /api/v1/logs` 不得将整个日志文件加载到内存
- 分页必须基于流式或反向读取，内存占用与 `page_size` 成正比，与文件大小无关
- 大文件（超过安全阈值）时自动降级为 tail 模式或返回提示
- `GET /api/v1/logs/tail` 不得全量读入文件，必须从文件末尾反向读取

### R2: 正则编译优化（后端）

- `stripAnsiCodes` 不得在每次调用时重新编译正则
- 正则表达式必须提升为包级变量，只编译一次

### R3: 日志主文件清理能力（后端）

- 保留天数策略不仅要清理滚动备份，还要能处理主文件 `backend.log`
- 主文件过大时应触发滚动或截断，不能等到 50MB 才被动处理
- 降低 `DefaultAppLogMaxSizeMB` 默认值，让主文件更早滚动

### R4: 前端加载策略（前端）

- `/logs` 页面不得同时触发 stats 和 logs 两个全文件读取请求
- 设置页"应用运行日志保留天数"的说明文案需明确区分主文件滚动阈值和备份清理

### R5: 不破坏现有 API 契约

- `GET /api/v1/logs` 的请求/响应格式不变（`items`、`total`、`page`、`page_size`）
- `GET /api/v1/logs/tail` 的请求/响应格式不变（`lines`、`total`）
- `GET /api/v1/logs/stats` 的请求/响应格式不变
- 分页行为：最新日志在前，旧日志在后

## Acceptance Criteria

- [ ] AC1: 用 50MB 测试日志文件请求 `GET /api/v1/logs?page=1&page_size=50`，峰值内存不超过 20MB
- [ ] AC2: `GET /api/v1/logs/tail?lines=100` 对 50MB 文件峰值内存不超过 5MB
- [ ] AC3: `stripAnsiCodes` 的正则表达式在包级只编译一次（代码审查确认）
- [ ] AC4: `DefaultAppLogMaxSizeMB` 从 50 降为 10 或更低
- [ ] AC5: `CleanupRotatedAppLogs` 执行后，主文件 `backend.log` 超过新阈值时被滚动或截断
- [ ] AC6: `/logs` 页面初始加载不再同时发起 stats + logs 两个全文件读取请求
- [ ] AC7: 设置页"应用运行日志保留天数"说明文案准确描述其作用范围
- [ ] AC8: `go build ./...` 通过
- [ ] AC9: `go test ./...` 通过
- [ ] AC10: 现有日志相关测试不回归

## Out of Scope

- 日志级别动态调整功能
- 日志搜索性能优化（关键词搜索仍为全量扫描，本次只防 OOM）
- 前端日志展示 UI 重构
- 日志存储迁移到数据库