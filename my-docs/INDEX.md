# 文档索引

> **团队必读速查**：[BRANCHES.md](BRANCHES.md) — 本地分支清单与命名约定

## 00-09 · 00-project-meta（项目元信息）

| 文件 | 说明 |
|------|------|
| [BRANCHES.md](../BRANCHES.md) | 根目录速查卡：本地分支清单、命名约定、常用命令 |
| [分支管理说明.md](00-project-meta/分支管理说明.md) | 团队内部分支策略、活跃分支清单、个人备注 |
| [PR历史记录.md](00-project-meta/PR历史记录.md) | 向上游提交的 PR 历史记录 |
| [文档缺口分析与L1-L4补全计划.md](00-project-meta/文档缺口分析与L1-L4补全计划.md) | 46 包 vs 文档交叉比对 + 26 篇新文档规划 |
| [FAQ审计与文档归流报告.md](00-project-meta/FAQ审计与文档归流报告.md) | FAQ 痛点→知识盲区映射 + 内容归流 + 7 篇新文档计划 |
| [上游合并最佳实践与自动化机制.md](00-project-meta/上游合并最佳实践与自动化机制.md) | 二开分支同步上游的流程、冲突处理与 agent 自动化边界 |

## 05-09 · 05-overview（L1 快速上手）

| 文件 | 说明 |
|------|------|
| [Reasonix是什么.md](05-overview/Reasonix是什么.md) | 项目定位、设计哲学、三前端架构 |
| [快速上手.md](05-overview/快速上手.md) | 安装→配置→第一条消息→核心交互 |
| [核心概念词汇表.md](05-overview/核心概念词汇表.md) | Agent/Turn/Tool/Skill/MCP 等 20 个概念 |
| [项目结构导览.md](05-overview/项目结构导览.md) | 46 个 internal 包分类地图 |

## 05-09 · 06-architecture（L2 系统架构）

| 文件 | 说明 |
|------|------|
| [系统架构总图.md](06-architecture/系统架构总图.md) | 完整分层图 + 组件交互矩阵 + 关键路径 |
| [单Turn全链路数据流.md](06-architecture/单Turn全链路数据流.md) | 时序图 + 数据结构变化 + 事件流 |
| [Skill调用全链路.md](06-architecture/Skill调用全链路.md) | inline/subagent 两条路径 |
| [Goal模式数据流.md](06-architecture/Goal模式数据流.md) | Goal 全链路 + 最佳实践 + 双模型警告 ← FAQ 归流 |
| [用户发送消息全生命周期.md](06-architecture/用户发送消息全生命周期.md) | Desktop 前端输入→后端处理→事件回流→渲染完整链路 |

## 05-09 · 07-extension-guide（L4 扩展指南）

| 文件 | 说明 |
|------|------|
| [开发者扩展指南.md](07-extension-guide/开发者扩展指南.md) | 如何新增 Tool / Skill / MCP / Provider / Hook / Command / Frontend 组件 |
| [排错指南.md](07-extension-guide/排错指南.md) | 常见问题排错 — 性能/沙箱/模型/Goal/MCP ← FAQ 归流 |

## 10-19 · 01-dev-read（L3 源码阅读分析）

