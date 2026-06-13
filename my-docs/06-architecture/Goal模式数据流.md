# Goal 模式 — 全链路数据流与最佳实践

<!--
版本: v1.0
创建: 2026-06-14 01:20:00
更新: 2026-06-14 01:20:00
-->

**归流自**: `02-dev-faq/Goal模式完整分析.md` + `Desktop-UI面板FAQ.md` Q4

## 一、全链路时序

```
用户: /goal 重构 internal/boot 包
  │
  ▼
controller.applyGoalCommand()
  ├─ SetGoal("重构 internal/boot 包")
  └─ runGoalLoopWithRawDisplay(input, raw, display)
       │
       ▼
  ┌─ Turn 1 ──────────────────────────────────┐
  │  Compose(input)                            │
  │    └─ inject <active-goal> block            │
  │  Agent.Run(composed)                       │
  │  model → [goal:continue]                   │
  └────────────────────────────────────────────┘
       │
       ▼ continueGoal() ← advanceGoalAfterTurn() 解析标记
       │
  ┌─ Turn 2 ──────────────────────────────────┐
  │  Compose(goalContinueTurn)                 │
  │    └─ inject <active-goal> block AGAIN     │
  │  Agent.Run(composed)                       │
  │  model → [goal:complete]                   │
  └────────────────────────────────────────────┘
       │
       ▼ advanceGoalAfterTurn() → status=complete → return false
       │
       ▼ Goal 结束
```

## 二、关键机制

### 2.1 Goal 块注入（每次 turn 无条件）

```go
// input.go:93-103
func (c *Controller) Compose(text string) string {
    if goal active {
        text = activeGoalBlock(goal) + "\n\n" + text
    }
    // ...
}
```

**不区分**当前 turn 是 planner 还是 executor——导致双模型下的指令冲突。

### 2.2 advanceGoalAfterTurn — 状态机

```
advanceGoalAfterTurn()
  ├─ parseGoalStatusMarker(reply)
  │     ├─ [goal:continue] → goalBlocks=0, return true (继续)
  │     ├─ [goal:complete] → goal 清空, return false (完成)
  │     └─ [goal:blocked:<reason>] → sameGoalBlock?
  │           ├─ 相同原因 ≥ 3 次 → return false
  │           └─ 新原因 → goalBlocks=1, return true
  └─ goalTurns ≥ maxGoalAutoTurns(50)? → return false
```

### 2.3 保护机制

| 机制 | 阈值 | 行为 |
|------|------|------|
| maxGoalAutoTurns | 50 | 强制 `GoalStatusBlocked` |
| Blocked 检测 | 3 次相同原因 | 标记 blocked，停止 |
| Context 取消 | — | `stopGoal(GoalStatusStopped)` |
| PlannerMaxSteps | 配置 | Planner 单轮最大步数 |

## 三、使用方法

### 开启 Goal

| 方法 | 操作 |
|------|------|
| TUI 命令 | `/goal 重构 internal/boot 包` |
| Desktop 意图菜单 | 左下角 mode chip → Goal → 输入目标 → 发送 |
| 查看状态 | `/goal`（不带参数） |
| 清除 | `/goal clear` |

### Goal 提示词原则

| ✅ 适合 Goal | ❌ 不适合 Goal |
|-------------|---------------|
| "为 API 添加 OAuth2 登录" | "研究一下认证机制" |
| "修复 invoices 表 tax_rate bug" | "了解代码库结构" |
| "把 Dashboard 从 N+1 改成 JOIN" | "看看有没有优化空间" |
| "提升登录页面的可访问性" | "分析数据库样本,寻找商机" |

**核心原则**: Goal 假设"知道要做什么，只需执行"。如果任务需要"先研究再决定做什么"，拆成多步人工引导。

### 双模型 + Goal 特别警告

如果配置了 `planner_model`，Goal 模式下 planner 会陷入**分析瘫痪**：
- Planner 只有只读工具，但 Goal 块强制"执行下一步"
- Planner 无法写文件 → 只能不断读更多文件
- 每次 `[goal:continue]` 都消耗 API 费用但无代码产出

**缓解**: 使用具体可执行的目标 + 设置 `PlannerMaxSteps ≤ 15`

## 四、与 /loop 的区别

| 维度 | Goal | /loop |
|------|------|-------|
| 决策者 | 模型自主决定下一步 | 用户预设固定 prompt |
| 推进方式 | `[goal:continue]` 标记驱动 | 定时器驱动 |
| 适用 | 多步编码任务 | 重复监控/检查 |

## 五、代码索引

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/input.go` | 93-103 | Compose goal 注入 |
| `internal/control/input.go` | 134-146 | activeGoalBlock() |
| `internal/control/controller.go` | 206-207 | maxGoalAutoTurns + goalContinueTurn |
| `internal/control/controller.go` | 514-522 | runGoalLoopWithRawDisplay |
| `internal/control/controller.go` | 590-656 | continueGoal + advanceGoalAfterTurn |
| `internal/control/controller.go` | 1293-1314 | SetGoal / ClearGoal |
