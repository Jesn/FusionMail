# 打磨 P2b：GetDB() 全局读取消除 —— main.go 显式数据流

## Goal

main.go 9 处 database.GetDB() 隐式全局读取改为显式传参；/ready 探针改走 RouterDeps.DB

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
