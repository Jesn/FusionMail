# P3 Cleanup

## Goal

清理架构标准化过程中剩余的低风险历史遗留项，让代码库目录、命名和无用代码更一致。

P3 是收尾阶段，主要处理不会显著改变核心业务行为的整理工作。

## Background

前序分析发现若干低优先级但影响长期维护的遗留问题：

- `backend/internal/adapter/manager.go` 目前只被测试使用，生产路径未接入。
- `backend/test/` 和 `backend/tests/` 并存，测试目录命名混乱。
- `config/config.go` 依赖 `pkg/crypto` 的默认密钥常量，配置层边界不够干净。
- `internal/webhook` 与 `internal/adapter` 概念命名接近，容易混淆主动拉取适配器和 webhook 接收适配器。

## Scope

### Included

1. 判断并处理 `AdapterManager`：接入或删除。
2. 统一后端测试目录命名。
3. 调整 `config` 与 `pkg/crypto` 的默认常量依赖。
4. 对 webhook 接收相关包做命名消歧。
5. 清理已被 P0/P1/P2 替代的旧接口、旧注释和死参数。

### Excluded

- 不再做大规模业务重构。
- 不改变 API 契约；P2 已处理。
- 不改变同步核心行为；P0/P1 已处理。
- 不删除未确认用途的代码。

## Requirements

### R1: AdapterManager 必须有明确状态

- 如果接入：说明它在连接复用和生命周期管理中的职责，并有调用路径。
- 如果删除：删除 `manager.go` 和只服务于它的测试，确保没有生产引用。
- 不允许保留“看起来重要但无人使用”的大段死代码。

### R2: 测试目录统一

- 保留一种后端测试目录约定，推荐 `backend/test/`。
- `backend/tests/` 中仍有价值的内容迁移到统一目录。
- 根目录 `tests/e2e/` 保持独立，因为它是端到端测试体系。

### R3: 配置层边界清晰

- `config` 不直接依赖 `pkg/crypto` 仅为读取默认值。
- 默认密钥常量归属清晰，release 模式校验仍保持严格。

### R4: Webhook 命名消歧

- 明确区分：
  - 主动拉取/发送协议适配器：`internal/adapter`
  - 被动接收外部推送：建议 `internal/receiver` 或 `internal/webhook/receiver`
- 如涉及目录改名，必须一次性更新所有 import。

## Acceptance Criteria

- [ ] `AdapterManager` 被接入或删除，状态明确。
- [ ] 后端测试目录不再同时存在 `test` 和 `tests` 两套入口，或至少文档明确职责区别。
- [ ] `config` 与 `pkg/crypto` 的默认常量依赖被解耦。
- [ ] webhook 接收相关命名更清晰，import 更新完整。
- [ ] 清理项不改变核心业务行为。
- [ ] `go test ./...` 可运行。

## Validation

- `cd backend && go test ./...`
- 搜索确认 `NewAdapterManager` 是否只存在于保留路径。
- 搜索确认旧测试目录是否已清理或职责明确。
- 搜索确认 webhook 包改名后的 import 无遗漏。

## Notes

P3 必须在 P0/P1/P2 后执行，避免和核心重构产生冲突。