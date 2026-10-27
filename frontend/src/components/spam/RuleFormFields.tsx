import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import type { SpamRuleRequest } from '../../services/spamService';

// 规则类别选项
export const CATEGORY_OPTIONS = [
  { value: 'keyword', label: '关键词' },
  { value: 'pattern', label: '正则表达式' },
  { value: 'sender', label: '发件人' },
  { value: 'domain', label: '域名' },
];

export const getCategoryLabel = (category: string) => {
  const opt = CATEGORY_OPTIONS.find((o) => o.value === category);
  return opt?.label || category;
};

export const getCategoryColor = (category: string) => {
  switch (category) {
    case 'keyword':
      return 'bg-blue-100 text-blue-800';
    case 'pattern':
      return 'bg-purple-100 text-purple-800';
    case 'sender':
      return 'bg-orange-100 text-orange-800';
    case 'domain':
      return 'bg-green-100 text-green-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
};

interface RuleFormFieldsProps {
  formData: SpamRuleRequest;
  /** 局部更新表单字段（页面侧合并进 formData） */
  onChange: (patch: Partial<SpamRuleRequest>) => void;
  /** create：带占位符与提示文案；edit：纯字段（与拆分前两对话框渲染一致） */
  mode: 'create' | 'edit';
}

export const RuleFormFields = ({ formData, onChange, mode }: RuleFormFieldsProps) => {
  const isCreate = mode === 'create';
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor={`${mode}-name`}>
          规则名称
        </Label>
        <Input
          id={`${mode}-name`}
          value={formData.name}
          onChange={(e) => onChange({ name: e.target.value })}
          placeholder={isCreate ? '输入规则名称' : undefined}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor={`${mode}-category`}>
          规则类别
        </Label>
        <Select
          value={formData.category}
          onValueChange={(value) => onChange({ category: value })}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {CATEGORY_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label htmlFor={`${mode}-pattern`}>
          匹配模式
        </Label>
        <Input
          id={`${mode}-pattern`}
          value={formData.pattern}
          onChange={(e) => onChange({ pattern: e.target.value })}
          placeholder={
            isCreate
              ? formData.category === 'pattern'
                ? '输入正则表达式'
                : '输入关键词'
              : undefined
          }
        />
        {isCreate && (
          <p className="text-xs text-muted-foreground">
            {formData.category === 'pattern'
              ? '使用正则表达式进行匹配'
              : '使用关键词进行匹配（不区分大小写）'}
          </p>
        )}
      </div>
      <div className="space-y-2">
        <Label htmlFor={`${mode}-score`}>
          评分权重
        </Label>
        <Input
          id={`${mode}-score`}
          type="number"
          min={1}
          max={100}
          value={formData.score}
          onChange={(e) => onChange({ score: parseInt(e.target.value) || 10 })}
        />
        {isCreate && (
          <p className="text-xs text-muted-foreground">
            匹配时增加的垃圾邮件评分（1-100）
          </p>
        )}
      </div>
      <div className="space-y-2">
        <Label htmlFor={`${mode}-description`}>
          描述{isCreate ? '（可选）' : ''}
        </Label>
        <Textarea
          id={`${mode}-description`}
          value={formData.description}
          onChange={(e) => onChange({ description: e.target.value })}
          placeholder={isCreate ? '输入规则描述' : undefined}
          rows={2}
        />
      </div>
    </div>
  );
};
