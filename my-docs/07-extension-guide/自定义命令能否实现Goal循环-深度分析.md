# 自定义命令能否实现 Goal 循环？—— 深度代码分析

<!--
版本: v1.0
创建: 2026-06-14 11:57:14
更新: 2026-06-14 11:57:14
-->

## TL;DR

| 答案 | 说明 |
|------|------|
| ❌ 纯自定义命令**不能** | 缺少两样东西：goal 状态未设置、协议指令未注入 |
| ⚡ 但**架构上极其接近** | 两者共享 `runGoalLoopWithRawDisplay` 同一条通道，打通成本极低 |

---

## 1. Goal 循环的三层机制

Goal 模式通过三个组件协作实现**智能循环**（不是死循环）：

```
┌─────────────────────────────────────────────────────────────┐
│  注入层: Compose() → activeGoalBlock()                       │
│  每轮在 prompt 前插入 <active-goal> 块，包含目标文本 + 标记协议 │
├─────────────────────────────────────────────────────────────┤
│  执行层: continueGoal() (controller.go:612)                  │
│  真正的 for 循环，每轮 model 回复后检查是否继续                   │
├─────────────────────────────────────────────────────────────┤
│  决策层: advanceGoalAfterTurn() (controller.go:631)          │
│  解析模型最后一行的状态标记 [goal:continue/complete/blocked]   │
└─────────────────────────────────────────────────────────────┘
```

### 1.1 注入层 —— `activeGoalBlock()`

```go
// input.go:137
func activeGoalBlock(goal string) string {
    // ── 返回一段注入文本 ──
    // "<active-goal>\n{目标文本}\n\nGoal mode: pursue this goal autonomously...
    // End every goal-mode assistant reply with exactly one status marker
    // on its own line: [goal:continue], [goal:complete], or [goal:blocked:<reason>]\n</active-goal>"
}
```

- `Compose()`（input.go:92）在 goal 活跃时自动注入，**不需要额外改动**
- 注入位置是每次用户消息前，不是 system prompt，因此不破坏 prefix cache

### 1.2 执行层 —— `continueGoal()`

```go
// controller.go:612
func (c *Controller) continueGoal(ctx context.Context) error {
    for {
        cont := c.advanceGoalAfterTurn()      // ← 检查标记
        if !cont { return nil }                // ← 决定是否继续
        if err := ctx.Err(); err != nil { ... }
        // 继续下一轮
        c.runTurnWithRawDisplay(ctx, goalContinueTurn, goalContinueTurn, "")
    }
}
```

### 1.3 决策层 —— `advanceGoalAfterTurn()`

```go
// controller.go:631
func (c *Controller) advanceGoalAfterTurn() bool {
    reply := lastAssistantText(c.History())
    status, reason, _ := parseGoalStatusMarker(reply)  // ← 解析 [goal:*] 标记
    c.mu.Lock()
    // ★★★ 关键检查 ★★★
    if strings.TrimSpace(c.goal) == "" || c.goalStatus != GoalStatusRunning {
        c.mu.Unlock()
        return false       // ← goal 为空，直接终止循环
    }
    // ... 根据 status 处理继续/完成/阻塞 ...
}
```

---

## 2. 自定义命令的执行路径

当用户输入 `/review "关注认证逻辑"` 时（controller.go:888-892）：

```go
// controller.go:888
if sent, ok := c.CustomCommand(trimmed); ok {
    c.runGuarded(func(ctx context.Context) error {
        return c.runGoalLoopWithRawDisplay(ctx, sent, sent, display)
        //     ↑ CustomCommand 也进了 Goal 循环入口！
    })
    return
}
```

关键发现：自定义命令和 `/goal` 命令**走的是同一个入口函数** `runGoalLoopWithRawDisplay`。对比 `/goal` 命令的路径：

```go
// controller.go:923-931 — /goal 命令
case GoalCommandSet:
    c.SetGoal(cmd.Text)                          // ← ① 先设 goal
    c.runGuarded(func(ctx context.Context) error {
        return c.runGoalLoopWithRawDisplay(...)  // ← ② 再进循环
    })
```

两者都调 `runGoalLoopWithRawDisplay`，差别仅在第 ① 步——自定义命令**跳过了 `SetGoal()`**。

---

## 3. 为什么跳过 SetGoal 就会立刻退出

`runGoalLoopWithRawDisplay` 的伪代码流程：

```
runGoalLoopWithRawDisplay()
  ├── runTurnWithRawDisplay()      → 跑第一轮（正常回复）
  └── continueGoal()               → 进入循环
        └── advanceGoalAfterTurn()
              ├── c.goal == ""     → return false  ← ★ 立刻退出
              └── c.goalStatus != Running → return false
```

