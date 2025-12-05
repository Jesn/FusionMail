---
title: FusionMail
emoji: 📧
colorFrom: blue
colorTo: purple
sdk: docker
pinned: false
license: mit
app_port: 7860
---

# FusionMail

轻量级邮件接收聚合系统，支持 Gmail、Outlook、QQ、163 等主流邮箱。

## 功能特性

- 📬 多邮箱账户聚合管理
- 🔄 后台自动同步
- 🔍 全文搜索
- 🏷️ 智能标签和规则
- 🔗 Webhook 集成
- 🔒 本地部署，数据安全

## 配置说明

本应用需要配置以下 Secrets（在 Space Settings 中设置）：

### 必需配置

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `DB_HOST` | PostgreSQL 主机地址 | `aws-0-xxx.pooler.supabase.com` |
| `DB_USER` | 数据库用户 | `postgres.your-project-ref` |
| `DB_PASSWORD` | 数据库密码 | `your-db-password` |
| `JWT_SECRET` | JWT 签名密钥（至少32字符） | `your-jwt-secret-key-32-chars` |
| `ENCRYPTION_KEY` | 数据加密密钥（必须32字节） | `your-32-byte-encryption-key` |

> ⚠️ **重要**：Supabase Session Pooler 用户名格式必须是 `postgres.{project-ref}`，不是简单的 `postgres`

### 可选配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_NAME` | 数据库名称 | `postgres` |
| `DB_SSLMODE` | SSL 模式 | `require` |
| `REDIS_HOST` | Redis 主机（可选） | - |
| `REDIS_PORT` | Redis 端口 | `6379` |
| `REDIS_PASSWORD` | Redis 密码 | - |
| `REDIS_TLS` | Redis TLS（Upstash 需要） | `true` |
| `ADMIN_PASSWORD` | 管理员初始密码 | 自动生成 |

## 推荐的免费数据库服务

### PostgreSQL
- [Neon](https://neon.tech) - 免费额度充足，推荐
- [Supabase](https://supabase.com) - 免费 500MB

### Redis（可选）
- [Upstash](https://upstash.com) - 免费 10,000 请求/天

## 首次使用

1. 配置好 Secrets 后，Space 会自动构建部署
2. 访问应用，使用管理员账号登录
3. 默认用户名：`admin`
4. 默认密码：查看日志或设置 `ADMIN_PASSWORD`

## 自定义域名

如需绑定自定义域名（如 `fusionmail.100420.xyz`）：

1. 在 Space Settings → Custom domain 添加域名
2. 在 DNS 服务商添加 CNAME 记录：
   - 主机记录：`fusionmail`
   - 记录值：`jesn-fusionmail.hf.space`
3. 等待 SSL 证书自动签发（通常几分钟）

## 技术栈

- 后端：Go + Gin + GORM
- 前端：React + TypeScript + Vite + Tailwind CSS
- 数据库：PostgreSQL
- 缓存：Redis（可选）

## 开源地址

[GitHub Repository](https://github.com/your-repo/fusionmail)
