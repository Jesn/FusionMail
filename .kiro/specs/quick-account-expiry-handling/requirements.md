# 短期邮箱账号过期自动处理需求文档

## 引言

本文档定义了 FusionMail 系统中短期邮箱账号（`AuthType = "quick"`）过期检测和自动禁用的功能需求。当短期邮箱账号连续多次同步失败且错误为认证失败时，系统应自动禁用该账号，避免浪费系统资源。

## 术语表

- **System**：FusionMail 邮件聚合系统
- **Quick Account**：使用快速认证方式（`AuthType = "quick"`）的短期邮箱账号
- **Sync Failure**：邮件同步过程中发生的错误
- **Auth Error**：认证相关的错误，包括 401 状态码和 token 过期错误
- **Consecutive Failures**：连续的同步失败次数，中间没有成功的同步
- **Auto-Disable**：系统自动将账号状态设置为 `disabled`

## 需求

### 需求 1：短期邮箱账号识别

**用户故事**：作为系统管理员，我希望系统能够识别短期邮箱账号，以便对其应用特殊的过期处理策略。

#### 验收标准

1. THE System SHALL identify an account as a Quick Account WHEN the account's `auth_type` field equals "quick"
2. THE System SHALL treat Quick Accounts differently from regular accounts in sync error handling
3. THE System SHALL maintain a failure counter for each Quick Account to track consecutive sync failures

### 需求 2：同步失败计数

**用户故事**：作为系统，我需要记录每个短期邮箱账号的连续同步失败次数，以便判断账号是否已过期。

#### 验收标准

1. WHEN a Quick Account sync fails with an Auth Error, THE System SHALL increment the account's consecutive failure counter by 1
2. WHEN a Quick Account sync succeeds, THE System SHALL reset the account's consecutive failure counter to 0
3. THE System SHALL persist the consecutive failure counter in the database
4. THE System SHALL include the consecutive failure counter in the account model with field name `consecutive_auth_failures` of type integer with default value 0

### 需求 3：认证错误识别

**用户故事**：作为系统，我需要准确识别认证相关的错误，以区分临时网络错误和永久性的账号过期。

#### 验收标准

1. THE System SHALL classify an error as an Auth Error WHEN the error contains HTTP status code 401
2. THE System SHALL classify an error as an Auth Error WHEN the error message contains the text "token expired" (case-insensitive)
3. THE System SHALL classify an error as an Auth Error WHEN the error message contains the text "invalid_grant" (case-insensitive)
4. THE System SHALL classify an error as an Auth Error WHEN the error message contains the text "authentication failed" (case-insensitive)
5. THE System SHALL NOT classify network timeout errors as Auth Errors
6. THE System SHALL NOT classify connection refused errors as Auth Errors

### 需求 4：自动禁用触发条件

**用户故事**：作为系统，当短期邮箱账号连续 3 次同步失败且都是认证错误时，我应该自动禁用该账号。

#### 验收标准

1. WHEN a Quick Account's consecutive failure counter reaches 3, THE System SHALL automatically set the account status to "disabled"
2. WHEN a Quick Account is auto-disabled, THE System SHALL record the disable reason as "auto_disabled_auth_failure"
3. WHEN a Quick Account is auto-disabled, THE System SHALL record the timestamp in a new field `auto_disabled_at`
4. THE System SHALL NOT auto-disable accounts with `auth_type` other than "quick"
5. THE System SHALL NOT auto-disable accounts if any of the 3 failures is not an Auth Error

### 需求 5：禁用后的行为

**用户故事**：作为系统，当短期邮箱账号被自动禁用后，我应该停止对其进行同步尝试，并保留其历史数据。

#### 验收标准

1. WHEN an account status is "disabled", THE System SHALL exclude the account from scheduled sync tasks
2. WHEN an account status is "disabled", THE System SHALL reject manual sync requests for the account with error message "Account is disabled"
3. WHEN an account is auto-disabled, THE System SHALL preserve all existing email data associated with the account
4. WHEN an account is auto-disabled, THE System SHALL preserve the account configuration in the database
5. WHEN an account is auto-disabled, THE System SHALL update the `last_sync_error` field with message "Account auto-disabled due to consecutive authentication failures"

### 需求 6：用户通知

**用户故事**：作为用户，当我的短期邮箱账号被自动禁用时，我希望在前端界面看到明确的提示信息。

#### 验收标准

1. WHEN an account is auto-disabled, THE System SHALL display a warning badge on the account card in the frontend
2. WHEN a user views an auto-disabled account, THE System SHALL show the disable reason "账号已自动禁用（连续认证失败）"
3. WHEN a user views an auto-disabled account, THE System SHALL show the auto-disable timestamp
4. THE System SHALL display the consecutive failure count in the account details page
5. THE System SHALL provide a visual indicator (icon or color) to distinguish auto-disabled accounts from manually disabled accounts

