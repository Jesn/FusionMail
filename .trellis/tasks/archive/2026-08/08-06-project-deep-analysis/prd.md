# 路径 A1：统一收件箱 MVP 收口

## Goal

把 FusionMail 收敛为 **可对外解释、主路径可稳定使用的自托管多邮箱聚合阅读产品**。

北极星场景（已确认）：

> 用户在单一 Web 控制台登录后，接入多个邮箱账户，后台同步邮件，在统一收件箱中阅读/搜索/管理状态，并用规则或 Webhook 做轻量自动化。

**用户价值**：半成品感下降——文档可信、导航干净、主旅程可预期；不为运营台或 SaaS 分心。

## Background

| 决策 | 选择 | 日期 |
|------|------|------|
| 工程主路径 | A 产品 MVP 收口 | 2026-08-06 |
| 北极星场景 | A1 统一收件箱 | 2026-08-06 |

分析依据：`research/project-maturity-analysis.md`（成熟度约 3.0/5）。

与 A1 相关的仓库事实：

- 主路径能力代码已存在（认证/2FA、多协议账户、同步、收件箱、详情安全渲染、搜索、规则、Webhook、发送）
- README「待实现」与代码不符；部分文档链接失效（如 `docs/development-progress.md`、`docs/startup-guide.md`）
- 生产可访问调试路由：`/oauth2-test`（无登录）、`/debug/sse`
- 设置路由并存：`/settings`、`/settings/dashboard`、`/admin/settings`、`/public-settings`、`/settings/legacy`
- WebAPI 作为账户来源已实现；本 MVP **保留能力、不升格为产品主叙事**
- 单租户不变；标签/会话/S3/设置导入导出仍为半闭环

## Requirements

### R1. 产品叙事对齐

- 根 `README.md` 能力矩阵以代码为准：已实现标 ✅，未闭环标「未提供 / 规划中」，禁止把已实现标成待办
- 写明产品定位：**自托管多邮箱聚合（只读镜像为主）**；WebAPI/临时邮箱记为可选账户类型
- 修正失效文档链接或改为「见 docs/ 目录 / AGENTS.md 部署说明」
- Go 版本要求与 `go.mod`（≥1.25）一致

### R2. 生产导航卫生

- 生产构建（`import.meta.env.PROD`）下：
  - `/oauth2-test`、`/debug/sse` **不注册路由**，或仅 `DEV` 注册
  - 侧栏/菜单无调试入口
- 开发构建可继续保留调试页
- 直链访问生产调试路径：落到 404 或重定向收件箱（二选一，推荐重定向收件箱以保持现有 catch-all 行为一致）

### R3. 设置入口收敛（低风险）

- 主入口保留：`/settings`（个人）、`/settings/system`（系统，AdminMenu 已有）
- `/settings/legacy`：生产隐藏路由；文档不提
- `/settings/dashboard`、`/admin/settings`、`/public-settings`：若无菜单硬依赖，生产可不挂菜单；不强制删文件
- 不在本任务重做设置信息架构

### R4. 主旅程可用性门禁

下列旅程必须可演示/可验收（环境：本地 `./start.sh` 或已部署实例 + 至少一个可同步测试账户）：

| ID | 旅程 | 可观察结果 |
|----|------|------------|
| J1 | 登录 | 正确密码进入收件箱；错误密码有明确错误 |
| J2 | 添加账户 | IMAP 或 OAuth 之一成功创建账户 |
| J3 | 同步 | 手动同步后列表出现邮件或明确同步状态/错误 |
| J4 | 阅读 | 打开邮件详情；HTML 经 sanitize/Shadow 渲染；附件入口可用（若有附件） |
| J5 | 状态 | 已读/星标/删除至少一项本地状态变更成功 |
| J6 | 搜索 | 关键词搜索返回相关结果或空态 |
| J7 | 自动化 | **规则创建并匹配** 或 **Webhook 配置保存** 二选一即可 |

验收方式：手动检查清单优先；已有 E2E 可复用则优先跑相关用例，不强制新增大套 E2E。

### R5. 半闭环能力策略（文档 + 入口，不实现功能）

| 能力 | MVP 策略 |
|------|----------|
| 邮件标签产品化 | Out of scope；README 标未提供 |
| 会话视图 | Out of scope；README 标未提供 |
| 设置导入/导出 | 保持 501；UI 若暴露则禁用或隐藏 |
| S3/OSS 附件存储 | Out of scope |
| 多用户数据隔离 | Out of scope（单实例管理员模型） |
| WebAPI 子邮箱运营深度 | 保留代码；README 仅作「可选 WebAPI 账户」；不新增运营功能 |

### R6. 仅修主路径阻塞缺陷

实现阶段仅纳入 **验收时复现且阻塞 J1–J7** 的 bug；不主动开启架构大重构。

已知候选（实现前再确认是否仍复现）：

- 无强制清单；以验收时发现为准
- 文档/路由卫生本身不依赖修业务 bug

### R7. 版本与对外信号（轻量）

- README 或 health 文案体现「聚合收件箱 MVP」叙事即可
- 不强制改造构建期 version injection（可 defer）

## Acceptance Criteria

- [ ] AC1：README 无「Webhook/发送/JWT/前端/附件」等已实现项出现在待实现列表
- [ ] AC2：生产前端构建后，访问 `/oauth2-test` 与 `/debug/sse` 不能进入调试 UI
- [ ] AC3：开发构建仍可访问上述调试页（或等价 DEV-only 路径）
- [ ] AC4：J1–J7 主旅程检查清单全部通过并记录结果（任务 notes 或 check 日志）
- [ ] AC5：半闭环能力在 README「明确不提供 / 非目标」中有对应条目
- [ ] AC6：`git diff` 无多租户、无新协议适配器、无无关大规模重构
- [ ] AC7：前端生产构建成功（`npm run build`）；若改动后端则 `go test` 相关包或 `go build ./...` 通过

## Out of Scope

- 多租户 / user_id 归属 / SaaS
- 标签、会话视图完整实现
- S3/OSS、设置导入导出实现
- 全面 React Query 迁移、AccountsPage 大拆分
- Adapter 大规模拆分
- 重写全部 `docs/**`（只保证根 README + 失效链接不误导）
- 性能压测、完整安全审计
- 把 WebAPI 做成独立产品线 UX

## Technical Notes

- 调试路由：`frontend/src/App.tsx` 中 `/oauth2-test`、`/debug/sse`
- 已有 DEV 门控先例：`main.tsx`、`authTest.ts` 使用 `import.meta.env.DEV`
- 邮件安全渲染：`ShadowHtmlComponent` + `sanitize.ts`（保留，不回退）
- 部署说明已在 `AGENTS.md`（Fly.io）；README 可交叉引用
- 详细设计见 `design.md`；执行清单见 `implement.md`

## Risks

| 风险 | 缓解 |
|------|------|
| 主旅程依赖真实邮箱，本地难验 | 用已有测试配置/mock 账户；至少 API+UI 冒烟 |
| 隐藏设置路由导致书签失效 | 保留路由文件，仅去菜单/生产调试；legacy 可 redirect |
| 只改 README 不够 | AC4 强制主旅程门禁 |
| 范围 creep 回 WebAPI 深水区 | Out of scope 明确；发现运营需求另开任务 |

## Open Questions

无阻塞项。

## Artifacts

| 文件 | 状态 |
|------|------|
| `research/project-maturity-analysis.md` | 完成 |
| `prd.md` | 本文件（PRD 收敛后） |
| `design.md` | 完成 |
| `implement.md` | 完成 |
