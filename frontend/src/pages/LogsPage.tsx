import { useState, useEffect, useCallback, useRef } from 'react';
import { FileText, Search, Download, Trash2, RefreshCw, Filter, ChevronDown, ChevronUp, Clock, Code, MapPin } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Badge } from '../components/ui/badge';
import { ScrollArea } from '../components/ui/scroll-area';
import { Collapsible, CollapsibleContent } from '../components/ui/collapsible';
import { toast } from 'sonner';
import { api } from '../services/api';
import { cn } from '../lib/utils';

// 统计缓存类型
interface StatsCache {
  data: LogStats;
  timestamp: number;
}

// 缓存有效期（60秒）
const STATS_CACHE_TTL = 60 * 1000;

// 日志条目类型
interface LogEntry {
  timestamp: string;
  level: string;
  module: string;
  message: string;
  fields: string;
  location: string;
  raw: string;
}

// 日志文件信息
interface LogFile {
  name: string;
  display_name: string;
  size: number;
  modified_at: string;
}

// 日志统计
interface LogStats {
  debug: number;
  info: number;
  warn: number;
  error: number;
  fatal: number;
  total: number;
}

// 日志级别样式配置
const levelConfig: Record<string, { bg: string; text: string; border: string; icon: string }> = {
  DEBUG: { 
    bg: 'bg-slate-50 dark:bg-slate-900/50', 
    text: 'text-slate-600 dark:text-slate-400',
    border: 'border-l-slate-400',
    icon: '🔍'
  },
  INFO: { 
    bg: 'bg-blue-50 dark:bg-blue-950/30', 
    text: 'text-blue-600 dark:text-blue-400',
    border: 'border-l-blue-500',
    icon: 'ℹ️'
  },
  WARN: { 
    bg: 'bg-amber-50 dark:bg-amber-950/30', 
    text: 'text-amber-600 dark:text-amber-400',
    border: 'border-l-amber-500',
    icon: '⚠️'
  },
  ERROR: { 
    bg: 'bg-red-50 dark:bg-red-950/30', 
    text: 'text-red-600 dark:text-red-400',
    border: 'border-l-red-500',
    icon: '❌'
  },
  FATAL: { 
    bg: 'bg-purple-50 dark:bg-purple-950/30', 
    text: 'text-purple-600 dark:text-purple-400',
    border: 'border-l-purple-500',
    icon: '💀'
  },
};

// 日志级别 Badge 颜色
const levelBadgeColors: Record<string, string> = {
  DEBUG: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
  INFO: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  WARN: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300',
  ERROR: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  FATAL: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
};

