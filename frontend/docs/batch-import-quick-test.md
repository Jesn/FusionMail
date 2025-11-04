# 批量导入功能快速测试指南

## 🚀 快速开始

### 1. 启动前端开发服务器

```bash
cd frontend
npm run dev
```

### 2. 访问账户页面

打开浏览器访问：`http://localhost:5173/accounts`

### 3. 打开批量导入对话框

1. 点击右上角的"添加账户"按钮
2. 在下拉菜单中选择"批量导入"

### 4. 测试账户数据

使用以下测试账户（已验证可用）：

```
cohuuexdw097@outlook.com----fqfvqLGz1kIQ----M.C534_BAY.0.U.-CrSmXoA*9zP*UGc7J23aQhYranb0hAF!wbo9ss6P4SN28hlLn3YUwF7s!OrEv2O759zN0zOcrPC8v8erMAshg553ITekSoEIZHIaEiIgjhQ4JIJKdSmfBHSBgmPyv*8o6nMrkgQfzOoMqlY9xlmCDZmfiNebOQgwwCYXBEpi7hEqK*99wZTC32yNOnoEb2hMvvjDePSEio9fbMnaZuzoL6LVka*gz4w5hMR5b058uXtMWGfMsAutjj9mpTuBOc8e7LQ26yLcs*ZLf1XYicLc5V2MPzmv9bL67Mwl3Z7bp7e*6XSrKoiSNCQ0T1p5pz*x9dPDUFl3H0*T!siWR8L*L4QQW61h3kyn6Ngz*zJT*r3fqAvvoAyrJQxWdJ2Kfb4h1lyikdBHQE8Fls9gSqACcfM$----8b4ba9dd-3ea5-4e5f-86f1-ddba2230dcf2
```

### 5. 执行导入

1. 将测试账户粘贴到文本框
2. 确认显示"已识别 1 个账号"
3. 点击"开始导入"按钮
4. 等待导入完成
5. 查看导入结果

## ✅ 预期结果

### 成功场景

**显示内容**：
- 成功数量：1
- 失败数量：0
- 详细结果：
  - ✓ cohuuexdw097@outlook.com - 成功

**Toast 通知**：
- "成功导入 1 个账户"

### 失败场景（如果 token 过期）

**显示内容**：
- 成功数量：0
- 失败数量：1
- 详细结果：
  - ✗ cohuuexdw097@outlook.com - 失败
  - 错误信息：[具体错误]

**Toast 通知**：
- "1 个账户导入失败"

## 🧪 测试用例

### 测试用例 1：单个账户导入

**输入**：
```
cohuuexdw097@outlook.com----fqfvqLGz1kIQ----[token]----8b4ba9dd-3ea5-4e5f-86f1-ddba2230dcf2
```

**预期**：成功导入 1 个账户

### 测试用例 2：多个账户导入

**输入**：
```
account1@outlook.com----pass1----token1----client_id1
account2@outlook.com----pass2----token2----client_id2
account3@outlook.com----pass3----token3----client_id3
```

**预期**：显示 3 个账户的导入结果

### 测试用例 3：格式错误

**输入**：
```
invalid-format-account
```

**预期**：不识别任何账户，导入按钮禁用

### 测试用例 4：空输入

**输入**：（空）

**预期**：导入按钮禁用

### 测试用例 5：混合有效和无效

**输入**：
```
valid@outlook.com----pass----token----client_id
invalid-format
another@outlook.com----pass----token----client_id
```

**预期**：识别 2 个有效账户

## 🔍 调试检查

### 前端检查

1. **浏览器控制台**：
   - 打开开发者工具（F12）
   - 查看 Console 标签
   - 检查是否有错误信息

2. **网络请求**：
   - 打开 Network 标签
   - 查找 `/api/v1/accounts/batch-import` 请求
   - 检查请求和响应数据

3. **React DevTools**：
   - 检查 `BatchImportDialog` 组件状态
   - 查看 `accountsText` 和 `importResult` 状态

### 后端检查

