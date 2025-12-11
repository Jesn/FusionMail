import { useState } from 'react';
import { Loader2, Users } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { GroupSelector } from './GroupSelector';
import { groupService } from '../../services/groupService';
import { useGroupStore } from '../../stores/groupStore';
import toast from 'react-hot-toast';

interface GroupBatchAssignProps {
  /** 是否打开对话框 */
  open: boolean;
  /** 关闭对话框的回调 */
  onOpenChange: (open: boolean) => void;
  /** 选中的账号 UID 列表 */
  accountUids: string[];
  /** 操作完成后的回调 */
  onComplete?: () => void;
}

/**
 * 批量分配账号到分组的对话框
 */
export const GroupBatchAssign = ({
  open,
  onOpenChange,
  accountUids,
  onComplete,
}: GroupBatchAssignProps) => {
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { fetchGroups } = useGroupStore();

  const handleSubmit = async () => {
    if (accountUids.length === 0) {
      toast.error('请先选择账号');
      return;
    }

    setIsSubmitting(true);
    try {
      const result = await groupService.batchAssignAccounts(accountUids, selectedGroupId);
      
      if (selectedGroupId === null) {
        toast.success(`已将 ${result.count} 个账号移出分组`);
      } else {
        toast.success(`已将 ${result.count} 个账号分配到分组`);
      }

      // 刷新分组列表以更新账号数量
      await fetchGroups();
      
      onComplete?.();
      onOpenChange(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            批量分配分组
          </DialogTitle>
          <DialogDescription>
            将选中的 {accountUids.length} 个账号分配到指定分组
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">目标分组</label>
            <GroupSelector
              value={selectedGroupId}
              onChange={setSelectedGroupId}
              placeholder="选择目标分组（留空表示移出分组）"
              showClear={false}
            />
            <p className="text-xs text-muted-foreground">
              选择"未分组"将把账号从当前分组中移出
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            确认分配
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
