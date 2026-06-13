# 文档索引

## 00-09 · 00-project-meta（项目元信息）

| 文件 | 说明 |
|------|------|
| [pr-history.md](00-project-meta/pr-history.md) | 向上游提交的 PR 历史记录 |

## 10-19 · 01-dev-read（源码阅读分析）

| 序号 | 文件 | 说明 |
|------|------|------|
| 01 | [01-项目架构总览与二次开发分析报告.md](01-dev-read/01-项目架构总览与二次开发分析报告.md) | 项目整体架构、模块划分、二次开发切入点 |
| 02 | [02-Loop模式深度分析.md](01-dev-read/02-Loop模式深度分析.md) | `/loop` 指令的实现机制与生命周期 |
| 03 | [03-TUI功能深度分析报告.md](01-dev-read/03-TUI功能深度分析报告.md) | TUI 的渲染、事件、交互完整分析 |
| 04 | [04-Desktop会话持久化分析.md](01-dev-read/04-Desktop会话持久化分析.md) | Desktop 端会话存储、恢复、autosave 机制 |
| 05 | [05-loop-lifecycle.md](01-dev-read/05-loop-lifecycle.md) | Loop 模式生命周期追踪记录 |
| 06 | [06-agent-capability-assessment.md](01-dev-read/06-agent-capability-assessment.md) | Agent 能力评估报告 |
| 07 | [07-vs-claude-code-codex-analysis.md](01-dev-read/07-vs-claude-code-codex-analysis.md) | 与 Claude Code / Codex 的对比分析 |
| 08 | [08-three-modes-analysis.md](01-dev-read/08-three-modes-analysis.md) | Normal / Plan / Goal 三种模式的对比分析 |

## 20-29 · 02-dev-faq（知识问答与诊断）

| 文件 | 说明 |
|------|------|
| [branch-switching-mechanism.md](02-dev-faq/branch-switching-mechanism.md) | 分叉会话的切换机制、cpBound 判定逻辑 |
| [desktop-ui-panels-questions.md](02-dev-faq/desktop-ui-panels-questions.md) | Desktop UI 面板相关 FAQ（项目树、变更、依赖文件、Goal 模式等） |
| [goal-mode-comprehensive.md](02-dev-faq/goal-mode-comprehensive.md) | Goal 模式完整分析（机制、风险、反面案例、单双模型对比） |
| [planner-role-tool-limits.md](02-dev-faq/planner-role-tool-limits.md) | Planner 角色与工具限制说明 |

### 诊断记录

| 文件 | 说明 |
|------|------|
| [diagnosis/20260610-diagnosis-reasonix-high-cpu.md](02-dev-faq/diagnosis/20260610-diagnosis-reasonix-high-cpu.md) | 2026.06.10 Reasonix 高 CPU 占用的诊断 |

## 30-39 · 03-feature-action（功能设计与实施）

| 文件 | 说明 |
|------|------|
| [chat-file-link-navigation.md](03-feature-action/20260613-chat-file-link-navigation.md) | 对话文件链接导航 — 设计草案 → 实施实录 |
| [chat-file-link-navigation-regression.md](03-feature-action/20260613-chat-file-link-navigation-regression.md) | 文件链接导航 — 回归影响分析 |
| [git-branch-display-bug.md](03-feature-action/20260613-git-branch-display-bug.md) | 顶部 git 分支显示 bug 分析 |
| [hope-command-design.md](03-feature-action/20260613-hope-command-design.md) | Hope 指令功能设计草案 |
| [message-send-queue-design.md](03-feature-action/20260613-message-send-queue-design.md) | 消息发送队列设计草案 |

## 40-49 · 04-capabilities（基础能力清单）

> 待补充：`ListWorkspaceFiles()`、`SwitchBranch()`、Markdown `<a>` 拦截等可复用能力。
