import { useState } from 'react';
import { Upload, AlertCircle, CheckCircle2, XCircle, Loader2 } from 'lucide-react';
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
import { Alert, AlertDescription } from '../ui/alert';
import { Progress } from '../ui/progress';
import { ScrollArea } from '../ui/scroll-area';

interface BatchImportDialogProps {
  open: boolean;
  onClose: () => void;
  onImport: (accounts: string[]) => Promise<BatchImportResult>;
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

  const handleClose = () => {
    if (!isImporting) {
      setAccountsText('');
      setImportResult(null);
      setProgress(0);
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

      const result = await onImport(accounts);
      
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
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>批量导入短效邮箱</DialogTitle>
          <DialogDescription>
            粘贴短效邮箱账号字符串，每行一个账号
          </DialogDescription>
        </DialogHeader>

        {!importResult ? (
          <>
            <div className="space-y-4">
              {/* 格式说明 */}
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <div className="space-y-2">
                    <p className="font-medium">账号格式说明：</p>
                    <code className="block rounded bg-muted p-2 text-xs">
                      email----password----refresh_token----client_id
                    </code>
                    <p className="text-xs text-muted-foreground">
                      每行一个账号，使用 ---- 分隔各个字段
                    </p>
                  </div>
                </AlertDescription>
              </Alert>

              {/* 输入框 */}
              <div className="space-y-2">
                <Textarea
                  placeholder="粘贴账号字符串，每行一个..."
                  value={accountsText}
                  onChange={(e) => setAccountsText(e.target.value)}
                  className="min-h-[200px] font-mono text-sm"
                  disabled={isImporting}
                />
                {accounts.length > 0 && (
                  <p className="text-sm text-muted-foreground">
                    已识别 {accounts.length} 个账号
                  </p>
                )}
              </div>

              {/* 进度条 */}
              {isImporting && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span>正在导入...</span>
                    <span>{progress}%</span>
                  </div>
                  <Progress value={progress} />
                </div>
              )}
            </div>

            <DialogFooter>
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
            </DialogFooter>
          </>
        ) : (
          <>
            {/* 导入结果 */}
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
                <ScrollArea className="h-[200px] rounded-md border">
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
                </ScrollArea>
              </div>
            </div>

            <DialogFooter>
              <Button onClick={handleClose}>
                完成
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
};
