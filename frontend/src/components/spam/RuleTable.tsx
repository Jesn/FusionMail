import { Shield, Edit, Trash2, TestTube } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Switch } from '../ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../ui/table';
import { getCategoryColor, getCategoryLabel } from './ruleMeta';
import type { SpamRule } from '../../services/spamService';

interface RuleTableProps {
  rules: SpamRule[];
  isLoading: boolean;
  onToggle: (rule: SpamRule) => void;
  onTest: (rule: SpamRule) => void;
  onEdit: (rule: SpamRule) => void;
  onDelete: (rule: SpamRule) => void;
  page: number;
  totalPages: number;
  total: number;
  onPageChange: (page: number) => void;
}

export const RuleTable = ({
  rules,
  isLoading,
  onToggle,
  onTest,
  onEdit,
  onDelete,
  page,
  totalPages,
  total,
  onPageChange,
}: RuleTableProps) => {
  return (
    <>
      <div className="flex-1 overflow-auto p-4">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-12">状态</TableHead>
              <TableHead>名称</TableHead>
              <TableHead className="w-24">类别</TableHead>
              <TableHead>匹配模式</TableHead>
              <TableHead className="w-16 text-center">评分</TableHead>
              <TableHead className="w-20 text-center">命中</TableHead>
              <TableHead className="w-16">类型</TableHead>
              <TableHead className="w-24 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rules.map((rule) => (
              <TableRow key={rule.id}>
                <TableCell>
                  <Switch
                    checked={rule.enabled}
                    onCheckedChange={() => onToggle(rule)}
                  />
                </TableCell>
                <TableCell>
                  <div>
                    <div className="font-medium">{rule.name}</div>
                    {rule.description && (
                      <div className="text-xs text-muted-foreground truncate max-w-xs">
                        {rule.description}
                      </div>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge className={getCategoryColor(rule.category)}>
                    {getCategoryLabel(rule.category)}
                  </Badge>
                </TableCell>
                <TableCell>
                  <code className="text-xs bg-muted px-1.5 py-0.5 rounded truncate max-w-xs block">
                    {rule.pattern}
                  </code>
                </TableCell>
                <TableCell className="text-center">
                  <Badge variant="outline">{rule.score}</Badge>
                </TableCell>
                <TableCell className="text-center text-muted-foreground">
                  {rule.hit_count.toLocaleString()}
                </TableCell>
                <TableCell>
                  {rule.is_builtin ? (
                    <Badge variant="secondary">内置</Badge>
                  ) : (
                    <Badge variant="outline">自定义</Badge>
                  )}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0"
                      onClick={() => onTest(rule)}
                      title="测试规则"
                    >
                      <TestTube className="h-3.5 w-3.5" />
                    </Button>
                    {!rule.is_builtin && (
                      <>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0"
                          onClick={() => onEdit(rule)}
                          title="编辑规则"
                        >
                          <Edit className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0 text-destructive hover:text-destructive"
                          onClick={() => onDelete(rule)}
                          title="删除规则"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>

        {rules.length === 0 && !isLoading && (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <Shield className="h-12 w-12 mb-4 text-muted-foreground/50" />
            <p className="text-lg">暂无规则</p>
            <p className="text-sm mt-1">点击"添加规则"创建新的垃圾邮件规则</p>
          </div>
        )}
      </div>

      {/* 分页 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between border-t bg-background px-4 py-2">
          <div className="text-sm text-muted-foreground">
            第 {page} 页，共 {totalPages} 页 · 总计 {total} 条规则
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onPageChange(page - 1)}
              disabled={page === 1}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onPageChange(page + 1)}
              disabled={page === totalPages}
            >
              下一页
            </Button>
          </div>
        </div>
      )}
    </>
  );
};
