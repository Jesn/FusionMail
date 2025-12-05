# HuggingFace Spaces 部署教程

本教程将指导你将 FusionMail 部署到 HuggingFace Spaces。

## 适用场景

- ✅ 快速体验和演示
- ✅ 使用 Gmail、Outlook 等国际邮箱
- ❌ 不适合连接国内邮箱（QQ、163 等，网络受限）
- ❌ 免费版不支持自定义域名

## 前置准备

### 1. 注册 HuggingFace 账号

访问 https://huggingface.co 注册账号

### 2. 创建 Access Token

1. 访问 https://huggingface.co/settings/tokens
2. 点击 "New token"
3. 选择 **Write** 权限
4. 保存 Token（格式：`hf_xxxxxxxx`）

### 3. 准备数据库

**PostgreSQL（必需）- 推荐 Supabase**

1. 访问 https://supabase.com 创建项目
2. 进入 Settings → Database → Connection Pooling
3. 选择 **Session pooler**
4. 记录连接信息：
   - Host: `aws-0-xxx.pooler.supabase.com`
   - User: `postgres.{project-ref}`
   - Password: 你设置的密码
   - Database: `postgres`

**Redis（可选）- 推荐 Upstash**

1. 访问 https://upstash.com 创建数据库
2. 启用 TLS
3. 记录连接信息

---

## 部署步骤

### 步骤 1：创建 Space

1. 访问 https://huggingface.co/new-space
2. 填写信息：
   - **Space name**: `FusionMail`
   - **License**: MIT
   - **SDK**: Docker
   - **Visibility**: Public（免费版）
3. 点击 "Create Space"

### 步骤 2：配置 Secrets

1. 进入 Space Settings（齿轮图标）
2. 找到 "Repository secrets" 部分
3. 添加以下环境变量：

| 名称 | 值 | 说明 |
|------|-----|------|
| `DB_HOST` | `aws-0-xxx.pooler.supabase.com` | 数据库主机 |
| `DB_PORT` | `5432` | 数据库端口 |
| `DB_USER` | `postgres.your-project-ref` | 数据库用户 |
| `DB_PASSWORD` | `your-password` | 数据库密码 |
| `DB_NAME` | `postgres` | 数据库名 |
| `DB_SSLMODE` | `require` | SSL 模式 |
| `JWT_SECRET` | `your-32-char-secret` | JWT 密钥 |
| `ENCRYPTION_KEY` | `your-32-byte-key` | 加密密钥 |
| `ADMIN_PASSWORD` | `your-admin-password` | 管理员密码 |
| `REDIS_HOST` | `xxx.upstash.io` | Redis 主机（可选） |
| `REDIS_PORT` | `6379` | Redis 端口 |
| `REDIS_PASSWORD` | `your-redis-password` | Redis 密码 |
| `REDIS_TLS` | `true` | Redis TLS |

**生成密钥：**
```bash
# JWT_SECRET
openssl rand -base64 32

# ENCRYPTION_KEY（必须 32 字节）
openssl rand -base64 32 | cut -c1-32
```

### 步骤 3：部署代码

**方式 A：使用部署脚本（推荐）**

```bash
# 设置 Token
export HF_TOKEN=hf_xxxxxxxx

# 运行部署脚本
./scripts/deploy/huggingface/deploy-to-hf.sh
```

**方式 B：手动 Git 推送**

```bash
# 克隆 Space
git clone https://huggingface.co/spaces/YOUR_USERNAME/FusionMail hf-space
cd hf-space

# 复制文件
cp ../scripts/deploy/huggingface/Dockerfile .
cp ../scripts/deploy/huggingface/README.md .
cp ../scripts/deploy/huggingface/nginx.conf .
cp ../scripts/deploy/huggingface/start.sh .
cp -r ../backend .
cp -r ../frontend .

# 推送
git add .
git commit -m "Deploy FusionMail"
git push
```

### 步骤 4：等待构建

1. 访问你的 Space 页面
2. 查看 "Building" 状态
3. 构建完成后自动启动

首次构建约需 5-10 分钟。

### 步骤 5：验证部署

访问 `https://YOUR_USERNAME-fusionmail.hf.space`

---

## 查看日志

1. 进入 Space 页面
2. 点击右上角 "..." → "See logs"
3. 或访问 `https://huggingface.co/spaces/YOUR_USERNAME/FusionMail/logs`

---

## 常见问题

### 构建失败

1. 检查 Dockerfile 语法
2. 查看构建日志中的错误信息
3. 确认所有文件已正确上传

### 运行时错误

1. 检查 Secrets 是否正确配置
2. 查看日志中的错误信息
3. 确认数据库连接信息正确

### 数据库连接失败

错误信息：`Tenant or user not found`

解决方案：
- Supabase Session Pooler 用户名格式：`postgres.{project-ref}`
- 不是简单的 `postgres`

### 邮箱连接超时

错误信息：`dial tcp xxx:993: i/o timeout`

原因：HuggingFace 免费版网络受限，无法连接部分服务器

解决方案：
- 使用 Gmail、Outlook 等国际邮箱
- 或改用 Fly.io 部署

### 应用休眠

HuggingFace 免费版会在无访问时休眠，首次访问需要等待启动。

---

## 重新部署

```bash
export HF_TOKEN=hf_xxxxxxxx
./scripts/deploy/huggingface/deploy-to-hf.sh
```

或在 Space Settings 中点击 "Factory reboot"。

---

## 升级到 Pro

HuggingFace Pro（$9/月）提供：
- 自定义域名
- 更好的网络支持
- 不休眠

---

## 下一步

1. 访问你的 Space URL
2. 使用 `admin` 和设置的密码登录
3. 添加 Gmail/Outlook 邮箱账户

如需连接国内邮箱，建议使用 [Fly.io 部署](../fly/README.md)。
