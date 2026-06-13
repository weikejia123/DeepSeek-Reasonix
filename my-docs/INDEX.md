# 文档索引

## 00-09 · 00-project-meta（项目元信息）

| 文件 | 说明 |
|------|------|
| [pr-history.md](00-project-meta/pr-history.md) | 向上游提交的 PR 历史记录 |
| [gap-analysis-plan.md](00-project-meta/gap-analysis-plan.md) | 文档缺口分析报告 & L1-L4 补全计划 |

## 05-09 · 05-overview（L1 快速上手）

| 文件 | 说明 |
|------|------|
| [what-is-reasonix.md](05-overview/what-is-reasonix.md) | Reasonix 是什么 — 项目定位、设计哲学、三前端架构 |
| [quick-start.md](05-overview/quick-start.md) | 快速上手 — 安装→配置→第一条消息→核心交互 |
| [core-concepts.md](05-overview/core-concepts.md) | 核心概念词汇表 — Agent/Turn/Tool/Skill/MCP 等 20 个概念 |
| [project-structure.md](05-overview/project-structure.md) | 项目结构导览 — 46 个 internal 包分类地图 |

## 10-19 · 01-dev-read（L3 源码阅读分析）

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
| 09 | [09-tool-system.md](01-dev-read/09-tool-system.md) | Tool System — 工具注册、分发、执行 + 全量工具目录 |
| 10 | [10-agent-core-loop.md](01-dev-read/10-agent-core-loop.md) | Agent Core Loop — Run 完整循环 + 六道防护机制 |
| 11 | [11-skill-system.md](01-dev-read/11-skill-system.md) | Skill System — 发现、加载、inline/subagent 执行 |
| 12 | [12-mcp-system.md](01-dev-read/12-mcp-system.md) | MCP System — 插件连接、传输与工具代理 |
| 13 | [13-context-compaction.md](01-dev-read/13-context-compaction.md) | Context & Compaction — 上下文管理与三阈值压缩 |
| 14 | [14-hook-system.md](01-dev-read/14-hook-system.md) | Hook System — 10 事件点 + 阻塞/非阻塞扩展 |
| 15 | [15-boot-sequence.md](01-dev-read/15-boot-sequence.md) | Boot Sequence — Build() 从零到就绪的完整启动流程 |
| 16 | [16-session-persistence.md](01-dev-read/16-session-persistence.md) | Session & Persistence — 会话生命周期、JSONL持久化、检查点 |
| 17 | [17-provider-system.md](01-dev-read/17-provider-system.md) | Provider System — LLM调用、流式、多提供商、前缀缓存 |
| 18 | [18-subagent-task.md](01-dev-read/18-subagent-task.md) | Subagent/Task — 子代理隔离、FilterRegistry、continuation |
| 19 | [19-memory-system.md](01-dev-read/19-memory-system.md) | Memory System — 层级加载、四种类型、REASONIX.md |
| 20 | [20-plan-mode.md](01-dev-read/20-plan-mode.md) | Plan Mode — 双轮规划→批准→执行架构 |
| 21 | [21-approval-gate.md](01-dev-read/21-approval-gate.md) | Approval & Gate — 三层权限、YOLO、SessionGrant |
| 22 | [22-evidence-todo.md](01-dev-read/22-evidence-todo.md) | Evidence & Todo — complete_step验证、task list状态机 |
| 23 | [23-checkpoint-rewind.md](01-dev-read/23-checkpoint-rewind.md) | Checkpoint & Rewind — 每turn快照、code/conversation回退 |
| 24 | [24-auto-plan.md](01-dev-read/24-auto-plan.md) | Auto-Plan — 分类器评分、自动触发plan模式 |
| 25 | [25-event-system.md](01-dev-read/25-event-system.md) | Event System — 19种事件、Sink接口、多Tab路由 |
| 26 | [26-config-system.md](01-dev-read/26-config-system.md) | Config System — reasonix.toml结构、加载优先级 |
| 27 | [27-frontend-architecture.md](01-dev-read/27-frontend-architecture.md) | Frontend Architecture — React状态管理、Bridge IPC |
| 28 | [28-sandbox.md](01-dev-read/28-sandbox.md) | Sandbox — macOS Seatbelt、bash安全隔离 |

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

## 40-49 · 04-capabilities（基础能力清单）

| 文件 | 说明 |
|------|------|
| [workspace-file-list.md](04-capabilities/workspace-file-list.md) | `ListWorkspaceFiles()` — 全量工作区文件枚举 |
| [steer-queue.md](04-capabilities/steer-queue.md) | `steerQueue` — Mid-Turn 消息队列机制与功能边界 |
| [slash-commands.md](04-capabilities/slash-commands.md) | 指令系统完整参考 — 全部内置/管理/TUI/自定义指令的解读与场景 |

> 按需补充：`SwitchBranch()`、Markdown `<a>` 拦截等可复用能力。
