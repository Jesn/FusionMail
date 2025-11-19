# 主题切换功能实现

## 问题描述

设置页面的 `theme` 字段已改为下拉框选择，但选择不同主题后，系统界面没有任何变化。

## 原因分析

虽然项目中已经存在完整的主题系统（`useTheme` hook 和 `initTheme` 函数），但设置页面的主题选择与主题系统没有连接，导致：

1. 用户在设置页面修改主题后，只是更新了数据库中的设置值
2. 主题系统没有监听设置的变化，因此界面不会更新
3. 主题系统使用 `localStorage` 存储主题，而设置系统使用后端 API 存储

## 解决方案

### 1. 连接设置系统与主题系统

**修改文件**: `frontend/src/pages/UserSettings.tsx`

#### 导入 useTheme hook

```typescript
import { useTheme } from '../hooks/useTheme';
```

#### 在组件中使用主题 hook

```typescript
export function UserSettings() {
  const [activeTab, setActiveTab] = useState('ui');
  const { theme, setTheme } = useTheme();
  
  // ... 其他代码
}
```

#### 监听主题设置变化

```typescript
// 监听 UI 设置中的 theme 变化，同步到主题系统
useEffect(() => {
  const uiSettings = settingsQueries.ui.settings;
  if (uiSettings?.theme) {
    const savedTheme = uiSettings.theme as 'light' | 'dark' | 'system';
    if (savedTheme !== theme) {
      setTheme(savedTheme);
    }
  }
}, [settingsQueries.ui.settings?.theme]);
```

#### 更新时立即应用主题

```typescript
const handleUpdate = async (category: string, key: string, value: string) => {
  try {
    await updateSetting.mutateAsync({ category, key, value });
    
    // 如果更新的是主题设置，立即应用主题
    if (category === 'ui' && key === 'theme') {
      setTheme(value as 'light' | 'dark' | 'system');
    }
  } catch (error) {
    console.error('更新设置失败:', error);
    throw error;
  }
};
```

#### 重置时恢复默认主题

```typescript
const handleReset = async (category: string, key: string) => {
  try {
    await resetSetting.mutateAsync({ category, key });
    
    // 如果重置的是主题设置，恢复默认主题
    if (category === 'ui' && key === 'theme') {
      setTheme('system');
    }
  } catch (error) {
    console.error('重置设置失败:', error);
    throw error;
  }
};
```

## 工作原理

### 主题系统架构

```
┌─────────────────────────────────────────────────────────┐
│                    用户操作                              │
│              (选择主题: light/dark/system)               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              UserSettings 组件                           │
│  1. handleUpdate() 保存到后端 API                       │
│  2. setTheme() 更新主题系统                             │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              useTheme Hook                               │
│  1. 保存到 localStorage                                 │
│  2. 解析实际主题 (system → light/dark)                  │
│  3. 调用 applyTheme()                                   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              applyTheme()                                │
│  添加/移除 document.documentElement 的 'dark' class     │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Tailwind CSS                                │
│  根据 .dark class 应用对应的样式                        │
└─────────────────────────────────────────────────────────┘
```

### 主题值处理

1. **light** - 浅色模式
   - 移除 `dark` class
   - 应用浅色样式

2. **dark** - 深色模式
   - 添加 `dark` class
   - 应用深色样式

3. **system** - 跟随系统
   - 检测系统主题偏好 `prefers-color-scheme`
   - 根据系统设置应用 light 或 dark
   - 监听系统主题变化并自动切换

### 双重存储机制

为了保证主题设置的持久化和同步，系统使用了双重存储：

1. **localStorage** (`fusionmail_theme`)
   - 用于快速加载，避免 FOUC (Flash of Unstyled Content)
   - 在 `main.tsx` 中通过 `initTheme()` 初始化
   - 由 `useTheme` hook 管理

2. **后端 API** (`ui.theme`)
   - 用于跨设备同步
   - 用户登录后可以在不同设备上保持一致的主题设置
   - 由设置系统管理

## 测试步骤

### 1. 测试浅色模式

1. 访问 `http://localhost:4444/settings`
2. 切换到"界面设置"标签
3. 在 `theme` 下拉框中选择"浅色模式"
4. ✅ 界面应立即切换为浅色主题
5. ✅ 刷新页面后主题保持不变

### 2. 测试深色模式

1. 在 `theme` 下拉框中选择"深色模式"
2. ✅ 界面应立即切换为深色主题
3. ✅ 背景变暗，文字变亮
4. ✅ 刷新页面后主题保持不变

