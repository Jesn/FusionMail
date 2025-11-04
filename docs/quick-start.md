# FusionMail 快速开始指南

本指南将帮助您快速启动 FusionMail 并测试核心功能。

## 前置要求

- Docker 和 Docker Compose
- Go 1.21+（如需本地开发）
- curl 和 jq（用于测试 API）

---

## 步骤 1：启动基础设施

启动 PostgreSQL 和 Redis：

```bash
./scripts/dev-start.sh
```

等待服务启动完成（约 10 秒）。

---

## 步骤 2：启动后端服务

```bash
cd backend
go run cmd/server/main.go
```

您应该看到类似的输出：

```
Starting FusionMail server...
Configuration loaded: DB=localhost:5432, Server=0.0.0.0:3333
Database initialization completed successfully
Sync manager started successfully
Server listening on 0.0.0.0:3333
API endpoint: http://0.0.0.0:3333/api/v1
```

---

## 步骤 3：添加邮箱账户

### 方法 1：使用 curl

```bash
curl -X POST http://localhost:3333/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your@qq.com",
    "provider": "qq",
    "protocol": "imap",
    "auth_type": "password",
    "password": "your_authorization_code",
    "sync_enabled": true,
    "sync_interval": 5
  }'
```

### 方法 2：使用测试配置文件

如果您有 `.test-config` 文件：

```bash
# 从配置文件读取账户信息
source .test-config

curl -X POST http://localhost:3333/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"provider\": \"qq\",
    \"protocol\": \"imap\",
    \"auth_type\": \"password\",
    \"password\": \"$PASSWORD\",
    \"sync_enabled\": true,
    \"sync_interval\": 5
  }"
```

**响应示例**：

```json
{
  "uid": "acc_1730188800_abc123",
  "email": "your@qq.com",
  "provider": "qq",
  "protocol": "imap",
  "sync_enabled": true,
  "sync_interval": 5,
  "created_at": "2025-10-29T10:00:00Z"
}
```

**保存账户 UID**，后续步骤会用到：

```bash
export ACCOUNT_UID="acc_1730188800_abc123"
```

---

## 步骤 4：同步邮件

### 手动触发同步

```bash
curl -X POST http://localhost:3333/api/v1/sync/accounts/$ACCOUNT_UID
```

**响应**：

```json
{
  "message": "Sync started"
}
```

### 查看同步状态

```bash
curl http://localhost:3333/api/v1/sync/status
```

**响应**：

```json
{
  "running": true
}
```

同步可能需要几分钟，取决于邮件数量。您可以查看后端日志了解进度。

---

## 步骤 5：查看邮件

### 获取邮件列表

```bash
curl "http://localhost:3333/api/v1/emails?account_uid=$ACCOUNT_UID&page=1&page_size=10" | jq '.'
```

**响应示例**：

```json
{
  "emails": [
    {
      "id": 1,
      "subject": "欢迎使用 QQ 邮箱",
      "from_address": "noreply@qq.com",
      "from_name": "QQ邮箱团队",
      "snippet": "欢迎使用 QQ 邮箱...",
      "is_read": false,
      "sent_at": "2025-10-29T10:30:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "page_size": 10,
  "total_pages": 15
}
```

### 获取邮件详情

```bash
# 使用上面获取的邮件 ID
curl http://localhost:3333/api/v1/emails/1 | jq '.'
```

### 搜索邮件

```bash
curl "http://localhost:3333/api/v1/emails/search?q=通知&account_uid=$ACCOUNT_UID" | jq '.'
```

### 获取未读邮件数

```bash
curl "http://localhost:3333/api/v1/emails/unread-count?account_uid=$ACCOUNT_UID" | jq '.'
```

### 获取账户统计

```bash
curl "http://localhost:3333/api/v1/emails/stats/$ACCOUNT_UID" | jq '.'
```

**响应示例**：

```json
{
  "total_count": 150,
  "unread_count": 42,
  "starred_count": 5,
  "archived_count": 10
}
```

---

## 步骤 6：管理邮件状态

### 标记为已读