因为自定义命令没有调用 `SetGoal()`，`c.goal` 保持空字符串，所以 `advanceGoalAfterTurn()` 在第一轮就返回 `false`，循环终止。

同时，由于 `c.goal` 为空，`Compose()` 中的 `activeGoalBlock()` 不会被触发注入，模型也**不知道**要输出 `[goal:continue]` 标记。

---

## 4. 打通路径 —— 极低成本扩展方案

### 方案 A（理论可做）：为 Command 结构加 `run-as-goal` 字段

改动只需三处：

| 改动位置 | 当前代码 | 改动后 |
|----------|----------|--------|
| **① `Command` 结构体** `internal/command/command.go:21` | 无 goal 相关字段 | 增加 `RunAsGoal bool`，从 frontmatter `run-as-goal: true` 解析 |
| **② `CustomCommand` 处理** `controller.go:888-892` | 直接 `runGoalLoopWithRawDisplay` | 如果 `cmd.RunAsGoal`，先 `c.SetGoal(cmd.Body)` |
| **③ 命令文件的 frontmatter** | `description:` / `argument-hint:` | 可选加 `run-as-goal: true` |

具体改动示例：

```diff
// internal/command/command.go
type Command struct {
    Name        string
    Description string
    ArgHint     string
+   RunAsGoal   bool       // 是否以 Goal 循环模式运行
    Body        string
    Source      string
}

// internal/command/command.go — parseFile
return Command{
    Name:        name,
    Description: fm["description"],
    ArgHint:     fm["argument-hint"],
+   RunAsGoal:   fm["run-as-goal"] == "true",
    Body:        strings.TrimSpace(body),
    Source:      path,
}
```

```diff
// internal/control/controller.go — 自定义命令分发
if sent, ok := c.CustomCommand(trimmed); ok {
+   // 查一下命令是否设置了 run-as-goal
+   fields := strings.Fields(trimmed)
+   name := strings.TrimPrefix(fields[0], "/")
+   for _, cmd := range c.commands {
+       if cmd.Name == name && cmd.RunAsGoal {
+           c.SetGoal(sent)
+           break
+       }
+   }
    c.runGuarded(func(ctx context.Context) error {
        return c.runGoalLoopWithRawDisplay(ctx, sent, sent, display)
    })
    return
}
```

> 注：更整洁的做法是让 `CustomCommand()` 多返回一个 `Command` 结构体，但上面的内联查找已经足够说明原理。

### 方案 B（当前即用）：两步串联

不需要任何代码改动，用户直接操作：

1. 先用 `/goal <目标>` 设定 Goal 并进入循环
2. Goal 循环结束后，用自定义命令 `/review` 做检查收尾

```
/goal 重构 internal/auth 包的错误处理，全部改为使用哨兵错误
[Goal 自动循环执行…]
/review "检查重构后的错误处理是否完整"
```

---

## 5. 与 `-y` 模式和 `/loop` 的对比

| 特性 | Goal 循环 | 自定义命令+方案A | `/loop <秒> <prompt>` |
|------|-----------|-----------------|----------------------|
| 循环控制 | 模型自决策（continue/complete/blocked） | 同 Goal | 固定时间间隔 |
| 限流保护 | 相同原因 blocked 3次停止 + 50轮上限 | 同 Goal | 无（靠超时/取消） |
| 可用范围 | 全前端（Desktop + CLI TUI） | 同 Goal | 仅 CLI TUI |
| 上下文累积 | 每轮追加，prefix cache 保留 | 同 Goal | 同 Goal |
| 注入协议 | 自动 `activeGoalBlock()` | 同 Goal（SetGoal后自动生效） | 无 |
| 适用场景 | 多步自主任务 | 多步自主任务（通过命令触发） | 定期监控/轮询 |

---

## 6. 结论

1. **纯自定义命令不能实现 Goal 循环**——因为 `c.goal` 为空导致 `continueGoal()` 第一轮就退出
2. **但架构上有极强的亲近性**——两者已共享 `runGoalLoopWithRawDisplay` 入口
3. **打通成本极低**——只需在 Command 结构加 `RunAsGoal bool` 字段，加载时从 frontmatter 解析，分发时调一次 `SetGoal()`
4. **这种扩展不会破坏现有行为**——对未设置 `run-as-goal` 的旧命令完全无影响

本质上，Goal 循环和自定义命令是「同一个管道，两个阀门」的关系。加上一个 frontmatter 标记就能让自定义命令获得完整的 Goal 循环能力。
