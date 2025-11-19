/**
 * 配置管理类型定义
 * 统一所有配置相关的 TypeScript 类型
 */

// ==================== 基础类型 ====================

/**
 * 配置值类型
 */
export type SettingValueType = 'string' | 'number' | 'boolean' | 'json';

/**
 * 配置项
 */
export interface SettingItem {
  id?: string;
  category: string;
  key: string;
  value: string;
  valueType: SettingValueType;
  isSensitive: boolean;
  isPublic: boolean;
  description?: string;
  updatedAt?: string;
}

/**
 * 配置分类
 */
export interface SettingCategory {
  name: string;
  displayName: string;
  icon: React.ReactNode;
  description?: string;
  color?: string;
  settings?: SettingItem[];
}

/**
 * 配置分类元数据
 */
export interface CategoryMeta {
  displayName: string;
  description: string;
  icon: React.ReactNode;
  color?: string;
}

// ==================== API 请求/响应类型 ====================

/**
 * 获取配置响应
 */
export interface SettingsResponse {
  category: string;
  settings: Record<string, string>;
}

/**
 * 更新配置请求
 */
export interface UpdateSettingRequest {
  category: string;
  key: string;
  value: string;
  isSensitive?: boolean;
  userId?: number;
}

/**
 * 批量更新配置请求
 */
export interface BatchUpdateSettingsRequest {
  category: string;
  settings: Record<string, string>;
  userId?: number;
  isSensitiveMap?: Record<string, boolean>;
}

/**
 * 创建配置请求
 */
export interface CreateSettingRequest {
  category: string;
  key: string;
  value: string;
  valueType?: SettingValueType;
  isSensitive?: boolean;
  isPublic?: boolean;
  description?: string;
}

/**
 * 删除配置请求
 */
export interface DeleteSettingRequest {
  category: string;
  key: string;
  userId?: number;
}

/**
 * 重置配置请求
 */
export interface ResetSettingRequest {
  category: string;
  key: string;
  userId?: number;
}

/**
 * 搜索配置选项
 */
export interface SearchSettingsOptions {
  category?: string;
  onlyPublic?: boolean;
}

/**
 * 导出配置选项
 */
export interface ExportSettingsOptions {
  category?: string;
  includeSensitive?: boolean;
  userId?: number;
  format?: 'json' | 'env';
}

/**
 * 导入配置选项
 */
export interface ImportSettingsOptions {
  category?: string;
  overwrite?: boolean;
  userId?: number;
}

/**
 * 导入配置结果
 */
export interface ImportSettingsResult {
  success: number;
  failed: number;
  errors: string[];
}

// ==================== UI 组件类型 ====================

/**
 * 选择框选项
 */
export interface SelectOption {
  value: string;
  label: string;
  description?: string;
}

/**
 * 数字范围配置
 */
export interface NumberRange {
  min: number;
  max: number;
  step: number;
  suffix?: string;
  prefix?: string;
}

/**
 * 字段类型
 */
export type FieldType = 'select' | 'number' | 'switch' | 'input' | 'password' | 'textarea';

/**
 * 字段配置
 */
export interface SettingFieldConfig {
  type: FieldType;
  options?: SelectOption[];
  numberRange?: NumberRange;
  placeholder?: string;
  validation?: {
    pattern?: RegExp;
    minLength?: number;
    maxLength?: number;
  };
}

/**
 * 筛选条件
 */
export interface FilterCriteria {
  query: string;
  category: 'all' | string;
  valueType: 'all' | SettingValueType;
  sensitivity: 'all' | 'sensitive' | 'non-sensitive';
  hasDescription: boolean | null;
}

// ==================== Hook 返回类型 ====================

/**
 * 配置查询返回类型
 */
export interface UseSettingsReturn {
  settings: Record<string, string> | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

/**
 * 配置更新 Mutation 返回类型
 */
export interface UseSettingsMutationReturn {
  mutate: (vars: UpdateSettingRequest) => void;
  mutateAsync: (vars: UpdateSettingRequest) => Promise<void>;
  isPending: boolean;
  error: Error | null;
}

/**
 * 批量配置更新 Mutation 返回类型
 */
export interface UseBatchSettingsMutationReturn {
  mutate: (vars: BatchUpdateSettingsRequest) => void;
  mutateAsync: (vars: BatchUpdateSettingsRequest) => Promise<void>;
  isPending: boolean;
  error: Error | null;
}

/**
 * 配置删除 Mutation 返回类型
 */
export interface UseDeleteSettingMutationReturn {
  mutate: (vars: DeleteSettingRequest) => void;
  mutateAsync: (vars: DeleteSettingRequest) => Promise<void>;
  isPending: boolean;
  error: Error | null;
}

/**
 * 配置重置 Mutation 返回类型
 */
export interface UseResetSettingMutationReturn {
  mutate: (vars: ResetSettingRequest) => void;
  mutateAsync: (vars: ResetSettingRequest) => Promise<void>;
  isPending: boolean;
  error: Error | null;
}

/**
 * 搜索结果返回类型
 */
export interface UseSearchSettingsReturn {
  results: SettingItem[] | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

/**
 * 分类列表返回类型
 */
export interface UseSettingCategoriesReturn {
  categories: SettingCategory[] | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

// ==================== 缓存统计类型 ====================

/**
 * 缓存统计信息
 */
export interface CacheStats {
  localCache?: {
    hitRate: number;
    size: number;
  };
  redisCache?: {
    hitRate: number;
  };
  totalRequests?: number;
}

/**
 * 缓存统计返回类型
 */
export interface UseGetStatsReturn {
  data: CacheStats | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

// ==================== 组件 Props 类型 ====================

/**
 * SettingItem 组件 Props
 */
export interface SettingItemProps {
  item: {
    key: string;
    value: string;
    valueType: SettingValueType;
    isSensitive?: boolean;
    description?: string;
  };
  onChange: (key: string, value: string) => void;
  onReset?: (key: string) => void;
  isLoading?: boolean;
  isDirty?: boolean;
  error?: string;
  disabled?: boolean;
}

/**
 * SettingsCategory 组件 Props
 */
export interface SettingsCategoryProps {
  name: string;
  displayName: string;
  description?: string;
  icon?: React.ReactNode;
  items: SettingItem[];
  onUpdate: (key: string, value: string) => void;
  onReset?: (key: string) => void;
  isLoading?: boolean;
  isEditable?: boolean;
}

/**
 * SettingsToolbar 组件 Props
 */
export interface SettingsToolbarProps {
  category: string;
  itemCount: number;
  onRefresh?: () => void;
  onExport?: () => void;
  onImport?: () => void;
  isLoading?: boolean;
}

/**
 * SearchAndFilter 组件 Props
 */
export interface SearchAndFilterProps {
  onFilter: (criteria: FilterCriteria) => void;
  isLoading?: boolean;
}

/**
 * ImportExport 组件 Props
 */
export interface ImportExportProps {
  settings: SettingItem[];
  onImport: (data: any) => Promise<void>;
  onExport: () => Promise<any>;
  isLoading?: boolean;
}

// ==================== 常量类型 ====================

/**
 * 配置分类名称
 */
export type CategoryName = 'ui' | 'sync' | 'notification' | 'security' | 'api' | 'oauth' | 'smtp';

/**
 * 配置分类映射
 */
export type CategoriesMetaMap = Record<CategoryName, CategoryMeta>;
