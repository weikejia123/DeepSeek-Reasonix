# Hope 指令 — 功能设计草案

## 概述

设计一个名为 `hope` 的指令，复刻 goal 的自主多轮执行机制，但具备：

1. **独立代码副本** — 凡是 goal 链路中有"goal 印记"的非复用代码，都复刻一份
2. **多环节系统提示词可配置** — 不再硬编码，用户可在配置文件中自由定义
3. **保留 goal 不变** — 不动原有 goal 的代码和行为

## 核心设计理念

### goal ≈ loop

goal 和普通模式的核心差异其实只有两点：

```
runGoalLoopWithRawDisplay          // controller.go:514 — goal/loop/普通 turn 的统一入口
  ├─ runTurnWithRawDisplay         // 所有 turn 都走这里
  └─ continueGoal                  // ★ 唯一差异：goal 激活时循环
       └─ advanceGoalAfterTurn     // ★ 解析 [goal:xxx] 标记，判断是否继续
```

以及 `Compose()` (input.go:93-103) 中的 `<active-goal>` 块注入——**这是 goal 和普通模式的全部差异**。除此之外链路完全复用。

`/loop` 指令（定时重复同一 prompt）和 goal（模型驱动的自主推进）本质差异在于**谁来决策"下一步做什么"**，而不是底层的回合循环机制。

## Goal 完整触达面清单

以下源文件/位置有 goal 印记，hope 需要对应处理：

| 层次 | 文件:行 | 内容 | 处理策略 |
|------|---------|------|----------|
| **常量** | `input.go:19-20` | `activeGoalOpen/Close = "<active-goal>"` | 新增 `activeHopeOpen/Close` |
| **常量** | `input.go:24-27` | `GoalStatusRunning/Complete/Blocked/Stopped` | 可复用或新增独立 status |
| **常量** | `controller.go:206-207` | `maxGoalAutoTurns` + `goalContinueTurn` | 新增 `maxHopeAutoTurns` + `hopeContinueTurn`（可配置化） |
| **注入** | `input.go:93-103` | `Compose()` 中的 goal 块注入 | 新增 hope 块注入分支 |
| **注入** | `input.go:134-146` | `activeGoalBlock()` 生成 goal 提示词 | 新增 `activeHopeBlock()` 读取配置 |
| **状态字段** | `controller.go:129-133` | `goal`/`goalStatus`/`goalTurns`/`goalBlocks`/`goalBlock` | 新增对称的 `hope` 字段组 |
| **命令解析** | `input.go:178-205` | `ParseGoalCommand` + `GoalCommand` | 新增 `ParseHopeCommand` + 注册 `/hope` |
| **命令处理** | `controller.go:896-923` | `applyGoalCommand()` | 新增 `applyHopeCommand()` |
| **循环** | `controller.go:590-656` | `continueGoal()` + `advanceGoalAfterTurn()` | 新增 `continueHope()` + `advanceHopeAfterTurn()` |
| **循环** | `controller.go:609-648` | `parseGoalStatusMarker()` | 新增 `parseHopeStatusMarker()` |
| **停止** | `controller.go:708-713` | `stopGoal()` | 新增 stop 方法或参数化 |
| **公共 API** | `controller.go:1293-1334` | `SetGoal`/`ClearGoal`/`Goal`/`GoalStatus` | 新增对称方法 |
| **Submit 路由** | `controller.go:746-757` | `submit()` 中 `applyGoalCommand` | 新增 `/hope` 检测 |
| **计划互斥** | `auto_plan.go:44-47` | goal 激活时跳过 auto-plan | hope 激活时同样跳过 |
| **Desktop 后端** | `app.go` | Tab 持久化、API、Help 列表 | 新增对称处理 |
| **CLI** | `chat_tui.go` | CLI 界面的 goal 显示和命令路由 | 新增对称处理 |
| **CLI** | `complete.go:89` | `/goal` 自动补全 | 新增 `/hope` 补全项 |
| **i18n** | `i18n.go` + 各语言文件 | Goal 的国际化字符串 | 新增 Hope 的 i18n 条目 |
| **前端** | `App.tsx`/`Composer.tsx`/`StatusBar.tsx`/`TabBar.tsx` | 前端 Goal 模式 UI | 可选独立指示或复用 goal 的 UI |

