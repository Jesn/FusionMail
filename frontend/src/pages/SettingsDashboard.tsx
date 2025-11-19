/**
 * 配置仪表盘
 * 提供配置系统的全面概览和快速访问
 */

import { } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { Progress } from '../components/ui/progress';
import {
  Activity,
  Settings,
  Shield,
  Globe,
  Database,
  Layers,
  CheckCircle,
  RefreshCw,
  ArrowRight,
  Loader2,
} from 'lucide-react';

// 配置分类元数据
const CATEGORIES_META = [
  {
    key: 'ui',
    name: '界面设置',
    icon: <Settings className="h-5 w-5" />,
    color: 'bg-blue-100 text-blue-700 border-blue-200',
    userAccess: true,
  },
  {
    key: 'sync',
    name: '同步设置',
    icon: <RefreshCw className="h-5 w-5" />,
    color: 'bg-green-100 text-green-700 border-green-200',
    userAccess: true,
  },
  {
    key: 'notification',
    name: '通知设置',
    icon: <Activity className="h-5 w-5" />,
    color: 'bg-purple-100 text-purple-700 border-purple-200',
    userAccess: true,
  },
  {
    key: 'security',
    name: '安全设置',
    icon: <Shield className="h-5 w-5" />,
    color: 'bg-red-100 text-red-700 border-red-200',
    userAccess: false,
  },
  {
    key: 'api',
    name: 'API设置',
    icon: <Layers className="h-5 w-5" />,
    color: 'bg-orange-100 text-orange-700 border-orange-200',
    userAccess: false,
  },
  {
    key: 'oauth',
    name: 'OAuth设置',
    icon: <Shield className="h-5 w-5" />,
    color: 'bg-yellow-100 text-yellow-700 border-yellow-200',
    userAccess: false,
  },
  {
    key: 'smtp',
    name: 'SMTP设置',
    icon: <Globe className="h-5 w-5" />,
    color: 'bg-indigo-100 text-indigo-700 border-indigo-200',
    userAccess: false,
  },
];

