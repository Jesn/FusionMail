---
inclusion: always
---

# Git 提交安全规范

## 🔒 隐私保护规则

在进行任何 Git 提交前，必须遵守以下安全规范：

### 禁止提交的敏感信息

**严格禁止**在以下位置包含真实的敏感信息：

1. **代码文件**
   - ❌ 硬编码的邮箱地址
   - ❌ 硬编码的密码或授权码
   - ❌ API 密钥和 Token
   - ❌ 数据库连接字符串（包含密码）
   - ❌ 私钥和证书

2. **配置文件**
   - ❌ 包含真实账号的配置文件
   - ❌ 环境变量文件（.env）
   - ❌ 测试配置文件（.test-config）

3. **文档和注释**
   - ❌ 示例中的真实邮箱地址
   - ❌ 示例中的真实密码
   - ❌ 真实的服务器地址和端口
   - ❌ 真实的用户名和凭证

4. **Git 提交信息**
   - ❌ 提交消息中的邮箱地址
   - ❌ 提交消息中的密码
   - ❌ 提交消息中的任何凭证信息

### 正确的做法

**✅ 使用占位符**

代码示例：
```go
// ❌ 错误
email := "user@example.com"
password := "real_password_123"

// ✅ 正确
email := os.Getenv("EMAIL")
password := os.Getenv("PASSWORD")
```

文档示例：
```markdown
❌ 错误：
邮箱: user@qq.com
密码: abc123xyz

✅ 正确：
邮箱: your@qq.com
密码: your_authorization_code
```

**✅ 使用配置文件模板**

```bash
# ✅ 提交模板文件
.env.example
.test-config.example

# ❌ 不要提交实际配置
.env
.test-config
```

**✅ 更新 .gitignore**

确保以下文件被忽略：
```gitignore
# 敏感配置文件
.env
.env.local
.env.*.local
.test-config
.test_account_uid
测试账号信息.md

# 包含凭证的文件
*_credentials.json
*_secrets.yaml
*.pem
*.key
```

### Git 提交前检查清单

在执行 `git commit` 前，必须检查：

- [ ] 代码中没有硬编码的邮箱地址
- [ ] 代码中没有硬编码的密码或授权码
- [ ] 配置文件使用占位符或环境变量
- [ ] 文档示例使用通用占位符
- [ ] .gitignore 包含所有敏感文件
- [ ] 提交消息不包含敏感信息

### 检查命令

提交前运行以下命令检查：

```bash
# 检查暂存的文件
git diff --cached

# 搜索可能的邮箱地址
git diff --cached | grep -E "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"

# 搜索可能的密码字段
git diff --cached | grep -i "password.*=.*['\"]"

# 检查 .gitignore
cat .gitignore | grep -E "(\.env|\.test-config|credentials|secrets)"
```

### 如果不小心提交了敏感信息

**立即采取行动**：

1. **重置提交**（如果还未推送）
   ```bash
   git reset --soft HEAD~1
   ```

2. **修改文件**
   - 移除敏感信息
   - 使用占位符替换
   - 更新 .gitignore

3. **重新提交**
   ```bash
   git add .
   git commit -m "fix: 移除敏感信息"
   ```

4. **如果已推送到远程**
   ```bash
   # 警告：这会改写历史
   git push --force-with-lease
   ```

5. **更换泄露的凭证**
   - 立即更换密码
   - 重新生成授权码
   - 撤销 API 密钥

### 示例：正确的提交

**✅ 好的提交消息**：
```
feat(test): 添加自动化测试脚本

- 支持从环境变量读取配置
- 支持从配置文件读取配置
- 添加配置文件模板
- 更新 .gitignore
```

**❌ 错误的提交消息**：
```
feat(test): 添加测试脚本

使用账号 user@qq.com 进行测试
密码: abc123xyz
```

### 配置文件管理

**配置文件命名规范**：

```
✅ 提交到 Git：
- .env.example
- .test-config.example
- config.example.yaml

❌ 不要提交：
- .env
- .test-config
- config.yaml（如果包含真实凭证）
```

**配置文件内容示例**：

```bash
# .test-config.example ✅
EMAIL="your@example.com"
PASSWORD="your_password_here"

# .test-config ❌（不提交）
EMAIL="real@qq.com"
PASSWORD="real_password_123"
```

### 代码审查要点

在代码审查时，特别注意：

1. **搜索硬编码的凭证**
   - 邮箱地址模式
   - 密码赋值语句
   - API 密钥字符串

2. **检查配置文件**
   - 是否使用环境变量
   - 是否提供了示例文件
   - 是否在 .gitignore 中

3. **验证文档**
   - 示例是否使用占位符
   - 是否包含安全提示
   - 是否说明如何配置

### 自动化检查

**Git Pre-commit Hook**（可选）：

创建 `.git/hooks/pre-commit`：

```bash
#!/bin/bash

# 检查是否包含邮箱地址
if git diff --cached | grep -qE "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"; then
    echo "⚠️  警告：检测到邮箱地址"
    echo "请确认是否为示例占位符"
    read -p "继续提交？(y/N): " confirm
    if [ "$confirm" != "y" ]; then
        exit 1
    fi
fi

# 检查是否包含密码
if git diff --cached | grep -qiE "password.*=.*['\"][^'\"]{8,}"; then
    echo "⚠️  警告：检测到可能的密码"
    echo "请确认是否为占位符"
    read -p "继续提交？(y/N): " confirm
    if [ "$confirm" != "y" ]; then
        exit 1
    fi
fi

exit 0
```

### 紧急响应流程

如果发现已提交的敏感信息：

1. **评估影响**
   - 信息是否已推送到远程
   - 有多少人可能看到
   - 凭证是否仍然有效

2. **立即行动**
   - 更换所有泄露的凭证
   - 通知相关人员
   - 清理 Git 历史

3. **预防措施**
   - 更新 .gitignore
   - 添加 pre-commit hook
   - 加强代码审查

### 总结

**核心原则**：
- 🔒 永远不要提交真实的敏感信息
- 📝 使用占位符和示例值
- 🔧 使用环境变量和配置文件
- ✅ 提交前仔细检查
- 🚨 发现问题立即处理

**记住**：一旦提交到 Git，即使删除也可能被恢复。预防胜于补救！
