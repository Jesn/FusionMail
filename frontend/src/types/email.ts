/**
 * 邮件发送相关类型定义
 *
 * API 契约类型以 openapi-typescript 生成的 api.generated.ts 为单一来源（backend/docs/swagger.json）。
 */
import type { components } from './api.generated';

type Schemas = components['schemas'];

// 发送邮件请求
//
// 注：保留手写，未改用 handler.SendEmailRequest。字段集合不同：
//  - 本类型多出 in_reply_to/references（后端 Reply/Forward 自动推导，见下方字段注释）
//  - 后端多出 reply_to（backend/internal/handler/send_handler.go:91），本类型尚未声明
//  - attachments 元素类型不同：本类型 AttachmentInfo（5 字段）vs 生成 handler.AttachmentInput（3 字段）
export interface SendEmailRequest {
  account_uid: string;           // 发件账户 UID
  to: string[];                  // 收件人列表
  cc?: string[];                 // 抄送列表
  bcc?: string[];                // 密送列表
  subject: string;               // 邮件主题
  text_body?: string;            // 纯文本正文
  html_body?: string;            // HTML 正文
  attachments?: AttachmentInfo[]; // 附件列表
  // 后端 Reply/Forward 自动从原邮件推导（send_service.go:222），此字段发送时被忽略，仅为类型完备保留
  in_reply_to?: string;          // 回复的邮件 Message-ID
  // 后端 Reply/Forward 自动从原邮件推导（send_service.go:222），此字段发送时被忽略，仅为类型完备保留
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

// 发送结果 —— 与 service.SendEmailResponse 一致，改用生成类型
// （生成类型为超集：额外包含 provider_msg_id、sender_type 字段）
export type SendResult = Schemas['service.SendEmailResponse'];

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
  attachment_count: number;      // 后端 response.SentEmailResponse 单数命名（sent_email.go:35）
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
  search?: string;               // 后端 form 绑定名为 search（sent_email_service.go:31），非 keyword
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
// 注意：SMTP 使用与 IMAP/POP3 相同的邮箱地址和密码，只需配置启用状态
//
// 注：保留手写，未改用 handler.UpdateSMTPConfigRequest（{enabled?: boolean}）。
// 本类型是前端本地形态，字段名 smtp_enabled 与线上 DTO 的 enabled 不同，
// 由 services/smtpService.ts 的 updateConfig 在发送时映射（enabled: config.smtp_enabled），
// 线上契约本身是正确的；生成类型为全可选，改用会丢掉必填语义。
export interface UpdateSMTPConfigRequest {
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
//
// 已收敛到生成类型：字段集合与 response.AttachmentUploadResponse 完全一致，
// 且后端 DTO 已标 validate:"required"（4 字段全部必填），生成类型不再全可选。
export type AttachmentUploadResponse = Schemas['response.AttachmentUploadResponse'];
