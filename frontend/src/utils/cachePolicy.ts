/**
 * 缓存语义统一（P12 建立，P20 收敛）—— 全项目唯一的缓存策略声明。
 *
 * 决策（P20，2026-10-28 用户确认，见
 * .trellis/tasks/10-27-2.0-p20-cache-unification/prd.md）：
 *
 *   业务数据（账户/分组/邮件/设置）只在内存中缓存，带 TTL，不跨刷新；
 *   只有三类东西进 localStorage：登录凭证、UI 偏好（主题、侧栏、搜索历史、
 *   导入格式）、以及「设置白名单」中按 key 列举的少量值。
 *
 * 为什么设置白名单只留 email_page_size：stores/emailStore.ts 在模块初始化时
 * **同步**读取它填充 pageSize 初值，而 hooks/useEmails.ts 的 loadEmails
 * useCallback deps 含 pageSize。若不持久化，首屏会先按默认 20 发一次列表请求，
 * setEmails 用服务端 page_size 覆盖后 deps 变化再发一次——多一次请求且条数闪变。
 *
 * 理由：邮件客户端的业务数据天然易变，跨刷新持久化换来的秒级首屏优势，抵不上
 * 陈旧数据风险与 localStorage 配额管理成本（emailCacheStore 曾为此维护整套配额
 * 估算 + 清理逻辑）。刷新后由 store 正常拉取。
 *
 * 本文件是唯一策略源：各 store / hook / service 的 TTL、容量上限、React Query
 * 语义、持久化白名单全部从这里取，禁止再散落魔法数字。
 */

// ---------------------------------------------------------------------------
// 1) 内存 TTL（业务数据，刷新即拉取）
// ---------------------------------------------------------------------------

/** 账户列表 */
export const ACCOUNT_CACHE_TTL = 10 * 60 * 1000;
/** 分组列表 */
export const GROUP_CACHE_TTL = 5 * 60 * 1000;
/** 邮件列表页 */
export const EMAIL_LIST_CACHE_TTL = 5 * 60 * 1000;
/** 邮件详情 */
export const EMAIL_DETAIL_CACHE_TTL = 10 * 60 * 1000;
/** 搜索结果 */
export const SEARCH_CACHE_TTL = 2 * 60 * 1000;
/** 未读邮件数（频繁调用，短缓存） */
export const UNREAD_COUNT_CACHE_TTL = 30 * 1000;
/** 日志统计（LogsPage 内联缓存） */
export const LOG_STATS_CACHE_TTL = 60 * 1000;

/** 内存缓存的定期清理间隔（emailCacheStore 的 cleanup 定时器） */
export const CACHE_CLEANUP_INTERVAL = 60 * 1000;

// ---------------------------------------------------------------------------
// 2) LRU 容量上限（内存缓存条目数）
// ---------------------------------------------------------------------------

/** 邮件列表页最多缓存 5 页 */
export const MAX_EMAIL_CACHE_ENTRIES = 5;
/** 邮件详情最多 20 封 */
export const MAX_EMAIL_DETAIL_CACHE_ENTRIES = 20;
/** 搜索结果最多 10 条 */
export const MAX_SEARCH_CACHE_ENTRIES = 10;

// ---------------------------------------------------------------------------
// 3) React Query 语义（hooks/useSettings.ts）
// ---------------------------------------------------------------------------

/** 按分类取设置 */
export const SETTINGS_STALE_TIME = 5 * 60 * 1000;
export const SETTINGS_GC_TIME = 10 * 60 * 1000;
/** 单个设置项 */
export const SETTINGS_ITEM_STALE_TIME = 5 * 60 * 1000;
/** 公开配置（变化少，缓存更久） */
export const SETTINGS_PUBLIC_STALE_TIME = 10 * 60 * 1000;
/** 分类列表 */
export const SETTINGS_CATEGORIES_STALE_TIME = 30 * 60 * 1000;
/** 设置搜索 */
export const SETTINGS_SEARCH_STALE_TIME = 2 * 60 * 1000;
/** 设置统计（占位数据，短缓存） */
export const SETTINGS_STATS_STALE_TIME = 30 * 1000;

// ---------------------------------------------------------------------------
// 4) 持久化（localStorage）
// ---------------------------------------------------------------------------

/** 设置缓存 TTL：超过则视为过期并清除 */
export const SETTINGS_CACHE_TTL = 30 * 60 * 1000;

/**
 * 允许落盘的设置 key 白名单（按 key 列举，**不按分类**）。
 *
 * 历史上 settingsCache 按整个分类落盘，而 SettingItem 带 isSensitive 字段，
 * 存在敏感值被写入 localStorage 的风险。新增可持久化项必须在此显式登记，
 * 并说明为何需要跨刷新（通常理由：模块初始化时被同步读取）。
 */
export const PERSISTED_SETTING_KEYS = ['email_page_size'] as const;

export type PersistedSettingKey = (typeof PERSISTED_SETTING_KEYS)[number];

/** 判断某个设置 key 是否允许持久化 */
export const isPersistedSettingKey = (key: string): key is PersistedSettingKey =>
  (PERSISTED_SETTING_KEYS as readonly string[]).includes(key);

// ---------------------------------------------------------------------------
// 5) 过期判断
// ---------------------------------------------------------------------------

/** 统一的过期判断：无时间戳视为过期 */
export const isCacheExpired = (
  cacheTimestamp: number | null | undefined,
  ttlMs: number
): boolean => {
  if (!cacheTimestamp) return true;
  return Date.now() - cacheTimestamp > ttlMs;
};
