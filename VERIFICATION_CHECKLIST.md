# FusionMail 功能验证清单

使用本清单验证新实现的功能是否正常工作。

## 前置准备

- [ ] 已启动 PostgreSQL 和 Redis（`./scripts/dev-start.sh`）
- [ ] 已启动后端服务（`cd backend && go run cmd/server/main.go`）
- [ ] 已添加至少一个邮箱账户
- [ ] 已完成至少一次邮件同步
- [ ] 已安装 `curl` 和 `jq` 工具

## 环境变量设置

```bash
# 设置您的账户 UID（从添加账户的响应中获取）
export ACCOUNT_UID="acc_xxx"

# 设置 API 基础 URL
export API_BASE_URL="http://localhost:8080/api/v1"
```

---

## 📧 邮件管理功能验证

### 1. 获取邮件列表

```bash
curl -s "${API_BASE_URL}/emails?account_uid=${ACCOUNT_UID}&page=1&page_size=10" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应包含 `emails` 数组
- [ ] 响应包含 `total`、`page`、`page_size`、`total_pages` 字段
- [ ] 邮件数据包含 `id`、`subject`、`from_address` 等字段

### 2. 获取邮件详情

```bash
# 先获取第一封邮件的 ID
EMAIL_ID=$(curl -s "${API_BASE_URL}/emails?account_uid=${ACCOUNT_UID}&page=1&page_size=1" | jq -r '.emails[0].id')

# 获取详情
curl -s "${API_BASE_URL}/emails/${EMAIL_ID}" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应包含完整的邮件信息
- [ ] 包含 `text_body` 和 `html_body` 字段
- [ ] 如果有附件，包含 `attachments` 数组

### 3. 搜索邮件

```bash
curl -s "${API_BASE_URL}/emails/search?q=test&account_uid=${ACCOUNT_UID}" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应格式与邮件列表相同
- [ ] 只返回匹配搜索关键词的邮件

### 4. 获取未读邮件数

```bash
curl -s "${API_BASE_URL}/emails/unread-count?account_uid=${ACCOUNT_UID}" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应包含 `unread_count` 字段
- [ ] 数字合理（≥ 0）

### 5. 获取账户统计

```bash
curl -s "${API_BASE_URL}/emails/stats/${ACCOUNT_UID}" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应包含 `total_count`、`unread_count`、`starred_count`、`archived_count`
- [ ] 所有数字合理（≥ 0）

### 6. 标记邮件为已读

```bash
# 获取一封未读邮件的 ID
UNREAD_ID=$(curl -s "${API_BASE_URL}/emails?is_read=false&page=1&page_size=1" | jq -r '.emails[0].id')

# 标记为已读
curl -s -X POST "${API_BASE_URL}/emails/mark-read" \
  -H "Content-Type: application/json" \
  -d "{\"ids\": [${UNREAD_ID}]}" | jq '.'

# 验证：再次获取该邮件，检查 is_read 字段
curl -s "${API_BASE_URL}/emails/${UNREAD_ID}" | jq '.is_read'
```

**验证点**：
- [ ] 标记操作返回状态码 200
- [ ] 响应包含 `message` 字段
- [ ] 再次查询时 `is_read` 为 `true`

### 7. 切换星标状态

```bash
# 获取第一封邮件的 ID
EMAIL_ID=$(curl -s "${API_BASE_URL}/emails?page=1&page_size=1" | jq -r '.emails[0].id')

# 切换星标
curl -s -X POST "${API_BASE_URL}/emails/${EMAIL_ID}/toggle-star" | jq '.'

# 验证：查看星标状态
curl -s "${API_BASE_URL}/emails/${EMAIL_ID}" | jq '.is_starred'
```

**验证点**：
- [ ] 切换操作返回状态码 200
- [ ] 星标状态已改变

### 8. 归档邮件

```bash
EMAIL_ID=$(curl -s "${API_BASE_URL}/emails?page=1&page_size=1" | jq -r '.emails[0].id')

curl -s -X POST "${API_BASE_URL}/emails/${EMAIL_ID}/archive" | jq '.'

# 验证
curl -s "${API_BASE_URL}/emails/${EMAIL_ID}" | jq '.is_archived'
```

**验证点**：
- [ ] 归档操作返回状态码 200
- [ ] `is_archived` 为 `true`

### 9. 删除邮件

```bash
EMAIL_ID=$(curl -s "${API_BASE_URL}/emails?page=1&page_size=1" | jq -r '.emails[0].id')

