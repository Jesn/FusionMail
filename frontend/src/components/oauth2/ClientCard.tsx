import { Trash2, Star } from 'lucide-react';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import { Label } from '../ui/label';
import type { OAuth2Client } from '../../types';

interface ClientCardProps {
  client: OAuth2Client;
  providerDisplayName: string;
  loading: boolean;
  onEdit: (client: OAuth2Client) => void;
  onSetDefault: (client: OAuth2Client) => void;
  /** 删除入口（页面负责打开确认框） */
  onDelete: (client: OAuth2Client) => void;
}

/** OAuth2 客户端卡片（自 OAuth2ClientsPage 同位搬移）。
 * 纯展示：不请求、不碰 store。 */
export function ClientCard({
  client,
  providerDisplayName,
  loading,
  onEdit,
  onSetDefault,
  onDelete,
}: ClientCardProps) {
  return (    <Card key={client.id} className="transition-all hover:shadow-md">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-lg">{client.name}</CardTitle>
              {client.is_default && (
                <Badge variant="secondary" className="gap-1">
                  <Star className="h-3 w-3" />
                  默认
                </Badge>
              )}
              {!client.enabled && (
                <Badge variant="outline">已禁用</Badge>
              )}
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>{providerDisplayName}</span>
              <span>•</span>
              <span>使用 {client.usage_count} 次</span>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <Label className="text-muted-foreground">Client ID</Label>
            <p className="font-mono text-xs mt-1 truncate">{client.client_id}</p>
          </div>
          <div>
            <Label className="text-muted-foreground">配额</Label>
            <p className="mt-1">
              日: {client.quota_daily >= 0 ? client.quota_daily : '无限制'} /
              月: {client.quota_monthly >= 0 ? client.quota_monthly : '无限制'}
            </p>
          </div>
        </div>

        {client.metadata && (
          <div className="text-sm">
            <Label className="text-muted-foreground">描述</Label>
            <p className="mt-1 text-muted-foreground">{
              (() => {
                try {
                  const parsed = JSON.parse(client.metadata);
                  return parsed.raw || client.metadata;
                } catch {
                  return client.metadata;
                }
              })()
            }</p>
          </div>
        )}

        <div className="flex items-center gap-2 pt-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => onEdit(client)}
            disabled={loading}
          >
            编辑
          </Button>
          {!client.is_default && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => onSetDefault(client)}
              disabled={loading}
            >
              <Star className="h-4 w-4 mr-1" />
              设为默认
            </Button>
          )}
          <Button
            size="sm"
            variant="destructive"
            onClick={() => {
              onDelete(client);
            }}
            disabled={loading}
          >
            <Trash2 className="h-4 w-4 mr-1" />
            删除
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
