/**
 * 配置分类常量定义
 * 统一管理所有配置分类的元数据
 */

import { Settings, Shield, Bell, Zap, Key, Mail } from 'lucide-react';
import type { CategoryMeta } from '../types/settings';

/**
 * 用户级配置分类
 */
export const USER_CATEGORIES_META: Record<string, CategoryMeta> = {
  ui: {
    displayName: '界面设置',
    description: '自定义您的邮箱界面体验',
    icon: <Settings className="h-5 w-5" />,
    color: 'bg-blue-100 text-blue-700',
  },
  sync: {
    displayName: '同步设置',
    description: '配置邮件同步和检查频率',
    icon: <Zap className="h-5 w-5" />,
    color: 'bg-green-100 text-green-700',
  },
  notification: {
    displayName: '通知设置',
    description: '管理邮件和桌面通知偏好',
    icon: <Bell className="h-5 w-5" />,
    color: 'bg-purple-100 text-purple-700',
  },
} as const;

/**
 * 系统级配置分类（管理员）
 */
export const ADMIN_CATEGORIES_META: Record<string, CategoryMeta> = {
  ...USER_CATEGORIES_META,
  security: {
    displayName: '安全设置',
    description: '认证、授权和加密配置',
    icon: <Shield className="h-5 w-5" />,
    color: 'bg-red-100 text-red-700',
  },
  api: {
    displayName: 'API设置',
    description: 'API调用和速率限制配置',
    icon: <Settings className="h-5 w-5" />,
    color: 'bg-orange-100 text-orange-700',
  },
  oauth: {
    displayName: 'OAuth设置',
    description: '第三方登录认证配置',
    icon: <Key className="h-5 w-5" />,
    color: 'bg-yellow-100 text-yellow-700',
  },
  smtp: {
    displayName: 'SMTP设置',
    description: '邮件发送服务器配置',
    icon: <Mail className="h-5 w-5" />,
    color: 'bg-indigo-100 text-indigo-700',
  },
} as const;

/**
 * 用户级配置分类列表
 */
export const USER_CATEGORIES = Object.keys(USER_CATEGORIES_META) as Array<
  keyof typeof USER_CATEGORIES_META
>;

/**
 * 管理员配置分类列表
 */
export const ADMIN_CATEGORIES = Object.keys(ADMIN_CATEGORIES_META) as Array<
  keyof typeof ADMIN_CATEGORIES_META
>;

/**
 * 获取分类元数据
 */
export function getCategoryMeta(category: string, isAdmin: boolean = false) {
  const meta = isAdmin ? ADMIN_CATEGORIES_META : USER_CATEGORIES_META;
  return meta[category as keyof typeof meta];
}

/**
 * 检查是否为有效的分类
 */
export function isValidCategory(category: string, isAdmin: boolean = false): boolean {
  const categories = isAdmin ? ADMIN_CATEGORIES : USER_CATEGORIES;
  return categories.includes(category as any);
}

/**
 * 获取分类显示名称
 */
export function getCategoryDisplayName(category: string, isAdmin: boolean = false): string {
  const meta = getCategoryMeta(category, isAdmin);
  return meta?.displayName || category;
}

/**
 * 获取分类描述
 */
export function getCategoryDescription(category: string, isAdmin: boolean = false): string {
  const meta = getCategoryMeta(category, isAdmin);
  return meta?.description || '';
}

/**
 * 获取分类图标
 */
export function getCategoryIcon(category: string, isAdmin: boolean = false) {
  const meta = getCategoryMeta(category, isAdmin);
  return meta?.icon || <Settings className="h-5 w-5" />;
}

/**
 * 获取分类颜色
 */
export function getCategoryColor(category: string, isAdmin: boolean = false): string {
  const meta = getCategoryMeta(category, isAdmin);
  return meta?.color || 'bg-gray-100 text-gray-700';
}
