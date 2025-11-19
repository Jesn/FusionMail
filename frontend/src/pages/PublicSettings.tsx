/**
 * 公开配置页面
 * 未认证用户可查看的公开配置信息
 */

import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Badge } from '../components/ui/badge';
import { Settings, Globe, Loader2, CheckCircle } from 'lucide-react';

// 获取公开配置的API函数
const fetchPublicSettings = async () => {
  const response = await fetch('/api/v1/settings/public');
  if (!response.ok) {
    throw new Error('获取公开配置失败');
  }
  return response.json();
};

export function PublicSettings() {
  const [settings, setSettings] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await fetchPublicSettings();
        setSettings(data.data || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : '加载失败');
      } finally {
        setLoading(false);
      }
    };

    loadSettings();
  }, []);

  // 按分类组织配置
  const settingsByCategory = settings.reduce((acc, setting) => {
    const category = setting.category || 'other';
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(setting);
    return acc;
  }, {} as Record<string, any[]>);

  // 配置分类元数据
  const categoriesMeta: Record<string, { name: string; description: string }> = {
    ui: { name: '界面设置', description: '用户界面相关配置' },
    sync: { name: '同步设置', description: '邮件同步相关配置' },
    notification: { name: '通知设置', description: '通知相关配置' },
  };

  if (loading) {
    return (
      <div className="container mx-auto py-12">
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <span className="ml-2 text-muted-foreground">加载中...</span>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto py-12">
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <p className="text-destructive">{error}</p>
            <button
              onClick={() => window.location.reload()}
              className="mt-4 text-sm text-primary hover:underline"
            >
              点击重试
            </button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">公开信息</h1>
        <p className="text-muted-foreground">
          以下是系统公开的配置信息
        </p>
      </div>

      {/* 状态信息 */}
      <Card className="bg-green-50 border-green-200">
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2 text-green-700">
            <CheckCircle className="h-4 w-4" />
            服务状态正常
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-green-600">
            当前可以正常访问公开配置信息
          </p>
        </CardContent>
      </Card>

      {/* 配置列表 */}
      {Object.keys(settingsByCategory).length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Settings className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">暂无公开配置信息</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-6">
          {Object.entries(settingsByCategory).map(([category, items]) => {
            const meta = categoriesMeta[category] || {
              name: category,
              description: '系统配置',
            };
            const itemsArray = items as any[];

            return (
              <Card key={category}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-lg">{meta.name}</CardTitle>
                      <CardDescription className="mt-1">
                        {meta.description}
                      </CardDescription>
                    </div>
                    <Badge variant="secondary">{itemsArray.length} 项</Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {itemsArray.map((item: any) => (
                      <div
                        key={item.key}
                        className="flex items-start justify-between p-3 border rounded-lg hover:bg-muted/50 transition-colors"
                      >
                        <div className="space-y-1 flex-1">
                          <div className="flex items-center gap-2">
                            <code className="text-sm font-mono">{item.key}</code>
                            <Badge variant="outline" className="text-xs">
                              {item.value_type || 'string'}
                            </Badge>
                          </div>
                          {item.description && (
                            <p className="text-xs text-muted-foreground">
                              {item.description}
                            </p>
                          )}
                        </div>
                        <div className="ml-4">
                          <code className="text-sm bg-muted px-2 py-1 rounded">
                            {item.value || '(未设置)'}
                          </code>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      {/* 底部信息 */}
      <Card className="bg-muted/50">
        <CardContent className="pt-6">
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <Globe className="h-4 w-4" />
              <span>公开配置信息</span>
            </div>
            <span>最后更新: {new Date().toLocaleString()}</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default PublicSettings;
