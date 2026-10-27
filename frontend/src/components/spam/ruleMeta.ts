// 规则类别元数据（自 SpamRulesPage 原位搬移，P12 大文件治理）
// 注意：选项与颜色必须与后端 spam 规则类别保持一致，勿凭记忆增删。

export const CATEGORY_OPTIONS = [
  { value: 'keyword', label: '关键词' },
  { value: 'pattern', label: '正则表达式' },
  { value: 'header', label: '邮件头' },
  { value: 'content', label: '邮件内容' },
  { value: 'url', label: 'URL' },
  { value: 'attachment', label: '附件' },
];

// 获取类别标签
export const getCategoryLabel = (category: string) => {
  const option = CATEGORY_OPTIONS.find((opt) => opt.value === category);
  return option?.label || category;
};

// 获取类别颜色
export const getCategoryColor = (category: string) => {
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
