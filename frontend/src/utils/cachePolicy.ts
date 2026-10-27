/**
 * 缓存语义统一（P12）—— 全项目唯一的缓存策略声明。
 *
 * 决策：业务数据（账户/分组/邮件/设置）只在内存中缓存，带 TTL，不跨刷新；
 * 只有 UI 偏好（主题、侧栏状态）与登录凭证进 localStorage。
 *
 * 理由：邮件客户端的业务数据天然易变，跨刷新持久化换来的秒级首屏
 * 优势，抵不上陈旧数据风险与 localStorage 配额管理成本（emailCacheStore
 * 曾为此维护整套配额估算 + 清理逻辑）。刷新后由 store 正常拉取。
 *
 * 各 store 的 TTL 语义统一从这里取，禁止再散落魔法数字。
 */

/** 业务数据缓存 TTL（毫秒）：账户列表 */
export const ACCOUNT_CACHE_TTL = 10 * 60 * 1000;
/** 业务数据缓存 TTL（毫秒）：分组列表 */
export const GROUP_CACHE_TTL = 5 * 60 * 1000;

/** 统一的过期判断：无时间戳视为过期 */
export const isCacheExpired = (
  cacheTimestamp: number | null | undefined,
  ttlMs: number
): boolean => {
  if (!cacheTimestamp) return true;
  return Date.now() - cacheTimestamp > ttlMs;
};
