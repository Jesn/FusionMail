# 端口迁移总结

## 修改概述

已将整个代码库中的端口配置统一修改为：
- **后端端口**: 8080 → **3333**
- **前端端口**: 3000/5173 → **4444**

## 修改的文件类别

### 1. 配置文件
- `docker-compose.yml` - Docker 容器端口映射
- `backend/.env` - 后端环境变量
- `backend/.env.test` - 测试环境变量
- `backend/.env.example` - 环境变量示例
- `frontend/.env` - 前端环境变量
- `frontend/vite.config.ts` - Vite 开发服务器配置

### 2. 文档文件
- `README.md` - 项目主文档
- `DEPLOYMENT.md` - 部署文档
- `docs/*.md` - 所有文档目录下的 Markdown 文件
- `.kiro/steering/*.md` - 项目规范文档
- `.kiro/specs/**/*.md` - 项目规格文档
- `backend/docs/*.md` - 后端文档
- `frontend/docs/*.md` - 前端文档

### 3. 脚本文件
- `start.sh` - 项目启动脚本
- `scripts/start-dev.sh` - 开发环境启动脚本
- `scripts/restart-backend.sh` - 后端重启脚本
- `tests/e2e/run-tests.sh` - E2E 测试运行脚本

### 4. 测试文件
- `tests/e2e/**/*.ts` - 所有 E2E 测试文件
- `tests/e2e/playwright.config.ts` - Playwright 配置
- `tests/e2e/README.md` - 测试文档

### 5. 源代码文件
- `backend/**/*.go` - 后端 Go 代码中的测试配置
- `frontend/src/lib/constants.ts` - 前端常量配置

## 关键配置验证

### 后端配置 (端口 3333)
```bash
# backend/.env
SERVER_PORT=3333

# docker-compose.yml
ports:
  - "${APP_PORT:-3333}:3333"
environment:
  SERVER_PORT: 3333
```

### 前端配置 (端口 4444)
```typescript
// frontend/vite.config.ts
server: {
  port: 4444,
  strictPort: true,
  proxy: {
    '^/api/': {
      target: 'http://localhost:3333',
    },
  },
}
```

### 环境变量
```bash
# frontend/.env
VITE_API_BASE_URL=http://localhost:3333/api/v1
VITE_WS_URL=ws://localhost:3333/ws
```

## 访问地址

修改后的访问地址：
- **前端**: http://localhost:4444
- **后端 API**: http://localhost:3333/api/v1
- **健康检查**: http://localhost:3333/api/v1/health

## 验证结果

✅ 所有 8080 端口引用已清除
✅ 所有 3000/5173 端口引用已清除
✅ 新端口配置已生效
✅ 配置文件、文档、测试、脚本全部更新

## 注意事项

1. **OAuth 回调地址**: 如果使用了 Google/Microsoft OAuth，需要在对应的开发者控制台更新回调地址为 `http://localhost:3333/api/v1/auth/*/callback`

2. **代理配置**: 如果使用了 HTTP 代理，示例端口已从 8080 改为 3128（标准代理端口）

3. **Docker 部署**: 使用 Docker Compose 时，确保 `.env` 文件中的 `APP_PORT` 设置为 3333

4. **测试运行**: 运行测试前确保后端在 3333 端口，前端在 4444 端口启动

## 快速启动

```bash
# 启动整个项目
./start.sh

# 或分别启动
cd backend && ./server  # 后端在 3333
cd frontend && npm run dev  # 前端在 4444
```

---
修改完成时间: 2025-11-04
