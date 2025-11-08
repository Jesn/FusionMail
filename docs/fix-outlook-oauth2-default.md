# 修复 Outlook 默认协议选择问题

## 问题描述

当用户第一次选择 Outlook/Hotmail 提供商时，协议没有自动选择为 OAuth2（推荐），而是保持为 IMAP。但是当用户先选择 Gmail，再切换到 Outlook 时，协议会正确地切换为 OAuth2。

## 问题原因

### 时序问题

1. **组件初始化**：
   - `formData.provider` 默认为 `'qq'`
   - `formData.protocol` 默认为 `'imap'`
   - `providers` 数据开始异步加载

2. **用户第一次选择 Outlook**：
   - 触发 `handleProviderChange('outlook')`
   - 此时 `providers` 可能还没加载完成
   - `getProviderByName('outlook')` 返回 `null`
   - 走到 else 分支，手动设置 `protocol = 'oauth2'`（这部分是正确的）

3. **providers 加载完成**：
   - `providers` 数据加载完成
   - 但是没有重新触发 `handleProviderChange`
   - 表单状态没有更新

### 为什么切换时正常？

当用户先选择 Gmail，再切换到 Outlook 时：
- Gmail 选择时，`providers` 已经加载完成
- 切换到 Outlook 时，`getProviderByName('outlook')` 能正确返回数据
- 使用 `providerInfo.recommended_protocol` 设置协议

## 解决方案

添加一个 `useEffect`，当 `providers` 数据加载完成后，检查当前选择的提供商，如果协议不是推荐协议，则自动更新为推荐协议。

### 实现代码

```typescript
// 当 providers 加载完成后，更新当前选择的提供商配置
useEffect(() => {
  if (!isEditMode && providers.length > 0 && formData.provider) {
    const providerInfo = getProviderByName(formData.provider);
    if (providerInfo && formData.protocol !== providerInfo.recommended_protocol) {
      // 如果当前协议不是推荐协议，更新为推荐协议
      setFormData(prev => ({
        ...prev,
        protocol: providerInfo.recommended_protocol,
        auth_type: providerInfo.recommended_protocol === 'oauth2' ? 'oauth2' : 'password',
        imap_host: providerInfo.imap_host || prev.imap_host,
        imap_port: providerInfo.imap_port || prev.imap_port,
        pop3_host: providerInfo.pop3_host || prev.pop3_host,
        pop3_port: providerInfo.pop3_port || prev.pop3_port,
      }));
    }
  }
}, [providers, formData.provider, isEditMode, getProviderByName]);
```

### 工作流程

1. **组件初始化**：
   - `formData.provider = 'qq'`
   - `formData.protocol = 'imap'`
   - `providers = []`（开始加载）

2. **用户选择 Outlook**：
   - 触发 `handleProviderChange('outlook')`
   - 如果 `providers` 未加载完成，使用 else 分支设置 `protocol = 'oauth2'`
   - 如果 `providers` 已加载完成，使用 `providerInfo.recommended_protocol`

3. **providers 加载完成**：
   - 触发新的 `useEffect`
   - 检查 `formData.provider = 'outlook'`
   - 获取 `providerInfo.recommended_protocol = 'oauth2'`
   - 如果当前 `protocol !== 'oauth2'`，更新为 `'oauth2'`

4. **结果**：
   - 无论 `providers` 何时加载完成
   - Outlook 的协议都会正确设置为 OAuth2

## 测试场景

### 场景 1：providers 加载慢

1. 打开添加账户对话框
2. 立即选择 Outlook/Hotmail
3. ✅ 验证：协议自动选择为 OAuth2

### 场景 2：providers 已加载

1. 打开添加账户对话框
2. 等待 1 秒（确保 providers 加载完成）
3. 选择 Outlook/Hotmail
4. ✅ 验证：协议自动选择为 OAuth2

### 场景 3：切换提供商

1. 打开添加账户对话框
2. 选择 Gmail（协议自动选择为 OAuth2）
3. 切换到 Outlook/Hotmail
4. ✅ 验证：协议保持为 OAuth2

### 场景 4：切换到其他提供商

1. 打开添加账户对话框
2. 选择 Outlook/Hotmail（协议自动选择为 OAuth2）
3. 切换到 QQ 邮箱
4. ✅ 验证：协议自动切换为 IMAP

### 场景 5：编辑模式

1. 编辑现有账户
2. ✅ 验证：不会自动更改协议（保持原有设置）

## 技术细节

### 依赖项说明

```typescript
useEffect(() => {
  // ...
}, [providers, formData.provider, isEditMode, getProviderByName]);
```

- `providers`：当 providers 数据加载完成时触发
- `formData.provider`：当用户切换提供商时触发
- `isEditMode`：确保只在新建模式下生效
- `getProviderByName`：函数依赖（来自 useProviders hook）

### 条件判断

```typescript
if (!isEditMode && providers.length > 0 && formData.provider) {
  // 只在新建模式、providers 已加载、且已选择提供商时执行
}
```

### 协议更新条件

```typescript
if (providerInfo && formData.protocol !== providerInfo.recommended_protocol) {
  // 只有当前协议不是推荐协议时才更新
  // 避免不必要的状态更新
}
```

## 后端配置

后端已正确配置 Outlook 的推荐协议：

```go
"outlook": {
    Name:                "outlook",
    DisplayName:         "Outlook / Hotmail",
    SupportedProtocols:  []string{"oauth2", "imap"},
    RecommendedProtocol: "oauth2",  // ← 推荐协议
    RequiresOAuth:       true,
    IMAPHost:            "outlook.office365.com",
    IMAPPort:            993,
},
```

## 相关文件

- `frontend/src/components/account/AccountForm.tsx` - 主要修改文件
- `frontend/src/hooks/useProviders.ts` - providers 数据加载
- `backend/internal/adapter/factory.go` - 后端提供商配置

## 优势

1. ✅ **自动修正**：即使初始设置不正确，也会在 providers 加载后自动修正
2. ✅ **用户友好**：用户无需手动选择协议，系统自动选择最佳选项
3. ✅ **向后兼容**：不影响编辑模式和其他提供商
4. ✅ **性能优化**：只在必要时更新状态，避免不必要的重渲染

## 更新日期

2025-01-08
