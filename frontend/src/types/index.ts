// 导出所有类型定义
export * from '../stores/emailStore';
export * from '../stores/accountStore';

// API 响应类型
export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

// 分页参数
export interface PaginationParams {
  page?: number;
  page_size?: number;
}

// 规则类型
export interface Rule {
  id: number;
  name: string;
  account_uid: string;
  description?: string;
  conditions: string; // JSON 字符串
  actions: string; // JSON 字符串
  priority: number;
  stop_processing: boolean;
  enabled: boolean;
  execution_count: number;
  last_executed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface RuleCondition {
  field: 'from_address' | 'from_name' | 'subject' | 'body' | 'to_addresses';
  operator: 'contains' | 'not_contains' | 'equals' | 'not_equals' | 'starts_with' | 'ends_with';
  value: string;
}

export interface RuleAction {
  type: 'mark_read' | 'mark_unread' | 'star' | 'archive' | 'delete' | 'add_label' | 'trigger_webhook';
  value?: string;
}

// Webhook 类型
export interface Webhook {
  id: number;
  name: string;
  url: string;
  events: string[]; // ['email.received', 'email.read', etc.]
  enabled: boolean;
  secret?: string;
  created_at: string;
  updated_at: string;
}

// 同步日志类型
export interface SyncLog {
  id: number;
  account_uid: string;
  sync_type: string;
  status: string;
  emails_fetched: number;
  emails_new: number;
  emails_updated: number;
  error_message?: string;
  started_at: string;
  completed_at?: string;
  duration_ms: number;
}
