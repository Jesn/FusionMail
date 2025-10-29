# FusionMail 测试指南

## 🚀 快速开始测试

### 启动服务

**方法 1：使用启动脚本**
```bash
./scripts/start-dev.sh
```

**方法 2：手动启动**

终端 1 - 基础设施：
```bash
./scripts/dev-start.sh
```

终端 2 - 后端：
```bash
cd backend && go run cmd/server/main.go
```

终端 3 - 前端：
```bash
cd frontend && npm run dev
```

### 访问地址
- 前端：http://localhost:3000
- 后端：http://localhost:8080
- API：http://localhost:8080/api/v1

---

## 📋 测试清单

### 1. 环境检查
- [ ] PostgreSQL 运行（端口 5432）
- [ ] Redis 运行（端口 6379）
- [ ] 后端启动（端口 8080）
- [ ] 前端启动（端口 3000）

### 2. 账户管理测试
1. [ ] 访问 http://localhost:3000/accounts
2. [ ] 点击"添加账户"
3. [ ] 填写邮箱信息
4. [ ] 提交并验证成功
5. [ ] 测试连接
6. [ ] 手动同步

### 3. 邮件列表测试
1. [ ] 访问 http://localhost:3000/inbox
2. [ ] 验证邮件列表显示
3. [ ] 测试虚拟滚动
4. [ ] 点击邮件查看详情

### 4. 邮件详情测试
1. [ ] 验证邮件正文显示
2. [ ] 测试星标功能
3. [ ] 测试归档功能
4. [ ] 测试删除功能

### 5. 搜索测试
1. [ ] 在头部搜索框输入关键词
2. [ ] 验证搜索结果

---

## 🐛 常见问题

### 后端启动失败
```bash
./scripts/dev-stop.sh
./scripts/dev-start.sh
```

### 前端无法连接后端
检查 `frontend/.env` 配置

### 邮件同步失败
1. 检查邮箱授权码
2. 检查网络连接
3. 查看后端日志

---

## ✅ 测试完成

所有测试通过后，FusionMail 核心功能可正常使用！
