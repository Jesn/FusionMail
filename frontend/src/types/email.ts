/**
 * 邮件发送相关类型定义
 */

// 发送邮件请求
export interface SendEmailRequest {
  account_uid: string;           // 发件账户 UID
  to: string[];                  // 收件人列表
  cc?: string[];                 // 抄送列表
  bcc?: string[];                // 密送列表
  subject: string;               // 邮件主题
  text_body?: string;            // 纯文本正文
  html_body?: string;            // HTML 正文
  attachments?: AttachmentInfo[]; // 附件列表
  in_reply_to?: string;          // 回复的邮件 Message-ID
  references?: string[];         // 引用的邮件 Message-ID 列表
}

// 附件信息
export interface AttachmentInfo {
  filename: string;              // 文件名
  content_type: string;          // MIME 类型
  size: number;                  // 文件大小（字节）
  temp_path?: string;            // 临时存储路径（上传后返回）
  data?: string;                 // Base64 编码的文件内容（小文件直接传输）
}

// 发送结果
export interface SendResult {
  success: boolean;              // 是否成功
  message_id?: string;           // 发送成功后的 Message-ID
  error?: string;                // 错误信息
  sent_email_id?: number;        // 已发送邮件记录 ID
}

// 回复邮件请求
export interface ReplyEmailRequest {
  account_uid: string;           // 发件账户 UID
  text_body?: string;            // 纯文本正文
  html_body?: string;            // HTML 正文
  attachments?: AttachmentInfo[]; // 附件列表
}

// 转发邮件请求
export interface ForwardEmailRequest {
  account_uid: string;           // 发件账户 UID
  to: string[];                  // 收件人列表
  cc?: string[];                 // 抄送列表
  bcc?: string[];                // 密送列表
  text_body?: string;            // 附加的纯文本正文
  html_body?: string;            // 附加的 HTML 正文
  include_attachments?: boolean; // 是否包含原邮件附件
}

// 已发送邮件
export interface SentEmail {
  id: number;
  account_uid: string;
  message_id: string;
  to_addresses: string;          // JSON 字符串
  cc_addresses?: string;         // JSON 字符串
  bcc_addresses?: string;        // JSON 字符串
  subject: string;
  text_body?: string;
  html_body?: string;
  has_attachments: boolean;
  attachments_count: number;
  status: 'pending' | 'sent' | 'failed';
  error_message?: string;
  sender_type: string;           // 'smtp' | 'gmail_api' | 'graph_api'
  in_reply_to?: string;
  references?: string;
  sent_at?: string;
  created_at: string;
  updated_at: string;
}

// 已发送邮件列表响应
export interface SentEmailListResponse {
  emails: SentEmail[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// 已发送邮件筛选条件
export interface SentEmailFilter {
  account_uid?: string;
  status?: 'pending' | 'sent' | 'failed';
  start_date?: string;
  end_date?: string;
  keyword?: string;
}

// 已发送邮件统计
export interface SentEmailStats {
  total: number;
  sent: number;
  failed: number;
  pending: number;
}

// SMTP 配置（从后端获取的响应）
export interface SMTPConfig {
  smtp_host: string;             // 实际使用的 SMTP 服务器（可能来自 Provider）
  smtp_port: number;             // 实际使用的端口
  smtp_encryption: 'none' | 'tls' | 'starttls' | 'ssl';  // 实际使用的加密方式
  smtp_username: string;
  smtp_password?: string;        // 仅在更新时使用
  smtp_enabled: boolean;
  from_provider?: boolean;       // 服务器配置是否来自 Provider
  provider_name?: string;        // Provider 名称（如果有）
}

// SMTP 配置更新请求
// 注意：host/port/encryption 从 Provider 继承，Account 只需配置用户名和密码
export interface UpdateSMTPConfigRequest {
  smtp_username: string;
  smtp_password: string;
  smtp_enabled: boolean;
}

// SMTP 连接测试结果
export interface SMTPTestResult {
  success: boolean;
  message: string;
  error?: string;
}

// 默认 SMTP 配置（按服务商）
export interface DefaultSMTPConfig {
  provider: string;
  name: string;
  smtp_host: string;
  smtp_port: number;
  smtp_encryption: 'none' | 'tls' | 'starttls';
  note?: string;
}

// 附件上传响应
export interface AttachmentUploadResponse {
  filename: string;
  content_type: string;
  size: number;
  temp_path: string;
}
