# Testing — 测试约定与守卫测试清单

> 来源：2.0 P0/P3，2.0 后续 A（P19）补充 PG 夹具约定。当前基线：后端 23 包全绿（-race），前端 8 文件 36 测试。

---

## 后端测试环境约定

### SQLite 内存库（单元/集成测试）

```go
// test/integration/helper_test.go
dsn := "file:" + path + "?mode=memory&cache=shared&_busy_timeout=10000"
db, _ := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
sqlDB.SetMaxOpenConns(1)   // 必须：见下方 gotcha
```

> **Warning**: `cache=private`（默认）会让连接池里每个连接各自一个独立内存库，
> 报 `no such table: spam_rules`。必须 `cache=shared` + `MaxOpenConns(1)`。

### PostgreSQL（仅迁移测试）

```bash
# 测试库：fm_automigrate / fm_baseline / fm_migrate_test / fm_v2
# 用户 fusionmail / fusionmail_test_pw @ 127.0.0.1:5432
export DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=fusionmail DB_PASSWORD=fusionmail_test_pw
export DB_NAME=fm_migrate_test DB_SSLMODE=disable
export ENCRYPTION_KEY="test-2fa-secret-key-32-bytes-ok!"   # 必须 32 字节
export JWT_SECRET="12345678901234567890123456789012"
```

> **Warning**: 多个会话/智能体并发跑迁移测试会互相 drop 表 —— 同一时刻只允许
> 一个后端全量测试在跑。

### PostgreSQL 测试夹具 `internal/testutil/pgtest`（核心流程测试默认底座）

**什么时候用它**：测试要断言「真实落库语义」（唯一索引冲突、软删作用域、`UPDATE ... RETURNING` 行数、
`jsonb`/`ILIKE` 行为）时，必须用真实 PG。纯逻辑/解析/状态机仍可用 SQLite 或纯内存替身。

```go
// 每个使用夹具的测试包一个 TestMain（包级一次建库 + AutoMigrate）
func TestMain(m *testing.M) { pgtest.Init("service"); os.Exit(m.Run()) }

func TestSomething(t *testing.T) {
    db := pgtest.DB(t)   // 不可达时 t.Skip（不是 Fatal）
    pgtest.Truncate(t)   // TRUNCATE ... RESTART IDENTITY CASCADE
    ...
}
```

约定：

- **连接串**：`TEST_DATABASE_URL`（完整 DSN）优先；否则由 `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_SSLMODE`
  拼装，默认 `127.0.0.1:5432 fusionmail/fusionmail_test_pw`。
- **隔离粒度**：每个测试包一个库 `fm_test_<pkg>`（`TestMain` 里 drop+create → `AutoMigrate` 一次 →
  每个测试 `TRUNCATE`）。包之间库不同，`go test ./...` 并行无冲突；**不要**用 `fm_baseline`/`fm_automigrate`。
- **不可达即跳过**：`DB(t)` 在 PG 连不上时 `t.Skip`，`TEST_PG_DISABLE=1` 可强制跳过。
  CI（`.cnb.yml`）没有 PG 服务，靠这条保持绿。
- **禁止 `t.Parallel()`**：`database.DB` 是包级全局，夹具会改写它。
- **种子表**：`providers` / `adapters` / `provider_adapters` / `settings` 不在 TRUNCATE 范围内。
  用例自己插入的 `settings` 行必须在 `t.Cleanup` 里删掉（见 `cleanup_service_test.go` 的 `seedSetting`）。
- **时间戳伪造**：要造「很久以前删除」的数据，用 `UpdateColumns`（跳过 GORM 自动时间戳），
  不要 `Save`/`Update`。

#### 夹具契约速查（环境变量 → 行为）

| 环境变量 | 取值 | 行为 |
|----------|------|------|
| `TEST_DATABASE_URL` | 完整 DSN | 优先使用，忽略下面 5 个 `DB_*` |
| `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_SSLMODE` | 缺省 `127.0.0.1:5432 fusionmail/fusionmail_test_pw disable` | 拼装 DSN |
| `TEST_PG_DISABLE` | `1` | 强制跳过所有 DB 用例（不尝试连接） |
| `TEST_REDIS_ADDR` | 缺省 `127.0.0.1:6379`（DB 15） | Redis 依赖用例的连接目标 |

| 条件 | 期望行为 |
|------|----------|
| PG 连不上（ping 超时 1.5s） | `t.Skip`，**不是** `t.Fatal`；整包退出码 0 |
| `AutoMigrate` 失败 | `t.Fatal`（环境坏了，不能静默跳过） |
| `Truncate` 遇到种子表 | 跳过该表，不报错 |
| Redis 连不上 | 仅 Redis 依赖用例 `t.Skip` |

#### Wrong vs Correct

