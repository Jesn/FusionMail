# FusionMail 部署指南

本文档介绍如何将 FusionMail 部署到不同平台。

## 目录结构

```
scripts/deploy/
├── DEPLOYMENT.md          # 本文档
├── fly/                   # Fly.io 部署配置
│   ├── fly.toml          # Fly.io 配置文件
│   ├── Dockerfile        # Fly.io 专用 Dockerfile
│   ├── nginx.conf        # Nginx 配置
│   └── start.sh          # 启动脚本
├── huggingface/          # HuggingFace Spaces 部署配置
│   ├── Dockerfile        # HF 专用 Dockerfile
│   ├── nginx.conf        # Nginx 配置
│   ├── start.sh          # 启动脚本
│   ├── README.md         # Space 配置
│   └── deploy-to-hf.sh   # 部署脚本
├── render/               # Render 部署配置
│   ├── Dockerfile        # Render 专用 Dockerfile
│   ├── nginx.conf        # Nginx 配置
│   ├── start.sh          # 启动脚本
│   ├── render.yaml       # Blueprint 配置
│   └── deploy-to-render.sh
└── docker/               # 通用 Docker 部署
    └── docker-compose.prod.yml
```

## 环境变量配置

所有平台都需要配置以下环境变量：

### 必需变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `DB_HOST` | PostgreSQL 主机 | `aws-1-ap-northeast-1.pooler.supabase.com` |
| `DB_USER` | 数据库用户 | `postgres.oeufkcyahfhtpemzwsdt` |
| `DB_PASSWORD` | 数据库密码 | `your-password` |
| `JWT_SECRET` | JWT 签名密钥（≥32字符） | `openssl rand -base64 32` |
| `ENCRYPTION_KEY` | 数据加密密钥（32字节） | `openssl rand -base64 32 \| cut -c1-32` |

### 可选变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DB_PORT` | `5432` | 数据库端口 |
| `DB_NAME` | `postgres` | 数据库名 |
| `DB_SSLMODE` | `require` | SSL 模式 |
| `REDIS_HOST` | - | Redis 主机（可选） |
| `REDIS_PORT` | `6379` | Redis 端口 |
| `REDIS_PASSWORD` | - | Redis 密码 |
| `REDIS_TLS` | `false` | Redis TLS |
| `ADMIN_PASSWORD` | 自动生成 | 管理员密码 |

---

## Fly.io 部署（推荐）

Fly.io 支持本地推送、自定义域名、网络限制少。

### 前置条件

```bash
# 安装 Fly CLI
brew install flyctl

# 登录
flyctl auth login
```

### 部署步骤

```bash
# 1. 进入部署目录
cd scripts/deploy/fly

# 2. 创建应用（首次）
flyctl apps create fusionmail

# 3. 创建存储卷
flyctl volumes create fusionmail_data --region nrt --size 1 -a fusionmail

# 4. 设置环境变量
flyctl secrets set -a fusionmail \
  DB_HOST=aws-1-ap-northeast-1.pooler.supabase.com \
  DB_USER=postgres.oeufkcyahfhtpemzwsdt \
  'DB_PASSWORD=your-password' \
  'JWT_SECRET=your-jwt-secret' \
  ENCRYPTION_KEY=your-32-byte-key \
  REDIS_HOST=your-redis-host \
  REDIS_PASSWORD=your-redis-password \
  REDIS_TLS=true

# 5. 部署
flyctl deploy -a fusionmail --remote-only

# 6. 添加自定义域名（可选）
flyctl certs add fusionmail.100420.xyz -a fusionmail
```

### 常用命令

```bash
# 查看状态
flyctl status -a fusionmail

# 查看日志
flyctl logs -a fusionmail

# 重启
flyctl apps restart fusionmail

# SSH 连接
flyctl ssh console -a fusionmail
```

---

## HuggingFace Spaces 部署

HuggingFace 免费版有网络限制，部分国内邮箱服务器无法连接。

### 前置条件

- HuggingFace 账号
- HuggingFace Token（Write 权限）

### 部署步骤

```bash
# 1. 设置 Token
export HF_TOKEN=hf_xxxxxxxx

# 2. 运行部署脚本
./scripts/deploy/huggingface/deploy-to-hf.sh
```

