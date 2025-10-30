import { Edit, Trash2, ToggleLeft, ToggleRight, Play, Clock } from 'lucide-react';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader } from '../ui/card';
import { Badge } from '../ui/badge';
import { Rule, RuleCondition, RuleAction } from '../../types';
import { ruleService } from '../../services/ruleService';

interface RuleCardProps {
  rule: Rule;
  onEdit: () => void;
  onDelete: () => void;
  onToggle: () => void;
}

export const RuleCard = ({ rule, onEdit, onDelete, onToggle }: RuleCardProps) => {
  const conditions = ruleService.parseConditions(rule.conditions);
  const actions = ruleService.parseActions(rule.actions);

  const formatCondition = (condition: RuleCondition) => {
    const fieldNames = {
      from_address: '发件人地址',
      from_name: '发件人姓名',
      subject: '主题',
      body: '正文',
      to_addresses: '收件人地址',
    };

    const operatorNames = {
      contains: '包含',
      not_contains: '不包含',
      equals: '等于',
      not_equals: '不等于',
      starts_with: '开头是',
      ends_with: '结尾是',
    };

    return `${fieldNames[condition.field]} ${operatorNames[condition.operator]} "${condition.value}"`;
  };

  const formatAction = (action: RuleAction) => {
    const actionNames = {
      mark_read: '标记为已读',
      mark_unread: '标记为未读',
      star: '添加星标',
      archive: '归档',
      delete: '删除',
      add_label: `添加标签: ${action.value}`,
      trigger_webhook: `触发 Webhook: ${action.value}`,
    };

    return actionNames[action.type] || action.type;
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN');
  };

  return (
    <Card className={`transition-all duration-200 ${rule.enabled ? 'border-green-200 dark:border-green-800' : 'border-gray-200 dark:border-gray-700 opacity-60'}`}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              {rule.name}
            </h3>
            <Badge variant={rule.enabled ? 'default' : 'secondary'}>
              {rule.enabled ? '启用' : '禁用'}
            </Badge>
            <Badge variant="outline" className="text-xs">
              优先级 {rule.priority}
            </Badge>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={onToggle}
              className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
            >
              {rule.enabled ? (
                <ToggleRight className="h-4 w-4 text-green-600" />
              ) : (
                <ToggleLeft className="h-4 w-4 text-gray-400" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={onEdit}
              className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
            >
              <Edit className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={onDelete}
              className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>
        {rule.description && (
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {rule.description}
          </p>
        )}
      </CardHeader>
      <CardContent className="pt-0">
        <div className="space-y-4">
          {/* 条件 */}
          <div>
            <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
              触发条件 ({conditions.length > 1 ? '满足所有条件' : '满足条件'})
            </h4>
            <div className="space-y-1">
              {conditions.map((condition, index) => (
                <div
                  key={index}
                  className="text-sm text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800 px-3 py-2 rounded"
                >
                  {formatCondition(condition)}
                </div>
              ))}
            </div>
          </div>

          {/* 动作 */}
          <div>
            <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
              执行动作
            </h4>
            <div className="space-y-1">
              {actions.map((action, index) => (
                <div
                  key={index}
                  className="text-sm text-gray-600 dark:text-gray-400 bg-blue-50 dark:bg-blue-900/20 px-3 py-2 rounded"
                >
                  {formatAction(action)}
                </div>
              ))}
            </div>
          </div>

          {/* 统计信息 */}
          <div className="flex items-center justify-between pt-2 border-t border-gray-200 dark:border-gray-700">
            <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
              <div className="flex items-center gap-1">
                <Play className="h-3 w-3" />
                执行 {rule.execution_count} 次
              </div>
              {rule.last_executed_at && (
                <div className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  最后执行: {formatDate(rule.last_executed_at)}
                </div>
              )}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              创建于 {formatDate(rule.created_at)}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};