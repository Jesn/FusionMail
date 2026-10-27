import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import type { OAuth2Client } from '../../types';

interface OAuth2StatsCardsProps {
  clients: OAuth2Client[];
}

/** 统计卡片组（自 OAuth2ClientsPage 同位搬移）。纯展示。 */
export function OAuth2StatsCards({ clients }: OAuth2StatsCardsProps) {
  return (
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              总配置数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{Array.isArray(clients) ? clients.length : 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              启用中
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(clients) ? clients.filter(c => c.enabled).length : 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              默认配置
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(clients) ? clients.filter(c => c.is_default).length : 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              总使用次数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(clients) ? clients.reduce((sum, c) => sum + c.usage_count, 0) : 0}
            </div>
          </CardContent>
        </Card>
      </div>
  );
}
