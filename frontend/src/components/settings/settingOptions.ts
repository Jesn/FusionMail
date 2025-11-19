/**
 * 配置项选项映射
 * 为不同配置项提供预定义的选项和约束
 */

import type { SettingFieldConfig } from '../../types/settings';

// 配置项选项映射
export const SETTING_OPTIONS: Record<string, SettingFieldConfig> = {
  // UI设置
  theme: {
    type: 'select',
    options: [
      { value: 'light', label: '浅色模式', description: '明亮的浅色主题' },
      { value: 'dark', label: '深色模式', description: '舒适的深色主题' },
      { value: 'system', label: '跟随系统', description: '根据系统设置自动切换' },
    ],
    placeholder: '选择主题',
  },
  language: {
    type: 'select',
    options: [
      { value: 'zh-CN', label: '简体中文', description: '中文（简体）' },
      { value: 'zh-TW', label: '繁体中文', description: '中文（繁体）' },
      { value: 'en-US', label: 'English', description: '英文（美国）' },
      { value: 'ja-JP', label: '日本語', description: '日语' },
      { value: 'ko-KR', label: '한국어', description: '韩语' },
    ],
    placeholder: '选择语言',
  },
  default_view: {
    type: 'select',
    options: [
      { value: 'compact', label: '紧凑视图', description: '显示更多邮件' },
      { value: 'comfortable', label: '舒适视图', description: '平衡显示' },
      { value: 'spacious', label: '宽松视图', description: '更宽松的间距' },
    ],
    placeholder: '选择默认视图',
  },
  email_page_size: {
    type: 'number',
    numberRange: { min: 10, max: 100, step: 10, suffix: ' 封/页' },
    placeholder: '每页邮件数量',
  },

  // 同步设置
  enable_auto_sync: {
    type: 'switch',
    placeholder: '是否启用自动同步',
  },
  sync_interval: {
    type: 'number',
    numberRange: { min: 60, max: 3600, step: 60, suffix: ' 秒' },
    placeholder: '同步间隔（秒）',
  },
  max_concurrent_syncs: {
    type: 'number',
    numberRange: { min: 1, max: 20, step: 1, suffix: ' 个' },
    placeholder: '最大并发同步数',
  },

  // 通知设置
  enable_desktop_notification: {
    type: 'switch',
    placeholder: '是否启用桌面通知',
  },
  enable_email_push: {
    type: 'switch',
    placeholder: '是否启用邮件推送',
  },

  // 安全设置
  password_complexity: {
    type: 'switch',
    placeholder: '是否启用密码复杂度要求',
  },
  login_max_attempts: {
    type: 'number',
    numberRange: { min: 3, max: 10, step: 1, suffix: ' 次' },
    placeholder: '最大登录尝试次数',
  },
  session_timeout: {
    type: 'number',
    numberRange: { min: 5, max: 1440, step: 5, suffix: ' 分钟' },
    placeholder: '会话超时时间',
  },
  jwt_expiry: {
    type: 'number',
    numberRange: { min: 1, max: 168, step: 1, suffix: ' 小时' },
    placeholder: 'JWT令牌过期时间',
  },

  // OAuth配置（敏感）
  gmail_client_id: {
    type: 'input',
    placeholder: 'Gmail客户端ID',
  },
  gmail_client_secret: {
    type: 'password',
    placeholder: 'Gmail客户端密钥',
  },
  microsoft_client_id: {
    type: 'input',
    placeholder: 'Microsoft客户端ID',
  },
  microsoft_client_secret: {
    type: 'password',
    placeholder: 'Microsoft客户端密钥',
  },

  // API配置
  rate_limit_enabled: {
    type: 'switch',
    placeholder: '是否启用API速率限制',
  },
  rate_limit_site: {
    type: 'number',
    numberRange: { min: 60, max: 10000, step: 10, suffix: ' 次/分钟' },
    placeholder: '站点API限制',
  },
  rate_limit_public: {
    type: 'number',
    numberRange: { min: 10, max: 1000, step: 10, suffix: ' 次/分钟' },
    placeholder: '公开API限制',
  },

  // SMTP配置
  smtp_host: {
    type: 'input',
    placeholder: 'SMTP服务器地址',
  },
  smtp_port: {
    type: 'number',
    numberRange: { min: 1, max: 65535, step: 1, suffix: '' },
    placeholder: 'SMTP端口',
  },
  smtp_username: {
    type: 'input',
    placeholder: 'SMTP用户名',
  },
  smtp_password: {
    type: 'password',
    placeholder: 'SMTP密码',
  },
  smtp_from: {
    type: 'input',
    placeholder: '发件人邮箱地址',
  },
  smtp_from_name: {
    type: 'input',
    placeholder: '发件人名称',
  },

  // 密钥类（敏感）
  jwt_secret: {
    type: 'password',
    placeholder: 'JWT签名密钥',
  },
  master_password: {
    type: 'password',
    placeholder: '主密码',
  },
};

/**
 * 根据配置键获取UI配置
 */
export function getFieldConfig(key: string): SettingFieldConfig | undefined {
  return SETTING_OPTIONS[key];
}

/**
 * 判断是否为敏感配置
 */
export function isSensitiveKey(key: string): boolean {
  return SETTING_OPTIONS[key]?.type === 'password' || key.includes('password') || key.includes('secret');
}

/**
 * 判断是否为开关类型配置
 */
export function isSwitchKey(key: string): boolean {
  return key.startsWith('enable') || key.startsWith('disabled');
}

/**
 * 判断是否为数字类型配置
 */
export function isNumberKey(key: string): boolean {
  return SETTING_OPTIONS[key]?.type === 'number' ||
         key.includes('interval') ||
         key.includes('timeout') ||
         key.includes('size') ||
         key.includes('count') ||
         key.includes('attempts') ||
         key.includes('port') ||
         key === 'email_page_size';
}

/**
 * 判断是否为选择类型配置
 */
export function isSelectKey(key: string): boolean {
  return SETTING_OPTIONS[key]?.type === 'select' ||
         ['theme', 'language', 'default_view'].includes(key);
}
