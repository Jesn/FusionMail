import { useState } from 'react';
import { CheckCircle, XCircle, Clock, RefreshCw, Calendar, User, Mail } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../ui/dialog';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Card, CardContent } from '../ui/card';
import { ScrollArea } from '../ui/scroll-area';
import { useSync, SyncLog } from '../../hooks/useSync';
import { formatDistanceToNow, format } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface SyncLogsDialogProps {
  open: boolean;
  onClose: () => void;
}

export const SyncLogsDialog = ({ open, onClose }: SyncLogsDialogProps) => {
  const { syncLogs, isLoadingLogs, fetchSyncLogs } = useSync();
  const [selectedLog, setSelectedLog] = useState<SyncLog | null>(null);

  const getStatusIcon = (status: SyncLog['status']) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-600" />;
      default:
        return <Clock className="h-4 w-4 text-gray-400" />;
    }
  };

  const getStatusBadge = (status: SyncLog['status']) => {
    switch (status) {
      case 'success':
        return <Badge variant="default" className="bg-green-600">成功</Badge>;
      case 'failed':
        return <Badge variant="destructive">失败</Badge>;
      default:
        return <Badge variant="outline">未知</Badge>;
    }
  };

  const formatDuration = (duration?: number) => {
    if (!duration) return '-';
    if (duration < 1000) return `${duration}ms`;
    if (duration < 60000) return `${Math.round(duration / 1000)}s`;
    return `${Math.round(duration / 60000)}m`;
  };

  const formatTimeAgo = (dateString: string) => {
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) {
        return '时间无效';
      }
      return formatDistanceToNow(date, { 
        addSuffix: true, 
        locale: zhCN 
      });
    } catch (error) {
      return '时间无效';
    }
  };

  const formatDateTime = (dateString: string) => {
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) {
        return '时间无效';
      }
      return format(date, 'yyyy-MM-dd HH:mm:ss');
    } catch (error) {
      return '时间无效';
    }
  };

  const handleRefresh = () => {
    fetchSyncLogs(50);
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <DialogTitle>同步历史记录</DialogTitle>
            <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoadingLogs}>
              <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingLogs ? 'animate-spin' : ''}`} />
              刷新
            </Button>
          </div>
        </DialogHeader>

        <div className="flex gap-4 h-[60vh]">
          {/* 左侧：同步日志列表 */}
          <div className="flex-1">
            <ScrollArea className="h-full">
              <div className="space-y-2 pr-4">
                {syncLogs.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">
                    暂无同步记录
                  </div>
                ) : (
                  syncLogs.map((log) => (
                    <Card
                      key={log.id}
                      className={`cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-gray-800 ${
                        selectedLog?.id === log.id ? 'ring-2 ring-blue-500' : ''
                      }`}
                      onClick={() => setSelectedLog(log)}
                    >
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2">
                            {getStatusIcon(log.status)}
                            <span className="font-medium">{log.account_name}</span>
                            {getStatusBadge(log.status)}
                          </div>
                          <div className="text-xs text-gray-500">
                            {formatTimeAgo(log.start_time)}
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
                          <div className="flex items-center gap-4">
                            <span className="flex items-center gap-1">
                              <Mail className="h-3 w-3" />
                              {log.emails_new + log.emails_updated} 封邮件
                            </span>
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {formatDuration(log.duration)}
                            </span>
                          </div>
                          {log.error_message && (
                            <span className="text-red-600 text-xs truncate max-w-32">
                              {log.error_message}
                            </span>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  ))
                )}
              </div>
            </ScrollArea>
          </div>

          {/* 右侧：详细信息 */}
          <div className="w-80 border-l pl-4">
            {selectedLog ? (
              <div className="space-y-4">
                <div>
                  <h3 className="font-semibold mb-2">同步详情</h3>
                  <div className="space-y-3">
                    <div className="flex items-center gap-2">
                      <User className="h-4 w-4 text-gray-400" />
                      <span className="text-sm">{selectedLog.account_name}</span>
                    </div>
                    
                    <div className="flex items-center gap-2">
                      {getStatusIcon(selectedLog.status)}
                      <span className="text-sm">
                        {selectedLog.status === 'success' && '同步成功'}
                        {selectedLog.status === 'failed' && '同步失败'}
                      </span>
                      {getStatusBadge(selectedLog.status)}
                    </div>

                    <div className="flex items-center gap-2">
                      <Calendar className="h-4 w-4 text-gray-400" />
                      <div className="text-sm">
                        <div>开始: {formatDateTime(selectedLog.start_time)}</div>
                        {selectedLog.end_time && (
                          <div>完成: {formatDateTime(selectedLog.end_time)}</div>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <Mail className="h-4 w-4 text-gray-400" />
                      <span className="text-sm">新增 {selectedLog.emails_new} 封，更新 {selectedLog.emails_updated} 封邮件</span>
                    </div>

                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-gray-400" />
                      <span className="text-sm">耗时 {formatDuration(selectedLog.duration)}</span>
                    </div>
                  </div>
                </div>

                {selectedLog.error_message && (
                  <div>
                    <h4 className="font-semibold mb-2 text-red-600">错误信息</h4>
                    <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded p-3">
                      <p className="text-sm text-red-700 dark:text-red-300">
                        {selectedLog.error_message}
                      </p>
                    </div>
                  </div>
                )}

                <div>
                  <h4 className="font-semibold mb-2">统计信息</h4>
                  <div className="bg-gray-50 dark:bg-gray-800 rounded p-3 space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>账户 UID:</span>
                      <span className="font-mono text-xs">{selectedLog.account_uid}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>同步 ID:</span>
                      <span className="font-mono text-xs">#{selectedLog.id}</span>
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                <div className="text-center">
                  <Clock className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>选择一条记录查看详情</p>
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="flex justify-end">
          <Button onClick={onClose}>
            关闭
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};