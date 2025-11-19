/**
 * 配置搜索和筛选组件
 * 提供高级搜索、筛选和排序功能
 */

import { useState, useEffect } from 'react';
import { Input } from '../ui/input';
import { Button } from '../ui/button';
import { Card, CardContent } from '../ui/card';
import { Badge } from '../ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '../ui/popover';
import {
  Search,
  X,
  History,
  SlidersHorizontal,
} from 'lucide-react';

// 筛选条件类型
export interface FilterCriteria {
  query: string;
  category: string;
  valueType: string;
  sensitivity: string; // 'all', 'sensitive', 'public'
  hasDescription: boolean | null;
}

// 配置分类
const CATEGORIES = [
  { value: 'all', label: '全部分类' },
  { value: 'ui', label: '界面设置' },
  { value: 'sync', label: '同步设置' },
  { value: 'notification', label: '通知设置' },
  { value: 'security', label: '安全设置' },
  { value: 'api', label: 'API设置' },
  { value: 'oauth', label: 'OAuth设置' },
  { value: 'smtp', label: 'SMTP设置' },
];

// 配置类型
const VALUE_TYPES = [
  { value: 'all', label: '全部类型' },
  { value: 'string', label: '字符串' },
  { value: 'number', label: '数字' },
  { value: 'boolean', label: '布尔值' },
  { value: 'json', label: 'JSON' },
];

// 敏感度选项
const SENSITIVITY_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'sensitive', label: '仅敏感' },
  { value: 'public', label: '仅公开' },
];

interface SearchAndFilterProps {
  onFilter: (criteria: FilterCriteria) => void;
  isLoading?: boolean;
}

