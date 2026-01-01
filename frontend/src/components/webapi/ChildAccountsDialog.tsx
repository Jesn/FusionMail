// 子邮箱列表对话框 - 显示 WebAPI 服务端的账户列表
import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { 
  Loader2, 
  Mail, 
  RefreshCw,
  Inbox,
  User,
  Copy,
  Check,
  Cloud,
} from 'lucide-react';
import { Account } from '../../types';
import { webapiService } from '../../services/webapiService';
import toast from 'react-hot-toast';

interface ChildAccountsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  account: Account | null;
}

// 服务端账户信息
interface SubAccountInfo {
  account_id: number;
  email: string;
  name: string;
}

// 服务类型配置
interface ServiceTypeConfig {
  title: string;
  description: string;
  emptyMessage: string;
  icon: React.ReactNode;
}

// 服务类型映射
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

// 默认配置
const DEFAULT_CONFIG: ServiceTypeConfig = {
  title: 'WebAPI 邮箱账户',
  description: '服务端的邮箱账户',
  emptyMessage: '未找到邮箱账户',
  icon: <Mail className="h-5 w-5" />,
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
 * 子邮箱列表对话框
 * 显示 WebAPI 服务端的所有邮箱账户
 */
export const ChildAccountsDialog: React.FC<ChildAccountsDialogProps> = ({
  open,
  onOpenChange,
  account,
}) => {
  const [subAccounts, setSubAccounts] = useState<SubAccountInfo[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<number | null>(null);

  // 获取服务类型和配置
  const serviceType = useMemo(() => {
    return parseServiceType(account?.provider_ref?.metadata) || 'unknown';
  }, [account?.provider_ref?.metadata]);

  const config = useMemo(() => {
    return SERVICE_TYPE_CONFIGS[serviceType] || DEFAULT_CONFIG;
  }, [serviceType]);

  // 加载子邮箱账户列表
  useEffect(() => {
    if (open && account?.uid) {
      loadSubAccounts();
    }
  }, [open, account?.uid]);

  const loadSubAccounts = async () => {
    if (!account?.uid) return;
    
    setIsLoading(true);
    setError(null);
    try {
      const accounts = await webapiService.getCloudMailAccounts(account.uid);
      setSubAccounts(accounts);
    } catch (err: any) {
      console.error('加载子邮箱账户列表失败:', err);
      setError(err?.message || '加载失败');
    } finally {
      setIsLoading(false);
    }
  };

  // 复制邮箱地址
  const handleCopyEmail = async (email: string, accountId: number) => {
    try {
      await navigator.clipboard.writeText(email);
      setCopiedId(accountId);
      toast.success('已复制邮箱地址');
      // 2 秒后重置复制状态
      setTimeout(() => setCopiedId(null), 2000);
    } catch (err) {
      toast.error('复制失败');
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[600px] max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {config.icon}
            {config.title}
          </DialogTitle>
          <DialogDescription>
            {account?.email} {config.description}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">加载中...</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <p className="text-destructive mb-4">{error}</p>
              <Button variant="outline" size="sm" onClick={loadSubAccounts}>
                <RefreshCw className="h-4 w-4 mr-2" />
                重试
              </Button>
            </div>
          ) : subAccounts.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <Inbox className="h-12 w-12 mb-4 opacity-50" />
              <p>暂无邮箱账户</p>
              <p className="text-sm mt-2">{config.emptyMessage}</p>
            </div>
          ) : (
            <div className="space-y-3">
              {/* 统计信息 */}
              <div className="flex items-center justify-between px-1 mb-4">
                <span className="text-sm text-muted-foreground">
                  共 {subAccounts.length} 个邮箱账户
                </span>
                <Button variant="ghost" size="sm" onClick={loadSubAccounts}>
                  <RefreshCw className="h-4 w-4 mr-1" />
                  刷新
                </Button>
              </div>

              {/* 邮箱账户列表 */}
              {subAccounts.map((mailAccount) => (
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
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ChildAccountsDialog;
