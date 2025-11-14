# FusionMail SSE 事件接入方案（Cookie/Polyfill 全量细节）

> 目标：在全局建立一次 SSE 订阅，后端对“计数相关状态变化”（标记已读、星标/取消、归档/恢复、删除/恢复、新邮件等）进行事件推送；前端收到事件后按策略主动拉取统计或直接更新，从而避免在各页面重复拉取并保持状态实时一致。

---

## 1. 总览与设计原则
- 全局一次订阅：在 MainLayout/App 顶层建立一次 SSE，统一处理计数更新。
- 只在“可能有变化”时拉取：后端广播变化信号，前端再决定是否拉取统计（去抖/合并）。
- 首推 Cookie + HttpOnly 鉴权：登录时由后端下发会话 Cookie，SSE/REST 均可复用；避免把 token 放 URL。
- 兼容 polyfill：若受限于现状无法用 Cookie，可用 eventsource-polyfill/fetch-sse 以附带 Authorization 头。
- 渐进演进：初期沿用现有统计（4 个列表接口，仅在事件到达时调用）；后续收敛为单统计接口 `/emails/stats`。

---

## 2. 鉴权方案
### 2.1 Cookie（推荐）
- 登录成功后由后端设置 HttpOnly Cookie，如：
  - 名称：`fm_session`
  - 属性：`HttpOnly; Secure(生产); SameSite=Lax; Path=/; Max-Age=3600`（跨端口仍属同站，端口不影响 SameSite）
- 前端建立 `EventSource(url, { withCredentials: true })`，浏览器会自动携带 Cookie。
- 后端 SSE 响应需设置 CORS：
  - `Access-Control-Allow-Origin: http://localhost:4444`（不要用 `*`）
  - `Access-Control-Allow-Credentials: true`
  - `Vary: Origin`
- REST 接口保持向后兼容：可同时支持 Authorization 头与 Cookie 校验（便于过渡）。

### 2.2 eventsource-polyfill（备选）
- 若不改 Cookie 流程，可使用 polyfill 使 SSE 请求附带自定义头（如 `Authorization: Bearer <token>`）。
- 需服务端允许该头并完成校验；仍需设置 CORS 允许凭据与来源。

---

## 3. 服务端改造
### 3.1 SSE 端点契约
- 路径：`GET /api/v1/events`（可支持 `?topic=email_counts` 等查询参数）
- 响应头：
  - `Content-Type: text/event-stream`
  - `Cache-Control: no-cache`
  - `Connection: keep-alive`
  - CORS 同上（允许凭据，指定来源）
- 心跳：每 20~30s 发送一次注释或 ping 事件，防止中间代理断开。
- 事件格式（示例）：
  ```
  id: 1700000000001
  event: email_counts_maybe_changed
  data: {"source":"mark_as_read","email_id":165}

  ```
  - 空行分隔事件；`id` 便于断线后 Last-Event-ID 恢复。

### 3.2 广播总线（Hub）
- 维护所有在线客户端连接（每连接一个 channel/goroutine 或等价结构）。
- 提供 `broadcast(eventType, payload)`，向所有连接推送。
- 注意：
  - 背压与慢客户端处理（写超时或丢弃策略）。
  - 连接关闭清理资源，避免泄漏。

### 3.3 在业务操作处广播
- 以下操作成功提交后广播一次 `email_counts_maybe_changed`：
  - 标记已读/未读、星标/取消星标、归档/恢复、删除/恢复、新邮件创建/接收。
- 可在 service 层统一封装触发，避免遗漏。

### 3.4 反向代理与基础设施
- Nginx/网关需关闭响应缓冲并保持长连接：
  - `proxy_http_version 1.1;`
  - `proxy_set_header Connection '';`
  - `proxy_buffering off;`
  - `proxy_read_timeout 1h;`（视实际设置）
- 云平台空闲连接超时：通过定时心跳避免被提前回收。

---

## 4. 前端接入
### 4.1 订阅位置与生命周期
- 在 `frontend/src/components/layout/MainLayout.tsx`（或 App 顶层）建立一次订阅。
- 组件卸载时关闭连接；多 Tab 情况下各自维持连接，或使用 BroadcastChannel 复用。

### 4.2 Cookie 方式示例（TypeScript）
```ts
// 建议放在 MainLayout 顶层 useEffect 中
useEffect(() => {
  const url = `${API_BASE}/api/v1/events?topic=email_counts`;
  const es = new EventSource(url, { withCredentials: true });

  let debounceTimer: number | null = null;
  const triggerFetch = () => {
    if (debounceTimer) return;
    debounceTimer = window.setTimeout(async () => {
      try { await emailService.getGlobalStats(); } finally { debounceTimer = null; }
    }, 400);
  };

  es.onopen = () => console.info('SSE connected');
  es.addEventListener('email_counts_maybe_changed', triggerFetch);
  es.onmessage = triggerFetch; // 兜底
  es.onerror = (e) => console.warn('SSE error', e);

  return () => { es.close(); if (debounceTimer) clearTimeout(debounceTimer); };
}, []);
```

### 4.3 Polyfill 方式（仅当需传 Authorization 头）
- 安装：`npm i event-source-polyfill` 或 `yarn add event-source-polyfill`
- 使用：
```ts
import { EventSourcePolyfill } from 'event-source-polyfill';

const es = new EventSourcePolyfill(`${API_BASE}/api/v1/events`, {
  headers: { Authorization: `Bearer ${token}` },
  withCredentials: true,
});
```

