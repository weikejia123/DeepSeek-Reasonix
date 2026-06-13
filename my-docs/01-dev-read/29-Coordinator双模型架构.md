# Coordinator — Planner + Executor 双模型协作架构

<!--
版本: v1.0
创建: 2026-06-14 01:10:00
更新: 2026-06-14 01:10:00
-->

**归流自**: `02-dev-faq/Goal模式完整分析.md` + `Desktop-UI面板FAQ.md` Q6

## 架构概览

Coordinator 是双模型的"导演"——每次 turn 先调 planner（规划）再调 executor（执行）：

```
Coordinator.Run(ctx, input)
  │
  ├─ shouldPlan(input)?  ← 简单问题跳过规划
  │     └─ false → executor.Run(input)  直走执行
  │
  ├─ Phase: "deepseek-v4-pro · planning"
  ├─ plan(ctx, input)
  │     ├─ plannerAgent? → planWithTools()  (有工具集)
  │     └─ 否则 → planner.Stream()         (纯文本规划)
  │
  ├─ Phase: "deepseek-v4-flash · executing"
  └─ executor.Run(ctx, formatHandoff(input, plan))
        └─ executorHandoffGuard = true  ← nudge 机制
```

## 一、配置方式

```toml
[agent]
planner_model = "deepseek-v4-pro"   # 有此字段 = 双模型
# 不设 = 单模型（Agent 既是 planner 又是 executor）
```

启动时 `boot.go:848-871` 检测 `planner_model` 非空 → 创建 Coordinator：

```go
plannerProv, _ := NewProviderWithProxy(pe, proxySpec)
plannerSess := agent.NewSession(PlannerPromptWithContext(mem.Block()))
plannerTools := agent.PlannerToolRegistry(reg)

runner = agent.NewCoordinator(plannerProv, plannerSess, pe.Price,
    plannerTools, agent.Options{...}, executor, ...)
```

## 二、Planner 的隔离边界

| 维度 | Planner | Executor |
|------|---------|----------|
| 模型 | `planner_model` (如 pro) | 默认模型 (如 flash) |
| Session | 独立 `plannerSess` | 主 `session` |
| 工具 | `PlannerToolRegistry`（仅只读） | 完整 Registry |
| 系统提示 | `DefaultPlannerPrompt` | 常规系统提示 |
| prefix cache | 独立缓存 | 独立缓存 |

**两个 Session 完全隔离**——planner 的消息不进 executor 的 context，prefix cache 互不干扰。

## 三、shouldPlan — 跳过规划

```go
// boot.go:878-882
shouldPlan := func(input string) bool {
    return len(input) > 500 || strings.Contains(input, "\n")
}
```

简单短句（如"你好"、"现在几点了"）跳过 planner 直走 executor，节省一轮 API 调用。

## 四、Planner 的系统提示

`DefaultPlannerPrompt` (`coordinator.go:21-29`)：

> You are the planner in a two-model coding agent. Given a task, produce a concise, ordered plan for the executor model to carry out. Use the read-only tools available to you when the task needs context from the workspace, user rules, or docs; keep that research targeted and stop once you have enough evidence. Do not write full implementations or attempt side effects.

## 五、Goal 模式下的 Planner-Goal 冲突

**这是 Goal 模式分析瘫痪的根因**：

| 来源 | 指令 | 模型能力 |
|------|------|---------|
| Planner 系统提示 | "stop once you have enough evidence" → 产出计划 | 只读 |
| Goal 块 (`Compose()`) | "do not stop after describing a plan; **execute**" | 只读 |
| Planner 实际能力 | 无 `write_file`/`bash` — **只能读不能写** | 只读 |

**矛盾链**：Goal 块强制"执行下一步" → Planner 无法执行（无写工具） → 唯一能动 = 读更多文件 → 分析瘫痪。

`Compose()` 是**无条件注入**的——不区分当前 turn 是否经过 planner。Planner 和 Executor 看到的是同一份被 goal 块污染的 user 消息。

### 单模型 vs 双模型的 Goal 差异

| 维度 | 单模型 | 双模型 |
|------|--------|--------|
| 每次 turn 角色 | 同一模型既规划又执行 | Planner 只读 → Executor 执行 |
| Goal 块影响 | "执行下一步"合理——有写工具 | 与 planner 系统提示**直接冲突** |
| 分析瘫痪风险 | 低 | **高** |
| 适用场景 | ✅ Goal 友好 | ⚠️ Goal 需特别谨慎 |

## 六、Executor Handoff 机制

```go
executor.executorHandoffGuard = true  // coordinator.go:76
```

Executor 被设置为"从 planner 接手"模式——如果 executor 第一轮没调用任何工具（以为 plan 只是信息性的），Agent.Run 会 nudge 它（最多 3 次）：

```go
// agent.go:642-648
if executorHandoff && !usedAnyTool && handoffNudges < maxExecutorHandoffNudges {
    handoffNudges++
    a.session.Add(executorHandoffRetryMessage())
    continue
}
```

## 七、代码索引

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/agent/coordinator.go` | 45-59 | Coordinator 结构 |
| `internal/agent/coordinator.go` | 65-88 | NewCoordinator |
| `internal/agent/coordinator.go` | 91-104 | Run() — plan → execute |
| `internal/agent/coordinator.go` | 108-158 | plan() / planWithTools() |
| `internal/boot/boot.go` | 848-871 | 双模型检测与构建 |
| `internal/boot/boot.go` | 878-882 | shouldPlan |
| `internal/control/input.go` | 93-103 | Compose 无条件注入 goal 块 |
| `internal/agent/agent.go` | 642-648 | executorHandoff nudge |
