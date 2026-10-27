// WebAPI 账户编辑对话框
import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Switch } from '../ui/switch';
import { Badge } from '../ui/badge';
import { Alert, AlertDescription } from '../ui/alert';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
// RadioGroup 已移除，Webhook 模式使用独立的配置界面
import {
  Loader2,
  ChevronDown,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Globe,
  Shield,
} from 'lucide-react';
import { GroupSelector } from '../group';
import { CloudflareTempEmailAuthFields } from './CloudflareTempEmailAuthFields';
import type { CloudflareTempEmailAuthData } from './CloudflareTempEmailAuthFields';
import { CloudMailAuthFields } from './CloudMailAuthFields';
import type { CloudMailAuthData } from './CloudMailAuthFields';
import { CustomWebAPIAuthFields } from './CustomWebAPIAuthFields';
import type { CustomWebAPIAuthData } from './CustomWebAPIAuthFields';
import { ChildAccountsSection } from './ChildAccountsSection';
import type { ChildAccountInfo } from './ChildAccountsSection';
import { Account } from '../../types';
import { webapiService } from '../../services/webapiService';
import { accountService } from '../../services/accountService';
import { toast } from 'sonner';

interface WebAPIAccountEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  account: Account | null;
  onSuccess?: () => void;
}

// WebAPI 服务类型
type WebAPIServiceType = 'cloudflare_temp_email' | 'cloud_mail' | 'custom';

/**
 * WebAPI 账户编辑对话框
 * 支持编辑 Cloudflare Temp Email、Cloud Mail、Custom 三种类型的 WebAPI 账户
 */
