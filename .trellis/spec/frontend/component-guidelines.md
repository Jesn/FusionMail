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

## File Size 治理原则（2026-10-27 修订：内聚优先，行数是信号不是目标）

**原则：规范书写、合理模块化、美观整洁 —— 而不是机械压行数。**

- 行数超 500 先问"为什么大"：多职责混杂 → 按自然边界拆；单一职责写得多 → 保留；
  重复代码撑大 → 提炼共用方法/组件
- 拆分只在有自然边界处动手：按服务类型/步骤/接口/主题聚类；拆出的块必须能独立命名
  （说得出它是什么，而不只是"第 300-500 行"）
- 两个及以上对话框重复的字段块 → 共享字段组件（参考 `components/spam/RuleFormFields.tsx`）
- 纯函数/常量随唯一消费组件走；多处消费的提为共享模块（参考 `components/oauth2/oauth2ClientMeta.ts`）
- 搬移类重构逐行对照原文，严禁凭记忆重写选项列表/字段名（ruleMeta 教训）
- 保留不动的记录：`AccountsPage`(989，hook 再抽会破坏纯展示约束)、
  `AccountTableSection`(551，48 props 再拆纯增管道)、`useSettings`(563，相关 hooks 集合)、
  `types/api.generated.ts`（生成物）
