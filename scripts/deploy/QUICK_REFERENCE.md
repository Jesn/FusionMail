# FusionMail 部署快速参考

## 🔑 密钥生成

```bash
# JWT 密钥（至少 32 字符）
openssl rand -base64 32

# 加密密钥（必须 32 字节）
openssl rand -base64 32 | cut -c1-32
```

---

## 📊 环境变量速查

### 必需变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `DB_HOST` | 数据库主机 | `aws-0-xxx.pooler.supabase.com` |
| `DB_USER` | 数据库用户 | `postgres.your-project-ref` |
| `DB_PASSWORD` | 数据库密码 | `your-password` |
| `JWT_SECRET` | JWT 密钥 | 32+ 字符随机字符串 |
| `ENCRYPTION_KEY` | 加密密钥 | 32 字节随机字符串 |

### 常用可选变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_PORT` | `5432` | 数据库端口 |
| `DB_NAME` | `postgres` | 数据库名 |
| `DB_SSLMODE` | `require` | SSL 模式 |
| `REDIS_HOST` | - | Redis 主机 |
| `REDIS_PASSWORD` | - | Redis 密码 |
| `REDIS_TLS` | `false` | Redis TLS |
| `ADMIN_PASSWORD` | 自动生成 | 管理员密码 |

---

## 🚀 Fly.io 命令速查

```bash
# 安装 CLI
brew install flyctl

# 登录
flyctl auth login

# 创建应用
flyctl apps create fusionmail

# 创建存储卷
flyctl volumes create fusionmail_data --region nrt --size 1 -a fusionmail -y

# 设置环境变量
flyctl secrets set -a fusionmail KEY=value
flyctl secrets set -a fusionmail 'KEY=value with spaces'

# 部署
flyctl deploy -a fusionmail --remote-only

# 查看状态
flyctl status -a fusionmail

# 查看日志
flyctl logs -a fusionmail

# 重启
flyctl apps restart fusionmail

# SSH 连接
flyctl ssh console -a fusionmail

# 添加域名
flyctl certs add your-domain.com -a fusionmail

# 检查证书
flyctl certs check your-domain.com -a fusionmail
```

---

## 🤗 HuggingFace 命令速查

```bash
# 设置 Token
export HF_TOKEN=hf_xxxxxxxx

# 一键部署
./scripts/deploy/huggingface/deploy-to-hf.sh

# 手动推送
git clone https://huggingface.co/spaces/YOUR_USERNAME/FusionMail hf-space
cd hf-space
# 复制文件...
git add . && git commit -m "Deploy" && git push
```

---

## 🐳 Docker 命令速查

```bash
# 启动
docker compose -f scripts/deploy/docker/docker-compose.prod.yml up -d

# 停止
docker compose -f scripts/deploy/docker/docker-compose.prod.yml down

# 查看日志
docker compose -f scripts/deploy/docker/docker-compose.prod.yml logs -f

# 重启
docker compose -f scripts/deploy/docker/docker-compose.prod.yml restart

# 进入容器
docker exec -it fusionmail sh

# 重新构建
docker compose -f scripts/deploy/docker/docker-compose.prod.yml up -d --build
```

---

## 📧 邮箱服务器配置

### Gmail
- IMAP: `imap.gmail.com:993` (SSL)
- SMTP: `smtp.gmail.com:587` (TLS)

### Outlook/Hotmail
- IMAP: `outlook.office365.com:993` (SSL)
- SMTP: `smtp.office365.com:587` (TLS)

### QQ 邮箱
- IMAP: `imap.qq.com:993` (SSL)
- SMTP: `smtp.qq.com:587` (TLS)

### 163 邮箱
- IMAP: `imap.163.com:993` (SSL)
- SMTP: `smtp.163.com:465` (SSL)

### iCloud
- IMAP: `imap.mail.me.com:993` (SSL)
- SMTP: `smtp.mail.me.com:587` (TLS)

---

## 🔧 故障排查速查

### 数据库连接失败

```bash
# 检查连接
nc -zv your-db-host 5432

# Supabase 用户名格式
# ✅ postgres.your-project-ref
# ❌ postgres
```

### 健康检查

```bash
# 基础检查
curl https://your-domain.com/api/v1/health

# 详细检查
curl https://your-domain.com/api/v1/health/detailed
```

### 查看日志

```bash
# Fly.io
flyctl logs -a fusionmail

# Docker
docker logs fusionmail

# HuggingFace
# 在 Space 页面点击 "See logs"
```

---

## 🌐 DNS 配置

### A 记录
```
类型: A
主机: @
值: Fly.io 提供的 IP
```

### CNAME 记录
```
类型: CNAME
主机: @
值: fusionmail.fly.dev
```

---

## 📋 部署检查清单

- [ ] 数据库已创建并获取连接信息
- [ ] 用户名格式正确（Supabase: `postgres.xxx`）
- [ ] JWT_SECRET 已生成（≥32 字符）
- [ ] ENCRYPTION_KEY 已生成（32 字节）
- [ ] 环境变量已配置
- [ ] 部署成功
- [ ] 健康检查通过
- [ ] 可以登录管理后台
- [ ] 可以添加邮箱账户

---

## 🆘 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| `Tenant or user not found` | Supabase 用户名格式错误 | 使用 `postgres.{project-ref}` |
| `dial tcp xxx:993: i/o timeout` | 网络受限 | 使用 Fly.io 或 Docker |
| `health check failed` | 启动慢或数据库连接失败 | 检查日志和数据库配置 |
| `invalid JWT secret` | JWT_SECRET 太短 | 使用 ≥32 字符的密钥 |
| `encryption key must be 32 bytes` | ENCRYPTION_KEY 长度错误 | 使用正好 32 字节的密钥 |

---

## 📚 详细文档

- [部署指南总览](README.md)
- [技术文档](DEPLOYMENT.md)
- [Fly.io 教程](fly/README.md)
- [HuggingFace 教程](huggingface/TUTORIAL.md)
- [Render 教程](render/TUTORIAL.md)
- [Docker 教程](docker/TUTORIAL.md)
