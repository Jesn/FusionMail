import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import type { AccountGroupWithCount } from '../../types';

interface GroupDialogProps {
  /** 是否打开对话框 */
  open: boolean;
  /** 关闭对话框的回调 */
  onOpenChange: (open: boolean) => void;
  /** 编辑模式下的分组数据，为 null 时表示创建模式 */
  group?: AccountGroupWithCount | null;
  /** 提交回调 */
  onSubmit: (name: string, description: string) => Promise<void>;
  /** 是否正在提交 */
  isSubmitting?: boolean;
}

/**
 * 分组创建/编辑对话框
 */
export const GroupDialog = ({
  open,
  onOpenChange,
  group,
  onSubmit,
  isSubmitting = false,
}: GroupDialogProps) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState<string | null>(null);

  const isEditMode = !!group;

  // 编辑模式下填充表单
  useEffect(() => {
    if (group) {
      setName(group.name);
      setDescription(group.description || '');
    } else {
      setName('');
      setDescription('');
    }
    setError(null);
  }, [group, open]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // 验证
    const trimmedName = name.trim();
    if (!trimmedName) {
      setError('分组名称不能为空');
      return;
    }

    if (trimmedName.length > 50) {
      setError('分组名称不能超过 50 个字符');
      return;
    }

    try {
      await onSubmit(trimmedName, description.trim());
      onOpenChange(false);
    } catch (err) {
      if (err instanceof Error) {
        // 处理重复名称错误
        if (err.message.includes('已存在') || err.message.includes('duplicate')) {
          setError('分组名称已存在');
        } else {
          setError(err.message);
        }
      } else {
        setError('操作失败，请重试');
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditMode ? '编辑分组' : '创建分组'}</DialogTitle>
            <DialogDescription>
              {isEditMode
                ? '修改分组的名称和描述'
                : '创建一个新的分组来组织您的邮箱账号'}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">
                名称 <span className="text-destructive">*</span>
              </Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="输入分组名称"
                maxLength={50}
                autoFocus
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="description">描述</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="输入分组描述（可选）"
                rows={3}
                maxLength={200}
              />
            </div>

            {error && (
              <div className="text-sm text-destructive">{error}</div>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              取消
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEditMode ? '保存' : '创建'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
