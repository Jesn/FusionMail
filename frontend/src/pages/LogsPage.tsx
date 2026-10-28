import { useState, useEffect, useCallback, useRef } from 'react';
import { FileText, Download, Trash2, RefreshCw } from 'lucide-react';
import { Button } from '../components/ui/button';
import { toast } from 'sonner';
import { api } from '../services/api';
import { cn } from '../lib/utils';
import { LOG_STATS_CACHE_TTL } from '../utils/cachePolicy';
import { LogsFilterBar } from '../components/logview/LogsFilterBar';
import { LogListCard } from '../components/logview/LogListCard';
import type { LogEntry, LogFile, LogStats } from '../components/logview/logMeta';

interface StatsCache {
  data: LogStats;
  timestamp: number;
}

// 缓存有效期统一由 utils/cachePolicy.ts 声明

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
    
    if (!forceRefresh && cached && (now - cached.timestamp) < LOG_STATS_CACHE_TTL) {
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

  // 加载日志和统计（串行：先加载日志列表，再加载统计，避免同时全量读取同一文件）
  useEffect(() => {
    loadLogs().then(() => loadStats());
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

        <LogsFilterBar
          stats={stats}
          level={level}
          onLevelChange={handleLevelChange}
          module={module}
          onModuleChange={setModule}
          keyword={keyword}
          onKeywordChange={setKeyword}
          selectedFile={selectedFile}
          onFileChange={handleFileChange}
          logFiles={logFiles}
          showFilters={showFilters}
          onShowFiltersChange={setShowFilters}
          onSearch={handleSearch}
          onReset={handleReset}
        />

        <LogListCard
          logs={logs}
          isLoading={isLoading}
          expandedRows={expandedRows}
          onToggleRow={toggleRow}
          page={page}
          onPageChange={setPage}
          pageSize={pageSize}
          onPageSizeChange={handlePageSizeChange}
          totalPages={totalPages}
          total={total}
        />

      </div>
    </div>
  );
};
