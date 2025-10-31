import { api } from './api';

// 邮箱提供商信息接口
export interface Provider {
  name: string;                 // 提供商标识
  display_name: string;         // 显示名称
  supported_protocols: string[]; // 支持的协议
  recommended_protocol: string; // 推荐协议
  requires_oauth: boolean;      // 是否需要OAuth
  imap_host?: string;          // IMAP服务器地址
  imap_port?: number;          // IMAP端口
  pop3_host?: string;          // POP3服务器地址
  pop3_port?: number;          // POP3端口
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

  /**
   * 获取系统健康状态
   */
  async getHealth() {
    return api.get('/system/health');
  },

  /**
   * 获取系统统计信息
   */
  async getStats() {
    return api.get('/system/stats');
  },
};