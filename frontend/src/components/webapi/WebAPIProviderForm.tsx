// WebAPI Provider 表单组件（容器）：步骤状态机、校验与提交逻辑留在本文件，
// 各步骤渲染拆分为纯展示子组件（ServiceSelectStep / ConfigStep / TestConnectionStep）。
import React, { useState, useCallback, useEffect } from 'react';
import { PRESET_SERVICES } from './WebAPIServiceSelector';
import { ServiceSelectStep } from './ServiceSelectStep';
import { ConfigStep } from './ConfigStep';
import { TestConnectionStep } from './TestConnectionStep';
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
        
        // Webhook 模式：只需要 webhook_secret
        if (data.sync_mode === 'webhook') {
          if (!data.webhook_secret) {
            newErrors.webhook_secret = '请输入或生成 Webhook Secret';
          }
          break;
        }
        
        // 轮询模式：需要 base_url 和认证信息
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
        // Webhook 模式直接创建，不需要测试连接
        if (isWebhookMode) {
          handleSubmit();
        } else {
          setStep('test');
        }
      }
    }
  }, [step, validateConfig, isWebhookMode, handleSubmit]);

  // 测试步骤的分组名称（store 读取留在容器组件，作为 props 传给纯展示步骤）
  const { groups } = useGroupStore.getState();
  const groupName = groupId 
    ? groups.find(g => g.id === groupId)?.name || '未知分组'
    : '未分组';

  return (
    <div className="w-full max-w-2xl mx-auto">
      {step === 'select' && (
        <ServiceSelectStep
          serviceType={serviceType}
          onSelect={handleServiceSelect}
        />
      )}
      {step === 'config' && (
        <ConfigStep
          template={template}
          name={name}
          onNameChange={setName}
          errors={errors}
          serviceType={serviceType}
          authData={authData}
          onAuthDataChange={setAuthData}
          groupId={groupId}
          onGroupChange={setGroupId}
          isWebhookMode={isWebhookMode}
          syncEnabled={syncEnabled}
          onSyncEnabledChange={setSyncEnabled}
          syncInterval={syncInterval}
          onSyncIntervalChange={setSyncInterval}
          submitError={submitError}
          isSubmitting={isSubmitting}
          onBack={handleBack}
          onNext={handleNext}
          onCancel={onCancel}
        />
      )}
      {step === 'test' && (
        <TestConnectionStep
          name={name}
          template={template}
          authData={authData}
          groupName={groupName}
          isWebhookMode={isWebhookMode}
          syncEnabled={syncEnabled}
          syncInterval={syncInterval}
          testResult={testResult}
          submitError={submitError}
          isTesting={isTesting}
          isSubmitting={isSubmitting}
          onBack={handleBack}
          onTestConnection={handleTestConnection}
          onSubmit={handleSubmit}
          onCancel={onCancel}
        />
      )}
    </div>
  );
};

export default WebAPIProviderForm;
