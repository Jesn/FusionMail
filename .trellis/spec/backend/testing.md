# Testing — 测试约定与守卫测试清单

> 来源：2.0 P0/P3。当前基线：后端 22 包全绿（-race），前端 8 文件 36 测试。

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
cd backend  && go build ./... && go vet ./... && go test ./... -race -count=1   # 22 包
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
