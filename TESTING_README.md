# 测试说明

## ✅ 已解决隐私问题

所有测试脚本已更新，不再包含硬编码的邮箱账号信息。

### 安全措施

1. ✅ 移除了所有硬编码的邮箱地址和密码
2. ✅ 支持从环境变量读取配置
3. ✅ 支持从配置文件读取配置
4. ✅ `.test-config` 已添加到 `.gitignore`
5. ✅ 提供了 `.test-config.example` 作为模板
6. ✅ Git 历史中不包含敏感信息

## 快速开始

### 1. 配置测试账号

```bash
# 复制配置示例
cp .test-config.example .test-config

# 编辑配置文件，填入你的测试账号
vim .test-config
```

### 2. 运行测试

```bash
# 快速测试（推荐）
./quick-test.sh

# 完整测试
./test-qq-account.sh

# 完整验证（包含数据库检查）
./test-and-verify.sh
```

## 配置方式

### 方式 1：配置文件（推荐）

创建 `.test-config` 文件：

```bash
EMAIL="your@qq.com"
PASSWORD="your_authorization_code"
PROVIDER="qq"
PROTOCOL="imap"
```

### 方式 2：环境变量

```bash
export TEST_EMAIL="your@qq.com"
export TEST_PASSWORD="your_authorization_code"
./quick-test.sh
```

## 测试脚本说明

| 脚本 | 用途 | 耗时 | 特点 |
|------|------|------|------|
| `quick-test.sh` | 快速测试 | ~15秒 | 最简单，一键运行 |
| `test-qq-account.sh` | 完整测试 | ~20秒 | 详细步骤，完整流程 |
| `test-and-verify.sh` | 完整验证 | ~30秒 | 包含数据库验证 |

## 获取授权码

### QQ 邮箱
1. 登录 QQ 邮箱网页版
2. 设置 → 账户 → 开启 IMAP/SMTP 服务
3. 生成授权码

### 163 邮箱
1. 登录 163 邮箱网页版
2. 设置 → POP3/SMTP/IMAP
3. 设置客户端授权密码

### Gmail
1. 开启两步验证
2. 生成应用专用密码

### iCloud
1. 登录 appleid.apple.com
2. 生成应用专用密码

## 安全提示

⚠️ **重要**：
- `.test-config` 文件不会被提交到 Git
- 请勿在公开仓库中分享真实账号信息
- 建议使用测试专用邮箱账号
- 定期更换授权码

## 详细文档

- [测试配置说明](TEST_SETUP.md)
- [完整测试指南](docs/testing-guide.md)
- [API 使用示例](docs/api-examples.md)

## 故障排查

### 提示缺少测试账号信息

```bash
# 检查配置文件是否存在
ls -la .test-config

# 如果不存在，创建配置文件
cp .test-config.example .test-config
vim .test-config
```

### 连接测试失败

1. 检查邮箱 IMAP 服务是否开启
2. 确认使用的是授权码而非登录密码
3. 检查网络连接

### 同步没有邮件

- 首次同步只拉取最近 30 天的邮件
- 可以发送测试邮件后再次同步

## 测试流程

```
1. 配置测试账号 (.test-config)
   ↓
2. 启动服务器 (./bin/server)
   ↓
3. 运行测试脚本 (./quick-test.sh)
   ↓
4. 查看测试结果
   ↓
5. 验证数据库数据
```

## 下一步

测试成功后，可以：
1. 开发邮件查询 API
2. 开发前端界面
3. 添加更多邮箱提供商
4. 实现高级功能（规则引擎、Webhook 等）