```bash
curl -X POST http://localhost:3333/api/v1/emails/mark-read \
  -H "Content-Type: application/json" \
  -d '{"ids": [1, 2, 3]}'
```

### 添加星标

```bash
curl -X POST http://localhost:3333/api/v1/emails/1/toggle-star
```

### 归档邮件

```bash
curl -X POST http://localhost:3333/api/v1/emails/1/archive
```

---

## 步骤 7：创建自动化规则

### 示例 1：自动归档通知邮件

```bash
curl -X POST http://localhost:3333/api/v1/rules \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"自动归档通知邮件\",
    \"account_uid\": \"$ACCOUNT_UID\",
    \"description\": \"将所有包含'通知'的邮件自动归档\",
    \"conditions\": \"[{\\\"field\\\":\\\"subject\\\",\\\"operator\\\":\\\"contains\\\",\\\"value\\\":\\\"通知\\\"}]\",
    \"actions\": \"[{\\\"type\\\":\\\"archive\\\"},{\\\"type\\\":\\\"mark_read\\\"}]\",
    \"priority\": 10,
    \"enabled\": true
  }"
```

### 示例 2：自动星标重要邮件

```bash
curl -X POST http://localhost:3333/api/v1/rules \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"自动星标重要邮件\",
    \"account_uid\": \"$ACCOUNT_UID\",
    \"conditions\": \"[{\\\"field\\\":\\\"subject\\\",\\\"operator\\\":\\\"contains\\\",\\\"value\\\":\\\"重要\\\"}]\",
    \"actions\": \"[{\\\"type\\\":\\\"star\\\"}]\",
    \"priority\": 1,
    \"enabled\": true
  }"
```

### 查看规则列表

```bash
curl "http://localhost:3333/api/v1/rules?account_uid=$ACCOUNT_UID" | jq '.'
```

### 对账户应用规则

```bash
curl -X POST "http://localhost:3333/api/v1/rules/apply/$ACCOUNT_UID"
```

这将对账户中的所有邮件应用规则。

---

## 步骤 8：使用测试脚本

我们提供了一个自动化测试脚本：

```bash
# 测试所有邮件 API
ACCOUNT_UID=$ACCOUNT_UID ./scripts/test-email-api.sh
```

脚本会自动测试以下功能：
- ✓ 获取邮件列表
- ✓ 获取邮件详情
- ✓ 搜索邮件
- ✓ 获取未读邮件数
- ✓ 获取账户统计
- ✓ 标记邮件为已读
- ✓ 切换星标状态

---

## 常见问题

### Q1: 同步失败怎么办？

**检查账户连接**：

```bash
curl -X POST http://localhost:3333/api/v1/accounts/$ACCOUNT_UID/test
```

如果连接失败，请检查：
- 邮箱地址和授权码是否正确
- 网络连接是否正常
- 是否需要配置代理

### Q2: 邮件列表为空？

确保：
1. 同步已完成（查看后端日志）
2. 账户 UID 正确
3. 邮箱中确实有邮件

### Q3: 规则不生效？

检查：
1. 规则是否启用（`enabled: true`）
2. 条件是否正确匹配
3. 是否手动触发了规则应用

### Q4: 如何停止服务？

```bash
# 停止后端服务
Ctrl+C

# 停止基础设施
./scripts/dev-stop.sh
```

---

## 下一步

现在您已经成功运行了 FusionMail 的核心功能！接下来可以：

1. **查看 API 文档**
   - [邮件管理 API](./email-api.md)
   - [规则引擎 API](./rule-api.md)

2. **开发前端界面**
   - 参考 `frontend/` 目录
   - 使用 React + TypeScript

3. **配置更多账户**
   - 支持 Gmail、Outlook、iCloud、163 等

4. **创建更多规则**
   - 自动分类
   - 自动标签
   - 触发 Webhook

5. **集成第三方服务**
   - Zapier
   - Make
   - n8n

---

## 获取帮助

- 查看 [开发进度文档](./development-progress.md)
- 查看 [测试指南](./testing-guide.md)
- 提交 GitHub Issue

---

**祝您使用愉快！** 🎉
