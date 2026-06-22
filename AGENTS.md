<!-- TRELLIS:START -->
# Trellis Instructions

These instructions are for AI assistants working in this project.

This project is managed by Trellis. The working knowledge you need lives under `.trellis/`:

- `.trellis/workflow.md` — development phases, when to create tasks, skill routing
- `.trellis/spec/` — package- and layer-scoped coding guidelines (read before writing code in a given layer)
- `.trellis/workspace/` — per-developer journals and session traces
- `.trellis/tasks/` — active and archived tasks (PRDs, research, jsonl context)

If a Trellis command is available on your platform (e.g. `/trellis:finish-work`, `/trellis:continue`), prefer it over manual steps. Not every platform exposes every command.

If you're using Codex or another agent-capable tool, additional project-scoped helpers may live in:
- `.agents/skills/` — reusable Trellis skills
- `.codex/agents/` — optional custom subagents

Managed by Trellis. Edits outside this block are preserved; edits inside may be overwritten by a future `trellis update`.

<!-- TRELLIS:END -->

# Deployment Guide

## Fly.io 部署

### 基本信息
- **应用名**: `fusionmail`
- **域名**: `fusionmail.fly.dev`（自定义域名: `fusionmail.100420.xyz`）
- **区域**: `sin`（新加坡）
- **Dockerfile**: 项目根目录 `Dockerfile`，单镜像包含前端 + 后端
- **Go 构建镜像**: `golang:1.25-alpine`（go.mod 要求 Go >= 1.25.0）
- **端口**: 3333

### 发布命令
```bash
flyctl deploy --app fusionmail
```
部署后验证：
```bash
curl -s https://fusionmail.fly.dev/api/v1/health   # liveness
curl -s https://fusionmail.fly.dev/api/v1/ready     # readiness (DB + Redis)
curl -s https://fusionmail.fly.dev/metrics | head -5 # Prometheus
flyctl status --app fusionmail                       # 机器状态
```

### 数据库 Migration
- **release 模式默认不执行 AutoMigrate**（`ENABLE_AUTO_MIGRATE` 默认 false）
- 启动时执行 `ensureStartupSchemaReady()` 校验必需表是否存在，缺失则拒绝启动
- 如有新 SQL migration 文件（`backend/migrations/`），部署前先执行：
  ```bash
  flyctl ssh console -c "./migrate up"
  ```

### Secrets 管理
```bash
flyctl secrets set ENCRYPTION_KEY=<key> JWT_SECRET=<secret>  # 设置
flyctl secrets list --app fusionmail                           # 查看
flyctl secrets unset ENCRYPTION_KEY_PREVIOUS                   # 删除
```

### 可选功能环境变量
| 变量 | 用途 | 默认值 |
|------|------|--------|
| `LOG_FORMAT=json` | JSON 结构化日志 | text |
| `OTEL_ENABLED=true` | 启用 OpenTelemetry 追踪 | false |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector 地址（启用 OTel 时必填） | - |
| `OTEL_SERVICE_NAME` | 追踪服务名 | fusionmail |
| `JWT_PREVIOUS_SECRET` | JWT secret 轮换过渡期旧密钥 | - |
| `ENABLE_AUTO_MIGRATE=true` | release 模式强制执行 AutoMigrate | false |
| `ENABLE_STARTUP_SEED=true` | release 模式执行 seed 数据 | false |

### 发布前检查清单
1. `go build ./...` 通过
2. `go test ./...` 通过
3. 如果有新的 SQL migration 文件，先在 Fly 上执行 `./migrate up`
4. `flyctl deploy --app fusionmail`
5. 验证 `/api/v1/health` 和 `/api/v1/ready` 返回 200
