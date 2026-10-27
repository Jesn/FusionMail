import { Filter, ChevronDown, ChevronUp, Search } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Card, CardContent } from '../ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Collapsible, CollapsibleContent } from '../ui/collapsible';
import { formatSize } from './logMeta';
import type { LogFile, LogStats } from './logMeta';

interface LogsFilterBarProps {
  stats: LogStats | null;
  level: string;
  onLevelChange: (v: string) => void;
  module: string;
  onModuleChange: (v: string) => void;
  keyword: string;
  onKeywordChange: (v: string) => void;
  selectedFile: string;
  onFileChange: (v: string) => void;
  logFiles: LogFile[];
  showFilters: boolean;
  onShowFiltersChange: (v: boolean) => void;
  onSearch: () => void;
  onReset: () => void;
}

/** 日志统计与筛选卡片（自 LogsPage 同位搬移）。
 * 纯展示：不请求、不碰 store；类型与 formatSize 自 logMeta 导入。 */
export function LogsFilterBar({
  stats,
  level,
  onLevelChange,
  module,
  onModuleChange,
  keyword,
  onKeywordChange,
  selectedFile,
  onFileChange,
  logFiles,
  showFilters,
  onShowFiltersChange,
  onSearch,
  onReset,
}: LogsFilterBarProps) {
  return (
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
              <Select value={selectedFile} onValueChange={onFileChange}>
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
                    onClick={() => onShowFiltersChange(!showFilters)}
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
                    <Select value={level || 'all'} onValueChange={onLevelChange}>
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
                      onChange={(e) => onModuleChange(e.target.value)}
                      className="w-24 h-8 text-xs"
                    />

                    <Input
                      placeholder="关键词搜索"
                      value={keyword}
                      onChange={(e) => onKeywordChange(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && onSearch()}
                      className="w-32 h-8 text-xs"
                    />

                    <Button size="sm" className="h-8 px-3" onClick={onSearch}>
                      <Search className="h-3.5 w-3.5 mr-1" />
                      <span className="text-xs">搜索</span>
                    </Button>

                    <Button variant="outline" size="sm" className="h-8 px-2" onClick={onReset}>
                      <span className="text-xs">重置</span>
                    </Button>
                  </CollapsibleContent>
                </div>
              </Collapsible>
            </div>
          </CardContent>
        </Card>

  );
}
