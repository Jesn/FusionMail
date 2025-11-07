# FusionMail 快速参考指南

---

## 1. 项目快速启动

### 1.1 环境要求

```bash
# 后端
Go 1.24.0+
PostgreSQL 15+
Redis 7+

# 前端
Node.js 20.19.0+
npm 10+

# 开发工具
Docker & Docker Compose
Git
```

### 1.2 快速启动

```bash
# 1. 克隆项目
git clone <repo-url>
cd FusionMail

# 2. 启动数据库和缓存
docker-compose -f docker-compose.dev.yml up -d

# 3. 后端启动
cd backend
go mod download
go run cmd/server/main.go

# 4. 前端启动 (新终端)
cd frontend
npm install
npm run dev

# 5. 访问应用
# 前端: http://localhost:4444
# 后端 API: http://localhost:3333/api/v1
```

---

## 2. 核心概念速查

### 2.1 账户 (Account)

**定义**: 用户添加的邮箱账户

**关键字段**:
- `uid` - 唯一标识符
- `email` - 邮箱地址
- `provider` - 提供商 (gmail, outlook, icloud, qq, 163, generic)
- `protocol` - 协议 (imap, pop3, gmail_api, graph, graph_quick)
- `auth_type` - 认证类型 (oauth2, password, app_password, quick)
- `sync_interval` - 同步间隔 (分钟)
- `last_sync_at` - 最后同步时间

**支持的提供商**:
| 提供商 | 协议 | 认证方式 | 说明 |
|--------|------|---------|------|
| Gmail | gmail_api | oauth2 | 官方 API |
| Outlook | graph | oauth2 | 官方 API |
| iCloud | imap | password | IMAP 协议 |
| QQ | pop3 | app_password | POP3 协议 |
| 163 | pop3 | password | POP3 协议 |
| Generic | imap/pop3 | password | 自定义 IMAP/POP3 |

### 2.2 邮件 (Email)

**定义**: 从邮箱同步的邮件

**关键字段**:
- `provider_id` - 邮箱提供商的邮件 ID
- `account_uid` - 所属账户 UID
- `subject` - 主题
- `from_address` - 发件人
- `to_address` - 收件人
- `sent_at` - 发送时间
- `is_read` - 是否已读 (本地状态)
- `is_starred` - 是否星标 (本地状态)
- `is_archived` - 是否归档 (本地状态)
- `is_deleted` - 是否删除 (本地状态)

**唯一约束**: (provider_id, account_uid)

### 2.3 规则 (Rule)

**定义**: 自动化规则，用于处理邮件

**结构**:
```json
{
  "id": 1,
  "account_uid": "acc_123",
  "name": "重要邮件",
  "conditions": [
    {
      "field": "from",
      "operator": "contains",
      "value": "boss@company.com"
    }
  ],
  "match_mode": "all",
  "actions": [
    {
      "type": "star",
      "params": {}
    },
    {
      "type": "mark_read",
      "params": {}
    }
  ],
  "priority": 1,
  "enabled": true,
  "stop_processing": false
}
```

**条件字段**: from, to, subject, body, has_attachment

**操作符**: contains, not_contains, equals, not_equals, starts_with, ends_with, regex

**动作类型**: mark_read, mark_unread, star, unstar, archive, delete, move_folder, add_label, webhook

### 2.4 Webhook

**定义**: 邮件事件推送到外部系统

**事件类型**:
- email.received - 邮件接收
- email.updated - 邮件更新
- email.deleted - 邮件删除
- sync.completed - 同步完成
- sync.failed - 同步失败

**请求格式**:
```json
{
  "event": "email.received",
  "timestamp": "2025-11-05T10:00:00Z",
  "data": {
    "email_id": "email_123",
    "account_uid": "acc_123",
    "subject": "Test Email",
    "from": "sender@example.com"
  }
}
```

---

## 3. API 端点速查

### 3.1 认证相关

```
POST   /api/v1/auth/login              - 用户登录
POST   /api/v1/auth/logout             - 用户登出
GET    /api/v1/auth/google/authorize   - Google OAuth2 授权
GET    /api/v1/auth/google/callback    - Google OAuth2 回调
GET    /api/v1/auth/microsoft/authorize - Microsoft OAuth2 授权
GET    /api/v1/auth/microsoft/callback - Microsoft OAuth2 回调
```

