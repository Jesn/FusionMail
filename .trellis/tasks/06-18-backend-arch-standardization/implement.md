# Backend Architecture Standardization Implementation Plan

## Execution Order

严格按阶段执行：

1. `06-18-p0-sync-credential-di-fix`
2. `06-18-p1-structure-split`
3. `06-18-p2-api-contract`
4. `06-18-p3-cleanup`

不要跳过 P0 直接做 P1。P0 是行为稳定性和依赖边界前置修复。

## Phase Gates

### Before starting a child task

- 阅读父任务 `prd.md`、`design.md`、`implement.md`。
- 阅读对应子任务 `prd.md`。
- 对该阶段涉及文件做最小必要复核，确认没有新的用户改动冲突。
- 如 Trellis 当前任务不是目标子任务，先 `task.py start <task>`。

### Before finishing a child task

- 运行阶段要求的最小验证。
- 记录无法运行的验证和原因。
- 确认没有跨阶段提前实现的内容。
- 更新必要的阶段说明或 follow-up。

## P0 Implementation Checklist

### P0.1 Fix batch sync counters

- Locate `doSyncWithBatch` and `processBatchEmails` in `backend/internal/service/sync_service.go`.
- Change batch result semantics to per-batch delta.
- Add or adjust tests for multiple batches.
- Verify final `syncLog.EmailsNew` / `EmailsUpdated` equals real processed counts.

Rollback point: revert only `sync_service.go` and related tests.

### P0.2 Add CredentialResolver

- Create `backend/internal/service/credential_resolver.go`.
- Move shared logic from:
  - `syncService.parseCredentials`
  - `emailService.tryServerSoftDelete`
  - `emailService.tryRepairEmailBody`
  - `SyncManager.TestAccountConnection`
- Preserve quick/OAuth2/password behavior and host compatibility fix.
- Replace callers incrementally.

Rollback point: resolver file plus touched callers.

### P0.3 Inject rule application

- Define a small interface if needed:
  ```go
  type RuleApplier interface {
      ApplyRules(ctx context.Context, email *model.Email) error
  }
  ```
- Add it to `syncService` dependencies.
- Create rule service at construction boundary.
- Remove `database.GetDB()` from `applyRulesForEmail`.

Rollback point: constructor signature and `sync_manager.go` injection path.

### P0.4 Introduce SyncNotifier

- Define notifier interface.
- Provide SSE-backed implementation.
- Replace service-level direct `sse.Broadcast` where low risk.
- Keep handler-level direct broadcast if changing it creates unnecessary scope.

Validation:

- `cd backend && go test ./...`
- Search checks:
  - `database.GetDB()` in service package should not increase.
  - Direct `sse.Broadcast` in service package should decrease.

## P1 Implementation Checklist

### P1.1 Extract server container

- Create `backend/cmd/server/container.go`.
- Move repository/service/handler creation from `main.go` into typed groups.
- Keep runtime start/stop in `main.go`.
- Avoid changing constructor behavior during move.

### P1.2 Split sync service files

- Move methods without changing package or exported signatures.
- Keep `syncService` struct in `sync_service.go`.
- Split by responsibility:
  - scheduler
  - executor
  - processor
  - error policy
  - webapi
- Run tests after each large move if possible.

### P1.3 Extract seed package

- Create `backend/internal/seed`.
- Move provider/adapter/settings seed and repair functions.
- Update callers from `database.SeedInitialData()` to new seed entry.
- Keep `pkg/database` focused on DB initialization.

### P1.4 RouterDeps

- Add `RouterDeps` and grouped handler structs.
- Change `SetupRouter` signature.
- Update call site in container/main.
- Remove `swaggerEnabled` dead parameter.

### P1.5 Split high-risk large files

Prioritize in order:

1. `email_service.go`: deletion and repair helpers.
2. `account_service.go`: sync/status-related helpers.
3. `webapi_provider_service.go`: test/sync/config segments.
4. `oauth2_service.go`: provider-specific handlers/helpers.
5. `spam_detector.go`: pipeline and result aggregation.

Validation:

- `cd backend && go test ./...`
- Check no import cycles.
- Review file sizes for core service files.

## P2 Implementation Checklist

### P2.1 DTO for core responses

- Add response DTOs under `internal/dto/response` or nearby existing pattern.
- Email detail no longer returns raw `model.Email`.
- Account detail/list no longer returns raw `model.EmailAccount` with internal fields.
- Keep JSON field names compatible.

### P2.2 Service query inputs

- Add `EmailQueryParams` or equivalent in service layer.
- Handler parses HTTP inputs into request/service input type.
- Service converts to repository filter.
- Move default deleted/spam filtering to service.

### P2.3 Repository interface split

- Introduce small interfaces by consumer need.
- Keep concrete implementation unchanged where possible.
- Update service constructors to depend on smaller interfaces.

Validation:

- `cd backend && go test ./...`
- Swagger/core API shape check.
- Frontend service field compatibility spot check.

## P3 Implementation Checklist

### P3.1 AdapterManager decision

- Search all production references.
- If unused, remove `manager.go` and `manager_test.go`.
- If kept, document and wire it into actual adapter lifecycle.

### P3.2 Test directory normalization

- Inventory `backend/test` and `backend/tests`.
- Merge useful files into chosen structure.
- Update docs/scripts if paths are referenced.

### P3.3 Config/crypto boundary

- Move default key ownership or pass default explicitly.
- Preserve release-mode secret validation.

### P3.4 Webhook naming cleanup

- Rename package only if import churn is contained.
- Update all imports atomically.
- Run full tests.

Validation:

- `cd backend && go test ./...`
- Search for stale paths/imports.

## Global Validation Commands

Preferred after each phase:

```bash
cd backend && go test ./...
```

If faster scoped checks are needed during development:

```bash
cd backend && go test ./internal/service ./internal/repository ./internal/router ./cmd/server
```

For structural checks:

```bash
cd backend && go test ./...
```

## Rollback Strategy

- P0 rollback is behavior-level: revert touched sync/credential/notifier files and tests.
- P1 rollback is structure-level: revert file moves in one phase chunk; avoid mixing behavior changes.
- P2 rollback is API-level: keep DTO additions isolated so old model responses can be restored if needed.
- P3 rollback is cleanup-level: each cleanup item should be independently revertible.

## Review Strategy

Each phase should be reviewed with a different focus:

- P0: correctness and behavior preservation.
- P1: module boundaries and import stability.
- P2: API compatibility and frontend impact.
- P3: absence of accidental behavior changes.

## Completion Criteria

Parent task is complete only after all child tasks are completed or explicitly cancelled with rationale. Do not archive the parent while any child task remains active or pending.