import { useState, useEffect, useRef, useMemo } from 'react';
import { Upload, CheckCircle2, XCircle, Loader2, ChevronDown, ChevronUp, Eye } from 'lucide-react';
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
import { Input } from '../ui/input';
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
import { SortableFieldList } from './SortableFieldList';
import {
  type ImportFormatConfig,
  type FieldType,
  DEFAULT_FORMAT,
  DELIMITER_PRESETS,
  FIELD_LABELS,
  FORMAT_STORAGE_KEY,
} from './import-types';

// 批量导入配置
export interface BatchImportConfig {
  accounts: string[];
  format?: ImportFormatConfig;
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

  // 格式配置
  const [delimiter, setDelimiter] = useState(DEFAULT_FORMAT.delimiter);
  const [fields, setFields] = useState<FieldType[]>(DEFAULT_FORMAT.fields);
  const [showFormatConfig, setShowFormatConfig] = useState(false);
  
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

  // 加载格式配置 from localStorage
  useEffect(() => {
    try {
      const saved = localStorage.getItem(FORMAT_STORAGE_KEY);
      if (saved) {
        const config = JSON.parse(saved) as ImportFormatConfig;
        if (config.delimiter && Array.isArray(config.fields) && config.fields.length > 0) {
          setDelimiter(config.delimiter);
          setFields(config.fields);
        }
      }
    } catch {
      // ignore parse errors
    }
  }, []);

  // 保存格式配置 to localStorage
  useEffect(() => {
    localStorage.setItem(FORMAT_STORAGE_KEY, JSON.stringify({ delimiter, fields }));
  }, [delimiter, fields]);

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
      setShowFormatConfig(false);
      onClose();
    }
  };

  const parseAccounts = (text: string): string[] => {
    return text
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0 && line.includes(delimiter));
  };

  const accounts = parseAccounts(accountsText);
  const isValid = accounts.length > 0;

  // 实时预览：解析第一行展示字段映射
  const preview = useMemo(() => {
    const firstLine = accountsText.split('\n').map(l => l.trim()).find(l => l.length > 0);
    if (!firstLine || !firstLine.includes(delimiter)) return null;

    const parts = firstLine.split(delimiter);
    return fields.map((field, i) => ({
      field,
      label: FIELD_LABELS[field],
      value: parts[i] || '(缺失)',
    }));
  }, [accountsText, delimiter, fields]);

  const handleImport = async () => {
    const parsedAccounts = parseAccounts(accountsText);
    
    if (parsedAccounts.length === 0) {
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
        accounts: parsedAccounts,
        format: { delimiter, fields },
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
      const failedAccounts = parseAccounts(accountsText);
      setImportResult({
        success: 0,
        failed: failedAccounts.length,
        results: failedAccounts.map(acc => ({
          email: acc.split(delimiter)[0],
          status: 'failed',
          error: '导入失败',
        })),
      });
    } finally {
      setIsImporting(false);
    }
  };

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
              <div className="rounded-md border bg-muted/50 p-2 space-y-2">
                <div className="flex items-center justify-between">
                  <p className="text-xs text-muted-foreground">
                    <span className="font-medium">当前格式：</span>
                    <code className="ml-1 rounded bg-background px-1">
                      {fields.join(delimiter)}
                    </code>
                  </p>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={() => setShowFormatConfig(!showFormatConfig)}
                  >
                    {showFormatConfig ? '收起' : '自定义格式'}
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">每行一个账号</p>
              </div>

              {showFormatConfig && (
                <div className="space-y-3 rounded-md border p-3">
                  <div className="space-y-1.5">
                    <Label className="text-xs">分隔符</Label>
                    <div className="flex gap-2">
                      <Input
                        value={delimiter === '\t' ? '\\t' : delimiter}
                        onChange={(e) => {
                          const v = e.target.value;
                          setDelimiter(v === '\\t' ? '\t' : v);
                        }}
                        className="h-8 text-sm font-mono"
                        placeholder="输入分隔符"
                      />
                      <Select
                        value={delimiter}
                        onValueChange={setDelimiter}
                      >
                        <SelectTrigger className="h-8 w-[180px] text-xs">
                          <SelectValue placeholder="预设" />
                        </SelectTrigger>
                        <SelectContent>
                          {DELIMITER_PRESETS.map((preset) => (
                            <SelectItem key={preset.value} value={preset.value}>
                              {preset.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <Label className="text-xs">字段顺序（拖拽排序）</Label>
                    <SortableFieldList fields={fields} onChange={setFields} />
                  </div>

                  {preview && (
                    <div className="space-y-1">
                      <div className="flex items-center gap-1 text-xs text-muted-foreground">
                        <Eye className="h-3 w-3" />
                        <span>预览（第一行解析结果）</span>
                      </div>
                      <div className="rounded-md border bg-muted/30 p-2 space-y-1">
                        {preview.map((item) => (
                          <div key={item.field} className="flex items-center gap-2 text-xs">
                            <span className="font-medium text-muted-foreground w-20 shrink-0">
                              {item.label}
                            </span>
                            <code className="truncate text-foreground">
                              {item.value}
                            </code>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

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
