import { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Switch } from '../ui/switch';
import { Textarea } from '../ui/textarea';
import { Alert, AlertDescription } from '../ui/alert';
import { Progress } from '../ui/progress';
import { ScrollArea } from '../ui/scroll-area';
import { Badge } from '../ui/badge';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { AlertCircle, CheckCircle2, XCircle, Loader2, ChevronDown, ChevronRight, Server, Send, Mail } from 'lucide-react';
import { CreateAccountRequest, accountService } from '../../services/accountService';
import { smtpService } from '../../services/smtpService';
import { Account } from '../../types';
import { useProviders } from '../../hooks/useProviders';
import { OAuth2AuthButton } from '../auth/OAuth2AuthButton';
import { OAuth2ClientSelector } from '../oauth2';
import { GroupSelector } from '../group';
import toast from 'react-hot-toast';

interface AccountFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateAccountRequest | Partial<CreateAccountRequest>) => Promise<void>;
  account?: Account | null;
}

export const AccountForm = ({ open, onClose, onSubmit, account }: AccountFormProps) => {
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
  const [showSmtpSection, setShowSmtpSection] = useState(false);

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

  const isBatchImportMode = formData.protocol === 'batch_import' && !isEditMode;

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

  const getEncryptionLabel = (enc: string) => {
    switch (enc) {
      case 'tls': case 'ssl': return 'SSL/TLS';
      case 'starttls': return 'STARTTLS';
      case 'none': return '无加密';
      default: return enc || '未配置';
    }
  };

  // 邮箱地址变化 - 自动识别提供商
  const handleEmailChange = (email: string) => {
    setFormData(prev => ({ ...prev, email }));
    if (!isEditMode && email.includes('@')) {
      const provider = findByEmail(email) || getProviderByEmail(email);
      if (provider) {
        setMatchedProvider(provider);
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

  const getAvailableProtocols = () => {
    if (!matchedProvider) return [];
    const protocols: Array<{ value: string; label: string; recommended?: boolean }> = [];
    const supported = matchedProvider.supported_protocols || [];
    const rec = matchedProvider.recommended_protocol;
    if (matchedProvider.requires_oauth) protocols.push({ value: 'oauth2', label: 'OAuth2 安全认证', recommended: rec === 'oauth2' });
    if (matchedProvider.imap_host?.trim()) protocols.push({ value: 'imap', label: 'IMAP 协议', recommended: rec === 'imap' });
    if (matchedProvider.pop3_host?.trim()) protocols.push({ value: 'pop3', label: 'POP3 协议', recommended: rec === 'pop3' });
    if (supported.includes('batch_import')) protocols.push({ value: 'batch_import', label: '批量导入（短效邮箱）' });
    return protocols;
  };

  const hasSmtpSupport = () => matchedProvider ? !!(matchedProvider.smtp_host?.trim()) : false;


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
                        {matchedProvider?.display_name || formData.provider} · {formData.protocol.toUpperCase()}
                      </p>
                    </div>
                  </div>
                </div>
              )}

              {/* 新建模式：邮箱优先流程 */}
              {!isEditMode && !isBatchImportMode && (
                <>
                  {/* 邮箱地址输入（非 OAuth2） */}
                  {formData.protocol !== 'oauth2' && (
                    <div className="space-y-2">
                      <Label htmlFor="email">
                        <span className="flex items-center gap-2">
                          <Mail className="h-4 w-4" />
                          邮箱地址 *
                        </span>
                      </Label>
                      <Input
                        id="email"
                        type="email"
                        placeholder="your@example.com"
                        value={formData.email}
                        onChange={(e) => handleEmailChange(e.target.value)}
                        required
                        autoFocus
                      />
                      <p className="text-xs text-muted-foreground">输入邮箱地址，系统将自动识别邮箱提供商</p>
                    </div>
                  )}

                  {/* 匹配状态显示 */}
                  {matchedProvider && (
                    <Alert className="border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950">
                      <CheckCircle2 className="h-4 w-4 text-green-600" />
                      <AlertDescription className="text-green-700 dark:text-green-300">
                        <div className="flex items-center justify-between">
                          <span>
                            已识别为 <strong>{matchedProvider.display_name}</strong>
                            {matchedProvider.recommended_protocol && (
                              <span className="ml-2 text-xs">（推荐：{matchedProvider.recommended_protocol.toUpperCase()}）</span>
                            )}
                          </span>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="h-6 px-2 text-xs text-green-700 hover:text-green-800 hover:bg-green-100"
                            onClick={() => {
                              setMatchedProvider(null);
                              setFormData(prev => ({ ...prev, provider: '', protocol: 'imap', auth_type: 'password' }));
                            }}
                          >
                            重新选择
                          </Button>
                        </div>
                      </AlertDescription>
                    </Alert>
                  )}

                  {/* 未匹配时显示警告提示 - 只有当邮箱格式完整时才显示 */}
                  {!matchedProvider && formData.email.includes('@') && formData.email.split('@')[1]?.includes('.') && (
                    <Alert className="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950">
                      <AlertCircle className="h-4 w-4 text-amber-600" />
                      <AlertDescription className="text-amber-700 dark:text-amber-300">
                        <div className="space-y-2">
                          <p>未识别的邮箱域名 <strong>{formData.email.split('@')[1]}</strong>，请手动选择提供商或前往「提供商管理」添加新的提供商配置。</p>
                          <Select value={formData.provider} onValueChange={handleProviderChange}>
                            <SelectTrigger className="bg-white dark:bg-gray-900">
                              <SelectValue placeholder="选择邮箱提供商..." />
                            </SelectTrigger>
                            <SelectContent>
                              {providers.map((p) => (
                                <SelectItem key={p.name} value={p.name}>{p.display_name}</SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>
                      </AlertDescription>
                    </Alert>
                  )}

                  {/* 协议选择 */}
                  {matchedProvider && getAvailableProtocols().length > 1 && (
                    <div className="space-y-2">
                      <Label>认证方式 / 协议</Label>
                      <Select value={formData.protocol} onValueChange={handleProtocolChange}>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {getAvailableProtocols().map((p) => (
                            <SelectItem key={p.value} value={p.value}>
                              {p.label} {p.recommended && <Badge variant="secondary" className="ml-2">推荐</Badge>}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  {/* OAuth2 认证 */}
                  {formData.protocol === 'oauth2' && matchedProvider && (
                    <div className="space-y-4 p-4 rounded-lg border bg-muted/20">
                      <div className="flex items-center gap-2 text-sm font-medium">
                        <Server className="h-4 w-4" />
                        OAuth2 安全认证
                      </div>
                      <OAuth2ClientSelector
                        providerId={matchedProvider.id}
                        value={selectedOAuth2ClientId}
                        onChange={setSelectedOAuth2ClientId}
                      />
                      <OAuth2AuthButton
                        provider={formData.provider === 'gmail' ? 'google' : 'microsoft'}
                        email={formData.email}
                        selectedClientId={selectedOAuth2ClientId}
                        onSuccess={() => { toast.success('OAuth2 认证成功'); onClose(); }}
                        onError={(err) => toast.error(err)}
                      />
                    </div>
                  )}

                  {/* 密码/授权码输入 */}
                  {formData.protocol !== 'oauth2' && matchedProvider && (
                    <div className="space-y-2">
                      <Label htmlFor="password">
                        {['qq', '163', '126', '139', '189'].includes(formData.provider) ? '授权码 / 应用专用密码 *' : '密码 *'}
                      </Label>
                      <Input
                        id="password"
                        type="password"
                        placeholder={['qq', '163', '126', '139', '189'].includes(formData.provider) ? '请输入授权码' : '请输入密码'}
                        value={formData.password}
                        onChange={(e) => setFormData(prev => ({ ...prev, password: e.target.value }))}
                        required
                      />
                      {['qq', '163', '126', '139', '189'].includes(formData.provider) && (
                        <p className="text-xs text-muted-foreground">
                          国内邮箱通常需要使用授权码而非登录密码，请在邮箱设置中开启 IMAP/POP3 服务并获取授权码
                        </p>
                      )}
                    </div>
                  )}

                  {/* 发件功能开关（新建模式） - 只要有匹配的提供商且非 OAuth2 就显示 */}
                  {matchedProvider && formData.protocol !== 'oauth2' && (
                    <div className="flex items-center justify-between p-3 rounded-lg border bg-muted/20">
                      <div className="flex items-center gap-2">
                        <Send className="h-4 w-4 text-muted-foreground" />
                        <div>
                          <Label htmlFor="smtp_enabled" className="cursor-pointer">启用发件功能</Label>
                          <p className="text-xs text-muted-foreground">
                            {hasSmtpSupport() 
                              ? '使用 SMTP 发送邮件（服务器配置继承自提供商）' 
                              : '该提供商暂未配置 SMTP 服务器'}
                          </p>
                        </div>
                      </div>
                      <Switch
                        id="smtp_enabled"
                        checked={formData.smtp_enabled}
                        onCheckedChange={(checked) => setFormData(prev => ({ ...prev, smtp_enabled: checked }))}
                        disabled={!hasSmtpSupport()}
                      />
                    </div>
                  )}

                  {/* 分组选择 */}
                  <div className="space-y-2">
                    <Label>分组</Label>
                    <GroupSelector
                      value={formData.group_id}
                      onChange={(id) => setFormData(prev => ({ ...prev, group_id: id }))}
                    />
                  </div>

                  {/* 同步设置折叠区 */}
                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2 px-3 rounded-lg border bg-muted/30 hover:bg-muted/50 transition-colors">
                      <ChevronRight className="h-4 w-4 transition-transform duration-200 [[data-state=open]>&]:rotate-90" />
                      高级设置
                      <span className="ml-auto text-xs text-muted-foreground font-normal">同步频率、首次同步天数等</span>
                    </CollapsibleTrigger>
                    <CollapsibleContent className="space-y-4 pt-3 pl-3 border-l-2 border-muted ml-2 mt-2">
                      {/* 同步开关 */}
                      <div className="flex items-center justify-between">
                        <Label htmlFor="sync_enabled">启用自动同步</Label>
                        <Switch
                          id="sync_enabled"
                          checked={formData.sync_enabled}
                          onCheckedChange={(checked) => setFormData(prev => ({ ...prev, sync_enabled: checked }))}
                        />
                      </div>
                      {/* 同步频率 */}
                      {formData.sync_enabled && (
                        <div className="space-y-2">
                          <Label>同步频率（分钟）</Label>
                          <Select
                            value={String(formData.sync_interval)}
                            onValueChange={(v) => setFormData(prev => ({ ...prev, sync_interval: Number(v) }))}
                          >
                            <SelectTrigger><SelectValue /></SelectTrigger>
                            <SelectContent>
                              {[1, 2, 5, 10, 15, 30, 60].map((m) => (
                                <SelectItem key={m} value={String(m)}>{m} 分钟</SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>
                      )}
                      {/* 首次同步天数 */}
                      <div className="space-y-2">
                        <Label>首次同步天数</Label>
                        <Select
                          value={String(formData.first_sync_days)}
                          onValueChange={(v) => setFormData(prev => ({ ...prev, first_sync_days: Number(v) }))}
                        >
                          <SelectTrigger><SelectValue /></SelectTrigger>
                          <SelectContent>
                            {[7, 14, 30, 60, 90, 180, 365, 0].map((d) => (
                              <SelectItem key={d} value={String(d)}>{d === 0 ? '全部邮件' : `最近 ${d} 天`}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      {/* 删除策略 */}
                      <div className="space-y-2">
                        <Label>删除策略</Label>
                        <Select
                          value={formData.server_delete_policy}
                          onValueChange={(v) => setFormData(prev => ({ ...prev, server_delete_policy: v }))}
                        >
                          <SelectTrigger><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="off">仅本地删除</SelectItem>
                            <SelectItem value="soft">同步删除到服务器</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </CollapsibleContent>
                  </Collapsible>
                </>
              )}


              {/* 编辑模式设置 */}
              {isEditMode && !isBatchImportMode && (
                <>
                  {/* 更新凭证 */}
                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
                      <ChevronRight className="h-4 w-4 transition-transform ui-open:rotate-90" />
                      更新凭证
                    </CollapsibleTrigger>
                    <CollapsibleContent className="space-y-2 pt-2">
                      <Label htmlFor="edit_password">新密码 / 授权码</Label>
                      <Input
                        id="edit_password"
                        type="password"
                        placeholder="留空则不修改"
                        value={formData.password}
                        onChange={(e) => setFormData(prev => ({ ...prev, password: e.target.value }))}
                      />
                    </CollapsibleContent>
                  </Collapsible>

                  {/* 分组设置 */}
                  <div className="space-y-2">
                    <Label>分组</Label>
                    <GroupSelector
                      value={formData.group_id}
                      onChange={(id) => setFormData(prev => ({ ...prev, group_id: id }))}
                    />
                  </div>

                  {/* 同步设置 */}
                  <Collapsible defaultOpen>
                    <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
                      <ChevronDown className="h-4 w-4 transition-transform ui-closed:rotate-[-90deg]" />
                      同步设置
                    </CollapsibleTrigger>
                    <CollapsibleContent className="space-y-4 pt-2">
                      <div className="flex items-center justify-between">
                        <Label htmlFor="edit_sync_enabled">启用自动同步</Label>
                        <Switch
                          id="edit_sync_enabled"
                          checked={formData.sync_enabled}
                          onCheckedChange={(checked) => setFormData(prev => ({ ...prev, sync_enabled: checked }))}
                        />
                      </div>
                      {formData.sync_enabled && (
                        <div className="space-y-2">
                          <Label>同步频率（分钟）</Label>
                          <Select
                            value={String(formData.sync_interval)}
                            onValueChange={(v) => setFormData(prev => ({ ...prev, sync_interval: Number(v) }))}
                          >
                            <SelectTrigger><SelectValue /></SelectTrigger>
                            <SelectContent>
                              {[1, 2, 5, 10, 15, 30, 60].map((m) => (
                                <SelectItem key={m} value={String(m)}>{m} 分钟</SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>
                      )}
                      <div className="space-y-2">
                        <Label>删除策略</Label>
                        <Select
                          value={formData.server_delete_policy}
                          onValueChange={(v) => setFormData(prev => ({ ...prev, server_delete_policy: v }))}
                        >
                          <SelectTrigger><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="off">仅本地删除</SelectItem>
                            <SelectItem value="soft">同步删除到服务器</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </CollapsibleContent>
                  </Collapsible>

                  {/* SMTP 发件设置（编辑模式） */}
                  <Collapsible open={showSmtpSection} onOpenChange={setShowSmtpSection}>
                    <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
                      <Send className="h-4 w-4" />
                      发件设置
                      {smtpConfig.smtp_enabled && <Badge variant="secondary" className="ml-2">已启用</Badge>}
                    </CollapsibleTrigger>
                    <CollapsibleContent className="space-y-4 pt-2">
                      {isLoadingSmtp ? (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          加载 SMTP 配置...
                        </div>
                      ) : (
                        <>
                          <div className="flex items-center justify-between">
                            <div>
                              <Label htmlFor="edit_smtp_enabled">启用发件功能</Label>
                              <p className="text-xs text-muted-foreground">使用 SMTP 发送邮件</p>
                            </div>
                            <Switch
                              id="edit_smtp_enabled"
                              checked={smtpConfig.smtp_enabled}
                              onCheckedChange={(checked) => setSmtpConfig({ smtp_enabled: checked })}
                            />
                          </div>
                          {smtpServerConfig.host && (
                            <div className="p-3 rounded-lg bg-muted/30 text-sm space-y-1">
                              <div className="flex justify-between">
                                <span className="text-muted-foreground">服务器</span>
                                <span>{smtpServerConfig.host}:{smtpServerConfig.port}</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-muted-foreground">加密</span>
                                <span>{getEncryptionLabel(smtpServerConfig.encryption)}</span>
                              </div>
                              {smtpServerConfig.fromProvider && (
                                <div className="flex justify-between">
                                  <span className="text-muted-foreground">来源</span>
                                  <span>继承自 {smtpServerConfig.providerName}</span>
                                </div>
                              )}
                            </div>
                          )}
                          {smtpConfig.smtp_enabled && smtpServerConfig.host && (
                            <div className="flex items-center gap-2">
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={handleTestSmtp}
                                disabled={isTestingSmtp}
                              >
                                {isTestingSmtp ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
                                测试连接
                              </Button>
                              {smtpTestResult && (
                                <span className={`text-sm ${smtpTestResult.success ? 'text-green-600' : 'text-red-600'}`}>
                                  {smtpTestResult.success ? <CheckCircle2 className="h-4 w-4 inline mr-1" /> : <XCircle className="h-4 w-4 inline mr-1" />}
                                  {smtpTestResult.message}
                                </span>
                              )}
                            </div>
                          )}
                        </>
                      )}
                    </CollapsibleContent>
                  </Collapsible>
                </>
              )}

              {/* 批量导入模式 */}
              {isBatchImportMode && (
                <>
                  <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                      批量导入适用于短效邮箱，每行一个账户，格式：邮箱{batchSeparator}密码
                    </AlertDescription>
                  </Alert>
                  <div className="space-y-2">
                    <Label>分隔符</Label>
                    <Select value={batchSeparator} onValueChange={setBatchSeparator}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="----">----（默认）</SelectItem>
                        <SelectItem value=":">:（冒号）</SelectItem>
                        <SelectItem value="|">|（竖线）</SelectItem>
                        <SelectItem value="\t">Tab</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>账户列表</Label>
                    <Textarea
                      placeholder={`user1@example.com${batchSeparator}password1\nuser2@example.com${batchSeparator}password2`}
                      value={batchAccountsText}
                      onChange={(e) => setBatchAccountsText(e.target.value)}
                      rows={8}
                    />
                    <p className="text-xs text-muted-foreground">
                      已输入 {batchAccountsText.split('\n').filter(l => l.trim() && l.includes(batchSeparator)).length} 个有效账户
                    </p>
                  </div>
                  {isBatchImporting && (
                    <div className="space-y-2">
                      <Progress value={batchImportProgress} />
                      <p className="text-xs text-center text-muted-foreground">导入中... {batchImportProgress}%</p>
                    </div>
                  )}
                  {batchImportResult && (
                    <div className="space-y-2">
                      <div className="flex gap-4 text-sm">
                        <span className="text-green-600">成功: {batchImportResult.success}</span>
                        <span className="text-red-600">失败: {batchImportResult.failed}</span>
                      </div>
                      {batchImportResult.failed > 0 && (
                        <ScrollArea className="h-32 rounded border p-2">
                          {batchImportResult.results.filter(r => r.status === 'failed').map((r, i) => (
                            <div key={i} className="text-xs text-red-600 py-1">
                              {r.email}: {r.error}
                            </div>
                          ))}
                        </ScrollArea>
                      )}
                    </div>
                  )}
                </>
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
