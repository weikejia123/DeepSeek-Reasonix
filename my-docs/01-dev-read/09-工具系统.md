# Tool System — 工具注册、分发与执行

<!--
版本: v1.0
创建: 2026-06-13 23:00:00
更新: 2026-06-13 23:00:00
-->

**能力类型**: 系统架构文档  
**涉及包**: `internal/tool/`、`internal/tool/builtin/`、`internal/agent/`、`internal/boot/`

## 架构总览

```
Tool 接口 (tool.go:18-33)
  ├─ Name() / Description() / Schema()      — 模型可见的元数据
  ├─ Execute(ctx, args) → (string, error)   — 实际执行
  └─ ReadOnly() bool                        — 并行调度判定

注册层次:
  ┌─ Builtin 层 (tool/builtin/*.go)
  │   └─ init() → RegisterBuiltin(t)        — 编译时注册，全局单例
  ├─ Agent 层 (agent/task.go, agent/ask.go)
  │   └─ task, ask                          — Agent 基础设施工具
  ├─ 子系统层 (history/, memory/, skill/, command/)
  │   └─ history, memory/remember/forget, skill tools, slash_command
  ├─ MCP 层 (mcp/)
  │   └─ mcp__<server>__<tool>             — 运行时代理
  └─ 动态层 (connect_tool_source)
      └─ skills, codegraph, lsp, web_fetch — 按需激活
```

---

## 一、Tool 接口

所有工具实现 `tool.Tool` 接口（`tool.go:18-33`）：

```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage      // JSON Schema 参数定义
    Execute(ctx context.Context, args json.RawMessage) (string, error)
    ReadOnly() bool                // 是否只读 → 并行执行判定
}
```

### 可选接口

| 接口 | 作用 | 实现者 |
|------|------|--------|
| `Previewer` (`tool.go:41-43`) | 预览文件变更而不实际写入 | `edit_file`, `write_file`, `multi_edit` 等 |
| `ResolveProfile(args) *event.Profile` | 返回子代理的 model/effort 配置 | `task`, `run_skill`, 子代理包装工具 |

---

## 二、Registry 注册表

`tool.Registry`（`tool.go:100-105`）是**每次 Build 创建一次的运行时工具集**：

```go
type Registry struct {
    tools map[string]Tool       // name → Tool
    order []string              // 插入顺序
    canon map[string]json.RawMessage  // 预规范化的 schema
}
```

### 关键方法

| 方法 | 作用 |
|------|------|
| `Add(t Tool)` | 插入/替换工具，首次时记录插入顺序 |
| `Get(name) (Tool, bool)` | 按名查找 |
| `RemovePrefix(prefix)` | 移除前缀匹配的所有工具（MCP 断连时用） |
| `Schemas() []provider.ToolSchema` | 每 turn 导出模型可见的工具列表 |
| `Names() []string` | 按插入顺序返回工具名 |

### MCP 命名空间

MCP 工具自动带 `mcp__<server>__<tool>` 前缀（`tool.go:129`），`SplitMCPName()` 可反向解析。

---

## 三、全局 Builtin 集

编译时通过 `init()` 注册（`tool.go:65-75`）：

```go
var builtins = map[string]Tool{}

func RegisterBuiltin(t Tool) {
    // panic on duplicate name
    builtins[t.Name()] = t
}
```

每个 `internal/tool/builtin/*.go` 在包初始化时注册自己的工具。`Builtins()` / `LookupBuiltin(name)` 提供只读访问。

---

## 四、执行模型

### dispatch → executeBatch 流水线

```
Agent Run loop (agent.go:553)
  ├─ stream(ctx) → model 返回 tool calls
  ├─ executeBatch(ctx, calls)                   — agent.go:1085
  │   ├─ 1. 发射 ToolDispatch 事件
  │   │     ├─ 查找工具、获取 Preview
  │   │     └─ 解析 ResolveProfile（子代理模型/effort）
  │   ├─ 2. partitionToolCalls()               — 并行/串行分批
  │   │     ├─ 全部 ReadOnly → 并行批
  │   │     ├─ 有 writer → 串行批（保证顺序）
  │   │     └─ complete_step/todo_write → 单列批（需读 evidence ledger）
  │   ├─ 3. executeOne(ctx, call)              — agent.go:1291
  │   │     ├─ 权限门控（approval gate）
  │   │     ├─ 沙箱（bash 工具）
  │   │     ├─ Execute(ctx, args)
  │   │     └─ 错误序列化 / 输出截断
  │   └─ 4. 发射 ToolResult 事件
  └─ applyStormBreaker(calls, outcomes)         — 不安全的并行写检测
```

### 并行执行规则（`partitionToolCalls`，agent.go:1155）

- **全部 ReadOnly** → 同批并行
- **任一 writer** → 拆为单独串行批
- **complete_step / todo_write** → 即使 ReadOnly 也不入并批（依赖 evidence ledger 状态）

### 权限门控

`executeOne` 中检查两个 Gate（`agent.go:1291` 起）：
1. **Approval Gate** — 需用户批准的 writer 工具被阻塞，emit `ApprovalRequest`
2. **Ask Gate** — `ask` 工具阻塞等待用户回答

YOLO 模式绕过 Approval Gate（ask 不受影响）。

---

## 五、全量工具目录

### 5.1 文件读写（6 个）

