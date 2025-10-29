import { useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { EmailDetail } from '../components/email/EmailDetail';
import { useEmails } from '../hooks/useEmails';
import { Button } from '../components/ui/button';
import { ArrowLeft, Loader2 } from 'lucide-react';

export const EmailDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const {
    selectedEmail,
    isLoadingDetail,
    loadEmailDetail,
    toggleStar,
    archiveEmail,
    deleteEmail,
  } = useEmails();

  useEffect(() => {
    if (id) {
      loadEmailDetail(parseInt(id, 10));
    }
  }, [id, loadEmailDetail]);

  const handleToggleStar = () => {
    if (selectedEmail) {
      toggleStar(selectedEmail.id, selectedEmail.is_starred);
    }
  };

  const handleArchive = () => {
    if (selectedEmail) {
      archiveEmail(selectedEmail.id);
      navigate('/inbox');
    }
  };

  const handleDelete = () => {
    if (selectedEmail && confirm('确定要删除这封邮件吗？')) {
      deleteEmail(selectedEmail.id);
      navigate('/inbox');
    }
  };

  const handleBack = () => {
    navigate('/inbox');
  };

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
    <div className="flex h-full flex-col">
      {/* 返回按钮 */}
      <div className="border-b px-4 py-2">
        <Button variant="ghost" size="sm" onClick={handleBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          返回列表
        </Button>
      </div>

      {/* 邮件详情 */}
      <div className="flex-1 overflow-hidden">
        <EmailDetail
          email={selectedEmail}
          onToggleStar={handleToggleStar}
          onArchive={handleArchive}
          onDelete={handleDelete}
        />
      </div>
    </div>
  );
};
