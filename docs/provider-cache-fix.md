# 邮箱提供商缓存刷新修复

## 问题描述

在"供应商管理"页面新增邮箱提供商后，在"添加邮箱账户"页面的提供商下拉列表中看不到新增的提供商。

## 问题原因

`useProviders` Hook 使用了全局缓存（`cachedProviders`）来存储提供商列表，避免重复请求。但是当在供应商管理页面进行增删改操作后，这个缓存没有被更新，导致其他页面无法看到最新的提供商列表。

## 解决方案

### 1. 修改 `useProviders` Hook

在 `frontend/src/hooks/useProviders.ts` 中添加 `refreshProviders` 方法，用于强制刷新提供商列表并清除缓存：

```typescript
// 强制刷新提供商列表（清除缓存）
const refreshProviders = useCallback(async () => {
  cachedProviders = null; // 清除缓存
  fetchPromise = null;
  await fetchProviders();
}, [fetchProviders]);

return {
  providers,
  isLoading,
  error,
  fetchProviders,
  refreshProviders, // 新增：强制刷新方法
  getProviderByEmail,
  getProviderByName,
  getPresetProviders,
  getGenericProvider,
};
```

### 2. 修改供应商管理页面

在 `frontend/src/pages/ProvidersPage.tsx` 中，在增删改操作成功后调用 `refreshProviders` 刷新全局缓存：

```typescript
// 导入 useProviders Hook
import { useProviders } from '../hooks/useProviders';

export const ProvidersPage = () => {
  // 使用 useProviders Hook 来刷新全局缓存
  const { refreshProviders } = useProviders();
  
  // 创建提供商成功后
  await providerService.create(createForm);
  loadProviders();
  await refreshProviders(); // 刷新全局缓存
  
  // 更新提供商成功后
  await providerService.update(selectedProvider.id, editForm);
  loadProviders();
  await refreshProviders(); // 刷新全局缓存
  
  // 删除提供商成功后
  await providerService.delete(selectedProvider.id);
  loadProviders();
  await refreshProviders(); // 刷新全局缓存
}
```

## 效果

修复后，当在"供应商管理"页面新增、编辑或删除提供商时，全局缓存会被刷新，其他页面（如"添加邮箱账户"页面）的提供商下拉列表会立即显示最新的提供商列表。

## 测试步骤

1. 打开"供应商管理"页面
2. 新增一个邮箱提供商
3. 打开"添加邮箱账户"页面
4. 验证新增的提供商出现在下拉列表中

## 相关文件

- `frontend/src/hooks/useProviders.ts` - 提供商 Hook
- `frontend/src/pages/ProvidersPage.tsx` - 供应商管理页面
- `frontend/src/components/account/AccountForm.tsx` - 添加账户表单
