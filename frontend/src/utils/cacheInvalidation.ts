/**
 * 缓存失效入口（P20）—— 全项目唯一的「让缓存作废」通道。
 *
 * 背景：P20 之前全仓没有任何显式失效入口（`grep invalidate` 为 0），唯一的
 * 「强制过期」手段是调用方自己写 `useGroupStore.getState().setCacheTimestamp(0)`，
 * 散落在 hooks/pages 里。本文件把这些动作收敛成具名函数。
 *
 * 语义约定（见 utils/cachePolicy.ts 顶部说明）：
 *   - 本地 patch（改名、改描述）**不**动缓存时钟——数据本身已经对了
 *   - 结构性变更（增删、影响聚合计数）才失效——让下次读取拿到服务端真相
 *
 * 账户增删会同时影响分组聚合计数，故调用方需按需同时调用 invalidateAccounts 与
 * invalidateGroups；注意 stores/accountStore.ts 的 removeAccount 已自行作废账户
 * 缓存，此时只需补 invalidateGroups。
 *
 * 邮件缓存不在此列：services/emailService.ts 与 services/spamService.ts 的每个
 * 写操作已经各自调用了 useEmailCacheStore 的 clear*，失效本来就挂在 mutation 上。
 */

import { useAccountStore } from '../stores/accountStore';
import { useGroupStore } from '../stores/groupStore';

/** 账户列表作废：下次 loadAccounts 必拉 */
export const invalidateAccounts = (): void => {
  useAccountStore.getState().invalidate();
};

/** 分组列表作废：下次 fetchGroups 必拉（含各分组 account_count 等聚合计数） */
export const invalidateGroups = (): void => {
  useGroupStore.getState().invalidate();
};
