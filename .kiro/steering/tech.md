# FusionMail 技术栈

## 后端技术栈

### 核心技术

| 技术 | 版本 | 用途 | 选型理由 |
|-----|------|------|---------|
| **Go** | 1.21+ | 后端开发语言 | 高性能、并发友好、内存安全、编译型语言 |
| **Gin** | 1.9+ | Web 框架 | 轻量级、高性能、中间件丰富 |
| **GORM** | 1.25+ | ORM 框架 | 功能完善、支持多数据库、迁移方便 |
| **PostgreSQL** | 15+ | 主数据库 | 开源、功能强大、支持 JSON、全文搜索 |
| **Redis** | 7+ | 缓存 + 队列 | 高性能、支持多种数据结构、持久化 |
| **JWT** | - | 认证方案 | 无状态、跨域友好、标准化 |

### 邮箱协议库

| 库 | 用途 | 说明 |
|---|------|------|
| **google.golang.org/api/gmail/v1** | Gmail API 客户端 | 官方 Go SDK，支持 OAuth2 |
| **github.com/Azure/azure-sdk-for-go** | Microsoft Graph API | 官方 Go SDK，访问 Outlook/Hotmail |
| **github.com/emersion/go-imap** | IMAP 协议 | 纯 Go 实现，功能完整 |
| **github.com/knadh/go-pop3** | POP3 协议 | 轻量级 POP3 客户端 |
| **golang.org/x/oauth2** | OAuth2 认证 | 官方 OAuth2 库 |

### 工具库

| 库 | 用途 |
|---|------|
| **github.com/spf13/viper** | 配置管理 |
| **go.uber.org/zap** | 结构化日志 |
| **github.com/robfig/cron/v3** | 定时任务 |
| **github.com/go-redis/redis/v9** | Redis 客户端 |
| **golang.org/x/crypto** | 加密工具（AES-256、bcrypt） |
| **github.com/aws/aws-sdk-go-v2** | AWS S3 客户端（可选） |
| **github.com/aliyun/aliyun-oss-go-sdk** | 阿里云 OSS 客户端（可选） |

## 前端技术栈

### 核心技术

| 技术 | 版本 | 用途 | 选型理由 |
|-----|------|------|---------|
| **React** | 19.2.0 | 前端框架 | 组件化、生态丰富、性能优秀 |
| **TypeScript** | 5.9.3 | 类型系统 | 类型安全、IDE 友好、减少错误 |
| **Vite** | 7.1.7 | 构建工具 | 快速启动、HMR、现代化 |
| **Tailwind CSS** | 4.1.14 | CSS 框架 | 实用优先、响应式、可定制 |
| **shadcn/ui** | latest | UI 组件库 | 高质量、可定制、无依赖锁定 |

### 前端库

| 库 | 用途 |
|---|------|
| **React Router** | 路由管理 |
| **Axios** | HTTP 客户端 |
| **Zustand** | 状态管理 |
| **@tanstack/react-virtual** | 虚拟滚动 |
| **date-fns** | 日期处理 |
| **react-hot-toast** | 通知提示 |
| **lucide-react** | 图标库 |

## 基础设施

### 数据库

**PostgreSQL 15+**
- **用途**：主数据存储
- **特性**：
  - 支持 JSON 字段（存储邮件元数据）
  - 支持全文搜索（tsvector）
  - 事务支持完善
  - 性能优秀

**关键配置**：
```sql
-- 全文搜索索引
CREATE INDEX idx_fulltext_search ON emails 
USING gin(to_tsvector('english', subject || ' ' || text_body));

-- 复合唯一索引
CREATE UNIQUE INDEX idx_provider_account ON emails(provider_id, account_uid);
```

### 缓存与队列

**Redis 7+**
- **用途**：
  - 缓存（邮件列表、未读数、用户会话）
  - 任务队列（同步任务、Webhook 任务）
  - 分布式锁（防止重复同步）
  - 事件总线（Pub/Sub）
  - 速率限制（API 限流）

**关键数据结构**：
```
# 任务队列
LIST: sync:queue:pending
LIST: sync:queue:processing

# 分布式锁
STRING: lock:sync:{account_uid}

# 缓存
STRING: cache:email:list:{params_hash}
STRING: cache:account:{uid}

# 速率限制
STRING: ratelimit:api:{api_key}:{minute}

# 事件总线
PUBSUB: events:email:received
```

### 存储

**本地存储（默认）**
- **路径**：`/data/attachments/`
- **结构**：`{account_uid}/{email_id}/{filename}`

**对象存储（可选）**
- **AWS S3**：支持预签名 URL
- **阿里云 OSS**：支持预签名 URL
- **配置**：通过环境变量切换

## 部署架构

### Docker 容器化

**开发环境快速启动**：
```bash
# 启动基础设施（PostgreSQL + Redis）
./scripts/dev-start.sh

# 停止基础设施
./scripts/dev-stop.sh
```

