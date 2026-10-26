/**
 * 配置管理API服务
 * 提供获取、设置、删除配置的接口
 */

import { api } from './api';
import type {
  SettingItem,
  SettingsResponse,
  SettingCategory,
  ImportSettingsResult,
  ExportSettingsOptions,
  ImportSettingsOptions,
  SearchSettingsOptions,
} from '../types/settings';

class SettingsService {
  /**
   * 获取分类下的所有配置
   */
  async getSettingsByCategory(
    category: string,
    options?: {
      includeSensitive?: boolean;
      onlyPublic?: boolean;
      userId?: number;
    }
  ): Promise<SettingsResponse> {
    const params = new URLSearchParams();
    if (options?.includeSensitive) params.append('include_sensitive', 'true');
    if (options?.onlyPublic) params.append('only_public', 'true');
    if (options?.userId) params.append('user_id', options.userId.toString());

    const queryString = params.toString();
    const url = `/settings/${category}${queryString ? `?${queryString}` : ''}`;

    // api.get 已解包一层 axios 响应，response 即后端响应体 { success, data }
    const response = await api.get<{ success: boolean; data: SettingsResponse }>(url);
    return response.data;
  }

  /**
   * 获取单个配置项
   */
  async getSetting(
    category: string,
    key: string,
    options?: {
      includeSensitive?: boolean;
      userId?: number;
    }
  ): Promise<string> {
    const params = new URLSearchParams();
    if (options?.includeSensitive) params.append('include_sensitive', 'true');
    if (options?.userId) params.append('user_id', options.userId.toString());

    const queryString = params.toString();
    const url = `/settings/${category}/${key}${queryString ? `?${queryString}` : ''}`;

    const response = await api.get<{
      success: boolean;
      data: { category: string; key: string; value: string };
    }>(url);
    return response.data.value;
  }

  /**
   * 设置单个配置项
   */
  async setSetting(
    category: string,
    key: string,
    value: string,
    options?: {
      isSensitive?: boolean;
      userId?: number;
    }
  ): Promise<void> {
    const url = `/settings/${category}/${key}`;
    await api.put(url, {
      value,
      is_sensitive: options?.isSensitive || false,
      user_id: options?.userId,
    });
  }

  /**
   * 批量设置配置项
   */
  async batchSetSettings(
    category: string,
    settings: Record<string, string>,
    options?: {
      userId?: number;
      isSensitiveMap?: Record<string, boolean>;
    }
  ): Promise<void> {
    const url = `/settings/${category}/batch`;
    await api.post(url, {
      settings,
      is_sensitive_map: options?.isSensitiveMap || {},
      user_id: options?.userId,
    });
  }

  /**
   * 删除配置项
   */
  async deleteSetting(
    category: string,
    key: string
  ): Promise<void> {
    const url = `/settings/${category}/${key}`;
    await api.delete(url);
  }

  /**
   * 获取所有公开配置（前端可访问）
   */
  async getPublicSettings(): Promise<Record<string, Record<string, string>>> {
    const response = await api.get<{
      success: boolean;
      data: { settings: Record<string, Record<string, string>> };
    }>('/settings/public');
    return response.data.settings;
  }

  /**
   * 重置配置为默认值
   */
  async resetSetting(
    category: string,
    key: string,
    options?: {
      userId?: number;
    }
  ): Promise<void> {
    const url = `/settings/${category}/${key}/reset`;
    await api.post(url, {
      user_id: options?.userId,
    });
  }

  /**
   * 获取配置分类列表
   */
  async getSettingCategories(): Promise<SettingCategory[]> {
    // 注意：后端当前没有 /settings/categories 路由（会被 /settings/:category 匹配），
    // 这里按实际响应体解包并做兜底，避免返回 undefined
    const response = await api.get<{
      success: boolean;
      data: { categories?: SettingCategory[] };
    }>('/settings/categories');
    return response.data.categories ?? [];
  }

  /**
   * 搜索配置项
   */
  async searchSettings(
    query: string,
    options?: SearchSettingsOptions
  ): Promise<SettingItem[]> {
    const params = new URLSearchParams({
      q: query,
    });
    if (options?.category) params.append('category', options.category);
    if (options?.onlyPublic) params.append('only_public', 'true');

    // 后端 /settings/search 返回 { success, data: { query, category, results } }
    const response = await api.get<{
      success: boolean;
      data: { query: string; category: string; results: SettingItem[] };
    }>(`/settings/search?${params.toString()}`);
    return response.data.results;
  }

  /**
   * 导出配置
   */
  async exportSettings(
    category?: string,
    options?: ExportSettingsOptions
  ): Promise<Blob> {
    const params = new URLSearchParams();
    if (category) params.append('category', category);
    if (options?.includeSensitive) params.append('include_sensitive', 'true');
    if (options?.userId) params.append('user_id', options.userId.toString());
    if (options?.format) params.append('format', options.format);

    // responseType 为 blob 时，api.get 返回的就是 Blob 本身，不能再取 .data
    const response = await api.get<Blob>(`/settings/export?${params.toString()}`, {
      responseType: 'blob',
    });

    return response;
  }

  /**
   * 导入配置
   */
  async importSettings(
    file: File,
    options?: ImportSettingsOptions
  ): Promise<ImportSettingsResult> {
    const formData = new FormData();
    formData.append('file', file);
    if (options?.category) formData.append('category', options.category);
    if (options?.overwrite !== undefined) formData.append('overwrite', options.overwrite.toString());
    if (options?.userId) formData.append('user_id', options.userId.toString());

    const response = await api.post<{ success: boolean; data: ImportSettingsResult }>(
      '/settings/import',
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    );

    return response.data;
  }
}

export const settingsService = new SettingsService();
