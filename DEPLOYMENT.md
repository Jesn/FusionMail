# 🚀 FusionMail 部署指南

欢迎使用 FusionMail！本文档将指导你快速部署 FusionMail 到各种平台。

## 📍 部署文档位置

所有部署相关的文档和配置文件都位于：

```
📁 scripts/deploy/
```

👉 **[点击这里查看完整部署指南](scripts/deploy/README.md)**

---

## ⚡ 快速开始

### 1️⃣ 选择部署平台

| 平台 | 适合场景 | 部署时间 | 费用 |
|------|----------|----------|------|
| **[Fly.io](scripts/deploy/fly/README.md)** | 生产使用 | 5 分钟 | 免费额度 |
| **[HuggingFace](scripts/deploy/huggingface/TUTORIAL.md)** | 快速体验 | 3 分钟 | 完全免费 |
| **[Docker](scripts/deploy/docker/TUTORIAL.md)** | 自有服务器 | 10 分钟 | 服务器费用 |
| **[Render](scripts/deploy/render/TUTORIAL.md)** | 简单部署 | 8 分钟 | 免费额度 |

### 2️⃣ 准备数据库

**推荐：Supabase（免费）**
1. 访问 [supabase.com](https://supabase.com) 创建项目
2. 获取 Session Pooler 连接信息
3. 用户名格式：`postgres.{project-ref}`

### 3️⃣ 开始部署

选择你的平台，跟随对应教程：

- 🌟 **推荐新手**：[HuggingFace 教程](scripts/deploy/huggingface/TUTORIAL.md)
- 🚀 **推荐生产**：[Fly.io 教程](scripts/deploy/fly/README.md)
- 🔧 **自建服务器**：[Docker 教程](scripts/deploy/docker/TUTORIAL.md)
- 📦 **Git 部署**：[Render 教程](scripts/deploy/render/TUTORIAL.md)

---

## 📚 详细文档

### 部署教程
- [📖 部署指南总览](scripts/deploy/README.md) - 选择最适合的部署方式
- [🔧 技术文档](scripts/deploy/DEPLOYMENT.md) - 详细配置和故障排查

### 平台特定教程
- [✈️ Fly.io 部署](scripts/deploy/fly/README.md) - 推荐生产使用
- [🤗 HuggingFace 部署](scripts/deploy/huggingface/TUTORIAL.md) - 快速体验
- [🎨 Render 部署](scripts/deploy/render/TUTORIAL.md) - Git 自动部署
- [🐳 Docker 部署](scripts/deploy/docker/TUTORIAL.md) - 本地/VPS 部署

---

## 🎯 推荐部署路径

### 🆕 第一次使用？

```
1. 快速体验（3分钟）
   👉 HuggingFace Spaces 部署

2. 满意后升级到生产环境
   👉 Fly.io 部署 + 自定义域名
```

### 💼 团队/企业使用？

```
1. 自有服务器部署
   👉 Docker 部署

2. 配置域名和 SSL
   👉 Nginx + Let's Encrypt
```

### 👨‍💻 开发者？

```
1. Git 自动部署
   👉 Render 部署

2. 或本地开发环境
   👉 Docker Compose
```

---

## 🔧 环境变量配置

所有平台都需要配置这些环境变量：

```env
# 数据库（必需）
DB_HOST=your-database-host
DB_USER=postgres.your-project-ref  # 注意 Supabase 格式
DB_PASSWORD=your-password

# 安全（必需）
JWT_SECRET=your-32-char-secret
ENCRYPTION_KEY=your-32-byte-key

# Redis（可选但推荐）
REDIS_HOST=your-redis-host
REDIS_PASSWORD=your-redis-password
REDIS_TLS=true

# 管理员（可选）
ADMIN_PASSWORD=your-admin-password
```

**生成密钥：**
```bash
# JWT 密钥
openssl rand -base64 32

# 加密密钥
openssl rand -base64 32 | cut -c1-32
```

---

## 🌐 自定义域名

| 平台 | 域名支持 | 配置方法 |
|------|----------|----------|
| Fly.io | ✅ 免费 | `flyctl certs add your-domain.com` |
| Render | ✅ 免费 | Dashboard → Custom Domains |
| Docker | ✅ 自配置 | Nginx + Let's Encrypt |
| HuggingFace | ❌ 需要 Pro | 付费功能 |

---

## 🆘 遇到问题？

### 常见问题
1. **数据库连接失败** → 检查用户名格式（Supabase 需要 `postgres.project-ref`）
2. **邮箱连接超时** → HuggingFace 网络受限，改用 Fly.io
3. **健康检查失败** → 等待数据库初始化完成

### 获取帮助
1. 📖 查看 [详细故障排查](scripts/deploy/DEPLOYMENT.md#故障排查)
2. 📋 检查部署日志中的错误信息
3. 🐛 提交 [GitHub Issue](https://github.com/your-repo/fusionmail/issues)

---

## 🎉 部署成功后

1. 🌐 访问你的应用 URL
2. 👤 使用 `admin` 账号登录
3. 📧 添加邮箱账户（支持 Gmail、Outlook、QQ、163 等）
4. 🏷️ 配置邮件规则和标签
5. 🚀 享受统一的邮件管理！

---

**🚀 现在就开始：[选择部署平台](scripts/deploy/README.md)**