## 可配置提示词设计

### 当前 goal 的两个硬编码

| 提示词 | 位置 | 内容 |
|--------|------|------|
| `activeGoalBlock()` 正文 | `input.go:142` | "Goal mode: pursue this goal autonomously... Do not stop after describing a plan; execute the next useful step... End with [goal:continue\|complete\|blocked]." |
| `goalContinueTurn` | `controller.go:207` | "Continue pursuing the active goal... Otherwise do the next useful work and end with [goal:continue]." |

### 配置方案

在 `~/.reasonix/reasonix.toml`（或项目级配置）中：

```toml
[agent.hope]
max_turns = 50
system_block = """
Hope mode: pursue this task autonomously. Keep working across turns until the hope is fulfilled.
You have read and write tools available — use them to make progress every turn.
Prefer sensible defaults over asking the user.
End every hope-mode assistant reply with exactly one status marker: [hope:continue], [hope:complete], or [hope:blocked:<reason>].
"""
continue_prompt = """
Continue pursuing the active hope. If complete, provide the concise final result and end with [hope:complete].
If truly blocked after trying sensible defaults, end with [hope:blocked:<reason>].
Otherwise do the next useful work and end with [hope:continue].
"""
```

如果不配置，fallback 到代码内置默认值（与 goal 当前行为一致）。

### 关键差异

hope 相比 goal 的核心改进：

| 维度 | goal | hope |
|------|------|------|
| 提示词控制 | 硬编码在源码中 | 可配置（TOML） |
| 标记名称 | `[goal:xxx]` | `[hope:xxx]` |
| 状态管理 | 与 controller 耦合 | 独立状态字段 |
| 最大轮次 | 常量 `maxGoalAutoTurns = 50` | 可配置 `max_turns` |
| 代码复用 | — | 完全独立副本，不依赖 goal 逻辑 |

## 实现策略

### 原则

1. **复制非复用** — 凡是 goal 功能中带有 `goal` 字样的独立代码段，都复制一份改为 `hope`。注意辨认哪些是真正的"goal 印记"，哪些只是函数名恰好包含 `goal`
2. **不改原有 goal** — 不动任何 `internal/control/controller.go` 中与 goal 相关的逻辑
3. **配置驱动** — hope 的提示词优先从 TOML 配置读取，兜底用内置默认值

### 代码副本策略

| 策略 | 适用场景 |
|------|----------|
| **复制并重命名**（推荐） | 独立函数（`activeGoalBlock` → `activeHopeBlock`）、独立方法（`SetGoal` → `SetHope`） |
| **参数化复用** | 完全相同的逻辑但参数不同（如 `maxGoalAutoTurns` 和 `maxHopeAutoTurns`） |
| **新增分支** | `Compose()` 中的注入逻辑、`submit()` 中的命令路由 |

### 核心模块拆分建议

```
internal/control/
  ├── controller.go    # goal 不动；新增 hope 字段组 + 方法组
  ├── input.go         # goal 不动；新增 hope 常量 + activeHopeBlock + ParseHopeCommand
  ├── auto_plan.go     # goal 和 hope 互斥逻辑（对称处理）
  └── hope.go          # [建议新增] 集中存放 hope 的循环/状态/标记解析逻辑
```

将 hope 的 `continueHope()`、`advanceHopeAfterTurn()`、`parseHopeStatusMarker()`、`stopHope()` 等独立方法抽到 `hope.go`，避免 controller.go 膨胀。

## 未决问题

1. **hope 是否需要一个独立的前端 UI 指示**，还是复用现有的 goal mode indicator？
2. **系统提示词模板语法**：是否需要支持变量插值（如 `{hope_text}`、`{max_turns}`）？当前 TOML 多行字符串 + 硬编码组合足够
3. **hope 与 goal 是否互斥**？一个会话能否同时运行 hope 和 goal？建议互斥（与 plan 互斥的设计一致）
4. **最大轮次是否允许运行时通过 `/hope --max-turns N` 覆盖**？建议允许，但不超过全局配置的硬上限