| 工具 | 包 | ReadOnly | 说明 |
|------|-----|:---:|------|
| `read_file` | builtin/readfile.go | ✅ | 读文本文件，支持 offset/limit 分页 |
| `write_file` | builtin/writefile.go | ❌ | 写/覆盖文件，自动创建父目录 |
| `edit_file` | builtin/editfile.go | ❌ | 精确字符串替换编辑 |
| `multi_edit` | builtin/multiedit.go | ❌ | 原子化多步编辑（全部成功才落盘） |
| `delete_range` | builtin/delete_range.go | ❌ | 按起止行锚点删除 |
| `delete_symbol` | builtin/delete_symbol.go | ❌ | AST 级符号删除（Go 文件） |

### 5.2 搜索与浏览（4 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `grep` | ✅ | 正则搜索，支持 .gitignore，200 条上限 |
| `glob` | ✅ | 文件模式匹配（`**/*.go`） |
| `ls` | ✅ | 目录列表，支持递归 |
| `web_fetch` | ✅ | HTTP GET，HTML 自动转可读文本 |

### 5.3 Shell 执行（4 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `bash` | ❌ | 执行 shell 命令，沙箱可选（macOS Seatbelt） |
| `bash_output` | ✅ | 读后台 bash 任务的增量输出 |
| `kill_shell` | ❌ | 终止后台 bash 任务 |
| `wait` | ❌ | 阻塞等待后台任务完成 |

### 5.4 任务与交互（4 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `todo_write` | ✅ | 读写任务列表（只读是因为不涉及文件 I/O，但被排除于并批） |
| `complete_step` | ✅ | 签收执行计划中的一个步骤（读 evidence ledger） |
| `ask` | ❌ | 向用户提问（多选），阻塞等待回答 |
| `task` | ❌ | 创建子代理执行子任务，返回最终答案 |

### 5.5 笔记本编辑（1 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `notebook_edit` | ❌ | 编辑 .ipynb 单元格（replace/insert/delete） |

### 5.6 记忆管理（3 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `memory` | ✅ | 搜索/列出/读取已保存的记忆 |
| `remember` | ❌ | 保存/更新记忆（写文件） |
| `forget` | ❌ | 删除记忆（删文件） |

### 5.7 会话（1 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `history` | ✅ | 搜索本地会话历史（BM25 检索） |

### 5.8 技能（3+4 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `run_skill` | ❌ | 调用技能：inline→工具结果，subagent→子代理 |
| `read_skill` | ✅ | 只读加载 inline 技能内容（plan 模式可用） |
| `install_skill` | ❌ | 创建/保存新技能文件 |
| `explore` | ❌ | 子代理：只读代码探索 |
| `research` | ❌ | 子代理：web_fetch + 代码阅读 |
| `review` | ❌ | 子代理：diff 审查 |
| `security_review` | ❌ | 子代理：安全审查 |

### 5.9 命令（1 个）

| 工具 | ReadOnly | 说明 |
|------|:---:|------|
| `slash_command` | ✅ | 执行自定义斜杠命令模板 |

### 5.10 代码智能（按需激活）

| 工具组 | 激活方式 | 说明 |
|--------|----------|------|
| `codegraph_*` | `connect_tool_source("codegraph")` | 符号搜索、调用图、影响分析 |
| `lsp_*` | `connect_tool_source("lsp")` | LSP 定义跳转、悬停、诊断 |
| `web_fetch` | `connect_tool_source("web_fetch")` | 在 economy 模式下按需激活 |

---

## 六、FilterRegistry — 子代理工具剪裁

`FilterRegistry`（`agent/task.go:357`）从父 Registry 克隆并过滤工具：

```go
func FilterRegistry(parent *Registry, names []string, exclude ...string) *Registry
```

- `names` 非空 → 白名单模式（只保留 named tools）
- `names` 为空 → 继承全部，排除 `exclude` 列表
- 子代理必排除：`SubagentMetaTools()` = `["task", "run_skill", "read_skill", "install_skill", "explore", "research", "review", "security_review"]`
- Planner 额外排除：`plannerNonResearchTools` = `["ask", "bash_output", "complete_step", "slash_command", "todo_write", "wait"]`，且只保留 `ReadOnly() == true` 的工具

---

## 七、错误处理

`executeOne` 统一处理（`agent.go:1291` 起）：

1. `ctx.Err()` → 返回 cancelled/timeout 错误
2. `json.Unmarshal` 失败 → 返回 `"[invalid args] ..."` 
3. `Execute` 返回 error → 捕获错误文本
4. 输出过长 → 截断到 `maxToolOutput`，emit Notice

---

## 八、代码索引

| 文件 | 内容 |
|------|------|
| `internal/tool/tool.go` | Tool 接口、Registry、Builtin 全局集、Previewer、MCP 命名 |
| `internal/tool/builtin/*.go` | 16 个内置工具实现 |
| `internal/agent/agent.go:1085-1141` | `executeBatch` 并行分批执行 |
| `internal/agent/agent.go:1155` | `partitionToolCalls` 串行/并行判定 |
| `internal/agent/agent.go:1291` | `executeOne` 单工具执行 + 门控 |
| `internal/agent/task.go:350-418` | `FilterRegistry` / `PlannerToolRegistry` |
| `internal/boot/boot.go:360-730` | Build 阶段工具注册全流程 |