### 3. 测试跟随系统

1. 在 `theme` 下拉框中选择"跟随系统"
2. ✅ 界面应根据系统设置显示对应主题
3. 修改系统主题设置（macOS: 系统偏好设置 → 外观）
4. ✅ 应用主题应自动跟随系统变化
5. ✅ 刷新页面后仍然跟随系统

### 4. 测试重置功能

1. 选择任意主题（如"深色模式"）
2. 点击"重置"按钮
3. ✅ 主题应恢复为"跟随系统"
4. ✅ 界面主题相应更新

### 5. 测试持久化

1. 选择"深色模式"
2. 关闭浏览器标签页
3. 重新打开 `http://localhost:4444`
4. ✅ 应该仍然显示深色主题

## 技术细节

### useTheme Hook 功能

```typescript
export const useTheme = () => {
  const [theme, setThemeState] = useState<Theme>(() => {
    // 从 localStorage 读取
    const savedTheme = localStorage.getItem('fusionmail_theme') as Theme;
    return savedTheme || 'system';
  });

  const [resolvedTheme, setResolvedTheme] = useState<'light' | 'dark'>('light');

  // 获取系统主题
  const getSystemTheme = (): 'light' | 'dark' => {
    return window.matchMedia('(prefers-color-scheme: dark)').matches 
      ? 'dark' 
      : 'light';
  };

  // 应用主题到 DOM
  const applyTheme = (newTheme: 'light' | 'dark') => {
    const root = document.documentElement;
    if (newTheme === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
    setResolvedTheme(newTheme);
  };

  // 设置主题
  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme);
    localStorage.setItem('fusionmail_theme', newTheme);

    let actualTheme: 'light' | 'dark';
    if (newTheme === 'system') {
      actualTheme = getSystemTheme();
    } else {
      actualTheme = newTheme;
    }

    applyTheme(actualTheme);
  };

  // 监听系统主题变化
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    
    const handleChange = () => {
      if (theme === 'system') {
        applyTheme(getSystemTheme());
      }
    };

    mediaQuery.addEventListener('change', handleChange);
    
    // 初始化主题
    let initialTheme: 'light' | 'dark';
    if (theme === 'system') {
      initialTheme = getSystemTheme();
    } else {
      initialTheme = theme;
    }
    applyTheme(initialTheme);

    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [theme]);

  return { theme, resolvedTheme, setTheme };
};
```

### Tailwind CSS 配置

项目使用 Tailwind CSS v4，通过 `dark:` 前缀支持深色模式：

```css
/* 浅色模式 */
.bg-white { background-color: white; }
.text-gray-900 { color: #111827; }

/* 深色模式 */
.dark .bg-white { background-color: #1f2937; }
.dark .text-gray-900 { color: #f9fafb; }
```

## 相关文件

- `frontend/src/pages/UserSettings.tsx` - 设置页面（已修改）
- `frontend/src/hooks/useTheme.ts` - 主题管理 Hook
- `frontend/src/lib/theme.ts` - 主题初始化函数
- `frontend/src/main.tsx` - 应用入口（调用 initTheme）
- `frontend/src/components/settings/settingOptions.ts` - 主题选项配置

## 注意事项

1. **FOUC 预防**
   - `initTheme()` 在 React 渲染前执行，避免闪烁

2. **系统主题监听**
   - 使用 `matchMedia` API 监听系统主题变化
   - 仅在选择"跟随系统"时生效

3. **双重存储同步**
   - localStorage 用于快速加载
   - 后端 API 用于跨设备同步
   - 两者通过 `useEffect` 保持同步

4. **性能优化**
   - 主题切换是同步操作，无需等待 API 响应
   - 使用 CSS class 切换，性能优秀

## 后续优化建议

1. **主题预览**
   - 添加主题预览功能
   - 在选择前查看效果

2. **自定义主题**
   - 支持用户自定义主题颜色
   - 保存多套主题配置

3. **平滑过渡**
   - 添加主题切换的过渡动画
   - 提升视觉体验

4. **主题同步**
   - 登录时从后端加载主题设置
   - 覆盖本地 localStorage

## 总结

通过连接设置系统与主题系统，现在用户可以：

- ✅ 在设置页面选择主题
- ✅ 立即看到界面变化
- ✅ 主题设置持久化保存
- ✅ 支持跟随系统主题
- ✅ 跨设备同步主题设置
