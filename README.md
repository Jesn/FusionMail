# FusionMail

轻量级邮件接收聚合系统 - 统一管理多个邮箱账户，通过自动化机制与其他产品和系统集成。

## 🔒 安全提示

**重要**：本项目包含敏感信息管理功能，请注意：

- ⚠️ 不要在代码中硬编码邮箱地址和密码
- ⚠️ 不要提交包含真实凭证的配置文件
- ⚠️ 使用 `.test-config` 存储测试账号（已在 .gitignore 中）
- ✅ 提交前运行 `./check-commit-security.sh` 检查安全性

详见：[Git 提交安全规范](.kiro/steering/git-commit-security.md)

## 快速开始

### 前置要求

- Docker 和 Docker Compose
- Go 1.21+ (后端开发)
- Node.js 18+ (前端开发)
- lsof（端口检查工具）

### 一键启动（推荐）

```bash
# 1. 克隆项目
git clone <repository-url>
cd fusionmail

# 2. 完整启动（自动启动所有服务）
./start.sh
```

就这么简单！脚本会自动：
- ✅ 检查并安装依赖
- ✅ 启动 PostgreSQL 和 Redis
- ✅ 构建并启动后端服务
- ✅ 安装并启动前端服务
- ✅ 自动处理端口冲突

### 启动选项

```bash
# 显示帮助信息
./start.sh -h

# 开发模式（热重载）
./start.sh -w

# 仅启动后端
./start.sh -b

# 仅启动前端
./start.sh -f

# 清理数据后启动
./start.sh -c

# 调试模式
./start.sh -d

# 组合使用
./start.sh -w -d  # 开发模式 + 调试日志
```

详细说明请查看：[启动指南](docs/startup-guide.md)

### 停止服务

```bash
# 停止前后端服务（保留数据库）
./stop.sh

# 停止所有服务（包括 Docker）
./stop.sh -a

# 停止并清理所有数据
./stop.sh -c
```

### 访问应用

- **前端**: http://localhost:4444
- **后端 API**: http://localhost:3333/api/v1
- **健康检查**: http://localhost:3333/api/v1/health

### 管理员账号

首次启动后，系统会自动创建管理员账号：
- **用户名**: admin
- **密码**: 保存在 `backend/passwd` 文件中
- ⚠️ 首次登录后请修改密码！

## ✨ 核心功能

### 已实现功能

#### 📧 邮件管理
- ✅ 多邮箱账户管理（Gmail、Outlook、QQ、163、iCloud、IMAP/POP3）
- ✅ 后台自动同步（增量同步，可配置频率）
- ✅ 邮件列表查询（分页、筛选、排序）
- ✅ 邮件详情查看（包含附件）
- ✅ 全文搜索（基于 PostgreSQL tsvector）
- ✅ 邮件状态管理（已读、星标、归档、删除）
- ✅ 未读邮件统计
- ✅ 账户邮件统计

#### 🤖 自动化规则
- ✅ 规则引擎（条件匹配 + 动作执行）
- ✅ 支持多种条件（发件人、主题、正文等）
- ✅ 支持多种动作（标记已读、星标、归档、删除）
- ✅ 优先级排序
- ✅ 规则执行统计

#### 🔄 同步管理
- ✅ 手动同步
- ✅ 自动定时同步
- ✅ 增量同步（避免重复）
- ✅ 同步日志记录

### 待实现功能

- [ ] Webhook 集成
- [ ] 邮件标签功能
- [ ] 邮件发送功能
- [ ] 用户认证（JWT）
- [ ] 前端界面
- [ ] 附件下载
- [ ] 邮件会话视图

## 📚 文档

- [快速开始指南](docs/quick-start.md) - 5 分钟快速上手
- [邮件管理 API](docs/email-api.md) - 邮件查询和管理接口
- [规则引擎 API](docs/rule-api.md) - 自动化规则配置
- [开发进度](docs/development-progress.md) - 功能实现状态
- [测试指南](docs/testing-guide.md) - 如何测试功能

