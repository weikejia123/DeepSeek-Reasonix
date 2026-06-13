# Plan Mode — 双轮规划架构

<!--
版本: v1.0
创建: 2026-06-13 23:48:00
更新: 2026-06-13 23:48:00
-->

**能力类型**: 系统架构文档
**代码位置**: `internal/control/controller.go:524-588`

## 双轮流程

```
Turn 1 — 规划 (planMode=true, Compose 注入 PlanModeMarker)
  ├─ Agent.Run(ctx, plan-marked input)
  │     ├─ PlannerToolRegistry → 仅只读工具 (read_file, grep, glob, ls, web_fetch, ask, ...)
  │     └─ 写入工具不可用 — harness 拒绝
  ├─ 模型返回 plan (markdown 分层任务列表)
  ├─ ApprovalRequest → 用户批准
  │     ├─ 批准 → SetPlanMode(false) + 注册 auto-approve
  │     └─ 拒绝 → 保持 plan 模式
  │
Turn 2 — 执行 (planMode=false)
  ├─ Agent.Run(ctx, planApprovedMessage)
  │     ├─ 完整工具集
  │     ├─ 写工具 auto-approve (approved plan turn only)
  │     └─ seedPlanTodos(plan) → todo_write 播种
  └─ completePlanTodos(args) → 标记完成
```

## 一、PlanModeMarker

`input.go:16` — 写入工具不可用的长 system-prompt 级指令，指导模型使用只读研究工具并提出分层计划。

## 二、PlannerToolRegistry

```go
// task.go:391-393
func PlannerToolRegistry(parent *Registry) *Registry {
    exclude := append(SubagentMetaTools(), plannerNonResearchTools...)
    return FilterReadOnlyRegistry(parent, exclude...)
}
```

排除: `ask`, `bash_output`, `complete_step`, `slash_command`, `todo_write`, `wait`

## 三、Approved Plan Turn

```
planMode=false
approvedPlanAutoApproveTools=true  → 工具 auto-approve
seedPlanTodos → todo_write 播种
Agent.Run → 执行 → completePlanTodos 标记完成
```

## 四、Plan 互斥

- Goal 激活时跳过 auto-plan
- Plan 模式下 goal 被忽略
- `/goal` 与 plan 互斥切换

## 五、代码索引

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/input.go:16` | PlanModeMarker |
| `internal/control/controller.go:555-588` | plan approval 子 turn |
| `internal/agent/task.go:391` | PlannerToolRegistry |
| `internal/control/controller.go:2701-2725` | seedPlanTodos / completePlanTodos |
