# Kiro Agent Hooks 说明

## 概述

本项目配置了多个 Agent Hooks 来自动检查代码安全性，防止敏感信息泄露。

## 可用的 Hooks

### 1. Git 提交前安全检查 🔐

**文件**: `before-git-commit.json`

**触发方式**: 手动触发

**用途**: 在执行 `git commit` 前检查暂存文件是否包含敏感信息

**使用方法**:
1. 在 Kiro IDE 中打开 Agent Hooks 面板
2. 找到 "Git 提交前安全检查" Hook
3. 点击运行
4. 查看检查结果
5. 根据建议决定是否提交

**检查内容**:
- ✅ 邮箱地址（排除占位符）
- ✅ 密码和授权码
- ✅ API 密钥和 Token
- ✅ 敏感配置文件
- ✅ 数据库连接字符串

### 2. 文件保存安全检查 🛡️

**文件**: `on-file-save-security.json`

**触发方式**: 保存文件时自动触发

**用途**: 保存代码文件时自动检查是否包含敏感信息

**适用文件类型**:
- Go 文件 (*.go)
- TypeScript/JavaScript (*.ts, *.tsx, *.js, *.jsx)
- Markdown (*.md)
- 配置文件 (*.yaml, *.yml, *.json)
- Shell 脚本 (*.sh)

**工作方式**:
- 自动在后台运行
- 发现问题时立即警告
- 不会阻止文件保存

### 3. Git 提交安全检查（简化版）🔒

**文件**: `pre-commit-security-check.json`

**触发方式**: 手动触发

**用途**: 快速检查暂存文件的安全性

**特点**:
- 更简洁的检查流程
- 快速反馈
- 适合日常使用

## 使用场景

### 场景 1：日常开发

1. 编写代码
2. 保存文件 → **自动触发** "文件保存安全检查"
3. 如果有警告，立即修复
4. 继续开发

### 场景 2：准备提交

1. 完成开发，准备提交
2. 执行 `git add .`
3. 在 Kiro IDE 中运行 **"Git 提交前安全检查"** Hook
4. 查看检查结果
5. 如果通过，执行 `git commit`
6. 如果有问题，先修复再提交

### 场景 3：快速检查

1. 不确定代码是否安全
2. 运行 **"Git 提交安全检查（简化版）"** Hook
3. 快速获得反馈

## 配置 Hooks

### 查看已配置的 Hooks

在 Kiro IDE 中：
1. 打开命令面板（Cmd/Ctrl + Shift + P）
2. 搜索 "Kiro: Open Agent Hooks"
3. 查看所有可用的 Hooks

### 启用/禁用 Hooks

在 Agent Hooks 面板中：
- 点击 Hook 旁边的开关来启用/禁用
- 自动触发的 Hook 可以临时禁用

### 修改 Hooks

编辑对应的 JSON 文件：
- `before-git-commit.json` - Git 提交前检查
- `on-file-save-security.json` - 文件保存检查
- `pre-commit-security-check.json` - 简化版检查

## Hook 配置说明

### 基本结构

```json
{
  "name": "Hook 名称",
  "description": "Hook 描述",
  "trigger": "manual | onSave | onOpen",
  "filePattern": "**/*.{go,ts,js}",
  "icon": "🔒",
  "category": "git",
  "prompt": "给 AI 的提示词"
}
```

### 触发类型

- `manual`: 手动触发（需要用户点击）
- `onSave`: 保存文件时触发
- `onOpen`: 打开文件时触发
- `onCommit`: Git 提交时触发（计划中）

### 文件模式

使用 glob 模式匹配文件：
- `**/*.go` - 所有 Go 文件
- `**/*.{ts,tsx}` - 所有 TypeScript 文件
- `*.md` - 根目录的 Markdown 文件

## 最佳实践

### 1. 提交前必检查

**强烈建议**在每次 `git commit` 前运行 "Git 提交前安全检查" Hook。

### 2. 关注警告

如果 Hook 发出警告，**不要忽视**：
- 仔细检查标记的内容
- 确认是否为真实的敏感信息
- 如果是，立即修复

### 3. 使用配置文件

不要硬编码敏感信息，使用：
- 环境变量
- 配置文件（添加到 .gitignore）
- 配置模板（.example 文件）

### 4. 定期审查

定期审查 .gitignore 文件，确保：
- 所有敏感文件都被忽略
- 配置正确且完整

## 故障排查

### Hook 没有触发

1. 检查 Hook 是否启用
2. 检查文件类型是否匹配 filePattern
3. 查看 Kiro IDE 日志

### Hook 误报

如果 Hook 将占位符标记为敏感信息：
- 确认占位符格式正确（如 your@example.com）
- 检查 Hook 的检查规则
- 可以临时禁用该 Hook

### Hook 漏报

如果 Hook 没有检测到敏感信息：
- 手动运行 `./check-commit-security.sh`
- 检查 Hook 的检查规则是否完整
- 提交 Issue 改进 Hook

## 相关文档

- [Git 提交安全规范](../.kiro/steering/git-commit-security.md)
- [安全指南](../../SECURITY.md)
- [测试配置说明](../../TEST_SETUP.md)

## 贡献

如果你有改进 Hook 的建议：
1. 修改对应的 JSON 文件
2. 测试 Hook 是否正常工作
3. 提交 Pull Request

## 支持

如果遇到问题：
1. 查看本文档的故障排查部分
2. 查看 Kiro IDE 文档
3. 提交 Issue

---

**记住**：这些 Hooks 是辅助工具，最终的安全责任在于开发者自己。始终保持警惕！
