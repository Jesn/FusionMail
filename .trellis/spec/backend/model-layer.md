# Model Layer — gorm 模型边界契约

> 来源：2.0 P6（分层边界强制）。核心教训：**"model 不含 json tag"按字面执行会破坏线上数据**。

---

## 三类类型的判定规则

`internal/model/` 里允许存在的 struct 只有三类，写代码前先判断目标属于哪类：

| 分类 | 判定特征 | json tag | 例子 |
|------|----------|----------|------|
| **gorm 表模型** | 映射数据库表，出现在 `pkg/database/migration.go` 的 AutoMigrate 注册表 | ❌ 禁止 | `Email`, `EmailAccount`, `User` |
| **JSON 持久化值对象** | 实现 `sql.Scanner`/`driver.Valuer`，或被 `json.Marshal` 后写入文本/jsonb 列 | ✅ 必须有（tag 就是存储格式） | `RuleCondition`, `RuleAction`, `SyncProgress` |
| **请求/响应 DTO** | 出现在 handler 的请求/响应路径 | ✅ 有，但**应该迁到 `internal/dto/`** | `OAuth2ClientResponse` 等 9 个遗留 |

## 为什么 JSON 持久化值对象的 tag 不可删（Wrong vs Correct）

#### Wrong
```go
// "清理 model 层 json tag"时把 RuleCondition 的 tag 也删了
type RuleCondition struct {
    Field string `gorm:"..."`   // 删掉了 json:"field"
    Value string `gorm:"..."`
}
// 后果：RuleConditions.Scan 用 json.Unmarshal 读 email_rules.conditions 列，
// 历史数据存的是 {"field":"subject"}，没有 tag 后键名变 "Field"，
// 所有已存规则的条件静默失配 —— 且不报错。
```

#### Correct
```go
// 1. 先判定：RuleCondition 经 RuleConditions.Scan/Value 持久化 → tag 承重，保留
// 2. 若确属表模型，删 tag，并确认 swagger.json 逐字节不变
// 3. 把例外登记进 json_tag_guard_test.go 的 jsonTagAllowlist，附理由
```

## 守卫测试（改 model 层前必读）

`internal/model/json_tag_guard_test.go`：

| 测试 | 拦截的违规 |
|------|-----------|
| `TestNoJSONTagsOnGormTableModels` | 给表模型加回 json tag |
| `TestJSONTagAllowlistIsCurrent` | 白名单里的类型已不再带 tag（防腐化） |
| `TestModelPackageHasNoStrayJSONTags` | 新 DTO 直接落在 model 包 |
| `TestModelDirIsScannable` | glob 写错导致前三个测试假通过 |

**新增例外类型的流程**：在 `jsonTagAllowlist` 加条目 + 注释写明持久化证据
（哪个列、经哪个 Scan/Value/Marshal 路径），否则测试不过。

## 纯 JSON 配置结构的归属

WebAPI 适配器的认证/分页/字段映射结构（`CloudflareTempEmailAuthData` 等 7 个）
**不是** gorm 模型，是 `email_accounts.auth_data` / `providers.metadata` 列的持久化格式。
它们住在 `internal/dto/webapiconfig/`（2026-10-27 从 model 包迁出），引用方式：

```go
import "fusionmail/internal/dto/webapiconfig"

var config webapiconfig.CloudMailAuthData
json.Unmarshal([]byte(account.AuthData), &config)
```

不要把它们移回 model 包，也不要"顺手"删它们的 json tag。

## Validation & Error Matrix

| 条件 | 结果 |
|------|------|
| 表模型带 json tag | `TestNoJSONTagsOnGormTableModels` FAIL，CI 阻断 |
| swagger.json 出现 gorm 模型 | `TestNoGORMModelLeaksIntoAPIContract` FAIL，CI 阻断 |
| 删了 JSON 持久化值对象的 tag | 编译期不报错、测试不报错 —— **线上数据静默损坏**，只能靠 code review 拦 |
| model 包新增带 tag 的 DTO | `TestModelPackageHasNoStrayJSONTags` FAIL |

## Tests Required

改 model 层后必须全跑：

```bash
cd backend
go build ./... && go vet ./...
go test ./internal/model/ ./internal/dto/... -v -count=1
# 契约不变性证明：
cp docs/swagger.json /tmp/before.json
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
diff /tmp/before.json docs/swagger.json   # 必须逐字节一致
```
