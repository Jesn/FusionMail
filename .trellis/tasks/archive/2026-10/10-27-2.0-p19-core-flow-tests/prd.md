# 2.0 后续 A：后端核心流程测试补强

## Goal

给 service 层核心业务流程补回归测试，覆盖率从 ~10% 提升到"重构有保护网"的水平。
这是 2.0 重构收官后最值得投入的方向：已完成的拆分/契约收紧目前主要靠
`-race` 全绿 + 守卫测试保护，业务逻辑回归保护薄。

## 基线（2026-10-27 实测，`go test ./... -cover`）

| 包 | 覆盖率 |
|---|---|
| internal/service | **10.2%** |
| internal/handler | 6.9% |
| internal/repository | 11.5% |
| internal/service/spam | 10.3% |
| 参照高位：adapter/webapi/cloudmail 67.9%、pkg/crypto 64.1% |

## Requirements

- 优先覆盖三条核心流程（按业务风险排序）：
  1. **发信链路**：send_service（含 reply/forward/附件/凭据解密）→ send_handler
  2. **同步链路**：sync_scheduler / sync_executor / uid / batch（P9 拆分产物）
  3. **垃圾判定链路**：spam_service（rules/detection/stats/bayesian 四文件）+
     spam/bayesian_classifier
- 其次：cleanup_service（清理任务）、account_service（CRUD+组合器）
- 测试放同包 `*_test.go`；用例走真实 PostgreSQL（`TEST_DATABASE_URL` 或 `DB_*` 拼装，
  每包独立库 `fm_test_*`；缺 PG 环境时 skip 而非 fail），Redis 依赖用 miniredis 替身
  —— 详见 `design.md` D1/D2 与被拒方案
- 不为凑覆盖率写快照式断言；断言业务语义（状态流转、副作用、错误路径）

## Acceptance Criteria

- [ ] internal/service 覆盖率 ≥ 30%（`go test ./internal/service/ -cover`）
- [ ] internal/service/spam 覆盖率 ≥ 30%
- [ ] 每条核心流程至少含：正常路径 1 + 错误路径 1 + 边界 1
- [ ] `go test ./... -race -count=1` 全绿（22 包）
- [ ] 覆盖率数据记入本文件验收记录

## Notes

- 并行测试注意 PostgreSQL 测试库冲突：每包使用独立库 `fm_test_<pkg>`（见 design D1），
  不碰 `fm_automigrate`/`fm_baseline`；全量 `-race` 由主会话统一跑一次
- 已知缺口：CI（`.cnb.yml`）无 PG/Redis，本任务的 PG 测试在 CI 会跳过；
  「CI 内起 PG 服务」作为后续任务输入，记入 `design.md` §5
- 测试环境变量：DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=fusionmail
  DB_PASSWORD=fusionmail_test_pw DB_SSLMODE=disable
  ENCRYPTION_KEY="test-2fa-secret-key-32-bytes-ok!"
  JWT_SECRET="12345678901234567890123456789012"
