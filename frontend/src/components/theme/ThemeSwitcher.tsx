/**
 * 主题切换器组件
 * 提供颜色主题和明暗模式的选择界面
 */
import { Sun, Moon, Monitor, Palette, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useTheme } from './ThemeProvider';
import { cn } from '@/lib/utils';

interface ThemeSwitcherProps {
  // 是否只显示明暗模式切换（简洁模式）
  compact?: boolean;
  // 自定义类名
  className?: string;
}

export function ThemeSwitcher({ compact = false, className }: ThemeSwitcherProps) {
  const { theme, mode, resolvedMode, availableThemes, setTheme, setMode } = useTheme();

  // 简洁模式：只切换明暗
  if (compact) {
    return (
      <Button
        variant="ghost"
        size="icon"
        className={className}
        title={resolvedMode === 'light' ? '切换到深色模式' : '切换到浅色模式'}
        onClick={() => setMode(resolvedMode === 'light' ? 'dark' : 'light')}
      >
        {resolvedMode === 'light' ? (
          <Moon className="h-5 w-5" />
        ) : (
          <Sun className="h-5 w-5" />
        )}
      </Button>
    );
  }

  // 完整模式：颜色主题 + 明暗模式
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className={className} title="主题设置">
          <Palette className="h-5 w-5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        {/* 明暗模式选择 */}
        <DropdownMenuLabel>外观模式</DropdownMenuLabel>
        <DropdownMenuItem onClick={() => setMode('light')}>
          <Sun className="mr-2 h-4 w-4" />
          浅色
          {mode === 'light' && <Check className="ml-auto h-4 w-4" />}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setMode('dark')}>
          <Moon className="mr-2 h-4 w-4" />
          深色
          {mode === 'dark' && <Check className="ml-auto h-4 w-4" />}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setMode('system')}>
          <Monitor className="mr-2 h-4 w-4" />
          跟随系统
          {mode === 'system' && <Check className="ml-auto h-4 w-4" />}
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        {/* 颜色主题选择 */}
        <DropdownMenuLabel>颜色主题</DropdownMenuLabel>
        {availableThemes.map((t) => (
          <DropdownMenuItem key={t.name} onClick={() => setTheme(t.name)}>
            <ThemeColorPreview theme={t} className="mr-2" />
            {t.label}
            {theme.name === t.name && <Check className="ml-auto h-4 w-4" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// 主题颜色预览小圆点
interface ThemeColorPreviewProps {
  theme: { light: { primary: string }; dark: { primary: string } };
  className?: string;
}

function ThemeColorPreview({ theme, className }: ThemeColorPreviewProps) {
  return (
    <div
      className={cn('h-4 w-4 rounded-full border', className)}
      style={{ backgroundColor: theme.light.primary }}
    />
  );
}

// 导出独立的明暗模式切换按钮
export function ModeToggle({ className }: { className?: string }) {
  const { resolvedMode, setMode } = useTheme();

  return (
    <Button
      variant="ghost"
      size="icon"
      className={className}
      title={resolvedMode === 'light' ? '切换到深色模式' : '切换到浅色模式'}
      onClick={() => setMode(resolvedMode === 'light' ? 'dark' : 'light')}
    >
      {resolvedMode === 'light' ? (
        <Moon className="h-5 w-5" />
      ) : (
        <Sun className="h-5 w-5" />
      )}
    </Button>
  );
}
