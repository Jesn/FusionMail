# 打磨 P2c：附件元数据落库

## Goal

附件元数据随同步落库（StorageType=external，本地不存内容），修复收件附件列表恒空

## Requirements

- TBD

## Acceptance Criteria

- [x] adapter.Attachment + ExternalRef；cloudmail AttList 映射（此前只取 count）
- [x] createEmailFromAdapter 级联落库；更新路径 ReplaceAttachments 单事务删旧插新
- [x] 存储语义：StorageType=external / StoragePath=""（内容在服务商侧，本地不存）
- [x] TestAttachmentMetadataPersistence ✅；22 包 -race 全绿
- [x] 提交 05a1103 已推送

## 验收记录（2026-10-27）

提交 `05a1103`。下载（按需拉取）为适配器层独立课题，ExternalRef 已预留。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
