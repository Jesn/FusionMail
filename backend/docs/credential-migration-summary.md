# 凭证迁移问题总结

## 问题分析

通过分析发现，数据库中的 6 个账户使用了不同的加密密钥：

| 账户 | 状态 | 说明 |
|------|------|------|
| cohuuexdw097@outlook.com | ✓ 可解密 | 使用当前密钥，已修复 |
| jesn@linux.do | ✗ 无法解密 | 使用未知密钥 |
| 794382693@qq.com | ✗ 无法解密 | 使用未知密钥 |
| 15026732619@163.com | ✗ 无法解密 | 使用未知密钥 |
| jesn2013@gmail.com | ✗ 无法解密 | 使用未知密钥 |
| jesn2013@hotmail.com | ✗ 无法解密 | 使用未知密钥 |

## 尝试过的密钥

已尝试以下可能的密钥，均无法解密：
1. `test-encryption-key-32-bytes-long!!` (当前密钥)
2. `fusionmail-default-key-32-bytes` (代码默认值)
3. `dev-secret-key-for-testing-only`
4. `fusionmail_dev_encryption_key_32`
5. `development-encryption-key-32!!`
6. `admin123`
7. 空字符串（使用默认值）

## 解决方案

### 方案 1：手动重新添加账户（推荐）

由于无法找到正确的加密密钥，建议通过前端界面手动处理：

1. **删除无法解密的账户**
   - 登录前端界面
   - 进入账户管理页面
   - 删除以下账户：
     - jesn@linux.do
     - 794382693@qq.com
     - 15026732619@163.com
     - jesn2013@gmail.com
     - jesn2013@hotmail.com

2. **重新添加账户**
   - Gmail (jesn2013@gmail.com): 通过 OAuth2 重新授权
   - Outlook (jesn2013@hotmail.com): 通过 OAuth2 重新授权
   - QQ (794382693@qq.com): 使用授权码重新添加
   - 163 (15026732619@163.com): 使用授权码重新添加
   - Linux.do (jesn@linux.do): 使用密码重新添加

3. **验证同步功能**
   - 添加账户后，检查同步状态
   - 确认邮件能正常拉取

### 方案 2：如果记得旧密钥

如果你记得之前使用的加密密钥，可以运行迁移工具：

```bash
cd backend
export DATABASE_URL="postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail?sslmode=disable"
export ENCRYPTION_KEY="test-encryption-key-32-bytes-long!!"
export OLD_ENCRYPTION_KEY="你的旧密钥"
./scripts/run_migrate.sh
```

## 已修复的问题

1. **短效邮箱凭证格式** - 已修复 `account_service.go`，现在正确加密 JSON 格式的凭证
2. **同步服务凭证解析** - 已修复 `sync_service.go`，现在支持解析短效邮箱凭证
3. **智能适配器选择** - 同步服务现在使用 `CreateProviderAuto` 自动选择正确的适配器

## 后续建议

1. **统一加密方式**
   - 确保所有服务使用相同的加密器
   - 在创建账户时记录使用的加密密钥版本

2. **添加密钥版本管理**
   - 在数据库中记录每个账户使用的密钥版本
   - 支持多版本密钥共存

3. **改进错误提示**
   - 在前端显示更友好的错误信息
   - 区分"密钥错误"和"其他同步错误"

4. **备份机制**
   - 在修改加密密钥前提醒用户备份
   - 提供凭证导出/导入功能
