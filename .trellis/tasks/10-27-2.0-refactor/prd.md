# FusionMail 2.0 重构总纲

## Goal

FusionMail 2.0 重构的父任务：把项目从「能跑但分层名存实亡、无反馈回路、三个地基错误」的状态，收敛为「边界由编译期强制、质量门禁可信、schema/密钥有单一真相」的可长期维护形态。

## 背景（2026-10-27 三份架构体检结论）

- 后端 291 文件 / 87,820 行；前端 202 文件 / 41,294 行
- 67% 的提交（236/354）集中在 2025-10 ~ 2025-11 两个月，之后全是打补丁
- Trellis 账本失真：20 个活跃任务仅 1 个归档，6 个 in_progress 实际已完成，8 个 planning 是 19 行 TBD 空壳
- .trellis/spec/ 只有 frontend，后端 88K 行零规范

## 三个错误的地基

1. schema 无单一真相：39 个 SQL 迁移从未被执行，线上 schema 由 AutoMigrate 推导；fly.toml release_command 每次部署跑 AutoMigrate+seed，绕过 ENABLE_AUTO_MIGRATE
2. 测试与 CI 无可信度：集成测试�� SQLite 而生产跑 PostgreSQL；CI 零 test/vet/lint
3. 加密密钥生命周期：密钥静默 pad/truncate、加密失败明文落库、无 key-id 无轮换路径

## 核心命题

2.0 不是「重写代码」，是「建立反馈回路 + 把已经做对的地方变成编译期强制」。不做 big-bang 重写（在 2-3% 覆盖率下重写 13 万行 = 蒙眼重写）。

## 阶段

- P0 环境与基线取证
- P1 CI 质量门禁
- P2 P0 缺陷修复
- P3 死代码清理
- P4 schema 单一真相
- P5 加密密钥生命周期
- P6 分层边界强制
- P7 统一响应与错误体系
- P8 消除全局状态与显式 DI
- P9 上帝对象绞杀
- P10 业务逻辑重构与功能缺陷
- P11 AI 接入点需求分析

## 验收标准

- [ ] go build / go vet / go test ./... 全绿且接入 CI 阻断
- [ ] 前端 tsc / vitest / eslint 接入 CI 阻断
- [ ] schema 由版本化迁移管理，AutoMigrate 仅限测试
- [ ] 加密方案带 key-id，失败 fail-closed
- [ ] model 不含 json tag，响应体强制走 DTO
- [ ] 单一响应层 + 单一错误体系
- [ ] 无全局可变 DB/Redis 单例
- [ ] 4 个上帝对象按用例拆分完成
- [ ] 死代码清理 ≥ 6000 行
- [ ] AI 接入点需求分析文档产出
- [ ] Trellis 任务账本与实际状态一致

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
