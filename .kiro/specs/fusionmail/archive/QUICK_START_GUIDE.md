# FusionMail 认证优化 - 快速开始指南

## 🚀 立即开始

### 1. 启动开发服务器

```bash
# 启动后端
cd backend
go run cmd/server/main.go

# 启动前端（新终端）
cd frontend
npm run dev
```

### 2. 访问应用

打开浏览器访问：http://localhost:4444

### 3. 测试认证功能

#### 方式 1：手动测试

1. **登录**
   - 访问登录页面
   - 输入密码：`admin123`（或后端日志中显示的密码）
   - 点击登录

2. **查看状态**
   - 打开浏览器控制台（F12）
   - 查看 Application -> Local Storage
   - 应该只看到 `fusionmail-auth` 一项

3. **测试自动刷新**
   - 保持登录状态
   - 观察控制台日志
   - 每 5 分钟会看到检查日志

4. **退出登录**
   - 点击退出按钮
   - 验证 localStorage 已清空
   - 验证被重定向到登录页

#### 方式 2：使用测试工具

打开浏览器控制台（F12），输入：

```javascript
// 查看所有可用命令
authTest.help()

// 运行完整测试套件
authTest.runAllTests('admin123')

// 或者单独测试各个功能
authTest.testLogin('admin123')
authTest.testIsAuthenticated()
authTest.testTokenValidity()
authTest.showCurrentState()
```

## 🧪 测试场景

### 场景 1：正常登录流程

```javascript
// 1. 测试登录
authTest.testLogin('admin123')

// 2. 查看状态
authTest.showCurrentState()

// 3. 测试认证检查
authTest.testIsAuthenticated()

// 4. 测试登出
authTest.testLogout()
```

### 场景 2：Token 自动刷新

```javascript
// 1. 登录
authTest.testLogin('admin123')

// 2. 模拟 Token 即将过期（9 分钟后）
authTest.simulateTokenExpiringSoon()

// 3. 启动自动刷新服务
authTest.testAutoRefreshService()

// 4. 等待 1-2 分钟，观察控制台日志
// 应该看到：[TokenRefresh] Token expiring soon, refreshing...

// 5. 验证 Token 已更新
authTest.showCurrentState()
```

### 场景 3：Token 过期处理

```javascript
// 1. 登录
authTest.testLogin('admin123')

// 2. 模拟 Token 已过期
authTest.simulateTokenExpired()

// 3. 测试认证检查
authTest.testIsAuthenticated()
// 应该返回 false 并自动清理数据

// 4. 验证数据已清理
authTest.showCurrentState()
```

### 场景 4：手动刷新 Token

```javascript
// 1. 登录
authTest.testLogin('admin123')

// 2. 查看当前 Token
authTest.showCurrentState()

// 3. 手动刷新
authTest.testTokenRefresh()

// 4. 验证 Token 已更新
authTest.showCurrentState()
```

## 📋 验收检查清单

### 功能测试

- [ ] 登录功能正常
  - [ ] 输入正确密码可以登录
  - [ ] 输入错误密码显示错误提示
  - [ ] 登录后跳转到收件箱页面
  - [ ] localStorage 中只有 `fusionmail-auth`

- [ ] 认证状态检查正常
  - [ ] 已登录用户可以访问受保护页面
  - [ ] 未登录用户被重定向到登录页
  - [ ] Token 过期后自动清理并重定向

- [ ] Token 自动刷新正常
  - [ ] 每 5 分钟检查一次
  - [ ] Token 过期前 10 分钟自动刷新
  - [ ] 刷新成功后 Token 更新
  - [ ] 刷新失败后自动登出

- [ ] 退出登录功能正常
  - [ ] 点击退出按钮可以登出
  - [ ] localStorage 完全清空
  - [ ] 被重定向到登录页
  - [ ] 无法访问受保护页面

- [ ] 错误处理正常
  - [ ] 401 错误自动清理并重定向
  - [ ] 403 错误显示权限不足提示
  - [ ] 404 错误显示资源不存在提示
  - [ ] 500 错误显示服务器错误提示
  - [ ] 网络错误显示网络连接失败提示

### 代码质量

- [x] 无 TypeScript 错误
- [x] 无 ESLint 警告
- [x] 代码注释清晰
- [x] 类型定义完整

### 性能测试

- [ ] 登录响应时间 < 1 秒
- [ ] Token 刷新响应时间 < 1 秒
- [ ] 页面加载时间 < 2 秒
- [ ] 自动刷新不影响用户操作

## 🐛 常见问题

### Q1: 登录后立即被登出

**原因**：Token 可能已过期或格式不正确

**解决**：
1. 检查后端返回的 `expiresAt` 格式是否正确
2. 检查系统时间是否正确
3. 在控制台运行 `authTest.showCurrentState()` 查看详细信息

### Q2: Token 不会自动刷新

**原因**：自动刷新服务可能未启动

**解决**：
1. 检查控制台是否有 `[TokenRefresh] Service started` 日志
2. 检查 `App.tsx` 中的 useEffect 是否正常执行
3. 手动启动：`authTest.testAutoRefreshService()`

### Q3: 401 错误后没有重定向

**原因**：拦截器可能未正确配置

**解决**：
1. 检查 `api.ts` 中的响应拦截器
2. 检查浏览器控制台是否有错误
3. 确认后端返回的是 401 状态码

### Q4: localStorage 中有多个认证相关的项

**原因**：可能是旧数据残留

**解决**：
1. 运行 `authTest.testClearData()` 清理所有数据
2. 重新登录
3. 检查是否只有 `fusionmail-auth` 一项

## 📊 性能监控

### 查看 Token 刷新日志

```javascript
// 启动自动刷新并观察日志
authTest.testAutoRefreshService()

// 预期日志：
// [TokenRefresh] Service started
// [TokenRefresh] Time until expiry: 15 minutes
// [TokenRefresh] Token expiring soon, refreshing...
// [TokenRefresh] Sending refresh request...
// [TokenRefresh] Token refreshed successfully
```

### 查看认证状态

```javascript
// 查看完整状态
authTest.showCurrentState()

// 预期输出：
// {
//   user: { id: 1, email: 'admin', name: 'Admin' },
//   token: 'eyJhbGciOiJIUzI1NiIs...',
//   expiresAt: '2025-10-31T10:00:00Z',
//   isAuthenticated: true,
//   isTokenValid: true
// }
```

## 🎯 下一步

完成测试后，可以：

1. **添加单元测试**
   ```bash
   cd frontend
   npm run test
   ```

2. **添加 E2E 测试**
   ```bash
   cd frontend
   npm run test:e2e
   ```

3. **部署到生产环境**
   ```bash
   # 构建前端
   cd frontend
   npm run build
   
   # 构建后端
   cd backend
   go build -o fusionmail cmd/server/main.go
   ```

## 📚 更多资源

- [认证逻辑分析与优化方案](./auth-logic-analysis-and-optimization.md)
- [优化完成报告](./auth-optimization-completed.md)
- [优化总结](./OPTIMIZATION_SUMMARY.md)

## 💡 提示

- 开发环境中，测试工具会自动加载到 `window.authTest`
- 生产环境中，测试工具不会被加载
- 所有日志都带有 `[TokenRefresh]` 或 `[AuthTest]` 前缀，便于过滤

---

**祝你测试顺利！** 🎉

如有问题，请查看控制台日志或运行 `authTest.help()` 获取帮助。