export function SearchAndFilter({ onFilter }: SearchAndFilterProps) {
  const [criteria, setCriteria] = useState<FilterCriteria>({
    query: '',
    category: 'all',
    valueType: 'all',
    sensitivity: 'all',
    hasDescription: null,
  });

  const [showHistory, setShowHistory] = useState(false);
  const [searchHistory, setSearchHistory] = useState<string[]>([]);

  // 从localStorage加载搜索历史
  useEffect(() => {
    const history = localStorage.getItem('settings-search-history');
    if (history) {
      setSearchHistory(JSON.parse(history));
    }
  }, []);

  // 应用筛选
  useEffect(() => {
    onFilter(criteria);
  }, [criteria, onFilter]);

  // 更新筛选条件
  const updateCriteria = (updates: Partial<FilterCriteria>) => {
    setCriteria((prev) => ({ ...prev, ...updates }));
  };

  // 处理搜索
  const handleSearch = (query: string) => {
    // 保存搜索历史
    if (query.trim() && !searchHistory.includes(query)) {
      const newHistory = [query, ...searchHistory.slice(0, 9)]; // 保留最新10条
      setSearchHistory(newHistory);
      localStorage.setItem('settings-search-history', JSON.stringify(newHistory));
    }

    updateCriteria({ query });
  };

  // 清除所有筛选
  const clearAllFilters = () => {
    setCriteria({
      query: '',
      category: 'all',
      valueType: 'all',
      sensitivity: 'all',
      hasDescription: null,
    });
  };

  // 检查是否有活动筛选器
  const hasActiveFilters =
    criteria.category !== 'all' ||
    criteria.valueType !== 'all' ||
    criteria.sensitivity !== 'all' ||
    criteria.hasDescription !== null;

  return (
    <Card>
      <CardContent className="pt-6 space-y-4">
        {/* 搜索栏 */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder="搜索配置项名称、值或描述..."
            value={criteria.query}
            onChange={(e) => handleSearch(e.target.value)}
            className="pl-10 pr-10"
          />
          {criteria.query && (
            <Button
              variant="ghost"
              size="sm"
              className="absolute right-1 top-1/2 transform -translate-y-1/2 h-8 w-8 p-0"
              onClick={() => handleSearch('')}
            >
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>



        {/* 高级筛选器 */}
        <div className="flex flex-wrap items-center gap-3">
          {/* 分类筛选 */}
          <Select
            value={criteria.category}
            onValueChange={(value) => updateCriteria({ category: value })}
          >
            <SelectTrigger className="w-[140px]">
              <SelectValue placeholder="分类" />
            </SelectTrigger>
            <SelectContent>
              {CATEGORIES.map((cat) => (
                <SelectItem key={cat.value} value={cat.value}>
                  {cat.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 类型筛选 */}
          <Select
            value={criteria.valueType}
            onValueChange={(value) => updateCriteria({ valueType: value })}
          >
            <SelectTrigger className="w-[120px]">
              <SelectValue placeholder="类型" />
            </SelectTrigger>
            <SelectContent>
              {VALUE_TYPES.map((type) => (
                <SelectItem key={type.value} value={type.value}>
                  {type.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 敏感度筛选 */}
          <Select
            value={criteria.sensitivity}
            onValueChange={(value) => updateCriteria({ sensitivity: value })}
          >
            <SelectTrigger className="w-[120px]">
              <SelectValue placeholder="敏感度" />
            </SelectTrigger>
            <SelectContent>
              {SENSITIVITY_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 描述筛选 */}
          <Select
            value={criteria.hasDescription === null ? 'all' : criteria.hasDescription.toString()}
            onValueChange={(value) =>
              updateCriteria({
                hasDescription: value === 'all' ? null : value === 'true',
              })
            }
          >
            <SelectTrigger className="w-[120px]">
              <SelectValue placeholder="描述" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="true">有描述</SelectItem>
              <SelectItem value="false">无描述</SelectItem>
            </SelectContent>
          </Select>

          {/* 清除按钮 */}
          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={clearAllFilters}>
              <X className="h-4 w-4 mr-1" />
              清除筛选
            </Button>
          )}

          {/* 搜索历史 */}
          {searchHistory.length > 0 && (
            <Popover open={showHistory} onOpenChange={setShowHistory}>
              <PopoverTrigger asChild>
                <Button variant="ghost" size="sm">
                  <History className="h-4 w-4 mr-1" />
                  历史
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-64 p-2" align="start">
                <div className="space-y-1">
                  <p className="text-sm font-medium mb-2">搜索历史</p>
                  {searchHistory.map((item, index) => (
                    <Button
                      key={index}
                      variant="ghost"
                      className="w-full justify-start text-sm h-8"
                      onClick={() => {
                        handleSearch(item);
                        setShowHistory(false);
                      }}
                    >
                      <History className="h-3 w-3 mr-2 text-muted-foreground" />
                      {item}
                    </Button>
                  ))}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full text-xs text-muted-foreground"
                    onClick={() => {
                      setSearchHistory([]);
                      localStorage.removeItem('settings-search-history');
                      setShowHistory(false);
                    }}
                  >
                    清除历史
                  </Button>
                </div>
              </PopoverContent>
            </Popover>
          )}
        </div>

        {/* 活动筛选器显示 */}
        {hasActiveFilters && (
          <div className="flex items-center gap-2 flex-wrap">
            <SlidersHorizontal className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">活动筛选器:</span>
            {criteria.category !== 'all' && (
              <Badge variant="secondary">
                分类: {CATEGORIES.find((c) => c.value === criteria.category)?.label}
                <X
                  className="ml-1 h-3 w-3 cursor-pointer"
                  onClick={() => updateCriteria({ category: 'all' })}
                />
              </Badge>
            )}
            {criteria.valueType !== 'all' && (
              <Badge variant="secondary">
                类型: {VALUE_TYPES.find((t) => t.value === criteria.valueType)?.label}
                <X
                  className="ml-1 h-3 w-3 cursor-pointer"
                  onClick={() => updateCriteria({ valueType: 'all' })}
                />
              </Badge>
            )}
            {criteria.sensitivity !== 'all' && (
              <Badge variant="secondary">
                敏感度: {SENSITIVITY_OPTIONS.find((s) => s.value === criteria.sensitivity)?.label}
                <X
                  className="ml-1 h-3 w-3 cursor-pointer"
                  onClick={() => updateCriteria({ sensitivity: 'all' })}
                />
              </Badge>
            )}
            {criteria.hasDescription !== null && (
              <Badge variant="secondary">
                {criteria.hasDescription ? '有描述' : '无描述'}
                <X
                  className="ml-1 h-3 w-3 cursor-pointer"
                  onClick={() => updateCriteria({ hasDescription: null })}
                />
              </Badge>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default SearchAndFilter;
