# P21 执行计划

## 环境

```bash
cd /opt/apps/coding/FusionMail/backend
export GOCACHE=/opt/apps/coding/FusionMail/.trellis/.runtime/gocache
export PATH="$PATH:/home/agentuser/go/bin"
```

## 步骤

### 1. 基线快照（回滚锚点）

```bash
cp docs/swagger.json /tmp/swagger.baseline.json
python3 -c "import json;d=json.load(open('docs/swagger.json'));print('paths',len(d['paths']),'defs',len(d['definitions']))"
```

预期：`paths 110 defs 110`（注意：提交版含 1 处代码生成不出的路径，见步骤 2）

### 2. R1 修孤儿注解

- 读 `internal/handler/oauth2_handler.go`（29 行），取出 18–29 行的注解块
- 读 `internal/handler/oauth2_google.go`，把注解块插到 `func (h *OAuth2Handler) GoogleAuthorize` 正上方
- 在注解块中**新增** `// @Param group_id query int false "分组 ID（可选，授权后直接归入该分组）"`
  （真实 handler 读 `c.Query("group_id")`）
- 从 `oauth2_handler.go` 删除孤儿注解块

验证：

```bash
grep -c "@Router" internal/handler/oauth2_handler.go   # 期望 0
grep -c "@Router /api/v1/auth/google/authorize" internal/handler/oauth2_google.go  # 期望 1
```

### 3. R2 统一 20 处路径前缀

逐文件编辑，把 `@Router /<无前缀路径>` 改为 `@Router /api/v1/<路径>`：

| 文件 | 处数 |
|---|---|
| `internal/handler/auth_2fa.go` | 6 |
| `internal/handler/auth.go` | 3（login / me / change-password） |
| `internal/handler/oauth2_google.go` | 4（callback / reauthorize / refresh / revoke） |
| `internal/handler/oauth2_microsoft.go` | 5 |
| `internal/handler/email_handler.go` | 1（`/emails`） |
| `internal/handler/rule_handler.go` | 1（`/rules/{id}` 的 PUT） |

注意：`/auth/google/authorize` 在步骤 2 搬迁时**直接写成带前缀形式**，不重复处理。

验证：

```bash
grep -rn "@Router /\(auth\|emails\|rules\)" --include=*.go internal/handler/ | grep -v "@Router /api/v1" | wc -l  # 期望 0
```

### 4. 重新生成并核对契约

```bash
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

逐项核对（**契约红线**）：

```bash
python3 - <<'PY'
import json
b = json.load(open('/tmp/swagger.baseline.json'))
a = json.load(open('docs/swagger.json'))
print('paths', len(b['paths']), '->', len(a['paths']))
print('defs ', len(b['definitions']), '->', len(a['definitions']))
print('removed paths:', sorted(set(b['paths'])-set(a['paths'])))
print('added paths  :', sorted(set(a['paths'])-set(b['paths'])))
print('removed defs :', sorted(set(b['definitions'])-set(a['definitions'])))
print('added defs   :', sorted(set(a['definitions'])-set(b['definitions'])))
PY
```

预期：
- `paths 110 → 111`（`/auth/google/authorize` 回归）
- `defs 110 → 110`（不变）
- removed defs / added defs 均为空
- 20 处无前缀路径消失，20 处带前缀路径出现（净 +1 因 google/authorize 回归）

**逐字段核对**：对每个「消失的无前缀路径」，确认其带前缀版本内容完全一致
（`json.dumps` 排序后比较），只允许 path 键不同。

### 5. R3 前端类型收敛

候选清单（12 个，来自 `swagger.json` definitions 短名匹配）：

| 文件 | 类型 | 生成名 |
|---|---|---|
| `adapter.ts` | `Adapter` | `response.Adapter` |
| `adapter.ts` | `AdapterResponse` | `model.AdapterResponse` |
| `email.ts` | `SendEmailRequest` | `handler.SendEmailRequest` |
| `email.ts` | `UpdateSMTPConfigRequest` | `handler.UpdateSMTPConfigRequest` |
| `email.ts` | `AttachmentUploadResponse` | `response.AttachmentUploadResponse` |
| `index.ts` | `EmailListResponse` | `service.EmailListResponse` |
| `index.ts` | `RuleCondition` | `model.RuleCondition` |
| `index.ts` | `RuleAction` | `model.RuleAction` |
| `oauth2.ts` | `OAuth2ClientListResponse` | `response.OAuth2ClientListResponse` |
| `provider.ts` | `Provider` | `response.Provider` |
| `webapi.ts` | `CreateWebAPIProviderRequest` | `handler.CreateWebAPIProviderRequest` |

对每个候选：打印手写字段与生成 schema 字段，逐字段比对后决定改/留。
**不等价的一律保留手写并写理由**。

验证：

```bash
cd ../frontend && npx tsc --noEmit
```

### 6. 重新生成前端类型

```bash
cd /opt/apps/coding/FusionMail/frontend
npm run generate:api-types
npx tsc --noEmit
```

### 7. 新增同步守卫测试

新建 `backend/internal/handler/swagger_sync_test.go`，见 design.md「门禁」节。

验证：临时把一处 `@Router` 改成不存在的路径，测试应失败；改回后通过。

### 8. 全量验证

```bash
cd backend && go build ./... && go vet ./... && go test ./... -race
cd ../frontend && npx tsc --noEmit && npx vitest run && npx vite build
```

## 回滚

```bash
git checkout backend/docs/ backend/internal/handler/ frontend/src/types/
```

## 记录项

- 收敛前手写 interface/type 数：**125**（`frontend/src/types/*.ts`，不含生成物）
- 收敛后数字与替换清单记入 PRD 验收
- `definitions` 数量必须保持 110

---

## 执行结果（2026-10-29）

计划与实际的偏差，逐条记录：

| 步骤 | 计划 | 实际 |
|---|---|---|
| 2 R1 | 修 1 处孤儿注解（Google） | **修了 2 处**。执行中发现 Microsoft 也有一处，且更隐蔽：注解块被拆成两半，`@Router` 单独留在 `oauth2_microsoft.go:14`，导致路径键还在但操作内容被清空。还按 `git show 41de287^` 补回了被截断的 `@Failure 400/500` |
| 3 R2 | 20 处前缀 | **21 处**（漏数了 `/auth/google/authorize`，因为它不在重新生成版里，只在提交版里） |
| 4 核对 | paths 110 → 111 | **111 → 110**。因为 `/rules/{id}` 的 PUT 并入已存在的 `/api/v1/rules/{id}`，21 个无前缀键被 20 个带前缀键取代 |
| 6 前端生成 | 只反映本次改动 | 一并补上了 **P16 积压的漂移**（P16 改了 swagger 但没重新生成前端类型，自 P6 `79a7370` 起未更新） |
| 8 后端测试 | 全绿 | `test/integration` 的 `TestSpamDetectionIntegration` **稳定失败**（`sql: database is closed`）。已在独立 worktree 上对原始 HEAD 复现，**确认是既有失败，与本次改动无关** |

### 契约核对最终结果

```
操作数       128 -> 128   （零丢失）
paths        111 -> 110
definitions  110 -> 110   （零增删、零字段变更）
内容差异操作 2 处，均为有意新增的 group_id 参数
```

### 门禁测试变异验证

注入 `// @Router /api/v1/this/does/not/exist [get]` 到 `oauth2_handler.go` 末尾
→ `TestSwaggerAnnotationsAreEmitted` **FAIL** 并准确指出文件:行号；删除后 **PASS**。

