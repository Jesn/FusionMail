# 账户状态管理功能

## 功能概述

新增了账户状态管理功能，允许用户启用或禁用邮箱账户，并在前端界面中清晰显示账户的当前状态。

## 功能特性

### 后端功能

1. **账户状态字段**
   - 在 `Account` 模型中添加了 `status` 字段
   - 支持三种状态：`active`（正常）、`disabled`（已禁用）、`error`（错误）
   - 默认状态为 `active`

2. **API 端点**
   - `POST /api/v1/accounts/:uid/disable` - 禁用账户
   - `POST /api/v1/accounts/:uid/enable` - 启用账户
   - 现有的账户列表和详情 API 会返回状态信息

3. **数据库迁移**
   - 创建了迁移文件 `002_add_account_status.sql`
   - 为现有账户设置默认状态为 `active`

### 前端功能

1. **状态显示**
   - 在账户卡片中显示账户状态
   - 使用不同颜色的徽章区分状态：
     - 正常：默认样式（蓝色）
     - 已禁用：红色徽章
     - 错误：红色徽章

2. **状态图标**
   - 正常状态：绿色勾选图标
   - 禁用/错误状态：红色/橙色叉号图标

3. **状态切换**
   - 添加了启用/禁用按钮（电源图标）
   - 按钮颜色根据当前状态变化：
     - 正常状态时显示橙色（禁用按钮）
     - 禁用状态时显示绿色（启用按钮）

4. **用户反馈**
   - 状态切换时显示成功提示
   - 操作失败时显示错误提示

## 使用方法

### 前端操作

1. 进入账户管理页面
2. 在账户卡片中查看当前状态
3. 点击电源按钮切换账户状态
4. 观察状态徽章和图标的变化

### API 调用

```bash
# 禁用账户
curl -X POST http://localhost:8080/api/v1/accounts/{uid}/disable

# 启用账户
curl -X POST http://localhost:8080/api/v1/accounts/{uid}/enable

# 查看账户状态
curl http://localhost:8080/api/v1/accounts/{uid}
```

## 测试

### 自动化测试

1. **后端测试脚本**
   ```bash
   ./test-account-status.sh
   ```

2. **前端 E2E 测试**
   ```bash
   npm run test:e2e -- account-status-display.spec.ts
   ```

### 手动测试

1. 启动后端和前端服务
2. 添加一个邮箱账户
3. 在账户页面验证状态显示
4. 测试启用/禁用功能
5. 验证状态变化和用户反馈

## 技术实现

### 后端修改

- `backend/internal/model/account.go` - 添加 status 字段
- `backend/internal/service/account_service.go` - 添加状态管理方法
- `backend/internal/handler/account_handler.go` - 添加 API 处理器
- `backend/internal/router/router.go` - 添加路由配置
- `backend/migrations/002_add_account_status.sql` - 数据库迁移

### 前端修改

- `frontend/src/types/index.ts` - 账户类型已包含 status 字段
- `frontend/src/components/account/AccountCard.tsx` - 添加状态显示和切换按钮
- `frontend/src/services/accountService.ts` - 添加启用/禁用 API 调用
- `frontend/src/hooks/useAccounts.ts` - 添加状态切换逻辑
- `frontend/src/pages/AccountsPage.tsx` - 集成状态切换功能

## 注意事项

1. **数据库迁移**
   - 需要运行迁移脚本来添加 status 字段
   - 现有账户会自动设置为 `active` 状态

2. **向后兼容**
   - 新字段有默认值，不会影响现有功能
   - API 响应格式保持兼容

3. **权限控制**
   - 状态切换需要认证
   - 遵循现有的权限控制机制

4. **同步行为**
   - 禁用的账户可能不会参与自动同步
   - 具体同步逻辑需要在同步引擎中实现

## 未来扩展

1. **批量操作**
   - 支持批量启用/禁用多个账户

2. **状态历史**
   - 记录状态变更历史和原因

3. **自动状态管理**
   - 根据连接错误自动设置为错误状态
   - 连接恢复后自动恢复正常状态

4. **更多状态类型**
   - 添加更细粒度的状态分类
   - 如：同步中、配置错误、认证失败等