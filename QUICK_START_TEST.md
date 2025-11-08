# 快速测试指南

## 🚀 5 分钟快速测试

### 步骤 1: 编译项目

```bash
cd backend
go build -o server ./cmd/server
```

**预期结果**: 编译成功，无错误 ✅

---

### 步骤 2: 启动服务

```bash
# 确保 PostgreSQL 和 Redis 已启动
./scripts/dev-start.sh

# 或手动启动
./server
```

**预期结果**: 服务在 `http://localhost:3333` 启动 ✅

---

### 步骤 3: 运行测试脚本

```bash
# 在项目根目录
./test_api.sh
```

**预期输出**:
```
🧪 FusionMail API 响应格式测试
================================

📡 检查服务状态...
✅ 服务正在运行

🧪 测试 1: 获取不存在的账户 (应返回 404 + 错误码 2000)
-----------------------------------------------------------
HTTP 状态码: 404
响应内容:
{
  "success": false,
  "code": 2000,
  "error": "账户不存在"
}
✅ 测试通过：返回正确的错误码 2000

...
```

---

### 步骤 4: 手动测试 API

#### 测试 1: 获取不存在的账户
```bash
curl -X GET http://localhost:3333/api/v1/accounts/test-uid | jq
```

**预期响应**:
```json
{
  "success": false,
  "code": 2000,
  "error": "账户不存在"
}
```

#### 测试 2: 获取账户列表
```bash
curl -X GET http://localhost:3333/api/v1/accounts | jq
```

**预期响应**:
```json
{
  "success": true,
  "data": [...]
}
```

#### 测试 3: 获取系统健康状态
```bash
curl -X GET http://localhost:3333/api/v1/system/health | jq
```

**预期响应**:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    ...
  }
}
```

---

## ✅ 验证清单

### 编译和运行
- [ ] 编译成功，无错误
- [ ] 服务启动成功
- [ ] 无运行时错误

### API 响应格式
- [ ] 错误响应包含 `success: false`
- [ ] 错误响应包含 `code` 字段
- [ ] 错误消息清晰易懂
- [ ] 成功响应包含 `success: true`
- [ ] 成功响应包含 `data` 字段

### HTTP 状态码
- [ ] 404 错误返回正确
- [ ] 400 错误返回正确
- [ ] 200 成功返回正确

---

## 🐛 常见问题

### Q1: 编译失败
**解决方案**:
```bash
cd backend
go mod tidy
go build -o server ./cmd/server
```

### Q2: 服务启动失败
**检查**:
- PostgreSQL 是否运行？
- Redis 是否运行？
- 端口 3333 是否被占用？

**解决方案**:
```bash
# 启动数据库
./scripts/dev-start.sh

# 检查端口
lsof -i :3333
```

### Q3: 测试脚本失败
**检查**:
- 服务是否正在运行？
- 是否安装了 `jq`？

**解决方案**:
```bash
# 安装 jq (macOS)
brew install jq

# 检查服务
curl http://localhost:3333/api/v1/system/health
```

---

## 📚 更多信息

- 完整测试文档: [TEST_API_RESPONSES.md](TEST_API_RESPONSES.md)
- 重构总结: [REFACTOR_SUMMARY.md](REFACTOR_SUMMARY.md)
- 重构进度: [REFACTOR_PROGRESS.md](REFACTOR_PROGRESS.md)

---

**提示**: 如果所有测试都通过，说明错误处理重构成功！🎉
