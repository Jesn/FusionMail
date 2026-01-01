// WebAPI Provider 管理对话框
import React from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { WebAPIProviderForm } from './WebAPIProviderForm';
import type { WebAPIServiceType } from '../../types/webapi';

interface WebAPIProviderDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
  /** 预选的服务类型，如果提供则跳过选择步骤 */
  preselectedServiceType?: WebAPIServiceType;
}

/**
 * WebAPI Provider 管理对话框
 * 包装 WebAPIProviderForm 组件
 */
export const WebAPIProviderDialog: React.FC<WebAPIProviderDialogProps> = ({
  open,
  onOpenChange,
  onSuccess,
  preselectedServiceType,
}) => {
  const handleSuccess = () => {
    onOpenChange(false);
    onSuccess?.();
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  // 根据预选服务类型生成标题
  const getDialogTitle = () => {
    if (preselectedServiceType === 'cloudflare_temp_email') {
      return '添加 Cloudflare Temp Email 账户';
    }
    if (preselectedServiceType === 'cloud_mail') {
      return '添加 Cloud Mail 账户';
    }
    return '添加 WebAPI 服务';
  };

  const getDialogDescription = () => {
    if (preselectedServiceType === 'cloudflare_temp_email') {
      return '配置 Cloudflare Workers 临时邮箱服务';
    }
    if (preselectedServiceType === 'cloud_mail') {
      return '配置 Cloud Mail 多账户邮箱服务';
    }
    return '配置 Web API 接入第三方邮件服务';
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{getDialogTitle()}</DialogTitle>
          <DialogDescription>
            {getDialogDescription()}
          </DialogDescription>
        </DialogHeader>
        <WebAPIProviderForm
          onSuccess={handleSuccess}
          onCancel={handleCancel}
          preselectedServiceType={preselectedServiceType}
        />
      </DialogContent>
    </Dialog>
  );
};

export default WebAPIProviderDialog;
