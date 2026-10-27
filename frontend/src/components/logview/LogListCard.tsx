import { FileText, RefreshCw, ChevronDown, ChevronUp, Clock, Code, MapPin } from 'lucide-react';
import { Button } from '../ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { ScrollArea } from '../ui/scroll-area';
import { Badge } from '../ui/badge';
import { cn } from '../../lib/utils';
import { levelConfig, levelBadgeColors } from './logMeta';
import type { LogEntry } from './logMeta';

interface LogListCardProps {
  logs: LogEntry[];
  isLoading: boolean;
  expandedRows: Set<number>;
  onToggleRow: (index: number) => void;
  page: number;
  onPageChange: (page: number) => void;
  pageSize: number;
  onPageSizeChange: (v: string) => void;
  totalPages: number;
  total: number;
}

/** 日志列表卡片（自 LogsPage 同位搬移，含分页）。
 * 纯展示：不请求、不碰 store；levelConfig/levelBadgeColors 自 logMeta 导入。 */
export function LogListCard({
  logs,
  isLoading,
  expandedRows,
  onToggleRow,
  page,
  onPageChange,
  pageSize,
  onPageSizeChange,
  totalPages,
  total,
}: LogListCardProps) {
  return (
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>日志记录</CardTitle>
                <CardDescription>
                  共 {total} 条记录，当前显示第 {page} 页
                </CardDescription>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground">每页</span>
                <Select value={String(pageSize)} onValueChange={onPageSizeChange}>
                  <SelectTrigger className="w-20 h-7 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="20">20 条</SelectItem>
                    <SelectItem value="50">50 条</SelectItem>
                    <SelectItem value="100">100 条</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            <ScrollArea className="h-[600px]">
              {isLoading ? (
                <div className="flex h-40 items-center justify-center">
                  <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : logs.length === 0 ? (
                <div className="flex h-40 flex-col items-center justify-center text-muted-foreground">
                  <FileText className="h-12 w-12 mb-2 opacity-50" />
                  <span>暂无日志记录</span>
                </div>
              ) : (
                <div className="divide-y">
                  {logs.map((log, index) => {
                    const config = levelConfig[log.level] || levelConfig.INFO;
                    const isExpanded = expandedRows.has(index);
                    
                    return (
                      <div
                        key={index}
                        className={cn(
                          'border-l-4 transition-colors cursor-pointer',
                          config.border,
                          isExpanded ? config.bg : 'hover:bg-muted/30'
                        )}
                        onClick={() => onToggleRow(index)}
                      >
                        {/* 主要内容行 */}
                        <div className="px-4 py-3">
                          <div className="flex items-start gap-3">
                            {/* 级别标签 */}
                            <Badge 
                              className={cn(
                                'shrink-0 font-mono text-xs px-2 py-0.5',
                                levelBadgeColors[log.level] || levelBadgeColors.INFO
                              )}
                            >
                              {log.level || 'INFO'}
                            </Badge>
                            
                            {/* 时间戳 */}
                            <div className="shrink-0 flex items-center gap-1 text-xs text-muted-foreground font-mono">
                              <Clock className="h-3 w-3" />
                              {log.timestamp}
                            </div>
                            
                            {/* 模块标签 */}
                            {log.module && (
                              <Badge variant="outline" className="shrink-0 text-xs font-normal">
                                {log.module}
                              </Badge>
                            )}
                            
                            {/* 展开/收起图标 */}
                            <div className="ml-auto shrink-0">
                              {isExpanded ? (
                                <ChevronUp className="h-4 w-4 text-muted-foreground" />
                              ) : (
                                <ChevronDown className="h-4 w-4 text-muted-foreground" />
                              )}
                            </div>
                          </div>
                          
                          {/* 消息内容 - 始终显示完整内容 */}
                          <div className="mt-2 pl-0">
                            <p className={cn(
                              'text-sm leading-relaxed',
                              isExpanded ? '' : 'line-clamp-2'
                            )}>
                              {log.message}
                            </p>
                          </div>
                        </div>
                        
                        {/* 展开的详细信息 */}
                        {isExpanded && (
                          <div className="px-4 pb-4 space-y-3">
                            {/* 原始日志 */}
                            <div className="rounded-md bg-muted/50 p-3">
                              <div className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                <Code className="h-3.5 w-3.5" />
                                <span className="font-medium">原始日志</span>
                              </div>
                              <pre className="text-xs font-mono whitespace-pre-wrap break-all leading-relaxed overflow-x-auto max-h-48 overflow-y-auto">
                                {log.raw}
                              </pre>
                            </div>
                            
                            {/* 附加信息 */}
                            {(log.fields || log.location) && (
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                                {log.fields && (
                                  <div className="rounded-md bg-muted/30 p-3">
                                    <div className="flex items-center gap-2 mb-1.5 text-xs text-muted-foreground">
                                      <span className="font-medium">附加字段</span>
                                    </div>
                                    <p className="text-xs font-mono break-all">{log.fields}</p>
                                  </div>
                                )}
                                {log.location && (
                                  <div className="rounded-md bg-muted/30 p-3">
                                    <div className="flex items-center gap-2 mb-1.5 text-xs text-muted-foreground">
                                      <MapPin className="h-3.5 w-3.5" />
                                      <span className="font-medium">代码位置</span>
                                    </div>
                                    <p className="text-xs font-mono break-all">{log.location}</p>
                                  </div>
                                )}
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </ScrollArea>

            {/* 分页 */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between border-t px-4 py-3">
                <div className="text-sm text-muted-foreground">
                  第 {page} / {totalPages} 页，共 {total} 条
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(1)}
                    disabled={page === 1}
                  >
                    首页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(page - 1)}
                    disabled={page === 1}
                  >
                    上一页
                  </Button>
                  <span className="text-sm px-2">{page}</span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(page + 1)}
                    disabled={page >= totalPages}
                  >
                    下一页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(totalPages)}
                    disabled={page >= totalPages}
                  >
                    末页
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
  );
}
