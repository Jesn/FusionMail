# 2.0 后续 A：核心流程测试补强 — 技术设计

## 1. 现状与约束冲突（必须先解决）

| 来源 | 说法 | 实测 |
|------|------|------|
| 本任务 `prd.md` | 「用例必须走真实 PostgreSQL（fm_automigrate/fm_baseline）」 | 与下两行冲突 |
| `.trellis/spec/backend/testing.md` | SQLite 内存库用于单元/集成测试；PG 仅用于迁移测试 | 现行约定 |
| `.cnb.yml` quality-gate | `docker run golang:1.25-alpine ... go test ./... -race`，**无 PG / 无 Redis 服务** | CI 里 PG 测试只能跳过 |
| 父任务 `prd.md` 缺陷 #2 | 「集成测试用 SQLite 而生产跑 PostgreSQL」= 2.0 要修的根因 | PRD 意图正确 |

本机实测：PG `127.0.0.1:5432`（用户 `fusionmail`，**superuser**，可 `CREATE DATABASE`）、Redis `127.0.0.1:6379` 均可用；`docker` 命令在本机不存在。

**结论**：以 PRD 意图为准（核心流程测试跑真实 PG），但必须让缺 PG 的环境优雅跳过，否则 CI 直接���。SQLite 只保留给纯逻辑/既有测试，不再作为核心流程的默认底座。

## 2. 决策

### D1 数据库：PG 主库 + 每包独立库 + 每测 TRUNCATE

- 连接串优先级：`TEST_DATABASE_URL`（完整 DSN）→ 由 `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD` 拼装（默认 `127.0.0.1:5432 fusionmail/fusionmail_test_pw`��。
- 隔离粒度：**每个测试包一个库** `fm_test_<pkg>`（`TestMain` 里 `DROP DATABASE IF EXISTS` + `CREATE DATABASE` → `AutoMigrate` 一次 → 每个测试 `TRUNCATE ... RESTART IDENTITY CASCADE`）。包之间由 `go test ./...` 并行调度，库不同即无冲突。
- 不可达即跳过：`TestMain` 里 ping 失败 → 打印原因并 `os.Exit(0)`（等价全包 skip），CI 保持绿。
- 复用单一真相：helper 设 `database.DB = testDB` 后调用 `database.AutoMigrate()`，不复制模型清单。

被拒方案：

| 方案 | 拒绝理由 |
|------|----------|
| 继续用 SQLite 内存库 | 正是父任务缺陷 #2；`ILIKE`/`jsonb`/部分唯一索引/`DISTINCT ON` 行为不同，测不出真回归 |
| 每测一个 schema + DSN `search_path` | 依赖 pgx 对未知 DSN 参数的透传行为，未经验证；且连接池下 `SET search_path` 不可靠 |
| 每测一个 database | 单次 `CREATE DATABASE` ≈ 数百 ms，200+ 用例不可接受 |
| 只用 `fm_baseline` 固定库 | 与并发包/并发会话互相 drop 表（spec 已记录该坑） |

### D2 外部依赖替身

| 依赖 | 替身 | 说明 |
|------|------|------|
| PostgreSQL | 真实测试库（D1） | 唯一需要外部进程的依赖 |
| Redis | 真实 Redis（`TEST_REDIS_ADDR`，默认 `127.0.0.1:6379`，独立 DB 15，用前 `FlushDB`） | 配置服务的写路径必须落 Redis（`cache.Set` 直接调 redis，nil 客户端会 panic），无法用 nil 替身；连不上时 `t.Skip` 而不是失败，CI 保持绿。原计划的 miniredis **未引入**（不为测试新增依赖） |
| WebAPI 上游（cloudmail/custom） | `httptest.NewServer` | 走真实 `WebAPIAdapterFactory` + 真实 adapter，只把上游换成假服务 |
| SMTP 发送 | 窄接口替身（D3） | 不启真实 SMTP 服务器，避免端口/超时抖动 |
| IMAP/POP3 拉取 | **不覆盖**（明确排除） | 需要假 IMAP 服务器或大改 `SyncExecutor` 依赖图，收益/风险不划算；在验收记录里显式声明未覆盖 |

### D3 生产代码改动（5 处，全部由用例暴露）

原则：只做「让测试能注入替身」和「用例证明是缺陷」的改动，不做顺带重构。