| 序号 | 文件 | 说明 |
|------|------|------|
| 01 | [01-项目架构总览与二次开发分析报告.md](01-dev-read/01-项目架构总览与二次开发分析报告.md) | 项目整体架构、模块划分、二次开发切入点 |
| 02 | [02-Loop模式深度分析.md](01-dev-read/02-Loop模式深度分析.md) | `/loop` 指令的实现机制与生命周期 |
| 03 | [03-TUI功能深度分析报告.md](01-dev-read/03-TUI功能深度分析报告.md) | TUI 的渲染、事件、交互完整分析 |
| 04 | [04-Desktop会话持久化分析.md](01-dev-read/04-Desktop会话持久化分析.md) | Desktop 端会话存储、恢复、autosave 机制 |
| 05 | [05-Loop生命周期追踪.md](01-dev-read/05-Loop生命周期追踪.md) | Loop 模式生命周期追踪记录 |
| 06 | [06-Agent能力评估报告.md](01-dev-read/06-Agent能力评估报告.md) | Agent 能力评估报告 |
| 07 | [07-与ClaudeCode-Codex对比分析.md](01-dev-read/07-与ClaudeCode-Codex对比分析.md) | 与 Claude Code / Codex 的对比分析 |
| 08 | [08-三种调用模式分析.md](01-dev-read/08-三种调用模式分析.md) | Normal / Plan / Goal 三种模式的对比分析 |
| 09 | [09-工具系统.md](01-dev-read/09-工具系统.md) | Tool System — 工具注册、分发、执行 + 全量工具目录 |
| 10 | [10-Agent核心循环.md](01-dev-read/10-Agent核心循环.md) | Agent Core Loop — Run 完整循环 + 六道防护机制 |
| 11 | [11-Skill系统.md](01-dev-read/11-Skill系统.md) | Skill System — 发现、加载、inline/subagent 执行 |
| 12 | [12-MCP系统.md](01-dev-read/12-MCP系统.md) | MCP System — 插件连接、传输与工具代理 |
| 13 | [13-上下文管理与压缩.md](01-dev-read/13-上下文管理与压缩.md) | Context & Compaction — 上下文管理与三阈值压缩 |
| 14 | [14-Hook系统.md](01-dev-read/14-Hook系统.md) | Hook System — 10 事件点 + 阻塞/非阻塞扩展 |
| 15 | [15-启动流程.md](01-dev-read/15-启动流程.md) | Boot Sequence — Build() 从零到就绪的完整启动流程 |
| 16 | [16-会话持久化.md](01-dev-read/16-会话持久化.md) | Session & Persistence — 会话生命周期、JSONL持久化、检查点 |
| 17 | [17-Provider系统.md](01-dev-read/17-Provider系统.md) | Provider System — LLM调用、流式、多提供商、前缀缓存 |
| 18 | [18-子代理系统.md](01-dev-read/18-子代理系统.md) | Subagent/Task — 子代理隔离、FilterRegistry、continuation |
| 19 | [19-记忆系统.md](01-dev-read/19-记忆系统.md) | Memory System — 层级加载、四种类型、REASONIX.md |
| 20 | [20-Plan模式.md](01-dev-read/20-Plan模式.md) | Plan Mode — 双轮规划→批准→执行架构 |
| 21 | [21-权限门控.md](01-dev-read/21-权限门控.md) | Approval & Gate — 三层权限、YOLO、SessionGrant |
| 22 | [22-证据与任务.md](01-dev-read/22-证据与任务.md) | Evidence & Todo — complete_step验证、task list状态机 |
| 23 | [23-检查点与回退.md](01-dev-read/23-检查点与回退.md) | Checkpoint & Rewind — 每turn快照、code/conversation回退 |
| 24 | [24-自动规划.md](01-dev-read/24-自动规划.md) | Auto-Plan — 分类器评分、自动触发plan模式 |
| 25 | [25-事件系统.md](01-dev-read/25-事件系统.md) | Event System — 19种事件、Sink接口、多Tab路由 |
| 26 | [26-配置系统.md](01-dev-read/26-配置系统.md) | Config System — reasonix.toml结构、加载优先级 |
| 27 | [27-前端架构.md](01-dev-read/27-前端架构.md) | Frontend Architecture — React状态管理、Bridge IPC |
| 28 | [28-沙箱系统.md](01-dev-read/28-沙箱系统.md) | Sandbox — macOS Seatbelt、bash安全隔离 |
| 29 | [29-Coordinator双模型架构.md](01-dev-read/29-Coordinator双模型架构.md) | Coordinator — planner+executor 双模型协作 ← FAQ 归流 |
| 30 | [30-Desktop数据模型.md](01-dev-read/30-Desktop数据模型.md) | Desktop UI — Trash/Changes/Referenced 数据模型 ← FAQ 归流 |

## 20-29 · 02-dev-faq（知识问答与诊断）