**配置文件位置**：
- 开发环境：`docker-compose.dev.yml`
- 生产环境：`docker-compose.yml`（待创建）

**服务组成**：
```yaml
services:
  postgres:
    image: postgres:15-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
  
  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
  
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
  
  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    depends_on:
      - backend
```

**详细配置说明**：参见 `.kiro/steering/development-setup.md`

### 环境变量

**后端配置**：
```bash
# 数据库
DATABASE_URL=postgresql://user:pass@postgres:5432/fusionmail

# Redis
REDIS_URL=redis://redis:6379/0

# 服务器
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# 加密
ENCRYPTION_KEY=your-32-byte-encryption-key

# 存储
STORAGE_TYPE=local  # local/s3/oss
STORAGE_PATH=/data/attachments

# S3 配置（可选）
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx
S3_BUCKET=fusionmail-attachments

# 同步配置
SYNC_WORKER_COUNT=5
SYNC_DEFAULT_INTERVAL=5  # 分钟
```

**前端配置**：
```bash
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws
```

## 架构模式

### 后端架构

**分层架构**：
```
cmd/
  └── server/
      └── main.go           # 入口文件

internal/
  ├── adapter/              # 邮箱协议适配层
  │   ├── gmail.go
  │   ├── graph.go
  │   ├── imap.go
  │   └── pop3.go
  ├── service/              # 业务逻辑层
  │   ├── account.go
  │   ├── email.go
  │   ├── sync.go
  │   ├── rule.go
  │   └── webhook.go
  ├── repository/           # 数据访问层
  │   ├── account.go
  │   ├── email.go
  │   └── rule.go
  ├── handler/              # API 处理层
  │   ├── account.go
  │   ├── email.go
  │   └── webhook.go
  └── model/                # 数据模型
      ├── account.go
      ├── email.go
      └── rule.go

pkg/
  ├── crypto/               # 加密工具
  ├── logger/               # 日志工具
  └── storage/              # 存储工具
```

**设计模式**：
- **适配器模式**：统一邮箱协议接口
- **工厂模式**：创建邮箱适配器实例
- **Repository 模式**：封装数据访问
- **依赖注入**：服务间解耦

### 前端架构

**目录结构**：
```
src/
  ├── components/           # 通用组件
  │   ├── ui/              # shadcn/ui 组件
  │   ├── EmailList/
  │   ├── EmailDetail/
  │   └── AccountCard/
  ├── pages/               # 页面组件
  │   ├── Inbox/
  │   ├── Accounts/
  │   ├── Rules/
  │   └── Settings/
  ├── stores/              # Zustand 状态管理
  │   ├── authStore.ts
  │   ├── emailStore.ts
  │   └── accountStore.ts
  ├── services/            # API 服务
  │   ├── api.ts
  │   ├── emailService.ts
  │   └── accountService.ts
  ├── hooks/               # 自定义 Hooks
  ├── utils/               # 工具函数
  └── types/               # TypeScript 类型
```

## 性能优化策略

### 后端优化

1. **数据库优化**
   - 合理使用索引
   - 查询结果分页
   - 使用 GORM 预加载避免 N+1 查询
   - PostgreSQL 连接池配置

2. **缓存策略**
   - 邮件列表缓存（5 分钟）
   - 账户信息缓存（10 分钟）
   - 未读数缓存（1 分钟）

3. **并发控制**
   - Worker Pool 限制并发数（5 个）
   - 使用 Context 控制超时
   - 分布式锁防止重复同步

### 前端优化

1. **虚拟滚动**
   - 使用 @tanstack/react-virtual
   - 只渲染可见区域的邮件

2. **代码分割**
   - React.lazy 懒加载页面
   - 动态导入大型组件

3. **资源优化**
   - 图片懒加载
   - 压缩打包体积
   - 使用 CDN 加速

## 安全措施

### 数据加密

- **传输加密**：HTTPS/TLS
- **存储加密**：AES-256 加密敏感数据
- **密码哈希**：bcrypt（cost=10）

### 认证授权

- **JWT Token**：有效期 24 小时
- **API Key**：SHA-256 哈希存储
- **会话管理**：30 分钟无操作自动登出

### 安全防护

- **速率限制**：API 每分钟 100 次
- **输入验证**：严格验证所有输入
- **SQL 注入防护**：使用 GORM 参数化查询
- **XSS 防护**：前端输出转义

## 监控与日志

### 日志系统

- **库**：go.uber.org/zap
- **级别**：DEBUG、INFO、WARN、ERROR
- **格式**：JSON 结构化日志
- **轮转**：按天轮转，保留 30 天

### 监控指标

- **系统指标**：CPU、内存、磁盘
- **业务指标**：同步次数、邮件数、API 调用数
- **性能指标**：响应时间、错误率

---

**注意**：所有技术选型都应该遵循"轻量化、高性能、易维护"的原则。在引入新技术或库时，请评估其对系统资源的影响。