### 配置 Secrets

在 HuggingFace Space Settings 中添加环境变量。

### 限制

- 免费版不支持自定义域名
- 网络受限，无法连接部分国内邮箱（QQ、163等）
- 适合 Gmail、Outlook 等国际邮箱

---

## Render 部署

Render 需要通过 Git 仓库部署。

### 部署步骤

1. 将代码推送到 GitHub
2. 在 Render Dashboard 创建 Web Service
3. 选择 Docker 运行时
4. 设置 Dockerfile 路径：`scripts/deploy/render/Dockerfile`
5. 添加环境变量

---

## Docker 本地/VPS 部署

适合自有服务器部署。

### 部署步骤

```bash
# 1. 复制配置文件
cp scripts/deploy/docker/docker-compose.prod.yml .
cp 部署账号信息.env .env

# 2. 编辑 .env 配置

# 3. 启动
docker compose -f docker-compose.prod.yml up -d

# 4. 查看日志
docker compose -f docker-compose.prod.yml logs -f
```

---

## 数据库配置

### Supabase（推荐）

1. 创建项目：https://supabase.com
2. 获取 Session Pooler 连接信息
3. 注意用户名格式：`postgres.{project-ref}`

### Neon

1. 创建项目：https://neon.tech
2. 获取连接字符串

---

## Redis 配置（可选）

### Upstash（推荐）

1. 创建数据库：https://upstash.com
2. 启用 TLS
3. 设置 `REDIS_TLS=true`

---

## 自定义域名

### Fly.io

```bash
flyctl certs add your-domain.com -a fusionmail
```

然后添加 DNS 记录：
- A 记录：指向 Fly.io IP
- 或 CNAME：指向 `xxx.fusionmail.fly.dev`

### HuggingFace

需要 Pro 账户。

---

## 故障排查

### 数据库连接失败

1. 检查 `DB_HOST`、`DB_USER`、`DB_PASSWORD`
2. Supabase Session Pooler 用户名格式：`postgres.{project-ref}`
3. 确认 SSL 模式：`DB_SSLMODE=require`

### Redis 连接失败

1. 检查 `REDIS_HOST`、`REDIS_PASSWORD`
2. Upstash 需要 `REDIS_TLS=true`

### 健康检查失败

1. 查看日志确认启动错误
2. 增加健康检查超时时间
3. 检查数据库连接

### 邮箱连接超时

- HuggingFace 免费版网络受限
- 建议使用 Fly.io 或自有服务器

---

## 邮箱服务商配置

### Gmail（推荐 OAuth2）

**方式 1：OAuth2（推荐）**

