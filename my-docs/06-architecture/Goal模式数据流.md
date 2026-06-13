# Goal 模式 — 全链路数据流与最佳实践

<!--
版本: v1.1
创建: 2026-06-14 01:20:00
更新: 2026-06-14 02:10:00
-->

**归流自**: `02-dev-faq/Goal模式完整分析.md` + `Desktop-UI面板FAQ.md` Q4
**v1.1 更新**: 补充完整文件清单（发现涉及 10 个文件，非仅 2 个）+ 行号对齐 v1.7.0

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

## 五、完整文件清单

Goal 模式涉及 **6 个 Go 文件** + **4 个前端文件**：

### Go 后端

| 文件 | 角色 | 行号 |
|------|------|------|
| `internal/control/controller.go` | **核心**：状态字段、SetGoal/ClearGoal、runGoalLoop、continueGoal、advanceGoalAfterTurn、parseGoalStatusMarker、stopGoal、applyGoalCommand | 多处（见下方详情） |
| `internal/control/input.go` | Goal 块生成 (`activeGoalBlock`)、命令解析 (`ParseGoalCommand`)、状态常量 (`GoalStatus*`) | 16-26, 91-103, 137-146, 181-206 |
| `internal/control/auto_plan.go` | Goal 活跃时跳过 auto-plan | 46-49 |
| `internal/cli/chat_tui.go` | CLI 层：goal 模式下的 tool approval 决策 + `/goal` slash command 分发 | 3138-3175, 3619-3632 |
| `internal/control/goal_test.go` | 测试：自动连续执行 + 跳过 plan approval | 全文 |
| `internal/control/input_test.go` | 测试：Goal 命令设置/报告/清除 | 157 |

### 前端 (Desktop)

| 文件 | 角色 |
|------|------|
| `desktop/frontend/src/lib/types.ts` | `GoalStatus` 类型 (`running\|complete\|blocked\|stopped`)、`CollaborationMode` 含 `"goal"`、`normalizeCollaborationMode` |
| `desktop/frontend/src/lib/useController.ts` | Goal 状态管理：`goal`/`goalStatus`/`goalTurns` 字段、`clearGoal`/`setGoal` actions |
| `desktop/frontend/src/components/Composer.tsx` | Goal UI：左下角 mode chip → Goal → 输入目标 |
| `desktop/frontend/src/App.tsx` | Goal 状态从 useController 传递到 Composer + StatusBar |

## 六、详细代码索引

### 6.1 常量与类型定义

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/input.go` | 16-26 | GoalStatus 常量 (`running`/`complete`/`blocked`/`stopped`) + `<active-goal>` XML 标签 |
| `internal/control/input.go` | 181-206 | `GoalCommandAction` 枚举 + `GoalCommand` 结构体 + `ParseGoalCommand()` |
| `internal/control/controller.go` | 134-138 | Controller 结构体中 goal 相关字段：`goal`、`goalStatus`、`goalTurns`、`goalBlocks`、`goalBlock` |
| `internal/control/controller.go` | 210-212 | `maxGoalAutoTurns = 50` + `goalContinueTurn` 常量 |

### 6.2 Goal 块注入

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/input.go` | 91-103 | `Compose()` 中注入 `<active-goal>` 块（每次 turn 无条件） |
| `internal/control/input.go` | 137-146 | `activeGoalBlock()` — 生成 goal 提示文本 |

### 6.3 Goal 命令处理

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/controller.go` | 778 | `Send()` 中调用 `applyGoalCommand(trimmed, display)` |
| `internal/control/controller.go` | 918-945 | `applyGoalCommand()` — 解析命令 → SetGoal/ClearGoal/Status，SetGoal 后立即启动首轮 |
| `internal/cli/chat_tui.go` | 3619-3632 | CLI 侧 `/goal` slash command → `ParseGoalCommand()` + `applyGoalCommand()` |

### 6.4 Goal 循环控制

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/controller.go` | 1326-1346 | `SetGoal()` — 设置目标并重置计数器 |
| `internal/control/controller.go` | 1348-1350 | `ClearGoal()` — 清除目标 |
| `internal/control/controller.go` | 1352-1368 | `Goal()` / `GoalStatus()` — 读取状态 |
| `internal/control/controller.go` | 526-538 | `runGoalLoopWithRawDisplay()` — 执行单轮 + `continueGoal()` |
| `internal/control/controller.go` | 612-629 | `continueGoal()` — 循环：`advanceGoalAfterTurn()` → 再执行 `goalContinueTurn` |
| `internal/control/controller.go` | 631-678 | `advanceGoalAfterTurn()` — 解析标记 → 状态转移 → 是否继续 |
| `internal/control/controller.go` | 730-736 | `stopGoal()` — 被 context 取消或 maxTurns 触发 |

### 6.5 标记解析

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/controller.go` | 680-701 | `parseGoalStatusMarker()` — 从最后一行解析 `[goal:continue\|complete\|blocked:...]` |
| `internal/control/controller.go` | 703-728 | `sameGoalBlock()` + `cleanGoalBlockReason()` + `normalizeGoalBlockReason()` |

### 6.6 Goal 对其他系统的影响

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/auto_plan.go` | 46-49 | `shouldAutoPlan()` — goal 活跃时跳过 auto-plan |
| `internal/cli/chat_tui.go` | 3138-3175 | Goal 模式下的 tool approval 行为调整（yolo/auto 审批表现不同） |

### 6.7 其他入口（非 /goal 命令也能触发 goal loop）

Controller 中多个 dispatch 路径使用 `runGoalLoopWithRawDisplay` — 这意味着如果 goal 在运行中，**任何**用户输入都会触发 continueGoal：

| 行号 | 触发条件 |
|------|---------|
| `controller.go:471` | `SendWithRaw()` |
| `controller.go:824` | `/mcp__*` 命令 |
| `controller.go:890` | Custom command |
| `controller.go:896` | Skill invocation (`/skill-name`) |
| `controller.go:1069` | Ref turn（文件引用等） |