export function SettingsDashboard() {
  const navigate = useNavigate();
  // const { useSettingsByCategory, useGetStats, useWarmUp } = useSettings();

  // 获取所有分类的设置
  const settingsQueries = CATEGORIES_META.reduce((acc, category) => {
    // acc[category.key] = useSettingsByCategory(category.key);
    acc[category.key] = { data: [], isLoading: false }; // 临时占位
    return acc;
  }, {} as Record<string, any>);

  // const statsQuery = { data: null, isLoading: false }; // 临时占位
  // const warmUpMutation = { mutate: () => {}, isPending: false }; // 临时占位
  // const warmUpMutation = useWarmUp();
  const statsQuery = { data: null, isLoading: false }; // 临时占位
  const warmUpMutation = { mutate: () => {}, isPending: false }; // 临时占位

  // 计算统计数据
  const totalSettings = Object.values(settingsQueries).reduce(
    (sum, query) => sum + (query.data?.length || 0),
    0
  );

  const sensitiveSettings = Object.values(settingsQueries).reduce(
    (sum, query) =>
      sum +
      (query.data?.filter((s: any) => s.is_sensitive).length || 0),
    0
  );

  const publicSettings = Object.values(settingsQueries).reduce(
    (sum, query) =>
      sum +
      (query.data?.filter((s: any) => s.is_public).length || 0),
    0
  );

  const userAccessibleSettings = Object.values(settingsQueries).reduce(
    (sum, query, index) => {
      const isUserAccessible = CATEGORIES_META[index]?.userAccess;
      return sum + (isUserAccessible ? (query.data?.length || 0) : 0);
    },
    0
  );

  const handleWarmUp = async () => {
    try {
      await warmUpMutation.mutate();
      // statsQuery 是临时占位，不需要 refetch
    } catch (error) {
      console.error('缓存预热失败:', error);
    }
  };

  const isLoading = Object.values(settingsQueries).some((query) => query.isLoading) || statsQuery.isLoading;

  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">配置仪表盘</h1>
          <p className="text-muted-foreground">
            系统配置管理和监控概览
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => navigate('/settings')}>
            <Settings className="h-4 w-4 mr-2" />
            用户设置
          </Button>
          <Button variant="outline" onClick={() => navigate('/admin/settings')}>
            <Shield className="h-4 w-4 mr-2" />
            管理员设置
          </Button>
          <Button
            onClick={handleWarmUp}
            disabled={warmUpMutation.isPending}
          >
            {warmUpMutation.isPending ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4 mr-2" />
            )}
            预热缓存
          </Button>
        </div>
      </div>

      {/* 加载状态 */}
      {isLoading && (
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <span className="ml-2 text-muted-foreground">加载仪表盘数据...</span>
          </CardContent>
        </Card>
      )}

      {/* 统计卡片 */}
      {!isLoading && (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {/* 总配置数 */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">总配置数</CardTitle>
                <Settings className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{totalSettings}</div>
                <p className="text-xs text-muted-foreground">
                  分布在 {CATEGORIES_META.length} 个分类
                </p>
              </CardContent>
            </Card>

            {/* 敏感配置 */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">敏感配置</CardTitle>
                <Shield className="h-4 w-4 text-red-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-red-500">{sensitiveSettings}</div>
                <p className="text-xs text-muted-foreground">
                  {((sensitiveSettings / totalSettings) * 100 || 0).toFixed(1)}% 的配置
                </p>
              </CardContent>
            </Card>

            {/* 公开配置 */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">公开配置</CardTitle>
                <Globe className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-500">{publicSettings}</div>
                <p className="text-xs text-muted-foreground">
                  {((publicSettings / totalSettings) * 100 || 0).toFixed(1)}% 的配置
                </p>
              </CardContent>
            </Card>

            {/* 用户可访问 */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">用户可访问</CardTitle>
                <CheckCircle className="h-4 w-4 text-blue-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-blue-500">{userAccessibleSettings}</div>
                <p className="text-xs text-muted-foreground">
                  {((userAccessibleSettings / totalSettings) * 100 || 0).toFixed(1)}% 的配置
                </p>
              </CardContent>
            </Card>
          </div>

          {/* 缓存统计 */}
          {statsQuery.data && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm flex items-center gap-2">
                  <Database className="h-4 w-4" />
                  缓存性能
                </CardTitle>
                <CardDescription>
                  配置缓存系统的运行状态
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-6">
                  {/* 本地缓存 */}
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">本地缓存命中率</span>
                      <span className="text-muted-foreground">
                        {((statsQuery.data as any).localCache?.hitRate * 100 || 0).toFixed(1)}%
                      </span>
                    </div>
                    <Progress
                      value={(statsQuery.data as any).localCache?.hitRate * 100 || 0}
                      className="h-2"
                    />
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span>命中: {(statsQuery.data as any).localCache?.hits || 0}</span>
                      <span>未命中: {(statsQuery.data as any).localCache?.misses || 0}</span>
                      <span>大小: {(statsQuery.data as any).localCache?.size || 0}</span>
                    </div>
                  </div>

                  {/* Redis缓存 */}
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">Redis缓存命中率</span>
                      <span className="text-muted-foreground">
                        {((statsQuery.data as any).redisCache?.hitRate * 100 || 0).toFixed(1)}%
                      </span>
                    </div>
                    <Progress
                      value={(statsQuery.data as any).redisCache?.hitRate * 100 || 0}
                      className="h-2"
                    />
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span>命中: {(statsQuery.data as any).redisCache?.hits || 0}</span>
                      <span>未命中: {(statsQuery.data as any).redisCache?.misses || 0}</span>
                      <span>大小: {(statsQuery.data as any).redisCache?.size || 0}</span>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* 配置分类 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">配置分类概览</CardTitle>
              <CardDescription>
                各类别的配置数量和状态
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {CATEGORIES_META.map((category) => {
                  const query = settingsQueries[category.key];
                  const count = query.data?.length || 0;
                  const sensitiveCount = query.data?.filter((s: any) => s.is_sensitive).length || 0;
                  const publicCount = query.data?.filter((s: any) => s.is_public).length || 0;

                  return (
                    <div
                      key={category.key}
                      className={`p-4 rounded-lg border-2 ${category.color} cursor-pointer hover:opacity-80 transition-opacity`}
                      onClick={() => navigate(category.userAccess ? '/settings' : '/admin/settings')}
                    >
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center gap-2">
                          {category.icon}
                          <span className="font-semibold">{category.name}</span>
                        </div>
                        <ArrowRight className="h-4 w-4" />
                      </div>

                      <div className="space-y-2 text-sm">
                        <div className="flex items-center justify-between">
                          <span>总计</span>
                          <span className="font-bold">{count}</span>
                        </div>

                        {sensitiveCount > 0 && (
                          <div className="flex items-center justify-between text-red-600">
                            <span>敏感</span>
                            <span className="font-bold">{sensitiveCount}</span>
                          </div>
                        )}

                        {publicCount > 0 && (
                          <div className="flex items-center justify-between text-green-600">
                            <span>公开</span>
                            <span className="font-bold">{publicCount}</span>
                          </div>
                        )}

                        <div className="flex items-center gap-1 text-xs text-muted-foreground">
                          {category.userAccess ? (
                            <>
                              <CheckCircle className="h-3 w-3" />
                              <span>用户可访问</span>
                            </>
                          ) : (
                            <>
                              <Shield className="h-3 w-3" />
                              <span>仅管理员</span>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>

          {/* 快速操作 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">快速操作</CardTitle>
              <CardDescription>
                常用的配置管理操作
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-2 md:grid-cols-3">
                <Button variant="outline" onClick={() => navigate('/settings')}>
                  <Settings className="h-4 w-4 mr-2" />
                  用户设置
                </Button>
                <Button variant="outline" onClick={() => navigate('/admin/settings')}>
                  <Shield className="h-4 w-4 mr-2" />
                  管理员设置
                </Button>
                <Button variant="outline" onClick={() => navigate('/public-settings')}>
                  <Globe className="h-4 w-4 mr-2" />
                  公开配置
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* 系统状态 */}
          <Card className="bg-green-50 border-green-200">
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2 text-green-700">
                <CheckCircle className="h-4 w-4" />
                系统状态
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-2 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-green-600">配置服务</span>
                  <span className="text-green-700 font-medium">运行正常</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-green-600">缓存系统</span>
                  <span className="text-green-700 font-medium">运行正常</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-green-600">加密服务</span>
                  <span className="text-green-700 font-medium">运行正常</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

export default SettingsDashboard;
