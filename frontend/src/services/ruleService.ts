import { api } from './api';
import { Rule, RuleCondition, RuleAction } from '../types';

export interface CreateRuleRequest {
  name: string;
  account_uid: string;
  description?: string;
  match_mode: 'all' | 'any'; // 匹配模式：all(所有条件) 或 any(任意条件)
  conditions: RuleCondition[];
  actions: RuleAction[];
  priority?: number;
  stop_processing?: boolean;
  enabled?: boolean;
}

export interface UpdateRuleRequest extends CreateRuleRequest {}

export const ruleService = {
  /**
   * 获取规则列表
   */
  getList: async (accountUid?: string): Promise<Rule[]> => {
    const params = accountUid ? { account_uid: accountUid } : {};
    const response = await api.get<{ success: boolean; data: Rule[] }>('/rules', { params });
    // 解包 response，获取 data 字段（数组）
    return response.data || [];
  },

  /**
   * 获取规则详情
   */
  getById: async (id: number): Promise<Rule> => {
    const response = await api.get<{ success: boolean; data: Rule }>(`/rules/${id}`);
    return response.data;
  },

  /**
   * 创建规则
   */
  create: async (data: CreateRuleRequest): Promise<Rule> => {
    const response = await api.post<{ success: boolean; data: Rule }>('/rules', data);
    return response.data;
  },

  /**
   * 更新规则
   */
  update: async (id: number, data: UpdateRuleRequest): Promise<Rule> => {
    const response = await api.put<{ success: boolean; data: Rule }>(`/rules/${id}`, data);
    return response.data;
  },

  /**
   * 删除规则
   */
  delete: async (id: number): Promise<void> => {
    await api.delete(`/rules/${id}`);
  },

  /**
   * 切换规则启用状态
   */
  toggle: async (id: number): Promise<void> => {
    await api.post(`/rules/${id}/toggle`);
  },

  /**
   * 对账户应用规则
   */
  applyToAccount: async (accountUid: string): Promise<void> => {
    await api.post(`/rules/apply/${accountUid}`);
  },

  /**
   * 解析规则条件（兼容后端返回数组或历史 JSON 字符串）
   */
  parseConditions: (conditions: string | null | RuleCondition[]): RuleCondition[] => {
    if (Array.isArray(conditions)) return conditions as RuleCondition[];
    if (!conditions) return [];
    try {
      return JSON.parse(conditions as string);
    } catch {
      return [];
    }
  },

  /**
   * 解析规则动作（兼容后端返回数组或历史 JSON 字符串）
   */
  parseActions: (actions: string | null | RuleAction[]): RuleAction[] => {
    if (Array.isArray(actions)) return actions as RuleAction[];
    if (!actions) return [];
    try {
      return JSON.parse(actions as string);
    } catch {
      return [];
    }
  },
};
