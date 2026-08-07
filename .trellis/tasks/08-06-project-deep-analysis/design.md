# 设计：统一收件箱 MVP 收口

## 1. 架构边界

本任务 **不改变** 后端模块化单体边界，不新增服务，不改 schema。

变更面：

| 层 | 是否改动 | 内容 |
|----|----------|------|
| 前端路由 | 是 | DEV-only 调试路由 |
| 前端菜单 | 视需要 | 去掉调试/过时入口引用 |
| 根 README | 是 | 能力矩阵与定位 |
| 后端 API | 默认否 | 仅主旅程阻塞 bug 时最小修复 |
| DB / migration | 否 | — |
| 部署拓扑 | 否 | — |

## 2. 产品信息架构（目标态）

### 邮件视图（主）

- 收件箱 / 已发送 / 星标 / 归档 / 垃圾邮件 / 回收站
- 搜索、写邮件
- 按账户/分组筛选（已有）

### 管理视图（辅）

- 账户管理（含可选 WebAPI 账户类型，不单独成「运营台」品牌）
- 规则、Webhook
- 个人设置、系统设置
- 高级：API Key、提供商、OAuth2 客户端、黑白名单

### 生产不可见

- OAuth2 测试页、SSE 调试页
- 设置 legacy 路由不作为导航目标

```
用户心智：
  [邮件] 读信/搜信/状态  ← 北极星
  [管理] 账户/规则/集成  ← 支撑
  [高级] 提供商/OAuth/API ← 可选
  [调试] 仅开发者本地    ← DEV
```

## 3. 调试路由设计

**方案**：条件注册（推荐，与现有 `import.meta.env.DEV` 模式一致）

```tsx
// App.tsx 伪代码
{import.meta.env.DEV && (
  <Route path="/oauth2-test" element={<OAuth2TestPage />} />
)}
{import.meta.env.DEV && (
  <Route path="/debug/sse" element={...SSEDebugPage...} />
)}
```

- 生产构建：路由不存在 → 被 `path="*"` 重定向到 `/inbox`（当前行为）
- 开发构建：行为不变
- 懒加载可选：进一步减小 prod bundle（非必须）

**备选（不采用）**：生产返回 403 专用页——多余；catch-all 已够用。

## 4. README 能力矩阵结构

建议章节：

1. 定位（一段话，A1 北极星）
2. 已支持能力（对齐代码）
3. 可选高级能力（WebAPI 账户、Spam 规则、发送等）
4. 明确非目标（多租户、会话视图、标签产品化、对象存储…）
5. 快速开始（保留 start.sh；Go 版本对齐 go.mod）
6. 文档索引（只链存在的文件；部署见 AGENTS.md）

已实现应包含（证据来自代码，非旧 README）：

- JWT/Cookie 认证、2FA
- 多协议账户与同步
- 收件箱/详情/搜索/状态
- 规则、Webhook、发送、附件
- Spam 检测（作为增强，非主叙事）

## 5. 设置入口

| 路径 | MVP 策略 |
|------|----------|
| `/settings` | 主个人设置 |
| `/settings/system` | 主系统设置 |
| `/settings/legacy` | 保留文件；DEV 注册旧页面；生产同路径重定向到 `/settings`；不作为菜单目标 |
| dashboard/admin/public | 保留代码；无强制生产菜单入口则不动逻辑 |

避免大删页导致未知深链断裂；收口以 **导航与文档** 为主。

## 6. 主旅程验收数据流（不变，仅验证）

```
Login (Cookie JWT)
  → Create Account (credentials encrypted)
  → SyncManager / manual sync
  → Email list + detail (sanitize HTML)
  → Local status update
  → Search (tsvector)
  → Rule engine on ingest OR Webhook config
```

无新契约；若验收发现 API/UI 断裂，再最小修复并记入 implement 增补项。

## 7. 兼容与迁移

- 无 DB migration
- 无 API 破坏性变更预期
- 生产用户书签 `/oauth2-test`：将进入收件箱（可接受）
- 文档消费者：以新 README 为准；旧 `docs/*` 不批量改写

## 8. 回滚

- 路由改动：还原 `App.tsx` 条件即可
- README：git revert 单文件
- 业务 bugfix：按提交粒度回滚

## 9. 权衡

| 选择 | 收益 | 代价 |
|------|------|------|
| 藏调试页不删文件 | 低风险、DEV 仍可用 | 死代码残留 |
| 只改根 README | 对外叙事立刻正确 | 深层 docs 仍旧 |
| 不做 WebAPI 降级删 | 不伤现有用户 | 配置面仍复杂 |
| 手动主旅程清单 | 快 | 无持续 CI 门禁（可后续加） |
