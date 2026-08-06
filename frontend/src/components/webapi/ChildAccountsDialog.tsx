// 子邮箱列表对话框 - 显示 WebAPI 服务端的账户列表或本地子账户
import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { 
  Loader2, 
  Mail, 
  RefreshCw,
  Inbox,
  User,
  Copy,
  Check,
  Cloud,
  Webhook,
  MailCheck,
  Clock,
  Search,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { Account } from '../../types';
import { webapiService } from '../../services/webapiService';
import toast from 'react-hot-toast';

const PAGE_SIZE = 20;

interface ChildAccountsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  account: Account | null;
}

// 服务端账户信息（轮询模式）
interface SubAccountInfo {
  account_id: number;
  email: string;
  name: string;
}

// 本地子账户信息（Webhook / 本地投影）
interface LocalChildAccount {
  uid: string;
  email: string;
  status: string;
  disable_reason?: string;
  total_emails: number;
  unread_count: number;
  last_sync_at: string | null;
  created_at: string;
  orphaned?: boolean;
}

// 服务类型配置
interface ServiceTypeConfig {
  title: string;
  description: string;
  emptyMessage: string;
  icon: React.ReactNode;
}

// 服务类型映射（轮询模式）
const SERVICE_TYPE_CONFIGS: Record<string, ServiceTypeConfig> = {
  cloud_mail: {
    title: 'Cloud Mail 邮箱账户',
    description: '服务端的所有邮箱账户',
    emptyMessage: 'Cloud Mail 服务端没有配置邮箱账户',
    icon: <Mail className="h-5 w-5" />,
  },
  cloudflare_temp_email: {
    title: 'Cloudflare 临时邮箱',
    description: '当前配置的邮箱地址',
    emptyMessage: '未配置邮箱地址',
    icon: <Cloud className="h-5 w-5" />,
  },
  custom: {
    title: '自定义 WebAPI 邮箱',
    description: '服务端的邮箱账户',
    emptyMessage: '未找到邮箱账户',
    icon: <Mail className="h-5 w-5" />,
  },
};

// 默认配置（轮询模式）
const DEFAULT_CONFIG: ServiceTypeConfig = {
  title: 'WebAPI 邮箱账户',
  description: '服务端的邮箱账户',
  emptyMessage: '未找到邮箱账户',
  icon: <Mail className="h-5 w-5" />,
};

// Webhook 模式配置
const WEBHOOK_CONFIG: ServiceTypeConfig = {
  title: 'Webhook 子邮箱',
  description: '通过 Webhook 推送自动创建的子邮箱',
  emptyMessage: '暂无子邮箱，子邮箱会在收到邮件时自动创建',
  icon: <Webhook className="h-5 w-5" />,
};

/**
 * 从 Provider metadata 中解析服务类型
 */
function parseServiceType(metadata?: string): string | null {
  if (!metadata) return null;
  try {
    const config = JSON.parse(metadata);
    return config.service_type || null;
  } catch {
    return null;
  }
}

/**
 * 格式化日期时间
 */
