import { useState } from 'react';
import { Moon, Sun, Bell, RefreshCw, Save, RotateCcw } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Switch } from '../components/ui/switch';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Label } from '../components/ui/label';
import { Separator } from '../components/ui/separator';
import { useTheme } from '../hooks/useTheme';

export const SettingsPage = () => {
  const { theme, setTheme } = useTheme();
  
  // 同步设置
  const [syncSettings, setSyncSettings] = useState({
    autoSync: true,
    syncInterval: '5', // 分钟
    syncOnStartup: true,
    maxRetries: '3',
  });

  // 通知设置
  const [notificationSettings, setNotificationSettings] = useState({
    emailNotifications: true,
    desktopNotifications: false,
    soundEnabled: true,
    notifyOnNewEmail: true,
    notifyOnSyncError: true,
  });

  // 界面设置
  const [uiSettings, setUiSettings] = useState({
    language: 'zh-CN',
    dateFormat: 'YYYY-MM-DD',
    timeFormat: '24h',
    emailsPerPage: '20',
  });

  const handleSaveSettings = () => {
    // 保存设置到本地存储
    localStorage.setItem('fusionmail_sync_settings', JSON.stringify(syncSettings));
    localStorage.setItem('fusionmail_notification_settings', JSON.stringify(notificationSettings));
    localStorage.setItem('fusionmail_ui_settings', JSON.stringify(uiSettings));
    
    // 这里可以添加成功提示
    console.log('设置已保存');
  };

  const handleResetSettings = () => {
    // 重置为默认设置
    setSyncSettings({
      autoSync: true,
      syncInterval: '5',
      syncOnStartup: true,
      maxRetries: '3',
    });
    
    setNotificationSettings({
      emailNotifications: true,
      desktopNotifications: false,
      soundEnabled: true,
      notifyOnNewEmail: true,
      notifyOnSyncError: true,
    });
    
    setUiSettings({
      language: 'zh-CN',
      dateFormat: 'YYYY-MM-DD',
      timeFormat: '24h',
      emailsPerPage: '20',
    });
  };

  const syncIntervalOptions = [
    { value: '1', label: '1 分钟' },
    { value: '5', label: '5 分钟' },
    { value: '10', label: '10 分钟' },
    { value: '15', label: '15 分钟' },
    { value: '30', label: '30 分钟' },
    { value: '60', label: '1 小时' },
  ];

  const languageOptions = [
    { value: 'zh-CN', label: '简体中文' },
    { value: 'en-US', label: 'English' },
  ];

  const dateFormatOptions = [
    { value: 'YYYY-MM-DD', label: '2024-12-31' },
    { value: 'MM/DD/YYYY', label: '12/31/2024' },
    { value: 'DD/MM/YYYY', label: '31/12/2024' },
  ];

  const timeFormatOptions = [
    { value: '24h', label: '24 小时制' },
    { value: '12h', label: '12 小时制' },
  ];

  const emailsPerPageOptions = [
    { value: '10', label: '10 封' },
    { value: '20', label: '20 封' },
    { value: '50', label: '50 封' },
    { value: '100', label: '100 封' },
  ];

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              系统设置
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-1">
              配置 FusionMail 的各项设置
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={handleResetSettings}>
              <RotateCcw className="h-4 w-4 mr-2" />
              重置
            </Button>
            <Button onClick={handleSaveSettings}>
              <Save className="h-4 w-4 mr-2" />
              保存设置
            </Button>
          </div>
        </div>

        <div className="space-y-6">
          {/* 外观设置 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                {theme === 'dark' ? <Moon className="h-5 w-5" /> : <Sun className="h-5 w-5" />}
                外观设置
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-base">主题模式</Label>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    选择浅色或深色主题
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Sun className="h-4 w-4" />
                  <Switch
                    checked={theme === 'dark'}
                    onCheckedChange={(checked) => setTheme(checked ? 'dark' : 'light')}
                  />
                  <Moon className="h-4 w-4" />
                </div>
              </div>

              <Separator />

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <Label htmlFor="language">界面语言</Label>
                  <Select value={uiSettings.language} onValueChange={(value) => setUiSettings(prev => ({ ...prev, language: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {languageOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="emailsPerPage">每页邮件数</Label>
                  <Select value={uiSettings.emailsPerPage} onValueChange={(value) => setUiSettings(prev => ({ ...prev, emailsPerPage: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {emailsPerPageOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="dateFormat">日期格式</Label>
                  <Select value={uiSettings.dateFormat} onValueChange={(value) => setUiSettings(prev => ({ ...prev, dateFormat: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {dateFormatOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="timeFormat">时间格式</Label>
                  <Select value={uiSettings.timeFormat} onValueChange={(value) => setUiSettings(prev => ({ ...prev, timeFormat: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {timeFormatOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 同步设置 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <RefreshCw className="h-5 w-5" />
                同步设置
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-base">自动同步</Label>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    启用后将定期自动同步邮件
                  </div>
                </div>
                <Switch
                  checked={syncSettings.autoSync}
                  onCheckedChange={(checked) => setSyncSettings(prev => ({ ...prev, autoSync: checked }))}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-base">启动时同步</Label>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    应用启动时自动执行一次同步
                  </div>
                </div>
                <Switch
                  checked={syncSettings.syncOnStartup}
                  onCheckedChange={(checked) => setSyncSettings(prev => ({ ...prev, syncOnStartup: checked }))}
                />
              </div>

              <Separator />

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <Label htmlFor="syncInterval">同步间隔</Label>
                  <Select 
                    value={syncSettings.syncInterval} 
                    onValueChange={(value) => setSyncSettings(prev => ({ ...prev, syncInterval: value }))}
                    disabled={!syncSettings.autoSync}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {syncIntervalOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="maxRetries">最大重试次数</Label>
                  <Select value={syncSettings.maxRetries} onValueChange={(value) => setSyncSettings(prev => ({ ...prev, maxRetries: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">1 次</SelectItem>
                      <SelectItem value="3">3 次</SelectItem>
                      <SelectItem value="5">5 次</SelectItem>
                      <SelectItem value="10">10 次</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 通知设置 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bell className="h-5 w-5" />
                通知设置
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-base">邮件通知</Label>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    接收新邮件时显示通知
                  </div>
                </div>
                <Switch
                  checked={notificationSettings.notifyOnNewEmail}
                  onCheckedChange={(checked) => setNotificationSettings(prev => ({ ...prev, notifyOnNewEmail: checked }))}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-base">桌面通知</Label>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    在系统桌面显示通知
                  </div>
                </div>
                <Switch
                  checked={notificationSettings.desktopNotifications}
                  onCheckedChange={(checked) => setNotificationSettings(prev => ({ ...prev, desktopNotifications: checked }))}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-base">声音提醒</Label>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    通知时播放提示音
                  </div>
                </div>
                <Switch
                  checked={notificationSettings.soundEnabled}
                  onCheckedChange={(checked) => setNotificationSettings(prev => ({ ...prev, soundEnabled: checked }))}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-base">同步错误通知</Label>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    同步失败时显示错误通知
                  </div>
                </div>
                <Switch
                  checked={notificationSettings.notifyOnSyncError}
                  onCheckedChange={(checked) => setNotificationSettings(prev => ({ ...prev, notifyOnSyncError: checked }))}
                />
              </div>
            </CardContent>
          </Card>

          {/* 高级设置 */}
          <Card>
            <CardHeader>
              <CardTitle>高级设置</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <p className="mb-2">
                  <strong>数据存储位置：</strong> 本地浏览器存储
                </p>
                <p className="mb-2">
                  <strong>版本信息：</strong> FusionMail v1.0.0
                </p>
                <p>
                  <strong>最后更新：</strong> 2025-10-30
                </p>
              </div>
              
              <Separator />
              
              <div className="flex items-center gap-4">
                <Button variant="outline" size="sm">
                  清除缓存
                </Button>
                <Button variant="outline" size="sm">
                  导出设置
                </Button>
                <Button variant="outline" size="sm">
                  导入设置
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};