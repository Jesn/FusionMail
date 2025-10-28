# FusionMail 前端

FusionMail 邮件聚合系统的 React 前端应用。

## 技术栈

- **React 19** - 前端框架
- **TypeScript 5.9** - 类型系统
- **Vite 7** - 构建工具
- **Tailwind CSS 4** - CSS 框架
- **shadcn/ui** - UI 组件库
- **Zustand** - 状态管理
- **React Router** - 路由管理
- **Axios** - HTTP 客户端
- **@tanstack/react-virtual** - 虚拟滚动
- **date-fns** - 日期处理
- **Lucide React** - 图标库

## 快速开始

### 前置要求

- Node.js 20.19.0+
- npm 或 yarn

### 安装依赖

```bash
npm install
```

### 配置环境变量

复制环境变量示例文件并修改配置：

```bash
cp .env.example .env
# 编辑 .env 文件，修改 API 地址等配置
```

### 启动开发服务器

```bash
npm run dev
```

应用将在 `http://localhost:3000` 启动。

## 项目结构

```
frontend/
├── public/              # 静态资源
├── src/
│   ├── components/     # 通用组件
│   │   ├── ui/        # shadcn/ui 基础组件
│   │   ├── layout/    # 布局组件
│   │   ├── email/     # 邮件相关组件
│   │   └── account/   # 账户相关组件
│   ├── pages/         # 页面组件
│   │   ├── Inbox/     # 收件箱页面
│   │   ├── Accounts/  # 账户管理页面
│   │   ├── Rules/     # 规则管理页面
│   │   └── Settings/  # 设置页面
│   ├── stores/        # Zustand 状态管理
│   ├── services/      # API 服务层
│   ├── hooks/         # 自定义 Hooks
│   ├── utils/         # 工具函数
│   ├── types/         # TypeScript 类型
│   ├── App.tsx        # 应用根组件
│   └── main.tsx       # 应用入口
├── .env               # 环境变量
├── vite.config.ts     # Vite 配置
├── tailwind.config.ts # Tailwind 配置
└── tsconfig.json      # TypeScript 配置
```

## 开发命令

### 启动开发服务器

```bash
npm run dev
```

### 构建生产版本

```bash
npm run build
```

### 预览生产构建

```bash
npm run preview
```

### 代码检查

```bash
# 运行 ESLint
npm run lint

# 严格模式（0 警告）
npm run lint:strict

# 自动修复
npm run lint:fix
```

### 代码格式化

```bash
# 格式化代码
npm run format

# 检查格式
npm run format:check
```

## 环境变量

详见 `.env.example` 文件。主要配置项：

- `VITE_API_BASE_URL` - API 基础 URL（默认：http://localhost:8080/api/v1）
- `VITE_WS_URL` - WebSocket URL（默认：ws://localhost:8080/ws）
- `VITE_APP_NAME` - 应用名称
- `VITE_APP_VERSION` - 应用版本

## API 代理

开发环境下，Vite 会自动代理 API 请求到后端服务器：

- `/api/*` → `http://localhost:8080/api/*`
- `/ws` → `ws://localhost:8080/ws`

配置详见 `vite.config.ts`。

## 代码规范

### 命名规范

- **组件**：大驼峰（PascalCase）- `EmailList.tsx`
- **文件**：组件文件大驼峰，工具文件小驼峰
- **变量**：小驼峰（camelCase）- `emailList`
- **常量**：大写 + 下划线 - `API_BASE_URL`
- **布尔值**：`is/has/should` 前缀 - `isLoading`

### 导入顺序

1. React 相关
2. 第三方库
3. UI 组件库
4. 项目组件
5. Hooks 和工具
6. 类型定义
7. 样式文件

### 组件规范

- 使用函数组件和 Hooks
- 优先使用 TypeScript 类型定义
- 组件拆分要合理，保持单一职责
- 使用 ErrorBoundary 捕获错误

## 性能优化

- 使用 `@tanstack/react-virtual` 实现虚拟滚动
- 使用 `React.lazy` 懒加载页面
- 使用 `React.memo` 优化组件渲染
- 合理使用 `useMemo` 和 `useCallback`

## 故障排查

### 依赖安装失败

```bash
# 清理缓存
rm -rf node_modules package-lock.json
npm cache clean --force

# 重新安装
npm install
```

### 端口被占用

修改 `vite.config.ts` 中的 `server.port` 配置。

### API 请求失败

1. 检查后端服务是否运行
2. 检查 `.env` 中的 API 地址配置
3. 检查浏览器控制台的网络请求

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License
