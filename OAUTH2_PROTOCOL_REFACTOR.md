# OAuth2 协议重构说明

## 背景

之前的设计中，OAuth2 认证被单独放在表单中作为"推荐选项"，但实际上它应该是一个协议选项。这导致了以下问题：

1. 用户需要在"密码登录"和"OAuth2 登录"之间做选择，但这个选择逻辑不够清晰
2. `auth_type` 和 `protocol` 字段的关系不够明确
3. UI 交互不够直观，用户可能会困惑

## 改进方案

将 OAuth2 作为一个独立的协议选项，与 IMAP、POP3 并列，这样逻辑会更清晰。

### 前端改动

#### 1. 协议下拉列表更新

**文件**: `frontend/src/components/account/AccountForm.tsx`

```tsx
{/* 协议 */}
<div className="space-y-2">
  <Label htmlFor="protocol">协议 *</Label>
  <Select
    value={formData.protocol}
    onValueChange={(value) => {
      setFormData({ ...formData, protocol: value });
      // 根据协议自动设置认证类型
      if (value === 'oauth2') {
        setFormData(prev => ({ ...prev, protocol: value, auth_type: 'oauth2' }));
      } else {
        setFormData(prev => ({ ...prev, protocol: value, auth_type: 'password' }));
      }
    }}
    disabled={isEditMode}
  >
    <SelectTrigger>
      <SelectValue />
    </SelectTrigger>
    <SelectContent>
      {/* Gmail 和 Outlook 支持 OAuth2 */}
      {(formData.provider === 'gmail' || formData.provider === 'outlook') && (
        <SelectItem value="oauth2">
          OAuth2（推荐 - 更安全）
        </SelectItem>
      )}
      <SelectItem value="imap">IMAP</SelectItem>
      <SelectItem value="pop3">POP3</SelectItem>
    </SelectContent>
  </Select>
  {formData.protocol === 'oauth2' && (
    <p className="text-xs text-blue-600 dark:text-blue-400">
      OAuth2 认证无需密码，通过官方授权页面安全登录
    </p>
  )}
</div>
```

#### 2. OAuth2 认证按钮显示逻辑

只在选择 OAuth2 协议时显示认证按钮：

```tsx
{/* OAuth2 认证按钮 */}
{!isEditMode && formData.protocol === 'oauth2' && (
  <div className="space-y-4 p-4 border rounded-lg bg-blue-50 dark:bg-blue-900/20">
    <div>
      <h4 className="font-medium text-sm text-gray-900 dark:text-white">
        OAuth2 安全认证
      </h4>
      <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
        点击下方按钮，通过官方授权页面安全登录，无需输入密码
      </p>
    </div>
    
    <OAuth2AuthButton
      provider={formData.provider === 'gmail' ? 'google' : 'microsoft'}
      email={formData.email}
      onSuccess={() => {
        onClose();
        navigate('/inbox');
      }}
      onError={(error) => {
        console.error('OAuth2 error:', error);
      }}
    />
  </div>
)}
```

#### 3. 密码输入框显示逻辑

只在非 OAuth2 协议时显示密码输入框：

```tsx
{/* 密码/授权码（仅在非 OAuth2 模式下显示） */}
{formData.protocol !== 'oauth2' && (
  <div className="space-y-2">
    <Label htmlFor="password">
      {isEditMode ? '新密码/授权码（留空则不修改）' : '密码/授权码 *'}
    </Label>
    <Input
      id="password"
      type="password"
      placeholder={isEditMode ? '留空则不修改密码' : '请输入密码或授权码'}
      value={formData.password}
      onChange={(e) =>
        setFormData({ ...formData, password: e.target.value })
      }
      required={!isEditMode}
    />
    {!isEditMode && (
      <p className="text-xs text-muted-foreground">
        {formData.provider === 'qq' || formData.provider === '163' 
          ? 'QQ/163 邮箱请使用授权码，而非登录密码'
          : formData.provider === 'gmail' || formData.provider === 'outlook'
          ? '建议使用应用专用密码，或切换到 OAuth2 协议获得更好的安全性'
          : '请输入邮箱密码或授权码'}
      </p>
    )}
  </div>
)}
```

### 后端改动

#### 1. 协议列表更新

**文件**: `backend/internal/adapter/factory.go`

