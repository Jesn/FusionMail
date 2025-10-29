# 规则引擎使用指南

## 概述

FusionMail 规则引擎允许用户创建自动化规则来处理邮件。规则由条件和动作组成，当邮件满足条件时，自动执行相应的动作。

## 规则结构

### 规则模型

```go
type EmailRule struct {
    ID             int64           // 规则 ID
    Name           string          // 规则名称
    AccountUID     string          // 所属账户 UID
    Description    string          // 规则描述
    Enabled        bool            // 是否启用
    Priority       int             // 优先级（数字越小优先级越高）
    MatchMode      string          // 匹配模式：all（所有条件）或 any（任意条件）
    StopProcessing bool            // 匹配后是否停止处理后续规则
    Conditions     []RuleCondition // 条件列表
    Actions        []RuleAction    // 动作列表
}
```

### 条件（Condition）

```go
type RuleCondition struct {
    Field    string // 字段：from, to, subject, body, has_attachment
    Operator string // 操作符：contains, not_contains, equals, not_equals, starts_with, ends_with, regex
    Value    string // 值
}
```

#### 支持的字段

- `from`: 发件人地址
- `to`: 收件人地址
- `subject`: 邮件主题
- `body`: 邮件正文
- `has_attachment`: 是否有附件（值为 "true" 或 "false"）

#### 支持的操作符

- `contains`: 包含（不区分大小写）
- `not_contains`: 不包含（不区分大小写）
- `equals`: 等于（不区分大小写）
- `not_equals`: 不等于（不区分大小写）
- `starts_with`: 开始于（不区分大小写）
- `ends_with`: 结束于（不区分大小写）
- `regex`: 正则表达式匹配

### 动作（Action）

```go
type RuleAction struct {
    Type  string // 动作类型
    Value string // 动作参数（某些动作需要）
}
```

#### 支持的动作类型

- `mark_read`: 标记为已读
- `mark_unread`: 标记为未读
- `star`: 添加星标
- `unstar`: 移除星标
- `archive`: 归档邮件
- `delete`: 删除邮件（软删除）
- `move_folder`: 移动到文件夹（需要 Value 参数）
- `add_label`: 添加标签（需要 Value 参数，待实现）
- `remove_label`: 移除标签（需要 Value 参数，待实现）
- `webhook`: 触发 Webhook（需要 Value 参数，待实现）

## API 使用示例

### 1. 创建规则

**请求**：
```http
POST /api/v1/rules
Content-Type: application/json

{
  "name": "自动归档通知邮件",
  "description": "将所有通知邮件自动归档",
  "account_uid": "account-123",
  "enabled": true,
  "priority": 10,
  "match_mode": "all",
  "stop_processing": false,
  "conditions": [
    {
      "field": "subject",
      "operator": "starts_with",
      "value": "【通知】"
    }
  ],
  "actions": [
    {
      "type": "mark_read"
    },
    {
      "type": "archive"
    }
  ]
}
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "自动归档通知邮件",
    "account_uid": "account-123",
    "enabled": true,
    "priority": 10,
    "match_mode": "all",
    "stop_processing": false,
    "conditions": [...],
    "actions": [...],
    "matched_count": 0,
    "created_at": "2025-10-29T10:00:00Z"
  }
}
```

### 2. 获取规则列表

**请求**：
```http
GET /api/v1/rules?account_uid=account-123
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "name": "自动归档通知邮件",
      "enabled": true,
      "priority": 10,
      "matched_count": 15
    }
  ]
}
```

### 3. 更新规则

**请求**：
```http
PUT /api/v1/rules/1
Content-Type: application/json

{
  "enabled": false
}
```

### 4. 测试规则

**请求**：
```http
POST /api/v1/rules/1/test
Content-Type: application/json

{
  "from_address": "system@example.com",
  "to_address": "user@example.com",
  "subject": "【通知】系统维护",
  "body": "系统将于今晚进行维护",
  "has_attachment": false
}
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "matched": true,
    "rule": {
      "id": 1,
      "name": "自动归档通知邮件",
      "conditions": [...],
      "actions": [...]
    }
  }
}
```

### 5. 删除规则

**请求**：
```http
DELETE /api/v1/rules/1
```

### 6. 切换规则启用状态

**请求**：
```http
POST /api/v1/rules/1/toggle
```

## 规则示例

### 示例 1：自动标记重要邮件

