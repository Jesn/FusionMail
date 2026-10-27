# P3 死代码清理

## Goal

删除项目中确认不可达的代码，验收线：净删除 ≥ 6,000 行。每处删除必须有
"零调用方"取证（grep 全仓库），禁止凭感觉删。

## Requirements

- 后端：无调用方的类型/构造函数/方法/整文件，删除后全量测试通过
- 前端：从 main.tsx 出发的 import 可达性分析，仅删整文件级死代码
- 删除前先检查（用户要求）：确认不是"未接线的能力"（如 errorHandler 那种
  建了但没接的子系统要区分对待 —— 未接线且无消费方才可删）

## Acceptance Criteria

- [x] 净删除 ≥ 6,000 行（实际 ~6,700 行，见下）
- [x] 每处删除有零引用证据
- [x] go build / vet / test -race 全绿（22 包）
- [x] 前端 tsc / vitest(36) / vite build 全绿

## 完成记录（2026-10-27）

| 提交 | 内容 | 净删除 |
|------|------|--------|
| `b1cfc90` | spam 包 cache_manager(721) + fallback_manager(401) + 关联死字段 | ~1,436 |
| `2117c0a` | spam 包未接线错误处理层 error_handler(420) + safety(97) + 2 个死构造函数 | ~572 |
| `1edd9af` | 前端 24 个不可达文件（19 零引用 + 5 死代码簇） | ~4,693 |

关键取证案例：

- `CacheManager`/`FallbackManager` 被 5 个结构体引用但从未实例化 —— 字段恒 nil，
  所有 `if x != nil` 分支是死分支
- `ErrorHandler` 全部导出符号外部引用数为 0；`SpamDetector.errorHandler`
  ��有赋值没有读取
- `GetSystemHealth` 在删掉 cache 分支后变成"恒返回 healthy"的说谎桩 ——
  无调用方，删除而非保留
- 前端 `DashboardPage.tsx` 不在 App.tsx 路由表中，全仓库零引用

## Notes

- 误报排除：`test/setup.ts`（vitest setupFiles）、`vite-env.d.ts`（tsconfig include）
- 存疑不删：`useEmails.ts`/`useSearch.ts`（可达，仅可能存在文件内死导出，
  文件内死代码不在本轮范围）