| # | 文件 | 改动 | 为什么必须改 |
|---|------|------|--------------|
| 1 | `send_service.go`（+`send_service_reply.go`/`send_service_forward.go`） | `SendService.factory` 由具体类型 `*adapter.SenderFactory` 改为包内窄接口 `mailSenderResolver`；新增 `sendVia` 统一把 nil `result` 归一为失败结果 | 具体类型无法注入发送替身；P9 已确立的窄接口模式（`accountSyncRunner`）。`*adapter.SenderFactory` 隐式满足该接口，生产调用点零改动 |
| 2 | `spam_service_detection.go` | `BatchDeleteSpam` 先 `FindByID` 确认存在，缺失的 ID `continue` 不计数 | 原实现对不存在的 ID 也返回 nil，删除计数被虚高（用例 `TestBatchDeleteSpamCountsOnlyExistingEmails` 暴露） |
| 3 | `spam/email_list_service.go` | `addToList` 在 `validateTarget` 前先 `ToLower(TrimSpace(target))` | 原实现用原始 target 查重、用规范化值写库，`"  Friend@Example.COM  "` 会插入第二行且缓存失效用错 key（用例 `TestAddToListNormalizesTarget` 暴露） |
| 4 | `cleanup_service.go` | `cleanupDeletedEmails` 改用 `UpdatedAt`（`DeletedAt` 有效时回退）判断删除时间，并调 `PermanentDelete` 而非软删 `Delete` | 原实现要求 `DeletedAt.Valid`，而 `List` 已按 `deleted_at IS NULL` 过滤 → 两者互斥，该函数恒清理 0 行（用例 `TestCleanupDeletedEmailsPhysicallyRemovesOldTrash` 暴露） |
| 5 | `setting_service.go` | `GetByCategory` 写缓存时直接存 `setting.Value`（数据库里已是密文），不再二次 `Encrypt` | 缓存命中路径只解密一层，二次加密会把密文当明文返回；与 `refreshCache` 的约定矛盾（用例 `TestSettingGetByCategoryCachesSensitiveValuesOnce` 暴��，已用「先还原缺陷→用例失败」验证） |

其余生产代码不动。`SyncExecutor` / `spamService` / `CleanupService` / `SyncScheduler` 的依赖已是接口或可由 D1/D2 满足。

### D4 断言原则

- 断言��务语义：状态流转（`SentEmail.Status`）、副作用（落库行数、reputation/bayesian 训练记录）、错误分类（`SyncErrorPolicy` 分类结果）。
- 禁止快照式断言、禁止只断言「不报错」、禁止 `time.Sleep` 等轮询（用 channel / 可控时钟 / 直接调用内部步骤）。
- 表驱动 + `t.Run` 子用例命名中文；错误路径用 `errors.Is/As` 或错误码，不断言错误字符串全文。
- 测试构造**被测子类型**（`&SyncProcessor{}` / `&spamService{}`），不构造组合器（service-layer.md 约定）。

### D5 覆盖率口径与目标

`go test ./internal/service/ -cover`（statements）。基线实测：service **10.2%**、spam **10.3%**。目标：均 ≥ 30%，且三条核心流程每条至少「正常 1 + 错误 1 + 边界 1」。

### D6 CI 现状与已知缺口（不在本任务内修）

CI 无 PG/Redis，本任务的 PG 测试在 CI 会跳过 → CI 覆盖率提升有限。本机无 `docker`，无法验证「CI 内起 PG」的改法，故**不在本任务盲改 `.cnb.yml`**。缺口记入验收记录，作为后续任务（CI 起 PG 服务 + 让 PG 测试在 CI 真跑）的输入。

## 3. 测试矩阵（实施后：文件名为实际落盘结果）

