import { useState, useEffect } from 'react';
import { ruleService, CreateRuleRequest, UpdateRuleRequest } from '../services/ruleService';
import { Rule } from '../types';
import { toast } from 'sonner';

export const useRules = (accountUid?: string) => {
  const [rules, setRules] = useState<Rule[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchRules = async () => {
    try {
      setIsLoading(true);
      const data = await ruleService.getList(accountUid);
      setRules(data);
    } catch (error) {
      console.error('Failed to fetch rules:', error);
      toast.error('获取规则列表失败');
    } finally {
      setIsLoading(false);
    }
  };

  const createRule = async (data: CreateRuleRequest) => {
    try {
      const newRule = await ruleService.create(data);
      setRules(prev => [newRule, ...prev]);
      toast.success('规则创建成功');
      return newRule;
    } catch (error) {
      console.error('Failed to create rule:', error);
      toast.error('创建规则失败');
      throw error;
    }
  };

  const updateRule = async (id: number, data: UpdateRuleRequest) => {
    try {
      const updatedRule = await ruleService.update(id, data);
      setRules(prev => prev.map(rule => 
        rule.id === id ? updatedRule : rule
      ));
      toast.success('规则更新成功');
      return updatedRule;
    } catch (error) {
      console.error('Failed to update rule:', error);
      toast.error('更新规则失败');
      throw error;
    }
  };

  const deleteRule = async (id: number) => {
    try {
      await ruleService.delete(id);
      setRules(prev => prev.filter(rule => rule.id !== id));
      toast.success('规则删除成功');
    } catch (error) {
      console.error('Failed to delete rule:', error);
      toast.error('删除规则失败');
      throw error;
    }
  };

  const toggleRule = async (id: number) => {
    try {
      await ruleService.toggle(id);
      setRules(prev => prev.map(rule => 
        rule.id === id ? { ...rule, enabled: !rule.enabled } : rule
      ));
      toast.success('规则状态更新成功');
    } catch (error) {
      console.error('Failed to toggle rule:', error);
      toast.error('更新规则状态失败');
      throw error;
    }
  };

  const applyRulesToAccount = async (accountUid: string) => {
    try {
      await ruleService.applyToAccount(accountUid);
      toast.success('规则应用成功');
    } catch (error) {
      console.error('Failed to apply rules:', error);
      toast.error('应用规则失败');
      throw error;
    }
  };

  useEffect(() => {
    fetchRules();
  }, [accountUid]);

  return {
    rules,
    isLoading,
    createRule,
    updateRule,
    deleteRule,
    toggleRule,
    applyRulesToAccount,
    refetch: fetchRules,
  };
};