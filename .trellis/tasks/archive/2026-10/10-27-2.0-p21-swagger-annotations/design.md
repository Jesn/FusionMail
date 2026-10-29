# P21 技术设计

## 现状与根因

### 根因 1：文件拆分遗留孤儿注解（两处）

`41de287`（handler 大文件拆分）把函数体搬到新文件，但 swag 注解块留在了原处。
swag 只处理**紧邻函数声明**的注释组，孤儿注释块被静默丢弃。

**第一处：GoogleAuthorize**

```
internal/handler/oauth2_handler.go   (29 行)
  ...NewOAuth2Handler...
  // GoogleAuthorize 生成 Google OAuth2 授权 URL
  // @Summary ...
  // @Router /auth/google/authorize [get]     <-- 后面没有任何 func
```

后果：整条路径从重新生成的文档里**消失**（提交版还有，因为生成于拆分前）。

**第二处：MicrosoftAuthorize**（更隐蔽）

拆分把这一块**切成了两半**，分放在两个文件里：

```
internal/handler/oauth2_google.go:386-393   （文件末尾）
  // MicrosoftAuthorize 生成 Microsoft OAuth2 授权 URL
  // @Summary / @Description / @Tags / @Accept / @Produce
  // @Param email ...
  // @Success 200 ...                          <-- 无 @Router，且 @Failure 400/500 被截断丢失

internal/handler/oauth2_microsoft.go:14
  // @Router /auth/microsoft/authorize [get]   <-- 单独一行，紧邻真实 func
  func (h *OAuth2Handler) MicrosoftAuthorize(...)
```

后果：路径键**还在**（由那行孤立的 `@Router` 产生），但**操作内容被清空** ——
summary / parameters / responses 全丢。这比第一处更难发现：只对比路径键的检查
完全看不出来。

**契约核对方式**：必须做**操作级**对比（`paths[path][method]` 的内容），
不能只比路径键。初版 PRD 就是只比了键，漏掉了第二处。

CI 只跑 `swag init` 覆盖、不 diff，故两处漂移长期不可见。

### 根因 2：路径前缀两套写法

`router.Group("/api/v1")` 是所有业务路由的挂载点，故真实路径一律带 `/api/v1`。
但注解有两种写法：

- 90 个写全路径 `/api/v1/xxx`（正确）
- 21 个写前端相对路径 `/auth/login`、`/emails`、`/rules/{id}`（照
  `frontend/src/lib/constants.ts` 抄的，那里 baseURL 会自动补 `/api/v1`）

`swagger.json` 无 `basePath`，所以后 21 个在文档里指向不存在的路径。

### 根因 3：前端生成物自 P6 起未更新

`frontend/src/types/api.generated.ts` 最后一次提交是 `79a7370`（P6）。
P16（`0f7fa1d`）改了 25 个文件、大幅扩充了 `definitions`（72 → 110），
**但没有重新生成前端类型**。因此本次重新生成会一并补上 P16 积压的漂移
（diff 约 2832+/2527-），这部分不是本次改动引入的。

### 根因 4：手写类型与生成物并存

`frontend/src/types/*.ts` 有 125 处手写 interface/type。P6 已确立「生成物为单一来源」
并示范了别名写法（`oauth2.ts`、`webapi.ts`），但只迁移了一部分。


## 设计

### R1 注解搬迁（两处）

**GoogleAuthorize**：把 `oauth2_handler.go` 的注解块整体移到 `oauth2_google.go`
的 `GoogleAuthorize` 函数正上方。`oauth2_handler.go` 只保留结构体与构造函数。

**MicrosoftAuthorize**：把 `oauth2_google.go` 末尾的半截注解块移到
`oauth2_microsoft.go` 的 `MicrosoftAuthorize` 正上方，与那行孤立的 `@Router` 合并；
并**按 `git show 41de287^` 的原始块补回被截断的 `@Failure 400/500`**。

两处都补 `group_id` 参数（真实 handler 读 `c.Query("group_id")`，用于把新授权账号
直接归入指定分组）—— 这是**唯一的新增语义**，且它描述的是**已经存在**的行为
（handler 早就在读这个参数），只是注解漏了。

除 `group_id` 外，不新增、不删除任何注解语义，仅改变注解的挂载位置。

### R2 前缀统一

给 21 处注解的 `@Router` 路径补 `/api/v1` 前缀。

选择「补前缀」而非「加 basePath」的理由：

- 90/111 已是全路径，补 21 个是少数服从多数，改动面最小
- 加 `basePath` 会让那 90 个变成 `/api/v1/api/v1/...`，需要同时改 90 个
- 前端 `constants.ts` 用的是相对路径 + baseURL 拼接，与 swagger.json 无关，
  不受影响

**影响面核实**：前端仅 4 个文件 import 生成物
（`services/smtpService.ts`、`types/oauth2.ts`、`types/webapi.ts`、`types/email.ts`），
且全部取 `components['schemas'][...]`，**无人使用 `paths` 键**。
故 `paths` 键变更对前端零影响。

### R3 类型收敛判据

对每个候选，逐字段比对「手写定义」与 `components['schemas'][<生成名>]`：

| 情况 | 处理 |
|---|---|
| 字段集合、类型、可选性**完全一致** | 改为 `export type X = Schemas['pkg.X']` |
| 生成类型**更严格**（手写用了 `any`/可选，生成物精确） | 改为别名，并在消费处修类型错误 |
| 生成类型**更宽松**（如全字段可选，而前端依赖必填） | **保留手写**，写明理由（先例：`OAuth2Client`） |
| 手写类型是前端专有的运行时形态（无后端对应） | 保留 |

**判据以 `swagger.json` 的 `definitions` 为 ground truth**，不以「名字像」为准。

### 门禁：swagger 同步守卫测试

新增 `internal/handler/swagger_sync_test.go`（package `handler_test`），
沿用 `internal/dto/response/boundary_test.go` 的「解析源码 + 解析 swagger.json + 断言」风格：

1. 遍历 `internal/handler/*.go`（跳过 `_test.go`）
2. 正则提取所有 `@Router <path> [<method>]`
3. 读 `docs/swagger.json`
4. 断言每个注解的 path 存在于 `paths`，且该 method 存在于 `paths[path]`

这条断言**恰好能抓住本次的 bug**：孤儿注解会被源码扫描到，但 swag 没把它写进
`swagger.json`，于是测试失败。

反向（路由注册了但没注解）**不做**：需要解析 `router.go` 的嵌套 Group 前缀，
脆弱且易误报。本次 113 个「无注解路由」中大部分是 `/metrics`、`/health` 这类
本就不该进 API 契约的端点。

## 风险

| 风险 | 缓解 |
|---|---|
| 补前缀后前端生成的 `paths` 键变化，破坏消费方 | 已核实无消费方使用 `paths` 键；且 tsc 会兜住 |
| 类型收敛时误删前端依赖的必填语义 | 逐字段比对；不等价的一律保留手写并写理由 |
| 注解搬迁时改动响应形状 | 只搬位置，不改任何 `@Success`/`@Param` 语义（仅**新增** `group_id` 参数，它本就存在） |
| 守卫测试过于严格导致误报 | 只断言「注解 → swagger.json」单向；断言前先确认提取到的注解数 > 0，防止正则失效后测试空转通过 |
| swag 版本差异（本机 v1.16.4 vs CI 提示 v1.16.6） | 生成后与提交版做零 diff 校验；若版本差异导致 diff，记录在案 |
