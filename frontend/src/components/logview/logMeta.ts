// 日志条目类型
export interface LogEntry {
  timestamp: string;
  level: string;
  module: string;
  message: string;
  fields: string;
  location: string;
  raw: string;
}

// 日志文件信息
export interface LogFile {
  name: string;
  display_name: string;
  size: number;
  modified_at: string;
}

// 日志统计
export interface LogStats {
  debug: number;
  info: number;
  warn: number;
  error: number;
  fatal: number;
  total: number;
}

// 日志级别样式配置

export const levelConfig: Record<string, { bg: string; text: string; border: string; icon: string }> = {
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
export const levelBadgeColors: Record<string, string> = {
  DEBUG: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
  INFO: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  WARN: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300',
  ERROR: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  FATAL: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
};

// 格式化文件大小（自 LogsPage 同位搬移）
export const formatSize = (bytes: number): string => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  };
