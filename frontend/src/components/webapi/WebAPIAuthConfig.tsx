// WebAPI 认证配置组件（容器）：按服务类型分发到纯展示字段组件，fetch 与状态留在本文件
import React, { useState } from 'react';
import { webapiService } from '../../services/webapiService';
import type {
  WebAPIServiceType,
  CloudflareTempEmailAuthData,
  CloudMailAuthData,
  CustomWebAPIAuthData,
} from '../../types/webapi';
import { CloudflareTempEmailConfigFields } from './CloudflareTempEmailConfigFields';
import { CloudMailConfigFields } from './CloudMailConfigFields';
import { CustomWebAPIConfigFields } from './CustomWebAPIConfigFields';

interface WebAPIAuthConfigProps {
  serviceType: WebAPIServiceType;
  authData: any;
  onChange: (data: any) => void;
  errors?: Record<string, string>;
}

/**
 * WebAPI 认证配置组件
 * 根据服务类型动态渲染不同的配置表单（纯展示子组件），容器负责 fetch 与状态
 */
export const WebAPIAuthConfig: React.FC<WebAPIAuthConfigProps> = ({
  serviceType,
  authData,
  onChange,
  errors = {},
}) => {
  const [isFetchingSettings, setIsFetchingSettings] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);

  // 更新字段值
  const updateField = (field: string, value: any) => {
    onChange({ ...authData, [field]: value });
  };

  // 自动获取设置信息（仅 Cloudflare Temp Email 轮询模式使用；fetch 职责留在容器）
  const handleFetchSettings = async () => {
    const data = authData as CloudflareTempEmailAuthData;
    if (!data.base_url || !data.jwt_token) {
      setFetchError('请先填写 API 地址和 JWT Token');
      return;
    }

    setIsFetchingSettings(true);
    setFetchError(null);

    try {
      const settings = await webapiService.fetchCloudflareTempEmailSettings(
        data.base_url,
        data.jwt_token
      );
      // 自动填充邮箱地址
      if (settings.email) {
        onChange({ ...authData, email: settings.email });
      }
    } catch (error) {
      setFetchError(error instanceof Error ? error.message : '获取设置失败');
    } finally {
      setIsFetchingSettings(false);
    }
  };

  // 根据服务类型渲染对应配置
  switch (serviceType) {
    case 'cloudflare_temp_email':
      return (
        <CloudflareTempEmailConfigFields
          data={authData as CloudflareTempEmailAuthData}
          onChange={updateField}
          errors={errors}
          onFetchSettings={handleFetchSettings}
          isFetchingSettings={isFetchingSettings}
          fetchError={fetchError}
        />
      );
    case 'cloud_mail':
      return (
        <CloudMailConfigFields
          data={authData as CloudMailAuthData}
          onChange={updateField}
          errors={errors}
        />
      );
    case 'custom':
      return (
        <CustomWebAPIConfigFields
          data={authData as CustomWebAPIAuthData}
          onChange={updateField}
          errors={errors}
        />
      );
    default:
      return <div className="text-muted-foreground">请先选择服务类型</div>;
  }
};

export default WebAPIAuthConfig;
