import { useState, useEffect, useCallback } from 'react';
import {
  Shield,
  Plus,
  RefreshCw,
  TestTube,
  Filter,
} from 'lucide-react';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';
import { toast } from 'sonner';
import {
  ruleService,
  SpamRule,
  SpamRuleRequest,
  RuleStats,
  RuleTestResponse,
} from '../services/spamService';
import { CATEGORY_OPTIONS } from '../components/spam/ruleMeta';
import { RuleFormFields } from '../components/spam/RuleFormFields';
import { RuleTable } from '../components/spam/RuleTable';
import { RuleTestDialog } from '../components/spam/RuleTestDialog';

export const SpamRulesPage = () => {
  // 状态
  const [rules, setRules] = useState<SpamRule[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [isLoading, setIsLoading] = useState(false);
  const [stats, setStats] = useState<RuleStats | null>(null);
  const [categoryFilter, setCategoryFilter] = useState<string>('');
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showTestDialog, setShowTestDialog] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [selectedRule, setSelectedRule] = useState<SpamRule | null>(null);
  const [testContent, setTestContent] = useState('');
  const [formData, setFormData] = useState<SpamRuleRequest>({
    name: '',
    description: '',
    category: 'keyword',
    pattern: '',
    score: 10,
    enabled: true,
  });
  const [testResult, setTestResult] = useState<RuleTestResponse | null>(null);

  // 加载规则列表
  const loadRules = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await ruleService.getRules({
        category: categoryFilter || undefined,
        page,
        page_size: pageSize,
      });
      setRules(response.data || []);
      setTotal(response.total || 0);
    } catch (error) {
      console.error('Failed to load rules:', error);
      toast.error('加载规则列表失败');
    } finally {
      setIsLoading(false);
    }
  }, [page, pageSize, categoryFilter]);

  // 加载统计信息
  const loadStats = useCallback(async () => {
    try {
      const data = await ruleService.getRuleStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load rule stats:', error);
    }
  }, []);

  // 初始加载
  useEffect(() => {
    loadRules();
    loadStats();
  }, [loadRules, loadStats]);

  // 刷新
  const handleRefresh = () => {
    loadRules();
    loadStats();
  };

  // 打开创建对话框
  const handleOpenCreate = () => {
    setFormData({
      name: '',
      description: '',
      category: 'keyword',
      pattern: '',
      score: 10,
      enabled: true,
    });
    setShowCreateDialog(true);
  };

  // 打开编辑对话框
  const handleOpenEdit = (rule: SpamRule) => {
    setSelectedRule(rule);
    setFormData({
      name: rule.name,
      description: rule.description,
      category: rule.category,
      pattern: rule.pattern,
      score: rule.score,
      enabled: rule.enabled,
    });
    setShowEditDialog(true);
  };

  // 打开删除对话框
  const handleOpenDelete = (rule: SpamRule) => {
    setSelectedRule(rule);
    setShowDeleteDialog(true);
  };

  // 打开测试对话框
  const handleOpenTest = (rule?: SpamRule) => {
    if (rule) {
      setFormData({
        ...formData,
        category: rule.category,
        pattern: rule.pattern,
      });
    }
    setTestContent('');
    setTestResult(null);
    setShowTestDialog(true);
  };

  // 创建规则
  const handleCreate = async () => {
    try {
      await ruleService.createRule(formData);
      toast.success('规则创建成功');
      setShowCreateDialog(false);
      handleRefresh();
    } catch (error) {
      console.error('Failed to create rule:', error);
      toast.error('创建规则失败');
    }
  };

  // 更新规则
  const handleUpdate = async () => {
    if (!selectedRule) return;
    try {
      await ruleService.updateRule(selectedRule.id, formData);
      toast.success('规则更新成功');
      setShowEditDialog(false);
      handleRefresh();
    } catch (error) {
      console.error('Failed to update rule:', error);
      toast.error('更新规则失败');
    }
  };

  // 删除规则
  const handleDelete = async () => {
    if (!selectedRule) return;
    try {
      await ruleService.deleteRule(selectedRule.id);
      toast.success('规则删除成功');
      setShowDeleteDialog(false);
      handleRefresh();
    } catch (error) {
      console.error('Failed to delete rule:', error);
      toast.error('删除规则失败');
    }
  };

  // 切换规则状态
  const handleToggle = async (rule: SpamRule) => {
    try {
      await ruleService.toggleRule(rule.id);
      toast.success(`规则已${rule.enabled ? '禁用' : '启用'}`);
      handleRefresh();
    } catch (error) {
      console.error('Failed to toggle rule:', error);
      toast.error('切换规则状态失败');
    }
  };

  // 测试规则
  const handleTest = async () => {
    setIsTesting(true);
    try {
      const result = await ruleService.testRule({
        pattern: formData.pattern,
        category: formData.category,
        content: testContent,
      });
      setTestResult(result);
    } catch (error) {
      console.error('Failed to test rule:', error);
      toast.error('测试规则失败');
    } finally {
      setIsTesting(false);
    }
  };

  // 计算总页数
  const totalPages = Math.ceil(total / pageSize);

  // 表单字段局部更新（传给 RuleFormFields / RuleTestDialog）
  const updateFormData = (patch: Partial<SpamRuleRequest>) => {
    setFormData((prev) => ({ ...prev, ...patch }));
  };

  return (
    <div className="flex h-full flex-col">
      {/* 工具栏 */}
      <div className="flex items-center justify-between border-b bg-background px-4 py-2">
        <div className="flex items-center gap-3">
          <Shield className="h-5 w-5 text-primary" />
          <span className="font-medium">垃圾邮件规则</span>
          {stats && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Badge variant="outline">{stats.total_count} 条规则</Badge>
              <Badge variant="outline" className="text-green-600">
                {stats.enabled_count} 启用
              </Badge>
              <Badge variant="outline" className="text-gray-500">
                {stats.builtin_count} 内置
              </Badge>
            </div>
          )}
        </div>

        <div className="flex items-center gap-2">
          {/* 类别筛选 */}
          <Select value={categoryFilter} onValueChange={setCategoryFilter}>
            <SelectTrigger className="w-32 h-8">
              <Filter className="h-3.5 w-3.5 mr-1" />
              <SelectValue placeholder="全部类别" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">全部类别</SelectItem>
              {CATEGORY_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={isLoading}
          >
            <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={() => handleOpenTest()}
          >
            <TestTube className="h-4 w-4 mr-1" />
            测试规则
          </Button>

          <Button size="sm" onClick={handleOpenCreate}>
            <Plus className="h-4 w-4 mr-1" />
            添加规则
          </Button>
        </div>
      </div>

      {/* 规则列表 + 分页 */}
      <RuleTable
        rules={rules}
        isLoading={isLoading}
        onToggle={handleToggle}
        onTest={handleOpenTest}
        onEdit={handleOpenEdit}
        onDelete={handleOpenDelete}
        page={page}
        totalPages={totalPages}
        total={total}
        onPageChange={setPage}
      />

      {/* 创建规则对话框 */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>添加规则</DialogTitle>
            <DialogDescription>
              创建新的垃圾邮件检测规则
            </DialogDescription>
          </DialogHeader>
          <RuleFormFields
            formData={formData}
            onChange={updateFormData}
            mode="create"
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              取消
            </Button>
            <Button onClick={handleCreate} disabled={!formData.name || !formData.pattern}>
              创建
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 编辑规则对话框 */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>编辑规则</DialogTitle>
            <DialogDescription>
              修改垃圾邮件检测规则
            </DialogDescription>
          </DialogHeader>
          <RuleFormFields
            formData={formData}
            onChange={updateFormData}
            mode="edit"
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditDialog(false)}>
              取消
            </Button>
            <Button onClick={handleUpdate} disabled={!formData.name || !formData.pattern}>
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认对话框 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除规则 "{selectedRule?.name}" 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 测试规则对话框 */}
      <RuleTestDialog
        open={showTestDialog}
        onOpenChange={setShowTestDialog}
        formData={formData}
        onFormChange={updateFormData}
        testContent={testContent}
        onTestContentChange={setTestContent}
        testResult={testResult}
        isTesting={isTesting}
        onTest={handleTest}
      />
    </div>
  );
};

export default SpamRulesPage;
