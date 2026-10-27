import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import type { Provider } from '../../types';

interface ProviderStatsCardsProps {
  /** Provider 完整列表（组件内部派生统计数字） */
  providers: Provider[];
}

/**
 * 提供商页顶部统计卡片（纯展示组件）：
 * 仅根据传入的 Provider 列表计算统计数字，不自己请求数据、不读写 store。
 */
export const ProviderStatsCards = ({ providers }: ProviderStatsCardsProps) => {
  // 后端返回异常时页面会置为空数组，这里保留 Array.isArray 防御
  const providerList = Array.isArray(providers) ? providers : [];
  const enabledCount = providerList.filter((p) => p.enabled).length;
  const oauth2Count = providerList.filter((p) => p.requires_oauth).length;
  const imapCount = providerList.filter((p) =>
    p.supported_protocols.includes('imap')
  ).length;

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            总提供商数
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{providerList.length}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            启用中
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{enabledCount}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            支持 OAuth2
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{oauth2Count}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            IMAP 支持
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{imapCount}</div>
        </CardContent>
      </Card>
    </div>
  );
};
