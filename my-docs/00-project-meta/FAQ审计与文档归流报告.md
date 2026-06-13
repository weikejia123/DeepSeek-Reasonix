# FAQ 审计与文档归流报告

<!--
版本: v1.0
创建: 2026-06-14 01:00:00
更新: 2026-06-14 01:00:00
-->

> 基于 5 篇 FAQ + 1 篇诊断 vs 52 篇现有文档的深度交叉分析。

---

## 一、FAQ 痛点 → 知识盲区映射

### 1.1 Goal模式完整分析（290 行，最深 FAQ）

| FAQ 涉及的痛点 | 背后的知识盲区 | 严重度 |
|---------------|---------------|:---:|
| planner-goal 指令冲突导致分析瘫痪 | **Coordinator 双模型架构完全未文档化** | 🔴 |
| goal 块注入机制（Compose 无条件注入） | Compose 动态注入虽有文档但缺 goal 特异性 | 🟡 |
| 单模型 vs 双模型行为差异表 | **无"模式选择决策树"文档** | 🔴 |
| 什么提示词适合/不适合 goal | **Goal 最佳实践无独立文档** | 🟡 |
| maxGoalAutoTurns / blocked 检测 / PlannerMaxSteps | 保护机制分散在 controller 文档中 | 🟢 |
| 反面案例分析（6 维度混杂 prompt） | **无"Goal 反模式"文档** | 🟡 |

### 1.2 Desktop-UI 面板 FAQ（290 行）

| FAQ 涉及的痛点 | 背后的知识盲区 | 严重度 |
|---------------|---------------|:---:|
| Q1-Q3: Trash/Changes/Referenced Files 面板 | **Desktop UI 数据模型无文档** | 🔴 |
| Q4: Goal 使用方法（128 行！） | **完整的 Goal 使用指南埋在 FAQ 里** | 🔴 |
| Q5: "effort max 90012 ms" 含义 | Subagent 模型/effort 选择无独立文档 | 🟡 |
| Q6: 双模型 pro+flash 协作流程 | 同 Coordinator 架构缺失 | 🔴 |
| Q7: 分叉深度无限制 | 分支文档基本够用 | 🟢 |

### 1.3 分支切换机制（123 行）

| FAQ 涉及的痛点 | 背后的知识盲区 | 严重度 |
|---------------|---------------|:---:|
| 分支切换是"完全替换"还是"叠加" | 会话模型已有文档基本覆盖 | 🟢 |
| cpBound 生命周期（何时建立/清空/删除） | **cpBound 机制无独立文档** | 🟡 |
| fork 按钮禁用条件链 | 前端逻辑基本够用 | 🟢 |

### 1.4 Planner 角色与工具限制（62 行）

| FAQ 涉及的痛点 | 背后的知识盲区 | 严重度 |
|---------------|---------------|:---:|
| planner 不能执行 `git status` | 09-工具系统文档已覆盖 FilterRegistry | 🟢 |
| bash.ReadOnly() 设计理由 | 可补充到工具系统文档 | 🟢 |

### 1.5 诊断：高 CPU 占用

| 痛点 | 知识盲区 |
|------|---------|
| 12 小时单核满载排查 | **无"性能排错指南"** |

---

## 二、FAQ 内容归流计划

### 2.1 Goal模式完整分析 → 归流到以下文档

| FAQ 内容 | 归流目标 | 操作 |
|---------|---------|------|
| Goal 机制全链路（Compose→runGoalLoop→advanceGoalAfterTurn） | **新建** `06-architecture/Goal模式数据流.md` | FAQ 内容提炼为 L2 数据流文档 |
| planner-goal 指令冲突根因 | **新建** `01-dev-read/29-Coordinator双模型架构.md` | FAQ 内容升级为 L3 深度文档 |
| 单/双模型差异、模式选择 | 追加到 `05-overview/核心概念词汇表.md` Goal 条目 | 补充最佳实践 |

### 2.2 Desktop-UI 面板 FAQ → 归流