### 4.4 事件处理策略
- 去抖 300–500ms：同一时间窗口内多事件合并为一次拉取。
- 首屏：应用启动时仍进行一次初始统计拉取；之后依赖事件触发。
- 乐观更新：例如标记已读时本地 `unreadCount--`，待 SSE 事件/拉取对齐。
- 断线重连：浏览器 EventSource 自带重连；服务端提供 `id`，必要时支持 `Last-Event-ID`。

---

## 5. 统计拉取与接口优化
- 初期：沿用现有统计逻辑（内部并发 4 请求，仅在事件到达时触发）。
- 收敛：新增 `GET /api/v1/emails/stats`，返回 `{ unread, starred, archived, deleted }`，用一次请求替代四次列表。
- 更进一步：变更类接口（标记已读/星标/归档/删除等）直接返回最新计数，前端可直接用响应更新并等待 SSE 最终校准。

---

## 6. 登录与 Cookie 细节
### 6.1 登录设置 Cookie（示例）
- Set-Cookie 响应头示例：
  ```http
  Set-Cookie: fm_session=eyJhbGciOi...; Path=/; HttpOnly; SameSite=Lax; Max-Age=3600; Secure
  ```
- 注意事项：
  - 本地开发可去掉 `Secure`；生产务必启用 HTTPS 与 `Secure`。
  - `SameSite=Lax` 通常足够；如有跨站需求评估 `SameSite=None; Secure`。
  - Domain 建议省略或设为主域，`localhost` 环境不必设置。

### 6.2 服务端校验
- SSE/REST 入口统一从 Cookie 读取并校验 `fm_session`（JWT 或会话 ID）。
- 同时兼容 `Authorization: Bearer` 以利于渐进迁移。

### 6.3 CSRF 说明
- SSE 为 GET 且只读，风险较小；变更类 REST 建议保留现有 CSRF 策略或使用双 Cookie/CSRF Token。

---

## 7. CORS/安全/代理配置清单
- SSE 响应头：
  - `Content-Type: text/event-stream`
  - `Cache-Control: no-cache`
  - `Connection: keep-alive`
  - `Access-Control-Allow-Origin: http://localhost:4444`
  - `Access-Control-Allow-Credentials: true`
  - `Vary: Origin`
- Nginx（片段）：
  ```nginx
  location /api/v1/events {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection '';
    proxy_buffering off;
    proxy_read_timeout 1h;
  }
  ```

---

## 8. 事件模型建议
- 事件类型（推荐最简）：`email_counts_maybe_changed`
  - 语义：“计数可能变化”，前端据此拉取最新统计；无需携带具体数值，避免耦合。
- 可选拓展：
  - `email_created`, `email_deleted`, `email_starred`, `email_archived` ...（特定场景可按需细化）。
- 事件字段：
  - `id`：严格递增或时间戳（便于重放/补偿）
  - `event`：事件类型
  - `data`：JSON 字符串（可含触发来源）

---

## 9. 端到端流程（Cookie 方案）
1. 用户登录 → 后端 `Set-Cookie: fm_session=...; HttpOnly; ...`。
2. 前端启动 → 首次 `getGlobalStats()` 初始化计数。
3. 前端建立 `EventSource(url, { withCredentials: true })`。
4. 用户操作导致变更（如标为已读）→ 后端业务成功 → 广播 `email_counts_maybe_changed`。
5. 前端收到事件（去抖）→ 调用 `getGlobalStats()` → Zustand/Store 更新 → Sidebar 等处即时刷新。
6. 断线自动重连；重连成功后可补一次全量拉取确保一致性。

---

## 10. 迁移计划
- 阶段 A：上线 SSE（Cookie 方案），REST 保持现状；在详情页移除“强制刷新四个统计”的逻辑，改由事件触发。
- 阶段 B：新增 `/emails/stats` 聚合接口，前端由 4 请求切换为 1 请求。
- 阶段 C：在变更接口返回最新计数，前端可直接使用响应乐观更新。
- 阶段 D：根据并发与稳定性评估，是否引入多实例下的集中式消息系统（如 Redis Pub/Sub）做跨进程广播。

---

## 11. 风险与对策
- Token/Cookie 泄露：使用 HttpOnly + Secure + 最小化暴露；生产启用 HTTPS。
- 事件风暴：服务端合并广播（一次变更只发一次），前端去抖。
- 断线与代理中断：服务端心跳 + 客户端自动重连 + 重连后补一次拉取。
- 慢客户端：写超时与断开策略，防止阻塞广播。
- 多 Tab 重复拉取：可通过 BroadcastChannel 协调去抖或共享状态。

---

## 12. 验收清单（手测指引）
- 登录后开发者工具检查 Cookie：存在 `fm_session`（HttpOnly）。
- Network 观察 `/api/v1/events`：
  - 响应头包含 `text/event-stream`、`Access-Control-Allow-Credentials: true`、正确 Origin。
  - 连接保持，周期心跳到达。
- 执行“标为已读” → 触发事件 → 前端仅一次统计拉取，Sidebar 计数正确更新。
- 断开网络 10s 再恢复 → 自动重连 → 收到事件后能恢复正常拉取。

---

## 13. FAQ
- Q: 必须用 Cookie 吗？
  - A: 推荐。原生 EventSource 对自定义头支持差，Cookie + withCredentials 最稳健；若无法调整，可用 polyfill 传递 Authorization。
- Q: 端口不同会影响 Cookie 吗？
  - A: 对 SameSite 判定不影响（端口不计入“站点”），但属于跨源请求，需正确设置 CORS 与 `withCredentials`。
- Q: 需要预防 CSRF 吗？
  - A: SSE 为只读，风险低；变更类 REST 保持/加强 CSRF 防护即可。

—— 完 ——
