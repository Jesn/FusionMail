# FusionMail

FusionMail 是一个**自托管的多邮箱聚合收件箱**。它把多个 IMAP、POP3、OAuth 和可选 WebAPI 邮箱账户同步到单一 Web 控制台，主要提供邮件阅读、搜索和本地状态管理；规则、Webhook 与发信能力用于轻量自动化和辅助管理。

当前产品北极星是：

> 用户登录 Web 控制台，接入多个邮箱账户，在统一收件箱中阅读、搜索和管理邮件状态。

FusionMail 不是 SaaS 多租户平台，也不以临时邮箱运营台或完整双向邮箱客户端为产品主叙事。

## 当前能力

### 统一收件箱主路径

- 多邮箱账户：IMAP、POP3、Gmail API、Microsoft Graph，以及可选 WebAPI 账户类型
- OAuth2、密码认证和 Cookie/JWT 会话认证，支持 2FA
- 后台定时同步、手动同步和增量同步
- 收件箱、已发送、垃圾邮件、回收站等邮件视图
- 邮件详情、安全 HTML 渲染和附件入口
- 关键词搜索、分页和账户筛选
- 已读、星标、归档、删除等本地状态管理

### 可选高级能力

- 规则引擎：按发件人、主题、正文等条件执行标记、归档、删除等动作
- 出站 Webhook 与入站 Webhook receiver
- 发信能力
- 垃圾邮件检测与规则
- API Key、Provider、OAuth2 客户端和 Swagger API 文档
- JSON 日志、Prometheus metrics、readiness 和 OpenTelemetry 追踪

WebAPI 账户保留在产品中，作为可选的账户接入方式，不单独形成第二套产品或运营台导航。

## 明确非目标与未闭环能力

以下内容当前不作为统一收件箱 MVP 的交付承诺：

- 多租户、邮箱账户用户归属和完整的数据隔离
- 邮件标签的完整 CRUD、管理界面和规则联动
- 会话/Conversation 视图的完整用户体验
- S3/OSS 等对象存储附件方案
- 设置导入/导出功能（当前接口仍返回 501）
- 面向所有协议的完整双向邮箱客户端语义
- WebAPI 子邮箱批量运营、计费或独立产品化能力

这些能力可以保留已有模型或代码，但不会被描述为当前 MVP 已提供的完整功能。

## 快速开始

### 前置要求

- Docker 和 Docker Compose
- Go `1.25.0` 或更高版本（后端开发）
- Node.js `20.19.0` 或更高版本（前端开发）
- `lsof`

### 启动开发环境

```bash
./start.sh
```

启动脚本会检查基础设施、启动 PostgreSQL 和 Redis，并启动后端 API 与前端开发服务。常用选项：

```bash
./start.sh -h  # 查看帮助
./start.sh -w  # 开发模式
./start.sh -b  # 仅启动后端
./start.sh -f  # 仅启动前端
./start.sh -d  # 调试日志
```

默认访问地址：

- 前端：`http://localhost:4444`
- 后端 API：`http://localhost:3333/api/v1`
- 健康检查：`http://localhost:3333/api/v1/health`
- 就绪检查：`http://localhost:3333/api/v1/ready`

首次启动后的管理员账号和密码以当前环境配置或 seed 输出为准。登录后应立即修改默认凭据；不要将真实邮箱密码、JWT secret 或加密密钥提交到仓库。

### 手动开发命令

```bash
cd frontend && npm install
cd frontend && npm run dev
cd frontend && npm test
cd frontend && npm run build

cd backend && go test ./...
cd backend && go build ./...
```

## 文档

- [文档目录](docs/README.md)
- [快速开始](docs/quick-start.md)
- [测试指南](docs/testing-guide.md)
- [邮件 API](docs/email-api.md)
- [规则 API](docs/rule-api.md)
- [环境变量](docs/environment-variables.md)
- [Swagger 指南](docs/swagger-guide.md)
- [生产部署检查清单](docs/production-deployment-checklist.md)
- [Fly.io 部署与 secrets 说明](AGENTS.md)

部署前请先执行后端构建和测试；如果新增数据库 migration，先在目标环境执行 migration，再部署应用。生产发布命令和健康检查步骤以 `AGENTS.md` 为准。

## API 示例

以下请求仅展示接口形状，实际调用需要先完成登录并携带会话 Cookie 或 API Key。请使用测试账户，不要把真实凭据写入命令历史或文档。

```bash
# 获取邮件列表
curl "http://localhost:3333/api/v1/emails?page=1&page_size=10"

# 搜索邮件
curl "http://localhost:3333/api/v1/emails/search?q=通知"

# 手动同步账户
curl -X POST "http://localhost:3333/api/v1/sync/accounts/<account_uid>"
```

## 项目结构

```text
fusionmail/
├── backend/       # Go API、同步服务、协议适配器和数据访问层
├── frontend/      # React + TypeScript Web 控制台
├── docs/          # 当前开发、API、部署和运维文档
├── .trellis/      # 任务、规范和工作流上下文
├── start.sh       # 本地开发环境启动脚本
└── AGENTS.md      # Fly.io 部署和生产运维说明
```

## 技术栈

- 后端：Go 1.25、Gin、GORM、PostgreSQL、Redis
- 前端：React 19、TypeScript 5.9、Vite 7、Tailwind CSS 4、shadcn/ui
- 协议：IMAP、POP3、Gmail API、Microsoft Graph、WebAPI
- 运维：Prometheus、JSON logging、OpenTelemetry、Fly.io

## 生产边界

生产构建不注册 OAuth2 测试页和 SSE 调试页；开发构建仍保留这些页面用于本地排查。旧设置路径 `/settings/legacy` 只在开发环境渲染旧页面，生产访问会重定向到 `/settings`。

生产环境请关闭 Swagger 等调试能力，并通过 secrets 管理 `JWT_SECRET`、`ENCRYPTION_KEY` 等敏感配置。具体部署、migration、secret 轮换和发布后检查步骤见 [AGENTS.md](AGENTS.md)。

## 安全

- 不在代码、README、测试配置以外的文件中写入真实邮箱凭据
- 不提交 `.env`、`.test-config` 或包含密钥的日志
- 邮件 HTML 必须经过现有 sanitize 和 Shadow DOM 渲染链
- 提交前检查敏感信息和部署配置

## 项目状态

FusionMail 当前处于**统一收件箱 MVP 收口阶段**：主路径能力已具备，正在同步收敛生产导航、文档叙事和验收边界。未闭环能力会明确标注，不以历史 TODO 代替当前代码事实。