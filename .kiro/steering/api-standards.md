---
inclusion: conditional
fileMatchPattern: "**/{handler,api,service}/**/*"
---

# FusionMail API 规范

## API 设计原则

### RESTful 设计
- 使用标准 HTTP 方法（GET、POST、PUT、PATCH、DELETE）
- 资源使用名词复数形式
- 使用 HTTP 状态码表示结果
- 支持资源嵌套和关联

### 版本控制
- API 版本通过 URL 路径指定：`/api/v1/`
- 主版本号变更表示不兼容的 API 变更
- 保持向后兼容性，废弃的 API 提前通知

### 统一响应格式
所有 API 响应使用统一的 JSON 格式

## 基础配置

### Base URL
```
开发环境：http://localhost:8080/api/v1
生产环境：https://api.fusionmail.com/v1
```

### 请求头
```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <jwt_token>
或
X-API-Key: <api_key>
```

## 响应格式规范

### 成功响应

**单个资源**：
```json
{
  "success": true,
  "data": {
    "id": 123,
    "email": "user@example.com",
    "created_at": "2025-10-27T10:00:00Z"
  },
  "meta": {
    "timestamp": "2025-10-27T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

**资源列表（带分页）**：
```json
{
  "success": true,
  "data": [
    { "id": 1, "subject": "Email 1" },
    { "id": 2, "subject": "Email 2" }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  },
  "meta": {
    "timestamp": "2025-10-27T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

**无数据响应**：
```json
{
  "success": true,
  "message": "操作成功",
  "meta": {
    "timestamp": "2025-10-27T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

### 错误响应

**标准错误格式**：
```json
{
  "success": false,
  "error": {
    "code": "INVALID_INPUT",
    "message": "邮箱地址格式不正确",
    "details": {
      "field": "email",
      "value": "invalid-email"
    }
  },
  "meta": {
    "timestamp": "2025-10-27T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

**验证错误（多个字段）**：
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "请求参数验证失败",
    "details": {
      "errors": [
        {
          "field": "email",
          "message": "邮箱地址不能为空"
        },
        {
          "field": "password",
          "message": "密码长度至少 8 个字符"
        }
      ]
    }
  },
  "meta": {
    "timestamp": "2025-10-27T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

## HTTP 状态码规范

### 成功状态码（2xx）
- **200 OK**：请求成功（GET、PUT、PATCH）
- **201 Created**：资源创建成功（POST）
- **204 No Content**：请求成功但无返回内容（DELETE）

### 客户端错误（4xx）
- **400 Bad Request**：请求参数错误或验证失败
- **401 Unauthorized**：未认证或认证失败
- **403 Forbidden**：无权限访问资源
- **404 Not Found**：资源不存在
- **409 Conflict**：资源冲突（如重复创建）
- **422 Unprocessable Entity**：请求格式正确但语义错误
- **429 Too Many Requests**：请求频率超限

### 服务器错误（5xx）
- **500 Internal Server Error**：服务器内部错误
- **502 Bad Gateway**：网关错误
- **503 Service Unavailable**：服务暂时不可用

## 错误码规范

### 通用错误码
| 错误码 | HTTP 状态 | 说明 |
|-------|----------|------|
| `INVALID_INPUT` | 400 | 输入参数无效 |
| `VALIDATION_ERROR` | 400 | 参数验证失败 |
| `UNAUTHORIZED` | 401 | 未认证 |
| `FORBIDDEN` | 403 | 无权限 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `CONFLICT` | 409 | 资源冲突 |
| `RATE_LIMIT_EXCEEDED` | 429 | 请求频率超限 |
| `INTERNAL_ERROR` | 500 | 服务器内部错误 |

### 业务错误码
| 错误码 | HTTP 状态 | 说明 |
|-------|----------|------|
| `ACCOUNT_NOT_FOUND` | 404 | 邮箱账户不存在 |
| `ACCOUNT_ALREADY_EXISTS` | 409 | 邮箱账户已存在 |
| `CONNECTION_FAILED` | 400 | 邮箱连接失败 |
| `SYNC_IN_PROGRESS` | 409 | 同步正在进行中 |
| `EMAIL_NOT_FOUND` | 404 | 邮件不存在 |
| `RULE_NOT_FOUND` | 404 | 规则不存在 |
| `WEBHOOK_NOT_FOUND` | 404 | Webhook 不存在 |
| `INVALID_CREDENTIALS` | 401 | 凭证无效 |
| `ENCRYPTION_ERROR` | 500 | 加密/解密失败 |

## 分页规范

### 请求参数
```
GET /api/v1/emails?page=1&page_size=20&sort=sent_at&order=desc
```

**参数说明**：
- `page`：页码，从 1 开始（默认：1）
- `page_size`：每页数量（默认：20，最大：100）
- `sort`：排序字段（默认：created_at）
- `order`：排序方向，`asc` 或 `desc`（默认：desc）

### 响应格式
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false
  }
}
```

## 过滤和搜索规范

### 过滤参数
```
GET /api/v1/emails?account_uid=xxx&is_read=false&has_attachments=true
```

**常用过滤参数**：
- `account_uid`：按账户过滤
- `is_read`：按已读状态过滤
- `is_starred`：按星标状态过滤
- `has_attachments`：按是否有附件过滤
- `from`：按发件人过滤
- `date_from`：起始日期（ISO 8601 格式）
- `date_to`：结束日期（ISO 8601 格式）

### 搜索参数
```
GET /api/v1/emails/search?q=关键词&fields=subject,body
```

**参数说明**：
- `q`：搜索关键词
- `fields`：搜索字段（逗号分隔）

## 认证规范

### JWT 认证

**登录获取 Token**：
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "password123"
}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 86400,
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

**使用 Token**：
```http
GET /api/v1/emails
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### API Key 认证

**使用 API Key**：
```http
GET /api/v1/emails
X-API-Key: sk_live_abc123def456
```

## 批量操作规范

### 批量更新
```http
POST /api/v1/emails/batch
Content-Type: application/json

{
  "action": "mark_read",
  "email_ids": [1, 2, 3, 4, 5]
}
```

**支持的操作**：
- `mark_read`：标记为已读
- `mark_unread`：标记为未读
- `star`：添加星标
- `unstar`：取消星标
- `archive`：归档
- `delete`：删除

**响应**：
```json
{
  "success": true,
  "data": {
    "success_count": 5,
    "failed_count": 0,
    "failed_ids": []
  }
}
```

## 时间格式规范

### ISO 8601 格式
所有时间字段使用 ISO 8601 格式（UTC 时区）：
```
2025-10-27T10:00:00Z
```

### 时区处理
- 服务器存储和返回 UTC 时间
- 客户端负责转换为本地时区
- 时间戳字段命名：`created_at`、`updated_at`、`sent_at`

## 速率限制规范

### 限制策略
- **JWT 认证**：每分钟 100 次请求
- **API Key 认证**：根据配置的 `rate_limit` 字段
- **未认证请求**：每分钟 10 次请求

### 响应头
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1635340800
```

### 超限响应
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "请求频率超限，请稍后再试",
    "details": {
      "limit": 100,
      "reset_at": "2025-10-27T10:01:00Z"
    }
  }
}
```

## API 端点命名规范

### 资源命名
- 使用名词复数：`/emails`、`/accounts`、`/rules`
- 使用小写字母和连字符：`/sync-logs`
- 避免动词：使用 HTTP 方法表示动作

### 嵌套资源
```
GET /api/v1/accounts/:uid/emails          # 获取账户的邮件
GET /api/v1/emails/:id/attachments        # 获取邮件的附件
```

### 特殊操作
使用动词表示非 CRUD 操作：
```
POST /api/v1/accounts/:uid/test           # 测试连接
POST /api/v1/accounts/:uid/sync           # 手动同步
POST /api/v1/webhooks/:id/test            # 测试 Webhook
```

## 请求示例

### 创建邮箱账户
```http
POST /api/v1/accounts
Content-Type: application/json
Authorization: Bearer <token>

{
  "email": "user@gmail.com",
  "provider": "gmail",
  "auth_type": "oauth2",
  "credentials": {
    "access_token": "xxx",
    "refresh_token": "yyy"
  },
  "sync_enabled": true,
  "sync_interval": 5
}
```

### 获取邮件列表
```http
GET /api/v1/emails?page=1&page_size=20&is_read=false&sort=sent_at&order=desc
Authorization: Bearer <token>
```

### 更新邮件状态
```http
PATCH /api/v1/emails/123/read
Authorization: Bearer <token>

{
  "is_read": true
}
```

### 搜索邮件
```http
POST /api/v1/emails/search
Content-Type: application/json
Authorization: Bearer <token>

{
  "query": "重要",
  "filters": {
    "account_uid": "xxx",
    "has_attachments": true,
    "date_from": "2025-10-01T00:00:00Z"
  },
  "page": 1,
  "page_size": 20
}
```

## 最佳实践

### 1. 幂等性
- GET、PUT、DELETE 操作应该是幂等的
- POST 操作可以使用幂等性键避免重复创建

### 2. 缓存控制
```http
Cache-Control: no-cache, no-store, must-revalidate
ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4"
```

### 3. 压缩
支持 gzip 压缩：
```http
Accept-Encoding: gzip, deflate
Content-Encoding: gzip
```

### 4. CORS
支持跨域请求：
```http
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization, X-API-Key
```

### 5. 请求 ID
每个请求返回唯一的 `request_id`，便于追踪和调试：
```json
{
  "meta": {
    "request_id": "req_abc123"
  }
}
```

### 6. 软删除
删除操作使用软删除，保留数据：
```http
DELETE /api/v1/emails/123
```
实际上是更新 `deleted_at` 字段，而非物理删除。

---

**注意**：所有 API 实现都应该遵循本规范。在添加新的 API 端点时，请参考现有端点的实现方式，保持一致性。
