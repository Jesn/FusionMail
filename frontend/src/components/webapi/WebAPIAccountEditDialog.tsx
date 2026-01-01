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
import { 
  Loader2, 
  ChevronDown, 
  ChevronRight,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Globe,
  Key,
  Shield,
  Mail,
  Users,
} from 'lucide-react';
import { GroupSelector } from '../group';
import { Account } from '../../types';
import { webapiService } from '../../services/webapiService';
import { accountService } from '../../services/accountService';
import toast from 'react-hot-toast';

interface WebAPIAccountEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  account: Account | null;
  onSuccess?: () => void;
}

// WebAPI 服务类型
type WebAPIServiceType = 'cloudflare_temp_email' | 'cloud_mail' | 'custom';

// 认证数据接口
interface CloudflareTempEmailAuthData {
  base_url: string;
  access_mode: 'single' | 'admin';
  jwt_token?: string;         // JWT Token（直接登录方式，永不过期）
  user_token?: string;        // User Token（第三方授权登录，30天过期，支持自动刷新）
  email?: string;
  admin_password?: string;
  domains?: string;  // 过滤域名列表（逗号分隔）
}

interface CloudMailAuthData {
  base_url: string;
  jwt_token: string;
  accounts?: Array<{ account_id?: number; email: string; name?: string }>;
}

interface CustomWebAPIAuthData {
  service_name: string;
  base_url: string;
  endpoint: string;
  auth_type: string;
  auth_token?: string;
  api_key?: string;
  username?: string;
  password?: string;
  data_path: string;
  target_email?: string;
}

