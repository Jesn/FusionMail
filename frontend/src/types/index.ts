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
  deleted_at?: string;
  
  // 新增：Provider 和 Adapter 关联字段
  provider_id?: number;           // 关联的 Provider ID
  adapter_id?: number;            // 关联的 Adapter ID
  provider_ref?: import('./provider').Provider;  // 预加载的 Provider 信息
  adapter_ref?: import('./adapter').Adapter;     // 预加载的 Adapter 信息
  
  // 分组字段
  group_id?: number | null;
  // 通用邮箱配置字段（保留用于向后兼容）
  imap_host?: string;
  imap_port?: number;
  pop3_host?: string;
  pop3_port?: number;
  encryption?: string;
  // 统计信息（可选，用于前端展示）
  unread_count?: number;
  total_count?: number;
  starred_count?: number;
  // 自动禁用相关字段（用于短期邮箱过期处理）
  consecutive_auth_failures: number;
  auto_disabled_at?: string;
  disable_reason?: string;
  // 删除策略配置
  server_delete_policy?: string; // 'off' 或 'soft'
  // 首次同步优化配置字段
  first_sync_days?: number;      // 首次同步天数（0 表示全量同步）
  batch_size?: number;           // 批次大小
  max_emails_per_sync?: number;  // 单次同步最大邮件数
}

// 同步进度类型
export interface SyncProgress {
  account_uid: string;
  status: 'started' | 'in_progress' | 'completed' | 'failed' | 'cancelled';
  phase: 'fetching' | 'processing' | 'finalizing';
  total_estimated: number;
  processed: number;
  new_emails: number;
  updated_emails: number;
  failed_emails: number;
  current_batch: number;
  total_batches: number;
  is_first_sync: boolean;
  started_at: string;
  last_update_at: string;
  error_message?: string;
}

export interface AccountStats {
  account_uid: string;
  total_count: number;
  unread_count: number;
  starred_count: number;
}

// 扩展 Account 类型，添加统计信息
export interface AccountWithStats extends Account {
  unread_count?: number;
  total_count?: number;
  starred_count?: number;
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
  snippet?: string;
  is_read: boolean;
  is_starred: boolean;
  is_archived: boolean;
  is_deleted: boolean;
  is_spam: boolean;
  spam_score?: number;
  spam_confidence?: number;
  spam_reason?: string;
  spam_detected_at?: string;
  spam_detected_by?: string;
  user_marked_spam?: boolean;
  user_marked_at?: string;
  has_attachments: boolean;
  attachments_count: number;
  labels?: string;
  sent_at: string;
  received_at: string;
  created_at: string;
  updated_at: string;
}

export interface EmailDetail extends Email {
  text_body?: string;
  html_body?: string;
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

  // CID 支持 - Content-ID 引用
  content_id?: string;

  // 邮件头信息（可选）
  headers?: Record<string, string>;

  // 附件数据（可选，用于内嵌图片）
  data?: string;

  // 附件内容（可选）
  content?: string;
}

export interface EmailFilter {
  account_uid?: string;
  group_id?: number; // 分组 ID：-1 表示所有账号，0 表示未分组，>0 表示具体分组
  is_read?: boolean;
  is_starred?: boolean;
  is_archived?: boolean;
  is_deleted?: boolean;
  is_spam?: boolean;
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
  match_mode: 'all' | 'any'; // 匹配模式：all(所有条件) 或 any(任意条件)
  conditions: string | null; // JSON 字符串，可能为 null（兼容性）
  actions: string | null; // JSON 字符串，可能为 null（兼容性）
  priority: number;
  stop_processing: boolean;
  enabled: boolean;
  matched_count: number; // API 字段名
  last_matched_at?: string; // API 字段名
  created_at: string;
  updated_at: string;
}

export interface RuleCondition {
  field: 'from' | 'to' | 'subject' | 'body' | 'has_attachment';
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

// API Key 类型
export interface APIKey {
  id: number;
  name: string;
  description: string;
  rate_limit: number;
  enabled: boolean;
  total_requests: number;
  last_used_at: string | null;
  created_at: string;
  expires_at: string | null;
}

export interface CreateAPIKeyRequest {
  name: string;
  description: string;
  rate_limit: number;
  expires_at?: string | null;
}

export interface CreateAPIKeyResponse {
  api_key: string; // 明文 Key，仅此一次返回
  key_info: APIKey;
}

export interface UpdateAPIKeyRequest {
  name: string;
  description: string;
  rate_limit: number;
}

// OAuth2 客户端配置类型
export type {
  OAuth2Client,
  OAuth2ClientCreateRequest,
  OAuth2ClientUpdateRequest,
  OAuth2ClientListResponse,
  OAuth2ClientApiResponse,
  OAuth2ClientSmartSelectParams,
  OAuth2ClientSmartSelectResponse,
} from './oauth2';

// Provider 邮箱提供商配置类型
export type {
  Provider,
  ProviderCreateRequest,
  ProviderUpdateRequest,
  ProviderListResponse,
  ProviderApiResponse,
} from './provider';

// Adapter 适配器类型
export type {
  Adapter,
  AdapterResponse,
  AdapterListResponse,
  AdapterApiResponse,
  ProviderAdapter,
  AuthType,
  AdapterName,
} from './adapter';

export {
  AdapterNames,
  AdapterDisplayNames,
  getAdapterDisplayName,
  isOAuth2Adapter,
  isOAuth2AdapterName,
} from './adapter';

// 账号分组类型
export interface AccountGroup {
  id: number;
  name: string;
  description: string;
  display_order: number;
  created_at: string;
  updated_at: string;
}

// 带账号数量的分组
export interface AccountGroupWithCount extends AccountGroup {
  account_count: number;
}

// 带账号列表的分组详情
export interface AccountGroupWithAccounts extends AccountGroup {
  accounts: Account[];
}

// 创建分组请求
export interface CreateGroupRequest {
  name: string;
  description?: string;
}

// 更新分组请求
export interface UpdateGroupRequest {
  name: string;
  description?: string;
}

// 分配账号到分组请求
export interface AssignAccountToGroupRequest {
  group_id: number | null;
}

// 批量分配账号请求
export interface BatchAssignAccountsRequest {
  account_uids: string[];
  group_id: number | null;
}

// 重排序分组请求
export interface ReorderGroupsRequest {
  group_ids: number[];
}

// 邮件发送相关类型
export type {
  SendEmailRequest,
  AttachmentInfo,
  SendResult,
  ReplyEmailRequest,
  ForwardEmailRequest,
  SentEmail,
  SentEmailListResponse,
  SentEmailFilter,
  SentEmailStats,
  SMTPConfig,
  UpdateSMTPConfigRequest,
  SMTPTestResult,
  DefaultSMTPConfig,
  AttachmentUploadResponse,
} from './email';

