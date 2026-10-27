// Cloud Mail 认证字段（WebAPI 账户编辑对话框 · 纯展示）
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { Key, ChevronRight } from 'lucide-react';

// Cloud Mail 认证数据接口
export interface CloudMailAuthData {
  base_url: string;
  jwt_token: string;
  accounts?: Array<{ account_id?: number; email: string; name?: string }>;
}

interface CloudMailAuthFieldsProps {
  /** Cloud Mail 认证数据 */
  data: CloudMailAuthData;
  /** 局部更新认证字段（对话框侧合并进 authData） */
  onChange: (field: string, value: any) => void;
}

/**
 * Cloud Mail 认证字段（纯展示组件）：API 地址与 JWT Token 更新，不自己请求数据。
 */
export const CloudMailAuthFields = ({
  data,
  onChange,
}: CloudMailAuthFieldsProps) => {
  return (
    <div className="space-y-4">
      {/* API 地址（可编辑） */}
      <div className="space-y-2">
        <Label htmlFor="cm_base_url">API 地址</Label>
        <Input
          id="cm_base_url"
          placeholder="https://api.example.com"
          value={data.base_url || ''}
          onChange={(e) => onChange('base_url', e.target.value)}
        />
        <p className="text-xs text-muted-foreground">
          Cloud Mail 的 API 地址
        </p>
      </div>
      
      {/* 只读信息 */}
      {data.accounts && data.accounts.length > 0 && (
        <div className="p-3 rounded-lg bg-muted/30 space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">关联账户</span>
            <span>{data.accounts.length} 个</span>
          </div>
        </div>
      )}

      {/* 可编辑的认证信息 */}
      <Collapsible>
        <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
          <Key className="h-4 w-4" />
          更新认证信息
          <ChevronRight className="h-4 w-4 ml-auto transition-transform ui-open:rotate-90" />
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-3 pt-2">
          <div className="space-y-2">
            <Label htmlFor="jwt_token">JWT Token</Label>
            <Input
              id="jwt_token"
              type="password"
              placeholder="留空则不修改"
              value={data.jwt_token || ''}
              onChange={(e) => onChange('jwt_token', e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              从浏览器开发者工具获取新的 JWT Token
            </p>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
};
