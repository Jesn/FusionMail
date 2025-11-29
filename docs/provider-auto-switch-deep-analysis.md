# 提供商自动切换问题深度分析

## 问题描述

用户在"添加邮箱账户"对话框中：
1. 对话框打开时，默认选择"QQ 邮箱"
2. 用户在邮箱地址输入框中输入 `794382693@`
3. 提供商自动从"QQ 邮箱"切换到"通用邮箱"

**用户期望：** 手动选择了提供商后，无论输入什么邮箱地址，都不应该自动切换。

## 根本原因分析

### 1. 初始化流程

```typescript
// 对话框打开时（account 为 null）
useEffect(() => {
  if (!account) {
    // 设置默认值
    setFormData({
      email: '',
      provider: 'qq',  // 默认选择 QQ 邮箱
      protocol: 'imap',
      // ...
    });
  }
  // 重置锁定状态
  setProviderLockedByUser(false);  // 关键：锁定状态被重置为 false
}, [account, open]);
```

**问题点：**
- 表单初始化时设置了 `provider: 'qq'`（默认值）
- 但这个默认值设置**不会触发** `handleProviderChange` 函数
- 所以 `providerLockedByUser` 保持为 `false`

### 2. 邮箱地址输入流程

```typescript
const handleEmailChange = (email: string) => {
  setFormData(prev => ({ ...prev, email }));

  // 检查条件：!providerLockedByUser
  if (!isEditMode && !providerLockedByUser && email.includes('@')) {
    const recommendedProvider = getProviderByEmail(email);
    if (recommendedProvider) {
      // 自动切换提供商
      setFormData(prev => ({
        ...prev,
        provider: recommendedProvider.name,
        // ...
      }));
    }
  }
};
```

**问题点：**
- 当用户输入 `794382693@` 时，`email.includes('@')` 为 `true`
- `providerLockedByUser` 为 `false`（因为默认值不算手动选择）
- `getProviderByEmail('794382693@')` 返回 `null`（因为域名不完整）
- 但是代码中有 fallback 逻辑，会返回 `generic` 提供商
- 所以提供商被自动切换到"通用邮箱"

### 3. getProviderByEmail 的逻辑

```typescript
const getProviderByEmail = useCallback((email: string): Provider | null => {
  if (!email || !email.includes('@')) {
    return null;
  }

  const domain = email.split('@')[1].toLowerCase();
  
  // 域名映射
  const domainMappings: Record<string, string> = {
    'qq.com': 'qq',
    '163.com': '163',
    // ...
  };

  const providerName = domainMappings[domain];
  if (providerName) {
    return providers.find(p => p.name === providerName) || null;
  }

  // 如果没有匹配的预设提供商，返回通用邮箱
  return providers.find(p => p.name === 'generic') || null;
}, [providers]);
```

**问题点：**
- 当输入 `794382693@` 时，`domain` 为空字符串 `""`
- 空字符串不在 `domainMappings` 中
- 所以返回 `generic` 提供商（通用邮箱）
- 导致提供商被自动切换

## 问题的本质

**核心矛盾：**
1. **默认值不算"手动选择"** - 对话框打开时的默认值不会设置 `providerLockedByUser = true`
2. **自动识别过于激进** - 只要邮箱地址包含 `@`，就会尝试自动识别
3. **Fallback 逻辑不合理** - 无法识别域名时，自动返回"通用邮箱"，而不是保持当前选择

## 解决方案

### 方案A：自动识别后锁定（推荐）

**核心思想：** 无论是默认值、自动识别还是手动选择，只要提供商被确定，就锁定，不再自动切换。

```typescript
const handleEmailChange = (email: string) => {
  setFormData(prev => ({ ...prev, email }));

  // 只有在未锁定时才自动识别
  if (!isEditMode && !providerLockedByUser && email.includes('@')) {
    const recommendedProvider = getProviderByEmail(email);
    if (recommendedProvider) {
      setFormData(prev => ({
        ...prev,
        provider: recommendedProvider.name,
        // ...
      }));
      // 关键：自动识别成功后也锁定
      setProviderLockedByUser(true);
    }
  }
};
```

**优点：**
- 逻辑简单，易于理解
- 用户体验好：提供商一旦确定就不会再变
- 符合用户预期：看到提供商已选好，就不希望它再变

**缺点：**
- 如果用户想修改邮箱地址到另一个域名，提供商不会自动更新
- 但这可以通过手动选择提供商来解决

### 方案B：改进 getProviderByEmail 逻辑

**核心思想：** 只有在能明确识别出提供商时才返回，否则返回 `null`，保持当前选择。

```typescript
const getProviderByEmail = useCallback((email: string): Provider | null => {
  if (!email || !email.includes('@')) {
    return null;
  }

  const domain = email.split('@')[1]?.toLowerCase();
  
  // 如果域名为空或无效，返回 null（不切换）
  if (!domain) {
    return null;
  }

  const domainMappings: Record<string, string> = {
    'qq.com': 'qq',
    '163.com': '163',
    // ...
  };

  const providerName = domainMappings[domain];
  if (providerName) {
    return providers.find(p => p.name === providerName) || null;
  }

  // 关键：无法识别时返回 null，而不是 generic
  return null;
}, [providers]);
```

