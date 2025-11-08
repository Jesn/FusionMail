import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
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
import { AlertCircle, CheckCircle2, XCircle, Loader2, Upload } from 'lucide-react';
import { CreateAccountRequest, accountService } from '../../services/accountService';
import { Account } from '../../types';
import { useProviders } from '../../hooks/useProviders';
import { OAuth2AuthButton } from '../auth/OAuth2AuthButton';
import toast from 'react-hot-toast';


interface AccountFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateAccountRequest | Partial<CreateAccountRequest>) => Promise<void>;
  account?: Account | null;
}

export const AccountForm = ({ open, onClose, onSubmit, account }: AccountFormProps) => {
  const isEditMode = !!account;
  const navigate = useNavigate();
  const { providers, getProviderByEmail, getProviderByName } = useProviders();
  
  const [formData, setFormData] = useState<CreateAccountRequest>({
    email: '',
    provider: 'qq',
    protocol: 'imap',
    auth_type: 'password',
    password: '',
    sync_enabled: true,
    sync_interval: 2,
    // 通用邮箱配置
    imap_host: '',
    imap_port: 993,
    pop3_host: '',
    pop3_port: 995,
    encryption: 'ssl',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // 批量导入相关状态
  const [batchAccountsText, setBatchAccountsText] = useState('');
  const [batchSeparator, setBatchSeparator] = useState('----'); // 分隔符，默认为 ----
  const [isBatchImporting, setIsBatchImporting] = useState(false);
  const [batchImportResult, setBatchImportResult] = useState<{
    success: number;
    failed: number;
    results: Array<{
      email: string;
      status: 'success' | 'failed';
      error?: string;
    }>;
  } | null>(null);
  const [batchImportProgress, setBatchImportProgress] = useState(0);
  
  // 判断是否为批量导入模式
  const isBatchImportMode = formData.protocol === 'batch_import' && !isEditMode;

  // 当 account 变化时，更新表单数据
  useEffect(() => {
    if (account) {
      setFormData({
        email: account.email,
        provider: account.provider,
        protocol: account.protocol,
        auth_type: account.auth_type,
        password: '', // 编辑时不显示密码
        sync_enabled: account.sync_enabled,
        sync_interval: account.sync_interval,
        // 通用邮箱配置 - 编辑时加载现有配置
        imap_host: account.imap_host || '',
        imap_port: account.imap_port || 993,
        pop3_host: account.pop3_host || '',
        pop3_port: account.pop3_port || 995,
        encryption: account.encryption || 'ssl',
      });
    } else {
      // 重置为默认值
      setFormData({
        email: '',
        provider: 'qq',
        protocol: 'imap',
        auth_type: 'password',
        password: '',
        sync_enabled: true,
        sync_interval: 2,
        // 通用邮箱配置
        imap_host: '',
        imap_port: 993,
        pop3_host: '',
        pop3_port: 995,
        encryption: 'ssl',
      });
    }
    // 重置批量导入状态
    setBatchAccountsText('');
    setBatchSeparator('----'); // 重置为默认分隔符
    setBatchImportResult(null);
    setBatchImportProgress(0);
  }, [account, open]);

  // 当 providers 加载完成后，更新当前选择的提供商配置
  useEffect(() => {
    if (!isEditMode && providers.length > 0 && formData.provider) {
      const providerInfo = getProviderByName(formData.provider);
      if (providerInfo && formData.protocol !== providerInfo.recommended_protocol) {
        // 如果当前协议不是推荐协议，更新为推荐协议
        setFormData(prev => ({
          ...prev,
          protocol: providerInfo.recommended_protocol,
          auth_type: providerInfo.recommended_protocol === 'oauth2' ? 'oauth2' : 'password',
          imap_host: providerInfo.imap_host || prev.imap_host,
          imap_port: providerInfo.imap_port || prev.imap_port,
          pop3_host: providerInfo.pop3_host || prev.pop3_host,
          pop3_port: providerInfo.pop3_port || prev.pop3_port,
        }));
      }
    }
  }, [providers, formData.provider, isEditMode, getProviderByName]);

  // 处理邮箱地址变化，自动识别提供商
  const handleEmailChange = (email: string) => {
    setFormData(prev => ({ ...prev, email }));

    if (!isEditMode && email.includes('@')) {
      const recommendedProvider = getProviderByEmail(email);
      if (recommendedProvider) {
        setFormData(prev => ({
          ...prev,
          provider: recommendedProvider.name,
          protocol: recommendedProvider.recommended_protocol,
          // 如果是预设提供商，填充服务器配置
          imap_host: recommendedProvider.imap_host || '',
          imap_port: recommendedProvider.imap_port || 993,
          pop3_host: recommendedProvider.pop3_host || '',
          pop3_port: recommendedProvider.pop3_port || 995,
        }));
      } else {
        // 如果还没有加载到提供商信息，根据邮箱域名手动识别
        const domain = email.split('@')[1].toLowerCase();

        if (domain === 'outlook.com' || domain === 'hotmail.com' || domain === 'live.com') {
          setFormData(prev => ({
            ...prev,
            provider: 'outlook',
            protocol: 'oauth2',
            auth_type: 'oauth2',
            imap_host: 'outlook.office365.com',
            imap_port: 993,
          }));
        } else if (domain === 'gmail.com' || domain === 'googlemail.com') {
          setFormData(prev => ({
            ...prev,
            provider: 'gmail',
            protocol: 'oauth2',
            auth_type: 'oauth2',
          }));
        } else if (domain === 'qq.com') {
          setFormData(prev => ({
            ...prev,
            provider: 'qq',
            protocol: 'imap',
            auth_type: 'password',
            imap_host: 'imap.qq.com',
            imap_port: 993,
          }));
        } else if (domain === '163.com' || domain === '126.com') {
          setFormData(prev => ({
            ...prev,
            provider: '163',
            protocol: 'imap',
            auth_type: 'password',
            imap_host: 'imap.163.com',
            imap_port: 993,
          }));
        } else if (domain === 'icloud.com' || domain === 'me.com' || domain === 'mac.com') {
          setFormData(prev => ({
            ...prev,
            provider: 'icloud',
            protocol: 'imap',
            auth_type: 'password',
            imap_host: 'imap.mail.me.com',
            imap_port: 993,
          }));
        }
      }
    }
  };

  // 处理提供商变化
  const handleProviderChange = (provider: string) => {
    const providerInfo = getProviderByName(provider);

    // 如果获取到提供商信息，使用提供商配置
    if (providerInfo) {
      setFormData(prev => ({
        ...prev,
        provider,
        protocol: providerInfo.recommended_protocol,
        // 根据协议自动设置认证类型
        auth_type: providerInfo.recommended_protocol === 'oauth2' ? 'oauth2' : 'password',
        // 填充服务器配置
        imap_host: providerInfo.imap_host || '',
        imap_port: providerInfo.imap_port || 993,
        pop3_host: providerInfo.pop3_host || '',
        pop3_port: providerInfo.pop3_port || 995,
      }));
    } else {
      // 如果还没有加载到提供商信息，根据提供商名称手动设置推荐配置
      let recommendedProtocol = 'imap';
      let authType = 'password';
      let imapHost = '';
      let imapPort = 993;

      if (provider === 'gmail' || provider === 'outlook') {
        recommendedProtocol = 'oauth2';
        authType = 'oauth2';
        if (provider === 'outlook') {
          imapHost = 'outlook.office365.com';
        }
      } else if (provider === 'icloud') {
        imapHost = 'imap.mail.me.com';
      } else if (provider === 'qq') {
        imapHost = 'imap.qq.com';
      } else if (provider === '163') {
        imapHost = 'imap.163.com';
      }

      setFormData(prev => ({
        ...prev,
        provider,
        protocol: recommendedProtocol,
        auth_type: authType,
        imap_host: imapHost,
        imap_port: imapPort,
      }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // 批量导入模式
    if (isBatchImportMode) {
      await handleBatchImport();
      return;
    }
    
    // 普通模式
    setIsSubmitting(true);
    try {
      if (isEditMode) {
        // 编辑模式：只提交可修改的字段
        const updateData: Partial<CreateAccountRequest> = {
          sync_enabled: formData.sync_enabled,
          sync_interval: formData.sync_interval,
        };
        // 如果输入了新密码，则包含密码
        if (formData.password) {
          updateData.password = formData.password;
        }
        // 如果是通用邮箱，包含服务器配置
        if (formData.provider === 'generic') {
          updateData.imap_host = formData.imap_host;
          updateData.imap_port = formData.imap_port;
          updateData.pop3_host = formData.pop3_host;
          updateData.pop3_port = formData.pop3_port;
          updateData.encryption = formData.encryption;
        }
        await onSubmit(updateData);
      } else {
        // 创建模式：提交所有字段
        await onSubmit(formData);
      }
      onClose();
    } catch (error) {
      // 错误已在 Hook 中处理
    } finally {
      setIsSubmitting(false);
    }
  };
  
  // 批量导入处理函数
  const handleBatchImport = async () => {
    // 验证分隔符
    if (!batchSeparator || batchSeparator.trim() === '') {
      toast.error('请设置有效的分隔符');
      return;
    }

    const accounts = batchAccountsText
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0 && line.includes(batchSeparator));
    
    if (accounts.length === 0) {
      toast.error('请输入有效的账号信息');
      return;
    }

    // 将自定义分隔符转换为标准分隔符（后端期望的格式）
    const normalizedAccounts = accounts.map(line => {
      if (batchSeparator === '----') {
        return line; // 已经是标准格式
      }
      // 替换自定义分隔符为标准分隔符
      return line.split(batchSeparator).join('----');
    });

    setIsBatchImporting(true);
    setBatchImportProgress(0);

    try {
      // 模拟进度更新
      const progressInterval = setInterval(() => {
        setBatchImportProgress(prev => Math.min(prev + 10, 90));
      }, 500);

      const result = await accountService.batchImport(
        normalizedAccounts,
        formData.sync_enabled,
        formData.sync_interval
      );
      
      clearInterval(progressInterval);
      setBatchImportProgress(100);
      setBatchImportResult(result);
      
      // 显示结果通知
      if (result.success > 0) {
        toast.success(`成功导入 ${result.success} 个账户`);
      }
      if (result.failed > 0) {
        toast.error(`${result.failed} 个账户导入失败`);
      }
    } catch (error) {
      console.error('批量导入失败:', error);
      setBatchImportResult({
        success: 0,
        failed: normalizedAccounts.length,
        results: normalizedAccounts.map(acc => ({
          email: acc.split('----')[0],
          status: 'failed',
          error: '导入失败',
        })),
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
            {isEditMode 
              ? '修改账户的同步设置' 
              : isBatchImportMode 
              ? '批量导入多个短效邮箱账户' 
              : '添加您的邮箱账户以开始接收邮件'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto">
          <form onSubmit={handleSubmit} className="flex flex-col h-full">
            <div className="space-y-4 py-4 px-1 flex-1 overflow-y-auto min-h-0">
            {/* 邮箱提供商 */}
            <div className="space-y-2">
              <Label htmlFor="provider">邮箱提供商 *</Label>
              <Select
                value={formData.provider}
                onValueChange={handleProviderChange}
                disabled={isEditMode}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {providers.map((provider) => (
                    <SelectItem key={provider.name} value={provider.name}>
                      {provider.display_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* 协议 */}
            <div className="space-y-2">
              <Label htmlFor="protocol">协议 *</Label>
              <Select
                key={`protocol-${formData.provider}-${formData.protocol}`}
                value={formData.protocol}
                onValueChange={(value) => {
                  setFormData({ ...formData, protocol: value });
                  // 根据协议自动设置认证类型
                  if (value === 'oauth2') {
                    setFormData(prev => ({ ...prev, protocol: value, auth_type: 'oauth2' }));
                  } else if (value === 'batch_import') {
                    setFormData(prev => ({ ...prev, protocol: value, auth_type: 'quick' }));
                    // 重置批量导入状态
                    setBatchAccountsText('');
                    setBatchSeparator('----'); // 重置为默认分隔符
                    setBatchImportResult(null);
                    setBatchImportProgress(0);
                  } else {
                    setFormData(prev => ({ ...prev, protocol: value, auth_type: 'password' }));
                  }
                }}
                disabled={isEditMode}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择协议" />
                </SelectTrigger>
                <SelectContent>
                  {/* Gmail 和 Outlook 支持 OAuth2 */}
                  {(formData.provider === 'gmail' || formData.provider === 'outlook') && (
                    <SelectItem value="oauth2">
                      OAuth2（推荐 - 更安全）
                    </SelectItem>
                  )}
                  {/* Outlook 支持批量导入 */}
                  {formData.provider === 'outlook' && (
                    <SelectItem value="batch_import">
                      批量导入（短效邮箱）
                    </SelectItem>
                  )}
                  <SelectItem value="imap">IMAP</SelectItem>
                  <SelectItem value="pop3">POP3</SelectItem>
                </SelectContent>
              </Select>
              {formData.protocol === 'oauth2' && (
                <p className="text-xs text-blue-600 dark:text-blue-400">
                  OAuth2 认证无需密码，通过官方授权页面安全登录
                </p>
              )}
              {formData.protocol === 'batch_import' && (
                <p className="text-xs text-amber-600 dark:text-amber-400">
                  批量导入模式：适用于导入多个短效邮箱账户
                </p>
              )}
            </div>

            {/* 邮箱地址 - 根据协议类型决定是否显示 */}
            {/* 编辑模式下始终显示，新建模式下根据协议类型决定 */}
            {!isBatchImportMode && (isEditMode || formData.protocol !== 'oauth2') && (
              <div className="space-y-2">
                <Label htmlFor="email">邮箱地址 *</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="your@example.com"
                  value={formData.email}
                  onChange={(e) => handleEmailChange(e.target.value)}
                  required
                  disabled={isEditMode}
                />
                {!isEditMode && (
                  <p className="text-xs text-muted-foreground">
                    请输入完整的邮箱地址
                  </p>
                )}
              </div>
            )}

            {/* 批量导入界面 */}
            {isBatchImportMode && !batchImportResult && (
              <div className="space-y-4">
                {/* 格式说明 */}
                <Alert>
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    <div className="space-y-2">
                      <p className="font-medium">账号格式说明：</p>
                      <code className="block rounded bg-muted p-2 text-xs">
                        email{batchSeparator}password{batchSeparator}refresh_token{batchSeparator}client_id
                      </code>
                      <p className="text-xs text-muted-foreground">
                        每行一个账号，使用分隔符分隔各个字段
                      </p>
                    </div>
                  </AlertDescription>
                </Alert>

                {/* 分隔符设置 */}
                <div className="space-y-2">
                  <Label htmlFor="batch_separator">分隔符</Label>
                  <div className="flex gap-2">
                    <Input
                      id="batch_separator"
                      type="text"
                      placeholder="----"
                      value={batchSeparator}
                      onChange={(e) => setBatchSeparator(e.target.value)}
                      className="max-w-[150px] font-mono"
                      disabled={isBatchImporting}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => setBatchSeparator('----')}
                      disabled={isBatchImporting || batchSeparator === '----'}
                    >
                      重置为默认
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    自定义字段分隔符，默认为 ----
                  </p>
                </div>

                {/* 同步设置 */}
                <div className="space-y-4 rounded-lg border p-4 bg-blue-50 dark:bg-blue-900/20">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="batch_sync_enabled">启用自动同步</Label>
                    <Switch
                      id="batch_sync_enabled"
                      checked={formData.sync_enabled}
                      onCheckedChange={(checked) =>
                        setFormData({ ...formData, sync_enabled: checked })
                      }
                      disabled={isBatchImporting}
                    />
                  </div>

                  {formData.sync_enabled && (
                    <div className="space-y-2">
                      <Label htmlFor="batch_sync_interval">同步频率（分钟）</Label>
                      <div className="flex items-center gap-2">
                        <Input
                          id="batch_sync_interval"
                          type="number"
                          min="1"
                          max="60"
                          value={formData.sync_interval}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              sync_interval: parseInt(e.target.value, 10) || 2,
                            })
                          }
                          className="max-w-[120px]"
                          disabled={isBatchImporting}
                        />
                        <span className="text-sm text-muted-foreground">分钟</span>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        所有导入的账户将使用此同步频率（1-60 分钟）
                      </p>
                    </div>
                  )}
                </div>

                {/* 输入框 */}
                <div className="space-y-2">
                  <Label htmlFor="batch_accounts">账号列表 *</Label>
                  <Textarea
                    id="batch_accounts"
                    placeholder={`粘贴账号字符串，每行一个...\n例如：email${batchSeparator}password${batchSeparator}refresh_token${batchSeparator}client_id`}
                    value={batchAccountsText}
                    onChange={(e) => setBatchAccountsText(e.target.value)}
                    className="min-h-[200px] font-mono text-sm"
                    disabled={isBatchImporting}
                  />
                  {batchAccountsText.split('\n').filter(line => line.trim() && line.includes(batchSeparator)).length > 0 && (
                    <p className="text-sm text-muted-foreground">
                      已识别 {batchAccountsText.split('\n').filter(line => line.trim() && line.includes(batchSeparator)).length} 个账号
                    </p>
                  )}
                </div>

                {/* 进度条 */}
                {isBatchImporting && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span>正在导入...</span>
                      <span>{batchImportProgress}%</span>
                    </div>
                    <Progress value={batchImportProgress} />
                  </div>
                )}
              </div>
            )}

            {/* 批量导入结果 */}
            {isBatchImportMode && batchImportResult && (
              <div className="space-y-4">
                {/* 统计信息 */}
                <div className="grid grid-cols-2 gap-4">
                  <div className="rounded-lg border p-4">
                    <div className="flex items-center gap-2">
                      <CheckCircle2 className="h-5 w-5 text-green-600" />
                      <div>
                        <p className="text-2xl font-bold">{batchImportResult.success}</p>
                        <p className="text-sm text-muted-foreground">成功</p>
                      </div>
                    </div>
                  </div>
                  <div className="rounded-lg border p-4">
                    <div className="flex items-center gap-2">
                      <XCircle className="h-5 w-5 text-red-600" />
                      <div>
                        <p className="text-2xl font-bold">{batchImportResult.failed}</p>
                        <p className="text-sm text-muted-foreground">失败</p>
                      </div>
                    </div>
                  </div>
                </div>

                {/* 详细结果 */}
                <div className="space-y-2">
                  <p className="text-sm font-medium">导入详情：</p>
                  <ScrollArea className="h-[200px] rounded-md border">
                    <div className="p-4 space-y-2">
                      {batchImportResult.results.map((result, index) => (
                        <div
                          key={index}
                          className="flex items-start gap-2 rounded-lg border p-3"
                        >
                          {result.status === 'success' ? (
                            <CheckCircle2 className="h-4 w-4 text-green-600 mt-0.5" />
                          ) : (
                            <XCircle className="h-4 w-4 text-red-600 mt-0.5" />
                          )}
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium truncate">{result.email}</p>
                            {result.error && (
                              <p className="text-xs text-red-600 mt-1">{result.error}</p>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </ScrollArea>
                </div>
              </div>
            )}

            {/* 通用邮箱配置 */}
            {formData.provider === 'generic' && (
              <div className="space-y-4 p-4 border rounded-lg bg-gray-50 dark:bg-gray-800/50">
                <h4 className="font-medium text-sm text-gray-900 dark:text-white">
                  服务器配置
                </h4>
                <p className="text-xs text-gray-600 dark:text-gray-400">
                  请联系您的邮箱服务商获取正确的服务器配置信息
                </p>
                
                {formData.protocol === 'imap' && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="imap_host">IMAP 服务器 *</Label>
                      <Input
                        id="imap_host"
                        placeholder="imap.example.com"
                        value={formData.imap_host}
                        onChange={(e) =>
                          setFormData({ ...formData, imap_host: e.target.value })
                        }
                        className="w-full"
                        required
                      />
                    </div>
                    <div>
                      <Label htmlFor="imap_port">IMAP 端口 *</Label>
                      <Input
                        id="imap_port"
                        type="number"
                        placeholder="993"
                        value={formData.imap_port}
                        onChange={(e) =>
                          setFormData({ ...formData, imap_port: parseInt(e.target.value) || 993 })
                        }
                        required
                      />
                    </div>
                  </div>
                )}

                {formData.protocol === 'pop3' && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="pop3_host">POP3 服务器 *</Label>
                      <Input
                        id="pop3_host"
                        placeholder="pop3.example.com"
                        value={formData.pop3_host}
                        onChange={(e) =>
                          setFormData({ ...formData, pop3_host: e.target.value })
                        }
                        required
                      />
                    </div>
                    <div>
                      <Label htmlFor="pop3_port">POP3 端口 *</Label>
                      <Input
                        id="pop3_port"
                        type="number"
                        placeholder="995"
                        value={formData.pop3_port}
                        onChange={(e) =>
                          setFormData({ ...formData, pop3_port: parseInt(e.target.value) || 995 })
                        }
                        required
                      />
                    </div>
                  </div>
                )}

                <div>
                  <Label htmlFor="encryption">加密方式 *</Label>
                  <Select
                    value={formData.encryption}
                    onValueChange={(value) =>
                      setFormData({ ...formData, encryption: value })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ssl">SSL/TLS (推荐)</SelectItem>
                      <SelectItem value="starttls">STARTTLS</SelectItem>
                      <SelectItem value="none">无加密</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            )}

            {/* OAuth2 认证按钮 - 批量导入模式下隐藏 */}
            {!isEditMode && !isBatchImportMode && formData.protocol === 'oauth2' && (
              <div className="space-y-4 p-4 border rounded-lg bg-blue-50 dark:bg-blue-900/20">
                <div>
                  <h4 className="font-medium text-sm text-gray-900 dark:text-white">
                    OAuth2 安全认证
                  </h4>
                  <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                    点击下方按钮，通过官方授权页面安全登录，无需输入邮箱地址和密码
                  </p>
                </div>
                
                <OAuth2AuthButton
                  provider={formData.provider === 'gmail' ? 'google' : 'microsoft'}
                  onSuccess={() => {
                    // OAuth2 成功后关闭表单
                    onClose();
                    // 刷新页面以加载新添加的账户
                    // 如果在账户页面，刷新当前页面；否则跳转到账户页面
                    if (window.location.pathname === '/accounts') {
                      window.location.reload();
                    } else {
                      navigate('/accounts');
                    }
                  }}
                  onError={(error) => {
                    console.error('OAuth2 error:', error);
                  }}
                />
              </div>
            )}

            {/* 密码/授权码（仅在非 OAuth2 和非批量导入模式下显示） */}
            {!isBatchImportMode && formData.protocol !== 'oauth2' && (
              <div className="space-y-2">
                <Label htmlFor="password">
                  {isEditMode ? '新密码/授权码（留空则不修改）' : '密码/授权码 *'}
                </Label>
                <Input
                  id="password"
                  type="password"
                  placeholder={isEditMode ? '留空则不修改密码' : '请输入密码或授权码'}
                  value={formData.password}
                  onChange={(e) =>
                    setFormData({ ...formData, password: e.target.value })
                  }
                  required={!isEditMode}
                />
                {!isEditMode && (
                  <p className="text-xs text-muted-foreground">
                    {formData.provider === 'qq' || formData.provider === '163' 
                      ? 'QQ/163 邮箱请使用授权码，而非登录密码'
                      : formData.provider === 'gmail' || formData.provider === 'outlook'
                      ? '建议使用应用专用密码，或切换到 OAuth2 协议获得更好的安全性'
                      : '请输入邮箱密码或授权码'}
                  </p>
                )}
              </div>
            )}

            {/* 同步设置 - 批量导入模式下隐藏 */}
            {!isBatchImportMode && (
            <div className="space-y-4 rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="sync_enabled">启用自动同步</Label>
                <Switch
                  id="sync_enabled"
                  checked={formData.sync_enabled}
                  onCheckedChange={(checked) =>
                    setFormData({ ...formData, sync_enabled: checked })
                  }
                />
              </div>

              {formData.sync_enabled && (
                <div className="space-y-2">
                  <Label htmlFor="sync_interval">同步频率（分钟）</Label>
                  <Input
                    id="sync_interval"
                    type="number"
                    min="1"
                    max="60"
                    value={formData.sync_interval}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        sync_interval: parseInt(e.target.value, 10),
                      })
                    }
                  />
                </div>
              )}
            </div>
            )}
            </div>

            <DialogFooter className="flex-shrink-0 mt-4">
              {/* 批量导入结果显示时只显示完成按钮 */}
              {isBatchImportMode && batchImportResult ? (
                <Button 
                  type="button" 
                  onClick={() => {
                    onClose();
                    // 刷新页面以加载新导入的账户
                    if (batchImportResult.success > 0) {
                      window.location.reload();
                    }
                  }}
                >
                  完成
                </Button>
              ) : (
                <>
                  <Button 
                    type="button" 
                    variant="outline" 
                    onClick={onClose}
                    disabled={isBatchImporting || isSubmitting}
                  >
                    取消
                  </Button>
                  <Button 
                    type="submit" 
                    disabled={isBatchImporting || isSubmitting || (isBatchImportMode && !batchAccountsText.trim())}
                  >
                    {isBatchImporting ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        导入中...
                      </>
                    ) : isSubmitting ? (
                      isEditMode ? '保存中...' : '添加中...'
                    ) : isBatchImportMode ? (
                      <>
                        <Upload className="mr-2 h-4 w-4" />
                        开始导入
                      </>
                    ) : (
                      isEditMode ? '保存' : '添加账户'
                    )}
                  </Button>
                </>
              )}
            </DialogFooter>
          </form>
        </div>
      </DialogContent>
    </Dialog>
  );
};