### 需求 7：手动重新启用

**用户故事**：作为用户，如果我重新获得了短期邮箱的访问权限，我希望能够手动重新启用该账号。

#### 验收标准

1. WHEN a user manually enables an auto-disabled account, THE System SHALL reset the consecutive failure counter to 0
2. WHEN a user manually enables an auto-disabled account, THE System SHALL clear the `auto_disabled_at` timestamp
3. WHEN a user manually enables an auto-disabled account, THE System SHALL set the account status to "active"
4. WHEN a user manually enables an auto-disabled account, THE System SHALL clear the `last_sync_error` field
5. WHEN a user manually enables an auto-disabled account, THE System SHALL allow the account to be included in sync tasks again

### 需求 8：日志记录

**用户故事**：作为系统管理员，我希望系统记录详细的日志，以便追踪短期邮箱账号的过期和禁用过程。

#### 验收标准

1. WHEN a Quick Account sync fails with an Auth Error, THE System SHALL log the error with level "WARN" including account UID, error message, and current failure count
2. WHEN a Quick Account is auto-disabled, THE System SHALL log the event with level "INFO" including account UID, email address, and total failure count
3. WHEN a Quick Account consecutive failure counter is reset, THE System SHALL log the event with level "DEBUG" including account UID
4. THE System SHALL include the sync attempt timestamp in all log entries
5. THE System SHALL include the error type classification (Auth Error vs Other Error) in log entries

### 需求 9：测试账号支持

**用户故事**：作为开发者，我需要使用已过期的测试账号来验证自动禁用功能是否正常工作。

#### 验收标准

1. THE System SHALL support testing with the expired account `cohuuexdw097@outlook.com`
2. WHEN the test account is synced, THE System SHALL correctly identify Auth Errors from the expired credentials
3. WHEN the test account fails 3 consecutive times, THE System SHALL auto-disable the account as expected
4. THE System SHALL allow manual re-enabling of the test account for repeated testing
5. THE System SHALL provide clear log output during testing to verify the auto-disable logic

### 需求 10：数据库模型扩展

**用户故事**：作为系统，我需要在数据库中存储与自动禁用相关的字段，以支持过期检测和状态管理。

#### 验收标准

1. THE System SHALL add field `consecutive_auth_failures` to the accounts table with type INTEGER and default value 0
2. THE System SHALL add field `auto_disabled_at` to the accounts table with type TIMESTAMP and nullable
3. THE System SHALL add field `disable_reason` to the accounts table with type VARCHAR(100) and nullable
4. THE System SHALL create a database migration script to add these fields to existing accounts table
5. THE System SHALL ensure the new fields are included in the Account model struct in Go backend
6. THE System SHALL ensure the new fields are included in the Account interface in TypeScript frontend

---

## 非功能需求

### 性能要求

- 连续失败计数的更新操作应在 100ms 内完成
- 自动禁用操作应在 200ms 内完成
- 不应影响正常账号的同步性能

### 可靠性要求

- 连续失败计数必须准确，不能因系统重启而丢失
- 自动禁用操作必须是原子性的，避免并发问题
- 日志记录必须完整，不能遗漏关键事件

### 可维护性要求

- 失败阈值（当前为 3）应可配置
- 认证错误的识别规则应易于扩展
- 代码应有清晰的注释和文档

---

## 实现优先级

1. **P0（必须实现）**：
   - 需求 1：短期邮箱账号识别
   - 需求 2：同步失败计数
   - 需求 3：认证错误识别
   - 需求 4：自动禁用触发条件
   - 需求 5：禁用后的行为
   - 需求 10：数据库模型扩展

2. **P1（应该实现）**：
   - 需求 6：用户通知
   - 需求 7：手动重新启用
   - 需求 8：日志记录

3. **P2（可以实现）**：
   - 需求 9：测试账号支持（开发阶段必需，生产环境可选）

---

## 测试策略

### 单元测试
- 测试认证错误识别逻辑
- 测试连续失败计数逻辑
- 测试自动禁用触发条件

### 集成测试
- 使用模拟的过期账号测试完整流程
- 测试数据库字段的正确更新
- 测试日志记录的完整性

### 端到端测试
- 使用真实的过期测试账号 `cohuuexdw097@outlook.com`
- 验证前端界面的正确显示
- 验证手动重新启用功能

---

## 附录：错误消息示例

### 认证错误示例

```
HTTP 401 Unauthorized
invalid_grant: Token has been expired or revoked
authentication failed: invalid credentials
token expired
AUTHENTICATE failed
```

### 非认证错误示例

```
connection timeout
connection refused
network unreachable
temporary failure
```
