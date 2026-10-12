# 执行计划：日志系统内存崩溃修复

## 实现顺序

### Step 1: stripAnsiCodes 正则提升（最低风险，先做）

- [ ] 文件：`backend/internal/handler/log_handler.go`
- [ ] 将 `stripAnsiCodes` 函数内的 `regexp.MustCompile` 提升为包级 `var`
- [ ] 同时检查 `readAndFilterLogs` 内的 `logPattern` 是否也应提升为包级
- [ ] 验证：`go build ./...`

### Step 2: 降低 DefaultAppLogMaxSizeMB

- [ ] 文件：`backend/pkg/logger/logger.go`
- [ ] `DefaultAppLogMaxSizeMB` 从 50 改为 10
- [ ] 验证：`go build ./...`

### Step 3: CleanupRotatedAppLogs 增加主文件滚动触发

- [ ] 文件：`backend/pkg/logger/logger.go`
- [ ] 在 `CleanupRotatedAppLogs` 函数开头，检查 `fileOutput` 主文件大小
- [ ] 若超过 `DefaultAppLogMaxSizeMB`，调用 `fileOutput.Rotate()` 强制滚动
- [ ] 验证：`go build ./...` + `go test ./pkg/logger/...`

### Step 4: readAndFilterLogs 流式重构（核心）

- [ ] 文件：`backend/internal/handler/log_handler.go`
- [ ] 重构 `readAndFilterLogs` 为流式扫描方案：
  - 获取文件大小，超过安全阈值（5MB）时降级为 tail 模式
  - 安全范围内：正向流式扫描，计数+过滤（不存储），记录匹配条目的文件 offset
  - 第二遍 Seek 到目标页 offset 范围，只解析该页的行
  - 退路：若实现复杂度过高，改用正向流式 + ring buffer（只保留最近 page*page_size 条）
- [ ] 确保 `GetLogs` 响应格式不变
- [ ] 验证：`go build ./...` + `go test ./internal/handler/...`

### Step 5: GetLogTail 反向读取重构

- [ ] 文件：`backend/internal/handler/log_handler.go`
- [ ] 重构 `GetLogTail` 为从文件末尾反向 Seek 读取
- [ ] 按 chunk（64KB）反向扫描 `\n` 行边界
- [ ] 收集到指定行数后停止
- [ ] 确保 `GetLogTail` 响应格式不变
- [ ] 验证：`go build ./...`

### Step 6: GetLogStats 流式优化（可选）

- [ ] 文件：`backend/internal/handler/log_handler.go`
- [ ] `GetLogStats` 当前不存储全量但读全文件，内存风险较低但耗时
- [ ] 若 Step 4 引入了文件大小检查，stats 也应复用同样的阈值保护
- [ ] 验证：`go build ./...`

### Step 7: 前端 LogsPage 加载策略调整

- [ ] 文件：`frontend/src/pages/LogsPage.tsx`
- [ ] 将 `loadLogs()` 和 `loadStats()` 改为串行：先 logs 后 stats
- [ ] 验证：前端编译通过

### Step 8: 前端设置页文案修正

- [ ] 文件：`frontend/src/pages/SystemSettingsPage.tsx`
- [ ] 修正"应用运行日志保留天数"说明文案
- [ ] 明确区分主文件体积滚动阈值和备份保留天数
- [ ] 验证：前端编译通过

## 验证命令

```bash
# 后端编译
cd backend && go build ./...

# 后端测试
cd backend && go test ./...

# 前端编译
cd frontend && npm run build

# 内存验证（手动）：生成大日志文件测试
# 在本地启动后端，写入 50MB 测试日志，请求 /api/v1/logs 观察内存
```

## Review Gates

- Step 3 完成后：审查 lumberjack Rotate 并发安全性
- Step 4 完成后：审查内存安全性，确认无全量加载路径
- Step 5 完成后：审查 tail 边界处理（空文件、行数超过文件行数、跨 chunk 行边界）

## Rollback Points

- 每个 Step 独立可回滚
- Step 4 是核心变更，若流式实现有严重问题，可退回原实现 + 文件大小保护（超过 5MB 拒绝请求并返回提示）