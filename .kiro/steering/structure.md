# FusionMail 项目结构

## 项目根目录结构

```
fusionmail/
├── backend/                 # Go 后端项目
├── frontend/                # React 前端项目
├── docker-compose.yml       # Docker Compose 配置
├── .gitignore
├── README.md
└── docs/                    # 项目文档
    ├── api/                 # API 文档
    ├── deployment/          # 部署文档
    └── development/         # 开发文档
```

## 后端项目结构

```
backend/
├── cmd/
│   └── server/
│       └── main.go                    # 应用入口
│
├── internal/                          # 内部包（不对外暴露）
│   ├── adapter/                       # 邮箱协议适配层
│   │   ├── adapter.go                 # 适配器接口定义
│   │   ├── factory.go                 # 适配器工厂
│   │   ├── gmail.go                   # Gmail API 适配器
│   │   ├── graph.go                   # Microsoft Graph 适配器
│   │   ├── imap.go                    # IMAP 适配器
│   │   └── pop3.go                    # POP3 适配器
│   │
│   ├── service/                       # 业务逻辑层
│   │   ├── account.go                 # 账户管理服务
│   │   ├── email.go                   # 邮件管理服务
│   │   ├── sync.go                    # 同步引擎服务
│   │   ├── rule.go                    # 规则引擎服务
│   │   ├── webhook.go                 # Webhook 服务
│   │   └── search.go                  # 搜索服务
│   │
│   ├── repository/                    # 数据访问层
│   │   ├── account.go                 # 账户数据仓库
│   │   ├── email.go                   # 邮件数据仓库
│   │   ├── rule.go                    # 规则数据仓库
│   │   ├── webhook.go                 # Webhook 数据仓库
│   │   └── sync_log.go                # 同步日志仓库
│   │
│   ├── handler/                       # HTTP 处理层
│   │   ├── auth.go                    # 认证处理
│   │   ├── account.go                 # 账户 API
│   │   ├── email.go                   # 邮件 API
│   │   ├── rule.go                    # 规则 API
│   │   ├── webhook.go                 # Webhook API
│   │   └── system.go                  # 系统 API
│   │
│   ├── middleware/                    # 中间件
│   │   ├── auth.go                    # 认证中间件
│   │   ├── cors.go                    # CORS 中间件
│   │   ├── logger.go                  # 日志中间件
│   │   ├── ratelimit.go               # 速率限制中间件
│   │   └── recovery.go                # 错误恢复中间件
│   │
│   ├── model/                         # 数据模型
│   │   ├── account.go                 # 账户模型
│   │   ├── email.go                   # 邮件模型
│   │   ├── attachment.go              # 附件模型
│   │   ├── label.go                   # 标签模型
│   │   ├── rule.go                    # 规则模型
│   │   ├── webhook.go                 # Webhook 模型
│   │   ├── user.go                    # 用户模型
│   │   └── api_key.go                 # API Key 模型
│   │
│   └── dto/                           # 数据传输对象
│       ├── request/                   # 请求 DTO
│       │   ├── account.go
│       │   ├── email.go
│       │   └── rule.go
│       └── response/                  # 响应 DTO
│           ├── account.go
│           ├── email.go
│           └── common.go
│
├── pkg/                               # 公共包（可对外暴露）
│   ├── crypto/                        # 加密工具
│   │   ├── aes.go                     # AES 加密
│   │   └── hash.go                    # 哈希工具
│   ├── logger/                        # 日志工具
│   │   └── logger.go
│   ├── storage/                       # 存储工具
│   │   ├── storage.go                 # 存储接口
│   │   ├── local.go                   # 本地存储
│   │   ├── s3.go                      # S3 存储
│   │   └── oss.go                     # OSS 存储
│   ├── queue/                         # 队列工具
│   │   └── redis_queue.go
│   └── event/                         # 事件总线
│       └── bus.go
│
├── config/                            # 配置文件
│   ├── config.go                      # 配置结构
│   └── config.yaml                    # 默认配置
│
├── migrations/                        # 数据库迁移
│   ├── 001_create_users.sql
│   ├── 002_create_accounts.sql
│   └── ...
│
├── scripts/                           # 脚本文件
│   ├── build.sh                       # 构建脚本
│   └── migrate.sh                     # 迁移脚本
│
├── Dockerfile                         # Docker 镜像
├── go.mod                             # Go 模块
├── go.sum
└── .env.example                       # 环境变量示例
```