| FAQ 内容 | 归流目标 | 操作 |
|---------|---------|------|
| Q1-Q3: Trash/Changes/Referenced 数据模型 | **新建** `01-dev-read/30-Desktop数据模型.md` | FAQ 内容升级为 L3 文档 |
| Q4: Goal 使用方法 | 合并到 `06-architecture/Goal模式数据流.md` | 使用指南+实现原理合为一体 |
| Q5: Subagent 模型/effort | 追加到 `01-dev-read/18-子代理系统.md` | 补充模型选择章节 |
| Q6: 双模型协作 | 同上 → `29-Coordinator双模型架构.md` | |

### 2.3 分支切换机制 → 归流

| FAQ 内容 | 归流目标 | 操作 |
|---------|---------|------|
| cpBound 生命周期 | 追加到 `01-dev-read/23-检查点与回退.md` | 补充 cpBound 章节 |
| 分支切换独立性 | 已有文档覆盖，FAQ 引用即可 | — |

### 2.4 Planner 工具限制 → 归流

| FAQ 内容 | 归流目标 | 操作 |
|---------|---------|------|
| bash.ReadOnly() 设计理由 | 补充到 `01-dev-read/09-工具系统.md` 错误处理/权限节 | — |

### 2.5 诊断 → 新建排错指南

| 诊断内容 | 归流目标 |
|---------|---------|
| 高 CPU 排查方法 | 新建 `07-extension-guide/排错指南.md` |

---

## 三、文档目录树与补全计划

### 3.1 新建文档（7 篇）

| # | 路径 | 标题 | 优先级 | 来源 |
|---|------|------|:---:|------|
| 29 | `01-dev-read/29-Coordinator双模型架构.md` | Coordinator — planner+executor 双模型协作 | 🔴 P0 | Goal FAQ + Desktop FAQ Q6 |
| 30 | `01-dev-read/30-Desktop数据模型.md` | Desktop UI — Trash/Changes/Referenced 数据模型 | 🔴 P0 | Desktop FAQ Q1-Q3 |
| L2 | `06-architecture/Goal模式数据流.md` | Goal 模式 — 全链路数据流与最佳实践 | 🔴 P0 | Goal FAQ 全部 |
| L4 | `07-extension-guide/排错指南.md` | 常见问题排错 — 性能/沙箱/模型/Goal 故障 | 🟡 P1 | 诊断 + Goal FAQ |
| L2 | `06-architecture/组件交互矩阵.md` | 核心组件交互 — Controller/Agent/Coordinator/Session 依赖 | 🟡 P1 | 多篇 FAQ 隐含 |
| — | `05-overview/模式选择指南.md` | Normal/Plan/Goal 三种模式的选择决策树 | 🟢 P2 | Goal FAQ |
| — | `05-overview/Goal反模式.md` | Goal 反面案例分析 — 什么不该用 Goal | 🟢 P2 | Goal FAQ 第六章 |

### 3.2 需补充的现有文档（3 处）

| 文件 | 补充内容 | 来源 |
|------|---------|------|
| `01-dev-read/23-检查点与回退.md` | cpBound 生命周期（建立/清空/回退/恢复） | 分支 FAQ |
| `01-dev-read/18-子代理系统.md` | Subagent 模型/effort 选择优先级 | Desktop FAQ Q5 |
| `01-dev-read/09-工具系统.md` | bash.ReadOnly() 设计理由 + 权限节 | Planner FAQ |

### 3.3 可归档的 FAQ（FAQ 内容已沉淀后可标记为"参考"）

| FAQ 文件 | 状态 | 沉淀后处理 |
|---------|------|----------|
| `Goal模式完整分析.md` | 内容可完全归流 | 保留为 FAQ，精简为引用链接 |
| `Desktop-UI面板FAQ.md` Q4 | Goal 用法已归流 | 精简，保留其他问题 |
| `分支切换机制.md` cpBound 部分 | 可归流 | 保留其他问题 |
| `Planner角色与工具限制.md` | 内容最小 | 保留原文 |

---

## 四、实施优先级

```
Phase 1 (P0 · 立即):
  29-Coordinator双模型架构.md    ← 最大知识盲区
  30-Desktop数据模型.md          ← FAQ 高频问题
  Goal模式数据流.md              ← 合并 Goal FAQ 全部内容

Phase 2 (P1 · 本周):
  排错指南.md                    ← 诊断能力
  组件交互矩阵.md                ← 串起零散知识

Phase 3 (P2 · 后续):
  模式选择指南 + Goal反模式      ← 最佳实践
  3 处现有文档补充               ← 轻量改动
```
