# Service Layer — 上帝对象绞杀模式与 DI 约定

> 来源：2.0 P9。四个上帝对象（emailService / accountService / WebAPIProviderService /
> syncService）的拆分实证，模式已验证可复用。

---

## 上帝对象绞杀模式（七步）

Go 没有 partial class —— 跨文件拆分只带来"文件级"错觉，必须**按用例拆成独立类型**。

### 1. 按用例分组方法，不按实体

先列出全部方法，按"调用方是谁、一起完成什么用例"分组，收窄每个新类型的字段依赖：

```
emailService (19 字段 / 22 方法)
→ EmailQueryService   (5 字段 / 9 方法)   查询用例
→ EmailStatusService  (3 字段 / 5 方法)   状态变更用例
→ EmailDeleteService  (5 字段 / 8 方法)   删除用例
```

### 2. 每个新类型独立构造函数

```go
func NewEmailQueryService(emailRepo, accountRepo, adapterFactory, credentialResolver) *EmailQueryService
```

### 3. 大接口拆成 N 个窄接口

```go
type EmailQueryServiceInterface  interface { /* 9 方法 */ }
type EmailStatusServiceInterface interface { /* 5 方法 */ }
type EmailDeleteServiceInterface interface { /* 8 方法 */ }
```

### 4. 保留组合接口 + 纯转发组合器（handler 零改动）

```go
type EmailService interface { EmailQueryServiceInterface; EmailStatusServiceInterface; EmailDeleteServiceInterface }

type compositeEmailService struct{ query *EmailQueryService; status *EmailStatusService; delete *EmailDeleteService }

// 19 个方法全部是纯转发，一行逻辑都不许有
func (c *compositeEmailService) GetEmailList(ctx context.Context, ...) (...) {
    return c.query.GetEmailList(ctx, ...)
}
```

### 5. 测试只构造被测类型

`&emailService{}` → `&EmailQueryService{}`（按被测方法归属选类型）。

### 6. 跨服务共享的辅助函数提升为包级函数

**不要**让一个新服务依赖另一个新服务（上帝对象 → 上帝依赖）。
依赖收窄后共同需要的方法（如 `toAccountResponse`、`loadWebAPIAccount`、
`isWebAPIAccount`）提升为包级函数，显式传依赖：

```go
// Wrong: AccountSyncStateService 依赖 AccountCRUDService 只为调一个转换函数
// Correct:
func toAccountResponse(account *model.EmailAccount) AccountResponse { ... }
func loadWebAPIAccount(ctx context.Context, accountRepo, providerRepo, uid) (*model.EmailAccount, *model.Provider, error) { ... }
```

### 7. 判定"哪些方法不该拆"的依据是依赖图，不是行数

同一文件里的两个方法，一个该拆一个不该拆，看它用哪些字段。

## 实证记录

| 上帝对象 | 拆分结果 | 提交 |
|----------|----------|------|
| emailService | 3 类型 + 组合器（19 转发方法） | `56e4355` |
| accountService | 3 类型 + 组合器（22 转发方法） | `a38ef9a` |
| WebAPIProviderService | 3 类型 + 组合器（16 转发方法） | `cc28788` |
| syncService | 第一阶段：SyncProcessor + SyncErrorPolicy 叶子服务 | `a3ed0e4` |

## DI 约定（P8）

- **禁止**包级可变 DB/Redis 单例；依赖经构造函数显式注入
- handler 不得 import gorm；通过 `handler.SSEUserStore` 这类窄接口拿数据
- 组合根在 `cmd/server/container.go`，构造顺序 = 依赖顺序

## Validation & Error Matrix

| 条件 | 结果 |
|------|------|
| 组合器方法里写了逻辑（非纯转发） | 违反模式 —— 逻辑放错类型，下次拆分更难 |
| 新服务字段依赖另一个新服务 | 违反模式 —— 用包级函数显式传依赖 |
| 测试构造组合器而非被测子类型 | 测试与实现耦合，拆分时全断 |
