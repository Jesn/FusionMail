import {
  Plus,
  Trash2,
  AlertCircle,
  RefreshCw,
  ShieldCheck,
  Globe,
  ExternalLink,
} from 'lucide-react';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import { Label } from '../ui/label';
import type { Provider } from '../../types';
import { getAdapterDisplayName } from '../../types/adapter';

interface ProviderCardProps {
  provider: Provider;
  /** 是否为系统内置的 WebAPI 供应商（页面计算） */
  isWebAPI: boolean;
  /** 全局 loading（禁用操作按钮） */
  loading: boolean;
  onEdit: (provider: Provider) => void;
  /** 点击删除（页面负责打开确认对话框） */
  onDelete: (provider: Provider) => void;
}

/**
 * 单个提供商卡片（纯展示组件，仅被 ProviderListSection 使用）：
 * WebAPI 供应商展示类型与 GitHub 链接，传统供应商展示服务器/协议信息。
 */
const ProviderCard = ({
  provider,
  isWebAPI,
  loading,
  onEdit,
  onDelete,
}: ProviderCardProps) => {
  return (
    <Card key={provider.id} className="transition-all hover:shadow-md">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-lg">{provider.display_name}</CardTitle>
              {!provider.enabled && (
                <Badge variant="outline">已禁用</Badge>
              )}
              {isWebAPI && (
                <Badge variant="secondary" className="gap-1">
                  <Globe className="h-3 w-3" />
                  WebAPI
                </Badge>
              )}
              {provider.requires_oauth && !isWebAPI && (
                <Badge variant="secondary" className="gap-1">
                  <ShieldCheck className="h-3 w-3" />
                  OAuth2
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>{provider.name}</span>
              <span>•</span>
              <span>排序: {provider.sort_order}</span>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* WebAPI 供应商显示不同的内容 */}
        {isWebAPI ? (
          <div className="space-y-3">
            <div className="text-sm">
              <Label className="text-muted-foreground">类型</Label>
              <p className="mt-1">Web API 服务（系统内置）</p>
            </div>
            {/* 显示 GitHub 链接 */}
            {(() => {
              try {
                const metadata = provider.metadata ? JSON.parse(provider.metadata) : null;
                const githubUrl = metadata?.github_url;
                if (githubUrl) {
                  return (
                    <div className="text-sm">
                      <Label className="text-muted-foreground">项目地址</Label>
                      <a
                        href={githubUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-1 flex items-center gap-1 text-primary hover:underline"
                      >
                        <ExternalLink className="h-3 w-3" />
                        GitHub
                      </a>
                    </div>
                  );
                }
                return null;
              } catch {
                return null;
              }
            })()}
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <Label className="text-muted-foreground">IMAP 服务器</Label>
              <p className="font-mono text-xs mt-1 truncate">{provider.imap_host}:{provider.imap_port}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">推荐协议</Label>
              <p className="mt-1">{provider.recommended_protocol.toUpperCase()}</p>
            </div>
          </div>
        )}

        {/* 连接方式 - 仅非 WebAPI 供应商显示 */}
        {!isWebAPI && provider.supported_adapters && provider.supported_adapters.length > 0 && (
          <div className="text-sm">
            <Label className="text-muted-foreground">连接方式</Label>
            <div className="flex flex-wrap gap-1 mt-1">
              {/* supported_adapters 元素是 ProviderAdapter 关联，适配器信息嵌套在 .adapter 中 */}
              {provider.supported_adapters.map((pa) => (
                <Badge key={pa.adapter_id} variant="outline" className="text-xs">
                  {getAdapterDisplayName(pa.adapter?.name ?? `#${pa.adapter_id}`)}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {/* 邮箱域名 */}
        {provider.email_domains && provider.email_domains.length > 0 && (
          <div className="text-sm">
            <Label className="text-muted-foreground flex items-center gap-1">
              <Globe className="h-3 w-3" />
              支持域名
            </Label>
            <p className="mt-1 text-xs text-muted-foreground">
              {provider.email_domains.join(', ')}
            </p>
          </div>
        )}

        {provider.description && (
          <div className="text-sm">
            <Label className="text-muted-foreground">描述</Label>
            <p className="mt-1 text-muted-foreground">{provider.description}</p>
          </div>
        )}

        <div className="flex items-center gap-2 pt-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => onEdit(provider)}
            disabled={loading}
          >
            编辑
          </Button>
          {/* 系统内置的 WebAPI 供应商不能删除，只能启用/禁用 */}
          {!isWebAPI && (
            <Button
              size="sm"
              variant="destructive"
              onClick={() => onDelete(provider)}
              disabled={loading}
            >
              <Trash2 className="h-4 w-4 mr-1" />
              删除
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
};

interface ProviderListSectionProps {
  /** Provider 完整列表 */
  providers: Provider[];
  /** 全局 loading（控制刷新按钮与卡片按钮禁用） */
  isLoading: boolean;
  /** 判断是否为系统内置 WebAPI 供应商（纯函数，页面持有） */
  isWebAPIProvider: (provider: Provider) => boolean;
  /** 点击刷新按钮 */
  onRefresh: () => void;
  /** 空状态下点击"创建第一个配置" */
  onCreateFirst: () => void;
  onEditProvider: (provider: Provider) => void;
  onDeleteProvider: (provider: Provider) => void;
}

/**
 * 提供商列表区块（纯展示组件）：
 * 含标题/刷新按钮、加载与空状态、提供商卡片网格。
 * 不自己请求数据，全部数据与回调来自页面。
 */
export const ProviderListSection = ({
  providers,
  isLoading,
  isWebAPIProvider,
  onRefresh,
  onCreateFirst,
  onEditProvider,
  onDeleteProvider,
}: ProviderListSectionProps) => {
  // 后端返回异常时页面会置为空数组，这里保留 Array.isArray 防御
  const hasNoProviders = !Array.isArray(providers) || providers.length === 0;
  const providerList = hasNoProviders ? [] : providers;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">配置列表</h2>
        <Button variant="outline" size="sm" onClick={onRefresh} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      {isLoading && hasNoProviders ? (
        <Card>
          <CardContent className="flex items-center justify-center py-8">
            <RefreshCw className="h-6 w-6 animate-spin mr-2" />
            加载中...
          </CardContent>
        </Card>
      ) : hasNoProviders ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-8">
            <AlertCircle className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">暂无邮箱提供商配置</p>
            <Button
              className="mt-4"
              onClick={onCreateFirst}
            >
              <Plus className="h-4 w-4 mr-2" />
              创建第一个配置
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {providerList.map((provider) => (
            <ProviderCard
              key={provider.id}
              provider={provider}
              isWebAPI={isWebAPIProvider(provider)}
              loading={isLoading}
              onEdit={onEditProvider}
              onDelete={onDeleteProvider}
            />
          ))}
        </div>
      )}
    </div>
  );
};