curl -s -X DELETE "${API_BASE_URL}/emails/${EMAIL_ID}" | jq '.'

# 验证
curl -s "${API_BASE_URL}/emails/${EMAIL_ID}" | jq '.is_deleted'
```

**验证点**：
- [ ] 删除操作返回状态码 200
- [ ] `is_deleted` 为 `true`

---

## 🤖 规则引擎功能验证

### 1. 创建规则

```bash
curl -s -X POST "${API_BASE_URL}/rules" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"测试规则\",
    \"account_uid\": \"${ACCOUNT_UID}\",
    \"description\": \"自动归档包含'测试'的邮件\",
    \"conditions\": \"[{\\\"field\\\":\\\"subject\\\",\\\"operator\\\":\\\"contains\\\",\\\"value\\\":\\\"测试\\\"}]\",
    \"actions\": \"[{\\\"type\\\":\\\"archive\\\"},{\\\"type\\\":\\\"mark_read\\\"}]\",
    \"priority\": 10,
    \"enabled\": true
  }" | jq '.'
```

**验证点**：
- [ ] 返回状态码 201
- [ ] 响应包含规则的完整信息
- [ ] 包含 `id` 字段（保存此 ID 用于后续测试）

```bash
# 保存规则 ID
export RULE_ID=$(curl -s "${API_BASE_URL}/rules?account_uid=${ACCOUNT_UID}" | jq -r '.[0].id')
```

### 2. 获取规则列表

```bash
curl -s "${API_BASE_URL}/rules?account_uid=${ACCOUNT_UID}" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应是一个数组
- [ ] 包含刚创建的规则

### 3. 获取规则详情

```bash
curl -s "${API_BASE_URL}/rules/${RULE_ID}" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应包含规则的完整信息
- [ ] 包含 `conditions` 和 `actions` 字段

### 4. 更新规则

```bash
curl -s -X PUT "${API_BASE_URL}/rules/${RULE_ID}" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"测试规则（已更新）\",
    \"account_uid\": \"${ACCOUNT_UID}\",
    \"description\": \"更新后的描述\",
    \"conditions\": \"[{\\\"field\\\":\\\"subject\\\",\\\"operator\\\":\\\"contains\\\",\\\"value\\\":\\\"测试\\\"}]\",
    \"actions\": \"[{\\\"type\\\":\\\"star\\\"}]\",
    \"priority\": 5,
    \"enabled\": true
  }" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 规则名称已更新
- [ ] 动作已更新为 `star`

### 5. 切换规则状态

```bash
curl -s -X POST "${API_BASE_URL}/rules/${RULE_ID}/toggle" | jq '.'

# 验证
curl -s "${API_BASE_URL}/rules/${RULE_ID}" | jq '.enabled'
```

**验证点**：
- [ ] 切换操作返回状态码 200
- [ ] `enabled` 状态已改变

### 6. 对账户应用规则

```bash
# 先启用规则
curl -s -X POST "${API_BASE_URL}/rules/${RULE_ID}/toggle" | jq '.'

# 应用规则
curl -s -X POST "${API_BASE_URL}/rules/apply/${ACCOUNT_UID}" | jq '.'

# 验证：检查规则执行统计
curl -s "${API_BASE_URL}/rules/${RULE_ID}" | jq '.execution_count'
```

**验证点**：
- [ ] 应用操作返回状态码 200
- [ ] `execution_count` 大于 0
- [ ] 匹配条件的邮件状态已改变

### 7. 删除规则

```bash
curl -s -X DELETE "${API_BASE_URL}/rules/${RULE_ID}" | jq '.'

# 验证
curl -s "${API_BASE_URL}/rules/${RULE_ID}"
```

**验证点**：
- [ ] 删除操作返回状态码 200
- [ ] 再次查询返回 404 或错误信息

---

## 🔄 同步功能验证

### 1. 手动同步

```bash
curl -s -X POST "${API_BASE_URL}/sync/accounts/${ACCOUNT_UID}" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应包含 `message` 字段
- [ ] 后端日志显示同步进度

### 2. 查看同步状态

```bash
curl -s "${API_BASE_URL}/sync/status" | jq '.'
```

**验证点**：
- [ ] 返回状态码 200
- [ ] 响应包含 `running` 字段

### 3. 验证增量同步

```bash
# 第一次同步
curl -s -X POST "${API_BASE_URL}/sync/accounts/${ACCOUNT_UID}"
sleep 5

