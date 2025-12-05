# FusionMail 部署指南

欢迎使用 FusionMail！本文档帮助你选择最适合的部署方式。

## 🚀 快速选择

| 平台 | 适用场景 | 优势 | 限制 | 推荐度 |
|------|----------|------|------|--------|
| [Fly.io](fly/README.md) | 生产使用 | 网络无限制、免费域名、本地推送 | 需要信用卡验证 | ⭐⭐⭐⭐⭐ |
| [Docker](docker/TUTORIAL.md) | 自有服务器 | 完全控制、无限制、数据私有 | 需要服务器 | ⭐⭐⭐⭐ |
| [Render](render/TUTORIAL.md) | Git 部署 | 免费域名、自动部署 | 需要 GitHub、有冷启动 | ⭐⭐⭐ |
| [HuggingFace](huggingface/TUTORIAL.md) | 快速体验 | 完全免费、无需信用卡 | 网络受限、无自定义域名 | ⭐⭐ |

---

## 📋 部署前准备

### 1. 数据库（必需）

**推荐：Supabase（免费）**

1. 访问 [supabase.com](https://supabase.com) 创建项目
2. 进入 Settings → Database → Connection Pooling
3. 选择 **Session pooler**
4. 记录连接信息：
   - Host: `aws-0-xxx.pooler.supabase.com`
   - User: `postgres.{project-ref}` ⚠️ 注意格式！
   - Password: 你设置的密码

**其他选择：**
- [Neon](https://neon.tech) - 免费 PostgreSQL
- [Railway](https://railway.app) - 简单易用
- 自建 PostgreSQL

### 2. Redis（可选但推荐）

**推荐：Upstash（免费）**

1. 访问 [upstash.com](https://upstash.com) 创建数据库
2. 启用 TLS
3. 记录连接信息

### 3. 生成密钥

```bash
# JWT 密钥（至少 32 字符）
openssl rand -base64 32

# 加密密钥（必须 32 字节）
openssl rand -base64 32 | cut -c1-32
```

---

## 🎯 推荐部署路径

### 🆕 新手用户 → HuggingFace

- ✅ 完全免费，无需信用卡
- ✅ 5 分钟快速体验
- ❌ 只能连接 Gmail、Outlook 等国际邮箱

👉 [HuggingFace 部署教程](huggingface/TUTORIAL.md)

### 👤 个人用户 → Fly.io

- ✅ 免费额度充足
- ✅ 支持所有邮箱（包括 QQ、163）
- ✅ 免费自定义域名 + SSL
- ✅ 本地直接推送，无需 GitHub

👉 [Fly.io 部署教程](fly/README.md)

### 👥 团队用户 → Docker VPS

- ✅ 完全控制和隐私
- ✅ 无任何限制
- ✅ 可自定义配置

👉 [Docker 部署教程](docker/TUTORIAL.md)

### 👨‍💻 开发者 → Render

- ✅ Git 自动部署
- ✅ 免费域名
- ❌ 需要 GitHub 仓库

👉 [Render 部署教程](render/TUTORIAL.md)

---

## ⚡ 5 分钟快速部署

### 选项 1：HuggingFace（最简单）

```bash
# 1. 设置 Token
export HF_TOKEN=hf_xxxxxxxx

# 2. 一键部署
./scripts/deploy/huggingface/deploy-to-hf.sh
```

### 选项 2：Fly.io（推荐）

```bash
# 1. 安装 CLI
brew install flyctl

# 2. 登录
flyctl auth login

# 3. 创建应用
flyctl apps create fusionmail

# 4. 创建存储卷
flyctl volumes create fusionmail_data --region nrt --size 1 -a fusionmail -y

# 5. 配置环境变量
flyctl secrets set -a fusionmail \
  DB_HOST=your-db-host \
  DB_USER=postgres.your-project-ref \
  DB_NAME=postgres \
  DB_SSLMODE=require
flyctl secrets set -a fusionmail 'DB_PASSWORD=your-password'
flyctl secrets set -a fusionmail 'JWT_SECRET=your-jwt-secret'
flyctl secrets set -a fusionmail ENCRYPTION_KEY=your-encryption-key

# 6. 部署
flyctl deploy -a fusionmail --remote-only
```

---

## 🔧 环境变量配置

所有平台都需要配置以下环境变量：

### 必需变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `DB_HOST` | 数据库主机 | `aws-0-xxx.pooler.supabase.com` |
| `DB_USER` | 数据库用户 | `postgres.your-project-ref` |
| `DB_PASSWORD` | 数据库密码 | `your-password` |
| `JWT_SECRET` | JWT 密钥（≥32字符） | `openssl rand -base64 32` |
| `ENCRYPTION_KEY` | 加密密钥（32字节） | `openssl rand -base64 32 \| cut -c1-32` |

### 可选变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_PORT` | `5432` | 数据库端口 |
| `DB_NAME` | `postgres` | 数据库名 |
| `DB_SSLMODE` | `require` | SSL 模式 |
| `REDIS_HOST` | - | Redis 主机 |
| `REDIS_PASSWORD` | - | Redis 密码 |
| `REDIS_TLS` | `false` | Redis TLS（Upstash 需要 `true`） |
| `ADMIN_PASSWORD` | 自动生成 | 管理员密码 |

---

## 🌐 自定义域名

| 平台 | 支持 | 配置方法 |
|------|------|----------|
| Fly.io | ✅ 免费 | `flyctl certs add your-domain.com` |
| Render | ✅ 免费 | Dashboard → Custom Domains |
| Docker | ✅ 自配置 | Nginx + Let's Encrypt |
| HuggingFace | ❌ 需要 Pro | 付费功能 |

---

## 🚨 常见问题

### 数据库连接失败

**错误**: `Tenant or user not found`

**解决**: Supabase Session Pooler 用户名格式必须是 `postgres.{project-ref}`，不是简单的 `postgres`

### 邮箱连接超时

**错误**: `dial tcp xxx:993: i/o timeout`

**原因**: HuggingFace 网络受限

**解决**: 使用 Fly.io 或 Docker 部署

### 健康检查失败

**原因**: 数据库初始化慢

**解决**: 等待几分钟或检查数据库连接

---

## 📚 详细文档

### 平台教程
- [Fly.io 部署教程](fly/README.md) - 推荐生产使用
- [HuggingFace 部署教程](huggingface/TUTORIAL.md) - 快速体验
- [Render 部署教程](render/TUTORIAL.md) - Git 自动部署
- [Docker 部署教程](docker/TUTORIAL.md) - 自建服务器

### 参考文档
- [技术文档](DEPLOYMENT.md) - 详细配置、邮箱设置、OAuth2、故障排查
- [快速参考](QUICK_REFERENCE.md) - 命令速查、环境变量、常见错误

---

## 🎉 部署成功后

1. 访问你的应用 URL
2. 使用 `admin` 和设置的密码登录
3. 添加邮箱账户：
   - Gmail（推荐 OAuth2）
   - Outlook（推荐 OAuth2）
   - QQ、163（需要应用密码）
4. 配置邮件规则和标签
5. 享受统一的邮件管理体验！

---

**选择困难？推荐顺序：Fly.io → Docker → Render → HuggingFace**
