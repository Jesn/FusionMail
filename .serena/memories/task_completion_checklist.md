# FusionMail 任务完成检查清单

## 任务完成后必须执行的步骤

### 1. 代码质量检查

#### Go 后端
```bash
# 代码格式化
gofmt -w .
goimports -w .

# 代码检查
go vet ./...

# 运行测试（如果有）
go test ./...

# 构建验证
go build -o bin/server cmd/server/main.go
```

#### TypeScript 前端
```bash
# 代码格式化
npm run format

# 代码检查
npm run lint

# 类型检查
npm run type-check

# 构建验证
npm run build
```

### 2. 功能验证

#### 后端服务
- 启动后端服务，确保无错误
- 测试相关 API 端点
- 检查数据库连接和操作
- 验证日志输出正常

#### 前端应用
- 启动前端开发服务器
- 测试相关页面和组件
- 检查控制台无错误
- 验证 API 调用正常

### 3. 文档更新
- 更新相关代码注释
- 更新 API 文档（如果有变更）
- 更新 README（如果有新功能）
- 更新任务清单状态

### 4. Git 提交
```bash
# 查看变更
git status
git diff

# 添加文件
git add .

# 提交（遵循 Conventional Commits）
git commit -m "<type>[scope]: <description>"

# 推送
git push origin <branch>
```

### 5. 任务状态更新
- 在 tasks.md 中将任务标记为完成 `[x]`
- 记录完成时间和主要成果（可选）
- 更新任务依赖关系（如果有）

## 验证清单

- [ ] 代码格式化完成
- [ ] 代码检查通过
- [ ] 测试通过（如果有）
- [ ] 构建成功
- [ ] 功能验证通过
- [ ] 文档更新完成
- [ ] Git 提交完成
- [ ] 任务状态更新

## 注意事项

1. **不要跳过验证步骤**：每个步骤都很重要
2. **及时提交代码**：完成一个功能点就提交一次
3. **保持代码整洁**：遵循代码规范和最佳实践
4. **记录问题**：遇到问题及时记录和解决
5. **更新文档**：保持文档与代码同步
