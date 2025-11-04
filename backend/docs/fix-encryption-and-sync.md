# 修复加密和同步问题

## 问题总结

1. **长效邮箱解密失败** - 旧账户使用了不同的加密密钥
2. **短效邮箱同步失败** - 凭证格式不正确，缺少 refresh_token

## 已修复的问题

### 1. 账户服务 - 短效邮箱凭证加密

**文件**: `backend/internal/service/account_service.go`

**修改内容**:
- 在 `Create` 方法中添加了对 `quick` 认证类型的支持
- 短效邮箱的凭证现在以 JSON 格式加密，包含 `refresh_token` 和 `client_id`
- 密码认证仍然直接加密密码字符串

```go
if req.AuthType == "quick" {
    // 短效认证：加密 JSON 格式的凭证
    credentials := map[string]interface{}{
        "email":         req.Email,
        "auth_type":     "quick",
        "refresh_token": req.RefreshToken,
        "client_id":     req.ClientID,
    }
    
    credentialsJSON, err := json.Marshal(credentials)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal credentials: %w", err)
    }
    
    encryptedCredentials, err = s.encryptor.Encrypt(string(credentialsJSON))
    if err != nil {
        return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
    }
} else {
    // 密码认证：直接加密密码
    encryptedCredentials, err = s.encryptor.Encrypt(req.Password)
    if err != nil {
        return nil, fmt.Errorf("failed to encrypt password: %w", err)
    }
}
```

### 2. 同步服务 - 短效邮箱凭证解析

**文件**: `backend/internal/service/sync_service.go`

**修改内容**:
- 在 `parseCredentials` 方法中添加了对 `quick` 认证类型的支持
- 正确解析短效邮箱的 JSON 格式凭证
- 使用 `CreateProviderAuto` 方法自动选择适配器类型

```go
} else if account.AuthType == "quick" {
    // 短效认证凭证是 JSON 格式
    var quickCreds struct {
        Email        string `json:"email"`
        AuthType     string `json:"auth_type"`
        RefreshToken string `json:"refresh_token"`
        ClientID     string `json:"client_id"`
    }

    if err := json.Unmarshal([]byte(decryptedData), &quickCreds); err != nil {
        return nil, fmt.Errorf("failed to parse quick credentials: %w", err)
    }

    credentials.RefreshToken = quickCreds.RefreshToken
    credentials.ClientID = quickCreds.ClientID
    // 短效适配器不需要 ClientSecret
}
```

### 3. 同步服务 - 智能适配器选择

**修改内容**:
- 使用 `CreateProviderAuto` 替代 `CreateProviderFromAccount`
- 自动根据凭证信息选择正确的适配器类型（标准 vs 短效）

```go
// 创建适配器配置
config := &adapter.Config{
    Provider:    account.Provider,
    Protocol:    account.Protocol,
    Credentials: credentials,
    Proxy:       proxy,
    Timeout:     0, // 使用默认超时
}

// 使用自动选择方法创建适配器（会智能判断是否使用短效适配器）
provider, err := s.adapterFactory.CreateProviderAuto(config)
if err != nil {
    return fmt.Errorf("failed to create adapter: %w", err)
}
```

## 解决方案

### 方案 1: 创建凭证迁移工具（推荐 - 保留现有数据）

为旧账户创建一个凭证重新加密工具，这样可以保留所有邮件数据。

**注意**：这个方案需要你提供旧的加密密钥。如果你不记得旧密钥，只能使用方案 2。

### 方案 2: 只修复短效邮箱（最安全）

只删除和重新导入短效邮箱，保留所有长效邮箱的数据：

#### 步骤：

1. **删除旧的短效邮箱账户**
```bash
cd backend
export DATABASE_URL="postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail?sslmode=disable"
go run scripts/delete_quick_accounts.go
```

2. **重启后端服务**
```bash
# 停止当前服务
pkill -f "go run cmd/server/main.go"

# 启动新服务
cd backend
go run cmd/server/main.go
```

3. **重新导入短效邮箱**
   - 使用前端的批量导入功能
   - 或使用测试脚本：
```bash
export QUICK_ACCOUNT="你的短效邮箱字符串"
go run scripts/test_batch_import_fixed.go
```

### 方案 3: 手动处理长效邮箱（如果需要）

对于无法解密的长效邮箱账户，你可以：

1. **通过前端界面删除单个账户**
   - 在账户管理页面找到对应账户
   - 点击删除按钮
   - 这样可以精确控制删除哪些账户

2. **重新通过 OAuth2 添加账户**
   - 使用前端的添加账户功能
   - 重新进行 OAuth2 授权
   - 系统会自动同步邮件

## 测试验证

### 1. 测试短效邮箱导入和同步

```bash
cd backend
export DATABASE_URL="postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail?sslmode=disable"
export QUICK_ACCOUNT="outlook|cohuuexdw097@outlook.com|refresh_token|client_id"
go run scripts/test_batch_import_fixed.go
```

预期输出：
```
=== 步骤 1: 解析账户字符串 ===
邮箱: cohuuexdw097@outlook.com
Provider: outlook
RefreshToken: 0.AXoA...
ClientID: 00000000-0000-0000-0000-000000000000

=== 步骤 2: 创建账户 ===
✓ 账户创建成功
  UID: xxx-xxx-xxx
  Email: cohuuexdw097@outlook.com
  Provider: outlook
  Protocol: graph_quick
  AuthType: quick
  Status: active

=== 步骤 3: 验证凭证加密 ===
加密凭证长度: 200+

=== 步骤 4: 测试同步 ===
✓ 同步完成

=== 步骤 5: 检查同步结果 ===
数据库中的邮件数量: 10+

最近的同步日志:
  状态: success
  拉取邮件数: 10+
  新增邮件数: 10+
  更新邮件数: 0
  耗时: xxx ms

=== 测试完成 ===
✓ 短效邮箱批量导入和同步功能正常
```

### 2. 诊断所有账户状态

```bash
cd backend
export DATABASE_URL="postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail?sslmode=disable"
export ENCRYPTION_KEY="test-encryption-key-32-bytes-long!!"
go run scripts/diagnose_all_accounts.go
```

这个脚本会显示：
- 所有账户的基本信息
- 凭证是否能正确解密
- 凭证的内容（脱敏显示）

### 3. 测试同步功能

```bash
cd backend
export DATABASE_URL="postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail?sslmode=disable"
go run scripts/test_sync_simple.go
```

## 注意事项

1. **加密密钥一致性**
   - 确保 `.env` 文件中的 `ENCRYPTION_KEY` 保持不变
   - 当前密钥：`test-encryption-key-32-bytes-long!!`
   - 如果修改密钥，所有账户都需要重新导入

2. **后端服务重启**
   - 修改代码后必须重启后端服务
   - 确保没有旧的进程在运行

3. **数据库状态**
   - 删除账户会同时删除相关的邮件和同步日志
   - 建议在删除前备份重要数据

4. **短效邮箱字符串格式**
   - 格式：`provider|email|refresh_token|client_id`
   - 示例：`outlook|user@outlook.com|0.AXoA...|00000000-0000-0000-0000-000000000000`

## 后续优化建议

1. **添加凭证迁移工具**
   - 创建工具来重新加密旧账户的凭证
   - 避免需要重新导入所有账户

2. **改进错误提示**
   - 在前端显示更友好的错误信息
   - 区分加密错误和其他同步错误

3. **添加凭证验证**
   - 在创建账户时验证凭证格式
   - 在同步前验证凭证是否有效

4. **支持凭证更新**
   - 允许用户更新账户凭证
   - 支持 token 刷新失败后的手动更新
