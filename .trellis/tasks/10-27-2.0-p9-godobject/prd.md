# P9 上帝对象绞杀

## Goal

按 `.trellis/spec/backend/service-layer.md` 的 7 步绞杀模式拆分全部上帝对象。
每步：按用例分组 → 独立构造函数 → 窄接口 → 组合器纯转发 → 测试只构造子类型
→ 跨服务辅助函数提升为包级函数（禁止 service→service 依赖）。

## Acceptance Criteria 与完成状态

- [x] emailService（原 21 字段 / 1,100+ 行）→ EmailCrud/EmailQuery/EmailReader
      等用例类型 + 纯转发组合器
- [x] accountService → AccountCrud/AccountQuery/AccountLifecycle
- [x] WebAPIProviderService → provider/child-account/adapter 三用例类型
- [x] syncService 第一阶段 → SyncProcessor（落库叶子）+ SyncErrorPolicy
- [x] syncService 收尾（`a612530`）→ SyncScheduler(7 字段) + SyncExecutor(14)
      + SyncProgressRegistry(4) + compositeSyncService（6 方法全单行转发）；
      原 19 字段中 ruleService/spamDetector/deletedKeyRepo 彻底消失
- [x] 每步全量 `go test ./... -race` 22 包绿；swagger.json 逐字节不变
- [x] handler 层零改动（依赖 6 方法 SyncService 接口不变）

## 模式沉淀

7 步模式已写入 `.trellis/spec/backend/service-layer.md`，后续新服务
按此评审。核心不变量：

- 组合器只做纯转发（一行逻辑都不许有）
- 调度/执行解耦走窄接口（accountSyncRunner），不依赖具体类型
- 共享可变状态收敛进小类型（如 SyncProgressRegistry），原子性方法
  （EvictStaleLock/TryRegister）保留原"持锁跨检查"语义，避免 TOCTOU