### 3.2 账户相关

```
GET    /api/v1/accounts                - 获取账户列表
POST   /api/v1/accounts                - 创建账户
GET    /api/v1/accounts/:uid           - 获取账户详情
PUT    /api/v1/accounts/:uid           - 更新账户
DELETE /api/v1/accounts/:uid           - 删除账户
POST   /api/v1/accounts/:uid/sync      - 手动同步
POST   /api/v1/accounts/:uid/test      - 测试连接
```

### 3.3 邮件相关

```
GET    /api/v1/emails                  - 获取邮件列表
GET    /api/v1/emails/:id              - 获取邮件详情
PUT    /api/v1/emails/:id              - 更新邮件状态
DELETE /api/v1/emails/:id              - 删除邮件
POST   /api/v1/emails/search           - 搜索邮件
GET    /api/v1/emails/:id/attachments  - 获取附件
```

### 3.4 规则相关

```
GET    /api/v1/rules                   - 获取规则列表
POST   /api/v1/rules                   - 创建规则
GET    /api/v1/rules/:id               - 获取规则详情
PUT    /api/v1/rules/:id               - 更新规则
DELETE /api/v1/rules/:id               - 删除规则
POST   /api/v1/rules/:id/test          - 测试规则
```

### 3.5 Webhook 相关

```
GET    /api/v1/webhooks                - 获取 Webhook 列表
POST   /api/v1/webhooks                - 创建 Webhook
GET    /api/v1/webhooks/:id            - 获取 Webhook 详情
PUT    /api/v1/webhooks/:id            - 更新 Webhook
DELETE /api/v1/webhooks/:id            - 删除 Webhook
GET    /api/v1/webhooks/:id/logs       - 获取 Webhook 日志
```

### 3.6 系统相关

```
GET    /api/v1/health                  - 健康检查
GET    /api/v1/system/providers        - 获取支持的提供商
GET    /api/v1/system/stats            - 获取系统统计
```

---

## 4. 数据库表速查

| 表名 | 用途 | 关键字段 |
|------|------|---------|
| users | 用户 | id, username, email, password_hash |
| accounts | 邮箱账户 | uid, email, provider, protocol, auth_type |
| emails | 邮件 | id, provider_id, account_uid, subject, from_address |
| email_attachments | 附件 | id, email_id, filename, storage_path |
| email_labels | 标签 | id, name, color |
| email_rules | 规则 | id, account_uid, conditions, actions, priority |
| webhooks | Webhook | id, url, events, enabled |
| webhook_logs | Webhook 日志 | id, webhook_id, request_url, response_status |
| sync_logs | 同步日志 | id, account_uid, status, emails_fetched |
| api_keys | API 密钥 | id, key_hash, permissions, rate_limit |

---

## 5. 常见任务

### 5.1 添加新邮箱账户

```bash
# 1. 前端: 点击 "添加账户"
# 2. 选择提供商 (Gmail, Outlook, etc.)
# 3. 选择认证方式 (OAuth2 或密码)
# 4. 完成认证
# 5. 系统自动同步邮件

# API 调用示例
curl -X POST http://localhost:3333/api/v1/accounts \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@gmail.com",
    "provider": "gmail",
    "protocol": "gmail_api",
    "auth_type": "oauth2",
    "sync_interval": 5
  }'
```

### 5.2 创建自动化规则

```bash
# API 调用示例
curl -X POST http://localhost:3333/api/v1/rules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "account_uid": "acc_123",
    "name": "重要邮件",
    "conditions": [
      {
        "field": "from",
        "operator": "contains",
        "value": "boss@company.com"
      }
    ],
    "match_mode": "all",
    "actions": [
      {
        "type": "star",
        "params": {}
      }
    ],
    "priority": 1,
    "enabled": true
  }'
```

### 5.3 搜索邮件

```bash
# API 调用示例
curl -X POST http://localhost:3333/api/v1/emails/search \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "important",
    "account_uid": "acc_123",
    "page": 1,
    "limit": 20
  }'
```

### 5.4 设置 Webhook

