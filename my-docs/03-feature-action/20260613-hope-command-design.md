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

---

## 可行性评估（2026-06-13 补充）

> 基于现有代码的完整走查，对上述设计草案的可行性、风险和替代方案评估。

### 结论速览

| 维度 | 评估 |
|------|------|
| 技术可行性 | ✅ 可实现 |
| 架构合理性 | ⚠️ 复制策略存在根本性问题 |
| 推荐方案 | 先参数化 goal，hope 作为薄配置层 |
| 实际工作量 | 设计草案估算的 2-3 倍 |

### 设计草案触达面审计

文档列的 20 个触达点基本准确，但遗漏了 4 个关键耦合点：

#### 遗漏 1：前端 `collaborationMode` 是闭合枚举（严重）

```typescript
// types.ts:296
export type CollaborationMode = "normal" | "plan" | "goal";
```

Hope 需要 `"hope"` 作为第四个值。涟漪效应：

| 文件 | 影响 |
|------|------|
| `types.ts` | 类型扩展 |
| `App.tsx` | ~10 处 `=== "goal"` 判断需加 hope 分支；`handleSend` 中 goal 模式自动 `/goal` 逻辑需 hope 对称处理 |
| `Composer.tsx` | `goalModeOn` 拆分为 `goalModeOn`/`hopeModeOn`；新 hope chip UI；placeholder/disabled 逻辑 |
| `StatusBar.tsx` | 新 hope 模式指示器 |
| `TabBar.tsx` | 新 hope 模式标记 |

#### 遗漏 2：Tab 持久化需要新字段（中等）

```go
// desktop/app.go:1841-1842
Goal       string `json:"goal,omitempty"`
GoalStatus string `json:"goalStatus,omitempty"`
```

Hope 需要 `Hope` + `HopeStatus` 字段，波及 tab 恢复（`app.go:402`）、mode 切换清理（`app.go:770-784`）、`newCtrl` 初始化（`app.go:3411/3507/3574`）。

#### 遗漏 3：`Compose()` 注入顺序

`input.go:93-103` 中 goal 和 plan 可以同时激活（goal turns 可能触发 plan approval）。Hope 与 plan 同时激活时的注入顺序需明确定义。

#### 遗漏 4：`auto_plan.go` 互斥检查

```go
// auto_plan.go:44-48
goalActive := c.goalActive()
if plan || goalActive { return false }
```

Hope 激活时也应跳过 auto-plan，需加 `hopeActive` 检查。

### 复制策略的核心问题

审视 goal 和 hope 的实际差异：

| 维度 | Goal | Hope (设计) | 实际差异 |
|------|------|------------|----------|
| 循环机制 | `continueGoal()` | `continueHope()` | **完全一致** |
| 状态标记 | `[goal:xxx]` | `[hope:xxx]` | 正则不同 |
| 提示词 | 硬编码 | 可配置 TOML | **唯一实质差异** |
| 最大轮次 | `maxGoalAutoTurns = 50` | 可配置 | 参数化即可 |
| 3-strike 阻塞检测 | `sameGoalBlock()` | 同样逻辑 | **完全一致** |
| stop 方法 | `stopGoal()` | `stopHope()` | **完全一致** |

`continueGoal` 的 **60 行核心循环逻辑在 hope 版本中是逐字相同的**，唯一的变量是标记正则、提示词文本、最大轮次常量。复制意味着未来任何一个 bug fix 需要同步两个文件。

### 工作量重估

| 层次 | 设计草案估算 | 实际预估 | 膨胀原因 |
|------|-------------|---------|----------|
| `controller.go` 字段+方法 | ~200 行 | ~250 行 | |
| `input.go` 常量+函数 | ~60 行 | ~80 行 | |
| `hope.go` 新文件 | 未提及 | ~80 行 | 循环/状态/标记解析 |
| `auto_plan.go` | 1 行 | 2 行 | |
| `desktop/app.go` | ~30 行 | ~60 行 | 持久化+API+help |
| 前端 | ~50 行 | **~150 行** | `CollaborationMode` 涟漪效应 |
| i18n + 配置 | ~20 行 | ~25 行 | |
| **合计** | ~350 行 | **~650 行** | |

### 风险矩阵

| 风险 | 严重度 | 描述 |
|------|--------|------|
| **双倍维护负担** | 🔴 高 | 60 行完全相同循环逻辑需在 goal/hope 两处同步修复 |
| **前端枚举爆炸** | 🔴 高 | `CollaborationMode` 从 3→4 值，每个 `=== "goal"` 需维护 hope 对等分支 |
| **goal bug 遗漏同步** | 🟡 中 | 修 goal 时可能忘记同修 hope |
| **hope.go 循环依赖** | 🟡 中 | `continueHope()` 需调用 controller 私有方法 |
| **/hope 与 /goal 混淆** | 🟡 中 | 用户可能不清楚场景区别 |
| **配置 schema 膨胀** | 🟢 低 | 向后兼容容易 |

### 推荐替代方案：参数化而非复制

**Phase 1（~3 天，不动前端）：参数化 goal**

```go
type GoalConfig struct {
    ContinuePrompt string // 替代 goalContinueTurn
    SystemBlock     string // 替代 activeGoalBlock 的提示词
    MaxTurns        int    // 替代 maxGoalAutoTurns
    MarkerPrefix    string // "goal" 或 "hope"
}

func (c *Controller) continueGoal(ctx context.Context, cfg GoalConfig) error {
    // 参数化后的循环逻辑，复用现有代码
}

func parseStatusMarker(reply, prefix string) (status, reason string, ok bool) {
    // 参数化正则生成
}
```

**Phase 2（~2 天）：hope 作为薄配置层**

```go
// hope.go — 仅 ~30 行
func (c *Controller) StartHope(hopeText string) {
    c.hopeConfig = loadHopeConfig() // 从 TOML 读取
}

func (c *Controller) continueHope(ctx context.Context) error {
    return c.continueGoal(ctx, c.hopeConfig)
}
```

输出：可扩展框架——第三个自主模式只需一个 config 对象。

### 与当前 goal 代码的对照索引

| 现有代码 | 行号 | 参数化后 |
|----------|------|----------|
| `activeGoalBlock()` | input.go:134-146 | 接受 `GoalConfig.SystemBlock` 参数 |
| `goalContinueTurn` | controller.go:207 | 移入 `GoalConfig.ContinuePrompt` |
| `continueGoal()` | controller.go:590-607 | 接受 `GoalConfig` 参数 |
| `advanceGoalAfterTurn()` | controller.go:609-656 | 接受 `GoalConfig` — 生成正确正则 |
| `parseGoalStatusMarker()` | controller.go:658-703 | 改为 `parseStatusMarker(reply, prefix)` |
| `Compose()` | input.go:93-103 | goal 和 hope 共用注入逻辑，提示词不同 |
| `auto_plan.go:44-48` | auto_plan.go | 增加 `hopeActive` 检查 |
