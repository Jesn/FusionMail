# 邮箱提供商自动切换问题修复

## 问题描述

在"添加邮箱账户"页面，当用户手动选择了某个邮箱提供商（如 QQ 邮箱）后，在输入邮箱地址时（如输入 `794382693@`），系统会根据邮箱域名自动切换提供商到"通用邮箱"，覆盖了用户的手动选择。

## 问题原因

在 `AccountForm` 组件的 `handleEmailChange` 函数中，每当邮箱地址包含 `@` 符号时，都会调用 `getProviderByEmail(email)` 来自动识别提供商，**无论用户是否已经手动选择了提供商**。

原有逻辑：
```typescript
const handleEmailChange = (email: string) => {
  setFormData(prev => ({ ...prev, email }));

  // 问题：总是自动识别提供商，即使用户已手动选择
  if (!isEditMode && email.includes('@')) {
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

## 解决方案

添加一个状态 `providerLockedByUser` 来跟踪用户是否手动选择了提供商：

### 1. 添加状态变量

```typescript
const [providerLockedByUser, setProviderLockedByUser] = useState(false);
```

### 2. 修改 `handleEmailChange` 函数

只有在用户**没有手动选择提供商**时，才根据邮箱地址自动识别提供商：

```typescript
const handleEmailChange = (email: string) => {
  setFormData(prev => ({ ...prev, email }));

  // 只有在用户没有手动选择提供商时，才自动识别
  if (!isEditMode && !providerLockedByUser && email.includes('@')) {
    const recommendedProvider = getProviderByEmail(email);
    if (recommendedProvider) {
      setFormData(prev => ({
        ...prev,
        provider: recommendedProvider.name,
        // ...
      }));
    }
  }
};
```

### 3. 修改 `handleProviderChange` 函数

当用户手动选择提供商时，设置 `providerLockedByUser` 为 `true`：

```typescript
const handleProviderChange = (provider: string) => {
  setProtocolLockedByUser(false);
  setProviderLockedByUser(true); // 标记用户已手动选择提供商
  // ...
};
```

### 4. 重置状态

在对话框关闭或重新打开时，重置 `providerLockedByUser` 状态：

```typescript
useEffect(() => {
  // ...
  setProtocolLockedByUser(false);
  setProviderLockedByUser(false); // 重置提供商锁定状态
  // ...
}, [account, open]);
```

## 行为说明

修复后的行为：

1. **首次打开表单**：`providerLockedByUser = false`
   - 用户输入邮箱地址时，系统会自动识别并切换提供商
   - 例如：输入 `user@qq.com`，自动切换到 QQ 邮箱

2. **用户手动选择提供商**：`providerLockedByUser = true`
   - 用户从下拉列表中选择提供商后，该选择被"锁定"
   - 之后无论输入什么邮箱地址，都不会自动切换提供商
   - 例如：手动选择 QQ 邮箱后，输入 `794382693@` 不会切换到通用邮箱

3. **关闭并重新打开表单**：`providerLockedByUser = false`
   - 状态被重置，恢复自动识别功能

## 测试步骤

1. 打开"添加邮箱账户"对话框
2. 从"邮箱提供商"下拉列表中手动选择"QQ 邮箱"
3. 在"邮箱地址"输入框中输入 `794382693@`
4. 验证提供商仍然是"QQ 邮箱"，没有自动切换到"通用邮箱"
5. 继续输入完整邮箱地址 `794382693@qq.com`
6. 验证提供商仍然是"QQ 邮箱"

## 相关文件

- `frontend/src/components/account/AccountForm.tsx` - 添加账户表单组件
- `frontend/src/hooks/useProviders.ts` - 提供商 Hook

## 类似机制

这个修复方案与现有的 `protocolLockedByUser` 机制保持一致：
- `protocolLockedByUser`：跟踪用户是否手动选择了协议
- `providerLockedByUser`：跟踪用户是否手动选择了提供商

两者都遵循"尊重用户手动选择"的原则。
