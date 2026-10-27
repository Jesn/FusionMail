// WebAPI Provider 表单 · 配置步骤（纯展示）
// 步骤状态机、校验与提交逻辑留在 WebAPIProviderForm；本组件只渲染表单并回传变更。
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Switch } from '../ui/switch';
import { Alert, AlertDescription } from '../ui/alert';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { Loader2, ArrowLeft, ArrowRight, ChevronDown } from 'lucide-react';
import { WebAPIAuthConfig } from './WebAPIAuthConfig';
import { GroupSelector } from '../group';
import type {
  WebAPIServiceType,
  WebAPIServiceTemplate,
  WebAPIAuthData,
} from '../../types/webapi';

interface ConfigStepProps {
  /** 服务模板（显示模板名） */
  template?: WebAPIServiceTemplate;
  /** 显示名称 */
  name: string;
  onNameChange: (name: string) => void;
  /** 字段校验错误 */
  errors: Record<string, string>;
  /** 当前服务类型（决定认证配置表单） */
  serviceType?: WebAPIServiceType;
  /** 认证数据 */
  authData: WebAPIAuthData;
  onAuthDataChange: (data: WebAPIAuthData) => void;
  /** 分组 */
  groupId: number | null;
  onGroupChange: (groupId: number | null) => void;
  /** Webhook 模式（隐藏同步设置、直接创建） */
  isWebhookMode: boolean;
  syncEnabled: boolean;
  onSyncEnabledChange: (enabled: boolean) => void;
  /** 同步频率（分钟） */
  syncInterval: number;
  onSyncIntervalChange: (interval: number) => void;
  /** 提交错误（Webhook 模式直接创建时显示） */
  submitError: string | null;
  isSubmitting: boolean;
  onBack: () => void;
  onNext: () => void;
  onCancel?: () => void;
}

/**
 * 配置步骤（纯展示组件）：显示名称、分组、认证配置与同步设置表单。
 */
export const ConfigStep = ({
  template,
  name,
  onNameChange,
  errors,
  serviceType,
  authData,
  onAuthDataChange,
  groupId,
  onGroupChange,
  isWebhookMode,
  syncEnabled,
  onSyncEnabledChange,
  syncInterval,
  onSyncIntervalChange,
  submitError,
  isSubmitting,
  onBack,
  onNext,
  onCancel,
}: ConfigStepProps) => (
  <div className="space-y-6">
    <div className="flex items-center gap-2 mb-4">
      <Button variant="ghost" size="sm" onClick={onBack}>
        <ArrowLeft className="h-4 w-4 mr-1" />
        返回
      </Button>
      <span className="text-sm text-muted-foreground">
        配置 {template?.name || '服务'}
      </span>
    </div>

    {/* 基础信息 */}
    <div className="space-y-2">
      <Label htmlFor="name">显示名称 *</Label>
      <Input
        id="name"
        placeholder="输入显示名称"
        value={name}
        onChange={(e) => onNameChange(e.target.value)}
        className={errors.name ? 'border-red-500' : ''}
      />
      {errors.name && <p className="text-sm text-red-500">{errors.name}</p>}
    </div>

    {/* 分组选择 */}
    <div className="space-y-2">
      <Label>分组</Label>
      <GroupSelector
        value={groupId}
        onChange={onGroupChange}
        placeholder="选择分组（可选）"
      />
      <p className="text-xs text-muted-foreground">
        将账户归类到指定分组，便于管理
      </p>
    </div>

    {/* 认证配置 */}
    {serviceType && (
      <WebAPIAuthConfig
        serviceType={serviceType}
        authData={authData}
        onChange={onAuthDataChange}
        errors={errors}
      />
    )}

    {/* 同步设置 - 仅轮询模式显示 */}
    {!isWebhookMode && (
      <Collapsible defaultOpen>
        <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
          <ChevronDown className="h-4 w-4 transition-transform ui-closed:rotate-[-90deg]" />
          同步设置
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-2">
          <div className="flex items-center justify-between">
            <Label htmlFor="sync_enabled">启用自动同步</Label>
            <Switch
              id="sync_enabled"
              checked={syncEnabled}
              onCheckedChange={onSyncEnabledChange}
            />
          </div>
          {syncEnabled && (
            <div className="space-y-2">
              <Label>同步频率（分钟）</Label>
              <Select
                value={String(syncInterval)}
                onValueChange={(v) => onSyncIntervalChange(Number(v))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[1, 2, 5, 10, 15, 30, 60].map((m) => (
                    <SelectItem key={m} value={String(m)}>
                      {m} 分钟
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                设置自动同步邮件的时间间隔
              </p>
            </div>
          )}
        </CollapsibleContent>
      </Collapsible>
    )}

    {/* 提交错误（Webhook 模式直接创建时显示） */}
    {isWebhookMode && submitError && (
      <Alert variant="destructive">
        <AlertDescription>{submitError}</AlertDescription>
      </Alert>
    )}

    {/* 操作按钮 */}
    <div className="flex justify-end gap-2 pt-4">
      <Button variant="outline" onClick={onCancel}>
        取消
      </Button>
      {isWebhookMode ? (
        // Webhook 模式：直接创建，不需要测试
        <Button onClick={onNext} disabled={isSubmitting}>
          {isSubmitting ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              创建中...
            </>
          ) : (
            '创建'
          )}
        </Button>
      ) : (
        // 轮询模式：进入测试步骤
        <Button onClick={onNext}>
          下一步
          <ArrowRight className="h-4 w-4 ml-1" />
        </Button>
      )}
    </div>
  </div>

);
