# 规则引擎 API 文档

## 概述

规则引擎允许用户创建自动化规则，根据邮件的特征（发件人、主题、正文等）自动执行操作（标记已读、星标、归档、删除、触发 Webhook 等）。

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`
- **认证**: 暂未实现（后续版本添加）

---

## 规则管理接口

### 1. 创建规则

创建新的邮件处理规则。

**请求**

```http
POST /api/v1/rules
Content-Type: application/json

{
  "name": "自动归档通知邮件",
  "account_uid": "acc_1234567890",
  "description": "将所有通知类邮件自动归档",
  "conditions": "[{\"field\":\"subject\",\"operator\":\"contains\",\"value\":\"通知\"}]",
  "actions": "[{\"type\":\"archive\"},{\"type\":\"mark_read\"}]",
  "priority": 10,
  "stop_processing": false,
  "enabled": true
}
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| name | string | 是 | 规则名称 |
| account_uid | string | 是 | 账户 UID |
| description | string | 否 | 规则描述 |
| conditions | string | 是 | 条件 JSON 字符串（见下方说明） |
| actions | string | 是 | 动作 JSON 字符串（见下方说明） |
| priority | integer | 否 | 优先级（数字越小优先级越高，默认 0） |
| stop_processing | boolean | 否 | 匹配后是否停止处理后续规则（默认 false） |
| enabled | boolean | 否 | 是否启用（默认 true） |

**条件格式（Conditions）**

条件是一个 JSON 数组，每个条件包含：

```json
[
  {
    "field": "subject",        // 字段名
    "operator": "contains",    // 操作符
    "value": "通知"            // 匹配值
  }
]
```

**支持的字段**

| 字段 | 说明 |
|-----|------|
| from_address | 发件人地址 |
| from_name | 发件人名称 |
| subject | 邮件主题 |
| body | 邮件正文 |
| to_addresses | 收件人地址 |

**支持的操作符**

| 操作符 | 说明 |
|-------|------|
| contains | 包含 |
| not_contains | 不包含 |
| equals | 等于 |
| not_equals | 不等于 |
| starts_with | 以...开头 |
| ends_with | 以...结尾 |

**动作格式（Actions）**

动作是一个 JSON 数组，每个动作包含：

```json
[
  {
    "type": "mark_read"        // 动作类型
  },
  {
    "type": "add_label",       // 动作类型
    "value": "重要"            // 动作参数（可选）
  }
]
```

**支持的动作类型**

| 动作类型 | 说明 | 是否需要 value |
|---------|------|---------------|
| mark_read | 标记为已读 | 否 |
| mark_unread | 标记为未读 | 否 |
| star | 添加星标 | 否 |
| archive | 归档 | 否 |
| delete | 删除（软删除） | 否 |
| add_label | 添加标签 | 是（标签名） |
| trigger_webhook | 触发 Webhook | 是（Webhook ID） |

**响应示例**

```json
{
  "id": 1,
  "name": "自动归档通知邮件",
  "account_uid": "acc_1234567890",
  "description": "将所有通知类邮件自动归档",
  "conditions": "[{\"field\":\"subject\",\"operator\":\"contains\",\"value\":\"通知\"}]",
  "actions": "[{\"type\":\"archive\"},{\"type\":\"mark_read\"}]",
  "priority": 10,
  "stop_processing": false,
  "enabled": true,
  "execution_count": 0,
  "created_at": "2025-10-29T10:00:00Z",
  "updated_at": "2025-10-29T10:00:00Z"
}
```

---

### 2. 获取规则列表

获取指定账户的所有规则。

**请求**

```http
GET /api/v1/rules?account_uid=acc_1234567890
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| account_uid | string | 否 | 账户 UID（不传则返回所有规则） |

**响应示例**

```json
[
  {
    "id": 1,
    "name": "自动归档通知邮件",
    "account_uid": "acc_1234567890",
    "enabled": true,
    "priority": 10,
    "execution_count": 150,
    "last_executed_at": "2025-10-29T09:30:00Z",
    "created_at": "2025-10-29T08:00:00Z"
  }
]
```

---

### 3. 获取规则详情

根据 ID 获取规则的详细信息。

**请求**

```http
GET /api/v1/rules/:id
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 规则 ID |

**响应示例**