export const LogsPage = () => {
  // 状态
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [logFiles, setLogFiles] = useState<LogFile[]>([]);
  const [stats, setStats] = useState<LogStats | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());

  // 统计缓存 (按文件名缓存)
  const statsCacheRef = useRef<Map<string, StatsCache>>(new Map());

  // 筛选参数
  const [selectedFile, setSelectedFile] = useState('backend');
  const [level, setLevel] = useState('');
  const [module, setModule] = useState('');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50); // 默认每页 50 条
  const [showFilters, setShowFilters] = useState(true); // 默认展开筛选

  // 切换筛选条件时重置页码
  const handleLevelChange = (newLevel: string) => {
    setLevel(newLevel === 'all' ? '' : newLevel);
    setPage(1); // 重置页码
  };

  const handleFileChange = (newFile: string) => {
    setSelectedFile(newFile);
    setPage(1); // 重置页码
  };

  // 切换每页条数时重置页码
  const handlePageSizeChange = (newSize: string) => {
    setPageSize(Number(newSize));
    setPage(1); // 重置页码
  };

  // 加载日志文件列表
  const loadLogFiles = useCallback(async () => {
    try {
      const response = await api.get<{ success: boolean; data: LogFile[] }>('/logs/files');
      if (response.data) {
        setLogFiles(response.data);
      }
    } catch (err) {
      console.error('加载日志文件列表失败:', err);
    }
  }, []);

  // 加载日志统计（带缓存）
  const loadStats = useCallback(async (forceRefresh = false) => {
    // 检查缓存
    const cached = statsCacheRef.current.get(selectedFile);
    const now = Date.now();
    
    if (!forceRefresh && cached && (now - cached.timestamp) < STATS_CACHE_TTL) {
      // 使用缓存数据
      setStats(cached.data);
      return;
    }

    try {
      const response = await api.get<{ success: boolean; data: LogStats }>('/logs/stats', {
        params: { log_file: selectedFile },
      });
      if (response.data) {
        // 更新缓存
        statsCacheRef.current.set(selectedFile, {
          data: response.data,
          timestamp: now,
        });
        setStats(response.data);
      }
    } catch (err) {
      console.error('加载日志统计失败:', err);
    }
  }, [selectedFile]);

  // 加载日志列表
  const loadLogs = useCallback(async () => {
    try {
      setIsLoading(true);
      const params: Record<string, string | number> = {
        page,
        page_size: pageSize,
        log_file: selectedFile,
      };
      if (level) params.level = level;
      if (module) params.module = module;
      if (keyword) params.keyword = keyword;

      const response = await api.get<{
        success: boolean;
        data: { items: LogEntry[]; total: number };
      }>('/logs', { params });

      if (response.data) {
        setLogs(response.data.items || []);
        setTotal(response.data.total || 0);
      }
    } catch (err) {
      console.error('加载日志失败:', err);
      toast.error('加载日志失败');
    } finally {
      setIsLoading(false);
    }
  }, [page, pageSize, selectedFile, level, module, keyword]);

  // 初始加载
  useEffect(() => {
    loadLogFiles();
  }, [loadLogFiles]);

  // 加载日志和统计
  useEffect(() => {
    loadLogs();
    loadStats();
  }, [loadLogs, loadStats]);

  // 下载日志
  const handleDownload = async () => {
    try {
      const response = await api.get('/logs/download', {
        params: { log_file: selectedFile },
        responseType: 'blob',
      });
      const url = window.URL.createObjectURL(new Blob([response as unknown as BlobPart]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `${selectedFile}_${new Date().toISOString().slice(0, 10)}.log`);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      toast.success('日志下载成功');
    } catch (err) {
      console.error('下载日志失败:', err);
      toast.error('下载日志失败');
    }
  };

  // 清空日志
  const handleClear = async () => {
    if (!confirm('确定要清空日志吗？此操作不可恢复。')) {
      return;
    }
    try {
      await api.post('/logs/clear', null, {
        params: { log_file: selectedFile },
      });
      toast.success('日志已清空');
      // 清空该文件的统计缓存
      statsCacheRef.current.delete(selectedFile);
      loadLogs();
      loadStats(true);
    } catch (err) {
      console.error('清空日志失败:', err);
      toast.error('清空日志失败');
    }
  };

  // 刷新（强制刷新统计缓存）
  const handleRefresh = () => {
    loadLogs();
    loadStats(true); // 强制刷新统计
  };

  // 搜索
  const handleSearch = () => {
    setPage(1);
    loadLogs();
  };

  // 重置筛选
  const handleReset = () => {
    setLevel('');
    setModule('');
    setKeyword('');
    setPage(1);
  };

  // 切换行展开
  const toggleRow = (index: number) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedRows(newExpanded);
  };

  // 格式化文件大小
  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  };

  // 计算总页数
  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-7xl p-6">
        {/* 头部 */}
        <div className="mb-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <FileText className="h-8 w-8 text-muted-foreground" />
              <div>
                <h1 className="text-3xl font-bold">系统日志</h1>
                <p className="text-muted-foreground">查看和管理系统运行日志</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoading}>
                <RefreshCw className={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
                刷新
              </Button>
              <Button variant="outline" size="sm" onClick={handleDownload}>
                <Download className="mr-2 h-4 w-4" />
                下载
              </Button>
              <Button variant="outline" size="sm" onClick={handleClear}>
                <Trash2 className="mr-2 h-4 w-4" />
                清空
              </Button>
            </div>
          </div>
        </div>

        {/* 统计和筛选区域 - 紧凑布局 */}
        <Card className="mb-4">
          <CardContent className="py-3 px-4">
            {/* 统计信息 - 单行紧凑显示 */}
            {stats && (
              <div className="flex items-center gap-4 flex-wrap mb-3">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-muted-foreground">总计</span>
                  <span className="font-semibold text-sm">{stats.total}</span>
                </div>
                <div className="h-4 w-px bg-border" />
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-slate-500">DEBUG</span>
                  <span className="font-semibold text-sm text-slate-600">{stats.debug}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-blue-500">INFO</span>
                  <span className="font-semibold text-sm text-blue-600">{stats.info}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-amber-500">WARN</span>
                  <span className="font-semibold text-sm text-amber-600">{stats.warn}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-red-500">ERROR</span>
                  <span className="font-semibold text-sm text-red-600">{stats.error}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-purple-500">FATAL</span>
                  <span className="font-semibold text-sm text-purple-600">{stats.fatal}</span>
                </div>
              </div>
            )}

            {/* 筛选区域 - 紧凑单行 */}
            <div className="flex items-center gap-3 flex-wrap">
              {/* 日志文件选择 */}
              <Select value={selectedFile} onValueChange={handleFileChange}>
                <SelectTrigger className="w-36 h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {logFiles.map((file) => (
                    <SelectItem key={file.name} value={file.name}>
                      {file.display_name} ({formatSize(file.size)})
                    </SelectItem>
                  ))}
                  {logFiles.length === 0 && (
                    <>
                      <SelectItem value="backend">后端日志</SelectItem>
                      <SelectItem value="frontend">前端日志</SelectItem>
                    </>
                  )}
                </SelectContent>
              </Select>

              <div className="h-4 w-px bg-border" />

              {/* 筛选控件 */}
              <Collapsible open={showFilters} className="flex-1">
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 px-2"
                    onClick={() => setShowFilters(!showFilters)}
                  >
                    <Filter className="h-3.5 w-3.5 mr-1" />
                    <span className="text-xs">筛选</span>
                    {showFilters ? (
                      <ChevronUp className="ml-1 h-3.5 w-3.5" />
                    ) : (
                      <ChevronDown className="ml-1 h-3.5 w-3.5" />
                    )}
                  </Button>

                  <CollapsibleContent className="flex items-center gap-2 flex-wrap">
                    <Select value={level || 'all'} onValueChange={handleLevelChange}>
                      <SelectTrigger className="w-24 h-8 text-xs">
                        <SelectValue placeholder="级别" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">全部级别</SelectItem>
                        <SelectItem value="DEBUG">DEBUG</SelectItem>
                        <SelectItem value="INFO">INFO</SelectItem>
                        <SelectItem value="WARN">WARN</SelectItem>
                        <SelectItem value="ERROR">ERROR</SelectItem>
                        <SelectItem value="FATAL">FATAL</SelectItem>
                      </SelectContent>
                    </Select>

                    <Input
                      placeholder="模块"
                      value={module}
                      onChange={(e) => setModule(e.target.value)}
                      className="w-24 h-8 text-xs"
                    />

                    <Input
                      placeholder="关键词搜索"
                      value={keyword}
                      onChange={(e) => setKeyword(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                      className="w-32 h-8 text-xs"
                    />

                    <Button size="sm" className="h-8 px-3" onClick={handleSearch}>
                      <Search className="h-3.5 w-3.5 mr-1" />
                      <span className="text-xs">搜索</span>
                    </Button>

                    <Button variant="outline" size="sm" className="h-8 px-2" onClick={handleReset}>
                      <span className="text-xs">重置</span>
                    </Button>
                  </CollapsibleContent>
                </div>
              </Collapsible>
            </div>
          </CardContent>
        </Card>

        {/* 日志列表 */}
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
                <Select value={String(pageSize)} onValueChange={handlePageSizeChange}>
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
                        onClick={() => toggleRow(index)}
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
                    onClick={() => setPage(1)}
                    disabled={page === 1}
                  >
                    首页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(page - 1)}
                    disabled={page === 1}
                  >
                    上一页
                  </Button>
                  <span className="text-sm px-2">{page}</span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(page + 1)}
                    disabled={page >= totalPages}
                  >
                    下一页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(totalPages)}
                    disabled={page >= totalPages}
                  >
                    末页
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
