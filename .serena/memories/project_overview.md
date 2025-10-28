# FusionMail 项目概述

## 项目简介
FusionMail 是一款轻量级邮件接收聚合系统，专注于从多个邮箱账户收集邮件，并通过自动化机制与其他产品和系统集成。

## 核心功能
- 多邮箱账户管理（Gmail、Outlook、iCloud、QQ、163、IMAP/POP3）
- 后台自动同步（可配置同步频率）
- 邮件存储与索引（全文搜索、高级筛选）
- 邮件查看与本地管理（只读镜像模式）
- 邮件规则引擎（自动分类、标签、触发动作）
- Webhook 集成（推送邮件事件到外部系统）
- RESTful API 接口（供第三方系统调用）

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

## 项目结构
```
fusionmail/
├── backend/                 # Go 后端项目
├── frontend/                # React 前端项目
├── docker-compose.dev.yml   # 开发环境 Docker 配置
├── scripts/                 # 开发脚本
└── .kiro/                   # Kiro IDE 配置
    ├── specs/              # 项目规格文档
    └── steering/           # Kiro 指导文档
```

## 开发状态
当前处于 MVP 阶段，正在实现核心功能。
