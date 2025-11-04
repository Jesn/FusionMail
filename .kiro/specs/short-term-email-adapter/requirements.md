# 短效邮箱适配器需求文档

## 简介

短效邮箱适配器是 FusionMail 系统中专门用于处理 Microsoft Outlook/Hotmail 账户快速导入和验证的功能模块。它提供了一种简化的认证方式，适用于批量导入、测试验证等场景，与标准 OAuth2 流程并行存在。

**重要约束**：短效邮箱适配器的后端实现流程必须与 `backend/micro.py` 的参考实现完全一致，以确保短效邮箱能够正确接收邮件。

## 术语表

- **短效适配器 (Short-Term Adapter)**: 使用简化认证流程的邮箱适配器
- **标准适配器 (Standard Adapter)**: 使用完整 OAuth2 流程的邮箱适配器  
- **Refresh Token**: Microsoft Graph API 的刷新令牌
- **Access Token**: Microsoft Graph API 的访问令牌
- **Client ID**: Microsoft Azure 应用程序的客户端标识符
- **Graph API**: Microsoft Graph REST API 服务
- **批量导入**: 一次性导入多个邮箱账户的功能
- **参考实现 (Reference Implementation)**: `backend/micro.py` 文件，定义了短效邮箱的标准实现流程

## 需求

### 需求 1: 短效适配器基础功能（必须与 micro.py 对齐）

**用户故事**: 作为系统管理员，我希望能够使用简化的认证方式快速导入 Microsoft 邮箱账户，以便进行批量数据迁移和测试验证。

#### 验收标准

1. WHEN 系统接收到包含 refresh_token 和 client_id 的账户信息时，短效适配器 SHALL 使用与 micro.py 完全相同的方式调用 `https://login.microsoftonline.com/common/oauth2/v2.0/token` 端点获取 access_token
2. WHEN 短效适配器获取到有效的 access_token 时，系统 SHALL 使用与 micro.py 完全相同的方式调用 `https://graph.microsoft.com/v1.0/me/mailFolders/inbox/messages` 端点获取收件箱邮件
3. WHEN 短效适配器连接测试时，系统 SHALL 使用与 micro.py 相同的 `/me` 端点验证连接状态
4. THE 短效适配器 SHALL 使用 scope "https://graph.microsoft.com/.default" 进行认证（与 micro.py 一致）
5. THE 短效适配器 SHALL 使用 POST 方法和 `application/x-www-form-urlencoded` 内容类型请求 token（与 micro.py 一致）
6. THE 短效适配器 SHALL 使用 `Bearer {access_token}` 格式的 Authorization 头访问 Graph API（与 micro.py 一致）
7. THE 短效适配器的核心邮件获取流程 SHALL 与 micro.py 的 `print_inbox` 函数逻辑完全一致

### 需求 2: 适配器选择机制

**用户故事**: 作为开发者，我希望系统能够根据账户类型和配置自动选择合适的适配器，以便为不同场景提供最优的处理方式。

#### 验收标准

1. WHEN 账户配置包含短效认证信息时，系统 SHALL 优先使用短效适配器
2. WHEN 账户配置包含标准 OAuth2 信息时，系统 SHALL 使用标准适配器
3. THE 适配器工厂 SHALL 根据账户配置自动选择合适的适配器类型
4. THE 系统 SHALL 支持在运行时切换适配器类型
5. WHERE 账户同时支持两种认证方式时，系统 SHALL 提供配置选项供用户选择

### 需求 3: 批量导入支持

**用户故事**: 作为用户，我希望能够批量导入多个 Microsoft 邮箱账户，以便快速完成数据迁移任务。

#### 验收标准

