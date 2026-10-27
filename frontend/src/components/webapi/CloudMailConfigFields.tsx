// Cloud Mail 配置字段（WebAPI 认证配置 · 纯展示）
// 注意：与 CloudMailAuthFields（WebAPI 账户编辑对话框）是两个组件，本组件用于新建/配置流程。
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import type { CloudMailAuthData } from '../../types/webapi';

interface CloudMailConfigFieldsProps {
  /** Cloud Mail 认证数据 */
  data: CloudMailAuthData;
  /** 局部更新认证字段（容器侧合并进 authData） */
  onChange: (field: string, value: any) => void;
  /** 字段校验错误 */
  errors: Record<string, string>;
}

/**
 * Cloud Mail 配置字段（纯展示组件）：API 地址与认证方式（邮箱密码 / JWT Token）表单，不自行请求数据。
 */
export const CloudMailConfigFields = ({
  data,
  onChange,
  errors,
}: CloudMailConfigFieldsProps) => {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="base_url">API 地址 *</Label>
        <Input
          id="base_url"
          type="url"
          placeholder="https://your-cloud-mail-domain.com"
          value={data.base_url || ''}
          onChange={(e) => onChange('base_url', e.target.value)}
          className={errors.base_url ? 'border-red-500' : ''}
        />
        {errors.base_url && <p className="text-sm text-red-500">{errors.base_url}</p>}
      </div>

      <div className="rounded-lg border p-4 space-y-4">
        <div className="text-sm font-medium">认证方式（二选一）</div>
        
        <div className="space-y-3">
          <div className="text-xs text-muted-foreground">方式一：邮箱+密码登录（推荐）</div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="email">登录邮箱</Label>
              <Input
                id="email"
                type="email"
                placeholder="user@example.com"
                value={data.email || ''}
                onChange={(e) => onChange('email', e.target.value)}
                className={errors.email ? 'border-red-500' : ''}
              />
              {errors.email && <p className="text-sm text-red-500">{errors.email}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">登录密码</Label>
              <Input
                id="password"
                type="password"
                placeholder="输入密码"
                value={data.password || ''}
                onChange={(e) => onChange('password', e.target.value)}
                className={errors.password ? 'border-red-500' : ''}
              />
              {errors.password && <p className="text-sm text-red-500">{errors.password}</p>}
            </div>
          </div>
        </div>

        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <span className="w-full border-t" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">或</span>
          </div>
        </div>

        <div className="space-y-3">
          <div className="text-xs text-muted-foreground">方式二：直接输入 JWT Token</div>
          <div className="space-y-2">
            <Label htmlFor="jwt_token">JWT Token</Label>
            <Input
              id="jwt_token"
              type="password"
              placeholder="从浏览器获取的 authorization token"
              value={data.jwt_token || ''}
              onChange={(e) => onChange('jwt_token', e.target.value)}
              className={errors.jwt_token ? 'border-red-500' : ''}
            />
            {errors.jwt_token && <p className="text-sm text-red-500">{errors.jwt_token}</p>}
          </div>
        </div>
      </div>
    </div>
  );
};
