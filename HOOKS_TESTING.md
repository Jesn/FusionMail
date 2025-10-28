# Agent Hooks 测试指南

## 如何查看 Hooks

### 方法 1：通过侧边栏

1. 在 Kiro IDE 左侧边栏找到 "Agent Hooks" 图标
2. 点击打开 Hooks 面板
3. 应该能看到所有配置的 Hooks

### 方法 2：通过命令面板

1. 按 `Cmd/Ctrl + Shift + P` 打开命令面板
2. 输入 "Kiro: Open Agent Hooks"
3. 选择并执行

### 方法 3：通过菜单

1. 点击顶部菜单
2. 找到 "View" 或 "窗口"
3. 选择 "Agent Hooks"

## 已配置的 Hooks

### 1. 测试简单 Hook 🧪

**ID**: `test-simple-hook`  
**触发**: 手动  
**用途**: 测试 Hook 系统是否正常工作

**测试步骤**:
1. 在 Hooks 面板找到 "测试简单 Hook"
2. 点击运行按钮
3. 应该看到 AI 回复："Hook 已成功触发！"

### 2. Git 提交安全检查 🔒

**ID**: `pre-commit-security-check`  
**触发**: 手动  
**用途**: 快速检查暂存文件的安全性

**测试步骤**:
1. 暂存一个包含敏感信息的文件：
   ```bash
   git add test-sensitive-info.txt
   ```

2. 在 Hooks 面板运行 "Git 提交安全检查"

3. AI 应该检测到以下问题：
   - 邮箱地址: test123@qq.com
   - 密码: mypassword123456
   - API 密钥: sk-1234567890...

4. AI 应该给出修复建议

### 3. Git 提交前安全检查 🔐

**ID**: `before-git-commit`  
**触发**: 手动  
**用途**: 完整的提交前安全检查

**测试步骤**:
1. 确保有暂存的文件
2. 运行此 Hook
3. AI 会执行完整的检查流程
4. 提供详细的分析和建议

### 4. 文件保存安全检查 🛡️

**ID**: `on-file-save-security`  
**触发**: 保存文件时自动  
**用途**: 实时检查保存的文件

**测试步骤**:
1. 创建或打开一个 Go/TS/JS 文件
2. 添加一个真实的邮箱地址：
   ```go
   email := "realuser@qq.com"
   ```
3. 保存文件（Cmd/Ctrl + S）
4. 应该自动触发检查
5. AI 应该警告发现敏感信息

## 故障排查

### 问题 1：看不到 Hooks 面板

**可能原因**:
- Hooks 功能未启用
- Kiro IDE 版本过旧
- 配置文件格式错误

**解决方法**:
1. 检查 Kiro IDE 版本
2. 重启 Kiro IDE
3. 查看控制台是否有错误信息

### 问题 2：Hooks 不显示

**可能原因**:
- JSON 配置文件格式错误
- 缺少必需字段
- 文件路径不正确

**解决方法**:
1. 验证 JSON 格式：
   ```bash
   cat .kiro/hooks/test-simple-hook.json | python3 -m json.tool
   ```

2. 检查必需字段：
   - `id`: 唯一标识符
   - `name`: 显示名称
   - `trigger`: 触发方式
   - `prompt`: AI 提示词

3. 查看 `.kiro/hooks/index.json` 是否正确

### 问题 3：Hook 不触发

**可能原因**:
- Hook 被禁用
- 触发条件不满足
- 文件模式不匹配

**解决方法**:
1. 检查 `enabled` 字段是否为 `true`
2. 对于 `onSave` Hook，确保文件类型匹配 `filePattern`
3. 查看 Kiro IDE 日志

### 问题 4：AI 没有执行命令

**可能原因**:
- AI 没有权限执行命令
- 命令格式不正确
- 工作目录不对

**解决方法**:
1. 检查 `.kiro/settings/kiro.project.json` 权限配置
2. 确保命令在代码块中：\`\`\`bash ... \`\`\`
3. 使用绝对路径或相对于项目根目录的路径

## 验证 Hooks 配置

### 检查 JSON 格式

```bash
# 验证所有 Hook 配置文件
for file in .kiro/hooks/*.json; do
    echo "检查: $file"
    python3 -m json.tool "$file" > /dev/null && echo "✓ 格式正确" || echo "✗ 格式错误"
done
```

### 列出所有 Hooks

```bash
# 查看已配置的 Hooks
cat .kiro/hooks/index.json | python3 -m json.tool
```

### 查看 Hook 详情

```bash
# 查看特定 Hook 的配置
cat .kiro/hooks/test-simple-hook.json | python3 -m json.tool
```

## 测试流程

### 完整测试流程

1. **测试基础功能**
   ```bash
   # 运行 "测试简单 Hook"
   # 验证 Hook 系统是否工作
   ```

2. **测试安全检查**
   ```bash
   # 暂存测试文件
   git add test-sensitive-info.txt
   
   # 运行 "Git 提交安全检查"
   # 验证是否检测到敏感信息
   ```

3. **测试自动触发**
   ```bash
   # 创建测试文件
   echo 'email := "test@qq.com"' > test-auto.go
   
   # 保存文件
   # 验证是否自动触发检查
   ```

4. **测试完整流程**
   ```bash
   # 修改代码
   # 暂存文件
   # 运行 "Git 提交前安全检查"
   # 根据建议修复问题
   # 重新检查
   # 提交代码
   ```

## 预期结果

### 测试简单 Hook

**输入**: 运行 Hook  
**输出**: "Hook 已成功触发！" + 时间戳和工作目录

### Git 提交安全检查

**输入**: 暂存 `test-sensitive-info.txt`  
**输出**: 
- ❌ 发现 3 个问题
- 📍 具体位置和内容
- 💡 修复建议

### 文件保存安全检查

**输入**: 保存包含 `realuser@qq.com` 的文件  
**输出**:
- ⚠️ 警告：发现真实邮箱地址
- 💡 建议使用环境变量

## 下一步

如果 Hooks 正常工作：
1. ✅ 可以安全地开发和提交代码
2. ✅ 享受自动化安全检查
3. ✅ 根据需要自定义 Hooks

如果 Hooks 不工作：
1. 📝 记录错误信息
2. 🔍 查看 Kiro IDE 日志
3. 📧 联系支持或提交 Issue

## 相关文档

- [Hooks README](.kiro/hooks/README.md)
- [Git 提交安全规范](.kiro/steering/git-commit-security.md)
- [安全指南](SECURITY.md)

---

**提示**: 如果你在 Kiro IDE 中看不到 Hooks，可能需要：
1. 重启 Kiro IDE
2. 检查 Kiro IDE 版本
3. 查看是否有配置错误