```json
{
  "id": 1,
  "name": "自动归档通知邮件",
  "account_uid": "acc_1234567890",
  "description": "将所有通知类邮件自动归档",
  "conditions": "[{\"field\":\"subject\",\"operator\":\"contains\",\"value\":\"通知\"}]",
  "actions": "[{\"type\":\"archive\"},{\"type\":\"mark_read\"}]",
  "priority": 10,
  "stop_processing": false,
  "enabled": true,
  "execution_count": 150,
  "last_executed_at": "2025-10-29T09:30:00Z",
  "created_at": "2025-10-29T08:00:00Z",
  "updated_at": "2025-10-29T08:00:00Z"
}
```

---

### 4. 更新规则

更新规则信息。

**请求**

```http
PUT /api/v1/rules/:id
Content-Type: application/json

{
  "name": "自动归档通知邮件（已更新）",
  "account_uid": "acc_1234567890",
  "description": "更新后的描述",
  "conditions": "[{\"field\":\"subject\",\"operator\":\"contains\",\"value\":\"通知\"}]",
  "actions": "[{\"type\":\"archive\"}]",
  "priority": 5,
  "stop_processing": false,
  "enabled": true
}
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 规则 ID |

**响应示例**

```json
{
  "id": 1,
  "name": "自动归档通知邮件（已更新）",
  ...
}
```

---

### 5. 删除规则

删除指定的规则。

**请求**

```http
DELETE /api/v1/rules/:id
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 规则 ID |

**响应示例**

```json
{
  "message": "rule deleted"
}
```

---

### 6. 切换规则启用状态

启用或禁用规则。

**请求**

```http
POST /api/v1/rules/:id/toggle
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| id | integer | 是 | 规则 ID |

**响应示例**

```json
{
  "message": "rule status toggled"
}
```

---

### 7. 对账户应用规则

对指定账户的所有邮件应用规则（批量处理）。

**请求**

```http
POST /api/v1/rules/apply/:account_uid
```

**路径参数**

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| account_uid | string | 是 | 账户 UID |

**响应示例**

```json
{
  "message": "rules applied to account"
}
```

---

## 规则执行逻辑

### 执行时机

规则会在以下时机自动执行：

1. **邮件同步完成后**：新同步的邮件会自动应用规则
2. **手动触发**：通过 API 手动对账户应用规则

### 执行顺序

1. 按 `priority` 字段排序（数字越小优先级越高）
2. 依次检查每个规则的条件
3. 如果条件匹配，执行规则的所有动作
4. 如果规则设置了 `stop_processing=true`，则停止处理后续规则

### 条件匹配

- 规则的所有条件必须同时满足（AND 逻辑）
- 字符串匹配不区分大小写

---

## 使用示例

### 示例 1：自动归档通知邮件

```bash
curl -X POST "http://localhost:8080/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "自动归档通知邮件",
    "account_uid": "acc_1234567890",
    "conditions": "[{\"field\":\"subject\",\"operator\":\"contains\",\"value\":\"通知\"}]",
    "actions": "[{\"type\":\"archive\"},{\"type\":\"mark_read\"}]",
    "enabled": true
  }'
```

### 示例 2：自动星标重要邮件

```bash
curl -X POST "http://localhost:8080/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "自动星标重要邮件",
    "account_uid": "acc_1234567890",
    "conditions": "[{\"field\":\"subject\",\"operator\":\"contains\",\"value\":\"重要\"}]",
    "actions": "[{\"type\":\"star\"}]",
    "priority": 1,
    "enabled": true
  }'
```

### 示例 3：自动删除垃圾邮件

```bash
curl -X POST "http://localhost:8080/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "自动删除垃圾邮件",
    "account_uid": "acc_1234567890",
    "conditions": "[{\"field\":\"from_address\",\"operator\":\"contains\",\"value\":\"spam\"}]",
    "actions": "[{\"type\":\"delete\"}]",
    "stop_processing": true,
    "enabled": true
  }'
```

### 示例 4：对账户应用规则

```bash
curl -X POST "http://localhost:8080/api/v1/rules/apply/acc_1234567890"
```

---

## 最佳实践

1. **优先级设置**：重要规则设置较小的优先级数字，确保优先执行
2. **停止处理**：删除类规则建议设置 `stop_processing=true`，避免后续规则处理已删除的邮件
3. **条件组合**：合理组合多个条件，提高规则的精确度
4. **测试规则**：创建规则后，先在小范围测试，确认效果后再启用
5. **定期审查**：定期检查规则的执行统计，优化不必要的规则

---

## 后续计划

- [ ] 支持 OR 逻辑（任一条件满足即可）
- [ ] 支持正则表达式匹配
- [ ] 支持时间条件（如工作日、特定时间段）
- [ ] 支持附件条件（如有附件、附件类型）
- [ ] 添加规则测试接口（预览规则效果）
- [ ] 支持规则模板（常用规则快速创建）