// 子邮箱账户信息
interface ChildAccountInfo {
  uid: string;
  email: string;
  status: string;
  total_emails: number;
  unread_count: number;
  last_sync_at: string | null;
  created_at: string;
}

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

  // 加载子邮箱列表
  const loadChildAccounts = async () => {
    if (!account?.uid) return;
    
    setIsLoadingChildren(true);
    try {
      const children = await webapiService.getChildAccounts(account.uid);
      setChildAccounts(children);
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

  // 渲染 Cloudflare Temp Email 配置
  const renderCloudflareTempEmailConfig = () => {
    const data = authData as CloudflareTempEmailAuthData;
    return (
      <div className="space-y-4">
        {/* 基本信息 */}
        <div className="space-y-3">
          {/* API 地址（可编辑） */}
          <div className="space-y-2">
            <Label htmlFor="cf_base_url">API 地址</Label>
            <Input
              id="cf_base_url"
              placeholder="https://api.example.com"
              value={data.base_url || ''}
              onChange={(e) => updateAuthField('base_url', e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Cloudflare Temp Email 的 API 地址（注意：不是前端网站地址）
            </p>
          </div>
          
          {/* 只读信息 */}
          <div className="p-3 rounded-lg bg-muted/30 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">访问模式</span>
              <Badge variant="outline">
                {data.access_mode === 'admin' ? 'Admin 模式' : 'Single 模式'}
              </Badge>
            </div>
            {data.access_mode === 'single' && data.email && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">目标邮箱</span>
                <span>{data.email}</span>
              </div>
            )}
            {data.access_mode === 'single' && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">认证方式</span>
                <Badge variant="outline">
                  {data.user_token ? 'User Token（自动刷新）' : 'JWT Token（永不过期）'}
                </Badge>
              </div>
            )}
            {data.access_mode === 'admin' && data.domains && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">过滤域名</span>
                <span>{data.domains}</span>
              </div>
            )}
          </div>
        </div>

        {/* 可编辑的认证信息 */}
        <Collapsible>
          <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
            <Key className="h-4 w-4" />
            更新认证信息
            <ChevronRight className="h-4 w-4 ml-auto transition-transform ui-open:rotate-90" />
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-3 pt-2">
            {data.access_mode === 'single' ? (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="jwt_token">JWT Token（永不过期）</Label>
                  <Input
                    id="jwt_token"
                    type="password"
                    placeholder="留空则不修改"
                    value={data.jwt_token || ''}
                    onChange={(e) => updateAuthField('jwt_token', e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    直接登录获取的 JWT Token，永不过期
                  </p>
                </div>
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-background px-2 text-muted-foreground">或</span>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="user_token">User Token（30天过期，自动刷新）</Label>
                  <Input
                    id="user_token"
                    type="password"
                    placeholder="留空则不修改"
                    value={data.user_token || ''}
                    onChange={(e) => updateAuthField('user_token', e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    第三方授权登录获取的 Token，系统会在过期前 7 天自动刷新
                  </p>
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="admin_password">管理员密码</Label>
                <Input
                  id="admin_password"
                  type="password"
                  placeholder="留空则不修改"
                  value={data.admin_password || ''}
                  onChange={(e) => updateAuthField('admin_password', e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  输入新的管理员密码以更新认证，留空保持不变
                </p>
              </div>
            )}
          </CollapsibleContent>
        </Collapsible>
      </div>
    );
  };

  // 渲染 Cloud Mail 配置
  const renderCloudMailConfig = () => {
    const data = authData as CloudMailAuthData;
    return (
      <div className="space-y-4">
        {/* API 地址（可编辑） */}
        <div className="space-y-2">
          <Label htmlFor="cm_base_url">API 地址</Label>
          <Input
            id="cm_base_url"
            placeholder="https://api.example.com"
            value={data.base_url || ''}
            onChange={(e) => updateAuthField('base_url', e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Cloud Mail 的 API 地址
          </p>
        </div>
        
        {/* 只读信息 */}
        {data.accounts && data.accounts.length > 0 && (
          <div className="p-3 rounded-lg bg-muted/30 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">关联账户</span>
              <span>{data.accounts.length} 个</span>
            </div>
          </div>
        )}

        {/* 可编辑的认证信息 */}
        <Collapsible>
          <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
            <Key className="h-4 w-4" />
            更新认证信息
            <ChevronRight className="h-4 w-4 ml-auto transition-transform ui-open:rotate-90" />
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-3 pt-2">
            <div className="space-y-2">
              <Label htmlFor="jwt_token">JWT Token</Label>
              <Input
                id="jwt_token"
                type="password"
                placeholder="留空则不修改"
                value={data.jwt_token || ''}
                onChange={(e) => updateAuthField('jwt_token', e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                从浏览器开发者工具获取新的 JWT Token
              </p>
            </div>
          </CollapsibleContent>
        </Collapsible>
      </div>
    );
  };

  // 渲染自定义 WebAPI 配置
  const renderCustomConfig = () => {
    const data = authData as CustomWebAPIAuthData;
    return (
      <div className="space-y-4">
        {/* 可编辑的基本信息 */}
        <div className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="custom_service_name">服务名称</Label>
            <Input
              id="custom_service_name"
              placeholder="自定义服务名称"
              value={data.service_name || ''}
              onChange={(e) => updateAuthField('service_name', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="custom_base_url">API 地址</Label>
            <Input
              id="custom_base_url"
              placeholder="https://api.example.com"
              value={data.base_url || ''}
              onChange={(e) => updateAuthField('base_url', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="custom_endpoint">端点</Label>
            <Input
              id="custom_endpoint"
              placeholder="/api/emails"
              value={data.endpoint || ''}
              onChange={(e) => updateAuthField('endpoint', e.target.value)}
            />
          </div>
        </div>
        
        {/* 只读信息 */}
        <div className="p-3 rounded-lg bg-muted/30 space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">认证类型</span>
            <Badge variant="outline">{data.auth_type || '-'}</Badge>
          </div>
        </div>

        {/* 可编辑的认证信息 */}
        <Collapsible>
          <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
            <Key className="h-4 w-4" />
            更新认证信息
            <ChevronRight className="h-4 w-4 ml-auto transition-transform ui-open:rotate-90" />
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-3 pt-2">
            {data.auth_type === 'bearer_token' && (
              <div className="space-y-2">
                <Label htmlFor="auth_token">Bearer Token</Label>
                <Input
                  id="auth_token"
                  type="password"
                  placeholder="留空则不修改"
                  value={data.auth_token || ''}
                  onChange={(e) => updateAuthField('auth_token', e.target.value)}
                />
              </div>
            )}
            {data.auth_type === 'api_key' && (
              <div className="space-y-2">
                <Label htmlFor="api_key">API Key</Label>
                <Input
                  id="api_key"
                  type="password"
                  placeholder="留空则不修改"
                  value={data.api_key || ''}
                  onChange={(e) => updateAuthField('api_key', e.target.value)}
                />
              </div>
            )}
            {data.auth_type === 'basic_auth' && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="username">用户名</Label>
                  <Input
                    id="username"
                    placeholder="留空则不修改"
                    value={data.username || ''}
                    onChange={(e) => updateAuthField('username', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="password">密码</Label>
                  <Input
                    id="password"
                    type="password"
                    placeholder="留空则不修改"
                    value={data.password || ''}
                    onChange={(e) => updateAuthField('password', e.target.value)}
                  />
                </div>
              </>
            )}
          </CollapsibleContent>
        </Collapsible>
      </div>
    );
  };

  // 根据服务类型渲染配置
  const renderAuthConfig = () => {
    switch (serviceType) {
      case 'cloudflare_temp_email':
        return renderCloudflareTempEmailConfig();
      case 'cloud_mail':
        return renderCloudMailConfig();
      case 'custom':
        return renderCustomConfig();
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

              {/* 测试连接 */}
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

              {/* 分组设置 */}
              <div className="space-y-2">
                <Label>分组</Label>
                <GroupSelector
                  value={groupId}
                  onChange={setGroupId}
                />
              </div>

              {/* 同步设置 */}
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

              {/* 子邮箱列表 */}
              {(childAccounts.length > 0 || isLoadingChildren) && (
                <Collapsible open={childrenExpanded} onOpenChange={setChildrenExpanded}>
                  <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
                    <Users className="h-4 w-4" />
                    关联邮箱账户
                    <Badge variant="secondary" className="ml-1 text-xs">
                      {childAccounts.length}
                    </Badge>
                    <ChevronRight className={`h-4 w-4 ml-auto transition-transform ${childrenExpanded ? 'rotate-90' : ''}`} />
                  </CollapsibleTrigger>
                  <CollapsibleContent className="pt-2">
                    {isLoadingChildren ? (
                      <div className="flex items-center justify-center py-4">
                        <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                        <span className="ml-2 text-sm text-muted-foreground">加载中...</span>
                      </div>
                    ) : (
                      <div className="space-y-2 max-h-[200px] overflow-y-auto">
                        {childAccounts.map((child) => (
                          <div
                            key={child.uid}
                            className="p-3 rounded-lg bg-muted/30 border text-sm"
                          >
                            <div className="flex items-center gap-2 mb-1">
                              <Mail className="h-4 w-4 text-muted-foreground" />
                              <span className="font-medium truncate flex-1">{child.email}</span>
                              <Badge
                                variant={child.status === 'active' ? 'default' : 'secondary'}
                                className="text-xs"
                              >
                                {child.status === 'active' ? '正常' : child.status}
                              </Badge>
                            </div>
                            <div className="flex items-center gap-4 text-xs text-muted-foreground ml-6">
                              <span>邮件: {child.total_emails}</span>
                              {child.unread_count > 0 && (
                                <span className="text-primary">未读: {child.unread_count}</span>
                              )}
                              {child.last_sync_at && (
                                <span>
                                  同步: {new Date(child.last_sync_at).toLocaleString('zh-CN', {
                                    month: 'numeric',
                                    day: 'numeric',
                                    hour: '2-digit',
                                    minute: '2-digit',
                                  })}
                                </span>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                    <p className="text-xs text-muted-foreground mt-2">
                      这些邮箱账户由此 WebAPI 账户统一管理和同步
                    </p>
                  </CollapsibleContent>
                </Collapsible>
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
