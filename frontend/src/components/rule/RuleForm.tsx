import { useState, useEffect } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Switch } from '../ui/switch';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../ui/dialog';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Rule, RuleCondition, RuleAction } from '../../types';
import { ruleService } from '../../services/ruleService';
import { useAccounts } from '../../hooks/useAccounts';
import { toast } from 'sonner';

interface RuleFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: any) => void;
  rule?: Rule | null;
}

export const RuleForm = ({ open, onClose, onSubmit, rule }: RuleFormProps) => {
  const { accounts } = useAccounts();
  const [formData, setFormData] = useState({
    name: '',
    account_uid: '',
    description: '',
    priority: 1,
    stop_processing: false,
    enabled: true,
  });
  const [conditions, setConditions] = useState<RuleCondition[]>([
    { field: 'from_address', operator: 'contains', value: '' }
  ]);
  const [actions, setActions] = useState<RuleAction[]>([
    { type: 'mark_read' }
  ]);

  useEffect(() => {
    if (rule) {
      setFormData({
        name: rule.name,
        account_uid: rule.account_uid,
        description: rule.description || '',
        priority: rule.priority,
        stop_processing: rule.stop_processing,
        enabled: rule.enabled,
      });
      setConditions(ruleService.parseConditions(rule.conditions));
      setActions(ruleService.parseActions(rule.actions));
    } else {
      // 重置表单
      setFormData({
        name: '',
        account_uid: accounts[0]?.uid || '',
        description: '',
        priority: 1,
        stop_processing: false,
        enabled: true,
      });
      setConditions([{ field: 'from_address', operator: 'contains', value: '' }]);
      setActions([{ type: 'mark_read' }]);
    }
  }, [rule, accounts, open]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    // 验证表单
    if (!formData.name.trim()) {
      toast.error('请输入规则名称');
      return;
    }
    if (!formData.account_uid) {
      toast.error('请选择邮箱账户');
      return;
    }
    if (conditions.some(c => !c.value.trim())) {
      toast.error('请填写所有条件的值');
      return;
    }

    onSubmit({
      ...formData,
      conditions,
      actions,
    });
  };

  const addCondition = () => {
    setConditions([...conditions, { field: 'from_address', operator: 'contains', value: '' }]);
  };

  const removeCondition = (index: number) => {
    if (conditions.length > 1) {
      setConditions(conditions.filter((_, i) => i !== index));
    }
  };

  const updateCondition = (index: number, field: keyof RuleCondition, value: string) => {
    const newConditions = [...conditions];
    newConditions[index] = { ...newConditions[index], [field]: value };
    setConditions(newConditions);
  };

  const addAction = () => {
    setActions([...actions, { type: 'mark_read' }]);
  };

  const removeAction = (index: number) => {
    if (actions.length > 1) {
      setActions(actions.filter((_, i) => i !== index));
    }
  };

  const updateAction = (index: number, field: keyof RuleAction, value: string) => {
    const newActions = [...actions];
    newActions[index] = { ...newActions[index], [field]: value };
    setActions(newActions);
  };

  const fieldOptions = [
    { value: 'from_address', label: '发件人地址' },
    { value: 'from_name', label: '发件人姓名' },
    { value: 'subject', label: '主题' },
    { value: 'body', label: '正文' },
    { value: 'to_addresses', label: '收件人地址' },
  ];

  const operatorOptions = [
    { value: 'contains', label: '包含' },
    { value: 'not_contains', label: '不包含' },
    { value: 'equals', label: '等于' },
    { value: 'not_equals', label: '不等于' },
    { value: 'starts_with', label: '开头是' },
    { value: 'ends_with', label: '结尾是' },
  ];

  const actionOptions = [
    { value: 'mark_read', label: '标记为已读', needsValue: false },
    { value: 'mark_unread', label: '标记为未读', needsValue: false },
    { value: 'star', label: '添加星标', needsValue: false },
    { value: 'archive', label: '归档', needsValue: false },
    { value: 'delete', label: '删除', needsValue: false },
    { value: 'add_label', label: '添加标签', needsValue: true },
    { value: 'trigger_webhook', label: '触发 Webhook', needsValue: true },
  ];

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {rule ? '编辑规则' : '创建规则'}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* 基本信息 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">基本信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="name">规则名称 *</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="输入规则名称"
                  />
                </div>
                <div>
                  <Label htmlFor="account">邮箱账户 *</Label>
                  <Select
                    value={formData.account_uid}
                    onValueChange={(value) => setFormData({ ...formData, account_uid: value })}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="选择邮箱账户" />
                    </SelectTrigger>
                    <SelectContent>
                      {accounts.map((account) => (
                        <SelectItem key={account.uid} value={account.uid}>
                          {account.email}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div>
                <Label htmlFor="description">描述</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="输入规则描述（可选）"
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <Label htmlFor="priority">优先级</Label>
                  <Input
                    id="priority"
                    type="number"
                    min="1"
                    max="100"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 1 })}
                  />
                </div>
                <div className="flex items-center space-x-2">
                  <Switch
                    id="stop_processing"
                    checked={formData.stop_processing}
                    onCheckedChange={(checked) => setFormData({ ...formData, stop_processing: checked })}
                  />
                  <Label htmlFor="stop_processing">停止处理后续规则</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <Switch
                    id="enabled"
                    checked={formData.enabled}
                    onCheckedChange={(checked) => setFormData({ ...formData, enabled: checked })}
                  />
                  <Label htmlFor="enabled">启用规则</Label>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 触发条件 */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">触发条件</CardTitle>
                <Button type="button" variant="outline" size="sm" onClick={addCondition}>
                  <Plus className="h-4 w-4 mr-2" />
                  添加条件
                </Button>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                邮件必须满足所有条件才会触发规则
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              {conditions.map((condition, index) => (
                <div key={index} className="flex items-center gap-4 p-4 border rounded-lg">
                  <div className="flex-1 grid grid-cols-3 gap-4">
                    <Select
                      value={condition.field}
                      onValueChange={(value) => updateCondition(index, 'field', value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {fieldOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>

                    <Select
                      value={condition.operator}
                      onValueChange={(value) => updateCondition(index, 'operator', value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {operatorOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>

                    <Input
                      value={condition.value}
                      onChange={(e) => updateCondition(index, 'value', e.target.value)}
                      placeholder="输入匹配值"
                    />
                  </div>
                  {conditions.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeCondition(index)}
                      className="text-red-600 hover:text-red-700"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
            </CardContent>
          </Card>

          {/* 执行动作 */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">执行动作</CardTitle>
                <Button type="button" variant="outline" size="sm" onClick={addAction}>
                  <Plus className="h-4 w-4 mr-2" />
                  添加动作
                </Button>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                当条件满足时将执行以下动作
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              {actions.map((action, index) => {
                const actionOption = actionOptions.find(opt => opt.value === action.type);
                return (
                  <div key={index} className="flex items-center gap-4 p-4 border rounded-lg">
                    <div className="flex-1 grid grid-cols-2 gap-4">
                      <Select
                        value={action.type}
                        onValueChange={(value) => updateAction(index, 'type', value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {actionOptions.map((option) => (
                            <SelectItem key={option.value} value={option.value}>
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>

                      {actionOption?.needsValue && (
                        <Input
                          value={action.value || ''}
                          onChange={(e) => updateAction(index, 'value', e.target.value)}
                          placeholder={
                            action.type === 'add_label' ? '输入标签名称' :
                            action.type === 'trigger_webhook' ? '输入 Webhook URL' :
                            '输入值'
                          }
                        />
                      )}
                    </div>
                    {actions.length > 1 && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => removeAction(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                );
              })}
            </CardContent>
          </Card>

          {/* 提交按钮 */}
          <div className="flex justify-end gap-4">
            <Button type="button" variant="outline" onClick={onClose}>
              取消
            </Button>
            <Button type="submit">
              {rule ? '更新规则' : '创建规则'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};