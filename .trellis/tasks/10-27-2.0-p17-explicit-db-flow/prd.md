# 打磨 P2b：GetDB() 全局读取消除 —— main.go 显式数据流

## Goal

main.go 9 处 database.GetDB() 隐式全局读取改为显式传参；/ready 探针改走 RouterDeps.DB

## Requirements

- TBD

## Acceptance Criteria

- [x] Initialize 返回 (*gorm.DB, error)；main.go 8 处 / migrate 4 处 / router /ready 1 处显式传参
- [x] RouterDeps + DB 字段；组合根接线；router.go database import 移除
- [x] 生产代码 database.GetDB() 调用归零（仅测试夹具保留全局替换）
- [x] 22 包 -race 全绿；提交 62920c3 已推送

## 验收记录（2026-10-27）

Redis ctx hack（GetDB().Statement.Context 抽取）改 context.Background()，
语义等价（原代码本就是为了拿一个非 nil ctx）。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
