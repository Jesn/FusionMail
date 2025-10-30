import { Plus } from 'lucide-react';
import { useState } from 'react';
import { Button } from '../components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';
import { RuleList } from '../components/rule/RuleList';
import { RuleForm } from '../components/rule/RuleForm';
import { useRules } from '../hooks/useRules';
import { useUIStore } from '../stores/uiStore';
import { Rule } from '../types';

export const RulesPage = () => {
  const {
    rules,
    isLoading,
    createRule,
    updateRule,
    deleteRule,
    toggleRule,
  } = useRules();
  const { isRuleDialogOpen, setRuleDialogOpen } = useUIStore();
  const [editingRule, setEditingRule] = useState<Rule | null>(null);
  const [deletingRule, setDeletingRule] = useState<{ id: number; name: string } | null>(null);

  const handleDeleteClick = (id: number, name: string) => {
    setDeletingRule({ id, name });
  };

  const handleDeleteConfirm = async () => {
    if (deletingRule) {
      await deleteRule(deletingRule.id);
      setDeletingRule(null);
    }
  };

  const handleDeleteCancel = () => {
    setDeletingRule(null);
  };

  const handleEdit = (rule: Rule) => {
    setEditingRule(rule);
    setRuleDialogOpen(true);
  };

  const handleCreate = () => {
    setEditingRule(null);
    setRuleDialogOpen(true);
  };

  const handleDialogClose = () => {
    setRuleDialogOpen(false);
    setEditingRule(null);
  };

  const handleSubmit = async (data: any) => {
    if (editingRule) {
      await updateRule(editingRule.id, data);
    } else {
      await createRule(data);
    }
    handleDialogClose();
  };

  const handleToggle = async (id: number) => {
    await toggleRule(id);
  };

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            邮件规则
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            创建自动化规则来处理收到的邮件
          </p>
        </div>
        <Button onClick={handleCreate} className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
          创建规则
        </Button>
      </div>

      <RuleList
        rules={rules}
        isLoading={isLoading}
        onEdit={handleEdit}
        onDelete={handleDeleteClick}
        onToggle={handleToggle}
      />

      {/* 规则表单对话框 */}
      <RuleForm
        open={isRuleDialogOpen}
        onClose={handleDialogClose}
        onSubmit={handleSubmit}
        rule={editingRule}
      />

      {/* 删除确认对话框 */}
      <AlertDialog open={!!deletingRule} onOpenChange={() => setDeletingRule(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除规则 "{deletingRule?.name}" 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleDeleteCancel}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteConfirm}
              className="bg-red-600 hover:bg-red-700"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};