export const WebAPIAccountEditDialog: React.FC<WebAPIAccountEditDialogProps> = ({
  open,
  onOpenChange,
  account,
  onSuccess,
}) => {
  // 状态
  const [isLoading, setIsLoading] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  
  // 账户配置
  const [serviceType, setServiceType] = useState<WebAPIServiceType | null>(null);
  const [authData, setAuthData] = useState<any>({});
  const [originalAuthData, setOriginalAuthData] = useState<any>({});
  
  // 显示名称（可编辑）
  const [displayName, setDisplayName] = useState('');
  const [originalDisplayName, setOriginalDisplayName] = useState('');
  
  // 同步设置
  const [syncEnabled, setSyncEnabled] = useState(true);
  const [syncInterval, setSyncInterval] = useState(2);
  const [groupId, setGroupId] = useState<number | null>(null);

  // 子邮箱列表
  const [childAccounts, setChildAccounts] = useState<ChildAccountInfo[]>([]);
  const [isLoadingChildren, setIsLoadingChildren] = useState(false);
  const [childrenExpanded, setChildrenExpanded] = useState(false);

  // 加载账户配置
  useEffect(() => {
    if (open && account?.uid) {
      loadAccountConfig();
    }
  }, [open, account?.uid]);

  const loadAccountConfig = async () => {
    if (!account?.uid) return;
    
    setIsLoading(true);
    try {
      // 从账户获取基本信息
      setSyncEnabled(account.sync_enabled);
      setSyncInterval(account.sync_interval);
      setGroupId(account.group_id || null);
      setDisplayName(account.email || '');
      setOriginalDisplayName(account.email || '');
      
      // 获取 WebAPI 配置
      const config = await webapiService.getAccountConfig(account.uid);
      if (config) {
        setServiceType(config.service_type as WebAPIServiceType);
        setAuthData(config.auth_data || {});
        setOriginalAuthData(config.auth_data || {});
      }

      // 加载子邮箱列表
      loadChildAccounts();
    } catch (error) {
      console.error('加载 WebAPI 配置失败:', error);
      toast.error('加载配置失败');
    } finally {
      setIsLoading(false);
    }
  };

  // 加载子邮箱列表预览（分页首页，完整列表见「查看子邮箱」对话框）
  const loadChildAccounts = async () => {
    if (!account?.uid) return;
    
    setIsLoadingChildren(true);
    try {
      const result = await webapiService.getChildAccounts(account.uid, {
        include: 'all',
        page: 1,
        page_size: 10,
      });
      setChildAccounts(result.items);
    } catch (error) {
      console.error('加载子邮箱列表失败:', error);
      // 不显示错误提示，因为可能没有子邮箱
    } finally {
      setIsLoadingChildren(false);
    }
  };

  // 更新认证字段
  const updateAuthField = (field: string, value: any) => {
    setAuthData((prev: any) => ({ ...prev, [field]: value }));
  };

  // 测试连接
  const handleTestConnection = async () => {
    if (!account?.uid) return;
    
    setIsTesting(true);
    setTestResult(null);
    
    try {
      // 编辑模式下使用 testConnectionByUID，使用后端存储的原始凭证
      const result = await webapiService.testConnectionByUID(account.uid);
      setTestResult({
        success: result.success,
        message: result.message || (result.success ? '连接成功' : '连接失败'),
      });
      if (result.success) {
        toast.success('连接测试成功');
      } else {
        toast.error(result.message || '连接测试失败');
      }
    } catch (error: any) {
      const msg = error?.response?.data?.message || error?.message || '连接测试失败';
      setTestResult({ success: false, message: msg });
      toast.error(msg);
    } finally {
      setIsTesting(false);
    }
  };

  // 提交保存
  const handleSubmit = async () => {
    if (!account?.uid) return;
    
    setIsSubmitting(true);
    try {
      // 构建更新数据
      const updateData: any = {
        sync_enabled: syncEnabled,
        sync_interval: syncInterval,
        group_id: groupId,
      };
      
      // 如果显示名称有变化，也更新
      if (displayName !== originalDisplayName && displayName.trim()) {
        updateData.email = displayName.trim();
      }
      
      // 更新账户设置
      await accountService.update(account.uid, updateData);
      
      // 检查认证数据是否有变化
      const authDataChanged = JSON.stringify(authData) !== JSON.stringify(originalAuthData);
      if (authDataChanged && serviceType) {
        await webapiService.updateAccountConfig(account.uid, {
          service_type: serviceType,
          auth_data: JSON.stringify(authData),
        });
      }
      
      toast.success('保存成功');
      onOpenChange(false);
      onSuccess?.();
    } catch (error: any) {
      const msg = error?.response?.data?.message || error?.message || '保存失败';
      toast.error(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  // 获取服务类型显示名称
  const getServiceTypeName = (type: WebAPIServiceType | null) => {
    switch (type) {
      case 'cloudflare_temp_email': return 'Cloudflare Temp Email';
      case 'cloud_mail': return 'Cloud Mail';
      case 'custom': return '自定义 Web API';
      default: return '未知';
    }
  };

  // 根据服务类型渲染配置
  const renderAuthConfig = () => {
    switch (serviceType) {
      case 'cloudflare_temp_email':
        return (
          <CloudflareTempEmailAuthFields
            data={authData as CloudflareTempEmailAuthData}
            onChange={updateAuthField}
          />
        );
      case 'cloud_mail':
        return (
          <CloudMailAuthFields
            data={authData as CloudMailAuthData}
            onChange={updateAuthField}
          />
        );
      case 'custom':
        return (
          <CustomWebAPIAuthFields
            data={authData as CustomWebAPIAuthData}
            onChange={updateAuthField}
          />
        );
      default:
        return (
          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>无法识别的服务类型</AlertDescription>
          </Alert>
        );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[500px] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>编辑 WebAPI 账户</DialogTitle>
          <DialogDescription>
            修改 WebAPI 账户的认证信息和同步设置
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4 space-y-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">加载中...</span>
            </div>
          ) : (
            <>
              {/* 账户信息概览 */}
              <div className="p-4 rounded-lg bg-muted/30 border">
                <div className="flex items-center gap-3 mb-3">
                  <div className="p-2 rounded-lg bg-primary/10">
                    <Globe className="h-5 w-5 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Badge variant="secondary" className="text-xs">
                        {getServiceTypeName(serviceType)}
                      </Badge>
                      <span>·</span>
                      <span>WebAPI</span>
                    </div>
                  </div>
                </div>
                {/* 显示名称编辑 */}
                <div className="space-y-2">
                  <Label htmlFor="display_name">显示名称</Label>
                  <Input
                    id="display_name"
                    placeholder="输入账户显示名称"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    此名称将显示在账户列表和邮件列表中
                  </p>
                </div>
              </div>

              {/* 认证配置 */}
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm font-medium">
                  <Shield className="h-4 w-4" />
                  认证配置
                </div>
                {renderAuthConfig()}
              </div>

              {/* 测试连接 - Webhook 模式不需要 */}
              {!((authData as CloudflareTempEmailAuthData)?.sync_mode === 'webhook') && (
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleTestConnection}
                  disabled={isTesting}
                >
                  {isTesting ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      测试中...
                    </>
                  ) : (
                    '测试连接'
                  )}
                </Button>
                {testResult && (
                  <span className={`text-sm flex items-center gap-1 ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
                    {testResult.success ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : (
                      <XCircle className="h-4 w-4" />
                    )}
                    {testResult.message}
                  </span>
                )}
              </div>
              )}

              {/* 分组设置 */}
              <div className="space-y-2">
                <Label>分组</Label>
                <GroupSelector
                  value={groupId}
                  onChange={setGroupId}
                />
              </div>

              {/* 同步设置 - Webhook 模式不需要 */}
              {!((authData as CloudflareTempEmailAuthData)?.sync_mode === 'webhook') && (
              <Collapsible defaultOpen>
                <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
                  <ChevronDown className="h-4 w-4 transition-transform ui-closed:rotate-[-90deg]" />
                  同步设置
                </CollapsibleTrigger>
                <CollapsibleContent className="space-y-4 pt-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="sync_enabled">启用自动同步</Label>
                    <Switch
                      id="sync_enabled"
                      checked={syncEnabled}
                      onCheckedChange={setSyncEnabled}
                    />
                  </div>
                  {syncEnabled && (
                    <div className="space-y-2">
                      <Label>同步频率（分钟）</Label>
                      <Select
                        value={String(syncInterval)}
                        onValueChange={(v) => setSyncInterval(Number(v))}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {[1, 2, 5, 10, 15, 30, 60].map((m) => (
                            <SelectItem key={m} value={String(m)}>
                              {m} 分钟
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}
                </CollapsibleContent>
              </Collapsible>
              )}

              {/* 子邮箱列表 */}
              {(childAccounts.length > 0 || isLoadingChildren) && (
                <ChildAccountsSection
                  childAccounts={childAccounts}
                  isLoadingChildren={isLoadingChildren}
                  expanded={childrenExpanded}
                  onExpandedChange={setChildrenExpanded}
                />
              )}
            </>
          )}
        </div>

        <DialogFooter className="border-t pt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting || isLoading}>
            {isSubmitting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                保存中...
              </>
            ) : (
              '保存'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default WebAPIAccountEditDialog;
