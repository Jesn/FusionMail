import { WebhookCard } from './WebhookCard';
import { Webhook } from '../../services/webhookService';

interface WebhookListProps {
  webhooks: Webhook[];
  isLoading: boolean;
  onEdit: (webhook: Webhook) => void;
  onDelete: (id: number, name: string) => void;
  onToggle: (id: number) => void;
  onTest: (id: number) => void;
  onViewLogs: (webhook: Webhook) => void;
}

export const WebhookList = ({
  webhooks,
  isLoading,
  onEdit,
  onDelete,
  onToggle,
  onTest,
  onViewLogs,
}: WebhookListProps) => {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-muted-foreground">加载中...</div>
      </div>
    );
  }

  if (webhooks.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12">
        <p className="mb-4 text-lg text-muted-foreground">
          还没有创建 Webhook
        </p>
        <p className="text-sm text-muted-foreground">
          创建 Webhook 来接收邮件事件通知
        </p>
      </div>
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {webhooks.map((webhook) => (
        <WebhookCard
          key={webhook.id}
          webhook={webhook}
          onEdit={onEdit}
          onDelete={onDelete}
          onToggle={onToggle}
          onTest={onTest}
          onViewLogs={onViewLogs}
        />
      ))}
    </div>
  );
};