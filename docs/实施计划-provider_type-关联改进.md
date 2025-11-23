# OAuth2 客户端与 Provider 关联改进计划

## 📋 改进目标

将 `oauth2_clients` 表与 `providers` 表的关联方式从字符串名称匹配改为更可靠的 `provider_type` 枚举值匹配。

## 🔄 变更内容

### 1. 数据库层面

#### 1.1 添加 `provider_type` 字段

**迁移文件：** `backend/migrations/009_add_provider_type_to_oauth2_clients.sql`

```sql
-- 添加字段
ALTER TABLE oauth2_clients
ADD COLUMN provider_type INTEGER NOT NULL DEFAULT 1;

-- 更新现有数据
UPDATE oauth2_clients SET provider_type = 1 WHERE provider_name = 'gmail';
UPDATE oauth2_clients SET provider_type = 2 WHERE provider_name = 'outlook';

-- 创建索引
CREATE INDEX idx_oauth2_clients_provider_type ON oauth2_clients(provider_type);

-- 更新唯一约束
CREATE UNIQUE INDEX uk_oauth2_clients_provider_type_default
ON oauth2_clients(provider_type)
WHERE is_default = TRUE AND enabled = TRUE;
```

#### 1.2 Provider 表添加 `provider_type` 字段

**迁移文件：** 通过 GORM AutoMigrate 自动添加

```go
type Provider struct {
  // ... 其他字段
  ProviderType int `gorm:"uniqueIndex;not null;default:1" json:"provider_type"`
  // ... 其他字段
}
```

### 2. 后端代码层面

#### 2.1 模型定义

**文件：** `backend/internal/model/provider.go`
- ✅ 添加 `ProviderType` 枚举常量
- ✅ 添加 `ProviderType` 的 `String()` 方法
- ✅ 在 `Provider` 模型中添加 `ProviderType` 字段

**文件：** `backend/internal/model/oauth2_client.go`
- ✅ 添加 `ProviderType` 枚举常量（与 Provider 保持一致）
- ✅ 在 `OAuth2Client` 模型中添加 `ProviderType` 字段
- ✅ 保留 `ProviderName` 字段（向前兼容）

#### 2.2 数据库初始化

**文件：** `backend/pkg/database/database.go`
- ✅ 更新 `seedProviders()` 函数，为每个 Provider 设置 `ProviderType`

#### 2.3 同步脚本

**文件：** `backend/internal/handler/dev_sync_handler.go`
- ✅ 添加 `mapProviderNameToType()` 函数
- ✅ 修改查询逻辑，使用 `provider_type` 字段
- ✅ 修改创建/更新逻辑，设置 `provider_type` 字段

### 3. 前端代码层面

#### 3.1 类型定义

**文件：** `frontend/src/types/providerType.ts`
- ✅ 创建 `ProviderType` 枚举（与后端一致）
- ✅ 创建 `ProviderTypeMap` 映射表
- ✅ 提供辅助函数：`getProviderTypeDisplayName()`、`getProviderTypeFromName()`

## 📊 ProviderType 枚举值

| 值 | 常量名 | 显示名称 | Provider Name |
|---|--------|----------|---------------|
| 1 | ProviderTypeGmail | Gmail | gmail |
| 2 | ProviderTypeOutlook | Outlook / Hotmail | outlook |
| 3 | ProviderTypeIcloud | iCloud Mail | icloud |
| 4 | ProviderTypeQQ | QQ 邮箱 | qq |
| 5 | ProviderType163 | 163 邮箱 | 163 |
| 6 | ProviderTypeGeneric | 通用邮箱 | generic |

## 🚀 实施步骤

### 步骤 1：执行数据库迁移

```bash
# 停止后端服务
pkill -f "./fusionmail"

# 执行迁移（通过 GORM AutoMigrate）
go run ./cmd/server
```

### 步骤 2：验证迁移

```sql
-- 检查 Provider 表是否添加了 provider_type 字段
SELECT id, name, display_name, provider_type FROM providers;

-- 检查 oauth2_clients 表是否添加了 provider_type 字段
SELECT id, provider_name, provider_type, name, client_id FROM oauth2_clients;
```

### 步骤 3：测试同步

```bash
# 重新启动后端服务
go build -o fusionmail ./cmd/server
./fusionmail &

# 调用同步端点
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  http://localhost:3333/api/v1/dev/sync-from-env
```

### 步骤 4：验证数据

```bash
# 检查同步后的 OAuth2 客户端
curl -s -H "Authorization: Bearer <token>" http://localhost:3333/api/v1/oauth2/clients | jq '.data[] | {id, provider_name, provider_type, name, client_id}'

# 应该看到：
# {
#   "id": 1,
#   "provider_name": "gmail",
#   "provider_type": 1,
#   "name": "默认配置",
#   "client_id": "28698829185-evea9bupqunm53pi5jdeajsspicsae0p.apps.googleusercontent.com"
# }
# {
#   "id": 2,
#   "provider_name": "outlook",
#   "provider_type": 2,
#   "name": "默认配置",
#   "client_id": "0ec56a84-6012-4ac5-81a5-e61f6a1f4438"
# }
```

## 🔍 优势对比

### 旧方案（字符串关联）
- ❌ `oauth2_clients.provider_name` → `providers.name`
- ❌ 通过字符串匹配，容易出错
- ❌ Provider name 变更会导致关联失效
- ❌ 拼写错误会导致数据不一致
- ❌ 缺少外键约束

### 新方案（枚举关联）
- ✅ `oauth2_clients.provider_type` → `providers.provider_type`
- ✅ 通过整数枚举匹配，不会出错
- ✅ Provider name 可以自由修改，不影响关联
- ✅ 数据类型安全
- ✅ 可以添加外键约束（可选）

## 📝 向前兼容

- ✅ 保留 `oauth2_clients.provider_name` 字段
- ✅ 保留 `providers.name` 字段
- ✅ 前端代码可以继续使用 `provider_name` 字段（用于显示）
- ✅ 新代码优先使用 `provider_type` 字段（用于关联）

## 🔮 未来改进（可选）

1. **添加外键约束**
   ```sql
   ALTER TABLE oauth2_clients
   ADD CONSTRAINT fk_oauth2_clients_provider_type
   FOREIGN KEY (provider_type)
   REFERENCES providers(provider_type);
   ```

2. **废弃 provider_name 字段**
   - 在未来版本中，将 `provider_name` 标记为弃用
   - 最终从数据库中移除

3. **优化查询性能**
   - 使用 `provider_type` 索引进行查询
   - 减少字符串比较操作

## ✅ 验收标准

1. ✅ 所有 Provider 记录都有 `provider_type` 字段
2. ✅ 所有 OAuth2Client 记录都有 `provider_type` 字段
3. ✅ `provider_type` 字段与 Provider 表中的值匹配
4. ✅ 同步脚本使用 `provider_type` 进行关联
5. ✅ 前端可以正确显示 Provider 关联信息
6. ✅ 数据库完整性检查通过

## 📞 支持

如果在实施过程中遇到问题，请：
1. 检查后端日志文件
2. 验证数据库迁移是否成功
3. 确保所有模型字段都正确同步
