# FusionMail 测试指南

## 快速开始

### 1. 启动服务

```bash
# 启动 Docker 容器（PostgreSQL + Redis）
docker-compose up -d

# 启动后端服务器
cd backend
./bin/server
```

### 2. 运行自动化测试

我们提供了三个测试脚本，按需选择：

#### 方案 A：快速测试（推荐新手）

```bash
./quick-test.sh
```

**特点**：
- ✅ 最简单，一键运行
- ✅ 自动完成所有步骤
- ✅ 快速验证功能
- ⏱️ 耗时约 15 秒

**输出示例**：
```
FusionMail 快速测试
====================

检查服务器... ✓
添加账户... ✓ (abc-123-def)
测试连接... ✓
触发同步... ✓
等待同步... ✓

同步结果:
--------
  邮箱: 794382693@qq.com
  状态: success
  邮件数: 42
  未读数: 5

数据库邮件数: 42

测试完成！
```

#### 方案 B：完整测试（推荐开发者）

```bash
./test-qq-account.sh
```

**特点**：
- ✅ 详细的步骤说明
- ✅ 完整的测试流程
- ✅ 提供后续操作命令
- ⏱️ 耗时约 20 秒

**测试步骤**：
1. 检查服务器状态
2. 添加 QQ 邮箱账户
3. 查看账户详情
4. 测试 IMAP 连接
5. 手动触发同步
6. 等待同步完成
7. 查看同步状态
8. 查看同步结果

#### 方案 C：完整验证（推荐测试人员）

```bash
./test-and-verify.sh
```

**特点**：
- ✅ 最完整的测试
- ✅ 包含数据库验证
- ✅ 显示详细统计
- ✅ 可选清理旧数据
- ⏱️ 耗时约 30 秒

**额外功能**：
- 检查 Docker 容器状态
- 清理旧测试数据（可选）
- 对比同步前后的邮件数
- 显示最新的 5 封邮件
- 提供详细的数据库查询命令

## 测试账号配置

### 方式 1：使用环境变量

```bash
export TEST_EMAIL="your@qq.com"
export TEST_PASSWORD="your_authorization_code"
export TEST_PROVIDER="qq"
export TEST_PROTOCOL="imap"

# 然后运行测试
./quick-test.sh
```

### 方式 2：使用配置文件（推荐）

```bash
# 1. 复制配置示例文件
cp .test-config.example .test-config

# 2. 编辑配置文件，填入你的测试账号信息
vim .test-config

# 3. 运行测试
./quick-test.sh
```

**配置文件示例** (`.test-config`)：
```bash
EMAIL="your@qq.com"
PASSWORD="your_authorization_code"
PROVIDER="qq"
PROTOCOL="imap"
```

**注意**：
- `.test-config` 文件已添加到 `.gitignore`，不会被提交到 Git
- 请勿在公开仓库中提交包含真实账号信息的文件
- 建议使用测试专用邮箱账号

## 手动测试步骤

如果你想手动测试，可以按照以下步骤：

### 1. 检查服务器

```bash
curl http://localhost:8080/api/v1/health
```

### 2. 添加账户

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
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

记录返回的 `account_uid`。

### 3. 测试连接

```bash
curl -X POST http://localhost:8080/api/v1/accounts/{account_uid}/test
```

### 4. 触发同步

```bash
curl -X POST http://localhost:8080/api/v1/sync/accounts/{account_uid}
```

### 5. 查看结果

```bash
# 查看账户详情
curl http://localhost:8080/api/v1/accounts/{account_uid}

# 查看同步状态
curl http://localhost:8080/api/v1/sync/status
```

### 6. 验证数据库

```bash
# 查看邮件总数
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "SELECT COUNT(*) FROM emails;"

# 查看最新邮件
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "SELECT id, subject, from_address, sent_at FROM emails ORDER BY sent_at DESC LIMIT 10;"

# 查看账户信息
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "SELECT uid, email, last_sync_status, total_emails FROM accounts;"
```

## 预期结果

### 成功的测试应该显示：

