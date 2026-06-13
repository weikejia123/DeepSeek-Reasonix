# Reasonix 是什么——项目定位与设计哲学

<!--
版本: v1.0
创建: 2026-06-14 00:05:00
更新: 2026-06-14 00:05:00
-->

## 一句话定义

Reasonix 是一个**多前端、可扩展的 AI 编码智能体**，运行在本地终端、桌面应用或 HTTP 服务器上，通过 LLM 驱动的 Tool-Use 循环自动完成编程任务。

## 解决什么问题

传统的 AI 编码助手通常是 IDE 插件（单前端、受限工具集）。Reasonix 的设计目标是：

1. **自主多步执行** — 不只是回答"怎么做"，而是实际执行：读代码→搜索→编辑文件→运行测试→修复→迭代
2. **前端无关** — 同一套 Agent 核心驱动 TUI（终端）、Desktop（Wails GUI）、HTTP/SSE 三个通道
3. **可扩展** — 通过 Skill、MCP、Hook、Custom Command 四套扩展机制，无需修改核心代码就能接入新能力
4. **开源模型优先** — 深度优化 DeepSeek 的前缀缓存，但也支持 OpenAI/Anthropic 等提供商

## 核心设计哲学

| 原则 | 体现 |
|------|------|
| **Cache-First** | 系统提示词在整个 session 期间不变，利用 DeepSeek 自动前缀缓存降低 token 成本 |
| **Turn-Based** | 用户消息→Agent 循环→Final answer 为一次 Turn；Goal 模式可自动多 Turn 推进 |
| **Tool 即能力** | 所有能力都通过 Tool 接口暴露——内置工具、MCP 工具、Skill 工具无差别对待 |
| **Event-Driven** | 19 种 Event 从 Agent 流向前端，前端通过 Reducer 模式消费 |
| **零侵入扩展** | Skill（Markdown 文件）、MCP（外部进程）、Hook（脚本）、Command（模板）都不需要重新编译 |

## 与同类工具的差异

| 维度 | Reasonix | Claude Code | Cursor |
|------|----------|------------|--------|
| 运行方式 | 本地进程（Go） | 本地进程（Node） | IDE 插件 |
| 前端覆盖 | TUI + Desktop + HTTP | TUI + IDE | IDE 内置 |
| 扩展机制 | Skill+MCP+Hook+Command | MCP+Hook | Rules+MCP |
| 前缀缓存 | DeepSeek 深度优化 | 通用 | N/A |
| 自主循环 | Goal/Loop 双模式 | 无 | Agent 模式 |
| 子代理隔离 | task + subagent session | task | 无 |
| 证据系统 | complete_step+evidence ledger | 无 | 无 |
| 多 Tab | 原生支持（每 Tab 独立 Controller） | 无 | 多文件 |

## 三前端架构

```
         ┌─────────────┐
         │  TUI (终端)  │── bubbletea + lipgloss
         ├─────────────┤
用户 ──▶ │  Desktop     │── Wails (Go + React)
         ├─────────────┤
         │  HTTP/SSE    │── net/http + SSE
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │  Controller  │── submit/compose/approve/plan/goal
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │  Agent.Run() │── stream → executeBatch → loop
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │  Provider    │── DeepSeek / OpenAI / Anthropic
         └─────────────┘
```

## 阅读路线建议

| 层级 | 目录 | 适合谁 | 预计时间 |
|------|------|--------|:---:|
| L1 | `05-overview/` | 刚听说 Reasonix 的开发者 | 15 分钟 |
| L2 | `06-architecture/` | 想理解系统架构的开发者 | 1 小时 |
| L3 | `01-dev-read/` | 要修改/扩展代码的开发者 | 1 天 |
| L4 | `07-extension-guide/` | 要给 Reasonix 加新能力的开发者 | 按需查阅 |
| FAQ | `02-dev-faq/` | 遇到具体问题的开发者 | 按需查阅 |
| 功能 | `03-feature-action/` | 想了解功能演进的开发者 | 按需查阅 |
| 能力 | `04-capabilities/` | 想了解现有可复用轮子的开发者 | 按需查阅 |