# 获取邮件总数
TOTAL_1=$(curl -s "${API_BASE_URL}/emails?account_uid=${ACCOUNT_UID}&page=1&page_size=1" | jq -r '.total')

# 第二次同步（应该是增量同步）
curl -s -X POST "${API_BASE_URL}/sync/accounts/${ACCOUNT_UID}"
sleep 5

# 再次获取邮件总数
TOTAL_2=$(curl -s "${API_BASE_URL}/emails?account_uid=${ACCOUNT_UID}&page=1&page_size=1" | jq -r '.total')

echo "第一次同步后: $TOTAL_1 封邮件"
echo "第二次同步后: $TOTAL_2 封邮件"
```

**验证点**：
- [ ] 第二次同步速度更快
- [ ] 邮件总数没有重复增加
- [ ] 后端日志显示"Incremental sync"

---

## 🧪 自动化测试脚本

运行自动化测试脚本：

```bash
ACCOUNT_UID=${ACCOUNT_UID} ./scripts/test-email-api.sh
```

**验证点**：
- [ ] 所有测试通过（显示绿色 ✓）
- [ ] 没有错误信息
- [ ] 最后显示"所有测试通过！"

---

## 📊 性能验证

### 1. 邮件列表查询性能

```bash
time curl -s "${API_BASE_URL}/emails?account_uid=${ACCOUNT_UID}&page=1&page_size=100" > /dev/null
```

**验证点**：
- [ ] 响应时间 < 2 秒

### 2. 搜索性能

```bash
time curl -s "${API_BASE_URL}/emails/search?q=test&account_uid=${ACCOUNT_UID}" > /dev/null
```

**验证点**：
- [ ] 响应时间 < 2 秒

### 3. 规则执行性能

```bash
time curl -s -X POST "${API_BASE_URL}/rules/apply/${ACCOUNT_UID}" > /dev/null
```

**验证点**：
- [ ] 执行时间合理（取决于邮件数量）

---

## 🐛 错误处理验证

### 1. 无效的邮件 ID

```bash
curl -s "${API_BASE_URL}/emails/999999" | jq '.'
```

**验证点**：
- [ ] 返回状态码 404 或 500
- [ ] 响应包含 `error` 字段

### 2. 无效的查询参数

```bash
curl -s "${API_BASE_URL}/emails?page=-1" | jq '.'
```

**验证点**：
- [ ] 返回合理的响应（自动修正为 page=1）

### 3. 无效的规则条件

```bash
curl -s -X POST "${API_BASE_URL}/rules" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"无效规则\",
    \"account_uid\": \"${ACCOUNT_UID}\",
    \"conditions\": \"invalid json\",
    \"actions\": \"[]\",
    \"enabled\": true
  }" | jq '.'
```

**验证点**：
- [ ] 返回状态码 400 或 500
- [ ] 响应包含错误信息

---

## ✅ 验证总结

完成上述所有验证后，请确认：

- [ ] 所有邮件管理 API 正常工作
- [ ] 所有规则引擎 API 正常工作
- [ ] 同步功能正常工作
- [ ] 性能符合预期
- [ ] 错误处理合理
- [ ] 自动化测试脚本通过

如果所有项目都已勾选，恭喜您！FusionMail 的核心功能已成功实现并验证通过。

---

## 🔧 故障排查

### 问题 1：API 返回 404

**可能原因**：
- 服务器未启动
- 路由未正确注册

**解决方法**：
```bash
# 检查服务器是否运行
curl http://localhost:8080/api/v1/health

# 查看后端日志
```

### 问题 2：邮件列表为空

**可能原因**：
- 同步未完成
- 账户 UID 错误

**解决方法**：
```bash
# 检查账户列表
curl http://localhost:8080/api/v1/accounts

# 手动触发同步
curl -X POST http://localhost:8080/api/v1/sync/accounts/${ACCOUNT_UID}
```

### 问题 3：规则不生效

**可能原因**：
- 规则未启用
- 条件不匹配
- 未手动触发规则应用

**解决方法**：
```bash
# 检查规则状态
curl http://localhost:8080/api/v1/rules/${RULE_ID}

# 手动应用规则
curl -X POST http://localhost:8080/api/v1/rules/apply/${ACCOUNT_UID}
```

---

**验证日期**: ___________  
**验证人**: ___________  
**结果**: [ ] 通过 / [ ] 未通过
