// WebAPI Provider 表单 · 测试连接步骤（纯展示）
// 分组名称由容器从 groupStore 读取后传入；测试与提交逻辑留在 WebAPIProviderForm。
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Alert, AlertDescription } from '../ui/alert';
import { Loader2, CheckCircle2, XCircle, ArrowLeft } from 'lucide-react';
import type {
  WebAPIServiceTemplate,
  WebAPIAuthData,
  TestConnectionResult,
} from '../../types/webapi';

interface TestConnectionStepProps {
  /** 显示名称 */
  name: string;
  /** 服务模板（显示模板名） */
  template?: WebAPIServiceTemplate;
  /** 认证数据（摘要中显示 API 地址） */
  authData: WebAPIAuthData;
  /** 分组名称（容器从 store 解析） */
  groupName: string;
  /** Webhook 模式（摘要显示同步模式而非轮询设置） */
  isWebhookMode: boolean;
  syncEnabled: boolean;
  /** 同步频率（分钟） */
  syncInterval: number;
  /** 连接测试结果 */
  testResult: TestConnectionResult | null;
  /** 提交错误 */
  submitError: string | null;
  isTesting: boolean;
  isSubmitting: boolean;
  onBack: () => void;
  onTestConnection: () => void;
  onSubmit: () => void;
  onCancel?: () => void;
}

/**
 * 测试连接步骤（纯展示组件）：配置摘要、测试结果与创建按钮。
 */
export const TestConnectionStep = ({
  name,
  template,
  authData,
  groupName,
  isWebhookMode,
  syncEnabled,
  syncInterval,
  testResult,
  submitError,
  isTesting,
  isSubmitting,
  onBack,
  onTestConnection,
  onSubmit,
  onCancel,
}: TestConnectionStepProps) => (
  <div className="space-y-6">
    <div className="flex items-center gap-2 mb-4">
      <Button variant="ghost" size="sm" onClick={onBack}>
        <ArrowLeft className="h-4 w-4 mr-1" />
        返回
      </Button>
      <span className="text-sm text-muted-foreground">测试连接</span>
    </div>

    {/* 配置摘要 */}
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">配置摘要</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-muted-foreground">名称</span>
          <span>{name}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-muted-foreground">服务类型</span>
          <span>{template?.name}</span>
        </div>
        {(authData as any).base_url && (
          <div className="flex justify-between">
            <span className="text-muted-foreground">API 地址</span>
            <span className="truncate max-w-[200px]">{(authData as any).base_url}</span>
          </div>
        )}
        <div className="flex justify-between">
          <span className="text-muted-foreground">分组</span>
          <span>{groupName}</span>
        </div>
        {/* 同步信息 - 仅轮询模式显示 */}
        {!isWebhookMode && (
          <div className="flex justify-between">
            <span className="text-muted-foreground">自动同步</span>
            <span>{syncEnabled ? `每 ${syncInterval} 分钟` : '已禁用'}</span>
          </div>
        )}
        {/* Webhook 模式显示同步模式 */}
        {isWebhookMode && (
          <div className="flex justify-between">
            <span className="text-muted-foreground">同步模式</span>
            <span>Webhook 推送</span>
          </div>
        )}
      </CardContent>
    </Card>

    {/* 测试结果 */}
    {testResult && (
      <Alert variant={testResult.success ? 'default' : 'destructive'}>
        <div className="flex items-center gap-2">
          {testResult.success ? (
            <CheckCircle2 className="h-4 w-4 text-green-500" />
          ) : (
            <XCircle className="h-4 w-4" />
          )}
          <AlertDescription>
            {testResult.message}
            {testResult.email_count !== undefined && (
              <span className="ml-2">（发现 {testResult.email_count} 封邮件）</span>
            )}
            {testResult.error && (
              <span className="block text-xs mt-1">{testResult.error}</span>
            )}
          </AlertDescription>
        </div>
      </Alert>
    )}

    {/* 提交错误 */}
    {submitError && (
      <Alert variant="destructive">
        <AlertDescription>{submitError}</AlertDescription>
      </Alert>
    )}

    {/* 操作按钮 */}
    <div className="flex justify-between pt-4">
      <Button
        variant="outline"
        onClick={onTestConnection}
        disabled={isTesting}
      >
        {isTesting ? (
          <>
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            测试中...
          </>
        ) : (
          '测试连接'
        )}
      </Button>
      <div className="flex gap-2">
        <Button variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button
          onClick={onSubmit}
          disabled={isSubmitting || (testResult !== null && !testResult.success)}
        >
          {isSubmitting ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              创建中...
            </>
          ) : (
            '创建'
          )}
        </Button>
      </div>
    </div>
  </div>

);
