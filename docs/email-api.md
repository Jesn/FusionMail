# 邮件管理 API 文档

## 概述

邮件管理 API 提供了完整的邮件查询、搜索和状态管理功能。所有邮件状态变更（已读、星标、归档、删除）仅在本地生效，不会同步到源邮箱服务器（只读镜像模式）。

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`
- **认证**: 暂未实现（后续版本添加）

---

## 邮件查询接口

### 1. 获取邮件列表

获取邮件列表，支持分页、筛选和排序。

**请求**

```http
GET /api/v1/emails
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| account_uid | string | 否 | 账户 UID，筛选指定账户的邮件 |
| is_read | boolean | 否 | 是否已读（true/false） |
| is_starred | boolean | 否 | 是否星标（true/false） |
| is_archived | boolean | 否 | 是否归档（true/false） |
| from_address | string | 否 | 发件人地址（模糊匹配） |
| subject | string | 否 | 主题（模糊匹配） |
| start_date | string | 否 | 开始日期（YYYY-MM-DD） |
| end_date | string | 否 | 结束日期（YYYY-MM-DD） |
| page | integer | 否 | 页码（默认 1） |
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |

**响应示例**

```json
{
  "emails": [
    {
      "id": 1,
      "provider_id": "18f2c3d4e5f6g7h8",
      "account_uid": "acc_1234567890",
      "subject": "欢迎使用 FusionMail",
      "from_address": "noreply@fusionmail.com",
      "from_name": "FusionMail Team",
      "to_addresses": "[\"user@example.com\"]",
      "snippet": "感谢您使用 FusionMail...",
      "is_read": false,
      "is_starred": false,
      "is_archived": false,
      "has_attachments": false,
      "sent_at": "2025-10-29T10:30:00Z",
      "created_at": "2025-10-29T10:35:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "page_size": 20,
  "total_pages": 8
}
```

---

### 2. 获取邮件详情

根据 ID 获取邮件的完整信息，包括附件列表。

**请求**

```http
GET /api/v1/emails/:id
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 邮件 ID |

**响应示例**

```json
{
  "id": 1,
  "provider_id": "18f2c3d4e5f6g7h8",
  "account_uid": "acc_1234567890",
  "subject": "欢迎使用 FusionMail",
  "from_address": "noreply@fusionmail.com",
  "from_name": "FusionMail Team",
  "to_addresses": "[\"user@example.com\"]",
  "text_body": "感谢您使用 FusionMail，这是一款轻量级邮件聚合系统...",
  "html_body": "<html><body>...</body></html>",
  "snippet": "感谢您使用 FusionMail...",
  "is_read": false,
  "is_starred": false,
  "is_archived": false,
  "is_deleted": false,
  "has_attachments": true,
  "attachments_count": 2,
  "attachments": [
    {
      "id": 1,
      "filename": "welcome.pdf",
      "content_type": "application/pdf",
      "size_bytes": 102400,
      "storage_path": "/data/attachments/acc_1234567890/1/welcome.pdf"
    }
  ],
  "sent_at": "2025-10-29T10:30:00Z",
  "created_at": "2025-10-29T10:35:00Z"
}
```

---

### 3. 搜索邮件

全文搜索邮件（主题、发件人、正文）。

**请求**

```http
GET /api/v1/emails/search
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| q | string | 是 | 搜索关键词 |
| account_uid | string | 否 | 账户 UID |
| page | integer | 否 | 页码（默认 1） |
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |

**响应示例**

```json
{
  "emails": [...],
  "total": 25,
  "page": 1,
  "page_size": 20,
  "total_pages": 2
}
```

---

### 4. 获取未读邮件数

获取指定账户或全部账户的未读邮件数。

**请求**

```http
GET /api/v1/emails/unread-count
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| account_uid | string | 否 | 账户 UID（不传则统计全部） |

**响应示例**

```json
{
  "unread_count": 42
}
```

---

### 5. 获取账户邮件统计

获取指定账户的邮件统计信息。

**请求**

```http
GET /api/v1/emails/stats/:account_uid
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| account_uid | string | 是 | 账户 UID |

**响应示例**

```json
{
  "total_count": 1500,
  "unread_count": 42,
  "starred_count": 15,
  "archived_count": 200
}
```

---

## 邮件状态管理接口

> **注意**：所有状态变更仅在本地生效，不会同步到源邮箱服务器。

### 6. 标记为已读

批量标记邮件为已读。

**请求**

```http
POST /api/v1/emails/mark-read
Content-Type: application/json

{
  "ids": [1, 2, 3]
}
```

**响应示例**

```json
{
  "message": "emails marked as read"
}
```

---

### 7. 标记为未读

批量标记邮件为未读。

**请求**

```http
POST /api/v1/emails/mark-unread
Content-Type: application/json

{
  "ids": [1, 2, 3]
}
```

**响应示例**

```json
{
  "message": "emails marked as unread"
}
```

---

### 8. 切换星标状态

切换邮件的星标状态。

**请求**

```http
POST /api/v1/emails/:id/toggle-star
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 邮件 ID |

**响应示例**

```json
{
  "message": "star status toggled"
}
```

---

### 9. 归档邮件

归档邮件（仅本地状态）。

**请求**

```http
POST /api/v1/emails/:id/archive
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 邮件 ID |

**响应示例**

```json
{
  "message": "email archived"
}
```

---

### 10. 删除邮件

删除邮件（软删除，仅本地状态）。

**请求**

```http
DELETE /api/v1/emails/:id
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 邮件 ID |

**响应示例**

```json
{
  "message": "email deleted"
}
```

---

## 错误响应

所有接口在出错时返回统一的错误格式：

```json
{
  "error": "错误描述信息"
}
```

**常见错误码**

| HTTP 状态码 | 说明 |
|-----------|------|
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 使用示例

### 示例 1：获取未读邮件列表

```bash
curl -X GET "http://localhost:8080/api/v1/emails?is_read=false&page=1&page_size=20"
```

### 示例 2：搜索包含"发票"的邮件

```bash
curl -X GET "http://localhost:8080/api/v1/emails/search?q=发票"
```

### 示例 3：标记邮件为已读

```bash
curl -X POST "http://localhost:8080/api/v1/emails/mark-read" \
  -H "Content-Type: application/json" \
  -d '{"ids": [1, 2, 3]}'
```

### 示例 4：获取账户统计信息

```bash
curl -X GET "http://localhost:8080/api/v1/emails/stats/acc_1234567890"
```

---

## 性能优化建议

1. **分页查询**：建议每页不超过 50 条记录，避免一次性加载过多数据
2. **筛选条件**：尽量使用 `account_uid` 筛选，提高查询效率
3. **搜索功能**：全文搜索使用 PostgreSQL 的 tsvector 索引，性能较好
4. **缓存策略**：前端可缓存邮件列表 5 分钟，减少 API 调用

---

## 后续计划

- [ ] 添加邮件标签功能
- [ ] 支持批量操作（批量归档、批量删除）
- [ ] 添加邮件导出功能
- [ ] 实现邮件附件下载接口
- [ ] 添加邮件会话（Thread）视图
