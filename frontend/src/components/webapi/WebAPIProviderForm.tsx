// WebAPI Provider 表单组件
import React, { useState, useCallback, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Switch } from '../ui/switch';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
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
import { Loader2, CheckCircle2, XCircle, ArrowLeft, ArrowRight, ChevronDown } from 'lucide-react';
import { WebAPIServiceSelector, PRESET_SERVICES } from './WebAPIServiceSelector';
import { WebAPIAuthConfig } from './WebAPIAuthConfig';
import { GroupSelector } from '../group';
import { useGroupStore } from '../../stores/groupStore';
import { webapiService } from '../../services/webapiService';
import type {
  WebAPIServiceType,
  WebAPIServiceTemplate,
  WebAPIAuthData,
  CloudflareTempEmailAuthData,
  CloudMailAuthData,
  CustomWebAPIAuthData,
  TestConnectionResult,
} from '../../types/webapi';

// 表单步骤
type FormStep = 'select' | 'config' | 'test';

interface WebAPIProviderFormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
  /** 预选的服务类型，如果提供则跳过选择步骤 */
  preselectedServiceType?: WebAPIServiceType;
}

/**
 * WebAPI Provider 表单组件
 * 集成服务选择、认证配置和连接测试
 */
export const WebAPIProviderForm: React.FC<WebAPIProviderFormProps> = ({
  onSuccess,
  onCancel,
  preselectedServiceType,
}) => {
  // 表单状态
  const [step, setStep] = useState<FormStep>('select');
  const [serviceType, setServiceType] = useState<WebAPIServiceType | undefined>();
  const [template, setTemplate] = useState<WebAPIServiceTemplate | undefined>();
  const [name, setName] = useState('');
  const [authData, setAuthData] = useState<WebAPIAuthData>({} as WebAPIAuthData);
  const [errors, setErrors] = useState<Record<string, string>>({});
  
  // 分组和同步设置
  const [groupId, setGroupId] = useState<number | null>(null);
  const [syncEnabled, setSyncEnabled] = useState(true);
  const [syncInterval, setSyncInterval] = useState(2);
  
  // 测试状态
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestConnectionResult | null>(null);
  
  // 提交状态
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // 检查是否为 Webhook 模式（需要在 useEffect 之前定义）
  const isWebhookMode = serviceType === 'cloudflare_temp_email' && 
    (authData as CloudflareTempEmailAuthData)?.sync_mode === 'webhook';

  // 处理预选服务类型：如果提供了预选类型，自动跳过选择步骤
  useEffect(() => {
    if (preselectedServiceType) {
      const tpl = PRESET_SERVICES.find(s => s.service_type === preselectedServiceType);
      if (tpl) {
        setServiceType(preselectedServiceType);
        setTemplate(tpl);
        setAuthData(tpl.default_config as WebAPIAuthData);
        setName(tpl.name);
        setStep('config');
      }
    }
  }, [preselectedServiceType]);

  // Webhook 模式下自动禁用同步
  useEffect(() => {
    if (isWebhookMode) {
      setSyncEnabled(false);
    }
  }, [isWebhookMode]);

  // 处理服务选择
  const handleServiceSelect = useCallback((type: WebAPIServiceType, tpl: WebAPIServiceTemplate) => {
    setServiceType(type);
    setTemplate(tpl);
    setAuthData(tpl.default_config as WebAPIAuthData);
    setName(tpl.name);
    setErrors({});
    setTestResult(null);
    // 重置分组和同步设置
    setGroupId(null);
    setSyncEnabled(true);
    setSyncInterval(2);
    setStep('config');
  }, []);

  // 验证配置
  const validateConfig = useCallback((): boolean => {
    const newErrors: Record<string, string> = {};
    
    if (!name.trim()) {
      newErrors.name = '请输入名称';
    }

    if (!serviceType) {
      newErrors.service_type = '请选择服务类型';
      setErrors(newErrors);
      return false;
    }

    // 根据服务类型验证
    switch (serviceType) {
      case 'cloudflare_temp_email': {
        const data = authData as CloudflareTempEmailAuthData;
        if (!data.base_url) newErrors.base_url = '请输入 API 地址';
        if (data.access_mode === 'single') {
          // jwt_token 或 user_token 至少需要一个
          if (!data.jwt_token && !data.user_token) {
            newErrors.jwt_token = '请输入 JWT Token 或 User Token（二选一）';
          }
          // 邮箱地址可选，可以通过自动获取填充
        } else if (data.access_mode === 'admin') {
          if (!data.admin_password) newErrors.admin_password = '请输入管理员密码';
        }
        break;
      }
      case 'cloud_mail': {
        const data = authData as CloudMailAuthData;
        if (!data.base_url) newErrors.base_url = '请输入 API 地址';
        // 必须提供 JWT Token 或者 邮箱+密码
        if (!data.jwt_token && (!data.email || !data.password)) {
          if (!data.jwt_token && !data.email) {
            newErrors.email = '请输入登录邮箱或 JWT Token';
          }
          if (!data.jwt_token && !data.password) {
            newErrors.password = '请输入登录密码或 JWT Token';
          }
        }
        break;
      }
      case 'custom': {
        const data = authData as CustomWebAPIAuthData;
        if (!data.service_name) newErrors.service_name = '请输入服务名称';
        if (!data.base_url) newErrors.base_url = '请输入 API 地址';
        if (!data.endpoint) newErrors.endpoint = '请输入邮件端点';
        if (!data.data_path) newErrors.data_path = '请输入数据路径';
        // 认证验证
        if (data.auth_type === 'bearer_token' && !data.auth_token) {
          newErrors.auth_token = '请输入 Token';
        }
        if (data.auth_type === 'api_key' && !data.api_key) {
          newErrors.api_key = '请输入 API Key';
        }
        if (data.auth_type === 'basic_auth') {
          if (!data.username) newErrors.username = '请输入用户名';
          if (!data.password) newErrors.password = '请输入密码';
        }
        break;
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [serviceType, authData, name]);

  // 测试连接
  const handleTestConnection = useCallback(async () => {
    if (!validateConfig() || !serviceType) return;

    setIsTesting(true);
    setTestResult(null);

    try {
      const result = await webapiService.testConnection({
        service_type: serviceType,
        auth_data: JSON.stringify(authData),
      });
      setTestResult(result);
    } catch (error) {
      setTestResult({
        success: false,
        message: '连接测试失败',
        error: error instanceof Error ? error.message : '未知错误',
      });
    } finally {
      setIsTesting(false);
    }
  }, [serviceType, authData, validateConfig]);

  // 提交表单
  const handleSubmit = useCallback(async () => {
    if (!validateConfig() || !serviceType) return;

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      await webapiService.create({
        name: name.trim(),
        service_type: serviceType,
        auth_data: JSON.stringify(authData),
        group_id: groupId,
        sync_enabled: syncEnabled,
        sync_interval: syncInterval,
      });
      onSuccess?.();
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : '创建失败');
    } finally {
      setIsSubmitting(false);
    }
  }, [serviceType, authData, name, groupId, syncEnabled, syncInterval, validateConfig, onSuccess]);

  // 返回上一步
  const handleBack = useCallback(() => {
    if (step === 'config') {
      // 如果有预选服务类型，返回时关闭对话框
      if (preselectedServiceType) {
        onCancel?.();
        return;
      }
      setStep('select');
      setServiceType(undefined);
      setTemplate(undefined);
      setAuthData({} as WebAPIAuthData);
      setErrors({});
      setTestResult(null);
    } else if (step === 'test') {
      setStep('config');
      setTestResult(null);
    }
  }, [step, preselectedServiceType, onCancel]);

  // 进入下一步
  const handleNext = useCallback(() => {
    if (step === 'config') {
      if (validateConfig()) {
        setStep('test');
      }
    }
  }, [step, validateConfig]);

  // 渲染服务选择步骤
  const renderSelectStep = () => (
    <div className="space-y-4">
      <div className="text-center mb-6">
        <h3 className="text-lg font-medium">选择服务类型</h3>
        <p className="text-sm text-muted-foreground">
          选择预置服务或配置自定义 Web API
        </p>
      </div>
      <WebAPIServiceSelector
        selectedType={serviceType}
        onSelect={handleServiceSelect}
      />
    </div>
  );

  // 渲染配置步骤
  const renderConfigStep = () => (
    <div className="space-y-6">
      <div className="flex items-center gap-2 mb-4">
        <Button variant="ghost" size="sm" onClick={handleBack}>
          <ArrowLeft className="h-4 w-4 mr-1" />
          返回
        </Button>
        <span className="text-sm text-muted-foreground">
          配置 {template?.name || '服务'}
        </span>
      </div>

      {/* 基础信息 */}
      <div className="space-y-2">
        <Label htmlFor="name">显示名称 *</Label>
        <Input
          id="name"
          placeholder="输入显示名称"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className={errors.name ? 'border-red-500' : ''}
        />
        {errors.name && <p className="text-sm text-red-500">{errors.name}</p>}
      </div>

      {/* 分组选择 */}
      <div className="space-y-2">
        <Label>分组</Label>
        <GroupSelector
          value={groupId}
          onChange={setGroupId}
          placeholder="选择分组（可选）"
        />
        <p className="text-xs text-muted-foreground">
          将账户归类到指定分组，便于管理
        </p>
      </div>

      {/* 认证配置 */}
      {serviceType && (
        <WebAPIAuthConfig
          serviceType={serviceType}
          authData={authData}
          onChange={setAuthData}
          errors={errors}
        />
      )}

      {/* 同步设置 - 仅轮询模式显示 */}
      {!isWebhookMode && (
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
                <p className="text-xs text-muted-foreground">
                  设置自动同步邮件的时间间隔
                </p>
              </div>
            )}
          </CollapsibleContent>
        </Collapsible>
      )}

      {/* 操作按钮 */}
      <div className="flex justify-end gap-2 pt-4">
        <Button variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button onClick={handleNext}>
          下一步
          <ArrowRight className="h-4 w-4 ml-1" />
        </Button>
      </div>
    </div>
  );

  // 渲染测试步骤
  const renderTestStep = () => {
    // 获取分组名称
    const { groups } = useGroupStore.getState();
    const groupName = groupId 
      ? groups.find(g => g.id === groupId)?.name || '未知分组'
      : '未分组';

    return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 mb-4">
        <Button variant="ghost" size="sm" onClick={handleBack}>
          <ArrowLeft className="h-4 w-4 mr-1" />
          返回
        </Button>
        <span className="text-sm text-muted-foreground">测试连接</span>
      </div>

      {/* 配置摘要 */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">配置摘要</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">名称</span>
            <span>{name}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">服务类型</span>
            <span>{template?.name}</span>
          </div>
          {(authData as any).base_url && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">API 地址</span>
              <span className="truncate max-w-[200px]">{(authData as any).base_url}</span>
            </div>
          )}
          <div className="flex justify-between">
            <span className="text-muted-foreground">分组</span>
            <span>{groupName}</span>
          </div>
          {/* 同步信息 - 仅轮询模式显示 */}
          {!isWebhookMode && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">自动同步</span>
              <span>{syncEnabled ? `每 ${syncInterval} 分钟` : '已禁用'}</span>
            </div>
          )}
          {/* Webhook 模式显示同步模式 */}
          {isWebhookMode && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">同步模式</span>
              <span>Webhook 推送</span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 测试结果 */}
      {testResult && (
        <Alert variant={testResult.success ? 'default' : 'destructive'}>
          <div className="flex items-center gap-2">
            {testResult.success ? (
              <CheckCircle2 className="h-4 w-4 text-green-500" />
            ) : (
              <XCircle className="h-4 w-4" />
            )}
            <AlertDescription>
              {testResult.message}
              {testResult.email_count !== undefined && (
                <span className="ml-2">（发现 {testResult.email_count} 封邮件）</span>
              )}
              {testResult.error && (
                <span className="block text-xs mt-1">{testResult.error}</span>
              )}
            </AlertDescription>
          </div>
        </Alert>
      )}

      {/* 提交错误 */}
      {submitError && (
        <Alert variant="destructive">
          <AlertDescription>{submitError}</AlertDescription>
        </Alert>
      )}

      {/* 操作按钮 */}
      <div className="flex justify-between pt-4">
        <Button
          variant="outline"
          onClick={handleTestConnection}
          disabled={isTesting}
        >
          {isTesting ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              测试中...
            </>
          ) : (
            '测试连接'
          )}
        </Button>
        <div className="flex gap-2">
          <Button variant="outline" onClick={onCancel}>
            取消
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting || (testResult !== null && !testResult.success)}
          >
            {isSubmitting ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                创建中...
              </>
            ) : (
              '创建'
            )}
          </Button>
        </div>
      </div>
    </div>
  );
  };

  return (
    <div className="w-full max-w-2xl mx-auto">
      {step === 'select' && renderSelectStep()}
      {step === 'config' && renderConfigStep()}
      {step === 'test' && renderTestStep()}
    </div>
  );
};

export default WebAPIProviderForm;
