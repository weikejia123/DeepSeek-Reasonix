# Planner 角色权限与工具限制

## 问题现象

在 planning 阶段，planner 尝试执行 `git status --short` 时报错 `unknown tool "bash"`。Planner 没有 shell/命令执行工具。

## 根因分析

Planner 的工具集由 `PlannerToolRegistry()` 过滤（`internal/agent/task.go:391-393`）：

```go
func PlannerToolRegistry(parent *tool.Registry) *tool.Registry {
    exclude := append(SubagentMetaTools(), plannerNonResearchTools...)
    return FilterReadOnlyRegistry(parent, exclude...)
}
```

过滤规则（`FilterReadOnlyRegistry`, `task.go:398-418`）：
1. 只保留 `ReadOnly() == true` 的工具
2. 排除 `plannerNonResearchTools`：`ask`, `bash_output`, `complete_step`, `slash_command`, `todo_write`, `wait`
3. 排除 `subagentMetaTools`：subagent/task 等元工具

**关键点**：`bash` 工具的 `ReadOnly()` 返回 `false`（`internal/tool/builtin/bash.go:115`），因为 shell 命令的副作用无法从参数预判（即使是 `git status` 这样的只读命令）。因此即使 `git status --short` 是纯只读操作，`bash` 工具也不会暴露给 planner。

Planner 可用的只读研究工具仅限于：
- `glob` — 文件模式匹配
- `grep` — 代码搜索
- `read_file` — 文件读取
- `ls` — 目录列表
- `web_fetch` — HTTP GET
- `lsp_definition` / `lsp_hover` / `lsp_references` / `lsp_diagnostics` — LSP 查询
- `memory` — 记忆检索
- `history` — 会话历史检索
- `codegraph_*` — 代码图谱只读查询

## 影响的场景

- 无法 `git status` / `git diff` / `git log`
- 无法 `go build` / `go test` 等任何命令
- 无法执行任何 shell 管道

## 设计理由

1. **安全边界**：planner 是只读角色，`bash` 的 `ReadOnly()` 契约必须保守（因为无法静态判定命令是否无副作用）
2. **缓存优化**：planner session 独立且轻量，保持低频调用
3. **职责分离**：planner 只需足够的上下文做出计划，不需要执行能力

## 能否"让 planner 执行 git status"？

不能。工具过滤是**工具级别**的开关，不是命令级别：
- `bash` 工具要么全部可用，要么全部不可用
- 它的 `ReadOnly()` 返回 `false`，所以整个被排除
- 无法"只允许 git status 而不允许 git push"

如果要加入这个能力，需要注册一个新的只读工具（如 `GitStatus()`），但过于专一，不推荐。

## 相关源码

- `internal/agent/task.go:378-418` — PlannerToolRegistry 和 FilterReadOnlyRegistry
- `internal/tool/builtin/bash.go:112-115` — bash.ReadOnly() = false
- `internal/agent/coordinator.go:20-29` — DefaultPlannerPrompt
- `internal/agent/coordinator.go:144-158` — planWithTools（planner agent loop）
- `internal/tool/tool.go:27-32` — ReadOnly 接口定义