**优点：**
- 更保守，只在明确识别时才切换
- 允许用户逐步输入邮箱地址

**缺点：**
- 对于非预设提供商的邮箱，永远不会自动切换到"通用邮箱"
- 用户需要手动选择"通用邮箱"

### 方案C：组合方案（最佳）

**结合方案A和方案B的优点：**

1. **改进 getProviderByEmail**：只在明确识别时返回提供商
2. **自动识别后锁定**：一旦识别成功就锁定
3. **手动选择也锁定**：用户手动选择后也锁定

```typescript
// 1. 改进 getProviderByEmail
const getProviderByEmail = useCallback((email: string): Provider | null => {
  if (!email || !email.includes('@')) {
    return null;
  }

  const domain = email.split('@')[1]?.toLowerCase();
  if (!domain) {
    return null;  // 域名为空，返回 null
  }

  const domainMappings: Record<string, string> = {
    'qq.com': 'qq',
    '163.com': '163',
    '126.com': '163',
    'gmail.com': 'gmail',
    'outlook.com': 'outlook',
    'hotmail.com': 'outlook',
    'live.com': 'outlook',
    'icloud.com': 'icloud',
    'me.com': 'icloud',
  };

  const providerName = domainMappings[domain];
  if (providerName) {
    return providers.find(p => p.name === providerName) || null;
  }

  // 对于未知域名，返回 null（保持当前选择）
  return null;
}, [providers]);

// 2. handleEmailChange 中自动识别后锁定
const handleEmailChange = (email: string) => {
  setFormData(prev => ({ ...prev, email }));

  if (!isEditMode && !providerLockedByUser && email.includes('@')) {
    const recommendedProvider = getProviderByEmail(email);
    if (recommendedProvider) {
      setFormData(prev => ({
        ...prev,
        provider: recommendedProvider.name,
        imap_host: recommendedProvider.imap_host || '',
        imap_port: recommendedProvider.imap_port || 993,
        pop3_host: recommendedProvider.pop3_host || '',
        pop3_port: recommendedProvider.pop3_port || 995,
      }));

      if (!protocolLockedByUser) {
        setFormData(prev => ({
          ...prev,
          protocol: recommendedProvider.recommended_protocol,
          auth_type: recommendedProvider.recommended_protocol === 'oauth2' ? 'oauth2' : 'password',
        }));
      }

      // 自动识别成功后锁定
      setProviderLockedByUser(true);
    }
  }
};

// 3. handleProviderChange 保持不变
const handleProviderChange = (provider: string) => {
  setProtocolLockedByUser(false);
  setProviderLockedByUser(true);  // 手动选择后锁定
  // ...
};
```

## 推荐方案

**采用方案C（组合方案）**，理由：

1. **用户体验最佳**
   - 输入 `794382693@` 时，提供商保持"QQ 邮箱"不变
   - 输入完整的 `794382693@qq.com` 时，自动识别为"QQ 邮箱"并锁定
   - 输入 `user@unknown.com` 时，保持当前选择，不会自动切换到"通用邮箱"

2. **逻辑清晰**
   - 只在明确识别出提供商时才切换
   - 一旦确定提供商（自动或手动），就锁定
   - 用户可以随时手动选择其他提供商

3. **符合预期**
   - 默认值"QQ 邮箱"会保持，直到用户输入完整的其他域名邮箱
   - 手动选择的提供商不会被自动切换覆盖
   - 对于未知域名，需要用户手动选择"通用邮箱"

## 实施步骤

1. 修改 `useProviders` Hook 中的 `getProviderByEmail` 函数
2. 修改 `AccountForm` 中的 `handleEmailChange` 函数
3. 测试各种场景
4. 更新文档

## 测试场景

修复后需要测试以下场景：

1. **默认值保持**
   - 打开对话框，默认"QQ 邮箱"
   - 输入 `794382693@`
   - 验证：提供商保持"QQ 邮箱"

2. **自动识别并锁定**
   - 打开对话框，默认"QQ 邮箱"
   - 输入 `user@163.com`
   - 验证：提供商自动切换到"163 邮箱"
   - 继续输入 `user@163.com123`
   - 验证：提供商保持"163 邮箱"（已锁定）

3. **手动选择后锁定**
   - 打开对话框
   - 手动选择"Gmail"
   - 输入 `user@qq.com`
   - 验证：提供商保持"Gmail"（已锁定）

4. **未知域名保持当前**
   - 打开对话框，默认"QQ 邮箱"
   - 输入 `user@unknown.com`
   - 验证：提供商保持"QQ 邮箱"（无法识别，不切换）

5. **重新打开对话框**
   - 关闭并重新打开对话框
   - 验证：锁定状态被重置，可以重新自动识别
