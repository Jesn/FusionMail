import { useEffect } from 'react';
import { Folder, X } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { useGroupStore } from '../../stores/groupStore';

interface GroupSelectorProps {
  /** 当前选中的分组 ID */
  value: number | null | undefined;
  /** 选择变化的回调 */
  onChange: (groupId: number | null) => void;
  /** 是否禁用 */
  disabled?: boolean;
  /** 占位符文本 */
  placeholder?: string;
  /** 是否显示清除按钮 */
  showClear?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 分组选择器组件
 * 用于在账号表单中选择分组
 */
export const GroupSelector = ({
  value,
  onChange,
  disabled = false,
  placeholder = '选择分组',
  showClear = true,
  className,
}: GroupSelectorProps) => {
  const { groups, fetchGroups, isLoading } = useGroupStore();

  // 组件挂载时获取分组列表
  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  const handleValueChange = (val: string) => {
    if (val === 'none') {
      onChange(null);
    } else {
      onChange(parseInt(val, 10));
    }
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    onChange(null);
  };

  // 将 value 转换为字符串用于 Select 组件
  const selectValue = value === null || value === undefined ? 'none' : String(value);

  return (
    <div className={className}>
      <Select
        value={selectValue}
        onValueChange={handleValueChange}
        disabled={disabled || isLoading}
      >
        <SelectTrigger className="w-full">
          <div className="flex items-center gap-2 flex-1">
            <Folder className="h-4 w-4 text-muted-foreground" />
            <SelectValue placeholder={placeholder} />
          </div>
          {/* 使用 span 而非 Button，避免 button 嵌套 button 的 HTML 错误 */}
          {showClear && value !== null && value !== undefined && (
            <span
              role="button"
              tabIndex={0}
              className="h-4 w-4 ml-auto inline-flex items-center justify-center rounded-sm hover:bg-accent hover:text-accent-foreground cursor-pointer"
              onClick={handleClear}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  handleClear(e as unknown as React.MouseEvent);
                }
              }}
            >
              <X className="h-3 w-3" />
            </span>
          )}
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">
            <span className="text-muted-foreground">未分组</span>
          </SelectItem>
          {groups.map((group) => (
            <SelectItem key={group.id} value={String(group.id)}>
              {group.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};
