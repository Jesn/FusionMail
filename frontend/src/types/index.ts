// 账户类型
export interface Account {
  id: number;
  uid: string;
  email: string;
  provider: string;
  protocol: string;
  auth_type: string;
  sync_enabled: boolean;
  sync_interval: number;
  last_sync_at?: string;
  last_sync_status?: string;
  last_sync_error?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface AccountStats {
  account_uid: string;
  total_count: number;
  unread_count: number;
  starred_count: number;
}

// 邮件类型
export interface Email {
  id: number;
  provider_id: string;
  account_uid: string;
  message_id: string;
  thread_id?: string;
  from_address: string;
  from_name?: string;
  to_addresses: string;
  cc_addresses?: string;
  bcc_addresses?: string;
  subject: string;
  text_body?: string;
  html_body?: string;
  snippet?: string;
  is_read: boolean;
  is_starred: boolean;
  is_archived: boolean;
  is_deleted: boolean;
  has_attachments: boolean;
  attachments_count: number;
  labels?: string;
  sent_at: string;
  received_at: string;
  created_at: string;
  updated_at: string;
  attachments?: EmailAttachment[];
}

export interface EmailAttachment {
  id: number;
  email_id: number;
  filename: string;
  content_type: string;
  size: number;
  size_bytes: number;
  attachment_id: string;
  storage_path: string;
  download_url?: string;
}

export interface EmailFilter {
  account_uid?: string;
  is_read?: boolean;
  is_starred?: boolean;
  is_archived?: boolean;
  from_address?: string;
  subject?: string;
  start_date?: string;
  end_date?: string;
}

export interface EmailListResponse {
  emails: Email[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

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
