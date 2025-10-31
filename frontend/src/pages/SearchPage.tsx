import { useState, useEffect, useCallback } from 'react';
import { Search, X, Clock, Filter, ChevronDown, ChevronUp } from 'lucide-react';
import { Input } from '../components/ui/input';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Badge } from '../components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { EmailList } from '../components/email/EmailList';
import { useSearch, useSearchHistory } from '../hooks/useSearch';
import { useAccounts } from '../hooks/useAccounts';
import { useNavigate } from 'react-router-dom';
import { Email } from '../types';

export const SearchPage = () => {
  const navigate = useNavigate();
  const { accounts } = useAccounts();
  const { 
    emails, 
    total, 
    isLoading, 
    error, 
    hasSearched, 
    currentQuery, 
    search, 
    loadMore, 
    clearSearch 
  } = useSearch();
  const { history, addToHistory, removeFromHistory, clearHistory } = useSearchHistory();

  const [query, setQuery] = useState('');
  const [selectedAccountUid, setSelectedAccountUid] = useState<string>('all');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);

  // 高级搜索参数
  const [advancedParams, setAdvancedParams] = useState({
    fromAddress: '',
    subject: '',
    startDate: '',
    endDate: '',
    hasAttachment: 'all',
    isRead: 'all',
    isStarred: 'all',
  });

  // 处理搜索
  const handleSearch = useCallback(async (searchQuery?: string, page = 1) => {
    const finalQuery = searchQuery || query;
    if (!finalQuery.trim()) return;

    // 构建搜索查询
    let fullQuery = finalQuery;
    
    // 添加高级搜索参数
    if (advancedParams.fromAddress) {
      fullQuery += ` from:${advancedParams.fromAddress}`;
    }
    if (advancedParams.subject) {
      fullQuery += ` subject:${advancedParams.subject}`;
    }
    if (advancedParams.startDate) {
      fullQuery += ` after:${advancedParams.startDate}`;
    }
    if (advancedParams.endDate) {
      fullQuery += ` before:${advancedParams.endDate}`;
    }
    if (advancedParams.hasAttachment === 'true') {
      fullQuery += ` has:attachment`;
    }
    if (advancedParams.isRead === 'true') {
      fullQuery += ` is:read`;
    } else if (advancedParams.isRead === 'false') {
      fullQuery += ` is:unread`;
    }
    if (advancedParams.isStarred === 'true') {
      fullQuery += ` is:starred`;
    }

    await search({
      query: fullQuery,
      accountUid: selectedAccountUid === 'all' ? undefined : selectedAccountUid,
      pagination: { page, page_size: 20 },
    });

    if (page === 1) {
      addToHistory(finalQuery);
      setCurrentPage(1);
    } else {
      setCurrentPage(page);
    }
  }, [query, selectedAccountUid, advancedParams, search, addToHistory]);

  // 处理加载更多
  const handleLoadMore = useCallback(() => {
    const nextPage = currentPage + 1;
    loadMore(nextPage);
    setCurrentPage(nextPage);
  }, [currentPage, loadMore]);

  // 处理邮件点击
  const handleEmailClick = useCallback((email: Email) => {
    navigate(`/emails/${email.id}`, { 
      state: { 
        from: 'search',
        query: currentQuery,
        accountUid: selectedAccountUid 
      } 
    });
  }, [navigate, currentQuery, selectedAccountUid]);

  // 处理历史记录点击
  const handleHistoryClick = useCallback((historyQuery: string) => {
    setQuery(historyQuery);
    handleSearch(historyQuery);
  }, [handleSearch]);

  // 清除搜索
  const handleClearSearch = useCallback(() => {
    setQuery('');
    clearSearch();
    setCurrentPage(1);
  }, [clearSearch]);

  // 重置高级搜索
  const handleResetAdvanced = useCallback(() => {
    setAdvancedParams({
      fromAddress: '',
      subject: '',
      startDate: '',
      endDate: '',
      hasAttachment: 'all',
      isRead: 'all',
      isStarred: 'all',
    });
  }, []);

  // 键盘快捷键
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // 按 / 键聚焦搜索框
      if (e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        const searchInput = document.getElementById('search-input');
        if (searchInput) {
          searchInput.focus();
        }
      }
      // 按 Escape 键清除搜索
      if (e.key === 'Escape' && currentQuery) {
        handleClearSearch();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [currentQuery, handleClearSearch]);

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="max-w-4xl mx-auto">
        {/* 搜索头部 */}
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
            邮件搜索
          </h1>
          
          {/* 主搜索框 */}
          <div className="relative mb-4">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
            <Input
              id="search-input"
              type="text"
              placeholder="搜索邮件... (按 / 键快速聚焦)"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  handleSearch();
                }
              }}
              className="pl-10 pr-20"
            />
            <div className="absolute right-2 top-1/2 transform -translate-y-1/2 flex items-center gap-2">
              {query && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleClearSearch}
                  className="h-6 w-6 p-0"
                >
                  <X className="h-4 w-4" />
                </Button>
              )}
              <Button
                onClick={() => handleSearch()}
                disabled={!query.trim() || isLoading}
                size="sm"
              >
                {isLoading ? '搜索中...' : '搜索'}
              </Button>
            </div>
          </div>

          {/* 账户筛选 */}
          <div className="flex items-center gap-4 mb-4">
            <div className="flex-1">
              <Select value={selectedAccountUid} onValueChange={setSelectedAccountUid}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择邮箱账户（可选）" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有账户</SelectItem>
                  {accounts.map((account) => (
                    <SelectItem key={account.uid} value={account.uid}>
                      {account.email} ({account.provider})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button
              variant="outline"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center gap-2"
            >
              <Filter className="h-4 w-4" />
              高级搜索
              {showAdvanced ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            </Button>
          </div>

          {/* 高级搜索 */}
          {showAdvanced && (
            <Card className="mb-4">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg">高级搜索选项</CardTitle>
                  <Button variant="outline" size="sm" onClick={handleResetAdvanced}>
                    重置
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium mb-2 block">发件人</label>
                    <Input
                      placeholder="发件人邮箱地址"
                      value={advancedParams.fromAddress}
                      onChange={(e) => setAdvancedParams(prev => ({ ...prev, fromAddress: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-2 block">主题包含</label>
                    <Input
                      placeholder="邮件主题关键词"
                      value={advancedParams.subject}
                      onChange={(e) => setAdvancedParams(prev => ({ ...prev, subject: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-2 block">开始日期</label>
                    <Input
                      type="date"
                      value={advancedParams.startDate}
                      onChange={(e) => setAdvancedParams(prev => ({ ...prev, startDate: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-2 block">结束日期</label>
                    <Input
                      type="date"
                      value={advancedParams.endDate}
                      onChange={(e) => setAdvancedParams(prev => ({ ...prev, endDate: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-2 block">阅读状态</label>
                    <Select value={advancedParams.isRead} onValueChange={(value) => setAdvancedParams(prev => ({ ...prev, isRead: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择阅读状态" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">全部</SelectItem>
                        <SelectItem value="true">已读</SelectItem>
                        <SelectItem value="false">未读</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-2 block">星标状态</label>
                    <Select value={advancedParams.isStarred} onValueChange={(value) => setAdvancedParams(prev => ({ ...prev, isStarred: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择星标状态" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">全部</SelectItem>
                        <SelectItem value="true">已星标</SelectItem>
                        <SelectItem value="false">未星标</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        {/* 搜索历史 */}
        {!hasSearched && history.length > 0 && (
          <Card className="mb-6">
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg flex items-center gap-2">
                  <Clock className="h-4 w-4" />
                  搜索历史
                </CardTitle>
                <Button variant="outline" size="sm" onClick={clearHistory}>
                  清除历史
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {history.map((historyQuery, index) => (
                  <Badge
                    key={index}
                    variant="secondary"
                    className="cursor-pointer hover:bg-gray-200 dark:hover:bg-gray-700 flex items-center gap-1"
                    onClick={() => handleHistoryClick(historyQuery)}
                  >
                    {historyQuery}
                    <X
                      className="h-3 w-3 ml-1 hover:text-red-500"
                      onClick={(e) => {
                        e.stopPropagation();
                        removeFromHistory(historyQuery);
                      }}
                    />
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* 搜索结果 */}
        {hasSearched && (
          <div>
            {/* 结果统计 */}
            <div className="mb-4 flex items-center justify-between">
              <div className="text-sm text-gray-600 dark:text-gray-400">
                {isLoading && currentPage === 1 ? (
                  '搜索中...'
                ) : (
                  `找到 ${total} 个结果${currentQuery ? ` "${currentQuery}"` : ''}`
                )}
              </div>
              {hasSearched && (
                <Button variant="outline" size="sm" onClick={handleClearSearch}>
                  清除搜索
                </Button>
              )}
            </div>

            {/* 错误提示 */}
            {error && (
              <Card className="mb-4 border-red-200 dark:border-red-800">
                <CardContent className="pt-6">
                  <div className="text-red-600 dark:text-red-400">{error}</div>
                </CardContent>
              </Card>
            )}

            {/* 邮件列表 */}
            {emails.length > 0 ? (
              <div>
                <EmailList
                  emails={emails}
                  onEmailClick={handleEmailClick}
                  isLoading={isLoading && currentPage > 1}
                  highlightQuery={query}
                />
                
                {/* 加载更多 */}
                {emails.length < total && (
                  <div className="mt-6 text-center">
                    <Button
                      variant="outline"
                      onClick={handleLoadMore}
                      disabled={isLoading}
                    >
                      {isLoading ? '加载中...' : '加载更多'}
                    </Button>
                  </div>
                )}
              </div>
            ) : !isLoading && hasSearched ? (
              <Card>
                <CardContent className="pt-6">
                  <div className="text-center py-8">
                    <Search className="mx-auto h-12 w-12 text-gray-400 mb-4" />
                    <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                      未找到相关邮件
                    </h3>
                    <p className="text-gray-600 dark:text-gray-400">
                      尝试使用不同的关键词或调整搜索条件
                    </p>
                  </div>
                </CardContent>
              </Card>
            ) : null}
          </div>
        )}

        {/* 搜索提示 */}
        {!hasSearched && (
          <Card>
            <CardHeader>
              <CardTitle>搜索提示</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3 text-sm text-gray-600 dark:text-gray-400">
                <div>
                  <strong>基础搜索：</strong>直接输入关键词搜索邮件主题、发件人和正文内容
                </div>
                <div>
                  <strong>高级搜索：</strong>使用高级搜索选项进行精确筛选
                </div>
                <div>
                  <strong>快捷键：</strong>
                  <ul className="mt-2 space-y-1 ml-4">
                    <li>• 按 <kbd className="px-2 py-1 bg-gray-100 dark:bg-gray-800 rounded text-xs">/</kbd> 键快速聚焦搜索框</li>
                    <li>• 按 <kbd className="px-2 py-1 bg-gray-100 dark:bg-gray-800 rounded text-xs">Enter</kbd> 键执行搜索</li>
                    <li>• 按 <kbd className="px-2 py-1 bg-gray-100 dark:bg-gray-800 rounded text-xs">Esc</kbd> 键清除搜索</li>
                  </ul>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
};