# Render 部署教程

本教程将指导你将 FusionMail 部署到 Render 平台。

## 特点

- ✅ 免费支持自定义域名 + SSL
- ✅ 自动从 Git 仓库部署
- ✅ 网络限制较少
- ❌ 需要 GitHub/GitLab 仓库
- ❌ 免费版有冷启动延迟

## 前置准备

### 1. 注册 Render 账号

访问 https://render.com 注册（支持 GitHub 登录）

### 2. 准备 Git 仓库

将 FusionMail 代码推送到 GitHub 或 GitLab：

```bash
# 创建 GitHub 仓库后
git remote add origin https://github.com/YOUR_USERNAME/fusionmail.git
git push -u origin main
```

### 3. 准备数据库

参考 [Fly.io 教程](../fly/README.md) 中的数据库准备部分。

---

## 部署步骤

### 步骤 1：创建 Web Service

1. 登录 https://dashboard.render.com
2. 点击 "New" → "Web Service"
3. 选择 "Build and deploy from a Git repository"
4. 连接你的 GitHub/GitLab 账号
5. 选择 FusionMail 仓库

### 步骤 2：配置服务

填写以下信息：

| 配置项 | 值 |
|--------|-----|
| **Name** | `fusionmail` |
| **Region** | Singapore 或其他亚太区域 |
| **Branch** | `main` |
| **Runtime** | Docker |
| **Dockerfile Path** | `scripts/deploy/render/Dockerfile` |
| **Docker Context** | `.` |

### 步骤 3：配置环境变量

在 "Environment" 部分添加：

**必需变量：**
```
DB_HOST=aws-0-xxx.pooler.supabase.com
DB_PORT=5432
DB_USER=postgres.your-project-ref
DB_PASSWORD=your-password
DB_NAME=postgres
DB_SSLMODE=require
JWT_SECRET=your-32-char-secret
ENCRYPTION_KEY=your-32-byte-key
```

**可选变量：**
```
REDIS_HOST=xxx.upstash.io
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
REDIS_TLS=true
ADMIN_PASSWORD=your-admin-password
```

### 步骤 4：创建服务

1. 选择 "Free" 计划（或付费计划）
2. 点击 "Create Web Service"
3. 等待构建和部署

首次部署约需 5-10 分钟。

### 步骤 5：验证部署

访问 Render 分配的 URL：`https://fusionmail.onrender.com`

---

## 配置自定义域名

### 步骤 1：添加域名

1. 进入服务 Settings
2. 找到 "Custom Domains"
3. 点击 "Add Custom Domain"
4. 输入你的域名

### 步骤 2：配置 DNS

根据提示添加 DNS 记录：

```
CNAME your-domain.com → fusionmail.onrender.com
```

### 步骤 3：等待验证

Render 会自动签发 SSL 证书，通常需要几分钟。

---

## 使用 Blueprint 部署

Render 支持通过 `render.yaml` 一键部署：

1. 确保仓库中有 `scripts/deploy/render/render.yaml`
2. 访问 https://dashboard.render.com/blueprints
3. 点击 "New Blueprint Instance"
4. 选择仓库
5. 配置环境变量
6. 部署

---

## 常用操作

### 查看日志

1. 进入服务 Dashboard
2. 点击 "Logs" 标签

### 手动部署

1. 进入服务 Dashboard
2. 点击 "Manual Deploy" → "Deploy latest commit"

### 重启服务

1. 进入服务 Settings
2. 点击 "Suspend Service" 然后 "Resume Service"

### 更新环境变量

1. 进入服务 Settings → Environment
2. 修改变量
3. 服务会自动重启

---

## 故障排查

### 构建失败

1. 检查 Dockerfile 路径是否正确
2. 查看构建日志中的错误
3. 确认 Docker Context 设置为 `.`

### 启动失败

1. 检查环境变量是否完整
2. 查看运行时日志
3. 确认数据库连接信息正确

### 冷启动慢

Render 免费版在无访问时会休眠，首次访问需要等待启动（约 30 秒）。

解决方案：
- 升级到付费计划
- 使用外部监控服务定期访问

---

## 费用说明

**免费计划：**
- 750 小时/月运行时间
- 自动休眠（无访问 15 分钟后）
- 100GB 带宽/月

**付费计划（$7/月起）：**
- 不休眠
- 更多资源
- 更快的构建

---

## 下一步

1. 访问你的 Render URL
2. 使用 `admin` 和设置的密码登录
3. 添加邮箱账户开始使用

如有问题，查看 Render 日志或提交 Issue。
