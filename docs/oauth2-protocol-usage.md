# OAuth2 协议使用指南

## 概述

FusionMail 现在将 OAuth2 作为一个独立的协议选项，与 IMAP、POP3 并列。这使得用户可以更清晰地选择认证方式。

## 支持的提供商

### Gmail

- **推荐协议**: OAuth2
- **支持的协议**: OAuth2, IMAP
- **OAuth2 优势**:
  - 无需应用专用密码
  - 更安全的授权机制
  - 支持细粒度权限控制

### Outlook / Hotmail

- **推荐协议**: OAuth2
- **支持的协议**: OAuth2, IMAP
- **OAuth2 优势**:
  - 无需应用专用密码
  - 支持 Microsoft 账户的多因素认证
  - 更好的安全性

### 其他提供商

- **QQ 邮箱**: IMAP / POP3（需要授权码）
- **163 邮箱**: IMAP / POP3（需要授权码）
- **iCloud Mail**: IMAP（需要应用专用密码）
- **通用邮箱**: IMAP / POP3

## 用户操作流程

### 添加 Gmail 账户（OAuth2）

1. 点击"添加邮箱账户"
2. 输入 Gmail 邮箱地址（如 `user@gmail.com`）
3. 系统自动识别为 Gmail，并选择 OAuth2 协议
4. 点击"通过 Google 授权"按钮
5. 在弹出的 Google 授权页面登录并授权
6. 授权成功后自动返回，账户添加完成

### 添加 Outlook 账户（OAuth2）

1. 点击"添加邮箱账户"
2. 输入 Outlook 邮箱地址（如 `user@outlook.com`）
3. 系统自动识别为 Outlook，并选择 OAuth2 协议
4. 点击"通过 Microsoft 授权"按钮
5. 在弹出的 Microsoft 授权页面登录并授权
6. 授权成功后自动返回，账户添加完成

## 协议选择建议

### 何时使用 OAuth2

✅ **推荐使用 OAuth2 的情况**:
- Gmail 和 Outlook 账户
- 需要更高的安全性
- 不想管理应用专用密码
- 账户启用了多因素认证

### 何时使用 IMAP

✅ **推荐使用 IMAP 的情况**:
- OAuth2 不可用或不支持的提供商
- 需要更多的控制权
- 在某些网络环境下 OAuth2 授权页面无法访问

## 常见问题

### Q: OAuth2 和 IMAP 有什么区别？

A: 
- **OAuth2**: 通过官方授权页面登录，无需输入密码，更安全
- **IMAP**: 需要输入密码或应用专用密码，是传统的邮件协议

### Q: 为什么 Gmail 推荐使用 OAuth2？

A: 
- Google 正在逐步淘汰"不够安全的应用"访问
- OAuth2 提供更好的安全性和用户体验
- 支持细粒度的权限控制

### Q: OAuth2 的 Token 会过期吗？

A: 
- Access Token 会在 1 小时后过期
- 系统会自动使用 Refresh Token 刷新
- Refresh Token 通常有效期为 6 个月或更长
- 如果 Refresh Token 过期，需要重新授权
