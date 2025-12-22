import { useState, useEffect, useRef } from 'react';
import { Upload, CheckCircle2, XCircle, Loader2, ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '../ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Textarea } from '../ui/textarea';
import { Progress } from '../ui/progress';
import { Label } from '../ui/label';
import { Switch } from '../ui/switch';
import { Slider } from '../ui/slider';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { useGroupStore } from '../../stores/groupStore';

// 批量导入配置
export interface BatchImportConfig {
  accounts: string[];
  groupId?: number;
  syncEnabled: boolean;
  syncInterval: number;
  firstSyncDays: number;
}

interface BatchImportDialogProps {
  open: boolean;
  onClose: () => void;
  onImport: (config: BatchImportConfig) => Promise<BatchImportResult>;
}

export interface BatchImportResult {
  success: number;
  failed: number;
  results: Array<{
    email: string;
    status: 'success' | 'failed';
    error?: string;
  }>;
}

export const BatchImportDialog = ({ open, onClose, onImport }: BatchImportDialogProps) => {
  const [accountsText, setAccountsText] = useState('');
  const [isImporting, setIsImporting] = useState(false);
  const [importResult, setImportResult] = useState<BatchImportResult | null>(null);
  const [progress, setProgress] = useState(0);
  
  // 分组选择
  const [selectedGroupId, setSelectedGroupId] = useState<number | undefined>(undefined);
  const { groups, fetchGroups } = useGroupStore();
  
  // 高级设置
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [syncEnabled, setSyncEnabled] = useState(true);
  const [syncInterval, setSyncInterval] = useState(2);
  const [firstSyncDays, setFirstSyncDays] = useState(7);
  
  // 滚动容器引用
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // 加载分组列表
  useEffect(() => {
    if (open) {
      fetchGroups();
    }
  }, [open, fetchGroups]);

  // 展开高级设置时自动滚动到底部
  useEffect(() => {
    if (showAdvanced && scrollContainerRef.current) {
      setTimeout(() => {
        scrollContainerRef.current?.scrollTo({
          top: scrollContainerRef.current.scrollHeight,
          behavior: 'smooth'
        });
      }, 100);
    }
  }, [showAdvanced]);

  const handleClose = () => {
    if (!isImporting) {
      setAccountsText('');
      setImportResult(null);
      setProgress(0);
      setSelectedGroupId(undefined);
      setShowAdvanced(false);
      setSyncEnabled(true);
      setSyncInterval(2);
      setFirstSyncDays(7);
      onClose();
    }
  };

  const parseAccounts = (text: string): string[] => {
    return text
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0 && line.includes('----'));
  };

  const handleImport = async () => {
    const accounts = parseAccounts(accountsText);
    
    if (accounts.length === 0) {
      return;
    }

    setIsImporting(true);
    setProgress(0);

    try {
      // 模拟进度更新
      const progressInterval = setInterval(() => {
        setProgress(prev => Math.min(prev + 10, 90));
      }, 500);

      const config: BatchImportConfig = {
        accounts,
        groupId: selectedGroupId,
        syncEnabled,
        syncInterval,
        firstSyncDays,
      };

      const result = await onImport(config);
      
      clearInterval(progressInterval);
      setProgress(100);
      setImportResult(result);
    } catch (error) {
      console.error('批量导入失败:', error);
      setImportResult({
        success: 0,
        failed: parseAccounts(accountsText).length,
        results: parseAccounts(accountsText).map(acc => ({
          email: acc.split('----')[0],
          status: 'failed',
          error: '导入失败',
        })),
      });
    } finally {
      setIsImporting(false);
    }
  };

  const accounts = parseAccounts(accountsText);
  const isValid = accounts.length > 0;

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>批量导入 Outlook 邮箱</DialogTitle>
          <DialogDescription>
            粘贴 Outlook/Hotmail 短效邮箱账号字符串，每行一个账号
          </DialogDescription>
        </DialogHeader>

        <div ref={scrollContainerRef} className="flex-1 overflow-y-auto pr-2 min-h-0">
          {!importResult ? (
            <div className="space-y-3 pb-2">
              {/* 格式说明 - 更紧凑 */}
              <div className="rounded-md border bg-muted/50 p-2">
                <p className="text-xs text-muted-foreground mb-1">
                  <span className="font-medium">格式：</span>
                  <code className="ml-1 rounded bg-background px-1">email----password----refresh_token----client_id</code>
                </p>
                <p className="text-xs text-muted-foreground">每行一个账号</p>
              </div>

              {/* 输入框 */}
              <div className="space-y-1">
                <Textarea
                  placeholder="粘贴账号字符串，每行一个..."
                  value={accountsText}
                  onChange={(e) => setAccountsText(e.target.value)}
                  className="min-h-[80px] max-h-[100px] font-mono text-sm"
                  disabled={isImporting}
                />
                {accounts.length > 0 && (
                  <p className="text-xs text-muted-foreground">
                    已识别 {accounts.length} 个账号
                  </p>
                )}
              </div>

              {/* 分组选择 */}
              <div className="space-y-1">
                <Label className="text-sm">分组</Label>
                <Select
                  value={selectedGroupId?.toString() || 'none'}
                  onValueChange={(value) => setSelectedGroupId(value === 'none' ? undefined : parseInt(value))}
                  disabled={isImporting}
                >
                  <SelectTrigger className="h-9">
                    <SelectValue placeholder="选择分组（可选）" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">未分组</SelectItem>
                    {groups.map((group) => (
                      <SelectItem key={group.id} value={group.id.toString()}>
                        {group.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* 高级设置 */}
              <Collapsible open={showAdvanced} onOpenChange={setShowAdvanced}>
                <CollapsibleTrigger asChild>
                  <Button variant="ghost" className="w-full justify-between p-2 h-8">
                    <span className="text-sm font-medium">高级设置</span>
                    {showAdvanced ? (
                      <ChevronUp className="h-4 w-4" />
                    ) : (
                      <ChevronDown className="h-4 w-4" />
                    )}
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent className="space-y-3 pt-2 pb-1">
                  {/* 启用同步 */}
                  <div className="flex items-center justify-between">
                    <div className="space-y-0">
                      <Label className="text-sm">启用自动同步</Label>
                      <p className="text-xs text-muted-foreground">
                        自动同步邮件到本地
                      </p>
                    </div>
                    <Switch
                      checked={syncEnabled}
                      onCheckedChange={setSyncEnabled}
                      disabled={isImporting}
                    />
                  </div>

                  {/* 同步频率 */}
                  {syncEnabled && (
                    <div className="space-y-1.5">
                      <div className="flex items-center justify-between">
                        <Label className="text-sm">同步频率</Label>
                        <span className="text-xs text-muted-foreground">
                          每 {syncInterval} 分钟
                        </span>
                      </div>
                      <Slider
                        value={[syncInterval]}
                        onValueChange={([value]) => setSyncInterval(value)}
                        min={1}
                        max={60}
                        step={1}
                        disabled={isImporting}
                        className="py-1"
                      />
                      <p className="text-xs text-muted-foreground">
                        建议 2-5 分钟，过于频繁可能被限制
                      </p>
                    </div>
                  )}

                  {/* 首次同步天数 */}
                  {syncEnabled && (
                    <div className="space-y-1.5">
                      <div className="flex items-center justify-between">
                        <Label className="text-sm">首次同步天数</Label>
                        <span className="text-xs text-muted-foreground">
                          最近 {firstSyncDays} 天
                        </span>
                      </div>
                      <Slider
                        value={[firstSyncDays]}
                        onValueChange={([value]) => setFirstSyncDays(value)}
                        min={1}
                        max={90}
                        step={1}
                        disabled={isImporting}
                        className="py-1"
                      />
                      <p className="text-xs text-muted-foreground">
                        首次同步时获取最近多少天的邮件
                      </p>
                    </div>
                  )}
                </CollapsibleContent>
              </Collapsible>

              {/* 进度条 */}
              {isImporting && (
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-sm">
                    <span>正在导入...</span>
                    <span>{progress}%</span>
                  </div>
                  <Progress value={progress} />
                </div>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              {/* 统计信息 */}
              <div className="grid grid-cols-2 gap-4">
                <div className="rounded-lg border p-4">
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="h-5 w-5 text-green-600" />
                    <div>
                      <p className="text-2xl font-bold">{importResult.success}</p>
                      <p className="text-sm text-muted-foreground">成功</p>
                    </div>
                  </div>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="flex items-center gap-2">
                    <XCircle className="h-5 w-5 text-red-600" />
                    <div>
                      <p className="text-2xl font-bold">{importResult.failed}</p>
                      <p className="text-sm text-muted-foreground">失败</p>
                    </div>
                  </div>
                </div>
              </div>

              {/* 详细结果 */}
              <div className="space-y-2">
                <p className="text-sm font-medium">导入详情：</p>
                <div className="max-h-[200px] overflow-y-auto rounded-md border">
                  <div className="p-4 space-y-2">
                    {importResult.results.map((result, index) => (
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
                </div>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="flex-shrink-0 mt-4 pt-4 border-t">
          {!importResult ? (
            <>
              <Button variant="outline" onClick={handleClose} disabled={isImporting}>
                取消
              </Button>
              <Button
                onClick={handleImport}
                disabled={!isValid || isImporting}
              >
                {isImporting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    导入中...
                  </>
                ) : (
                  <>
                    <Upload className="mr-2 h-4 w-4" />
                    开始导入
                  </>
                )}
              </Button>
            </>
          ) : (
            <Button onClick={handleClose}>
              完成
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