## 🚀 快速测试

### 1. 添加邮箱账户

```bash
curl -X POST http://localhost:3333/api/v1/accounts \
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

### 2. 同步邮件

```bash
# 替换为您的账户 UID
export ACCOUNT_UID="acc_xxx"
curl -X POST http://localhost:3333/api/v1/sync/accounts/$ACCOUNT_UID
```

### 3. 查看邮件

```bash
# 获取邮件列表
curl "http://localhost:3333/api/v1/emails?account_uid=$ACCOUNT_UID&page=1&page_size=10"

# 搜索邮件
curl "http://localhost:3333/api/v1/emails/search?q=通知"

# 获取未读邮件数
curl "http://localhost:3333/api/v1/emails/unread-count?account_uid=$ACCOUNT_UID"
```

### 4. 创建自动化规则

```bash
curl -X POST http://localhost:3333/api/v1/rules \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"自动归档通知邮件\",
    \"account_uid\": \"$ACCOUNT_UID\",
    \"conditions\": \"[{\\\"field\\\":\\\"subject\\\",\\\"operator\\\":\\\"contains\\\",\\\"value\\\":\\\"通知\\\"}]\",
    \"actions\": \"[{\\\"type\\\":\\\"archive\\\"},{\\\"type\\\":\\\"mark_read\\\"}]\",
    \"enabled\": true
  }"
```

### 5. 使用测试脚本

```bash
# 自动测试所有邮件 API
ACCOUNT_UID=$ACCOUNT_UID ./scripts/test-email-api.sh
```

## 项目结构

```
fusionmail/
├── backend/                 # Go 后端项目
├── frontend/                # React 前端项目
├── docker-compose.dev.yml   # 开发环境 Docker 配置
├── scripts/                 # 开发脚本
│   ├── dev-start.sh        # 启动开发环境
│   ├── dev-stop.sh         # 停止开发环境
│   └── README.md           # 脚本说明
├── .kiro/                   # Kiro IDE 配置
│   ├── specs/              # 项目规格文档
│   └── steering/           # Kiro 指导文档
└── docs/                    # 项目文档
```

## 核心功能

- ✅ 多邮箱账户管理（Gmail、Outlook、iCloud、QQ、163、IMAP/POP3）
- ✅ 后台自动同步（可配置同步频率）
- ✅ 邮件存储与索引（全文搜索、高级筛选）
- ✅ 邮件查看与本地管理（只读镜像模式）
- ✅ 邮件规则引擎（自动分类、标签、触发动作）
- ✅ Webhook 集成（推送邮件事件到外部系统）
- ✅ RESTful API 接口（供第三方系统调用）
- ✅ 代理支持（HTTP/SOCKS5）

## 技术栈

### 后端
- Go 1.21+
- Gin (Web 框架)
- GORM (ORM)
- PostgreSQL 15 (数据库)
- Redis 7 (缓存 + 队列)

### 前端
- React 19
- TypeScript 5.9
- Vite 7
- Tailwind CSS 4
- shadcn/ui

## 开发文档

- **需求文档**: `.kiro/specs/fusionmail/requirements.md`
- **设计文档**: `.kiro/specs/fusionmail/design.md`
- **任务清单**: `.kiro/specs/fusionmail/tasks.md`
- **开发环境配置**: `.kiro/steering/development-setup.md`
- **API 规范**: `.kiro/steering/api-standards.md`
- **代码规范**: `.kiro/steering/code-conventions.md`

## 常用命令

```bash
# 启动开发环境
./scripts/dev-start.sh

# 停止开发环境
./scripts/dev-stop.sh

# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f

# 进入 PostgreSQL
docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail

# 进入 Redis
docker exec -it fusionmail-redis redis-cli -a fusionmail_redis_password
```

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

[MIT License](LICENSE)

## 联系方式

- 项目主页: [GitHub Repository]
- 问题反馈: [GitHub Issues]
- 文档: [Documentation]

---

**注意**: 这是一个开发中的项目，当前处于 MVP 阶段。
