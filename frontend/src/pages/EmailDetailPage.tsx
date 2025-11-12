import { useEffect, useState, useMemo } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { EmailDetail } from '../components/email/EmailDetail';
import { useEmails } from '../hooks/useEmails';
import { useAccounts } from '../hooks/useAccounts';
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
import { Loader2 } from 'lucide-react';

export const EmailDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const location = useLocation();
  const includeDeleted = useMemo(() => {
    const sp = new URLSearchParams(location.search);
    return sp.get('include_deleted') === 'true' || sp.get('from') === 'trash';
  }, [location.search]);

  const {
    selectedEmail,
    isLoadingDetail,
    loadEmailDetail,
    toggleStar,
    archiveEmail,
    deleteEmail,
    markAsRead,
    restoreEmail,
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
      // 如果当前邮件在垃圾箱中，归档后应该跳转到归档页面
      if (selectedEmail.is_deleted) {
        navigate('/inbox'); // 先跳转到收件箱，然后用户可以去归档查看
      } else {
        navigate('/inbox');
      }
    }
  };

  const handleDeleteClick = () => {
    setShowDeleteDialog(true);
  };

  const handleDeleteConfirm = () => {
    if (selectedEmail) {
      deleteEmail(selectedEmail.id);
      setShowDeleteDialog(false);
      navigate('/inbox');
    }
  };
  const handleRestore = () => {
    if (selectedEmail) {
      restoreEmail(selectedEmail.id);
      navigate('/inbox');
    }
  };


  const handleBack = () => {
    navigate('/inbox');
  };

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
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
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
            onBack={handleBack}
            forceDeletedView={includeDeleted}
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
    </>
  );
};
