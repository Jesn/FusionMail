# 2.0 后续 C：swagger 注解质量补全

## Goal

把 handler 层 swagger 注解从"能生成"提升到"生成得准"，解除
openapi-typescript 类型生成覆盖率的天花板（P6 收官时记录的遗留项）。

## 背景

- P6 已落地类型代码生成（`79a7370`，7,850 行生成物，删 9 个手写 interface），
  并靠它抓出 3 个用户可见 bug（已修，见第四轮增量）
- 天花板：20+ 端点响应注解为裸 `dto.Response`（生成 `data: unknown`）、
  大量在用端点未文档化、swag 不产出 required
- 漂移报告：`.trellis/tasks/10-27-2.0-p6-boundary/research/frontend-type-drift-report.md`

## Requirements

- 排查并替换所有 `@Success ... dto.Response` 裸注解为具体响应类型
  （`dto.XxxResponse` 或内联结构）
- 文档化"在用但未注解"的端点（前端实际调用但 swagger.json 缺失的）
- required 字段标注：请求 DTO 的必填字段补 `binding:"required"` 之外
  还要让 swag 产出 required（swag 依据 `binding` 生成，确认版本行为）
- 每批注解完成后：`swag init` 重新生成 → `openapi-typescript` 重新生成 →
  tsc 检查前端消费处，diff 报告新增的类型覆盖
- **契约红线**：注解只描述现有行为，不得顺手改任何响应形状；
  swagger.json 变更必须逐字段核对（历史教训：盲目改 tag 会破坏线上数据）

## Acceptance Criteria

- [ ] swagger.json 中 `data: {}`（unknown）端点数：20+ → 0
- [ ] 前端 `api.generated.ts` 中 unknown 类型字段显著减少（前后数字记录于此）
- [ ] `go build/vet/test -race` 全绿；前端 tsc/vitest/build 全绿
- [ ] 重新生成的前端类型被实际消费（至少接掉 5 处手写 interface/any）

## Notes

- swag 工具：`/home/agentuser/go/bin/swag`
- 生成物：backend/docs/docs.go（豁免大文件治理，生成物）
- 前端生成物：frontend/src/types/api.generated.ts（同豁免）
