/**
 * API 文档页面静态数据（自 APIDocPage.tsx 机械搬移，数据块逐字符保留）。
 * 纯数据模块：APIEndpoint 接口 + 端点文档数组；curl 示例中的 apiKey/baseUrl 随数据同址解析。
 */

export interface APIEndpoint {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  title: string;
  path: string;
  description: string;
  params?: Array<{
    name: string;
    type: string;
    required: boolean;
    description: string;
  }>;
  bodyParams?: Array<{
    name: string;
    type: string;
    required: boolean;
    description: string;
  }>;
  curlExample: string;
  responseExample: string;
}

/**
 * 构建端点文档数据。函数体逐字符照搬自 APIDocPage.tsx 原组件体（含 apiKey/baseUrl 解析）。
 */
export const buildEndpoints = (): APIEndpoint[] => {
  const apiKey = 'YOUR_API_KEY';
  const baseUrl = import.meta.env.VITE_PUBLIC_API_BASE_URL || 'http://localhost:3333/api/v1/public';

  const endpoints: APIEndpoint[] = [
    {
      method: 'GET',
      title: '实时拉取邮件',
      path: '/mail/receive',
      description: '通过 API Key 实时拉取指定邮箱的邮件列表',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
        { name: 'limit', type: 'int', required: false, description: '返回邮件数量（默认 20，最大 100）' },
        { name: 'offset', type: 'int', required: false, description: '偏移量（默认 0）' },
        { name: 'is_read', type: 'bool', required: false, description: '是否已读' },
        { name: 'is_starred', type: 'bool', required: false, description: '是否星标' },
        { name: 'is_archived', type: 'bool', required: false, description: '是否归档' },
      ],
      curlExample: `curl -X GET "${baseUrl}/mail/receive?email=user@example.com&limit=5" \\
  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "emails": [
      {
        "id": 1,
        "subject": "Hello",
        "from_address": "sender@example.com",
        "from_name": "Sender",
        "to_address": "user@example.com",
        "snippet": "Email content...",
        "is_read": false,
        "is_starred": false,
        "is_archived": false,
        "sent_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 100,
    "limit": 5,
    "offset": 0
  }
}`,
    },
    {
      method: 'GET',
      title: '搜索邮件',
      path: '/mail/search',
      description: '通过 API Key 搜索指定邮箱的邮件',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
        { name: 'q', type: 'string', required: true, description: '搜索关键词' },
        { name: 'limit', type: 'int', required: false, description: '返回邮件数量（默认 20，最大 100）' },
        { name: 'offset', type: 'int', required: false, description: '偏移量（默认 0）' },
      ],
      curlExample: `curl -X GET "${baseUrl}/mail/search?email=user@example.com&q=important" \\
  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "emails": [
      {
        "id": 1,
        "subject": "Important Email",
        "from_address": "sender@example.com",
        "to_address": "user@example.com",
        "snippet": "This is important...",
        "sent_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 5,
    "limit": 10,
    "offset": 0,
    "query": "important"
  }
}`,
    },
    {
      method: 'POST',
      title: '发送邮件',
      path: '/mail/send',
      description: '通过 API Key 使用指定邮箱账户发送邮件。系统会根据 from 字段自动查找对应的邮箱账户。',
      bodyParams: [
        { name: 'from', type: 'string', required: true, description: '发件人邮箱地址（必须是已添加的账户）' },
        { name: 'to', type: 'string[]', required: true, description: '收件人列表' },
        { name: 'cc', type: 'string[]', required: false, description: '抄送列表' },
        { name: 'bcc', type: 'string[]', required: false, description: '密送列表' },
        { name: 'subject', type: 'string', required: false, description: '邮件主题' },
        { name: 'text_body', type: 'string', required: false, description: '纯文本正文' },
        { name: 'html_body', type: 'string', required: false, description: 'HTML 正文' },
        { name: 'reply_to', type: 'string', required: false, description: '回复地址' },
      ],
      curlExample: `curl -X POST "${baseUrl}/mail/send" \\\n  -H "Authorization: Bearer ${apiKey}" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "from": "sender@example.com",\n    "to": ["recipient@example.com"],\n    "subject": "Hello",\n    "text_body": "This is a test email."\n  }'`,
      responseExample: `{
  "success": true,
  "data": {
    "message_id": "<abc123@mail.example.com>",
    "sent_email_id": 42,
    "sender_type": "smtp",
    "provider_msg_id": ""
  }
}`,
    },
    {
      method: 'GET',
      title: '获取邮件详情',
      path: '/mail/detail',
      description: '通过 API Key 获取指定邮件的详细信息，包括纯文本正文、HTML 正文和元数据。仅能访问 email 邮箱对应账户下的邮件。',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
        { name: 'id', type: 'int', required: true, description: '邮件 ID' },
      ],
      curlExample: `curl -X GET "${baseUrl}/mail/detail?email=user@example.com&id=123" \\\n  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "id": 123,
    "subject": "Your code: 654321",
    "from_address": "noreply@github.com",
    "from_name": "GitHub",
    "to_address": "user@example.com",
    "text_body": "Your verification code is 654321",
    "html_body": "<p>Your code is 654321</p>",
    "is_read": false,
    "is_starred": false,
    "sent_at": "2024-03-05T12:18:41Z"
  }
}`,
    },
    {
      method: 'POST',
      title: '标记邮件为已读',
      path: '/mail/mark-read',
      description: '通过 API Key 批量标记邮件为已读（仅本地状态）。',
      bodyParams: [
        { name: 'ids', type: 'int[]', required: true, description: '邮件 ID 列表' },
      ],
      curlExample: `curl -X POST "${baseUrl}/mail/mark-read" \\\n  -H "Authorization: Bearer ${apiKey}" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "ids": [1, 2, 3]\n  }'`,
      responseExample: `{
  "success": true,
  "message": "邮件已标记为已读"
}`,
    },
    {
      method: 'DELETE',
      title: '删除邮件',
      path: '/mail/delete',
      description: '通过 API Key 删除指定邮件（软删除）。仅能删除 email 邮箱对应账户下的邮件。',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
        { name: 'id', type: 'int', required: true, description: '邮件 ID' },
      ],
      curlExample: `curl -X DELETE "${baseUrl}/mail/delete?email=user@example.com&id=123" \\\n  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "message": "Email deleted"
  }
}`,
    },
    {
      method: 'DELETE',
      title: '清空收件箱',
      path: '/mail/clear',
      description: '通过 API Key 清空指定邮箱的所有邮件（软删除）。返回被删除的邮件数量。',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
      ],
      curlExample: `curl -X DELETE "${baseUrl}/mail/clear?email=user@example.com" \\\n  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "count": 42
  }
}`,
    },
    {
      method: 'GET',
      title: '已发送邮件列表',
      path: '/mail/sent',
      description: '通过 API Key 获取指定邮箱的已发送邮件列表，支持分页和筛选。',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
        { name: 'status', type: 'string', required: false, description: '状态筛选（sent/failed）' },
        { name: 'search', type: 'string', required: false, description: '搜索关键词' },
        { name: 'page', type: 'int', required: false, description: '页码（默认 1）' },
        { name: 'page_size', type: 'int', required: false, description: '每页数量（默认 20）' },
      ],
      curlExample: `curl -X GET "${baseUrl}/mail/sent?email=user@example.com&page=1&page_size=10" \\\n  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "emails": [
      {
        "id": 1,
        "subject": "Sent email",
        "from_address": "user@example.com",
        "to_addresses": "[\\"recipient@example.com\\"]",
        "status": "sent",
        "sent_at": "2024-03-05T12:00:00Z"
      }
    ],
    "total": 15,
    "page": 1,
    "page_size": 10
  }
}`,
    },
    {
      method: 'POST',
      title: '批量导入邮箱账户',
      path: '/mail/import-accounts',
      description: '通过 API Key 批量导入 Outlook 邮箱账户。不传 format 时默认格式为 email----password----refresh_token----client_id，单次最多 50 个账户。同步配置使用固定默认值（启用同步、间隔 2 分钟、首次同步 7 天）。',
      bodyParams: [
        { name: 'accounts', type: 'string[]', required: true, description: '账户字符串数组，每条格式如 email----password----refresh_token----client_id' },
        { name: 'format', type: 'object', required: false, description: '格式配置（可选，不传则用默认 ----）。示例：{ delimiter: "----", fields: ["email","password","refresh_token","client_id"] }' },
        { name: 'group_id', type: 'int', required: false, description: '分组 ID（不传则为未分组）' },
      ],
      curlExample: `curl -X POST "${baseUrl}/mail/import-accounts" \\\n  -H "Authorization: Bearer ${apiKey}" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "accounts": [\n      "user1@outlook.com----pass1----token1----client1",\n      "user2@hotmail.com----pass2----token2----client2"\n    ],\n    "group_id": 1\n  }'`,
      responseExample: `{
  "success": true,
  "data": {
    "success": 2,
    "failed": 0,
    "results": [
      {
        "email": "user1@outlook.com",
        "status": "success"
      },
      {
        "email": "user2@hotmail.com",
        "status": "success"
      }
    ]
  }
}`,
    },
    {
      method: 'GET',
      title: '检查邮箱账户是否存在',
      path: '/mail/check-account',
      description: '通过 API Key 检查指定邮箱是否已存在于系统中。存在时返回账户基础信息（不含凭证等敏感字段），不存在时 account 为 null。',
      params: [
        { name: 'email', type: 'string', required: true, description: '要检查的邮箱地址' },
      ],
      curlExample: `curl -X GET "${baseUrl}/mail/check-account?email=user1@outlook.com" \\\n  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "account": {
      "uid": "acc_1730188800_abc123",
      "email": "user1@outlook.com",
      "status": "active",
      "last_sync_at": "2026-09-01T15:58:00+08:00",
      "group_id": 1,
      "group_name": "默认分组",
      "created_at": "2026-09-01T15:50:00+08:00"
    }
  }
}`,
    },
  ];
  return endpoints;
};