## 前端项目结构

```
frontend/
├── public/                            # 静态资源
│   ├── favicon.ico
│   └── logo.png
│
├── src/
│   ├── components/                    # 通用组件
│   │   ├── ui/                        # shadcn/ui 基础组件
│   │   │   ├── button.tsx
│   │   │   ├── input.tsx
│   │   │   ├── dialog.tsx
│   │   │   └── ...
│   │   ├── layout/                    # 布局组件
│   │   │   ├── MainLayout.tsx
│   │   │   ├── Header.tsx
│   │   │   └── Sidebar.tsx
│   │   ├── email/                     # 邮件相关组件
│   │   │   ├── EmailList.tsx
│   │   │   ├── EmailItem.tsx
│   │   │   ├── EmailDetail.tsx
│   │   │   └── EmailToolbar.tsx
│   │   ├── account/                   # 账户相关组件
│   │   │   ├── AccountCard.tsx
│   │   │   ├── AccountForm.tsx
│   │   │   └── AccountList.tsx
│   │   └── common/                    # 通用业务组件
│   │       ├── LoadingSpinner.tsx
│   │       ├── ErrorBoundary.tsx
│   │       └── EmptyState.tsx
│   │
│   ├── pages/                         # 页面组件
│   │   ├── Inbox/                     # 收件箱页面
│   │   │   └── index.tsx
│   │   ├── Accounts/                  # 账户管理页面
│   │   │   └── index.tsx
│   │   ├── Rules/                     # 规则管理页面
│   │   │   └── index.tsx
│   │   ├── Webhooks/                  # Webhook 管理页面
│   │   │   └── index.tsx
│   │   ├── Search/                    # 搜索页面
│   │   │   └── index.tsx
│   │   ├── Settings/                  # 设置页面
│   │   │   └── index.tsx
│   │   └── Login/                     # 登录页面
│   │       └── index.tsx
│   │
│   ├── stores/                        # Zustand 状态管理
│   │   ├── authStore.ts               # 认证状态
│   │   ├── emailStore.ts              # 邮件状态
│   │   ├── accountStore.ts            # 账户状态
│   │   ├── ruleStore.ts               # 规则状态
│   │   └── uiStore.ts                 # UI 状态
│   │
│   ├── services/                      # API 服务层
│   │   ├── api.ts                     # Axios 配置
│   │   ├── authService.ts             # 认证服务
│   │   ├── emailService.ts            # 邮件服务
│   │   ├── accountService.ts          # 账户服务
│   │   ├── ruleService.ts             # 规则服务
│   │   └── webhookService.ts          # Webhook 服务
│   │
│   ├── hooks/                         # 自定义 Hooks
│   │   ├── useAuth.ts                 # 认证 Hook
│   │   ├── useEmails.ts               # 邮件 Hook
│   │   ├── useAccounts.ts             # 账户 Hook
│   │   └── useDebounce.ts             # 防抖 Hook
│   │
│   ├── utils/                         # 工具函数
│   │   ├── format.ts                  # 格式化工具
│   │   ├── validation.ts              # 验证工具
│   │   └── constants.ts               # 常量定义
│   │
│   ├── types/                         # TypeScript 类型
│   │   ├── email.ts                   # 邮件类型
│   │   ├── account.ts                 # 账户类型
│   │   ├── rule.ts                    # 规则类型
│   │   └── api.ts                     # API 类型
│   │
│   ├── App.tsx                        # 应用根组件
│   ├── main.tsx                       # 应用入口
│   └── index.css                      # 全局样式
│
├── Dockerfile                         # Docker 镜像
├── vite.config.ts                     # Vite 配置
├── tailwind.config.js                 # Tailwind 配置
├── tsconfig.json                      # TypeScript 配置
├── package.json
└── .env.example                       # 环境变量示例
```

## 命名约定

