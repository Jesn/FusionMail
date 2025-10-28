# Kiro IDE Agent Hooks 使用指南

## 测试结果

✅ **命令行检查脚本工作正常！**

我们测试了 `./check-commit-security.sh` 脚本，成功检测到：
- ✅ 真实的邮箱地址
- ✅ 硬编码的密码
- ✅ 其他敏感信息

## 在 Kiro IDE 中创建 Agent Hooks

由于 Kiro IDE 的 Agent Hooks 需要通过 UI 界面创建，而不是直接创建 JSON 文件，请按照以下步骤操作：

### 方法 1：通过 UI 创建 Hook

1. **打开 Agent Hooks 面板**
   - 点击侧边栏的 "Agent Hooks" 图标
   - 或使用命令面板：`Cmd/Ctrl + Shift + P` → 搜索 "Kiro: Open Agent Hooks"

2. **创建新的 Hook**
   - 点击 "+" 或 "New Hook" 按钮
   - 填写 Hook 信息

3. **配置 Git 提交前检查 Hook**

   **基本信息**：
   - 名称：`Git 提交安全检查`
   - 描述：`在 Git 提交前自动检查是否包含敏感信息`
   - 图标：🔒
   - 触发方式：`Manual`（手动触发）

   **提示词**：
   ```
   请检查当前暂存的 Git 文件是否包含敏感信息。

   执行以下检查：

   1. 运行安全检查脚本：
   ```bash
   ./check-commit-security.sh
   ```

   2. 分析检查结果：
   - 如果发现问题，列出所有敏感信息
   - 指出具体的文件名和行号
   - 提供修复建议

   3. 如果没有问题：
   - 确认可以安全提交
   - 总结检查的内容

   参考规范：.kiro/steering/git-commit-security.md
   ```

4. **保存 Hook**

### 方法 2：使用命令行脚本（推荐）

由于 Agent Hooks 的配置可能比较复杂，我们提供了独立的命令行脚本：

```bash
# 在提交前运行
./check-commit-security.sh
```

**优势**：
- ✅ 无需配置，直接使用
- ✅ 快速反馈
- ✅ 详细的检查报告
- ✅ 跨平台兼容

### 方法 3：集成到 Git Hooks

创建 `.git/hooks/pre-commit` 文件：

```bash
#!/bin/bash

echo "运行安全检查..."
./check-commit-security.sh

if [ $? -ne 0 ]; then
    echo ""
    echo "⚠️  发现安全问题，提交已阻止"
    echo "请修复上述问题后再次提交"
    exit 1
fi

exit 0
```

然后设置执行权限：

```bash
chmod +x .git/hooks/pre-commit
```

## 测试安全检查

### 测试 1：检测敏感信息

我们创建了一个测试文件 `test-sensitive-info.txt`，包含：
- 真实的邮箱地址
- 硬编码的密码
- API 密钥
- 数据库连接字符串

运行检查：

```bash
git add test-sensitive-info.txt
./check-commit-security.sh
```

**预期结果**：应该检测到多个安全问题 ✅

### 测试 2：正常文件

创建一个只包含占位符的文件：

```bash
echo 'EMAIL="your@example.com"' > test-safe.txt
git add test-safe.txt
./check-commit-security.sh
```

**预期结果**：应该通过检查 ✅

### 测试 3：配置文件

测试 `.gitignore` 是否正确配置：

```bash
./check-commit-security.sh
```

应该确认 `.test-config` 和 `.env` 等文件已被忽略 ✅

## 日常使用流程

### 推荐工作流程

```
1. 编写代码
   ↓
2. git add .
   ↓
3. ./check-commit-security.sh  ← 运行安全检查
   ↓
4. 如果通过 → git commit -m "message"
   如果失败 → 修复问题 → 重新检查
```

### 快速检查命令

```bash
# 查看暂存的文件
git diff --cached --name-only

# 运行完整检查
./check-commit-security.sh

# 手动检查邮箱
git diff --cached | grep -E "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"

# 手动检查密码
git diff --cached | grep -i "password.*="
```

## 检查脚本功能

`./check-commit-security.sh` 会检查：

1. ✅ **邮箱地址**
   - 检测真实邮箱
   - 排除占位符（your@、example.com、test@）

2. ✅ **密码字段**
   - 检测硬编码密码
   - 排除占位符（your_、example、placeholder）

3. ✅ **API 密钥**
   - 检测 API 密钥和 Token
   - 检测长度超过 20 字符的密钥

4. ✅ **敏感文件**
   - 检测 .env、.test-config 等文件
   - 检测私钥文件（.pem、.key）

5. ✅ **.gitignore 配置**
   - 验证敏感文件是否被忽略

6. ✅ **提交消息**
   - 检查最近的提交消息

## 常见问题

### Q: 脚本报告误报怎么办？

A: 如果确认是占位符，可以：
1. 确保使用标准占位符格式（your@、example.com）
2. 临时忽略警告继续提交
3. 修改脚本的检查规则

### Q: 如何禁用某些检查？

A: 编辑 `check-commit-security.sh`，注释掉不需要的检查部分。

### Q: 可以在 CI/CD 中使用吗？

A: 可以！在 CI 配置中添加：

```yaml
# .github/workflows/security-check.yml
- name: Security Check
  run: |
    git diff --cached --name-only
    ./check-commit-security.sh
```

## 相关文档

- [Git 提交安全规范](.kiro/steering/git-commit-security.md)
- [测试配置说明](TEST_SETUP.md)
- [安全指南](SECURITY.md)（如果存在）

## 总结

虽然 Kiro IDE 的 Agent Hooks 需要通过 UI 创建，但我们提供的命令行脚本 `./check-commit-security.sh` 已经可以完美地完成安全检查任务。

**推荐使用命令行脚本**，因为：
- ✅ 简单易用
- ✅ 功能完整
- ✅ 快速反馈
- ✅ 可集成到 Git Hooks
- ✅ 可用于 CI/CD

---

**测试确认**：✅ 安全检查功能正常工作！
