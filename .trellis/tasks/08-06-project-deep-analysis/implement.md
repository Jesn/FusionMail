# 实现清单：统一收件箱 MVP 收口

> 批准本规划摘要后，再执行 `task.py start` 与代码修改。

## 0. 开始前

- [x] 用户已明确批准最新规划摘要
- [x] `python3 ./.trellis/scripts/task.py start 08-06-project-deep-analysis`（或当前任务目录名）
- [x] 阅读 `prd.md` + `design.md` + `research/project-maturity-analysis.md`

## 1. 生产调试路由隔离

**文件**：`frontend/src/App.tsx`（必要时 `routeUtils.ts` / 测试）

- [x] `/oauth2-test` 仅 `import.meta.env.DEV` 注册
- [x] `/debug/sse` 仅 DEV 注册
- [x] 检查菜单/Header 是否硬链调试页；有则 DEV-only 或删除链接（确认：AdminMenu/Header 无调试页链接）
- [x] 更新 `frontend/src/utils/routeUtils.test.ts` 若断言依赖生产可访问调试路由（确认：测试未断言调试路由，无需改动）

**验证**：

- [x] `npm run build` 通过
- [x] 生产构建中 `import.meta.env.DEV` 静态替换为 `false`，调试路由不注册

## 2. README 收口

**文件**：`README.md`

- [x] 重写「核心功能」：已实现 / 可选高级 / 非目标
- [x] 删除错误「待实现」：Webhook、发送、JWT、前端、附件等
- [x] 定位段落实 A1 统一收件箱 + 只读镜像
- [x] WebAPI 写成可选账户类型，非第二产品
- [x] Go 版本 ≥1.25；Node 与 frontend/package.json engines 对齐（≥20.19.0）
- [x] 文档链接：去掉不存在的 `docs/development-progress.md`、`docs/startup-guide.md`；部署指向 `AGENTS.md`
- [x] 保留快速开始 `./start.sh` 与安全提示

**验证**：

- [x] 人工通读 README；所有相对链接目标文件存在

## 3. 设置/导航轻量清理

- [x] `/settings/legacy`：DEV 环境保留旧页面；生产环境同路径重定向到 `/settings`，不进入任何菜单
- [x] 确认 AdminMenu / 其他菜单无指向 `/settings/legacy` 的主入口（确认：无菜单引用）
- [x] 不删除页面文件

## 4. 主旅程验收（J1–J7）

| ID | 步骤 | 通过标准 | 结果 |
|----|------|----------|------|
| J1 | 登录 | 进收件箱 / 错误提示正确 | ✅ 单测+E2E 覆盖 |
| J2 | 添加账户 | 账户出现在列表 | ⏳ 待手动验证（需真实账户） |
| J3 | 手动同步 | 有邮件或明确状态 | ⏳ 待手动验证（需运行实例） |
| J4 | 打开详情 | HTML 正常、无脚本弹窗 | ✅ 单测覆盖 |
| J5 | 已读或星标或删除 | 刷新后状态保持 | ✅ 代码完整 |
| J6 | 搜索 | 结果或空态合理 | ✅ E2E 覆盖 |
| J7 | 规则或 Webhook | 保存成功；规则尽量触发一次 | ✅ E2E 覆盖 |

- [x] 将结果记入 `research/mvp-journey-check.md`
- [x] 阻塞项：无；J2/J3 非阻塞，需运行实例手动验证

## 5. 构建与回归

- [x] `npm run build` 通过
- [x] `npm test` 通过（8 文件 36 用例）
- [x] 后端无改动，不需要 `go test`/`go build`
- [x] AC7 满足

## 6. 收尾

- [x] 对照 `prd.md` Acceptance Criteria 全部核对（见 `research/mvp-journey-check.md`）
- [ ] 提交信息中文 conventional：`docs:` / `fix:` / `chore:` 按改动拆分或单提交说明
- [x] **不要**在本任务做完后擅自开多租户/标签/WebAPI 深潜

## 风险文件

| 文件 | 风险 | 回滚 |
|------|------|------|
| `frontend/src/App.tsx` | 误伤 OAuth 正式回调路由 | 勿把 `/auth/*/callback` 放进 DEV-only |
| `README.md` | 写错能力 | 对照 handler/页面列表 |
| 业务 bugfix 文件 | 回归 | 小步提交 |

## 明确不做（实现时自检）

- 不改 `EmailAccount` 加 user_id
- 不实现 label/thread UI
- 不实现 S3/设置导入导出
- 不重写 AccountsPage / sync 核心
- 不批量改 `docs/**` 历史文

## 完成定义

满足 `prd.md` AC1–AC7，且 `implement.md` 第 1–5 节勾选完毕，即可视为路径 A1 MVP 收口完成（分析+收口任务闭环）。
