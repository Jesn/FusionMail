# 短效邮箱批量导入功能使用指南

## 功能概述

批量导入功能允许您一次性导入多个短效邮箱账户，适用于快速迁移和测试场景。

## 使用步骤

### 1. 打开批量导入对话框

1. 进入"邮箱账户"页面
2. 点击右上角的"添加账户"按钮
3. 在下拉菜单中选择"批量导入"

### 2. 准备账户数据

账户字符串格式：
```
email----password----refresh_token----client_id
```

**字段说明**：
- `email`: 邮箱地址
- `password`: 密码（可选，用于备用认证）
- `refresh_token`: Microsoft Graph API 的刷新令牌
- `client_id`: Microsoft Azure 应用程序的客户端 ID

**示例**：
```
cohuuexdw097@outlook.com----fqfvqLGz1kIQ----M.C534_BAY.0.U.-CrSmXoA*9zP*UGc7J23aQhYranb0hAF!wbo9ss6P4SN28hlLn3YUwF7s!OrEv2O759zN0zOcrPC8v8erMAshg553ITekSoEIZHIaEiIgjhQ4JIJKdSmfBHSBgmPyv*8o6nMrkgQfzOoMqlY9xlmCDZmfiNebOQgwwCYXBEpi7hEqK*99wZTC32yNOnoEb2hMvvjDePSEio9fbMnaZuzoL6LVka*gz4w5hMR5b058uXtMWGfMsAutjj9mpTuBOc8e7LQ26yLcs*ZLf1XYicLc5V2MPzmv9bL67Mwl3Z7bp7e*6XSrKoiSNCQ0T1p5pz*x9dPDUFl3H0*T!siWR8L*L4QQW61h3kyn6Ngz*zJT*r3fqAvvoAyrJQxWdJ2Kfb4h1lyikdBHQE8Fls9gSqACcfM$----8b4ba9dd-3ea5-4e5f-86f1-ddba2230dcf2
```

### 3. 粘贴账户数据

1. 将准备好的账户字符串粘贴到文本框中
2. 每行一个账户
3. 系统会自动识别账户数量

### 4. 开始导入

1. 点击"开始导入"按钮
2. 等待导入完成（会显示进度条）
3. 查看导入结果

### 5. 查看导入结果

导入完成后，会显示：
- **成功数量**：成功导入的账户数
- **失败数量**：导入失败的账户数
- **详细结果**：每个账户的导入状态和错误信息（如有）

## 注意事项

### 账户格式要求

1. **分隔符**：必须使用 `----` 分隔各个字段
2. **字段顺序**：必须按照 `email----password----refresh_token----client_id` 的顺序
3. **必需字段**：email、refresh_token、client_id 是必需的
4. **可选字段**：password 可以为空，但分隔符必须保留

### 导入限制

- **并发限制**：系统会自动控制并发导入数量，避免过载
- **超时设置**：单个账户导入超时时间为 30 秒
- **错误隔离**：单个账户导入失败不会影响其他账户

### 常见错误

#### 1. 格式错误
**错误信息**：`invalid account string format`

**解决方案**：
- 检查是否使用了正确的分隔符 `----`
- 确保字段数量正确（4个字段）
- 检查是否有多余的空格或换行符

#### 2. 认证失败
**错误信息**：`authentication failed` 或 `invalid credentials`

**解决方案**：
- 检查 refresh_token 是否有效
- 检查 client_id 是否正确
- 确认账户未被禁用或锁定

#### 3. 网络错误
**错误信息**：`network error` 或 `timeout`

**解决方案**：
- 检查网络连接
- 重试导入
- 如果持续失败，联系管理员

## 最佳实践

### 1. 分批导入

如果有大量账户需要导入，建议分批进行：
- 每批 10-20 个账户
- 等待一批完成后再导入下一批
- 避免一次性导入过多账户导致系统负载过高

### 2. 验证数据

导入前建议：
- 先用 1-2 个账户测试
- 确认格式正确后再批量导入
- 保存原始数据备份

### 3. 错误处理

如果导入失败：
- 查看详细错误信息
- 修正问题后重新导入失败的账户
- 不需要重新导入已成功的账户

## 技术细节

### API 端点

```
POST /api/v1/accounts/batch-import
```

### 请求格式

```json
{
  "accounts": [
    "email1----password1----refresh_token1----client_id1",
    "email2----password2----refresh_token2----client_id2"
  ]
}
```

### 响应格式

```json
{
  "success": true,
  "data": {
    "success": 2,
    "failed": 0,
    "results": [
      {
        "email": "email1@example.com",
        "status": "success"
      },
      {
        "email": "email2@example.com",
        "status": "success"
      }
    ]
  }
}
```

## 安全建议

1. **保护敏感信息**：
   - 不要在公共场所显示账户字符串
   - 导入完成后及时清除剪贴板
   - 不要将账户字符串保存在不安全的位置

2. **访问控制**：
   - 确保只有授权用户可以访问批量导入功能
   - 定期审查导入的账户

3. **数据加密**：
   - 系统会自动加密存储 refresh_token
   - 传输过程使用 HTTPS 加密

## 故障排除

### 问题：导入按钮不可用

**可能原因**：
- 文本框为空
- 没有识别到有效的账户格式

**解决方案**：
- 确保粘贴了账户数据
- 检查格式是否正确

### 问题：导入进度卡住

**可能原因**：
- 网络连接问题
- 服务器响应慢

**解决方案**：
- 等待一段时间（最多 2 分钟）
- 如果仍然卡住，刷新页面重试

### 问题：部分账户导入失败

**可能原因**：
- 个别账户的凭证无效
- 网络波动

**解决方案**：
- 查看失败账户的错误信息
- 修正问题后单独重新导入失败的账户

## 更新日志

### v1.0.0 (2025-11-04)
- ✅ 初始版本发布
- ✅ 支持批量导入短效邮箱
- ✅ 实时进度显示
- ✅ 详细的导入结果报告
- ✅ 错误隔离和重试机制

## 相关文档

- [短效邮箱适配器设计文档](../../.kiro/specs/short-term-email-adapter/design.md)
- [短效邮箱适配器需求文档](../../.kiro/specs/short-term-email-adapter/requirements.md)
- [micro.py 对齐测试报告](../../backend/docs/micro-alignment-test-report.md)
