import { useState, useEffect, useCallback } from 'react';
import { Progress } from '../ui/progress';
import { Button } from '../ui/button';
import { Alert, AlertDescription } from '../ui/alert';
import {
  AlertCircle,
  CheckCircle2,
  XCircle,
  Loader2,
  StopCircle,
} from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../ui/alert-dialog';
import { SyncProgress as SyncProgressType } from '../../types';
import { accountService } from '../../services/accountService';
import toast from 'react-hot-toast';

interface SyncProgressProps {
  accountUid: string;
  onComplete?: () => void;
  onCancel?: () => void;
  className?: string;
}

/**
 * 同步进度显示组件
 * Requirements: 2.1, 2.2, 2.3, 2.4 - 同步进度追踪和显示
 */
export const SyncProgressComponent = ({
  accountUid,
  onComplete,
  onCancel,
  className = '',
}: SyncProgressProps) => {
  const [progress, setProgress] = useState<SyncProgressType | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCancelling, setIsCancelling] = useState(false);
  const [showCancelDialog, setShowCancelDialog] = useState(false);

  // 获取同步进度
  const fetchProgress = useCallback(async () => {
    try {
      const data = await accountService.getSyncProgress(accountUid);
      setProgress(data);
      
      // 如果同步完成，触发回调
      if (data && (data.status === 'completed' || data.status === 'failed' || data.status === 'cancelled')) {
        onComplete?.();
      }
    } catch (error) {
      console.error('获取同步进度失败:', error);
    } finally {
      setIsLoading(false);
    }
  }, [accountUid, onComplete]);

  // 定时轮询进度
  useEffect(() => {
    fetchProgress();
    
    // 每 2 秒轮询一次进度
    const interval = setInterval(() => {
      if (progress && !['completed', 'failed', 'cancelled'].includes(progress.status)) {
        fetchProgress();
      }
    }, 2000);

    return () => clearInterval(interval);
  }, [fetchProgress, progress?.status]);

  // 取消同步
  const handleCancelSync = async () => {
    setIsCancelling(true);
    try {
      await accountService.cancelSync(accountUid);
      toast.success('同步已取消');
      setShowCancelDialog(false);
      onCancel?.();
      // 刷新进度
      await fetchProgress();
    } catch (error) {
      console.error('取消同步失败:', error);
      toast.error('取消同步失败');
    } finally {
      setIsCancelling(false);
    }
  };

  // 计算进度百分比
  const getPercent = () => {
    if (!progress || progress.total_estimated <= 0) return 0;
    const percent = (progress.processed / progress.total_estimated) * 100;
    return Math.min(percent, 100);
  };

  // 获取状态图标
  const getStatusIcon = () => {
    if (!progress) return null;
    
    switch (progress.status) {
      case 'started':
      case 'in_progress':
        return <Loader2 className="h-5 w-5 animate-spin text-blue-500" />;
      case 'completed':
        return <CheckCircle2 className="h-5 w-5 text-green-500" />;
      case 'failed':
        return <XCircle className="h-5 w-5 text-red-500" />;
      case 'cancelled':
        return <StopCircle className="h-5 w-5 text-yellow-500" />;
      default:
        return null;
    }
  };

  // 获取状态文本
  const getStatusText = () => {
    if (!progress) return '加载中...';
    
    switch (progress.status) {
      case 'started':
        return '同步已开始';
      case 'in_progress':
        return progress.phase === 'fetching' 
          ? '正在拉取邮件...' 
          : progress.phase === 'processing' 
            ? '正在处理邮件...' 
            : '正在完成同步...';
      case 'completed':
        return '同步完成';
      case 'failed':
        return '同步失败';
      case 'cancelled':
        return '同步已取消';
      default:
        return '未知状态';
    }
  };

  // 获取阶段文本
  const getPhaseText = () => {
    if (!progress) return '';
    
    switch (progress.phase) {
      case 'fetching':
        return '拉取邮件';
      case 'processing':
        return '处理邮件';
      case 'finalizing':
        return '完成同步';
      default:
        return '';
    }
  };

  if (isLoading) {
    return (
      <div className={`flex items-center justify-center p-4 ${className}`}>
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-sm text-muted-foreground">加载同步状态...</span>
      </div>
    );
  }

  if (!progress) {
    return (
      <div className={`text-sm text-muted-foreground p-4 ${className}`}>
        当前没有进行中的同步
      </div>
    );
  }

  const isActive = progress.status === 'started' || progress.status === 'in_progress';
  const percent = getPercent();

  return (
    <div className={`space-y-4 ${className}`}>
      {/* 状态头部 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {getStatusIcon()}
          <span className="font-medium">{getStatusText()}</span>
          {progress.is_first_sync && (
            <span className="text-xs bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 px-2 py-0.5 rounded">
              首次同步
            </span>
          )}
        </div>
        
        {/* 取消按钮 - 仅在同步进行中显示 */}
        {isActive && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowCancelDialog(true)}
            disabled={isCancelling}
          >
            {isCancelling ? (
              <Loader2 className="h-4 w-4 animate-spin mr-1" />
            ) : (
              <StopCircle className="h-4 w-4 mr-1" />
            )}
            取消同步
          </Button>
        )}
      </div>

      {/* 进度条 */}
      {isActive && (
        <div className="space-y-2">
          <Progress value={percent} className="h-2" />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>
              {progress.processed} / {progress.total_estimated > 0 ? progress.total_estimated : '?'} 封邮件
            </span>
            <span>{Math.round(percent)}%</span>
          </div>
        </div>
      )}

      {/* 详细信息 */}
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div className="space-y-1">
          <span className="text-muted-foreground">当前阶段</span>
          <p className="font-medium">{getPhaseText()}</p>
        </div>
        {progress.total_batches > 0 && (
          <div className="space-y-1">
            <span className="text-muted-foreground">批次进度</span>
            <p className="font-medium">
              {progress.current_batch} / {progress.total_batches}
            </p>
          </div>
        )}
        <div className="space-y-1">
          <span className="text-muted-foreground">新邮件</span>
          <p className="font-medium text-green-600">{progress.new_emails}</p>
        </div>
        <div className="space-y-1">
          <span className="text-muted-foreground">更新邮件</span>
          <p className="font-medium text-blue-600">{progress.updated_emails}</p>
        </div>
        {progress.failed_emails > 0 && (
          <div className="space-y-1">
            <span className="text-muted-foreground">失败邮件</span>
            <p className="font-medium text-red-600">{progress.failed_emails}</p>
          </div>
        )}
      </div>

      {/* 错误信息 */}
      {progress.error_message && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{progress.error_message}</AlertDescription>
        </Alert>
      )}

      {/* 取消确认对话框 */}
      <AlertDialog open={showCancelDialog} onOpenChange={setShowCancelDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认取消同步？</AlertDialogTitle>
            <AlertDialogDescription>
              取消同步后，已处理的邮件将保留，未处理的邮件将在下次同步时继续处理。
              {progress.is_first_sync && (
                <span className="block mt-2 text-yellow-600 dark:text-yellow-400">
                  ⚠️ 这是首次同步，取消后可能需要重新开始同步过程。
                </span>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isCancelling}>继续同步</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleCancelSync}
              disabled={isCancelling}
              className="bg-red-600 hover:bg-red-700"
            >
              {isCancelling ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-1" />
                  取消中...
                </>
              ) : (
                '确认取消'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default SyncProgressComponent;