```go
// Wrong：PG 不可达就 Fatal → CI（无 PG）整条流水线红
db, err := gorm.Open(postgres.Open(dsn))
if err != nil { t.Fatalf("连不上 PG: %v", err) }

// Wrong：DB 用例用 t.Parallel() → database.DB 全局被并发改写，随机失败
func TestX(t *testing.T) { t.Parallel(); db := pgtest.DB(t) }

// Correct
func TestMain(m *testing.M) { pgtest.Init("service"); os.Exit(m.Run()) }
func TestX(t *testing.T) {
    db := pgtest.DB(t) // 不可达 → Skip
    pgtest.Truncate(t)
}
```

### Redis 依赖

配置服务的写路径必须落 Redis（`cache.Set` 直接调 redis，nil 客户端会 panic），无法用 nil 替身。
用例用真实 Redis：`TEST_REDIS_ADDR`（默认 `127.0.0.1:6379`）+ **独立 DB 15** + 用前 `FlushDB`；
ping 失败就 `t.Skip`（CI 无 Redis）。参考 `setting_service_test.go` 的 `settingTestRedis`。

### 注入替身：窄接口 seam

生产类型若依赖具体结构体（无法注入替身），把它改成**包内窄接口**，只声明测试需要的那几个方法：

```go
// 先例：P9 的 accountSyncRunner、P19 的 mailSenderResolver
type mailSenderResolver interface {
    GetSenderWithFallback(account *model.EmailAccount, credentials *adapter.AccountCredentials) (adapter.MailSender, error)
}
```

具体类型隐式满足接口 → 生产调用点零改动。替身用「内嵌 nil 接口」写法，
只实现被测路径会走到的方法（未实现的方法调用即 panic = 作用域断言）：

```go
type stubAccountRepo struct{ repository.AccountRepository } // 内嵌 nil
func (r *stubAccountRepo) FindByUID(...) (...) { ... }
```

### GORM 三个反复踩到的坑

| 坑 | 现象 | 正确做法 |
|----|------|----------|
| `default:true` + `false` 零值 | `Create` 时 GORM 认为「未设置」，写回默认 `true` | 建行后用 `Update("enabled", false)` 显式落库 |
| `Save` 写全字段 | 想只改一个字段却覆盖了其它字段 | 只改状态用 `UpdateLocalStatus`/`UpdateColumns` |
| 软删作用域 vs 业务删除 | `List`/`Count` 自动加 `deleted_at IS NULL`，与 `is_deleted`（用户回收站）是两个概念 | 断言前想清楚测的是哪一个；`Unscoped()` 才能看到软删行 |

### 用例必须能发现缺陷（还原验证法）

写完「修了某个 bug」的用例后，**先把修复还原、确认用例失败**，再恢复修复。
否则用例可能只是断言了当前（错误）行为。P19 的 4 处缺陷修复都用此法验证过。

### 测试密钥长度

2FA/加密相关测试的密钥必须恰好 32 字节，如 `test-2fa-secret-key-32-bytes-ok!`。
（历史坑：35 字节密钥导致 round-trip 测试失败。）

## Mock 约定

- 手写 mock（同包 `mockEmailRepo` / `mockAccountRepo`），不用 testify/gomock
- mock 只实现被测路径需要的方法，其余返回零值
- 测试构造**被测子类型**而非组合器（见 service-layer.md）

## 守卫测试清单（动对应层前必读）

| 守卫 | 位置 | 拦截 |
|------|------|------|
| 表模型 json tag | `internal/model/json_tag_guard_test.go` | 表模型定义 API 线格式 |
| swagger 泄露 | `internal/dto/response/boundary_test.go` | gorm 模型进 API 契约 |
| CI 门禁 | `scripts/ci-quality-gate.sh` + `.cnb.yml` | build/vet/test 不全绿 |

注入验证法：往 swagger.json 塞一个 `model.Adapter` 定义，
`TestNoGORMModelLeaksIntoAPIContract` 必须失败并点名模型 —— 守卫自身要被测试。

## 提交前完整验证（全跑，不许跳）

```bash
cd backend  && go build ./... && go vet ./... && go test ./... -race -count=1   # 23 包（含 pgtest）
# 无 PG/Redis 环境（等价 CI）必须 0 FAIL，DB 用例整包跳过：
DB_HOST=127.0.0.1 DB_PORT=1 TEST_REDIS_ADDR=127.0.0.1:1 go test ./... -count=1
cd frontend && npx tsc --noEmit && npx vitest run                                # 36 测试
# 改了 API 注解/DTO 时另加：
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
git diff docs/swagger.json   # 非空 = 契约变更，需前端同步并显式声明
```

## Common Mistake: 静默 no-op 冒充已实现

**Symptom**: `validateAction` 声明动作合法、甚至要求必填 value，
`executeAction` 里却是 `case "add_label": return nil // TODO`。

**Why it's bad**: 用户配置成功 → 系统接受 → 什么都不发生，且日志无痕。
P10 实证：`add_label`/`remove_label`/`webhook` 三个动作如此存活了整个 1.0。

**Prevention**: 校验清单与执行分派必须同源维护；新增动作类型时若暂不实现，
校验层就应拒绝（fail-fast），而不是让用户以为配好了。