1. **服务器健康检查**：返回 `{"status":"ok"}`
2. **账户添加**：返回账户 UID
3. **连接测试**：返回 `{"success":true}`
4. **同步启动**：返回 `{"message":"Sync started"}`
5. **同步状态**：`last_sync_status` 为 `success`
6. **邮件数量**：`total_emails` > 0（如果邮箱有邮件）

### 数据库验证：

- `emails` 表中应该有新增的邮件记录
- `accounts` 表中应该有账户记录
- `sync_logs` 表中应该有同步日志

## 常见问题

### Q1: 连接测试失败？

**可能原因**：
- QQ 邮箱 IMAP 服务未开启
- 授权码错误或已过期
- 网络连接问题

**解决方法**：
1. 登录 QQ 邮箱网页版
2. 设置 → 账户 → 开启 IMAP/SMTP 服务
3. 重新生成授权码
4. 检查网络连接

### Q2: 同步成功但没有邮件？

**可能原因**：
- 邮箱中没有最近 30 天的邮件
- 邮件已经在之前的同步中拉取过

**解决方法**：
- 发送一封测试邮件到该邮箱
- 再次触发同步

### Q3: 数据库查询失败？

**可能原因**：
- PostgreSQL 容器未运行
- 容器名称不匹配

**解决方法**：
```bash
# 检查容器状态
docker ps

# 如果容器名称不同，修改命令中的容器名
docker exec <your-container-name> psql -U fusionmail -d fusionmail -c "SELECT COUNT(*) FROM emails;"
```

### Q4: 服务器未运行？

**解决方法**：
```bash
# 检查是否已编译
cd backend
ls -la bin/server

# 如果没有，先编译
go build -o bin/server cmd/server/main.go

# 启动服务器
./bin/server
```

## 测试数据清理

### 清理测试账户

```bash
# 获取所有账户
curl http://localhost:8080/api/v1/accounts

# 删除指定账户
curl -X DELETE http://localhost:8080/api/v1/accounts/{account_uid}
```

### 清理数据库

```bash
# 清理所有邮件
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "TRUNCATE emails CASCADE;"

# 清理所有账户
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "TRUNCATE accounts CASCADE;"

# 清理同步日志
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "TRUNCATE sync_logs CASCADE;"
```

## 性能测试

### 测试同步性能

```bash
# 记录开始时间
START_TIME=$(date +%s)

# 触发同步
curl -X POST http://localhost:8080/api/v1/sync/accounts/{account_uid}

# 等待同步完成
sleep 15

# 记录结束时间
END_TIME=$(date +%s)

# 计算耗时
DURATION=$((END_TIME - START_TIME))
echo "同步耗时: ${DURATION} 秒"

# 查看同步的邮件数
curl http://localhost:8080/api/v1/accounts/{account_uid} | grep total_emails
```

### 测试并发同步

```bash
# 添加多个账户后
curl -X POST http://localhost:8080/api/v1/sync/all

# 查看同步状态
curl http://localhost:8080/api/v1/sync/status
```

## 日志查看

### 服务器日志

服务器日志会输出到控制台，包含：
- 连接状态
- 同步进度
- 错误信息
- 性能统计

### 数据库日志

```bash
# 查看同步日志
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c \
  "SELECT account_uid, sync_type, status, emails_fetched, emails_new, duration_ms, started_at 
   FROM sync_logs 
   ORDER BY started_at DESC 
   LIMIT 10;"
```

## 下一步

测试成功后，你可以：

1. **开发邮件查询 API**：实现邮件列表、详情、搜索等功能
2. **开发前端界面**：创建邮件管理界面
3. **添加更多邮箱**：测试其他邮箱提供商（Gmail、163、iCloud 等）
4. **性能优化**：优化同步速度和数据库查询
5. **功能扩展**：实现规则引擎、Webhook 等高级功能

## 相关文档

- [API 使用示例](./api-examples.md)
- [项目设计文档](../.kiro/specs/fusionmail/design.md)
- [任务清单](../.kiro/specs/fusionmail/tasks.md)