function formatDateTime(dateStr: string | null): string {
  if (!dateStr) return '从未';
  const date = new Date(dateStr);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * 子邮箱列表对话框
 * 根据同步模式显示不同内容：
 * - Webhook 模式：显示本地数据库中的子账户
 * - 轮询模式：显示远程服务端的邮箱账户
 */
export const ChildAccountsDialog: React.FC<ChildAccountsDialogProps> = ({
  open,
  onOpenChange,
  account,
}) => {
  // 轮询模式：服务端账户列表
  const [subAccounts, setSubAccounts] = useState<SubAccountInfo[]>([]);
  // 本地子账户（服务端分页）
  const [localChildren, setLocalChildren] = useState<LocalChildAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [include, setInclude] = useState<'active' | 'orphaned' | 'all'>('all');
  const [emailQuery, setEmailQuery] = useState('');
  const [emailInput, setEmailInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isReconciling, setIsReconciling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<number | string | null>(null);

  // 判断是否为 Webhook 模式
  const isWebhookMode = useMemo(() => {
    return account?.sync_mode === 'webhook';
  }, [account?.sync_mode]);

  // 获取服务类型和配置
  const serviceType = useMemo(() => {
    return parseServiceType(account?.provider_ref?.metadata) || 'unknown';
  }, [account?.provider_ref?.metadata]);

  const config = useMemo(() => {
    // Webhook 模式使用专用配置
    if (isWebhookMode) {
      return WEBHOOK_CONFIG;
    }
    return SERVICE_TYPE_CONFIGS[serviceType] || DEFAULT_CONFIG;
  }, [serviceType, isWebhookMode]);

  // 打开对话框时重置条件
  useEffect(() => {
    if (open && account?.uid) {
      setPage(1);
      setEmailQuery('');
      setEmailInput('');
      setInclude('all');
    }
  }, [open, account?.uid]);

  // 条件变化时加载
  useEffect(() => {
    if (open && account?.uid) {
      loadAccounts();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, account?.uid, isWebhookMode, page, include, emailQuery]);

  const loadAccounts = async () => {
    if (!account?.uid) return;
    
    setIsLoading(true);
    setError(null);
    try {
      if (isWebhookMode) {
        const result = await webapiService.getChildAccounts(account.uid, {
          include,
          email: emailQuery || undefined,
          page,
          page_size: PAGE_SIZE,
        });
        setLocalChildren(result.items);
        setTotal(result.total);
        setSubAccounts([]);
      } else {
        // 轮询模式：远端全量列表（通常较少）+ 本地子账户分页
        const [accounts, children] = await Promise.all([
          webapiService.getCloudMailAccounts(account.uid),
          webapiService.getChildAccounts(account.uid, {
            include,
            email: emailQuery || undefined,
            page,
            page_size: PAGE_SIZE,
          }).catch(() => ({ items: [] as LocalChildAccount[], total: 0, page: 1, page_size: PAGE_SIZE })),
        ]);
        setSubAccounts(accounts);
        setLocalChildren(children.items);
        setTotal(children.total);
      }
    } catch (err: any) {
      console.error('加载子邮箱账户列表失败:', err);
      setError(err?.message || '加载失败');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = () => {
    setPage(1);
    setEmailQuery(emailInput.trim());
  };

  const handleIncludeChange = (next: 'active' | 'orphaned' | 'all') => {
    setPage(1);
    setInclude(next);
  };

  // 与远端对账：远端无 → 标记孤儿（保留邮件）
  const handleReconcile = async () => {
    if (!account?.uid) return;
    setIsReconciling(true);
    try {
      const result = await webapiService.reconcileChildAccounts(account.uid);
      if (result.skipped_remote) {
        toast.error(result.message || '无法获取远端列表，对账已跳过');
      } else {
        toast.success(
          result.message ||
            `对账完成：标记孤儿 ${result.marked_orphaned}，恢复 ${result.reactivated}`
        );
      }
      setPage(1);
      await loadAccounts();
    } catch (err: any) {
      toast.error(err?.message || '对账失败');
    } finally {
      setIsReconciling(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // 复制邮箱地址
  const handleCopyEmail = async (email: string, id: number | string) => {
    try {
      await navigator.clipboard.writeText(email);
      setCopiedId(id);
      toast.success('已复制邮箱地址');
      // 2 秒后重置复制状态
      setTimeout(() => setCopiedId(null), 2000);
    } catch (err) {
      toast.error('复制失败');
    }
  };

  const showLocalList = isWebhookMode || localChildren.length > 0;
  const isEmpty = isWebhookMode
    ? total === 0 && !isLoading
    : subAccounts.length === 0 && total === 0 && !isLoading;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[640px] max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {config.icon}
            {config.title}
          </DialogTitle>
          <DialogDescription>
            {account?.email} {config.description}
            {isWebhookMode && (
              <span className="block mt-1 text-xs">
                列表来自 FusionMail 本地记录；支持分页与邮箱搜索。域名删除后请点「与远端对账」。
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        {/* 搜索与过滤（本地子邮箱） */}
        {showLocalList && (
          <div className="flex flex-col gap-2 px-1 pb-2 border-b">
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  className="pl-8"
                  placeholder="按邮箱搜索，如 user@ 或 example.com"
                  value={emailInput}
                  onChange={(e) => setEmailInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleSearch();
                  }}
                />
              </div>
              <Button size="sm" onClick={handleSearch} disabled={isLoading}>
                搜索
              </Button>
              {(emailQuery || include !== 'all') && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    setEmailInput('');
                    setEmailQuery('');
                    setInclude('all');
                    setPage(1);
                  }}
                >
                  重置
                </Button>
              )}
            </div>
            <div className="flex items-center gap-1 flex-wrap">
              {(
                [
                  { key: 'all', label: '全部' },
                  { key: 'active', label: '有效' },
                  { key: 'orphaned', label: '远端已失效' },
                ] as const
              ).map((opt) => (
                <Button
                  key={opt.key}
                  size="sm"
                  variant={include === opt.key ? 'default' : 'outline'}
                  onClick={() => handleIncludeChange(opt.key)}
                >
                  {opt.label}
                </Button>
              ))}
              <div className="flex-1" />
              <Button
                variant="outline"
                size="sm"
                onClick={handleReconcile}
                disabled={isReconciling}
                title="对比远端有效地址，将本地多余子账户标记为失效"
              >
                {isReconciling ? (
                  <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4 mr-1" />
                )}
                与远端对账
              </Button>
              <Button variant="ghost" size="sm" onClick={loadAccounts} disabled={isLoading}>
                <RefreshCw className="h-4 w-4 mr-1" />
                刷新
              </Button>
            </div>
          </div>
        )}

        <div className="flex-1 overflow-y-auto py-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">加载中...</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <p className="text-destructive mb-4">{error}</p>
              <Button variant="outline" size="sm" onClick={loadAccounts}>
                <RefreshCw className="h-4 w-4 mr-2" />
                重试
              </Button>
            </div>
          ) : isEmpty ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <Inbox className="h-12 w-12 mb-4 opacity-50" />
              <p>{emailQuery ? '未找到匹配的邮箱' : '暂无邮箱账户'}</p>
              <p className="text-sm mt-2">{emailQuery ? `关键词：${emailQuery}` : config.emptyMessage}</p>
            </div>
          ) : (
            <div className="space-y-3">
              {/* 统计 */}
              <div className="flex items-center justify-between px-1 mb-2 gap-2 flex-wrap">
                <span className="text-sm text-muted-foreground">
                  {isWebhookMode
                    ? `共 ${total} 个，第 ${page}/${totalPages} 页`
                    : `远端 ${subAccounts.length} 个 · 本地 ${total} 个（第 ${page}/${totalPages} 页）`}
                </span>
              </div>

              {/* 本地子账户分页列表 */}
              {localChildren.map((child) => (
                <div
                  key={child.uid}
                  className={`p-4 rounded-lg border bg-card hover:bg-muted/50 transition-colors ${
                    child.orphaned ? 'opacity-80 border-amber-300/60 dark:border-amber-700/50' : ''
                  }`}
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-2 flex-wrap">
                        <Mail className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                        <span className="font-medium truncate" title={child.email}>
                          {child.email}
                        </span>
                        {child.orphaned ? (
                          <span className="text-xs px-2 py-0.5 rounded-full bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">
                            远端已失效
                          </span>
                        ) : (
                          <span className={`text-xs px-2 py-0.5 rounded-full ${
                            child.status === 'active'
                              ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                              : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                          }`}>
                            {child.status === 'active' ? '活跃' : child.status}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-4 text-sm text-muted-foreground flex-wrap">
                        <span className="flex items-center gap-1" title="邮件总数">
                          <MailCheck className="h-3.5 w-3.5" />
                          {child.total_emails} 封
                        </span>
                        {child.unread_count > 0 && (
                          <span className="flex items-center gap-1 text-blue-600 dark:text-blue-400" title="未读邮件">
                            <Mail className="h-3.5 w-3.5" />
                            {child.unread_count} 未读
                          </span>
                        )}
                        <span className="flex items-center gap-1" title="创建时间">
                          <Clock className="h-3.5 w-3.5" />
                          {formatDateTime(child.created_at)}
                        </span>
                      </div>
                      {child.orphaned && (
                        <p className="text-xs text-amber-700 dark:text-amber-400 mt-2">
                          域名侧已无此邮箱；本地邮件仍保留。再次收到该地址邮件时会自动恢复。
                        </p>
                      )}
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleCopyEmail(child.email, child.uid)}
                      className="flex-shrink-0"
                      title="复制邮箱地址"
                    >
                      {copiedId === child.uid ? (
                        <Check className="h-4 w-4 text-green-500" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>
              ))}

              {/* 轮询模式：显示服务端账户 */}
              {!isWebhookMode && subAccounts.map((mailAccount) => (
                <div
                  key={mailAccount.account_id}
                  className="p-4 rounded-lg border bg-card hover:bg-muted/50 transition-colors"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-2">
                        <Mail className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                        <span className="font-medium truncate" title={mailAccount.email}>
                          {mailAccount.email}
                        </span>
                      </div>
                      <div className="flex items-center gap-4 text-sm text-muted-foreground">
                        <span className="flex items-center gap-1">
                          <User className="h-3.5 w-3.5" />
                          {mailAccount.name || '未命名'}
                        </span>
                        {mailAccount.account_id > 0 && (
                          <span className="text-xs">
                            ID: {mailAccount.account_id}
                          </span>
                        )}
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleCopyEmail(mailAccount.email, mailAccount.account_id)}
                      className="flex-shrink-0"
                      title="复制邮箱地址"
                    >
                      {copiedId === mailAccount.account_id ? (
                        <Check className="h-4 w-4 text-green-500" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>
              ))}

              {/* 本地子邮箱分页 */}
              {total > PAGE_SIZE && (
                <div className="flex items-center justify-between pt-3 border-t px-1">
                  <span className="text-xs text-muted-foreground">
                    第 {page} / {totalPages} 页，共 {total} 条
                  </span>
                  <div className="flex items-center gap-1">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={page <= 1 || isLoading}
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                    >
                      <ChevronLeft className="h-4 w-4 mr-1" />
                      上一页
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={page >= totalPages || isLoading}
                      onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    >
                      下一页
                      <ChevronRight className="h-4 w-4 ml-1" />
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ChildAccountsDialog;
