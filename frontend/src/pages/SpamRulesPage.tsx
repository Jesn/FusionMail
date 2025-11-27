import { useState, useEffect, useCallback } from 'react';
import {
  Shield,
  Plus,
  RefreshCw,
  Edit,
  Trash2,
  TestTube,
  Filter,
} from 'lucide-react';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Badge } from '../components/ui/badge';
import { Switch } from '../components/ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../components/ui/table';
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
import { Label } from '../components/ui/label';
import { Textarea } from '../components/ui/textarea';
import { toast } from 'sonner';
import {
  ruleService,
  SpamRule,
  SpamRuleRequest,
  RuleStats,
  RuleTestResponse,
} from '../services/spamService';

// 规则类别选项
const CATEGORY_OPTIONS = [
  { value: 'keyword', label: '关键词' },
  { value: 'pattern', label: '正则表达式' },
  { value: 'header', label: '邮件头' },
  { value: 'content', label: '邮件内容' },
  { value: 'url', label: 'URL' },
  { value: 'attachment', label: '附件' },
];


// 获取类别标签
const getCategoryLabel = (category: string) => {
  const option = CATEGORY_OPTIONS.find((opt) => opt.value === category);
  return option?.label || category;
};

// 获取类别颜色
const getCategoryColor = (category: string) => {
  const colors: Record<string, string> = {
    keyword: 'bg-blue-100 text-blue-800',
    pattern: 'bg-purple-100 text-purple-800',
    header: 'bg-green-100 text-green-800',
    content: 'bg-yellow-100 text-yellow-800',
    url: 'bg-red-100 text-red-800',
    attachment: 'bg-orange-100 text-orange-800',
  };
  return colors[category] || 'bg-gray-100 text-gray-800';
};

export const SpamRulesPage = () => {
  // 状态
  const [rules, setRules] = useState<SpamRule[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [isLoading, setIsLoading] = useState(false);
  const [stats, setStats] = useState<RuleStats | null>(null);
  const [categoryFilter, setCategoryFilter] = useState<string>('');

  // 对话框状态
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showTestDialog, setShowTestDialog] = useState(false);
  const [selectedRule, setSelectedRule] = useState<SpamRule | null>(null);

  // 表单状态
  const [formData, setFormData] = useState<SpamRuleRequest>({
    name: '',
    description: '',
    category: 'keyword',
    pattern: '',
    score: 10,
    enabled: true,
  });

  // 测试状态
  const [testContent, setTestContent] = useState('');
  const [testResult, setTestResult] = useState<RuleTestResponse | null>(null);
  const [isTesting, setIsTesting] = useState(false);

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

      {/* 规则列表 */}
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
                    onCheckedChange={() => handleToggle(rule)}
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
                      onClick={() => handleOpenTest(rule)}
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
                          onClick={() => handleOpenEdit(rule)}
                          title="编辑规则"
                        >
                          <Edit className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0 text-destructive hover:text-destructive"
                          onClick={() => handleOpenDelete(rule)}
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
              onClick={() => setPage(page - 1)}
              disabled={page === 1}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(page + 1)}
              disabled={page === totalPages}
            >
              下一页
            </Button>
          </div>
        </div>
      )}


      {/* 创建规则对话框 */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>添加规则</DialogTitle>
            <DialogDescription>
              创建新的垃圾邮件检测规则
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">规则名称</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="输入规则名称"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="category">规则类别</Label>
              <Select
                value={formData.category}
                onValueChange={(value) => setFormData({ ...formData, category: value })}
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
              <Label htmlFor="pattern">匹配模式</Label>
              <Input
                id="pattern"
                value={formData.pattern}
                onChange={(e) => setFormData({ ...formData, pattern: e.target.value })}
                placeholder={formData.category === 'pattern' ? '输入正则表达式' : '输入关键词'}
              />
              <p className="text-xs text-muted-foreground">
                {formData.category === 'pattern'
                  ? '使用正则表达式进行匹配'
                  : '使用关键词进行匹配（不区分大小写）'}
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="score">评分权重</Label>
              <Input
                id="score"
                type="number"
                min={1}
                max={100}
                value={formData.score}
                onChange={(e) => setFormData({ ...formData, score: parseInt(e.target.value) || 10 })}
              />
              <p className="text-xs text-muted-foreground">
                匹配时增加的垃圾邮件评分（1-100）
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">描述（可选）</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="输入规则描述"
                rows={2}
              />
            </div>
          </div>
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
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">规则名称</Label>
              <Input
                id="edit-name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-category">规则类别</Label>
              <Select
                value={formData.category}
                onValueChange={(value) => setFormData({ ...formData, category: value })}
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
              <Label htmlFor="edit-pattern">匹配模式</Label>
              <Input
                id="edit-pattern"
                value={formData.pattern}
                onChange={(e) => setFormData({ ...formData, pattern: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-score">评分权重</Label>
              <Input
                id="edit-score"
                type="number"
                min={1}
                max={100}
                value={formData.score}
                onChange={(e) => setFormData({ ...formData, score: parseInt(e.target.value) || 10 })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-description">描述</Label>
              <Textarea
                id="edit-description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={2}
              />
            </div>
          </div>
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
      <Dialog open={showTestDialog} onOpenChange={setShowTestDialog}>
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
                onValueChange={(value) => setFormData({ ...formData, category: value })}
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
                onChange={(e) => setFormData({ ...formData, pattern: e.target.value })}
                placeholder="输入匹配模式"
              />
            </div>
            <div className="space-y-2">
              <Label>测试内容</Label>
              <Textarea
                value={testContent}
                onChange={(e) => setTestContent(e.target.value)}
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
            <Button variant="outline" onClick={() => setShowTestDialog(false)}>
              关闭
            </Button>
            <Button
              onClick={handleTest}
              disabled={!formData.pattern || !testContent || isTesting}
            >
              {isTesting ? '测试中...' : '测试'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default SpamRulesPage;
