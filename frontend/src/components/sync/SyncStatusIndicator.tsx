import { RefreshCw, CheckCircle, XCircle, Clock, Play, Square } from 'lucide-react';
import { Button } from '../ui/button';
import { Progress } from '../ui/progress';
import { Badge } from '../ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../ui/tooltip';
import { useSync } from '../../hooks/useSync';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface SyncStatusIndicatorProps {
  compact?: boolean;
  showControls?: boolean;
}

export const SyncStatusIndicator = ({ compact = false, showControls = true }: SyncStatusIndicatorProps) => {
  const { syncStatus, triggerSync, stopSync } = useSync();

  const handleTriggerSync = async () => {
    try {
      await triggerSync();
    } catch (error) {
      console.error('触发同步失败:', error);
    }
  };

  const handleStopSync = async () => {
    try {
      await stopSync();
    } catch (error) {
      console.error('停止同步失败:', error);
    }
  };

  const getStatusIcon = () => {
    if (syncStatus.isRunning) {
      return <RefreshCw className="h-4 w-4 animate-spin text-blue-600" />;
    }
    if (syncStatus.error) {
      return <XCircle className="h-4 w-4 text-red-600" />;
    }
    if (syncStatus.lastSyncTime) {
      return <CheckCircle className="h-4 w-4 text-green-600" />;
    }
    return <Clock className="h-4 w-4 text-gray-400" />;
  };

  const getStatusText = () => {
    if (syncStatus.isRunning) {
      if (syncStatus.currentAccount) {
        return `正在同步 ${syncStatus.currentAccount}...`;
      }
      return '正在同步...';
    }
    if (syncStatus.error) {
      return '同步失败';
    }
    if (syncStatus.lastSyncTime) {
      return `上次同步: ${formatDistanceToNow(new Date(syncStatus.lastSyncTime), { 
        addSuffix: true, 
        locale: zhCN 
      })}`;
    }
    return '未同步';
  };

  const getStatusBadge = () => {
    if (syncStatus.isRunning) {
      return <Badge variant="default" className="bg-blue-600">同步中</Badge>;
    }
    if (syncStatus.error) {
      return <Badge variant="destructive">失败</Badge>;
    }
    if (syncStatus.lastSyncTime) {
      return <Badge variant="secondary">已完成</Badge>;
    }
    return <Badge variant="outline">待同步</Badge>;
  };

  if (compact) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex items-center gap-2">
              {getStatusIcon()}
              {getStatusBadge()}
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <div className="space-y-1">
              <p>{getStatusText()}</p>
              {syncStatus.isRunning && (
                <p className="text-xs">
                  进度: {syncStatus.completedAccounts}/{syncStatus.totalAccounts}
                </p>
              )}
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return (
    <div className="flex items-center gap-3">
      <div className="flex items-center gap-2">
        {getStatusIcon()}
        <span className="text-sm text-gray-600 dark:text-gray-400">
          {getStatusText()}
        </span>
        {getStatusBadge()}
      </div>

      {syncStatus.isRunning && (
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <Progress 
            value={(syncStatus.completedAccounts / syncStatus.totalAccounts) * 100} 
            className="flex-1 max-w-32"
          />
          <span className="text-xs text-gray-500 whitespace-nowrap">
            {syncStatus.completedAccounts}/{syncStatus.totalAccounts}
          </span>
        </div>
      )}

      {showControls && (
        <div className="flex items-center gap-1">
          {syncStatus.isRunning ? (
            <Button
              variant="outline"
              size="sm"
              onClick={handleStopSync}
              className="h-8"
            >
              <Square className="h-3 w-3 mr-1" />
              停止
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={handleTriggerSync}
              className="h-8"
            >
              <Play className="h-3 w-3 mr-1" />
              同步
            </Button>
          )}
        </div>
      )}
    </div>
  );
};