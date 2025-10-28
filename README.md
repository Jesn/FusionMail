# FusionMail

轻量级邮件接收聚合系统 - 统一管理多个邮箱账户，通过自动化机制与其他产品和系统集成。

## 快速开始

### 前置要求

- Docker 和 Docker Compose
- Go 1.21+ (后端开发)
- Node.js 18+ (前端开发)

### 启动开发环境

```bash
# 1. 克隆项目
git clone <repository-url>
cd fusionmail

# 2. 启动基础设施（PostgreSQL + Redis）
./scripts/dev-start.sh

# 3. 启动后端（新终端）
cd backend
go mod download
go run cmd/server/main.go

# 4. 启动前端（新终端）
cd frontend
npm install
npm run dev
```

### 访问应用

- **前端**: http://localhost:3000
- **后端 API**: http://localhost:8080/api/v1

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
