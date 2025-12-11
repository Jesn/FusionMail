import { Loader2, AlertTriangle } from 'lucide-react';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../ui/alert-dialog';
import { Button } from '../ui/button';
import type { AccountGroupWithCount } from '../../types';

interface GroupDeleteDialogProps {
  /** 是否打开对话框 */
  open: boolean;
  /** 关闭对话框的回调 */
  onOpenChange: (open: boolean) => void;
  /** 要删除的分组 */
  group: AccountGroupWithCount | null;
  /** 确认删除的回调 */
  onConfirm: () => Promise<void>;
  /** 是否正在删除 */
  isDeleting?: boolean;
}

/**
 * 分组删除确认对话框
 */
export const GroupDeleteDialog = ({
  open,
  onOpenChange,
  group,
  onConfirm,
  isDeleting = false,
}: GroupDeleteDialogProps) => {
  if (!group) return null;

  const handleConfirm = async () => {
    await onConfirm();
    onOpenChange(false);
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-destructive" />
            确认删除分组
          </AlertDialogTitle>
          <AlertDialogDescription className="space-y-2">
            <p>
              您确定要删除分组 <strong>"{group.name}"</strong> 吗？
            </p>
            {group.account_count > 0 && (
              <p className="text-amber-600">
                该分组包含 {group.account_count} 个账号，删除后这些账号将变为"未分组"状态。
              </p>
            )}
            <p className="text-muted-foreground text-sm">
              注意：此操作不会删除账号本身，只会移除分组。
            </p>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isDeleting}
          >
            取消
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={isDeleting}
          >
            {isDeleting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            删除
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
};
