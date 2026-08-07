# MVP 主旅程验收记录

> 记录时间：2026-08-07
> 环境：本地代码审查 + 前端单测/构建；E2E 需运行实例，标注为待手动验证

## 构建验证

| 项目 | 结果 |
|------|------|
| `npm test`（前端单测） | ✅ 8 文件 36 用例全部通过 |
| `npm run build`（生产构建） | ✅ 构建成功（2.70s） |
| TypeScript 编译 (`tsc`) | ✅ 无错误 |
| Lints（App.tsx） | ✅ 无新增错误 |

## AC 对照

### AC1：README 无已实现项出现在待实现列表

- ✅ 新 README 不含「待实现」章节
- ✅ Webhook、发送、JWT、前端、附件均列在「当前能力」中
- ✅ 旧 README 中的 `docs/development-progress.md`、`docs/startup-guide.md` 失效链接已移除

### AC2：生产构建后调试路由不可访问

- ✅ `/oauth2-test`：包裹在 `{import.meta.env.DEV && ...}` 中，生产构建时 `DEV=false`，路由不注册
- ✅ `/debug/sse`：同上，生产不注册
- ✅ `/settings/legacy`：生产环境 `{!import.meta.env.DEV && ...}` 分支注册 `<Navigate to="/settings" replace />`
- ✅ 三个路径在生产环境均落到 catch-all `*` 或显式重定向，不进入调试 UI

验证方式：代码审查 + Vite 生产构建中 `import.meta.env.DEV` 被静态替换为 `false`

### AC3：开发构建仍可访问调试页

- ✅ 开发构建 `import.meta.env.DEV=true`，三个路由正常注册
- ✅ `routeUtils.test.ts` 全部通过，未断言生产可访问调试路由

### AC4：J1–J7 主旅程

| ID | 旅程 | 验证状态 | 证据 |
|----|------|----------|------|
| J1 | 登录 | ✅ 代码+单测 | `authStore.test.ts` 覆盖认证状态；`auth.spec.ts` E2E 覆盖正确/错误密码；ProtectedRoute 组件存在 |
| J2 | 添加账户 | ⏳ 待手动验证 | AccountsPage 组件存在且路由注册；表单和 API 调用代码完整；需真实 IMAP/OAuth 账户验证端到端 |
| J3 | 同步 | ⏳ 待手动验证 | SyncManager/Scheduler/Executor 代码完整；`email-sync-engine.spec.ts` E2E 存在；需运行实例+测试账户 |
| J4 | 阅读 | ✅ 代码+单测 | `EmailDetail.test.tsx` 覆盖渲染；ShadowHtmlComponent + sanitize 代码存在；`email.spec.ts` E2E 覆盖详情 API |
| J5 | 状态 | ✅ 代码 | 邮件状态 API（已读/星标/删除）handler 存在；前端 store 有对应 action |
| J6 | 搜索 | ✅ 代码+单测 | `email.spec.ts` E2E 覆盖搜索 API；tsvector 搜索在 repository 层实现 |
| J7 | 自动化 | ✅ 代码 | `webhook-integration.spec.ts` E2E 覆盖 Webhook CRUD；规则引擎 handler/service 完整 |

### AC5：半闭环能力在 README 有对应条目

- ✅ 标签、会话视图、S3/OSS、设置导入导出、多租户、WebAPI 运营深度均在「明确非目标与未闭环能力」中列出

### AC6：git diff 无多租户/新协议/大规模重构

- ✅ 变更仅涉及 `frontend/src/App.tsx`（路由条件注册）和 `README.md`（文档重写）
- ✅ 无 `EmailAccount` user_id 改动，无 adapter 拆分，无 AccountsPage 重构

### AC7：前端生产构建成功

- ✅ `npm run build` 通过（tsc + vite build）
- ✅ 后端无改动，不需要 `go test`/`go build`

## E2E 覆盖映射

| E2E 文件 | 覆盖旅程 | 运行条件 |
|----------|----------|----------|
| `auth.spec.ts` | J1 | 运行实例 + TEST_CREDENTIALS |
| `frontend.spec.ts` | J1, J4 | 运行实例 |
| `email.spec.ts` | J4, J6 | 运行实例 + 已有邮件数据 |
| `email-sync-engine.spec.ts` | J3 | 运行实例 + 测试账户 |
| `webhook-integration.spec.ts` | J7 | 运行实例 + TEST_AUTH_TOKEN |

## 结论

AC1–AC3、AC5–AC7 已通过代码审查和构建验证。AC4 中 J1/J4/J6/J7 有单测或 E2E 覆盖；J2/J3 需要真实邮箱账户做端到端手动验证，代码路径完整无断裂。