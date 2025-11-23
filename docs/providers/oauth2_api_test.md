# OAuth2 客户端管理 API 测试文档

## 概述

本文档提供 OAuth2 客户端配置管理功能的完整 API 测试用例。

## 基础信息

- **服务地址**: http://localhost:3333
- **API 前缀**: `/api/v1/oauth2/clients`
- **认证方式**: JWT Token（需要登录）

## API 端点列表

### 1. 创建 OAuth2 客户端配置

**端点**: `POST /api/v1/oauth2/clients`

**请求体**:
```json
{
  "provider_name": "gmail",
  "name": "生产环境配置",
  "client_id": "your-gmail-client-id",
  "client_secret": "your-client-secret",
  "redirect_uri": "http://localhost:3333/api/v1/auth/google/callback",
  "quota_daily": 100,
  "quota_monthly": 2000,
  "metadata": "{\"environment\": \"production\"}"
}
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "provider_name": "gmail",
    "name": "生产环境配置",
    "client_id": "your-gmail-client-id",
    "redirect_uri": "http://localhost:3333/api/v1/auth/google/callback",
    "enabled": true,
    "is_default": false,
    "usage_count": 0,
    "quota_daily": 100,
    "quota_monthly": 2000,
    "last_used_at": null,
    "metadata": "{\"environment\": \"production\"}",
    "created_at": "2025-11-22T00:00:00Z",
    "updated_at": "2025-11-22T00:00:00Z"
  }
}
```

### 2. 获取 OAuth2 客户端列表（分页）

**端点**: `GET /api/v1/oauth2/clients?page=1&page_size=20`

**查询参数**:
- `page`: 页码（默认1）
- `page_size`: 每页数量（默认20）

**响应示例**:
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "provider_name": "gmail",
      "name": "默认配置",
      "client_id": "your-gmail-client-id",
      "enabled": true,
      "is_default": true,
      "usage_count": 5,
      "quota_daily": 100,
      "quota_monthly": 2000,
      "created_at": "2025-11-22T00:00:00Z",
      "updated_at": "2025-11-22T00:00:00Z"
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 20,
  "total_page": 1
}
```

### 3. 获取指定提供商的客户端

**端点**: `GET /api/v1/oauth2/clients/provider/{provider_name}`

**示例**: `GET /api/v1/oauth2/clients/provider/gmail`

**响应示例**:
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "provider_name": "gmail",
      "name": "默认配置",
      "client_id": "your-gmail-client-id",
      "enabled": true,
      "is_default": true,
      "usage_count": 5
    },
    {
      "id": 2,
      "provider_name": "gmail",
      "name": "生产环境",
      "client_id": "prod-client-id",
      "enabled": true,
      "is_default": false,
      "usage_count": 10
    }
  ]
}
```

### 4. 获取默认客户端

**端点**: `GET /api/v1/oauth2/clients/provider/{provider_name}/default`

**示例**: `GET /api/v1/oauth2/clients/provider/gmail/default`

**响应示例**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "provider_name": "gmail",
    "name": "默认配置",
    "client_id": "your-gmail-client-id",
    "enabled": true,
    "is_default": true,
    "usage_count": 5,
    "quota_daily": 100,
    "quota_monthly": 2000
  }
}
```

### 5. 智能选择客户端

**端点**: `GET /api/v1/oauth2/clients/smart-select/{provider_name}`

**查询参数**:
- `client_id`: 可选，指定客户端ID

**示例1 - 自动选择**:
```
GET /api/v1/oauth2/clients/smart-select/gmail
```

**示例2 - 指定客户端**:
```
GET /api/v1/oauth2/clients/smart-select/gmail?client_id=1
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "provider_name": "gmail",
    "name": "默认配置",
    "client_id": "your-gmail-client-id",
    "enabled": true,
    "is_default": true,
    "usage_count": 6,
    "quota_daily": 100,
    "quota_monthly": 2000,
    "last_used_at": "2025-11-22T00:05:00Z"
  }
}
```

**智能选择逻辑**:
1. 如果指定了 `client_id`，优先使用指定的客户端
2. 否则尝试使用该提供商的默认客户端
3. 如果默认客户端不可用，选择第一个可用的客户端
4. 如果都不可用，返回错误

### 6. 设置默认客户端

**端点**: `POST /api/v1/oauth2/clients/{id}/default/{provider_name}`

**示例**: `POST /api/v1/oauth2/clients/2/default/gmail`

**响应示例**:
```json
{
  "success": true,
  "data": {
    "message": "Default client set successfully"
  }
}
```

### 7. 更新客户端配置

**端点**: `PUT /api/v1/oauth2/clients/{id}`

**请求体**:
```json
{
  "name": "新的名称",
  "client_id": "new-client-id",
  "enabled": true,
  "quota_daily": 200,
  "quota_monthly": 3000
}
```

### 8. 删除客户端配置

**端点**: `DELETE /api/v1/oauth2/clients/{id}`

**响应示例**:
```json
{
  "success": true,
  "data": {
    "message": "OAuth2 client deleted successfully"
  }
}
```

## 测试用例

### 用例1: 创建 Gmail 客户端配置

```bash
curl -X POST http://localhost:3333/api/v1/oauth2/clients \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{
    "provider_name": "gmail",
    "name": "生产环境",
    "client_id": "your-client-id",
    "client_secret": "your-secret",
    "redirect_uri": "http://localhost:3333/api/v1/auth/google/callback",
    "quota_daily": 100,
    "quota_monthly": 2000
  }'
