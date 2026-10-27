# 2.0 后续 A：核心流程测试补强 — 执行计划

> 依赖 `design.md` 的 D1–D6。每步结束都要能独立跑绿，失败即停、不上下一步。

## 环境（每条命令前 export）

```bash
cd backend
export DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=fusionmail DB_PASSWORD=fusionmail_test_pw DB_SSLMODE=disable
export ENCRYPTION_KEY="test-2fa-secret-key-32-bytes-ok!" JWT_SECRET="12345678901234567890123456789012"
```

## 步骤 0：测试夹具冒烟（先做，失败则改方案）〔必须〕

产出 `backend/internal/testutil/pgtest/pgtest.go`：

- `Open(tb)`：建库 `fm_test_<pkg>`（`DROP`+`CREATE`，admin 连接落到 `postgres` 库）→ 打开 gorm postgres → `database.DB = db` → `database.AutoMigrate()` → 注册 `Cleanup`。
- `Truncate(tb)`：查 `information_schema.tables`（`public`）→ `TRUNCATE ... RESTART IDENTITY CASCADE`。
- `SkipIfUnavailable(tb)`：DSN 为空或 ping 失败 → `tb.Skip`（打印原因）。
- 冒烟用例 `pgtest_test.go`：建 1 个 `model.Email` → `Truncate` → 断言 0 行。

验证：`go test ./internal/testutil/pgtest/ -count=1 -v` 全绿；再用 `DB_HOST=127.0.0.1 DB_PORT=1` 跑一次，必须 skip 而非 fail。

**Done**：PG 可达时建库+迁移成功；PG 不可达时优雅跳过。

## 步骤 1：发信链路〔必须〕

- D3 窄接口改造（`send_service.go`：新增 `mailSenderResolver`、字段与构造函数参数改类型），`go build ./...` 必须零改动通过。
- `send_service_flow_test.go`：SendEmail 正常 / 账户不存在 / 发送失败落 `failed` / 空收件人 / Bcc 不进 `ToAddresses`。
- `send_service_reply_test.go`：Reply / ReplyAll / Forward 的头部构造（`In-Reply-To`、`References`、`Re:`/`Fwd:` 前缀、ReplyAll 去重自身）+ 原邮件不存在。
- `send_service_credentials_test.go`：`oauth2` / `quick` / 空凭据 / 损坏密文 / 非法 JSON。

验证：`go test ./internal/service/ -run 'TestSend|TestReply|TestForward|TestParseCredentials' -count=1 -v`

**Done**：三条用例类型（正常/错误/边界）齐全，断言落库状态而非仅返回值。

## 步骤 2：同步链路〔必须〕

- `sync_scheduler_test.go`：用既有 `accountSyncRunner` 窄接口注入假 runner —— 并发上限、错误聚合、0 账户、`shouldSync` 间隔/首次同步、`CleanupStaleSyncLogs`。
- `sync_executor_webapi_test.go`：`webapi.WebAPIAdapterFactory` + cloudmail/custom 适配器 + `httptest` 上游；断言邮件落库、`SyncLog` 计数、进度游标、上游 500 的错误分类、重复 Message-ID 去重。Redis 用 `miniredis`。
- `sync_executor_uid_test.go`：`uidValidity` 变化 → 重置游标；空批次；落库失败计数。

验证：`go test ./internal/service/ -run 'TestSyncScheduler|TestSyncExecutor' -count=1 -v`

**Done**：调度器 6 类用例 + 执行器 WebAPI 路径正/错误/边界；日志里无 goroutine 泄漏告警（`-race` 下跑）。

## 步骤 3：垃圾判定链路〔必须〕

- `spam_service_flow_test.go`：Mark/Unmark 幂等 + reputation + bayesian 训练记录；`BatchDeleteSpam` / `EmptySpamFolder` 计数与非垃圾邮件不受影响；空 ID 列表。
- `spam_service_rules_test.go`：规则 CRUD、Toggle、`TestRule`（合法/非法正则）、`GetRuleStats`、分页边界、内置规则保护。
- `spam/bayesian_classifier_test.go`：训练后分类倾向、未训练模型、空语料/单类样本。

验证：`go test ./internal/service/spam/ ./internal/service/ -run 'TestSpam|TestBayesian|TestRule' -cover -count=1`

**Done**：spam 包覆盖率 ≥ 30%。

## 步骤 4：清理与账户（其次）〔必须〕

- `cleanup_service_test.go`：`ManualCleanup` / `ManualSpamCleanup` / `ManualLogCleanup` 正常、保留天数 0/负、无数据、`DeletedEmailKey` 孤儿清理。
- 补强 `account_service_test.go`：重复 UID、不存在、空字段校验、组合器纯转发（转发到正确子类型）。

验证：`go test ./internal/service/ -run 'TestCleanup|TestAccount' -count=1`

## 步骤 5：全量验收〔必须〕

```bash
cd backend
go build ./... && go vet ./...
go test ./... -race -count=1                      # 22 包全绿
go test ./internal/service/ -cover -count=1        # ≥ 30%
go test ./internal/service/spam/ -cover -count=1   # ≥ 30%
# 缺 PG 时的降级验证（必须 skip 而非 fail）
DB_HOST=127.0.0.1 DB_PORT=1 go test ./internal/service/ -count=1
```

- 更新 `design.md` §5 验收记录（实测覆盖率、未覆盖项）。
- 更新 `.trellis/spec/backend/testing.md`：新增「PG 测试夹具约定」小节（库命名、Truncate、skip 语义、禁止 t.Parallel）。
- 提交信息：`test(service): 核心流程回归测试（发信/同步/垃圾判定/清理）`。

**Done**：三条命令全绿 + 覆盖率达标 + spec 已更新 + 单次提交。

## 回滚点

- 步骤 0/1 任一失败：`git checkout -- backend/internal/service/send_service.go` 撤销 D3，回到纯 PRD 状态重新评估。
- 步骤 2 若 httptest 路径无法构造（适配器强依赖内部注册表）：降级为只覆盖 scheduler + processor + uid/batch，WebAPI 执行器路径记入未覆盖项。