```json
{
  "name": "标记重要邮件",
  "match_mode": "any",
  "conditions": [
    {
      "field": "subject",
      "operator": "contains",
      "value": "紧急"
    },
    {
      "field": "subject",
      "operator": "contains",
      "value": "重要"
    }
  ],
  "actions": [
    {
      "type": "star"
    },
    {
      "type": "mark_read"
    }
  ]
}
```

### 示例 2：自动归档营销邮件

```json
{
  "name": "归档营销邮件",
  "match_mode": "all",
  "conditions": [
    {
      "field": "from",
      "operator": "contains",
      "value": "marketing"
    },
    {
      "field": "subject",
      "operator": "regex",
      "value": "(促销|优惠|折扣)"
    }
  ],
  "actions": [
    {
      "type": "mark_read"
    },
    {
      "type": "archive"
    }
  ]
}
```

### 示例 3：处理带附件的邮件

```json
{
  "name": "处理带附件邮件",
  "match_mode": "all",
  "conditions": [
    {
      "field": "has_attachment",
      "operator": "equals",
      "value": "true"
    },
    {
      "field": "from",
      "operator": "contains",
      "value": "@company.com"
    }
  ],
  "actions": [
    {
      "type": "star"
    },
    {
      "type": "move_folder",
      "value": "工作文档"
    }
  ]
}
```

### 示例 4：自动删除垃圾邮件

```json
{
  "name": "删除垃圾邮件",
  "match_mode": "any",
  "stop_processing": true,
  "conditions": [
    {
      "field": "subject",
      "operator": "contains",
      "value": "中奖"
    },
    {
      "field": "subject",
      "operator": "contains",
      "value": "免费领取"
    }
  ],
  "actions": [
    {
      "type": "delete"
    }
  ]
}
```

## 规则执行流程

1. **触发时机**：当新邮件同步到系统时，自动触发规则引擎
2. **规则排序**：按优先级（priority）从小到大排序
3. **条件匹配**：
   - `match_mode = "all"`：所有条件都必须匹配
   - `match_mode = "any"`：任意一个条件匹配即可
4. **动作执行**：按顺序执行所有动作
5. **停止处理**：如果 `stop_processing = true`，匹配后不再处理后续规则
6. **统计更新**：更新规则的 `matched_count` 和 `last_matched_at`

## 最佳实践

### 1. 规则优先级

- 重要规则设置较小的优先级数字（如 1, 2, 3）
- 通用规则设置较大的优先级数字（如 100, 200）
- 删除规则应该设置最高优先级并启用 `stop_processing`

### 2. 条件设计

- 使用 `match_mode = "all"` 时，条件应该从严格到宽松排列
- 使用正则表达式时，注意性能影响
- 避免过于复杂的条件组合

### 3. 动作顺序

- 先执行标记类动作（mark_read, star）
- 再执行移动类动作（archive, move_folder）
- 最后执行删除动作（delete）

### 4. 性能优化

- 避免创建过多规则（建议每个账户不超过 50 个）
- 定期清理不再使用的规则
- 使用规则测试功能验证规则是否正确

## 注意事项

1. **只读镜像模式**：所有规则动作仅在 FusionMail 本地生效，不会同步到源邮箱
2. **规则冲突**：多个规则可能对同一封邮件执行冲突的动作，注意规则优先级和 `stop_processing` 设置
3. **正则表达式**：使用正则表达式时，确保语法正确，否则规则验证会失败
4. **性能影响**：规则过多或条件过于复杂会影响邮件同步性能

## 故障排查

### 规则不生效

1. 检查规则是否启用（`enabled = true`）
2. 检查规则条件是否正确匹配
3. 使用测试接口验证规则
4. 查看规则的 `matched_count` 是否增加

### 规则匹配错误

1. 检查 `match_mode` 设置（all/any）
2. 检查操作符是否正确（contains/equals/regex）
3. 检查字段名称是否正确
4. 使用测试接口调试

### 动作执行失败

1. 检查动作类型是否支持
2. 检查需要参数的动作是否提供了 `value`
3. 查看系统日志获取详细错误信息

## 未来功能

- [ ] 标签管理功能
- [ ] Webhook 触发功能
- [ ] 规则执行历史记录
- [ ] 规则模板库
- [ ] 批量应用规则到历史邮件
- [ ] 规则导入/导出功能
