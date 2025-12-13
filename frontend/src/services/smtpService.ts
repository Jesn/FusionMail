/**
 * SMTP 配置服务
 * 处理 SMTP 配置的增删改查和连接测试
 */

import { api } from './api';
import type {
  SMTPConfig,
  UpdateSMTPConfigRequest,
  SMTPTestResult,
  DefaultSMTPConfig,
} from '../types';

export const smtpService = {
  /**
   * 获取账户的 SMTP 配置
   */
  getConfig: async (accountUid: string): Promise<SMTPConfig> => {
    const response = await api.get<{
      success: boolean;
      data: {
        host: string;
        port: number;
        encryption: string;
        username: string;
        enabled: boolean;
      };
    }>(`/accounts/${accountUid}/smtp`);

    // 转换后端字段名为前端格式
    const data = response.data;
    return {
      smtp_host: data.host || '',
      smtp_port: data.port || 465,
      smtp_encryption: (data.encryption as 'none' | 'tls' | 'starttls') || 'tls',
      smtp_username: data.username || '',
      smtp_enabled: data.enabled || false,
    };
  },

  /**
   * 更新账户的 SMTP 配置
   */
  updateConfig: async (accountUid: string, config: UpdateSMTPConfigRequest): Promise<void> => {
    // 转换字段名以匹配后端 API 格式
    const apiConfig = {
      host: config.smtp_host,
      port: config.smtp_port,
      encryption: config.smtp_encryption,
      username: config.smtp_username,
      password: config.smtp_password,
      enabled: config.smtp_enabled,
    };
    await api.put(`/accounts/${accountUid}/smtp`, apiConfig);
  },

  /**
   * 测试 SMTP 连接
   */
  testConnection: async (accountUid: string): Promise<SMTPTestResult> => {
    const response = await api.post<{ success: boolean; data: SMTPTestResult }>(
      `/accounts/${accountUid}/smtp/test`
    );
    return response.data;
  },

  /**
   * 获取常见邮箱服务商的默认 SMTP 配置
   */
  getDefaultConfigs: async (): Promise<DefaultSMTPConfig[]> => {
    const response = await api.get<{ success: boolean; data: DefaultSMTPConfig[] }>(
      '/smtp/defaults'
    );
    return response.data;
  },
};
