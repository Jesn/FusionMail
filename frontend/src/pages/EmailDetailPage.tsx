import { useEffect, useState, useMemo, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { EmailDetail } from '../components/email/EmailDetail';
import { ComposeEmail, ComposeMode } from '../components/email/ComposeEmail';
import { useEmails } from '../hooks/useEmails';
import { useAccounts } from '../hooks/useAccounts';
import { useEmailStore } from '../stores/emailStore';
import { Button } from '../components/ui/button';
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
import { SkeletonEmailDetail } from '../components/ui/skeleton';

export const EmailDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const location = useLocation();
  
  // 解析 URL 参数
  const searchParams = useMemo(() => new URLSearchParams(location.search), [location.search]);
  const fromPage = searchParams.get('from'); // 来源页面：search, spam, trash 等
  
  const includeDeleted = useMemo(() => {
    return searchParams.get('include_deleted') === 'true' || fromPage === 'trash';
  }, [searchParams, fromPage]);

  // 邮件编写对话框状态
  const [composeOpen, setComposeOpen] = useState(false);
  const [composeMode, setComposeMode] = useState<ComposeMode>('new');

  const {
    selectedEmail,
    isLoadingDetail,
    loadEmailDetail,
    toggleStar,
    archiveEmail,
    deleteEmail,
    markAsRead,
    restoreEmail,
    permanentDeleteEmail,
  } = useEmails();
  const { accounts } = useAccounts();

  useEffect(() => {
    if (id) {
      loadEmailDetail(parseInt(id, 10), includeDeleted);
    }
  }, [id, includeDeleted, loadEmailDetail]);

  // 进入详情后，若邮件未读则自动标记为已读并刷新全局未读数
  useEffect(() => {
    if (selectedEmail && !selectedEmail.is_read) {
      markAsRead([selectedEmail.id]);
    }
  }, [selectedEmail, markAsRead]);

  const handleToggleStar = () => {
    if (selectedEmail) {
      toggleStar(selectedEmail.id, selectedEmail.is_starred);
    }
  };

  const handleArchive = () => {
    if (selectedEmail) {
      archiveEmail(selectedEmail.id);
      // 归档后返回来源页面
      handleBack();
    }
  };

  const handleDeleteClick = () => {
    setShowDeleteDialog(true);
  };

  const handleDeleteConfirm = () => {
    if (selectedEmail) {
      deleteEmail(selectedEmail.id);
      setShowDeleteDialog(false);
      // 删除后返回来源页面
      handleBack();
    }
  };
  const handleRestore = () => {
    if (selectedEmail) {
      restoreEmail(selectedEmail.id);
      // 恢复后返回来源页面（通常是垃圾箱）
      handleBack();
    }
  };

  // 永久删除（回收站）
  const [showPermanentDeleteDialog, setShowPermanentDeleteDialog] = useState(false);

  const handlePermanentDeleteClick = () => {
    setShowPermanentDeleteDialog(true);
  };

  const handlePermanentDeleteConfirm = async () => {
    if (selectedEmail) {
      await permanentDeleteEmail(selectedEmail.id);
      setShowPermanentDeleteDialog(false);
      // 永久删除后返回来源页面
      handleBack();
    }
  };


  const handleBack = () => {
    // 根据来源页面返回到正确的位置
    switch (fromPage) {
      case 'search':
        navigate('/search');
        break;
      case 'spam':
        navigate('/spam');
        break;
      case 'trash':
        navigate('/trash');
        break;
      case 'sent':
        navigate('/sent');
        break;
      default:
        navigate('/inbox');
    }
  };

  // 回复邮件
  const handleReply = useCallback(() => {
    setComposeMode('reply');
    setComposeOpen(true);
  }, []);

  // 回复全部
  const handleReplyAll = useCallback(() => {
    setComposeMode('replyAll');
    setComposeOpen(true);
  }, []);

  // 转发邮件
  const handleForward = useCallback(() => {
    setComposeMode('forward');
    setComposeOpen(true);
  }, []);

  // 处理垃圾邮件状态变化
  const handleSpamStatusChange = useCallback((isSpam: boolean) => {
    // 更新 store 中的 selectedEmail 状态
    const { updateEmailStatus, setSpamCount, spamCount } = useEmailStore.getState();
    if (selectedEmail) {
      updateEmailStatus(selectedEmail.id, { 
        is_spam: isSpam,
        user_marked_spam: true,
        user_marked_at: new Date().toISOString(),
      });
      // 更新垃圾邮件计数
      if (isSpam) {
        setSpamCount(spamCount + 1);
      } else {
        setSpamCount(Math.max(0, spamCount - 1));
      }
    }
  }, [selectedEmail]);

  // 获取当前邮件所属账号的删除策略
  const currentAccount = useMemo(() => {
    if (!selectedEmail || !accounts.length) return null;
    return accounts.find(acc => acc.uid === selectedEmail.account_uid);
  }, [selectedEmail, accounts]);

  // 生成删除提示文本
  const deleteMessage = useMemo(() => {
    if (!currentAccount) {
      return '确定要删除这封邮件吗？此操作仅在本地生效，不会影响源邮箱。';
    }

    if (currentAccount.server_delete_policy === 'soft') {
      return '确定要删除这封邮件吗？删除后邮件将从本地和服务器垃圾箱中移除。';
    }

    return '确定要删除这封邮件吗？此操作仅在本地生效，不会影响源邮箱。';
  }, [currentAccount]);

  if (isLoadingDetail) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex-1 overflow-hidden px-6 py-4">
          <SkeletonEmailDetail />
        </div>
      </div>
    );
  }

  if (!selectedEmail) {
    return (
      <div className="flex h-full flex-col items-center justify-center">
        <p className="mb-4 text-lg text-muted-foreground">邮件不存在</p>
        <Button onClick={handleBack}>返回收件箱</Button>
      </div>
    );
  }

  return (
    <>
      <div className="flex h-full flex-col">
        {/* 邮件详情 */}
        <div className="flex-1 overflow-hidden">
          <EmailDetail
            email={selectedEmail}
            onToggleStar={handleToggleStar}
            onArchive={handleArchive}
            onDelete={handleDeleteClick}
            onRestore={handleRestore}
            onPermanentDelete={handlePermanentDeleteClick}
            onBack={handleBack}
            onSpamStatusChange={handleSpamStatusChange}
            forceDeletedView={includeDeleted}
            onReply={handleReply}
            onReplyAll={handleReplyAll}
            onForward={handleForward}
          />
        </div>
      </div>

      {/* 删除确认对话框 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteMessage}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDeleteConfirm}>
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 永久删除确认对话框 */}
      <AlertDialog open={showPermanentDeleteDialog} onOpenChange={setShowPermanentDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认永久删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要永久删除这封邮件吗？
              <br />
              <br />
              <span className="text-destructive font-medium">
                此操作无法撤销！
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handlePermanentDeleteConfirm}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              永久删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 邮件编写对话框 */}
      {selectedEmail && (
        <ComposeEmail
          open={composeOpen}
          onOpenChange={setComposeOpen}
          mode={composeMode}
          originalEmail={selectedEmail}
          defaultAccountUid={selectedEmail.account_uid}
        />
      )}
    </>
  );
};
