import { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Loader2, Mail } from 'lucide-react';
import { CreateAccountRequest, accountService } from '../../services/accountService';
import { smtpService } from '../../services/smtpService';
import { Account } from '../../types';
import { useProviders } from '../../hooks/useProviders';
import { CreateModeFields } from './CreateModeFields';
import { EditModeSettings } from './EditModeSettings';
import { BatchImportMode } from './BatchImportMode';
import { toast } from 'sonner';

interface AccountFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateAccountRequest | Partial<CreateAccountRequest>) => Promise<void>;
  onOAuth2Success?: () => void;  // OAuth2 授权成功后的回调（用于刷新列表）
  account?: Account | null;
}

export const AccountForm = ({ open, onClose, onSubmit, onOAuth2Success, account }: AccountFormProps) => {
  const isEditMode = !!account;
  const { providers, getProviderByEmail, getProviderByName, findByEmail } = useProviders();

  // 匹配到的 Provider 状态
  const [matchedProvider, setMatchedProvider] = useState<typeof providers[0] | null>(null);

  // 表单数据
  const [formData, setFormData] = useState<CreateAccountRequest>({
    email: '',
    provider: '',
    protocol: 'imap',
    auth_type: 'password',
    password: '',
    sync_enabled: true,
    sync_interval: 2,
    server_delete_policy: 'off',
    first_sync_days: 7,  // 默认 7 天
    batch_size: 50,
    max_emails_per_sync: 1000,
    group_id: null,
    smtp_enabled: true,  // 默认启用发件功能
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedOAuth2ClientId, setSelectedOAuth2ClientId] = useState<number | undefined>();
  
  // SMTP 配置状态（编辑模式）
  const [smtpConfig, setSmtpConfig] = useState({ smtp_enabled: false });
  const [smtpServerConfig, setSmtpServerConfig] = useState({
    host: '', port: 0, encryption: '', fromProvider: false, providerName: '',
  });
  const [isLoadingSmtp, setIsLoadingSmtp] = useState(false);
  const [isTestingSmtp, setIsTestingSmtp] = useState(false);
  const [smtpTestResult, setSmtpTestResult] = useState<{ success: boolean; message: string } | null>(null);

  // 批量导入状态
  const [batchAccountsText, setBatchAccountsText] = useState('');
  const [batchSeparator, setBatchSeparator] = useState('----');
  const [isBatchImporting, setIsBatchImporting] = useState(false);
  const [batchImportResult, setBatchImportResult] = useState<{
    success: number;
    failed: number;
    results: Array<{ email: string; status: 'success' | 'failed'; error?: string }>;
  } | null>(null);
  const [batchImportProgress, setBatchImportProgress] = useState(0);
  const [showSmtpSection, setShowSmtpSection] = useState(false);

  const isBatchImportMode = formData.protocol === 'batch_import' && !isEditMode;

  // 清除已匹配提供商（新建模式「重新选择」按钮）
  const clearMatchedProvider = () => {
    setMatchedProvider(null);
  };

  // OAuth2 授权成功收尾：提示 + 关闭 + 触发列表刷新（原文内联逻辑搬移）
  const handleOAuth2AuthSuccess = () => {
    toast.success('OAuth2 认证成功');
    onClose();
    onOAuth2Success?.();
  };

  // 表单字段局部更新（传给 EditModeSettings 等子组件）
  const updateForm = (patch: Partial<CreateAccountRequest>) => {
    setFormData(prev => ({ ...prev, ...patch }));
  };

  // 重置表单
  useEffect(() => {
    if (account) {
      setFormData({
        email: account.email,
        provider: account.provider,
        protocol: account.protocol,
        auth_type: account.auth_type,
        password: '',
        sync_enabled: account.sync_enabled,
        sync_interval: account.sync_interval,
        server_delete_policy: account.server_delete_policy || 'off',
        group_id: account.group_id || null,
        smtp_enabled: false,
      });
      setMatchedProvider(getProviderByName(account.provider));
    } else {
      setFormData({
        email: '', provider: '', protocol: 'imap', auth_type: 'password', password: '',
        sync_enabled: true, sync_interval: 2, server_delete_policy: 'off',
        first_sync_days: 7, batch_size: 50, max_emails_per_sync: 1000,  // 默认 7 天
        group_id: null, smtp_enabled: true,  // 默认启用发件功能
      });
      setMatchedProvider(null);
    }
    setBatchAccountsText('');
    setBatchSeparator('----');
    setBatchImportResult(null);
    setBatchImportProgress(0);
    setSmtpConfig({ smtp_enabled: false });
    setSmtpServerConfig({ host: '', port: 0, encryption: '', fromProvider: false, providerName: '' });
    setSmtpTestResult(null);
    setShowSmtpSection(false);
    setSelectedOAuth2ClientId(undefined);
  }, [account, open, getProviderByName]);

  // 编辑模式加载 SMTP 配置
  useEffect(() => {
    if (isEditMode && open && account?.uid) {
      loadSmtpConfig(account.uid);
    }
  }, [isEditMode, open, account?.uid]);

  const loadSmtpConfig = async (accountUid: string) => {
    setIsLoadingSmtp(true);
    try {
      const config = await smtpService.getConfig(accountUid);
      setSmtpConfig({ smtp_enabled: config.smtp_enabled || false });
      setSmtpServerConfig({
        host: config.smtp_host || '', port: config.smtp_port || 0,
        encryption: config.smtp_encryption || '', fromProvider: config.from_provider || false,
        providerName: config.provider_name || '',
      });
      if (config.smtp_enabled) setShowSmtpSection(true);
    } catch (error) {
      console.error('加载 SMTP 配置失败:', error);
    } finally {
      setIsLoadingSmtp(false);
    }
  };

  const handleTestSmtp = async () => {
    if (!account?.uid || !smtpServerConfig.host) return;
    setIsTestingSmtp(true);
    setSmtpTestResult(null);
    try {
      const result = await smtpService.testConnection(account.uid);
      setSmtpTestResult({ success: result.success, message: result.message || (result.success ? '连接成功' : '连接失败') });
      result.success ? toast.success('SMTP 连接测试成功') : toast.error(result.message || '连接测试失败');
    } catch (error: any) {
      const msg = error?.response?.data?.message || error?.message || '连接测试失败';
      setSmtpTestResult({ success: false, message: msg });
      toast.error(msg);
    } finally {
      setIsTestingSmtp(false);
    }
  };

  // 邮箱地址变化 - 自动识别提供商
  const handleEmailChange = (email: string) => {
    setFormData(prev => ({ ...prev, email }));
    if (!isEditMode && email.includes('@')) {
      const provider = findByEmail(email) || getProviderByEmail(email);
      if (provider) {
        const hasImap = !!(provider.imap_host?.trim());
        const hasPop3 = !!(provider.pop3_host?.trim());
        let protocol = provider.recommended_protocol;
        if (protocol === 'oauth2' && !provider.requires_oauth) {
          protocol = hasImap ? 'imap' : (hasPop3 ? 'pop3' : 'imap');
        }
        setFormData(prev => ({
          ...prev, provider: provider.name, protocol,
          auth_type: protocol === 'oauth2' ? 'oauth2' : 'password',
        }));
      } else {
        setMatchedProvider(null);
      }
    }
  };

  const handleProviderChange = (providerName: string) => {
    const provider = getProviderByName(providerName);
    setMatchedProvider(provider);
    if (provider) {
      const hasImap = !!(provider.imap_host?.trim());
      const hasPop3 = !!(provider.pop3_host?.trim());
      let protocol = provider.recommended_protocol;
      if (protocol === 'oauth2' && !provider.requires_oauth) protocol = hasImap ? 'imap' : (hasPop3 ? 'pop3' : 'imap');
      else if (protocol === 'imap' && !hasImap) protocol = hasPop3 ? 'pop3' : 'imap';
      else if (protocol === 'pop3' && !hasPop3) protocol = hasImap ? 'imap' : 'pop3';
      setFormData(prev => ({
        ...prev, provider: providerName, protocol,
        auth_type: protocol === 'oauth2' ? 'oauth2' : (protocol === 'batch_import' ? 'quick' : 'password'),
      }));
    } else {
      setFormData(prev => ({ ...prev, provider: providerName, protocol: 'imap', auth_type: 'password' }));
    }
  };

  const handleProtocolChange = (protocol: string) => {
    setFormData(prev => ({
      ...prev, protocol,
      auth_type: protocol === 'oauth2' ? 'oauth2' : (protocol === 'batch_import' ? 'quick' : 'password'),
    }));
    if (protocol === 'batch_import') {
      setBatchAccountsText('');
      setBatchSeparator('----');
      setBatchImportResult(null);
      setBatchImportProgress(0);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (isBatchImportMode) { await handleBatchImport(); return; }

    setIsSubmitting(true);
    try {
      if (isEditMode) {
        const updateData: Partial<CreateAccountRequest> = {
          sync_enabled: formData.sync_enabled, sync_interval: formData.sync_interval,
          server_delete_policy: formData.server_delete_policy, group_id: formData.group_id,
        };
        if (formData.password) updateData.password = formData.password;
        await onSubmit(updateData);
        if (account?.uid) {
          try {
            await smtpService.updateConfig(account.uid, { smtp_enabled: smtpConfig.smtp_enabled });
          } catch { toast.error('SMTP 配置保存失败'); }
        }
      } else {
        await onSubmit({
          email: formData.email, provider: formData.provider, protocol: formData.protocol,
          auth_type: formData.auth_type, password: formData.password,
          sync_enabled: formData.sync_enabled, sync_interval: formData.sync_interval,
          server_delete_policy: formData.server_delete_policy, first_sync_days: formData.first_sync_days,
          batch_size: formData.batch_size, max_emails_per_sync: formData.max_emails_per_sync,
          group_id: formData.group_id, smtp_enabled: formData.smtp_enabled,
        });
      }
      onClose();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleBatchImport = async () => {
    if (!batchSeparator?.trim()) { toast.error('请设置有效的分隔符'); return; }
    const accounts = batchAccountsText.split('\n').map(l => l.trim()).filter(l => l.length > 0 && l.includes(batchSeparator));
    if (accounts.length === 0) { toast.error('请输入有效的账号信息'); return; }
    const normalized = accounts.map(l => batchSeparator === '----' ? l : l.split(batchSeparator).join('----'));

    setIsBatchImporting(true);
    setBatchImportProgress(0);
    try {
      const interval = setInterval(() => setBatchImportProgress(p => Math.min(p + 10, 90)), 500);
      const result = await accountService.batchImport(normalized, formData.sync_enabled, formData.sync_interval);
      clearInterval(interval);
      setBatchImportProgress(100);
      setBatchImportResult(result);
      if (result.success > 0) toast.success(`成功导入 ${result.success} 个账户`);
      if (result.failed > 0) toast.error(`${result.failed} 个账户导入失败`);
    } catch {
      setBatchImportResult({
        success: 0, failed: normalized.length,
        results: normalized.map(a => ({ email: a.split('----')[0], status: 'failed', error: '导入失败' })),
      });
      toast.error('批量导入失败');
    } finally {
      setIsBatchImporting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="w-full max-w-[95vw] sm:max-w-[600px] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>
            {isEditMode ? '编辑邮箱账户' : isBatchImportMode ? '批量导入邮箱账户' : '添加邮箱账户'}
          </DialogTitle>
          <DialogDescription>
            {isEditMode ? '修改账户的同步设置' : isBatchImportMode ? '批量导入多个短效邮箱账户' : '添加您的邮箱账户以开始接收邮件'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto">
          <form onSubmit={handleSubmit} className="flex flex-col h-full">
            <div className="space-y-4 py-4 px-1 flex-1 overflow-y-auto min-h-0">
              
              {/* 编辑模式：账户信息概览 */}
              {isEditMode && !isBatchImportMode && (
                <div className="p-4 rounded-lg bg-muted/30 border">
                  <div className="flex items-center gap-3">
                    <Mail className="h-5 w-5 text-muted-foreground" />
                    <div className="flex-1 min-w-0">
                      <p className="font-medium truncate">{formData.email}</p>
                      <p className="text-xs text-muted-foreground">
                        {matchedProvider?.display_name || formData.provider} · {formData.protocol?.toUpperCase() || 'N/A'}
                      </p>
                    </div>
                  </div>
                </div>
              )}

              {/* 新建模式：邮箱优先流程 */}
              {!isEditMode && !isBatchImportMode && (
                <CreateModeFields
                  formData={formData}
                  onFormChange={updateForm}
                  matchedProvider={matchedProvider}
                  onClearProvider={clearMatchedProvider}
                  onEmailChange={handleEmailChange}
                  onProviderChange={handleProviderChange}
                  onProtocolChange={handleProtocolChange}
                  selectedOAuth2ClientId={selectedOAuth2ClientId}
                  onOAuth2ClientChange={setSelectedOAuth2ClientId}
                  onOAuth2AuthSuccess={handleOAuth2AuthSuccess}
                  providers={providers}
                />
              )}


              {/* 编辑模式设置 */}
              {isEditMode && !isBatchImportMode && (
                <EditModeSettings
                  formData={formData}
                  onFormChange={updateForm}
                  smtpConfig={smtpConfig}
                  onSmtpConfigChange={setSmtpConfig}
                  smtpServerConfig={smtpServerConfig}
                  isLoadingSmtp={isLoadingSmtp}
                  isTestingSmtp={isTestingSmtp}
                  smtpSectionOpen={showSmtpSection}
                  onSmtpSectionOpenChange={setShowSmtpSection}
                  smtpTestResult={smtpTestResult}
                  onTestSmtp={handleTestSmtp}
                />
              )}

              {/* 批量导入模式 */}
              {isBatchImportMode && (
                <BatchImportMode
                  batchSeparator={batchSeparator}
                  onSeparatorChange={setBatchSeparator}
                  batchAccountsText={batchAccountsText}
                  onAccountsTextChange={setBatchAccountsText}
                  isBatchImporting={isBatchImporting}
                  batchImportProgress={batchImportProgress}
                  batchImportResult={batchImportResult}
                />
              )}

            </div>

            {/* 底部按钮 */}
            <DialogFooter className="flex-shrink-0 pt-4 border-t">
              <Button type="button" variant="outline" onClick={onClose}>取消</Button>
              {formData.protocol !== 'oauth2' && (
                <Button type="submit" disabled={isSubmitting || isBatchImporting}>
                  {isSubmitting || isBatchImporting ? (
                    <><Loader2 className="h-4 w-4 animate-spin mr-2" />处理中...</>
                  ) : (
                    isEditMode ? '保存' : isBatchImportMode ? '开始导入' : '添加账户'
                  )}
                </Button>
              )}
            </DialogFooter>
          </form>
        </div>
      </DialogContent>
    </Dialog>
  );
};
