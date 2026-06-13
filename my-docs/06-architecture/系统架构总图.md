# 系统架构总图

<!--
版本: v1.0
创建: 2026-06-14 00:15:00
更新: 2026-06-14 00:15:00
-->

## 完整分层图

```
┌─────────────────────────────────────────────────────────────┐
│                        前端层 (Frontend)                      │
│  ┌──────────┐  ┌──────────────┐  ┌───────────┐              │
│  │ TUI      │  │ Desktop      │  │ HTTP/SSE  │              │
│  │ bubbletea│  │ Wails+React  │  │ net/http  │              │
│  └────┬─────┘  └──────┬───────┘  └─────┬─────┘              │
│       │               │               │                     │
│       └───────────────┼───────────────┘                     │
│                       │ Submit / Cancel / Approve / Steer   │
├───────────────────────┼─────────────────────────────────────┤
│                       ▼                                     │
│                  Controller 层                               │
│  ┌────────────────────────────────────────────────────┐     │
│  │  submit() ─ 路由分发                                │     │
│  │    ├─ /goal /compact /new /clear ...  ─ 命令处理    │     │
│  │    ├─ !shell                            ─ Shell执行 │     │
│  │    ├─ /<skill-name>                     ─ Skill模板  │     │
│  │    └─ 普通输入                           ─ Agent提交 │     │
│  │                                                     │     │
│  │  Compose() ─ 动态注入                                │     │
│  │    ├─ <active-goal>                  (Goal运行中)    │     │
│  │    ├─ PlanModeMarker                 (Plan模式)      │     │
│  │    ├─ <memory-update>                (记忆变更)      │     │
│  │    └─ <background-jobs>              (后台任务)      │     │
│  │                                                     │     │
│  │  runGoalLoopWithRawDisplay() ─ Goal 自动循环        │     │
│  │  beginCheckpoint()           ─ 每Turn快照           │     │
│  │  gate                        ─ 权限门控             │     │
│  │  hooks                       ─ 钩子触发             │     │
│  └────────────────────┬───────────────────────────────┘     │
├───────────────────────┼─────────────────────────────────────┤
│                       ▼                                     │
│                     Agent 层                                 │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Agent.Run(ctx, composedInput)                     │     │
│  │    │                                               │     │
│  │    ├─ TurnStarted event                           │     │
│  │    ├─ session.Add(user msg)                       │     │
│  │    │                                               │     │
│  │    └─ for step 0..maxSteps:                        │     │
│  │         ├─ consumeSteer()    ← mid-turn消息        │     │
│  │         ├─ stream(ctx)       ← ★ 调模型            │     │
│  │         │    └─ Provider.Chat() → 流式chunks       │     │
│  │         ├─ usage event       ← token统计           │     │
│  │         ├─ calls == 0?       ← 最终答案检查        │     │
│  │         │    ├─ finalReadinessCheck                │     │
│  │         │    ├─ steerQueueLen > 0? → continue      │     │
│  │         │    └─ return nil     ← Final Answer!     │     │
│  │         └─ calls > 0:                              │     │
│  │              ├─ executeBatch(calls)                │     │
│  │              │    └─ partitionToolCalls → 并行/串行│     │
│  │              │         └─ executeOne → Gate → Run  │     │
│  │              ├─ session.Add(tool msgs)             │     │
│  │              └─ maybeCompact() → continue          │     │
│  │                                                     │     │
│  │  Events: 19 kinds → sink.Emit()                     │     │
│  └────────────────────┬───────────────────────────────┘     │
├───────────────────────┼─────────────────────────────────────┤
│                       ▼                                     │
│                   工具层 (Tool Layer)                        │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Registry { tools map[string]Tool }                │     │
│  │    ├─ builtin/  (16 工具: bash, edit, grep, ...)   │     │
│  │    ├─ agent/    (task, ask)                        │     │
│  │    ├─ skill/    (run_skill, read_skill, ...)       │     │
│  │    ├─ memory/   (memory, remember, forget)         │     │
│  │    ├─ history/  (history)                          │     │
│  │    ├─ command/  (slash_command)                    │     │
│  │    ├─ builtinmcp/ (codegraph_*, lsp_*)             │     │
│  │    └─ plugin/   (mcp__<server>__<tool>)            │     │
│  │                                                     │     │
│  │  MCP 连接: plugin.Host                              │     │
│  │    ├─ stdio     ─ 子进程 JSON-RPC                  │     │
│  │    ├─ http/sse  ─ Streamable HTTP                  │     │
│  │    └─ 3-tier startup: eager|background|lazy        │     │
│  └────────────────────────────────────────────────────┘     │
├─────────────────────────────────────────────────────────────┤
│                   LLM 提供商层 (Provider)                    │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Provider.Chat(ctx, Request) → (Response, Stream)  │     │
│  │    ├─ DeepSeek (v3/v4/r1) ─ 前缀缓存优化           │     │
│  │    ├─ OpenAI (gpt-4o/o3)                           │     │
│  │    └─ Anthropic (claude-*)                         │     │
│  └────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## 组件交互矩阵

| | Controller | Agent | Session | Registry | Provider | Plugin |
|----|:---:|:---:|:---:|:---:|:---:|:---:|
| **Controller** | - | 创建+调用 | 读写 | 创建 | - | - |
| **Agent** | - | - | 读写 | 读 | 调用 | - |
| **Session** | - | - | - | - | - | - |
| **Registry** | - | - | - | - | - | - |
| **Provider** | - | - | - | - | - | - |
| **Plugin** | - | - | - | 写(Add) | - | - |
| **Skill** | 调用 | - | - | 写(Add) | - | - |
| **Hook** | 触发 | 触发 | - | - | - | - |

## 关键数据路径

| 路径 | 典型延迟 | 说明 |
|------|:---:|------|
| 用户输入→Controller.submit | <1ms | 进程内函数调用 |
| submit→Agent.Run | 5-20ms | Compose + 前置检查 |
| stream→first token | 200-2000ms | 网络+模型推理 |
| executeBatch→tool result | 10-5000ms | 文件I/O或shell执行 |
| event→前端渲染 | <16ms | 进程内/bridge IPC |
| turn_done→下turn | 0ms | Goal模式无缝衔接 |
