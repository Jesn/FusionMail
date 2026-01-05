/**
 * 颜色主题设置组件
 * 在设置页面中展示颜色主题选择
 */
import { Check } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useTheme } from '@/hooks/useTheme';
import { cn } from '@/lib/utils';

export function ColorThemeSettings() {
  const { colorTheme, availableThemes, setColorTheme, resolvedTheme } = useTheme();

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">颜色主题</CardTitle>
        <CardDescription>
          选择您喜欢的颜色主题，主题会同时应用于浅色和深色模式
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
          {availableThemes.map((theme) => {
            const isSelected = colorTheme.name === theme.name;
            const colors = resolvedTheme === 'dark' ? theme.dark : theme.light;

            return (
              <button
                key={theme.name}
                onClick={() => setColorTheme(theme.name)}
                className={cn(
                  'relative flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all',
                  'hover:border-primary/50 hover:shadow-md',
                  isSelected
                    ? 'border-primary bg-primary/5'
                    : 'border-border bg-card'
                )}
              >
                {/* 颜色预览 */}
                <div className="flex gap-1">
                  <div
                    className="w-6 h-6 rounded-full border"
                    style={{ backgroundColor: colors.primary }}
                    title="主色"
                  />
                  <div
                    className="w-6 h-6 rounded-full border"
                    style={{ backgroundColor: colors.secondary }}
                    title="次要色"
                  />
                  <div
                    className="w-6 h-6 rounded-full border"
                    style={{ backgroundColor: colors.accent }}
                    title="强调色"
                  />
                </div>

                {/* 主题名称 */}
                <span className="text-sm font-medium">{theme.label}</span>

                {/* 选中标记 */}
                {isSelected && (
                  <div className="absolute top-2 right-2">
                    <Check className="h-4 w-4 text-primary" />
                  </div>
                )}
              </button>
            );
          })}
        </div>

        {/* 预览卡片 */}
        <div className="mt-6 p-4 rounded-lg border bg-card">
          <h4 className="text-sm font-medium mb-3">预览效果</h4>
          <div className="flex flex-wrap gap-2">
            <span className="px-3 py-1 rounded-md bg-primary text-primary-foreground text-sm">
              主要按钮
            </span>
            <span className="px-3 py-1 rounded-md bg-secondary text-secondary-foreground text-sm">
              次要按钮
            </span>
            <span className="px-3 py-1 rounded-md bg-accent text-accent-foreground text-sm">
              强调按钮
            </span>
            <span className="px-3 py-1 rounded-md bg-destructive text-destructive-foreground text-sm">
              危险按钮
            </span>
            <span className="px-3 py-1 rounded-md bg-muted text-muted-foreground text-sm">
              静音文本
            </span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
