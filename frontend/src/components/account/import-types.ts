export type FieldType = 'email' | 'password' | 'refresh_token' | 'client_id';

export interface ImportFormatConfig {
  delimiter: string;
  fields: FieldType[];
}

export const FIELD_LABELS: Record<FieldType, string> = {
  email: '邮箱',
  password: '密码',
  refresh_token: '刷新令牌',
  client_id: '客户端ID',
};

export const DEFAULT_FORMAT: ImportFormatConfig = {
  delimiter: '----',
  fields: ['email', 'password', 'refresh_token', 'client_id'],
};

export interface DelimiterPreset {
  label: string;
  value: string;
}

export const DELIMITER_PRESETS: DelimiterPreset[] = [
  { label: '---- (四连字符)', value: '----' },
  { label: '| (竖线)', value: '|' },
  { label: ': (冒号)', value: ':' },
  { label: ', (逗号)', value: ',' },
  { label: '\\t (Tab)', value: '\t' },
  { label: '; (分号)', value: ';' },
];

export const FORMAT_STORAGE_KEY = 'fusionmail_batch_import_format';