# Fly.io 部署教程

本教程将指导你将 FusionMail 部署到 Fly.io 平台。

## 为什么选择 Fly.io？

- ✅ 支持本地直接推送，无需 GitHub
- ✅ 免费支持自定义域名 + SSL
- ✅ 网络限制少，可连接国内邮箱服务器
- ✅ 全球多区域部署（推荐东京 nrt）
- ✅ 免费额度：3 个共享 CPU VM

## 前置准备

### 1. 注册 Fly.io 账号

访问 https://fly.io 注册账号（支持 GitHub 登录）

### 2. 安装 Fly CLI

```bash
# macOS
brew install flyctl

# Linux
curl -L https://fly.io/install.sh | sh

# Windows
powershell -Command "iwr https://fly.io/install.ps1 -useb | iex"
```

### 3. 登录

```bash
flyctl auth login
```

### 4. 准备数据库

推荐使用 Supabase（免费）：
1. 访问 https://supabase.com 创建项目
2. 进入 Settings → Database → Connection Pooling
3. 选择连接模式（见下方说明），记录连接信息
4. 注意用户名格式：`postgres.{project-ref}`

#### Supabase 连接模式选择

| 模式 | 端口 | 特点 | 推荐场景 |
|------|------|------|----------|
| **Session** | 5432 | 每个连接独立会话，支持 prepared statements | 连接数充足时 |
| **Transaction** | 6543 | 连接复用，不支持 prepared statements | 连接数受限时（免费版推荐） |

**⚠️ Transaction 模式关键配置：**

Supabase Transaction 模式（端口 6543）不支持 prepared statements，需要特殊配置：

```bash
# fly.toml 中必须设置
DB_PORT = "6543"
DB_DISABLE_PREPARE_STMT = "true"
DB_MAX_IDLE_CONNS = "2"
DB_MAX_OPEN_CONNS = "10"
DB_CONN_MAX_LIFETIME = "30"
```

**技术原理：**
- Transaction 模式的连接池会在事务结束后将连接归还池中
- Prepared statements 绑定到特定连接，连接复用后会报错：`prepared statement does not exist`
- 解决方案需要两步：
  1. GORM 配置 `PrepareStmt: false`
  2. DSN 添加 `default_query_exec_mode=exec`（禁用 pgx 驱动的 prepared statements）

可选 Redis（Upstash 免费）：
1. 访问 https://upstash.com 创建数据库
2. 启用 TLS，记录连接信息

---

## 部署步骤

### 步骤 1：创建应用

```bash
flyctl apps create fusionmail
```

### 步骤 2：创建存储卷

```bash
# 在东京区域创建 1GB 存储卷
flyctl volumes create fusionmail_data --region nrt --size 1 -a fusionmail -y
```

### 步骤 3：配置环境变量

> ⚠️ **时区说明**：fly.toml 中配置 `TZ=UTC`，所有内部时间处理统一使用 UTC。这是分布式部署的最佳实践，支持跨国多节点部署。前端会自动将时间转换为用户本地时区显示。

> 📝 **日志级别**：fly.toml 中默认配置 `LOG_LEVEL=info`，生产环境使用 info 级别可减少 IO 开销。如需调试可临时设置为 `debug`。

```bash
# 数据库配置（必需）
flyctl secrets set -a fusionmail \
  DB_HOST=aws-1-ap-northeast-1.pooler.supabase.com \
  DB_USER=postgres.your-project-ref \
  DB_NAME=postgres \
  DB_SSLMODE=require

# 数据库密码（单独设置，避免特殊字符问题）
flyctl secrets set -a fusionmail 'DB_PASSWORD=your-db-password'

# 安全配置（必需）
flyctl secrets set -a fusionmail 'JWT_SECRET=your-jwt-secret-at-least-32-chars'
flyctl secrets set -a fusionmail ENCRYPTION_KEY=your-32-byte-encryption-key

# Redis 配置（可选，支持 Upstash/Aiven 等云服务）
# Upstash 示例：
flyctl secrets set -a fusionmail \
  REDIS_HOST=your-redis-host.upstash.io \
  REDIS_TLS=true
flyctl secrets set -a fusionmail REDIS_PASSWORD=your-redis-password

# Aiven Valkey 示例（注意需要设置 REDIS_USER）：
flyctl secrets set -a fusionmail \
  REDIS_HOST=valkey-xxx.aivencloud.com \
  REDIS_PORT=16559 \
  REDIS_USER=default \
  REDIS_TLS=true
flyctl secrets set -a fusionmail REDIS_PASSWORD=your-aiven-password

# 管理员密码（可选，不设置则自动生成）
flyctl secrets set -a fusionmail 'ADMIN_PASSWORD=your-admin-password'
```

