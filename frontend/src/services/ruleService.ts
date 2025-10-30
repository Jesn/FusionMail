import { api } from './api';
import { Rule, RuleCondition, RuleAction } from '../types';

export interface CreateRuleRequest {
  name: string;
  account_uid: string;
  description?: string;
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
    // 将条件和动作转换为 JSON 字符串
    const payload = {
      ...data,
      conditions: JSON.stringify(data.conditions),
      actions: JSON.stringify(data.actions),
    };
    const response = await api.post<{ success: boolean; data: Rule }>('/rules', payload);
    return response.data;
  },

  /**
   * 更新规则
   */
  update: async (id: number, data: UpdateRuleRequest): Promise<Rule> => {
    // 将条件和动作转换为 JSON 字符串
    const payload = {
      ...data,
      conditions: JSON.stringify(data.conditions),
      actions: JSON.stringify(data.actions),
    };
    const response = await api.put<{ success: boolean; data: Rule }>(`/rules/${id}`, payload);
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
   * 解析规则条件（从 JSON 字符串）
   */
  parseConditions: (conditionsJson: string): RuleCondition[] => {
    try {
      return JSON.parse(conditionsJson);
    } catch {
      return [];
    }
  },

  /**
   * 解析规则动作（从 JSON 字符串）
   */
  parseActions: (actionsJson: string): RuleAction[] => {
    try {
      return JSON.parse(actionsJson);
    } catch {
      return [];
    }
  },
};
