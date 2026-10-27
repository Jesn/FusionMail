# Component Guidelines

> How components are built in this project.

---

## Overview

<!--
Document your project's component conventions here.

Questions to answer:
- What component patterns do you use?
- How are props defined?
- How do you handle composition?
- What accessibility standards apply?
-->

(To be filled by the team)

---

## Component Structure

<!-- Standard structure of a component file -->

(To be filled by the team)

---

## Props Conventions

<!-- How props should be defined and typed -->

(To be filled by the team)

---

## Styling Patterns

<!-- How styles are applied (CSS modules, styled-components, Tailwind, etc.) -->

(To be filled by the team)

---

## Accessibility

<!-- A11y requirements and patterns -->

(To be filled by the team)

---

## Common Mistakes

<!-- Component-related mistakes your team has made -->

(To be filled by the team)

---

## File Size Budget（P12 大文件治理，2026-10-27 起生效）

**手写文件目标 ≤500 行**；页面组件 ≤700 行（状态+处理函数+对话框接线的合理上限）。

- 单组件超过 ~300 行就该想拆：纯展示 JSX 块 → 显式 props 子组件（参考
  `components/account/GroupSidebar.tsx` / `components/spam/RuleTable.tsx`）
- 状态、处理函数、对话框接线留在页面；子组件不 fetch、不碰 store
- 两个及以上对话框重复的字段块 → 共享字段组件（参考 `components/spam/RuleFormFields.tsx`）
- 搬移类重构逐行对照原文：元数据/选项列表严禁凭记忆重写（ruleMeta 首版教训）
- 豁免：生成文件（`types/api.generated.ts`）不适用行数预算