1. **后端日志**：
```bash
# 查看后端日志
cd backend
go run cmd/server/main.go
```

2. **API 测试**：
```bash
# 使用 curl 测试 API
curl -X POST http://localhost:3333/api/v1/accounts/batch-import \
  -H "Content-Type: application/json" \
  -d '{
    "accounts": [
      "cohuuexdw097@outlook.com----fqfvqLGz1kIQ----[token]----8b4ba9dd-3ea5-4e5f-86f1-ddba2230dcf2"
    ]
  }'
```

## 🐛 常见问题

### 问题 1：导入按钮一直禁用

**原因**：账户格式不正确

**解决**：
- 检查是否使用了 `----` 分隔符
- 确保每行都是完整的账户字符串
- 删除空行

### 问题 2：导入后没有反应

**原因**：后端 API 未实现或未启动

**解决**：
- 确认后端服务正在运行
- 检查 API 端点是否正确
- 查看浏览器控制台错误

### 问题 3：导入成功但列表未更新

**原因**：当前使用 `window.location.reload()`

**解决**：
- 这是预期行为
- 页面会自动刷新
- 后续版本会优化为无刷新更新

### 问题 4：所有账户都导入失败

**原因**：Token 可能已过期

**解决**：
- 使用最新的测试账户
- 检查 token 有效期
- 联系管理员获取新的测试账户

## 📊 性能测试

### 小批量测试（1-10 个账户）

**预期**：
- 导入时间：< 10 秒
- 进度条流畅
- 无明显卡顿

### 中批量测试（10-50 个账户）

**预期**：
- 导入时间：< 30 秒
- 进度条正常
- 可能有短暂等待

### 大批量测试（50+ 个账户）

**建议**：
- 分批导入
- 每批 20-30 个
- 避免一次性导入过多

## ✨ UI/UX 检查

### 视觉检查

- [ ] 对话框居中显示
- [ ] 格式说明清晰可见
- [ ] 进度条动画流畅
- [ ] 结果卡片布局合理
- [ ] 滚动区域正常工作

### 交互检查

- [ ] 文本框可以正常输入
- [ ] 按钮状态正确切换
- [ ] 对话框可以正常关闭
- [ ] Toast 通知正常显示
- [ ] 错误信息清晰易懂

### 响应式检查

- [ ] 桌面端显示正常
- [ ] 平板端显示正常
- [ ] 移动端显示正常（如果支持）

## 🎯 验收标准

- [ ] 可以打开批量导入对话框
- [ ] 可以输入和解析账户字符串
- [ ] 可以显示识别的账户数量
- [ ] 可以执行导入操作
- [ ] 可以显示导入进度
- [ ] 可以显示导入结果统计
- [ ] 可以显示详细结果列表
- [ ] 可以显示错误信息
- [ ] 可以关闭对话框
- [ ] 导入后列表会更新

## 📝 测试报告模板

```markdown
## 批量导入功能测试报告

**测试日期**：YYYY-MM-DD
**测试人员**：[姓名]
**测试环境**：
- 浏览器：[Chrome/Firefox/Safari] [版本]
- 前端版本：[版本号]
- 后端版本：[版本号]

### 测试结果

| 测试用例 | 状态 | 备注 |
|---------|------|------|
| 单个账户导入 | ✅/❌ | |
| 多个账户导入 | ✅/❌ | |
| 格式错误处理 | ✅/❌ | |
| 空输入处理 | ✅/❌ | |
| 进度显示 | ✅/❌ | |
| 结果展示 | ✅/❌ | |
| 错误处理 | ✅/❌ | |

### 发现的问题

1. [问题描述]
   - 重现步骤：
   - 预期结果：
   - 实际结果：
   - 严重程度：

### 建议

1. [改进建议]

### 总体评价

[通过/不通过]
```

## 🔗 相关资源

- [批量导入使用指南](./batch-import-guide.md)
- [批量导入实现文档](./batch-import-implementation.md)
- [后端 API 文档](../../backend/docs/)
