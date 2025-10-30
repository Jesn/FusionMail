# FusionMail E2E 自动化测试

## 📋 概述

本目录包含 FusionMail 的端到端（E2E）自动化测试，使用 Playwright 框架。

## 🚀 快速开始

### 前置条件

1. **Node.js** (v18+)
2. **后端服务运行中** (http://localhost:8080)
3. **数据库和 Redis 运行中**

### 安装依赖

```bash
cd tests/e2e
npm install
npx playwright install chromium
```

### 运行测试

**方式 1：使用脚本（推荐）**

```bash
./run-tests.sh
```

**方式 2：手动运行**

```bash
# 运行所有测试
npm test

# 运行特定测试
npm run test:auth      # 认证测试
npm run test:api       # API 测试

# 调试模式
npm run test:debug

# UI 模式
npm run test:ui
```

## 📊 测试清单

测试清单位于：`.kiro/specs/fusionmail/test-checklist.md`

测试会自动更新清单状态：
- `[x]` - 测试通过
- `[!]` - 测试失败
- `[ ]` - 未测试

## 🧪 测试覆盖

### 1. 环境准备测试 (health.spec.ts)
- 后端服务健康检查
- 前端服务检查
- 数据库连接验证
- Redis 连接验证

### 2. 认证与授权测试 (auth.spec.ts)
- 登录功能（正确/错误密码）
- Token 验证
- Token 刷新
- 登出功能
- 未认证访问保护

### 3. API 功能测试 (api.spec.ts)
- 账户管理 API
  - 创建账户
  - 获取账户列表
  - 获取账户详情
  - 手动同步
  - 删除账户
- 邮件管理 API
  - 获取邮件列表
  - 邮件搜索
  - 未读数统计
- 规则引擎 API
  - 创建规则
  - 获取规则列表
  - 启用/禁用规则
  - 删除规则

## 📈 查看测试报告

```bash
# 查看 HTML 报告
npm run test:report

# 查看测试清单
cat ../../.kiro/specs/fusionmail/test-checklist.md
```

## 🔧 配置

### 环境变量

在 `.env` 文件中配置（可选）：

```bash
BASE_URL=http://localhost:8080
API_BASE_URL=http://localhost:8080/api/v1
MASTER_PASSWORD=admin123
```

### Playwright 配置

配置文件：`playwright.config.ts`

- 单线程执行（避免速率限制）
- 失败时自动截图和录屏
- 失败时自动重试（CI 环境）

## 🐛 故障排查

### 后端服务未运行

```bash
cd backend
go run cmd/server/main.go
```

### 端口被占用

检查 8080 端口是否被占用：

```bash
lsof -i :8080
```

### 测试超时

增加超时时间（在 playwright.config.ts 中）：

```typescript
use: {
  timeout: 30000, // 30 秒
}
```

## 📝 添加新测试

1. 在 `tests/` 目录创建新的 `.spec.ts` 文件
2. 使用 `updateChecklistStatus()` 更新测试清单
3. 在测试清单中添加对应的任务项

示例：

```typescript
import { test, expect, updateChecklistStatus } from './setup';

test('测试新功能', async ({ request }) => {
  // 测试逻辑
  expect(true).toBe(true);
  
  // 更新清单
  updateChecklistStatus('X.X 测试新功能', 'completed');
});
```

## 🎯 最佳实践

1. **串行执行**：避免并发导致的速率限制问题
2. **清理数据**：测试后清理创建的测试数据
3. **独立性**：每个测试应该独立，不依赖其他测试
4. **幂等性**：测试可以重复运行
5. **明确断言**：使用清晰的断言消息

## 📚 参考资料

- [Playwright 文档](https://playwright.dev/)
- [FusionMail API 文档](../../docs/)
- [测试清单](../../.kiro/specs/fusionmail/test-checklist.md)