```

### 用例2: 列出所有 Gmail 客户端

```bash
curl http://localhost:3333/api/v1/oauth2/clients/provider/gmail \
  -H "Authorization: Bearer <your-jwt-token>"
```

### 用例3: 智能选择客户端

```bash
curl "http://localhost:3333/api/v1/oauth2/clients/smart-select/gmail" \
  -H "Authorization: Bearer <your-jwt-token>"
```

### 用例4: 查看使用统计

```bash
curl http://localhost:3333/api/v1/oauth2/clients \
  -H "Authorization: Bearer <your-jwt-token>"
```

观察 `usage_count` 和 `last_used_at` 字段的变化。

## 前端集成测试

### 步骤1: 访问账户页面

1. 打开浏览器访问: http://localhost:3333
2. 登录系统
3. 导航到"账户"页面

### 步骤2: 添加账户

1. 点击"添加账户"按钮
2. 选择 Gmail 提供商
3. 选择 OAuth2 协议
4. 验证是否显示"OAuth2 客户端配置"选择器
5. 查看选项:
   - "使用智能选择（自动）"
   - 具体的客户端配置列表（名称、用途、配额信息）

### 步骤3: 验证选择器功能

1. 选择"智能选择"选项
2. 点击"通过 OAuth2 登录"按钮
3. 验证系统使用智能选择算法选择客户端

### 步骤4: 测试前端 API 服务

检查前端是否正确调用 API 服务:

```typescript
import { oauth2ClientService } from '@/services/oauth2ClientService';

// 获取 Gmail 提供商的客户端列表
const clients = await oauth2ClientService.getByProvider('gmail');

// 智能选择客户端
const selected = await oauth2ClientService.smartSelect('gmail');
```

## 错误处理

### 常见错误及解决方案

1. **401 Unauthorized**
   - 原因: 未登录或 token 过期
   - 解决: 重新登录获取新 token

2. **404 Not Found**
   - 原因: 客户端 ID 不存在
   - 解决: 检查 ID 是否正确

3. **400 Bad Request**
   - 原因: 请求参数错误
   - 解决: 验证请求体格式

4. **429 Too Many Requests**
   - 原因: 配额超限
   - 解决: 等待配额重置或联系管理员增加配额

## 性能优化建议

1. **缓存策略**: 客户端配置不经常变化，可以在前端缓存 5-10 分钟
2. **懒加载**: 仅在 OAuth2 协议下加载客户端配置
3. **错误重试**: 网络失败时自动重试 3 次
4. **加载状态**: 显示加载指示器，提升用户体验

## 监控指标

建议监控以下指标:

1. **客户端使用率**: `usage_count` 增长趋势
2. **配额使用情况**: 日/月配额使用百分比
3. **智能选择成功率**: 成功选择与失败的比率
4. **API 响应时间**: 平均响应时间 < 200ms
5. **错误率**: 4xx/5xx 错误占比 < 1%

## 注意事项

1. **安全性**:
   - `client_secret` 已加密存储，API 返回时不包含
   - 所有 API 都需要认证

2. **配额管理**:
   - `quota_daily` 和 `quota_monthly` 设置为 -1 表示无限制
   - 使用 `SmartSelect` 时会自动更新 `usage_count`

3. **默认客户端**:
   - 每个提供商只能有一个默认客户端
   - 设置新的默认客户端会取消之前的默认标记

4. **删除限制**:
   - 不能删除默认客户端（需要先设置其他客户端为默认）
   - 删除操作不可恢复

## 扩展开发

未来可考虑的扩展功能:

1. **批量操作**: 批量启用/禁用客户端
2. **使用统计**: 按时间范围查看使用情况图表
3. **告警通知**: 配额接近上限时发送通知
4. **客户端模板**: 预定义常用配置模板
5. **导入导出**: 配置文件格式的导入导出
6. **版本管理**: 客户端配置的版本历史
