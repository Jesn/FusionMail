import { useState, useEffect } from 'react';
import { CheckCircle, XCircle, Clock, RefreshCw } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../ui/dialog';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Card, CardContent } from '../ui/card';
import { ScrollArea } from '../ui/scroll-area';
import { Webhook, WebhookLog } from '../../services/webhookService';
import { formatDistanceToNow, format } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface WebhookLogsDialogProps {
  open: boolean;
  onClose: () => void;
  webhook: Webhook | null;
  logs: WebhookLog[];
  isLoading: boolean;
  onRefresh: () => void;
}

export const WebhookLogsDialog = ({
  open,
  onClose,
  webhook,
  logs,
  isLoading,
  onRefresh,
}: WebhookLogsDialogProps) => {
  const [selectedLog, setSelectedLog] = useState<WebhookLog | null>(null);

  useEffect(() => {
    if (logs.length > 0 && !selectedLog) {
      setSelectedLog(logs[0]);
    }
  }, [logs, selectedLog]);

  const getStatusIcon = (success: boolean) => {
    return success ? (
      <CheckCircle className="h-4 w-4 text-green-600" />
    ) : (
      <XCircle className="h-4 w-4 text-red-600" />
    );
  };

  const getStatusBadge = (success: boolean) => {
    return success ? (
      <Badge variant="default" className="bg-green-600">成功</Badge>
    ) : (
      <Badge variant="destructive">失败</Badge>
    );
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

  const formatResponseTime = (timeMs: number) => {
    if (timeMs < 1000) {
      return `${timeMs}ms`;
    }
    return `${(timeMs / 1000).toFixed(2)}s`;
  };

  const formatJson = (jsonString?: string) => {
    if (!jsonString) return '';
    try {
      return JSON.stringify(JSON.parse(jsonString), null, 2);
    } catch {
      return jsonString;
    }
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-6xl max-h-[80vh] overflow-hidden">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <DialogTitle>
              Webhook 调用日志 - {webhook?.name}
            </DialogTitle>
            <Button variant="outline" size="sm" onClick={onRefresh} disabled={isLoading}>
              <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              刷新
            </Button>
          </div>
        </DialogHeader>

        <div className="flex gap-4 h-[60vh]">
          {/* 左侧：日志列表 */}
          <div className="flex-1">
            <ScrollArea className="h-full">
              <div className="space-y-2 pr-4">
                {logs.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">
                    暂无调用记录
                  </div>
                ) : (
                  logs.map((log) => (
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
                            {getStatusIcon(log.success)}
                            <span className="font-mono text-sm">{log.request_method}</span>
                            {getStatusBadge(log.success)}
                          </div>
                          <div className="text-xs text-gray-500">
                            {formatTimeAgo(log.created_at)}
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
                          <div className="flex items-center gap-4">
                            <span>状态: {log.response_status}</span>
                            <span>耗时: {formatResponseTime(log.response_time_ms)}</span>
                            {log.retry_count > 0 && (
                              <span>重试: {log.retry_count}</span>
                            )}
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
          <div className="w-96 border-l pl-4">
            {selectedLog ? (
              <ScrollArea className="h-full">
                <div className="space-y-4">
                  {/* 基本信息 */}
                  <div>
                    <h3 className="font-semibold mb-2">请求信息</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex items-center gap-2">
                        <span className="font-medium">方法:</span>
                        <Badge variant="outline" className="font-mono">
                          {selectedLog.request_method}
                        </Badge>
                      </div>
                      
                      <div>
                        <span className="font-medium">URL:</span>
                        <div className="mt-1 p-2 bg-gray-50 dark:bg-gray-800 rounded text-xs font-mono break-all">
                          {selectedLog.request_url}
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-2">
                        <span className="font-medium">时间:</span>
                        <span>{formatDateTime(selectedLog.created_at)}</span>
                      </div>
                    </div>
                  </div>

                  {/* 响应信息 */}
                  <div>
                    <h3 className="font-semibold mb-2">响应信息</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex items-center gap-2">
                        {getStatusIcon(selectedLog.success)}
                        <span className="font-medium">状态:</span>
                        <Badge variant={selectedLog.success ? "default" : "destructive"}>
                          {selectedLog.response_status}
                        </Badge>
                      </div>
                      
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4 text-gray-400" />
                        <span className="font-medium">响应时间:</span>
                        <span>{formatResponseTime(selectedLog.response_time_ms)}</span>
                      </div>
                      
                      {selectedLog.retry_count > 0 && (
                        <div className="flex items-center gap-2">
                          <RefreshCw className="h-4 w-4 text-gray-400" />
                          <span className="font-medium">重试次数:</span>
                          <span>{selectedLog.retry_count}</span>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* 请求头 */}
                  {selectedLog.request_headers && (
                    <div>
                      <h4 className="font-semibold mb-2">请求头</h4>
                      <div className="bg-gray-50 dark:bg-gray-800 rounded p-3">
                        <pre className="text-xs font-mono whitespace-pre-wrap">
                          {formatJson(selectedLog.request_headers)}
                        </pre>
                      </div>
                    </div>
                  )}

                  {/* 请求体 */}
                  {selectedLog.request_body && (
                    <div>
                      <h4 className="font-semibold mb-2">请求体</h4>
                      <div className="bg-gray-50 dark:bg-gray-800 rounded p-3">
                        <pre className="text-xs font-mono whitespace-pre-wrap">
                          {formatJson(selectedLog.request_body)}
                        </pre>
                      </div>
                    </div>
                  )}

                  {/* 响应体 */}
                  {selectedLog.response_body && (
                    <div>
                      <h4 className="font-semibold mb-2">响应体</h4>
                      <div className="bg-gray-50 dark:bg-gray-800 rounded p-3">
                        <pre className="text-xs font-mono whitespace-pre-wrap">
                          {selectedLog.response_body}
                        </pre>
                      </div>
                    </div>
                  )}

                  {/* 错误信息 */}
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
                </div>
              </ScrollArea>
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