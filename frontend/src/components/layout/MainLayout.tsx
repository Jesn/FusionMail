import { ReactNode, useEffect, useRef } from 'react';
import { Header } from './Header';
import { Sidebar } from './Sidebar';
import { useUIStore } from '../../stores/uiStore';
import { cn } from '../../lib/utils';
import { emailService } from '../../services/emailService';
import { useEmailStore } from '../../stores/emailStore';
import { useAuthStore } from '../../stores/authStore';

interface MainLayoutProps {
  children: ReactNode;
}

export const MainLayout = ({ children }: MainLayoutProps) => {
  const { sidebarCollapsed } = useUIStore();
  const unreadCount = useEmailStore((state) => state.unreadCount);
  const prevUnreadCountRef = useRef(unreadCount);
  const blinkTimerRef = useRef<number | null>(null);

  // 全局一次性建立 SSE 订阅（默认基于 Cookie，可通过开关切到 Bearer Query 模式，带 400ms 去抖）
  useEffect(() => {
    const { setUnreadCount, setStarredCount, setArchivedCount, setDeletedCount } = useEmailStore.getState();
    const { token } = useAuthStore.getState();

    const API_BASE = import.meta.env.VITE_API_BASE_URL || '';
    const mode = import.meta.env.VITE_SSE_AUTH_MODE || 'cookie';

    let url = `${API_BASE}/api/v1/events`;
    let es: EventSource;

    if (mode === 'bearer-query' && token) {
      const sep = url.includes('?') ? '&' : '?';
      url = `${url}${sep}token=${encodeURIComponent(token)}`;
      console.log('[SSE] 准备建立连接 (bearer-query 模式):', url);
      es = new EventSource(url, { withCredentials: false });
    } else {
      console.log('[SSE] 准备建立连接 (cookie 模式):', url);
      es = new EventSource(url, { withCredentials: true });
    }

    let debounceTimer: number | undefined;
    let hasInitialFetch = false; // 标记是否已完成初始拉取

    const triggerFetch = async () => {
      if (debounceTimer) {
        console.log('[SSE] 防抖中，跳过本次请求');
        return;
      }

      console.log('[SSE] 触发统计更新');
      debounceTimer = window.setTimeout(async () => {
        try {
          // SSE 收到变更信号时，认为列表/搜索缓存可能已过期，统一清理
          emailService.clearAllCache();

          const stats = await emailService.getGlobalStats();
          console.log('[SSE] 统计更新成功:', stats);

          // 直接更新统计到 store
          setUnreadCount(stats.unread_count);
          setStarredCount(stats.starred_count);
          setArchivedCount(stats.archived_count);
          setDeletedCount(stats.deleted_count);

          // 自动刷新当前列表：使用当前筛选/分页条件重新拉取列表
          const {
            filter,
            searchQuery,
            page,
            pageSize,
            isLoading,
            setLoading: setListLoading,
            setError: setListError,
            setEmails,
          } = useEmailStore.getState();

          if (!isLoading) {
            try {
              setListLoading(true);
              setListError(null);

              const response = searchQuery
                ? await emailService.search(searchQuery, filter.account_uid, {
                    page,
                    page_size: pageSize,
                  })
                : await emailService.getList(filter, {
                    page,
                    page_size: pageSize,
                  });

              setEmails(response);
              console.log('[SSE] 自动刷新当前列表成功');
            } catch (listErr) {
              const message =
                listErr instanceof Error ? listErr.message : '自动刷新邮件列表失败';
              console.warn('[SSE] 自动刷新列表失败', listErr);
              setListError(message);
            } finally {
              setListLoading(false);
            }
          } else {
            console.log('[SSE] 当前列表正在加载，跳过自动刷新');
          }

          hasInitialFetch = true;
        } catch (err) {
          console.warn('[SSE] 刷新统计失败', err);
        } finally {
          debounceTimer = undefined;
        }
      }, 400);
    };

    es.addEventListener('email_counts_maybe_changed', () => {
      console.log('[SSE] 收到 email_counts_maybe_changed 事件');
      triggerFetch();
    });

    es.addEventListener('open', () => {
      console.log('[SSE] 连接已建立');
      // 仅在未完成初始拉取时才触发（避免与冷启动重复）
      if (!hasInitialFetch) {
        console.log('[SSE] 首次连接，触发初始拉取');
        triggerFetch();
      }
    });

    es.addEventListener('error', (e) => {
      console.warn('[SSE] 连接异常', e);
      // 检查连接状态
      if (es.readyState === EventSource.CLOSED) {
        console.error('[SSE] 连接已关闭');
      } else if (es.readyState === EventSource.CONNECTING) {
        console.log('[SSE] 正在重连...');
      }
    });

    // 冷启动：首次挂载时主动拉取一次，避免等待事件
    console.log('[SSE] 冷启动，触发初始拉取');
    triggerFetch();

    return () => {
      console.log('[SSE] 清理连接');
      es.close();
      if (debounceTimer) window.clearTimeout(debounceTimer);
    };
  }, []);

  // 根据未读邮件数更新浏览器标签标题，在工具栏上提供提醒，并在未读数增加时做短暂闪烁
  useEffect(() => {
    const baseTitle = 'FusionMail';

    // 先更新为稳定状态标题
    if (unreadCount > 0) {
      document.title = `(${unreadCount}) ${baseTitle}`;
    } else {
      document.title = baseTitle;
    }

    const prev = prevUnreadCountRef.current;

    // 未读数增加时，做一次短暂的标题闪烁效果
    if (unreadCount > prev) {
      if (blinkTimerRef.current) {
        window.clearInterval(blinkTimerRef.current);
        blinkTimerRef.current = null;
      }

      let visible = true;
      const start = Date.now();
      const duration = 8000; // 闪烁 8 秒

      blinkTimerRef.current = window.setInterval(() => {
        const now = Date.now();
        if (now - start >= duration) {
          if (blinkTimerRef.current) {
            window.clearInterval(blinkTimerRef.current);
            blinkTimerRef.current = null;
          }
          // 恢复为稳定标题
          if (unreadCount > 0) {
            document.title = `(${unreadCount}) ${baseTitle}`;
          } else {
            document.title = baseTitle;
          }
          return;
        }

        visible = !visible;
        document.title = visible
          ? `(${unreadCount}) ${baseTitle}`
          : baseTitle;
      }, 800);
    }

    prevUnreadCountRef.current = unreadCount;

    return () => {
      if (blinkTimerRef.current) {
        window.clearInterval(blinkTimerRef.current);
        blinkTimerRef.current = null;
      }
    };
  }, [unreadCount]);

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* 侧边栏 */}
      <Sidebar />

      {/* 主内容区域 */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* 头部 */}
        <Header />

        {/* 内容区域 */}
        <main
          className={cn(
            'flex-1 overflow-auto transition-all duration-300',
            sidebarCollapsed ? 'ml-0' : 'ml-0'
          )}
        >
          {children}
        </main>
      </div>
    </div>
  );
};
