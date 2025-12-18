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
        smtp_host: string;
        smtp_port: number;
        smtp_encryption: string;
        smtp_username: string;
        smtp_enabled: boolean;
        from_provider: boolean;
        provider_name: string;
      };
    }>(`/accounts/${accountUid}/smtp`);

    // 后端返回的字段名已经是 smtp_ 前缀格式
    const data = response.data;
    return {
      smtp_host: data.smtp_host || '',
      smtp_port: data.smtp_port || 465,
      smtp_encryption: (data.smtp_encryption as 'none' | 'tls' | 'starttls' | 'ssl') || 'tls',
      smtp_username: data.smtp_username || '',
      smtp_enabled: data.smtp_enabled || false,
      from_provider: data.from_provider || false,
      provider_name: data.provider_name || '',
    };
  },

  /**
   * 更新账户的 SMTP 配置
   * 注意：SMTP 使用与 IMAP/POP3 相同的邮箱地址和密码，只需配置启用状态
   */
  updateConfig: async (accountUid: string, config: UpdateSMTPConfigRequest): Promise<void> => {
    await api.put(`/accounts/${accountUid}/smtp`, {
      enabled: config.smtp_enabled,
    });
  },

  /**
   * 测试 SMTP 连接
   * @param accountUid 账户 UID
   * @param tempCredentials 临时凭证（用于测试未保存的配置）
   */
  testConnection: async (
    accountUid: string,
    tempCredentials?: { username?: string; password?: string }
  ): Promise<SMTPTestResult> => {
    const response = await api.post<{ success: boolean; data: SMTPTestResult }>(
      `/accounts/${accountUid}/smtp/test`,
      tempCredentials || {}
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
