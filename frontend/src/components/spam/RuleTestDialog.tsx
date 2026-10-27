import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Badge } from '../ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import { CATEGORY_OPTIONS } from './ruleMeta';
import type { SpamRuleRequest, RuleTestResponse } from '../../services/spamService';

interface RuleTestDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  formData: SpamRuleRequest;
  onFormChange: (patch: Partial<SpamRuleRequest>) => void;
  testContent: string;
  onTestContentChange: (content: string) => void;
  testResult: RuleTestResponse | null;
  isTesting: boolean;
  onTest: () => void;
}

export const RuleTestDialog = ({
  open,
  onOpenChange,
  formData,
  onFormChange,
  testContent,
  onTestContentChange,
  testResult,
  isTesting,
  onTest,
}: RuleTestDialogProps) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>测试规则</DialogTitle>
          <DialogDescription>
            输入测试内容，验证规则是否能正确匹配
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>规则类别</Label>
            <Select
              value={formData.category}
              onValueChange={(value) => onFormChange({ category: value })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CATEGORY_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>匹配模式</Label>
            <Input
              value={formData.pattern}
              onChange={(e) => onFormChange({ pattern: e.target.value })}
              placeholder="输入匹配模式"
            />
          </div>
          <div className="space-y-2">
            <Label>测试内容</Label>
            <Textarea
              value={testContent}
              onChange={(e) => onTestContentChange(e.target.value)}
              placeholder="输入要测试的邮件内容"
              rows={4}
            />
          </div>
          {testResult && (
            <div className={`p-3 rounded-lg ${testResult.matched ? 'bg-red-50 border border-red-200' : 'bg-green-50 border border-green-200'}`}>
              <div className="flex items-center gap-2 mb-2">
                {testResult.matched ? (
                  <Badge variant="destructive">匹配成功</Badge>
                ) : (
                  <Badge variant="outline" className="text-green-600 border-green-600">未匹配</Badge>
                )}
                <span className="text-xs text-muted-foreground">
                  耗时: {testResult.duration}
                </span>
              </div>
              {testResult.matched && testResult.matches.length > 0 && (
                <div className="text-sm">
                  <span className="text-muted-foreground">匹配内容: </span>
                  {testResult.matches.map((match, i) => (
                    <code key={i} className="bg-red-100 px-1 rounded mx-1">
                      {match}
                    </code>
                  ))}
                </div>
              )}
              {testResult.error && (
                <div className="text-sm text-destructive mt-1">
                  错误: {testResult.error}
                </div>
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
          <Button
            onClick={onTest}
            disabled={!formData.pattern || !testContent || isTesting}
          >
            {isTesting ? '测试中...' : '测试'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