| 文件 | 说明 |
|------|------|
| [分支切换机制.md](02-dev-faq/分支切换机制.md) | 分叉会话的切换机制、cpBound 判定逻辑 |
| [Desktop-UI面板FAQ.md](02-dev-faq/Desktop-UI面板FAQ.md) | Desktop UI 面板相关 FAQ（项目树、变更、依赖文件、Goal 模式等） |
| [Goal模式完整分析.md](02-dev-faq/Goal模式完整分析.md) | Goal 模式完整分析（机制、风险、反面案例、单双模型对比） |
| [Planner角色与工具限制.md](02-dev-faq/Planner角色与工具限制.md) | Planner 角色与工具限制说明 |

### 诊断记录

| 文件 | 说明 |
|------|------|
| [diagnosis/20260610-diagnosis-reasonix-high-cpu.md](02-dev-faq/diagnosis/20260610-diagnosis-reasonix-high-cpu.md) | 2026.06.10 Reasonix 高 CPU 占用的诊断 |

## 30-39 · 03-feature-action（功能设计与实施）

> 大功能用子目录管理全流程文档（`hope/`、`next-feature/`），小功能/单文件 bug 分析沿用 `YYYYMMDD-` 日期前缀放在根目录。

### 单文件

| 文件 | 说明 |
|------|------|
| [20260613-对话文件链接导航.md](03-feature-action/20260613-对话文件链接导航.md) | 对话文件链接导航 — 设计草案 → 实施实录 |
| [20260613-文件链接导航回归分析.md](03-feature-action/20260613-文件链接导航回归分析.md) | 文件链接导航 — 回归影响分析 |
| [20260613-Git分支显示Bug分析.md](03-feature-action/20260613-Git分支显示Bug分析.md) | 顶部 git 分支显示 bug 分析 |
| [20260614-file-tree-refresh-bug.md](03-feature-action/20260614-file-tree-refresh-bug.md) | Bug 分析 — Turn 完成后文件树不显示新文件 |

### 子目录 · hope/

| 文档 | 说明 |
|------|------|
| [00-设计草案.md](03-feature-action/hope/00-设计草案.md) | Hope 指令功能设计草案 & 初步可行性评估 |
| [01-可行性分析.md](03-feature-action/hope/01-可行性分析.md) | 基于 24 个源文件逐行追踪的全面诊断，精确到行号级 |

### 子目录 · aloop/

| 文档 | 说明 |
|------|------|
| [01-Desktop-Loop-评估.md](03-feature-action/aloop/01-Desktop-Loop-评估.md) | Desktop Loop 改造评估 — 与 Goal 共存互补的 aloop 方案 |
| [02-Desktop设计.md](03-feature-action/aloop/02-Desktop设计.md) | aloop 桌面端详细设计 — 配置驱动 + Controller + Desktop UI + 兼容策略 |
| [03-MVP文件驱动设计.md](03-feature-action/aloop/03-MVP文件驱动设计.md) | aloop MVP 简化方案 — .aloop/ 文件驱动，3 文件改动，不过度设计 |
| [04-脚本驱动设计.md](03-feature-action/aloop/04-脚本驱动设计.md) | aloop 最终简化版 — .aloop/main.* 脚本驱动，开关式，零配置 |
| [05-实现记录.md](03-feature-action/aloop/05-实现记录.md) | aloop 实现过程记录 — 改动文件、验证状态、用法 |

## 40-49 · 04-capabilities（基础能力清单）

| 文件 | 说明 |
|------|------|
| [工作区文件枚举.md](04-capabilities/工作区文件枚举.md) | `ListWorkspaceFiles()` — 全量工作区文件枚举 |
| [Steer队列机制.md](04-capabilities/Steer队列机制.md) | `steerQueue` — Mid-Turn 消息队列机制与功能边界 |
| [指令系统完整参考.md](04-capabilities/指令系统完整参考.md) | 全部内置/管理/TUI/自定义指令的解读与场景 |

> 按需补充：`SwitchBranch()`、Markdown `<a>` 拦截等可复用能力。
