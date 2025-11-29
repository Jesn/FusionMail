# 提供商自动切换问题修复总结

## 修复时间
2025-11-29

## 问题描述

用户在"添加邮箱账户"对话框中手动选择了"QQ 邮箱"后，在邮箱地址输入框中输入 `794382693@` 时，提供商会自动切换到"通用邮箱"，覆盖了用户的选择。

## 根本原因

1. **默认值不算手动选择**：对话框打开时设置的默认值 `provider: 'qq'` 不会触发 `handleProviderChange`，所以 `providerLockedByUser` 保持为 `false`

2. **自动识别过于激进**：只要邮箱地址包含 `@`，就会尝试自动识别提供商

3. **Fallback 逻辑不合理**：`getProviderByEmail` 在无法识别域名时会返回 `generic`（通用邮箱），导致自动切换

## 修复方案

采用**组合方案**，包含三个关键修改：

### 1. 改进 getProviderByEmail 逻辑

**文件：** `frontend/src/hooks/useProviders.ts`

**修改内容：**
- 添加域名有效性检查：如果域名为空或无效，返回 `null`
- 移除 fallback 逻辑：对于未知域名，返回 `null` 而不是 `generic`

**修改前：**
```typescript
const getProviderByEmail = useCallback((email: string): Provider | null => {
  if (!email || !email.includes('@')) {
    return null;
  }

  const domain = email.split('@')[1].toLowerCase();
  
  const domainMappings: Record<string, string> = {
    'qq.com': 'qq',
    // ...
  };

  const providerName = domainMappings[domain];
  if (providerName) {
    return providers.find(p => p.name === providerName) || null;
  }

  // 问题：无法识别时返回 generic
  return providers.find(p => p.name === 'generic') || null;
}, [providers]);
```

**修改后：**
```typescript
const getProviderByEmail = useCallback((email: string): Provider | null => {
  if (!email || !email.includes('@')) {
    return null;
  }

  const domain = email.split('@')[1]?.toLowerCase();
  
  // 新增：域名有效性检查
  if (!domain) {
    return null;
  }
  
  const domainMappings: Record<string, string> = {
    'qq.com': 'qq',
    // ...
  };

  const providerName = domainMappings[domain];
  if (providerName) {
    return providers.find(p => p.name === providerName) || null;
  }

  // 修复：无法识别时返回 null，保持当前选择
  return null;
}, [providers]);
```

### 2. 自动识别成功后锁定

**文件：** `frontend/src/components/account/AccountForm.tsx`

**修改内容：**
- 在 `handleEmailChange` 中，当自动识别成功后设置 `providerLockedByUser = true`
- 移除冗余的 fallback 逻辑

**修改前：**
```typescript
const handleEmailChange = (email: string) => {
  setFormData(prev => ({ ...prev, email }));

  if (!isEditMode && !providerLockedByUser && email.includes('@')) {
    const recommendedProvider = getProviderByEmail(email);
    if (recommendedProvider) {
      setFormData(prev => ({
        ...prev,
        provider: recommendedProvider.name,
        // ...
      }));
      // 问题：没有锁定，后续输入还会继续自动识别
    } else {
      // 冗余的 fallback 逻辑（80+ 行代码）
    }
  }
};
```

**修改后：**
```typescript
const handleEmailChange = (email: string) => {
  setFormData(prev => ({ ...prev, email }));

  if (!isEditMode && !providerLockedByUser && email.includes('@')) {
    const recommendedProvider = getProviderByEmail(email);
    if (recommendedProvider) {
      setFormData(prev => ({
        ...prev,
        provider: recommendedProvider.name,
        // ...
      }));
      
      // 关键修复：自动识别成功后锁定
      setProviderLockedByUser(true);
    }
    // 如果返回 null，保持当前选择，不做任何切换
  }
};
```

### 3. 手动选择后锁定（已存在）

**文件：** `frontend/src/components/account/AccountForm.tsx`

**现有代码：**
```typescript
const handleProviderChange = (provider: string) => {
  setProtocolLockedByUser(false);
  setProviderLockedByUser(true); // 手动选择后锁定
  // ...
};
```

## 修复效果

### 场景1：输入不完整邮箱地址
- **操作：** 打开对话框（默认"QQ 邮箱"），输入 `794382693@`
- **修复前：** 自动切换到"通用邮箱"
- **修复后：** 保持"QQ 邮箱"不变 ✅

### 场景2：输入完整的匹配邮箱
- **操作：** 打开对话框（默认"QQ 邮箱"），输入 `user@163.com`
- **修复前：** 自动切换到"163 邮箱"，继续输入还会变化
- **修复后：** 自动切换到"163 邮箱"并锁定，继续输入不再变化 ✅

### 场景3：手动选择后输入
- **操作：** 手动选择"Gmail"，然后输入 `user@qq.com`
- **修复前：** 自动切换到"QQ 邮箱"
- **修复后：** 保持"Gmail"不变 ✅

### 场景4：输入未知域名
- **操作：** 打开对话框（默认"QQ 邮箱"），输入 `user@unknown.com`
- **修复前：** 自动切换到"通用邮箱"
- **修复后：** 保持"QQ 邮箱"不变，用户需要手动选择"通用邮箱" ✅

### 场景5：重新打开对话框
- **操作：** 关闭并重新打开对话框
- **修复前/后：** 锁定状态被重置，可以重新自动识别 ✅

## 代码变更统计

- **修改文件数：** 2
- **新增代码行：** 8
- **删除代码行：** 85
- **净减少代码：** 77 行

## 相关文件

1. `frontend/src/hooks/useProviders.ts` - 提供商 Hook
2. `frontend/src/components/account/AccountForm.tsx` - 添加账户表单
3. `docs/provider-auto-switch-deep-analysis.md` - 深度分析文档
4. `docs/provider-auto-switch-fix.md` - 原始修复文档
5. `docs/playwright-test-results.md` - Playwright 测试结果

## 测试建议

修复后建议进行以下测试：

1. **基本功能测试**
   - 打开添加账户对话框
   - 测试各种邮箱地址输入场景
   - 验证提供商选择行为

2. **边界情况测试**
   - 输入不完整的邮箱地址（如 `user@`）
   - 输入无效的邮箱地址（如 `@qq.com`）
   - 输入未知域名的邮箱地址

3. **用户体验测试**
   - 手动选择提供商后输入不同域名的邮箱
   - 关闭并重新打开对话框
   - 编辑现有账户

## 部署步骤

1. 前端代码已构建成功
2. 重新构建 Docker 镜像：`docker-compose build`
3. 重启容器：`docker-compose up -d`
4. 验证修复效果

## 注意事项

1. **用户需要手动选择通用邮箱**：对于未知域名的邮箱，系统不会自动切换到"通用邮箱"，用户需要手动选择

2. **锁定机制**：一旦提供商被确定（自动识别或手动选择），就会锁定，不会再自动切换

3. **重置机制**：关闭并重新打开对话框时，锁定状态会被重置

## 总结

通过这次修复，我们：
1. ✅ 解决了手动选择提供商后被自动切换的问题
2. ✅ 改进了自动识别逻辑，更加保守和可预测
3. ✅ 简化了代码，删除了 77 行冗余代码
4. ✅ 提升了用户体验，行为更符合用户预期

修复已完成，可以进行测试和部署。