| 流程 | 被测类型 | 正常路径 | 错误路径 | 边界 | 文件 |
|------|----------|----------|----------|------|------|
| 发信 | `SendService.SendEmail` | 收件人/抄送/附件齐全 → 发送成功 + `SentEmail` 落库 `sent` | 账户不存在 / 凭据解密失败 / 发送器返回失败 → `SentEmail` 落库 `failed` + 错误透出 | 空收件人 / 无主题 / 附件 0 字节 / `Bcc` 不进 `ToAddresses` | `send_service_flow_test.go` |
| 发信 | `SendService.Reply/ReplyAll/Forward` | 引用头 `In-Reply-To`/`References`、`Re:`/`Fwd:` 前缀、ReplyAll 去重自身 | 原邮件不存在 / 原邮件不属于该账户 | 原邮件无 `Message-ID`、无 `References`、收件人含自己 | `send_service_reply_test.go` |
| 发信 | `parseCredentials` | `oauth2`/`quick`/`password` 三类凭据解析 | 密文损坏 → 解密错误；JSON 非法 → 解析错误 | `EncryptedCredentials` 为空 → `nil, nil` | `send_service_credentials_test.go` |
| 发信/同步 | `CredentialResolver.Resolve` | password / quick / oauth2 三类凭据解析 + IMAP/POP3 服务端配置 | 解密失败 / 非法 JSON / IMAP 缺 host+port | `mail.linuxdo.org` → `mail.linux.do` 改写；加密方式 ssl/starttls/none/未知；`token_expiry` 非法值 | `credential_resolver_test.go` |
| 同步 | `SyncScheduler.SyncAllAccounts` | 多账户并发（信号量 ≤5）全部调用 runner | runner 报错 → 聚合错误非 nil | 0 账户 / 全部禁用同步 | `sync_scheduler_test.go` |
| 同步 | `SyncScheduler.shouldSync` / `CleanupStaleSyncLogs` | 间隔到期应同步；陈旧日志被清理并计数 | repo 报错 | 首次同步（无 `LastSyncAt`）、`maxAge=0` | 同上 |
| 同步 | `SyncExecutor.SyncAccount`（WebAPI 路径） | httptest 上游返回 2 封 → 落库 2 行 + `SyncLog` 计数 | 上游失败 → `SyncLog` 落 `failed`；凭据解密失败 fail-closed；adapter 创建失败 | webhook-only 配置跳过；字段映射；重复 Message-ID 去重 | `sync_executor_webapi_test.go` |
| 同步 | `SyncProcessor` | 新邮件落库字段映射；已存在则更新不重复；批量计数 | 落库失败 → 计数不含失败项；context 取消即停 | 已删除 key 跳过、软删邮件跳过、垃圾检测/规则依赖缺失、附件行构造 | `sync_processor_test.go` |
| 垃圾判定 | `spamService.MarkAsSpam/UnmarkAsSpam` | 标记 → `is_spam` 置位 + reputation 更新 + bayesian 训练记录 | 邮件不存在 | 重复标记幂等 | `spam_service_flow_test.go` |
| 垃圾判定 | `BatchDeleteSpam` / `EmptySpamFolder` / `GetSpamEmails` | 计数正确、非垃圾邮件不受影响 | repo 报错 | 不存在的 ID 不计入（暴露缺陷 #2）、账户隔离 | 同上 |
| 垃圾判定 | 规则 CRUD + `TestRule` / `GetRuleStats` | 建/改/删/切换 + 统计计数 | 规则不存在、非法正则 | 分页 page=0、内置规则不可删 | `spam_service_rules_test.go` |
| 垃圾判定 | `RuleService`（规则引擎） | 条件/动作全类型、命中计数、命中记录批量落库、webhook 出站 | 非法配置拒绝落库、未知动作、webhook 非 2xx | all/any 模式、StopProcessing、标签动作幂等、SSRF 协议白名单 | `rule_service_flow_test.go` |
| 垃圾判定 | `spam/bayesian_classifier` | 训练后分类倾向正确 | 未训练模型 / 样本不足或失衡 | 空语料 / 单类样本 / 重置 | `spam/bayesian_classifier_test.go` |
| 垃圾判定 | `spam/reputation_manager` | 反馈调分、信任等级、统计 | 未知发件人 | 分数上下限钳制、阈值边界 | `spam/reputation_manager_test.go` |
| 垃圾判定 | `spam/email_list_service` | 白/黑名单增删查、域名匹配 | 非法 target、越权删除 | 重复添加幂等（暴露缺陷 #3）、内置规则初始化幂等 | `spam/email_list_service_test.go` |
| 清理 | `CleanupService.ManualCleanup/ManualSpamCleanup/ManualLogCleanup` | 过期回收站/垃圾/日志按保留天数清理并返回计数 | 保留天数为负数（禁用）/ 非法值 / 账户清理报错 | 无可清理数据、`DeletedEmailKey` 孤儿清理、`SpamDetectedAt` 回退 `SentAt` | `cleanup_service_test.go` |
| 清理 | `EmailDeleteService` | 软删标记、恢复、永久删除、清空回收站 | 不存在 / 非回收站邮件拒绝永久删除 | 批量去重、服务器软删除重试（404 视为成功 / 重试耗尽 / context 取消） | `email_delete_service_test.go` |
| 配置 | `SettingService` | 读写往返、用户级覆盖系统级、批量写、重置默认值、搜索/统计 | 敏感配置解密失败、非公开配置以 `OnlyPublic` 读取被拒 | 敏感值密文落库、缓存命中路径（暴露缺陷 #5）、公开白名单 | `setting_service_test.go` |
| 夹具 | `pgtest` 自身 | 建库 + AutoMigrate + 每测 TRUNCATE | — | 种子表（providers/adapters/settings）不被清空；去重唯一索引真实生效 | `internal/testutil/pgtest/pgtest_test.go` |

