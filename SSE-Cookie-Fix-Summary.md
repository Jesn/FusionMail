# SSE Cookie 鉴权问题修复总结

## 问题描述

1. **SSE 401 问题**：SSE 使用 Cookie 鉴权，但前端 EventSource 可能未正确发送 Cookie
2. **/emails/stats 请求两次问题**：需要进一步调试，可能是 MainLayout 中的 SSE 连接和其他地方同时调用导致

## 修复内容

### 1. 前端修改

#### 1.1 MainLayout.tsx 增强日志
- 文件：`frontend/src/components/layout/MainLayout.tsx`
- 修改内容：
  - 添加详细的 SSE 连接日志
  - 直接更新 emailStore 而不是通过 service
  - 添加连接状态监控
  - 改进防抖逻辑

主要改进：
```typescript
// 添加详细日志
console.log('[SSE] 准备建立连接:', url);
console.log('[SSE] 连接已建立');
console.log('[SSE] 收到 email_counts_maybe_changed 事件');

// 直接更新 store
const { setUnreadCount, setStarredCount, setArchivedCount, setDeletedCount } = useEmailStore.getState();
setUnreadCount(stats.unread_count);
setStarredCount(stats.starred_count);
setArchivedCount(stats.archived_count);
setDeletedCount(stats.deleted_count);

// 监控连接状态
if (es.readyState === EventSource.CLOSED) {
  console.error('[SSE] 连接已关闭');
} else if (es.readyState === EventSource.CONNECTING) {
  console.log('[SSE] 正在重连...');
}
```

#### 1.2 创建 SSE 调试页面
- 文件：`frontend/src/pages/SSEDebugPage.tsx`
- 功能：
  - 实时监控 SSE 连接状态
  - 显示 Cookie 信息
  - 测试 /emails/stats 接口
  - 显示事件日志
  - 手动控制连接/断开

访问路径：`http://localhost:4444/debug/sse`

### 2. 后端修改

#### 2.1 SSE Handler 增强日志
- 文件：`backend/internal/handler/sse_handler.go`
- 修改内容：
  - 添加详细的认证日志
  - 记录 Cookie 和 Bearer token 的使用情况
  - 记录连接建立和失败的原因

主要改进：
```go
fmt.Printf("[SSE] 收到连接请求 - Origin: %s, Cookie: %v\n", 
    c.GetHeader("Origin"), 
    c.Request.Header.Get("Cookie") != "")

fmt.Printf("[SSE] Cookie 认证失败: %v, 尝试 Bearer 认证\n", err)
fmt.Printf("[SSE] 使用 Cookie token\n")
fmt.Printf("[SSE] Token 验证失败: %v\n", err)
fmt.Printf("[SSE] 认证成功，建立连接\n")
```

### 3. 测试工具

#### 3.1 Playwright E2E 测试
- 文件：`tests/e2e/sse-cookie-test.spec.ts`
- 测试内容：
  1. 验证 SSE 连接能够正确使用 Cookie 鉴权
  2. 验证 /emails/stats 接口不会被重复调用
  3. 验证 SSE 事件能够正确触发统计更新
  4. 检查 Cookie 是否正确设置
  5. 验证 axios withCredentials 配置

#### 3.2 手动测试脚本
- 文件：`scripts/test-sse-manual.sh`
- 功能：
  1. 登录获取 Cookie
  2. 使用 curl 测试 SSE 连接
  3. 验证 Cookie 是否正确传递

#### 3.3 自动化测试脚本
- 文件：`scripts/test-sse.sh`
- 功能：
  1. 检查前后端服务状态
  2. 安装 Playwright（如果需要）
  3. 运行 E2E 测试
  4. 显示测试报告

## 使用方法

### 方法 1：使用 SSE 调试页面（推荐）

1. 启动前后端服务
2. 登录系统
3. 访问 `http://localhost:4444/debug/sse`
4. 点击"检查 Cookie"确认 fm_session 存在
5. 点击"连接 SSE"建立连接
6. 观察日志输出

### 方法 2：使用手动测试脚本

```bash
# 给脚本添加执行权限
chmod +x scripts/test-sse-manual.sh

# 运行测试
./scripts/test-sse-manual.sh
```

### 方法 3：使用 Playwright 测试

```bash
# 给脚本添加执行权限
chmod +x scripts/test-sse.sh

# 运行测试
./scripts/test-sse.sh
```

或者直接运行：

```bash
# 安装依赖
npm install -D @playwright/test
npx playwright install chromium

# 运行测试
npx playwright test tests/e2e/sse-cookie-test.spec.ts --headed
```

## 调试步骤

### 1. 检查 Cookie 是否正确设置

在浏览器开发者工具中：
1. 打开 Application/Storage -> Cookies
2. 查找 `fm_session` Cookie
3. 确认属性：
   - HttpOnly: true
   - Path: /
   - SameSite: Lax

### 2. 检查 SSE 连接请求

在浏览器开发者工具的 Network 标签中：
1. 找到 `/api/v1/events` 请求
2. 查看 Request Headers
3. 确认 Cookie 头存在且包含 fm_session

### 3. 检查后端日志

启动后端时查看控制台输出：
```
[SSE] 收到连接请求 - Origin: http://localhost:4444, Cookie: true
[SSE] 使用 Cookie token
[SSE] 认证成功，建立连接
```

### 4. 检查 /emails/stats 请求次数

在浏览器开发者工具的 Network 标签中：
1. 过滤 `/emails/stats` 请求
2. 观察请求次数和时间间隔
3. 正常情况下应该只有 1-2 次请求（冷启动 + SSE open 事件）

## 常见问题

### Q1: SSE 连接返回 401

**可能原因：**
1. Cookie 未正确设置
2. Cookie 已过期
3. CORS 配置问题

**解决方法：**
1. 检查登录是否成功
2. 检查 Cookie 是否存在
3. 检查后端 CORS 配置是否允许 credentials

### Q2: /emails/stats 被调用多次

**可能原因：**
1. SSE 连接和其他地方同时调用
2. 防抖未生效
3. 多个组件重复订阅

**解决方法：**
1. 检查 MainLayout 的防抖逻辑
2. 确保只在 MainLayout 中建立一次 SSE 连接
3. 使用 SSE 调试页面监控请求

### Q3: SSE 事件未触发统计更新

**可能原因：**
1. SSE 连接未建立
2. 后端未广播事件
3. 前端事件监听器未正确设置

**解决方法：**
1. 检查 SSE 连接状态
2. 检查后端是否调用了 `sse.Broadcast()`
3. 使用 SSE 调试页面查看事件日志

## 验证清单

- [ ] 登录后 fm_session Cookie 正确设置
- [ ] SSE 连接成功建立（状态码 200）
- [ ] SSE 连接请求携带 Cookie
- [ ] 后端日志显示"认证成功，建立连接"
- [ ] /emails/stats 初始加载时只调用 1-2 次
- [ ] 标记邮件为已读后，SSE 事件触发统计更新
- [ ] 统计数据正确更新到 UI

## 下一步

1. 运行测试验证修复效果
2. 如果仍有问题，使用 SSE 调试页面进行详细排查
3. 检查浏览器控制台和后端日志
4. 根据日志信息进一步调试

