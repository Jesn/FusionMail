# 测试配置说明

## 快速开始

### 1. 配置测试账号

**方式 A：使用配置文件（推荐）**

```bash
# 复制配置示例
cp .test-config.example .test-config

# 编辑配置文件
vim .test-config
```

在 `.test-config` 中填入你的测试账号信息：

```bash
EMAIL="your@qq.com"
PASSWORD="your_authorization_code"
PROVIDER="qq"
PROTOCOL="imap"
```

**方式 B：使用环境变量**

```bash
export TEST_EMAIL="your@qq.com"
export TEST_PASSWORD="your_authorization_code"
export TEST_PROVIDER="qq"
export TEST_PROTOCOL="imap"
```

### 2. 运行测试

```bash
# 快速测试
./quick-test.sh

# 完整测试
./test-qq-account.sh

# 完整验证
./test-and-verify.sh
```

## 获取 QQ 邮箱授权码

1. 登录 QQ 邮箱网页版
2. 设置 → 账户
3. 找到 "POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务"
4. 开启 "IMAP/SMTP 服务"
5. 生成授权码（这不是你的 QQ 密码！）

## 获取其他邮箱授权码

### 163 邮箱
1. 登录 163 邮箱网页版
2. 设置 → POP3/SMTP/IMAP
3. 开启 IMAP/SMTP 服务
4. 设置客户端授权密码

### Gmail
1. 开启两步验证
2. Google 账户 → 安全性 → 应用专用密码
3. 生成新的应用专用密码

### iCloud
1. 登录 appleid.apple.com
2. 安全 → 应用专用密码
3. 生成新密码

## 安全提示

⚠️ **重要**：
- `.test-config` 文件已添加到 `.gitignore`，不会被提交
- 请勿在公开仓库中提交包含真实账号信息的文件
- 建议使用测试专用邮箱账号
- 定期更换授权码

## 故障排查

### 连接失败

1. 检查邮箱 IMAP 服务是否开启
2. 确认使用的是授权码而非登录密码
3. 检查网络连接
4. 查看服务器日志

### 配置文件不生效

```bash
# 检查文件是否存在
ls -la .test-config

# 检查文件内容
cat .test-config

# 确保文件格式正确（无多余空格）
```

## 更多信息

详细的测试指南请查看：
- [测试指南](docs/testing-guide.md)
- [API 示例](docs/api-examples.md)
