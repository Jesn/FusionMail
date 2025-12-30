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

interface WebAPIProviderDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

/**
 * WebAPI Provider 管理对话框
 * 包装 WebAPIProviderForm 组件
 */
export const WebAPIProviderDialog: React.FC<WebAPIProviderDialogProps> = ({
  open,
  onOpenChange,
  onSuccess,
}) => {
  const handleSuccess = () => {
    onOpenChange(false);
    onSuccess?.();
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>添加 WebAPI 服务</DialogTitle>
          <DialogDescription>
            配置 Web API 接入第三方邮件服务
          </DialogDescription>
        </DialogHeader>
        <WebAPIProviderForm
          onSuccess={handleSuccess}
          onCancel={handleCancel}
        />
      </DialogContent>
    </Dialog>
  );
};

export default WebAPIProviderDialog;
