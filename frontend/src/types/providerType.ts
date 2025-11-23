// ProviderType 常量（与后端 ProviderType 保持一致）
export const ProviderType = {
  Gmail: 1,
  Outlook: 2,
  Icloud: 3,
  QQ: 4,
  Email163: 5,
  Generic: 6,
} as const;

// 类型定义
export type ProviderType = typeof ProviderType[keyof typeof ProviderType];

// ProviderType 映射表
export const ProviderTypeMap: Record<ProviderType, string> = {
  [ProviderType.Gmail]: 'Gmail',
  [ProviderType.Outlook]: 'Outlook / Hotmail',
  [ProviderType.Icloud]: 'iCloud Mail',
  [ProviderType.QQ]: 'QQ 邮箱',
  [ProviderType.Email163]: '163 邮箱',
  [ProviderType.Generic]: '通用邮箱',
};

// 从 ProviderType 获取显示名称
export const getProviderTypeDisplayName = (providerType: ProviderType | number): string => {
  const type = typeof providerType === 'number' ? providerType : providerType;
  return ProviderTypeMap[type as ProviderType] || `未知类型 (${type})`;
};

// 从 provider_name 获取 provider_type
export const getProviderTypeFromName = (providerName: string): ProviderType => {
  const map: Record<string, ProviderType> = {
    gmail: ProviderType.Gmail,
    outlook: ProviderType.Outlook,
    icloud: ProviderType.Icloud,
    qq: ProviderType.QQ,
    '163': ProviderType.Email163,
    generic: ProviderType.Generic,
  };
  return map[providerName] || ProviderType.Generic;
};

// ProviderType 工具函数
export const ProviderTypeUtils = {
  // 判断是否支持 OAuth2 的提供商类型
  supportsOAuth2: (providerType: ProviderType): boolean => {
    return providerType === ProviderType.Gmail || providerType === ProviderType.Outlook;
  },

  // 判断是否支持批量导入的提供商类型
  supportsBatchImport: (providerType: ProviderType): boolean => {
    return providerType === ProviderType.Outlook;
  },

  // 获取 OAuth2 提供商的认证路径
  getOAuth2Provider: (providerType: ProviderType): string => {
    switch (providerType) {
      case ProviderType.Gmail:
        return 'google';
      case ProviderType.Outlook:
        return 'microsoft';
      default:
        return providerType.toString();
    }
  }
};
