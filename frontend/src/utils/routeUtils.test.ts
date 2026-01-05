import { describe, it, expect } from 'vitest';
import { isSettingsRoute, getViewMode, SETTINGS_ROUTES, MAIL_ROUTES } from './routeUtils';

describe('routeUtils', () => {
  describe('isSettingsRoute', () => {
    it('应该正确识别设置路由', () => {
      // 测试所有定义的设置路由
      SETTINGS_ROUTES.forEach((route) => {
        expect(isSettingsRoute(route)).toBe(true);
      });
    });

    it('应该正确识别设置路由的子路由', () => {
      expect(isSettingsRoute('/settings/system')).toBe(true);
      expect(isSettingsRoute('/settings/profile')).toBe(true);
      expect(isSettingsRoute('/accounts/123')).toBe(true);
    });

    it('应该正确识别非设置路由', () => {
      expect(isSettingsRoute('/inbox')).toBe(false);
      expect(isSettingsRoute('/sent')).toBe(false);
      expect(isSettingsRoute('/spam')).toBe(false);
      expect(isSettingsRoute('/search')).toBe(false);
      expect(isSettingsRoute('/email/123')).toBe(false);
      expect(isSettingsRoute('/')).toBe(false);
    });

    it('应该正确处理邮件相关路由', () => {
      MAIL_ROUTES.forEach((route) => {
        expect(isSettingsRoute(route)).toBe(false);
      });
    });
  });

  describe('getViewMode', () => {
    it('应该为设置路由返回 settings 模式', () => {
      expect(getViewMode('/settings')).toBe('settings');
      expect(getViewMode('/accounts')).toBe('settings');
      expect(getViewMode('/rules')).toBe('settings');
      expect(getViewMode('/webhooks')).toBe('settings');
      expect(getViewMode('/api-keys')).toBe('settings');
      expect(getViewMode('/providers')).toBe('settings');
      expect(getViewMode('/oauth2-clients')).toBe('settings');
      expect(getViewMode('/email-list')).toBe('settings');
      expect(getViewMode('/logs')).toBe('settings');
      expect(getViewMode('/api-docs')).toBe('settings');
      expect(getViewMode('/trash')).toBe('settings');
    });

    it('应该为邮件路由返回 mail 模式', () => {
      expect(getViewMode('/inbox')).toBe('mail');
      expect(getViewMode('/sent')).toBe('mail');
      expect(getViewMode('/spam')).toBe('mail');
      expect(getViewMode('/search')).toBe('mail');
      expect(getViewMode('/email/123')).toBe('mail');
    });

    it('应该为未知路由返回 mail 模式（默认）', () => {
      expect(getViewMode('/')).toBe('mail');
      expect(getViewMode('/unknown')).toBe('mail');
      expect(getViewMode('/random/path')).toBe('mail');
    });
  });

  describe('SETTINGS_ROUTES 常量', () => {
    it('应该包含所有必需的设置路由', () => {
      const requiredRoutes = [
        '/settings',
        '/settings/system',
        '/accounts',
        '/trash',
        '/rules',
        '/webhooks',
        '/api-keys',
        '/providers',
        '/oauth2-clients',
        '/email-list',
        '/logs',
        '/api-docs',
      ];

      requiredRoutes.forEach((route) => {
        expect(SETTINGS_ROUTES).toContain(route);
      });
    });
  });

  describe('MAIL_ROUTES 常量', () => {
    it('应该包含所有必需的邮件路由', () => {
      const requiredRoutes = ['/inbox', '/sent', '/spam', '/search', '/email'];

      requiredRoutes.forEach((route) => {
        expect(MAIL_ROUTES).toContain(route);
      });
    });
  });
});