1. 访问 [Google Cloud Console](https://console.cloud.google.com)
2. 创建项目并启用 Gmail API
3. 配置 OAuth 同意屏幕
4. 创建 OAuth 2.0 客户端 ID
5. 在 FusionMail 中使用 OAuth2 添加账户

**方式 2：应用专用密码**

1. 启用两步验证：https://myaccount.google.com/security
2. 生成应用专用密码：https://myaccount.google.com/apppasswords
3. 使用 IMAP 协议添加账户

### Outlook/Hotmail（推荐 OAuth2）

**方式 1：OAuth2（推荐）**

1. 访问 [Azure Portal](https://portal.azure.com)
2. 注册应用程序
3. 配置 API 权限（Mail.Read、Mail.ReadWrite）
4. 创建客户端密钥
5. 在 FusionMail 中使用 OAuth2 添加账户

**方式 2：应用密码**

1. 启用两步验证
2. 生成应用密码
3. 使用 IMAP 协议添加账户

### QQ 邮箱

1. 登录 QQ 邮箱网页版
2. 设置 → 账户 → POP3/IMAP/SMTP 服务
3. 开启 IMAP/SMTP 服务
4. 生成授权码
5. 使用 IMAP 协议添加账户

**IMAP 配置：**
- 服务器：`imap.qq.com`
- 端口：`993`
- 加密：SSL/TLS

### 163 邮箱

1. 登录 163 邮箱网页版
2. 设置 → POP3/SMTP/IMAP
3. 开启 IMAP/SMTP 服务
4. 设置客户端授权密码
5. 使用 IMAP 协议添加账户

**IMAP 配置：**
- 服务器：`imap.163.com`
- 端口：`993`
- 加密：SSL/TLS

### iCloud 邮箱

1. 访问 https://appleid.apple.com
2. 登录并生成应用专用密码
3. 使用 IMAP 协议添加账户

**IMAP 配置：**
- 服务器：`imap.mail.me.com`
- 端口：`993`
- 加密：SSL/TLS

---

## OAuth2 配置（可选）

如需使用 OAuth2 登录 Gmail/Outlook，需要配置以下环境变量：

### Google OAuth2

```env
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URI=https://your-domain.com/api/v1/oauth/google/callback
```

### Microsoft OAuth2

```env
MICROSOFT_CLIENT_ID=your-microsoft-client-id
MICROSOFT_CLIENT_SECRET=your-microsoft-client-secret
MICROSOFT_REDIRECT_URI=https://your-domain.com/api/v1/oauth/microsoft/callback
```

---

## 性能优化

### 数据库优化

```env
# 连接池配置
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=300
```

### 同步配置

```env
# 同步间隔（分钟）
SYNC_INTERVAL=5

# 同步工作线程数
SYNC_WORKER_COUNT=3

# 每次同步最大邮件数
SYNC_MAX_EMAILS=100
```

### 缓存配置

```env
# 邮件列表缓存时间（秒）
CACHE_EMAIL_LIST_TTL=300

# 账户信息缓存时间（秒）
CACHE_ACCOUNT_TTL=600
```

---

## 安全建议

### 生产环境必须配置

1. **使用 HTTPS**：所有平台都应启用 SSL/TLS
2. **强密码**：JWT_SECRET 和 ENCRYPTION_KEY 使用随机生成的强密码
3. **数据库 SSL**：设置 `DB_SSLMODE=require`
4. **Redis TLS**：使用 Upstash 时设置 `REDIS_TLS=true`

### 推荐配置

```env
# 启用速率限制
RATE_LIMIT_ENABLED=true

# Cookie 安全设置
COOKIE_SECURE=true
COOKIE_SAME_SITE=strict

# CORS 配置（限制允许的域名）
CORS_ALLOWED_ORIGINS=https://your-domain.com
```

### 定期维护

1. 定期更新依赖和镜像
2. 监控日志中的异常
3. 备份数据库
4. 轮换密钥（如需要）

---

## 监控和日志

### 健康检查端点

```bash
# 基础健康检查
curl https://your-domain.com/api/v1/health

# 详细健康检查（包含数据库和 Redis 状态）
curl https://your-domain.com/api/v1/health/detailed
```

### 日志级别

```env
# 日志级别：debug, info, warn, error
LOG_LEVEL=info

# 日志格式：json, text
LOG_FORMAT=json
```

### 外部监控

推荐使用以下服务监控应用状态：

- [UptimeRobot](https://uptimerobot.com) - 免费监控
- [Better Uptime](https://betteruptime.com) - 免费监控
- [Pingdom](https://www.pingdom.com) - 付费监控

---

## 备份和恢复

### 数据库备份

**Supabase 自动备份**

Supabase 免费版每天自动备份，保留 7 天。

**手动备份**

```bash
# 导出数据库
pg_dump -h your-db-host -U your-db-user -d postgres > backup.sql

# 恢复数据库
psql -h your-db-host -U your-db-user -d postgres < backup.sql
```

### 附件备份

```bash
# Fly.io 备份存储卷
flyctl ssh console -a fusionmail
tar czf /tmp/attachments.tar.gz /data/attachments
flyctl ssh sftp get /tmp/attachments.tar.gz -a fusionmail
```

---

## 升级指南

### 更新部署

**Fly.io**

```bash
git pull
flyctl deploy -a fusionmail --remote-only
```

**Docker**

```bash
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

**HuggingFace**

```bash
./scripts/deploy/huggingface/deploy-to-hf.sh
```

### 数据库迁移

FusionMail 使用 GORM 自动迁移，启动时会自动更新数据库结构。

如需手动迁移：

```bash
# 进入容器
flyctl ssh console -a fusionmail

# 运行迁移
./server migrate
```
