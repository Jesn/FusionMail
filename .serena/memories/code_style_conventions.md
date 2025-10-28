# FusionMail 代码风格与规范

## Go 后端规范

### 命名规范
- **包命名**：小写单词，避免复数（`package adapter`）
- **文件命名**：小写 + 下划线（`email_service.go`）
- **变量命名**：驼峰命名（`emailService`、`accountRepo`）
- **常量命名**：大写驼峰（`const MaxRetryCount = 3`）
- **函数命名**：大写开头为公开，小写开头为私有（`GetEmailList()`、`parseEmail()`）
- **接口命名**：避免 I 前缀和 Interface 后缀

### 代码组织
- **分层架构**：Handler → Service → Repository → Model
- **依赖注入**：通过构造函数注入依赖
- **接口抽象**：定义接口，面向接口编程
- **错误处理**：使用 `fmt.Errorf` 包装错误，使用 `errors.Is` 检查错误
- **Context 传递**：所有服务方法第一个参数为 `context.Context`

### 导入顺序
1. 标准库
2. 第三方库
3. 项目内部包

### 注释规范
- 包注释：描述包的用途和功能
- 函数注释：说明参数、返回值和功能
- 结构体注释：描述结构体的用途
- 使用中文注释

## TypeScript/React 前端规范

### 命名规范
- **组件命名**：大驼峰（PascalCase）（`EmailList`、`AccountCard`）
- **文件命名**：组件文件大驼峰，工具文件小驼峰
- **变量命名**：小驼峰（camelCase）（`emailList`、`accountData`）
- **常量命名**：大写 + 下划线（`API_BASE_URL`）
- **布尔值**：`is/has/should` 前缀（`isLoading`、`hasError`）
- **函数命名**：小驼峰，事件处理用 `handle` 前缀，Hook 用 `use` 前缀

### 代码组织
- **组件化**：拆分可复用组件
- **状态管理**：使用 Zustand 管理全局状态
- **类型安全**：充分利用 TypeScript 类型系统
- **错误边界**：使用 ErrorBoundary 捕获错误

### 导入顺序
1. React 相关
2. 第三方库
3. UI 组件库
4. 项目组件
5. Hooks 和工具
6. 类型定义
7. 样式文件

## 通用规范

### 代码质量
- 单个函数不超过 50 行
- 单个文件不超过 500 行
- 圈复杂度不超过 10
- 遵循 DRY 原则
- 避免魔法数字，使用常量

### Git 提交规范
遵循 Conventional Commits：
- `feat`: 新功能
- `fix`: Bug 修复
- `refactor`: 重构
- `docs`: 文档更新
- `test`: 测试相关
- `chore`: 构建/工具相关

格式：`<type>[scope]: <description>`

示例：`feat(email): 添加邮件搜索功能`
