// WebAPI Provider 表单 · 服务选择步骤（纯展示）
// 步骤状态机留在 WebAPIProviderForm；本组件只渲染服务类型选择列表。
import { WebAPIServiceSelector } from './WebAPIServiceSelector';
import type { WebAPIServiceType, WebAPIServiceTemplate } from '../../types/webapi';

interface ServiceSelectStepProps {
  /** 当前选中的服务类型 */
  serviceType: WebAPIServiceType | undefined;
  /** 选中服务回调（状态机留在容器组件） */
  onSelect: (type: WebAPIServiceType, tpl: WebAPIServiceTemplate) => void;
}

/**
 * 服务选择步骤（纯展示组件）：选择预置服务或配置自定义 Web API。
 */
export const ServiceSelectStep = ({ serviceType, onSelect }: ServiceSelectStepProps) => (
  <div className="space-y-4">
    <div className="text-center mb-6">
      <h3 className="text-lg font-medium">选择服务类型</h3>
      <p className="text-sm text-muted-foreground">
        选择预置服务或配置自定义 Web API
      </p>
    </div>
    <WebAPIServiceSelector
      selectedType={serviceType}
      onSelect={onSelect}
    />
  </div>

);