**明确未覆盖**（在 §5 声明）：

- IMAP/POP3 拉取路径（需假 IMAP 服务器或大改 `SyncExecutor` 依赖图，收益/风险不划算）。
- `AccountService` CRUD 补强（`account_service_test.go`）：覆盖���在未动它的情况下已达标，未做。
- CI 内真跑 PG/Redis 测试（见 D6）。

## 4. 风险与回滚

| 风险 | 缓解 |
|------|------|
| `database.DB` 是包级全局，helper 会改写它 | DB 用例禁止 `t.Parallel()`；helper 在 `TestMain` 里一次性设置，测试内不再改 |
| `AutoMigrate` 在空库上可能需要扩展（`pg_trgm`/`uuid-ossp`） | 步骤 0 冒烟先验证；失败则在 helper 里 `CREATE EXTENSION IF NOT EXISTS`（`fusionmail` 是 superuser） |
| PG 建库 + AutoMigrate 增加测试耗时 | 每包一次（≈2-4s），可接受；不用每测重建 |
| 并发会话同时跑迁移测试 drop 表 | 本任务用独立库 `fm_test_*`，不碰 `fm_automigrate`/`fm_baseline` |
| 配置服务用���依赖真实 Redis | 连不上即 `t.Skip`（CI 无 Redis 时跳过而非失败）；用独立 DB 15 + `FlushDB`，不污染本机数据 |
| 生产改动可能引入回归 | 5 处改动都有对应用例；改动 2/3/4/5 先用「还原缺陷→用例必须失败」验证过 |
| 覆盖率不达标（30%） | 优先补高风险流程；若确实不足，在验收记录里给出实测值 + 未覆盖原因，不写空断言凑数 |

**回滚**：改动 = 新增 `*_test.go` + `internal/testutil/pgtest` + D3 的 5 处生产改动。`git revert` 即可，无 schema/迁移/API 契约变更。

## 5. 验收记录（实施后填写）

### 5.1 覆盖率（本机，真实 PG + Redis）

| 包 | 基线 | 实测 | 目标 | 结论 |
|----|------|------|------|------|
| `internal/service` | 10.2% | **32.2%**（1734/5383 statements） | ≥ 30% | 达标 |
| `internal/service/spam` | 10.3% | **36.5%** | ≥ 30% | 达标 |

命令：`go test ./internal/service/ -cover -count=1`、`go test ./internal/service/spam/ -cover -count=1`。

### 5.2 全量测试

- `go build ./...` 通过；`go vet ./...` 无告警。
- `go test ./... -race -count=1`：**39 个包全绿**（含 `internal/service` 42s、`internal/service/spam` 7s、`test/integration` 8s）。
- 观察到的既有抖动（非本任务引入）：首次全量跑时 `test/integration` 出现过一次失败，日志为 `sql: database is closed`（垃圾检测异步 goroutine 在测试关闭 DB 后继续写库）；随后单独重跑 3 次 + 全量重跑 1 次均通过。未在本任务修（不在范围内），仅记录。

### 5.3 降级模式（等价 CI：无 PG、无 Redis）

命令：`DB_HOST=127.0.0.1 DB_PORT=1 TEST_REDIS_ADDR=127.0.0.1:1 go test ./... -count=1`

- 退出码 0，**0 个 FAIL**。
- `internal/service` + `internal/service/spam` + `internal/testutil/pgtest` 明细：**125 SKIP / 102 PASS / 0 FAIL**（DB 用例整包跳过，纯逻辑用例照常跑）。
- 结论：CI（`.cnb.yml`，无 PG/Redis）保持绿。

### 5.4 未覆盖项（显式声明）

- **IMAP/POP3 拉取路径**：需要假 IMAP 服务器或大改 `SyncExecutor` 依赖图，本任务明确排除；`CredentialResolver` 的 IMAP/POP3 配置解析已覆盖，未覆盖的是真实协议交互。
- **`AccountService` CRUD 补强**：覆盖率在未改 `account_service_test.go` 的情况下已达标，未做。
- **CI 内真跑 PG/Redis**：见 D6，本机无 `docker` 无法验证，留给后续任务。
- `parseCredentials` 的 oauth2 分支在缺 `client_id` 时会解引用 `s.oauth2ConfigProvider`（潜在 panic 路径），本任务未修也未覆盖。

### 5.5 缺陷修复验证方式

D3 的改动 2/3/4/5 均用「先还原缺陷 → 对应用例必须失败 → 再恢复修复」验证过用例真的能发现该缺陷（改动 5 的验证记录见提交说明）。