### 后端命名规范

#### 文件命名
- **Go 文件**：小写 + 下划线，如 `email_service.go`
- **测试文件**：`*_test.go`，如 `email_service_test.go`
- **接口文件**：通常以接口名命名，如 `repository.go`

#### 包命名
- **小写单词**：`package adapter`、`package service`
- **避免复数**：`package model` 而非 `package models`
- **简短有意义**：`pkg/crypto` 而非 `pkg/cryptography`

#### 变量命名
- **驼峰命名**：`emailService`、`accountRepo`
- **常量大写**：`const MaxRetryCount = 3`
- **私有变量**：小写开头 `emailList`
- **公开变量**：大写开头 `EmailList`

#### 函数命名
- **驼峰命名**：`GetEmailList()`、`CreateAccount()`
- **接口方法**：动词开头，如 `Connect()`、`Fetch()`
- **私有函数**：小写开头 `parseEmail()`
- **公开函数**：大写开头 `ParseEmail()`

### 前端命名规范

#### 文件命名
- **组件文件**：大驼峰（PascalCase），如 `EmailList.tsx`
- **工具文件**：小驼峰（camelCase），如 `formatDate.ts`
- **类型文件**：小驼峰，如 `email.ts`
- **样式文件**：小驼峰，如 `emailList.module.css`

#### 组件命名
- **React 组件**：大驼峰，如 `EmailList`、`AccountCard`
- **组件文件夹**：与组件同名，如 `EmailList/index.tsx`

#### 变量命名
- **普通变量**：小驼峰，如 `emailList`、`accountData`
- **常量**：大写 + 下划线，如 `API_BASE_URL`
- **布尔值**：`is/has/should` 前缀，如 `isLoading`、`hasError`

#### 函数命名
- **普通函数**：小驼峰，如 `fetchEmails()`、`handleClick()`
- **事件处理**：`handle` 前缀，如 `handleSubmit()`
- **Hook 函数**：`use` 前缀，如 `useAuth()`、`useEmails()`

## 导入顺序

### 后端导入顺序

```go
package service

import (
    // 1. 标准库
    "context"
    "fmt"
    "time"
    
    // 2. 第三方库
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    
    // 3. 项目内部包
    "fusionmail/internal/model"
    "fusionmail/internal/repository"
    "fusionmail/pkg/logger"
)
```

### 前端导入顺序

```typescript
// 1. React 相关
import React, { useState, useEffect } from 'react';

// 2. 第三方库
import { useNavigate } from 'react-router-dom';
import axios from 'axios';

// 3. UI 组件库
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

// 4. 项目组件
import EmailList from '@/components/email/EmailList';
import AccountCard from '@/components/account/AccountCard';

// 5. Hooks 和工具
import { useAuth } from '@/hooks/useAuth';
import { formatDate } from '@/utils/format';

// 6. 类型定义
import type { Email, Account } from '@/types';

// 7. 样式文件
import './index.css';
```

## 架构决策

### 后端架构原则

1. **分层清晰**：Handler → Service → Repository → Model
2. **依赖注入**：通过构造函数注入依赖
3. **接口抽象**：定义接口，面向接口编程
4. **错误处理**：统一错误处理和响应格式
5. **上下文传递**：使用 Context 传递请求上下文

### 前端架构原则

1. **组件化**：拆分可复用组件
2. **状态管理**：使用 Zustand 管理全局状态
3. **类型安全**：充分利用 TypeScript 类型系统
4. **性能优化**：虚拟滚动、懒加载、代码分割
5. **错误边界**：使用 ErrorBoundary 捕获错误

## 代码组织原则

### 单一职责原则
- 每个文件只负责一个功能模块
- 每个函数只做一件事
- 每个组件只负责一个 UI 单元

### 模块化原则
- 相关功能放在同一目录
- 通过 index 文件统一导出
- 避免循环依赖

### 可测试性原则
- 业务逻辑与框架解耦
- 依赖注入便于 Mock
- 纯函数优先

---

**注意**：保持项目结构的一致性和清晰性，有助于团队协作和代码维护。在添加新功能时，请遵循现有的目录结构和命名约定。
