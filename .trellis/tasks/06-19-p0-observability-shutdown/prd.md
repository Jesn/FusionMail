# P0: Observability and Graceful Shutdown

## Requirements

### 1. Structured JSON Logging
- `pkg/logger` 支持 `LOG_FORMAT=json` 环境变量切换 JSON 输出
- JSON 格式包含: timestamp, level, module, message, fields
- 默认保持文本格式（开发环境），release 模式默认 JSON
- 不引入新依赖，手动实现 JSON 序列化

### 2. Prometheus Metrics Endpoint
- `GET /metrics` 端点，无认证，暴露 Prometheus 格式指标
- 核心指标: HTTP 请求计数/延迟、DB 连接池使用、goroutine 数量
- 用 `prometheus/client_golang` 库

### 3. Graceful Shutdown with Context Propagation
- `container.go` 创建全局 `context.Context`，shutdown 时 cancel
- 所有 `go func()` 后台任务使用全局 context 的派生 context
- shutdown 时等待后台任务完成或超时（30s）
- SSE 连接在 shutdown 时主动关闭

## Acceptance Criteria
- `LOG_FORMAT=json` 时日志输出为合法 JSON
- `/metrics` 返回 Prometheus 格式文本，包含 http_requests_total、http_request_duration_seconds、go_goroutines
- shutdown 时后台 goroutine 收到 context cancellation
- `go build ./...` + `go test ./...` 通过