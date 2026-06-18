# Backend Architecture Findings

## Evidence Summary

This research note captures the architecture findings that motivated the backend standardization task. It is intended to make future implementation sessions independent from chat history.

## File Size Hotspots

Observed non-test Go file line-count hotspots:

- `backend/internal/service/sync_service.go`: 1828 lines.
- `backend/internal/adapter/imap.go`: 1423 lines.
- `backend/internal/service/webapi_provider_service.go`: 1392 lines.
- `backend/internal/adapter/graph_quick.go`: 1341 lines.
- `backend/internal/service/account_service.go`: 959 lines.
- `backend/internal/service/email_service.go`: 953 lines.
- `backend/internal/service/oauth2_service.go`: 946 lines.
- `backend/internal/service/spam/spam_detector.go`: 846 lines.
- `backend/pkg/database/database.go`: 712 lines.
- `backend/cmd/server/main.go`: 692 lines.
- `backend/internal/router/router.go`: 606 lines.

Generated files such as `backend/docs/docs.go` are excluded from manual split targets.

## P0 Evidence

### Batch sync counter risk

`backend/internal/service/sync_service.go` batch flow uses batch return values as deltas:

```go
batchNew, batchUpdated, batchFailed := s.processBatchEmails(ctx, account.UID, emails, syncLog)
totalNew += batchNew
totalUpdated += batchUpdated
totalFailed += batchFailed
```

But `processBatchEmails` returns cumulative values from `syncLog`:

```go
newCount = int(syncLog.EmailsNew)
updatedCount = int(syncLog.EmailsUpdated)
return
```

This can over-count when more than one batch is processed.

### DI bypass in sync rule application

`backend/internal/service/sync_service.go` currently contains:

```go
func (s *syncService) applyRulesForEmail(ctx context.Context, email *model.Email) error {
    // 临时构建 ruleService（避免改动更大范围的依赖注入）
    ruleRepo := repository.NewRuleRepository(database.GetDB())
    rs := NewRuleService(ruleRepo, s.emailRepo)
    return rs.ApplyRules(ctx, email)
}
```

This is an explicit dependency injection bypass and should be replaced with constructor injection.

### Credential parsing duplication

Credential/account adapter configuration logic appears in at least:

- `backend/internal/service/sync_service.go` (`parseCredentials`).
- `backend/internal/service/email_service.go` (`tryServerSoftDelete`).
- `backend/internal/service/email_service.go` (`tryRepairEmailBody`).
- `backend/internal/service/sync_manager.go` (`TestAccountConnection`).

Shared behavior includes auth type branching, OAuth2/quick/password handling, protocol host/port/encryption setup, and `mail.linuxdo.org` compatibility repair.

### SSE coupling

Direct `sse.Broadcast` calls appear in service-level files, including:

- `backend/internal/service/sync_service.go`.
- `backend/internal/service/webhook_receiver_service.go`.
- `backend/internal/service/progress_tracker.go`.

Service layer should depend on a notification port/interface instead of directly knowing the SSE implementation.

## P1 Evidence

### main.go responsibility overload

`backend/cmd/server/main.go` mixes:

- env loading;
- config validation;
- database init and schema/seed handling;
- repository/service/handler construction;
- runtime monitor startup;
- sync manager and cleanup service startup;
- router setup;
- webhook registry;
- Swagger/pprof/static file routes;
- HTTP server lifecycle.

### Router parameter explosion

`backend/internal/router/router.go` `SetupRouter` accepts a large number of handler/config/runtime dependencies and includes a dead `swaggerEnabled` parameter that is only assigned to `_`.

### database.go mixed responsibility

`backend/pkg/database/database.go` includes infrastructure connection logic plus provider/adapter/settings seed data and repair helpers. Business seed data should move under `internal/seed`.

## P2 Evidence

### Model/API contract leakage

Handler/service paths return GORM model types directly in several places. Swagger has exposed `gorm.DeletedAt`, showing storage details have leaked into external API documentation.

### Repository filter leakage

`backend/internal/handler/email_handler.go` constructs `repository.EmailFilter` directly and passes it into service methods. The repository query object is therefore part of handler/service contract.

## P3 Evidence

### AdapterManager appears unused in production path

`NewAdapterManager` appears in:

- `backend/internal/adapter/manager.go`
- `backend/internal/adapter/manager_test.go`

No production call path was found during analysis.

### Test directory duplication

Both `backend/test/` and `backend/tests/` exist.

### Naming ambiguity

`internal/adapter` handles active protocol adapters, while `internal/webhook` also uses adapter concepts for inbound webhook providers. Naming should be clarified after core architecture work stabilizes.