import { Rule } from '../../types';
import { RuleCard } from './RuleCard';
import { Loader2 } from 'lucide-react';

interface RuleListProps {
  rules: Rule[];
  isLoading: boolean;
  onEdit: (rule: Rule) => void;
  onDelete: (id: number, name: string) => void;
  onToggle: (id: number) => void;
}

export const RuleList = ({ rules, isLoading, onEdit, onDelete, onToggle }: RuleListProps) => {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (rules.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="text-gray-400 mb-4">
          <svg
            className="mx-auto h-12 w-12"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"
            />
          </svg>
        </div>
        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
          暂无规则
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          创建第一个邮件处理规则来自动化您的邮件管理
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {rules.map((rule) => (
        <RuleCard
          key={rule.id}
          rule={rule}
          onEdit={() => onEdit(rule)}
          onDelete={() => onDelete(rule.id, rule.name)}
          onToggle={() => onToggle(rule.id)}
        />
      ))}
    </div>
  );
};