**生成密钥的方法：**
```bash
# JWT_SECRET
openssl rand -base64 32

# ENCRYPTION_KEY（必须 32 字节）
openssl rand -base64 32 | cut -c1-32
```

### 步骤 4：部署应用

```bash
# 在项目根目录执行
flyctl deploy -a fusionmail --remote-only
```

首次部署需要构建镜像，大约需要 3-5 分钟。

### 步骤 5：验证部署

```bash
# 查看状态
flyctl status -a fusionmail

# 查看日志
flyctl logs -a fusionmail

# 健康检查
curl https://fusionmail.fly.dev/api/v1/health
```

---

## 配置自定义域名

### 步骤 1：添加证书

```bash
flyctl certs add your-domain.com -a fusionmail
```

### 步骤 2：配置 DNS

根据提示添加 DNS 记录：

**方式 A：A + AAAA 记录（推荐）**
```
A    @ → 提示的 IPv4 地址
AAAA @ → 提示的 IPv6 地址
```

**方式 B：CNAME 记录**
```
CNAME @ → xxx.fusionmail.fly.dev
```

### 步骤 3：验证证书

```bash
flyctl certs check your-domain.com -a fusionmail
```

证书签发通常需要几分钟。

---

## 常用命令

```bash
# 查看应用状态
flyctl status -a fusionmail

# 查看实时日志
flyctl logs -a fusionmail

# 查看历史日志
flyctl logs -a fusionmail --no-tail

# 重启应用
flyctl apps restart fusionmail

# SSH 连接到容器
flyctl ssh console -a fusionmail

# 查看环境变量
flyctl secrets list -a fusionmail

# 更新环境变量
flyctl secrets set -a fusionmail KEY=value

# 删除环境变量
flyctl secrets unset -a fusionmail KEY

# 临时开启调试日志（排查问题时使用）
flyctl secrets set -a fusionmail LOG_LEVEL=debug
# 恢复生产日志级别
flyctl secrets set -a fusionmail LOG_LEVEL=info

# 扩容/缩容
flyctl scale count 2 -a fusionmail

# 查看机器列表
flyctl machines list -a fusionmail

# 销毁机器
flyctl machines destroy MACHINE_ID -a fusionmail --force
```

---

## 区域选择

```bash
# 查看可用区域
flyctl platform regions
```

推荐区域：
- `nrt` - 东京（连接亚太数据库最快）
- `sin` - 新加坡
- `hkg` - 香港（如果可用）

---

## 故障排查

### 数据库连接失败

1. 检查 `DB_HOST` 是否正确
2. Supabase Session Pooler 用户名格式：`postgres.{project-ref}`
3. 确认 `DB_SSLMODE=require`

```bash
# 查看日志中的数据库错误
flyctl logs -a fusionmail | grep -i "database\|postgres"
```

### Prepared Statement 错误

如果日志中出现以下错误：
```
ERROR: prepared statement "stmtcache_xxx" already exists (SQLSTATE 42P05)
ERROR: prepared statement "stmtcache_xxx" does not exist (SQLSTATE 26000)
```

**原因：** 使用了 Supabase Transaction 模式（端口 6543），但未正确禁用 prepared statements。

**解决方案：**

1. 确认 fly.toml 中设置了：
```toml
[env]
  DB_PORT = "6543"
  DB_DISABLE_PREPARE_STMT = "true"
```

2. 重新部署：
```bash
flyctl deploy -a fusionmail --remote-only --no-cache
```

3. 验证配置生效（查看启动日志）：
```bash
flyctl logs -a fusionmail | grep "DisablePrepareStmt"
# 应该看到: DisablePrepareStmt=true
```

### 健康检查失败

1. 查看日志确认启动错误
2. 可能是数据库初始化慢，等待几分钟
3. 检查 fly.toml 中的健康检查超时设置

### 部署失败

```bash
# 查看构建日志
flyctl logs -a fusionmail --no-tail | head -100

# 强制重新部署
flyctl deploy -a fusionmail --remote-only --strategy immediate
```

### 租约冲突

```bash
# 销毁旧机器
flyctl machines destroy MACHINE_ID -a fusionmail --force

# 重新部署
flyctl deploy -a fusionmail --remote-only
```

---

## 费用说明

Fly.io 免费额度：
- 3 个共享 CPU VM（256MB 内存）
- 3GB 持久化存储
- 160GB 出站流量/月

FusionMail 单实例部署在免费额度内。

---

## 下一步

1. 访问 https://fusionmail.fly.dev 或你的自定义域名
2. 使用 `admin` 和设置的密码登录
3. 添加邮箱账户开始使用

如有问题，查看日志或提交 Issue。
