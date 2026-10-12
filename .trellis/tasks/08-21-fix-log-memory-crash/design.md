# 技术设计：日志系统内存崩溃修复

## 涉及文件

| 文件 | 层 | 变更类型 |
|------|----|----------|
| `backend/pkg/logger/logger.go` | pkg/logger | 常量修改 + 新增函数 |
| `backend/internal/handler/log_handler.go` | handler | 核心重构 |
| `frontend/src/pages/LogsPage.tsx` | frontend | 加载策略调整 |
| `frontend/src/pages/SystemSettingsPage.tsx` | frontend | 文案修正 |

## D1: readAndFilterLogs 反向流式读取

### 问题

当前实现：`os.Open` → `bufio.Scanner` 逐行读入 → 全量解析为 `[]LogEntry` → 全量过滤 → in-place 反转 → slice 分页。内存 O(n)，n 为文件总行数。

### 设计

改为**反向读取 + 计数跳过 + 边读边过滤**策略：

1. `os.Open` + `file.Stat()` 获取文件大小
2. 若文件大小超过安全阈值（如 5MB），降级为 tail 模式：从末尾读取最后 N 字节，解析行边界，只处理这部分
3. 若文件大小在安全范围内（≤5MB），仍可全量读取但改用预分配 cap 的 slice
4. 正常路径：从文件末尾反向 `Seek` + 分块读取（如 64KB chunk），在 chunk 内按 `\n` 切分行，逐行解析+过滤+计数，达到目标页范围时收集到结果 slice
5. `total` 仍需全量计数：对于大文件，total 改为估算或标注"超过 N 行"

### 替代方案考量

- **方案 A：反向 chunk 读取** — 复杂度中等，内存最优，但需要处理跨 chunk 行边界、UTF-8 安全截断
- **方案 B：正向流式 + ring buffer 保留最近 N 行** — 实现简单，但过滤+分页时 total 需全量扫描
- **方案 C：正向流式 + 双 pass** — 第一 pass 只计数，第二 pass 跳过+收集。total 准确但需读两次

**选择方案 C 简化版**：正向流式扫描，第一遍计数+过滤（不存储），记录匹配条目的文件 offset；第二遍 Seek 到目标页的 offset 范围，只解析该页的行。内存 O(page_size)，total 准确。

> 如果实现复杂度过高，退化为方案 B：正向流式扫描，只保留最近 `page * page_size` 条匹配结果，超出后丢弃旧的。最后一页可能不完整，但可接受。

### 接口不变

响应 JSON 结构不变：
```json
{
  "items": [LogEntry],
  "total": int,
  "page": int,
  "page_size": int
}
```

## D2: GetLogTail 反向读取

### 问题

当前：全量读入 `[]string allLines`，取尾部。内存 O(n)。

### 设计

从文件末尾 `Seek` 反向读取，按块扫描 `\n` 行边界，收集到指定行数后停止。内存 O(lines * max_line_len)。

实现要点：
1. `file.Stat()` 获取文件大小
2. `file.Seek(size - chunkSize, io.SeekStart)` 反向定位
3. 读取 chunk，按 `\n` 切分，从后往前收集行
4. 不足指定行数时继续向前 Seek
5. 到达文件开头时停止

## D3: stripAnsiCodes 正则提升

### 问题

```go
func stripAnsiCodes(s string) string {
    ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`) // 每次调用都编译
    return ansiPattern.ReplaceAllString(s, "")
}
```

### 设计

提升为包级 `var`：
```go
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsiCodes(s string) string {
    return ansiPattern.ReplaceAllString(s, "")
}
```

同时检查 `parseLine` 中的 `logPattern` 是否已为包级（当前在函数内编译，每次调用 `readAndFilterLogs` 编译一次——可接受但建议一并提升）。

## D4: 日志主文件清理

### 问题

`CleanupRotatedAppLogs` 显式跳过 `backend.log` 主文件（`logger.go:198-200`）。主文件只能靠 lumberjack 在 `DefaultAppLogMaxSizeMB=50` 时被动滚动。

### 设计

两处修改：

1. **降低 `DefaultAppLogMaxSizeMB`**：从 50 改为 10。让主文件更早滚动，减少单文件体积。

2. **`CleanupRotatedAppLogs` 增加主文件滚动触发**：在清理备份前，检查主文件大小。若主文件已超过 `DefaultAppLogMaxSizeMB`，调用 lumberjack 的 `Rotate()` 强制滚动，使其变为备份文件，随后被保留天数策略清理。

```go
// 在 CleanupRotatedAppLogs 扫描前：
fileOutputMu.Lock()
lj := fileOutput
fileOutputMu.Unlock()
if lj != nil {
    if fi, err := os.Stat(lj.Filename); err == nil {
        if fi.Size() >= int64(DefaultAppLogMaxSizeMB) * 1024 * 1024 {
            _ = lj.Rotate() // 强制滚动，主文件变为备份
        }
    }
}
```

### 兼容性

- lumberjack `Rotate()` 会将当前 `backend.log` 重命名为带时间戳的备份，并创建新的空 `backend.log`
- 新备份会被后续的保留天数扫描覆盖
- 不影响正在写入的 logger（lumberjack 内部处理文件句柄切换）

## D5: 前端加载策略

### 问题

`LogsPage.tsx` 的 `useEffect` 同时调用 `loadLogs()` 和 `loadStats()`，两个请求各自打开并扫描整个日志文件。

### 设计

改为串行：先加载 `loadLogs()`，完成后再加载 `loadStats()`。或者更简单——初始加载时跳过 stats，仅在用户点击刷新或切换文件时才加载 stats。

```tsx
// 改为：
useEffect(() => {
    loadLogs().then(() => loadStats());
}, [loadLogs, loadStats]);
```

### D6: 设置页文案修正

### 问题

当前文案"对应「系统日志」页的 backend.log；单文件超过 50MB 会自动滚动，超过保留天数的备份会被删除"容易让用户误以为保留天数能控制主文件大小。

### 设计

修正为：
> 对应「系统日志」页的 backend.log 文件。主文件超过 {MaxSize}MB 时自动滚动为备份，超过保留天数的**备份文件**会被删除。主文件本身不因天数清理，只受体积阈值控制。

## 数据流

```
修改前：
  前端 /logs → GET /logs → readAndFilterLogs: 全量读取+解析+存储+反转+分页 → OOM
  定时清理 → CleanupRotatedAppLogs: 跳过主文件 → 主文件持续增长

修改后：
  前端 /logs → GET /logs → readAndFilterLogs: 流式扫描+计数+目标页offset定位+只解析目标页 → 内存安全
  定时清理 → CleanupRotatedAppLogs: 先检查主文件大小→必要时Rotate→再清理备份 → 主文件受控
```

## 风险

| 风险 | 缓解 |
|------|------|
| 反向读取跨 chunk 行边界处理复杂 | 退化为方案 B（正向流式 + ring buffer），实现简单 |
| `total` 对大文件不准确 | 标注"超过 N 行"或接受精度损失 |
| lumberjack `Rotate()` 并发安全 | lumberjack 内部有 mutex，且 `fileOutputMu` 保护句柄引用 |
| 降低 MaxSizeMB 后日志滚动更频繁 | 可接受，备份有保留天数控制 |

## Rollback

- `DefaultAppLogMaxSizeMB` 改回 50
- `readAndFilterLogs` 恢复原实现（git revert）
- 前端改动独立，可单独回滚