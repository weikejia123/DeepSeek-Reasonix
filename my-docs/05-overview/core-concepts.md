# 核心概念词汇表

<!--
版本: v1.0
创建: 2026-06-14 00:08:00
更新: 2026-06-14 00:08:00
-->

按字母序排列的 20 个核心概念，每个概念附相关文档链接。

## Agent
LLM 驱动的自动执行器。接收 Composed 输入，通过 `Agent.Run()` 循环调用模型→解析 tool calls→执行工具→反馈结果→继续循环，直到模型返回 final answer。

→ 详见: `01-dev-read/10-agent-core-loop.md`

## Boot
系统启动流程。`Build()` 函数完成：配置加载→Session 恢复→Memory 加载→Skill 发现→工具注册→MCP 启动→系统前缀构建→Controller 创建。

→ 详见: `01-dev-read/15-boot-sequence.md`

## Compaction
上下文压缩。当 prompt tokens 接近 context window 的 80% 时，自动将中间消息发给 summarizer 模型压缩为摘要，保留首尾。

→ 详见: `01-dev-read/13-context-compaction.md`

## Compose
每 Turn 的动态注入。`Controller.Compose()` 在用户输入前注入 goal block、plan marker、memory-update、background-jobs 等动态块。

→ 详见: `01-dev-read/13-context-compaction.md`

## Controller
前端的唯一后端入口。负责 Submit 路由（命令/普通输入/Skill）、Compose、Approval、Goal 循环、Plan 管理等。

→ 详见: `01-dev-read/`

## Event
Agent 到前端的通信单元。19 种 Event：TurnStarted/Text/Reasoning/ToolDispatch/ToolResult/Usage/Notice/Phase/ApprovalRequest/AskRequest/TurnDone/CompactionStarted/CompactionDone/Steer/Retrying/ToolProgress/Message/BackgroundJob/MCPServerReady。

→ 详见: `01-dev-read/25-event-system.md`

## Evidence
任务验证系统。每 Turn 的 evidence ledger 记录工具执行证据，`complete_step` 验证证据匹配后才签名步骤完成。

→ 详见: `01-dev-read/22-evidence-todo.md`

## Goal
自主执行模式。设定目标后 Controller 自动循环：等待模型输出 `[goal:continue|complete|blocked]` 标记，continue 时自动提交下一 Turn。

→ 详见: `02-dev-faq/goal-mode-comprehensive.md`

## Hook
外部脚本扩展。10 个事件点（PreToolUse/PostToolUse/UserPromptSubmit/Stop/PostLLMCall/SessionStart/SessionEnd/SubagentStop/Notification/PreCompact），阻塞式可否决操作，非阻塞式后台执行。

→ 详见: `01-dev-read/14-hook-system.md`

## MCP (Model Context Protocol)
外部工具协议。通过 stdio/HTTP/SSE 传输连接外部 MCP 服务器，其工具以 `mcp__<server>__<tool>` 命名自动注册到 Registry。支持 eager/background/lazy 三层启动。

→ 详见: `01-dev-read/12-mcp-system.md`

## Plan
双轮规划模式。第一轮：Planner 用只读研究工具分析→输出分层任务计划→用户批准。第二轮：Executor 用完整工具集+自动批准执行计划。

→ 详见: `01-dev-read/20-plan-mode.md`

## Prefix（系统前缀）
缓存稳定的系统提示词。在 session 期间内容不变，利用 DeepSeek 自动前缀缓存大幅降低 token 成本。Skill bodies、memory updates 不进 prefix。

→ 详见: `01-dev-read/13-context-compaction.md`

## Provider
LLM 提供商抽象。`Provider.Chat()` 接口统一 DeepSeek/OpenAI/Anthropic 的调用差异。

→ 详见: `01-dev-read/17-provider-system.md`

## Registry
每 Session 的工具注册表。`Registry.Add(t)` 注册工具，`Schemas()` 导出模型可见的工具列表。支持 `RemovePrefix` 批量移除（MCP 断连时）。

→ 详见: `01-dev-read/09-tool-system.md`

## Session
会话容器。`Session.Messages []provider.Message` 存储完整的 user/assistant/tool/steer 消息历史。JSONL 持久化。

→ 详见: `01-dev-read/16-session-persistence.md`

## Skill
可复用 Playbook。Markdown 文件定义，分 inline（工具结果注入）和 subagent（独立子代理循环）两种。通过 `run_skill`/`read_skill`/`/<name>` 调用。

→ 详见: `01-dev-read/11-skill-system.md`

## Steer
Mid-Turn 消息队列。Turn 运行期间用户发送的消息进入 `steerQueue`，在同 Turn 的循环迭代中逐条消费注入模型。

→ 详见: `04-capabilities/steer-queue.md`

## Subagent
隔离的子代理。`task` 工具和 subagent skill 创建独立 session + 过滤工具集 + 隔离 context 的子代理。支持 `continue_from`/`fork_from` 迭代和分叉。

→ 详见: `01-dev-read/18-subagent-task.md`

## Tool
模型可调用的能力单元。`Tool` 接口：`Name()/Description()/Schema()/Execute()/ReadOnly()`。30+ 内置工具 + MCP 动态工具 + Skill 工具。

→ 详见: `01-dev-read/09-tool-system.md`

## Turn
一次完整的 Agent.Run() 调用周期。从 TurnStarted 到 TurnDone，中间可能包含多次 model call→tool execution 循环。
