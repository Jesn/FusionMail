import { useState, useEffect, useCallback, useMemo } from 'react';
import { ShieldAlert, Save, RotateCcw, Info } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/card';
import { Switch } from '../components/ui/switch';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Label } from '../components/ui/label';
import { Separator } from '../components/ui/separator';
import { Input } from '../components/ui/input';
import { Badge } from '../components/ui/badge';
import { Alert, AlertDescription } from '../components/ui/alert';
import { toast } from 'sonner';
import api from '../services/api';
import { useAuthStore } from '../stores/authStore';

// 垃圾邮件设置接口
interface SpamSettings {
  spam_detection_enabled: boolean;
  user_spam_detection_enabled: boolean;
  spam_threshold: number;
  auto_cleanup_days: number;
  bayesian_enabled: boolean;
  rbl_enabled: boolean;
  surbl_enabled: boolean;
}

// 默认设置
const DEFAULT_SETTINGS: SpamSettings = {
  spam_detection_enabled: true,
  user_spam_detection_enabled: true,
  spam_threshold: 60,
  auto_cleanup_days: 30,
  bayesian_enabled: true,
  rbl_enabled: true,
  surbl_enabled: true,
};

export const SpamSettingsPage = () => {
  const [settings, setSettings] = useState<SpamSettings>(DEFAULT_SETTINGS);
  const [originalSettings, setOriginalSettings] = useState<SpamSettings>(DEFAULT_SETTINGS);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  
  // 从认证状态获取用户角色
  const user = useAuthStore(state => state.user);
  const isAdmin = useMemo(() => user?.role === 'admin', [user]);

  // 加载设置
  const loadSettings = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await api.get('/settings/spam');
      const data = response.data.data || DEFAULT_SETTINGS;
      setSettings(data);
      setOriginalSettings(data);
    } catch (error) {
      console.error('Failed to load spam settings:', error);
      // 使用默认设置
      setSettings(DEFAULT_SETTINGS);
      setOriginalSettings(DEFAULT_SETTINGS);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // 初始加载
  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  // 保存设置
  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.put('/settings/spam', settings);
      setOriginalSettings(settings);
      toast.success('设置保存成功');
    } catch (error) {
      console.error('Failed to save spam settings:', error);
      toast.error('保存设置失败');
    } finally {
      setIsSaving(false);
    }
  };

  // 重置设置
  const handleReset = () => {
    setSettings(originalSettings);
    toast.info('设置已重置');
  };

  // 检查是否有更改
  const hasChanges = JSON.stringify(settings) !== JSON.stringify(originalSettings);

  // 自动清理天数选项
  const cleanupDaysOptions = [
    { value: 7, label: '7 天' },
    { value: 14, label: '14 天' },
    { value: 30, label: '30 天' },
    { value: 60, label: '60 天' },
    { value: 90, label: '90 天' },
    { value: 0, label: '永不清理' },
  ];

  // 系统级别开关是否禁用
  const isSystemDisabled = !settings.spam_detection_enabled;

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="max-w-4xl mx-auto">
        {/* 页面标题 */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
              <ShieldAlert className="h-6 w-6 text-orange-500" />
              垃圾邮件设置
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-1">
              配置垃圾邮件检测和自动清理规则
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={handleReset}
              disabled={!hasChanges || isSaving}
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              重置
            </Button>
            <Button
              onClick={handleSave}
              disabled={!hasChanges || isSaving}
            >
              <Save className="h-4 w-4 mr-2" />
              {isSaving ? '保存中...' : '保存设置'}
            </Button>
          </div>
        </div>


        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        ) : (
          <div className="space-y-6">
            {/* 系统级别开关（仅管理员可见） */}
            {isAdmin && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    系统级别设置
                    <Badge variant="outline" className="text-xs">管理员</Badge>
                  </CardTitle>
                  <CardDescription>
                    这些设置影响所有用户的垃圾邮件检测功能
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="flex items-center justify-between">
                    <div className="space-y-0.5">
                      <Label className="text-base">启用垃圾邮件检测</Label>
                      <div className="text-sm text-muted-foreground">
                        关闭后，系统将不再检测垃圾邮件
                      </div>
                    </div>
                    <Switch
                      checked={settings.spam_detection_enabled}
                      onCheckedChange={(checked) =>
                        setSettings({ ...settings, spam_detection_enabled: checked })
                      }
                    />
                  </div>

                  {!settings.spam_detection_enabled && (
                    <Alert>
                      <Info className="h-4 w-4" />
                      <AlertDescription>
                        垃圾邮件检测已关闭，所有用户的垃圾邮件检测功能将被禁用。
                      </AlertDescription>
                    </Alert>
                  )}
                </CardContent>
              </Card>
            )}

            {/* 用户级别设置 */}
            <Card className={isSystemDisabled ? 'opacity-50' : ''}>
              <CardHeader>
                <CardTitle>用户设置</CardTitle>
                <CardDescription>
                  个人垃圾邮件检测偏好设置
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label className="text-base">启用我的垃圾邮件检测</Label>
                    <div className="text-sm text-muted-foreground">
                      关闭后，您的邮件将不会被检测为垃圾邮件
                    </div>
                  </div>
                  <Switch
                    checked={settings.user_spam_detection_enabled}
                    onCheckedChange={(checked) =>
                      setSettings({ ...settings, user_spam_detection_enabled: checked })
                    }
                    disabled={isSystemDisabled}
                  />
                </div>

                {isSystemDisabled && (
                  <Alert variant="destructive">
                    <Info className="h-4 w-4" />
                    <AlertDescription>
                      系统级别的垃圾邮件检测已关闭，此设置暂时无效。
                    </AlertDescription>
                  </Alert>
                )}
              </CardContent>
            </Card>

            {/* 检测阈值设置 */}
            <Card className={isSystemDisabled ? 'opacity-50' : ''}>
              <CardHeader>
                <CardTitle>检测阈值</CardTitle>
                <CardDescription>
                  设置垃圾邮件评分阈值，评分达到或超过此值的邮件将被标记为垃圾邮件
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <Label className="text-base">垃圾邮件阈值</Label>
                    <div className="flex items-center gap-2">
                      <Input
                        type="number"
                        min={30}
                        max={90}
                        step={5}
                        value={settings.spam_threshold}
                        onChange={(e) =>
                          setSettings({ ...settings, spam_threshold: parseInt(e.target.value) || 60 })
                        }
                        disabled={isSystemDisabled}
                        className="w-20 text-center"
                      />
                      <span className="text-muted-foreground">分</span>
                    </div>
                  </div>
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>30 (宽松)</span>
                    <span>60 (默认)</span>
                    <span>90 (严格)</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    较低的阈值会标记更多邮件为垃圾邮件，较高的阈值则更宽松。
                    建议使用默认值 60 分。有效范围：30-90 分。
                  </p>
                </div>
              </CardContent>
            </Card>

            {/* 检测功能开关 */}
            <Card className={isSystemDisabled ? 'opacity-50' : ''}>
              <CardHeader>
                <CardTitle>检测功能</CardTitle>
                <CardDescription>
                  启用或禁用特定的垃圾邮件检测功能
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label className="text-base">贝叶斯分类器</Label>
                    <div className="text-sm text-muted-foreground">
                      基于机器学习的垃圾邮件检测，会根据您的反馈不断学习
                    </div>
                  </div>
                  <Switch
                    checked={settings.bayesian_enabled}
                    onCheckedChange={(checked) =>
                      setSettings({ ...settings, bayesian_enabled: checked })
                    }
                    disabled={isSystemDisabled}
                  />
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label className="text-base">RBL 黑名单检查</Label>
                    <div className="text-sm text-muted-foreground">
                      检查发件人 IP 是否在实时黑名单中
                    </div>
                  </div>
                  <Switch
                    checked={settings.rbl_enabled}
                    onCheckedChange={(checked) =>
                      setSettings({ ...settings, rbl_enabled: checked })
                    }
                    disabled={isSystemDisabled}
                  />
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label className="text-base">SURBL URL 检查</Label>
                    <div className="text-sm text-muted-foreground">
                      检查邮件中的链接是否在 URL 黑名单中
                    </div>
                  </div>
                  <Switch
                    checked={settings.surbl_enabled}
                    onCheckedChange={(checked) =>
                      setSettings({ ...settings, surbl_enabled: checked })
                    }
                    disabled={isSystemDisabled}
                  />
                </div>
              </CardContent>
            </Card>

            {/* 自动清理设置 */}
            <Card className={isSystemDisabled ? 'opacity-50' : ''}>
              <CardHeader>
                <CardTitle>自动清理</CardTitle>
                <CardDescription>
                  设置垃圾邮件的自动清理规则
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-2">
                  <Label htmlFor="cleanup-days">自动清理天数</Label>
                  <Select
                    value={settings.auto_cleanup_days.toString()}
                    onValueChange={(value) =>
                      setSettings({ ...settings, auto_cleanup_days: parseInt(value) })
                    }
                    disabled={isSystemDisabled}
                  >
                    <SelectTrigger className="w-48">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {cleanupDaysOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value.toString()}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-sm text-muted-foreground">
                    超过指定天数的垃圾邮件将被自动永久删除。
                    选择"永不清理"将保留所有垃圾邮件。
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
};

export default SpamSettingsPage;