```go
// GetSupportedProtocols 获取支持的协议列表
func (f *Factory) GetSupportedProtocols() []string {
	return []string{
		"oauth2",      // OAuth2 认证（Gmail、Outlook）
		"imap",        // IMAP 协议
		"pop3",        // POP3 协议
		"gmail_api",   // Gmail API（向后兼容）
		"graph",       // Microsoft Graph（向后兼容）
		"graph_quick", // Microsoft Graph 短效（向后兼容）
	}
}
```

#### 2. 推荐协议更新

```go
// GetRecommendedProtocol 获取推荐的协议
func (f *Factory) GetRecommendedProtocol(provider string) string {
	switch provider {
	case "gmail", "outlook":
		return "oauth2" // Gmail 和 Outlook 优先使用 OAuth2
	case "icloud", "qq", "163":
		return "imap" // 其他提供商使用 IMAP
	default:
		return "imap" // 默认使用 IMAP
	}
}
```

#### 3. 提供商信息更新

```go
"gmail": {
	Name:                "gmail",
	DisplayName:         "Gmail",
	SupportedProtocols:  []string{"oauth2", "imap"},
	RecommendedProtocol: "oauth2",
	RequiresOAuth:       true,
	IMAPHost:            "imap.gmail.com",
	IMAPPort:            993,
},
"outlook": {
	Name:                "outlook",
	DisplayName:         "Outlook / Hotmail",
	SupportedProtocols:  []string{"oauth2", "imap"},
	RecommendedProtocol: "oauth2",
	RequiresOAuth:       true,
	IMAPHost:            "outlook.office365.com",
	IMAPPort:            993,
},
```

#### 4. 适配器创建逻辑

```go
// 根据协议类型创建对应的适配器
switch config.Protocol {
case "imap":
	return NewIMAPAdapter(config)
case "pop3":
	return NewPOP3Adapter(config)
case "oauth2":
	// OAuth2 协议根据提供商选择具体实现
	switch config.Provider {
	case "gmail":
		return NewGmailAdapter(config)
	case "outlook":
		return NewGraphAdapter(config)
	default:
		return nil, fmt.Errorf("provider %s does not support oauth2", config.Provider)
	}
case "gmail_api":
	// 保留向后兼容
	return NewGmailAdapter(config)
case "graph":
	// 保留向后兼容
	return NewGraphAdapter(config)
case "graph_quick":
	return NewGraphQuickAdapter(config)
default:
	return nil, fmt.Errorf("unsupported protocol: %s", config.Protocol)
}
```

## 用户体验改进

### 之前的流程

1. 用户选择 Gmail 或 Outlook
2. 看到一个"推荐 OAuth2"的提示框
3. 需要在"OAuth2 按钮"和"密码输入框"之间选择
4. 不清楚这两种方式的区别

### 改进后的流程

1. 用户选择 Gmail 或 Outlook
2. 协议下拉列表自动选择 "OAuth2（推荐 - 更安全）"
3. 显示 OAuth2 认证按钮和说明
4. 如果用户想用密码登录，可以手动切换协议为 IMAP
5. 切换到 IMAP 后，显示密码输入框

### 优势

1. **逻辑清晰**：协议选择和认证方式直接关联
2. **用户友好**：默认推荐最安全的方式
3. **灵活性**：用户可以自由切换协议
4. **一致性**：所有协议都在同一个下拉列表中

## 向后兼容性

为了保持向后兼容，我们保留了以下协议：

- `gmail_api`: 映射到 Gmail OAuth2
- `graph`: 映射到 Outlook OAuth2
- `graph_quick`: 保留用于短效邮箱

现有的账户不会受到影响，系统会自动识别并使用正确的适配器。

## 测试建议

1. **新建账户测试**
   - 测试 Gmail OAuth2 认证
   - 测试 Outlook OAuth2 认证
   - 测试切换到 IMAP 协议
   - 测试 QQ/163 邮箱（应该默认 IMAP）

2. **编辑账户测试**
   - 确保编辑时协议字段被禁用
   - 确保可以修改同步设置

3. **向后兼容测试**
   - 确保现有的 `gmail_api` 账户仍然可用
   - 确保现有的 `graph` 账户仍然可用

## 总结

这次重构将 OAuth2 从一个"推荐选项"提升为一个正式的协议选项，使得整个账户添加流程更加清晰和直观。用户可以更容易地理解不同协议的区别，并做出合适的选择。