```bash
# API 调用示例
curl -X POST http://localhost:3333/api/v1/webhooks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/webhook",
    "events": ["email.received", "email.updated"],
    "enabled": true
  }'
```

---

## 6. 环境变量配置

### 6.1 后端环境变量

```bash
# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=fusionmail

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24h

# 加密
ENCRYPTION_KEY=your-encryption-key

# OAuth2 - Google
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URI=http://localhost:3333/api/v1/auth/google/callback

# OAuth2 - Microsoft
MICROSOFT_CLIENT_ID=your-client-id
MICROSOFT_CLIENT_SECRET=your-client-secret
MICROSOFT_REDIRECT_URI=http://localhost:3333/api/v1/auth/microsoft/callback

# 服务器
SERVER_PORT=3333
SERVER_ENV=development

# 存储
STORAGE_TYPE=local
STORAGE_PATH=./uploads
```

### 6.2 前端环境变量

```bash
# API 基础 URL
VITE_API_BASE_URL=http://localhost:3333/api/v1

# 应用名称
VITE_APP_NAME=FusionMail

# 环境
VITE_ENV=development
```

---

## 7. 常见问题

### 7.1 邮件同步失败

**原因**:
- 凭证过期
- 网络连接问题
- 邮箱服务器问题

**解决**:
1. 检查账户状态
2. 重新认证
3. 查看同步日志

### 7.2 规则不生效

**原因**:
- 规则未启用
- 条件不匹配
- 优先级问题

**解决**:
1. 检查规则是否启用
2. 测试规则条件
3. 调整优先级

### 7.3 Webhook 不触发

**原因**:
- Webhook 未启用
- URL 不可达
- 事件类型不匹配

**解决**:
1. 检查 Webhook 是否启用
2. 测试 URL 可达性
3. 查看 Webhook 日志

---

## 8. 性能优化建议

### 8.1 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_emails_account_uid ON emails(account_uid);
CREATE INDEX idx_emails_sent_at ON emails(sent_at DESC);
CREATE INDEX idx_emails_is_read ON emails(is_read);

-- 分析查询
EXPLAIN ANALYZE SELECT * FROM emails WHERE account_uid = 'acc_123';
```

### 8.2 缓存优化

```go
// 缓存账户列表
cache.Set("accounts:user_1", accounts, 1*time.Hour)

// 缓存规则列表
cache.Set("rules:acc_123", rules, 30*time.Minute)
```

### 8.3 查询优化

```go
// 使用分页
emails, err := emailRepo.FindByAccountUID(ctx, accountUID, page, limit)

// 使用字段选择
emails, err := emailRepo.FindByAccountUID(ctx, accountUID).
    Select("id", "subject", "from_address", "sent_at").
    Limit(20).
    Find(&emails)
```

---

## 9. 调试技巧

### 9.1 查看日志

```bash
# 后端日志
tail -f backend/logs/app.log

# 前端控制台
# 打开浏览器开发者工具 (F12)
# 查看 Console 标签
```

### 9.2 数据库查询

```bash
# 连接数据库
psql -h localhost -U postgres -d fusionmail

# 查看表结构
\d emails

# 查询数据
SELECT * FROM emails LIMIT 10;
```

### 9.3 API 测试

```bash
# 使用 curl
curl -X GET http://localhost:3333/api/v1/emails \
  -H "Authorization: Bearer <token>"

# 使用 Postman
# 导入 API 集合
# 设置环境变量
# 执行请求
```

---

## 10. 快速命令

```bash
# 后端
cd backend
make build          # 构建
make run            # 运行
make test           # 测试
make fmt            # 格式化
make vet            # 静态分析
make migrate        # 迁移数据库

# 前端
cd frontend
npm run dev         # 开发
npm run build       # 构建
npm run preview     # 预览
npm run test        # 测试
npm run lint        # 代码检查

# Docker
docker-compose -f docker-compose.dev.yml up -d      # 启动
docker-compose -f docker-compose.dev.yml down       # 停止
docker-compose -f docker-compose.dev.yml logs -f    # 查看日志
```

---

**文档版本**: 1.0  
**最后更新**: 2025-11-05  
**维护者**: FusionMail 团队

