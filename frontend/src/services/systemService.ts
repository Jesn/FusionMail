import { api } from './api';

// 适配器简要信息（用于 Provider 关联）
export interface AdapterInfo {
  id: number;
  name: string;
  display_name: string;
  auth_type: 'oauth2' | 'password';
}

// 邮箱提供商信息接口
export interface Provider {
  id: number;                   // 提供商ID
  name: string;                 // 提供商标识
  display_name: string;         // 显示名称
  supported_protocols: string[]; // 支持的协议
  recommended_protocol: string; // 推荐协议
  requires_oauth: boolean;      // 是否需要OAuth
  enabled: boolean;             // 是否启用
  imap_host?: string;          // IMAP服务器地址
  imap_port?: number;          // IMAP端口
  pop3_host?: string;          // POP3服务器地址
  pop3_port?: number;          // POP3端口
  smtp_host?: string;          // SMTP服务器地址
  smtp_port?: number;          // SMTP端口
  // 加密配置
  imap_encryption?: string;    // IMAP加密方式 (ssl/starttls/none)
  pop3_encryption?: string;    // POP3加密方式 (ssl/starttls/none)
  smtp_encryption?: string;    // SMTP加密方式 (ssl/starttls/none)
  // 适配器关联字段
  default_adapter_id?: number;  // 默认适配器 ID
  email_domains?: string[];     // 支持的邮箱域名列表
  supported_adapters?: AdapterInfo[]; // 支持的适配器列表
  // 描述信息
  description?: string;         // 提供商描述（可用于密码提示等）
}

// 系统服务
export const systemService = {
  /**
   * 获取支持的邮箱提供商列表
   */
  async getProviders(): Promise<Provider[]> {
    const response = await api.get<{
      success: boolean;
      data: Provider[];
    }>('/system/providers');
    
    if (response.success && response.data) {
      return response.data;
    }
    
    throw new Error('获取邮箱提供商列表失败');
  },

  // 注意：以下方法用于运维监控，前端暂不使用
  // 如需使用，请取消注释并在相应页面中调用
  
  // /**
  //  * 获取系统健康状态（用于运维监控）
  //  */
  // async getHealth() {
  //   return api.get('/system/health');
  // },

  // /**
  //  * 获取系统统计信息（用于运维监控）
  //  */
  // async getStats() {
  //   return api.get('/system/stats');
  // },
};