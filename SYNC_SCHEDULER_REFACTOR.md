# 同步调度器重构说明

## 📋 问题描述

**原问题**：所有邮箱账号的最后同步时间都是同一时间，即使设置了不同的同步频率。

**根本原因**：
- 旧的调度器每 5 分钟触发一次 `SyncAllAccounts`
- `SyncAllAccounts` 会同时同步所有启用的账户
- 完全忽略了每个账户的 `sync_interval` 配置

## 🔧 重构方案

采用**方案 2：统一调度器 + 时间判断**

### 核心思路

1. **统一调度器**：每分钟检查一次所有账户
2. **个性化判断**：根据每个账户的 `last_sync_at` 和 `sync_interval` 判断是否需要同步
3. **独立触发**：只同步到达同步时间的账户

### 实现逻辑

```
每分钟检查一次：
├─ 获取所有启用同步的账户
├─ 遍历每个账户
│  ├─ 判断是否首次同步（last_sync_at == nil）
│  │  └─ 是 → 立即同步
│  └─ 判断是否到达同步时间
│     ├─ 计算：next_sync_time = last_sync_at + sync_interval
│     └─ 比较：now >= next_sync_time
│        └─ 是 → 触发同步
└─ 记录本次触发的同步数量
```

## 📝 代码修改

### 1. StartScheduler（调度器入口）

**修改前**：
```go
ticker := time.NewTicker(5 * time.Minute) // 固定 5 分钟
for {
    select {
    case <-ticker.C:
        s.SyncAllAccounts(ctx) // 同步所有账户
    }
}
```

**修改后**：
```go
ticker := time.NewTicker(1 * time.Minute) // 每分钟检查
for {
    select {
    case <-ticker.C:
        s.checkAndSyncAccounts(ctx) // 检查并同步需要的账户
    }
}
```

### 2. checkAndSyncAccounts（新增方法）

检查所有账户，只同步到达同步时间的账户：

```go
func (s *syncService) checkAndSyncAccounts(ctx context.Context) {
    accounts, _ := s.accountRepo.ListSyncEnabled(ctx)
    now := time.Now()
    
    for _, account := range accounts {
        if s.shouldSync(account, now) {
            // 异步同步该账户
            go s.SyncAccount(ctx, account.UID)
        }
    }
}
```

### 3. shouldSync（新增方法）

判断单个账户是否需要同步：

```go
func (s *syncService) shouldSync(account *model.Account, now time.Time) bool {
    // 首次同步
    if account.LastSyncAt == nil {
        return true
    }
    
    // 计算下次同步时间
    nextSyncTime := account.LastSyncAt.Add(
        time.Duration(account.SyncInterval) * time.Minute
    )
    
    // 判断是否到达同步时间
    return now.After(nextSyncTime) || now.Equal(nextSyncTime)
}
```

### 4. SyncAllAccounts（功能调整）

保留该方法，但调整为**手动触发全量同步**，不再用于定时调度：

```go
func (s *syncService) SyncAllAccounts(ctx context.Context) error {
    // 立即同步所有账户，不考虑同步间隔
    // 主要用于手动触发或 API 调用
}
```

## 🎯 效果对比

### 修改前（问题场景）

```
时间轴：
00:00 -> 账户A、B、C 同时同步 ✓
00:05 -> 账户A、B、C 同时同步 ✓
00:10 -> 账户A、B、C 同时同步 ✓

结果：所有账户的 last_sync_at 都相同
```

### 修改后（期望场景）

```
账户配置：
- 账户A: sync_interval = 5 分钟
- 账户B: sync_interval = 15 分钟
- 账户C: sync_interval = 30 分钟

时间轴：
00:00 -> 账户A、B、C 首次同步 ✓
00:05 -> 账户A 同步 ✓
00:10 -> 账户A 同步 ✓
00:15 -> 账户A、B 同步 ✓
00:20 -> 账户A 同步 ✓
00:25 -> 账户A 同步 ✓
00:30 -> 账户A、B、C 同步 ✓

结果：每个账户按照自己的 sync_interval 独立同步
```

## 📊 性能优化

### 资源占用

- **检查频率**：每分钟 1 次（固定）
- **Goroutine 数量**：仅为需要同步的账户创建
- **数据库查询**：每分钟 1 次（查询所有启用同步的账户）

### 调度精度

