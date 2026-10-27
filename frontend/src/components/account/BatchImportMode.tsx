import { AlertCircle } from 'lucide-react';
import { Alert, AlertDescription } from '../ui/alert';
import { Label } from '../ui/label';
import { Progress } from '../ui/progress';
import { ScrollArea } from '../ui/scroll-area';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Textarea } from '../ui/textarea';

export interface BatchImportResultShape {
  success: number;
  failed: number;
  results: Array<{ email: string; status: 'success' | 'failed'; error?: string }>;
}

interface BatchImportModeProps {
  batchSeparator: string;
  onSeparatorChange: (sep: string) => void;
  batchAccountsText: string;
  onAccountsTextChange: (text: string) => void;
  isBatchImporting: boolean;
  batchImportProgress: number;
  batchImportResult: BatchImportResultShape | null;
}

/** 批量导入模式表单（短效邮箱多账户录入）。纯展示：状态由 AccountForm 下传。 */
export function BatchImportMode({
  batchSeparator,
  onSeparatorChange,
  batchAccountsText,
  onAccountsTextChange,
  isBatchImporting,
  batchImportProgress,
  batchImportResult,
}: BatchImportModeProps) {
  return (
    <>
      <Alert>
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>
          批量导入适用于短效邮箱，每行一个账户，格式：邮箱{batchSeparator}密码
        </AlertDescription>
      </Alert>
      <div className="space-y-2">
        <Label>分隔符</Label>
        <Select value={batchSeparator} onValueChange={onSeparatorChange}>
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
          onChange={(e) => onAccountsTextChange(e.target.value)}
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
  );
}