1. WHEN 用户提交批量导入请求时，系统 SHALL 并行处理多个账户的验证
2. WHEN 单个账户验证失败时，系统 SHALL 继续处理其他账户而不中断整个流程
3. THE 系统 SHALL 返回每个账户的处理结果（成功/失败及错误信息）
4. THE 批量导入 SHALL 支持最多 50 个账户的并发处理
5. WHEN 批量导入完成时，系统 SHALL 提供详细的统计报告

### 需求 4: 错误处理和日志

**用户故事**: 作为系统维护人员，我希望系统能够提供详细的错误信息和日志记录，以便快速定位和解决问题。

#### 验收标准

1. WHEN 短效适配器遇到认证错误时，系统 SHALL 记录详细的错误信息和请求上下文
2. WHEN API 调用失败时，系统 SHALL 记录 HTTP 状态码、响应内容和请求参数
3. THE 系统 SHALL 为每个适配器操作生成唯一的追踪 ID
4. THE 错误信息 SHALL 包含用户友好的描述和技术详情
5. WHEN 连接测试失败时，系统 SHALL 提供具体的失败原因和建议解决方案

### 需求 5: 性能和可靠性

**用户故事**: 作为系统用户，我希望短效适配器能够提供稳定可靠的服务，确保邮件数据的准确性和完整性。

#### 验收标准

1. THE 短效适配器 SHALL 在 5 秒内完成单个账户的连接测试
2. THE 系统 SHALL 支持最多 10 个并发的短效适配器连接
3. WHEN access_token 过期时，短效适配器 SHALL 自动使用 refresh_token 获取新的令牌
4. THE 短效适配器 SHALL 实现指数退避重试机制处理临时网络错误
5. THE 系统 SHALL 缓存有效的 access_token 以减少不必要的刷新请求

### 需求 6: 安全性

**用户故事**: 作为安全管理员，我希望短效适配器能够安全地处理敏感的认证信息，确保用户数据的安全性。

#### 验收标准

1. THE 短效适配器 SHALL 使用 AES-256 加密存储 refresh_token
2. THE 系统 SHALL 在内存中安全地处理 access_token，避免明文日志记录
3. WHEN 适配器实例销毁时，系统 SHALL 清除内存中的敏感信息
4. THE 系统 SHALL 验证 client_id 的格式和有效性
5. THE 短效适配器 SHALL 支持代理配置以满足网络安全要求

### 需求 7: 自动化测试

**用户故事**: 作为质量保证工程师，我希望系统提供完整的自动化测试覆盖，确保短效适配器功能的正确性和稳定性。

#### 验收标准

1. THE 系统 SHALL 提供单元测试覆盖所有短效适配器的核心功能
2. THE 系统 SHALL 提供集成测试验证与 Microsoft Graph API 的交互
3. THE 系统 SHALL 提供端到端测试覆盖批量导入的完整流程
4. THE 测试套件 SHALL 包含错误场景和边界条件的测试用例
5. THE 系统 SHALL 支持使用模拟数据进行测试，避免依赖真实的 Microsoft 账户
6. THE 系统 SHALL 提供对比测试，验证 Go 实现与 micro.py 参考实现的行为一致性

### 需求 8: 与参考实现的一致性

**用户故事**: 作为开发者，我希望短效适配器的实现与 micro.py 参考实现完全一致，以确保功能的正确性和可靠性。

#### 验收标准

1. THE 短效适配器 SHALL 使用与 micro.py 相同的 HTTP 请求方法和参数
2. THE 短效适配器 SHALL 使用与 micro.py 相同的 API 端点路径
3. THE 短效适配器 SHALL 使用与 micro.py 相同的请求头设置
4. THE 短效适配器 SHALL 使用与 micro.py 相同的请求体格式
5. THE 短效适配器 SHALL 解析与 micro.py 相同的响应字段
6. WHEN 实现新功能时，系统 SHALL 首先确保核心流程与 micro.py 一致，然后再添加扩展功能
7. THE 系统 SHALL 提供文档说明哪些功能是核心流程（必须与 micro.py 一致），哪些是扩展功能