- **最小间隔**：1 分钟
- **精度误差**：±1 分钟
- **适用场景**：邮件同步（对精度要求不高）

### 扩展性

- **账户数量**：支持大量账户（数据库查询 + 内存判断）
- **并发控制**：每个账户独立 goroutine，自然并发
- **资源限制**：可添加并发数限制（如最多同时同步 10 个账户）

## 🔍 日志输出

### 调度器启动

```
[Scheduler] Sync scheduler started (checking every 1 minute)
```

### 检查周期

```
[Scheduler] Account abc123 ready for sync (last: 5m ago, interval: 5 min)
[Scheduler] Triggering sync for account abc123 (email: user@example.com, interval: 5 min)
[Scheduler] Triggered sync for 2/5 accounts
```

### 首次同步

```
[Scheduler] Account xyz789 needs first sync
[Scheduler] Triggering sync for account xyz789 (email: new@example.com, interval: 15 min)
```

## ✅ 验证方法

### 1. 创建测试账户

```sql
-- 账户A：5 分钟同步一次
INSERT INTO accounts (uid, email, sync_interval, sync_enabled, status) 
VALUES ('test-a', 'test-a@example.com', 5, true, 'active');

-- 账户B：15 分钟同步一次
INSERT INTO accounts (uid, email, sync_interval, sync_enabled, status) 
VALUES ('test-b', 'test-b@example.com', 15, true, 'active');

-- 账户C：30 分钟同步一次
INSERT INTO accounts (uid, email, sync_interval, sync_enabled, status) 
VALUES ('test-c', 'test-c@example.com', 30, true, 'active');
```

### 2. 观察日志

启动服务后，观察日志输出：

```bash
# 查看调度器日志
docker logs -f fusionmail-backend | grep "\[Scheduler\]"
```

### 3. 检查同步时间

```sql
-- 查看各账户的最后同步时间
SELECT 
    email,
    sync_interval,
    last_sync_at,
    EXTRACT(EPOCH FROM (NOW() - last_sync_at))/60 AS minutes_since_sync
FROM accounts
WHERE sync_enabled = true
ORDER BY last_sync_at DESC;
```

### 4. 预期结果

- 账户A 每 5 分钟同步一次
- 账户B 每 15 分钟同步一次
- 账户C 每 30 分钟同步一次
- 各账户的 `last_sync_at` 时间不同

## 🚀 部署说明

### 重启服务

```bash
# 重新编译后端
cd backend
go build -o bin/server cmd/server/main.go

# 重启服务
docker-compose restart backend
```

### 无需数据库迁移

此次修改仅涉及代码逻辑，不需要修改数据库结构。

### 兼容性

- ✅ 向后兼容：现有账户配置无需修改
- ✅ API 兼容：所有 API 接口保持不变
- ✅ 手动同步：手动触发同步功能正常工作

## 📌 注意事项

### 1. 同步间隔建议

- **最小间隔**：建议不低于 5 分钟（避免频繁同步）
- **最大间隔**：建议不超过 60 分钟（保证邮件及时性）
- **默认值**：5 分钟（适合大多数场景）

### 2. 首次同步

- 新添加的账户会在下一个检查周期（最多 1 分钟）内触发首次同步
- 首次同步不受 `sync_interval` 限制

### 3. 手动同步

- 手动触发的同步会立即执行，不受调度器影响
- 手动同步会更新 `last_sync_at`，影响下次自动同步时间

### 4. 禁用账户

- `status != 'active'` 的账户不会被同步
- `sync_enabled = false` 的账户不会被同步

## 🎉 总结

### 问题解决

✅ **已解决**：每个账户现在按照自己的 `sync_interval` 独立同步

### 优势

- ✅ **个性化配置**：支持每个账户独立的同步频率
- ✅ **资源优化**：只同步需要同步的账户
- ✅ **易于维护**：统一的调度器，简单的判断逻辑
- ✅ **可扩展性**：支持大量账户，易于添加并发控制

### 后续优化

- 可添加并发数限制（如最多同时同步 N 个账户）
- 可添加优先级队列（重要账户优先同步）
- 可添加智能调度（根据邮件活跃度动态调整间隔）

---

**修改时间**：2025-11-05  
**修改人**：Kiro AI Assistant  
**影响范围**：`backend/internal/service/sync_service.go`
