# Desktop Loop 改造评估

<!--
版本: v1.0
创建: 2026-06-14 04:47:48
更新: 2026-06-14 04:47:48
-->

**关联文档**: [00-Hope指令设计草案](../hope/00-设计草案.md), [01-Hope可行性分析](../hope/01-可行性分析.md)

---

## 目录

1. [背景：两套循环机制](#1-背景两套循环机制)
2. [代码触达面全景](#2-代码触达面全景)
3. [aloop 设计方案](#3-aloop-设计方案)
4. [Goal 系统的去留策略](#4-goal-系统的去留策略)
5. [实施路径](#5-实施路径)
6. [风险与注意事项](#6-风险与注意事项)
7. [结论与建议](#7-结论与建议)

---

## 1. 背景：两套循环机制

当前项目存在两套**完全独立的循环机制**：

| 维度 | Goal 系统 | TUI /loop |
|------|-----------|-----------|
| **位置** | Controller 层 (`internal/control/`) | TUI 层 (`internal/cli/loop.go`) |
| **状态字段** | `goal`, `goalStatus`, `goalTurns`, `goalBlocks`, `goalBlock` | `loopPrompt`, `loopInterval`, `loopIter` |
| **触发方式** | 模型 `[goal:continue]` 标记 | 定时器 `tea.Tick` |
| **决策者** | 模型自主决定 | 用户预设固定 prompt |
| **可用范围** | Desktop + CLI TUI + HTTP/SSE | **仅 CLI TUI** |
| **停止条件** | complete/blocked/50轮上限 | 用户输入 / Esc |
| **代码量** | ~450 行（引擎 + 注入 + 命令解析 + i18n + Desktop + 前端） | 88 行（纯 TUI） |

### 核心问题

```
TUI 有 /loop  → Desktop 没有
Desktop 有 Goal → TUI 也有但界面不同（/goal 命令 vs mode chip）
```

如果用户想在 Desktop 上实现「每 30 秒检查一次服务状态」，**目前做不到**——只有 TUI 的 `/loop 30 ...` 支持。

---

## 2. 代码触达面全景

基于对 24 个源文件的逐行扫描：

### 2.1 Controller 层（Goal 引擎核心）

| 文件 | 引用密度 | 关键行数 |
|------|:------:|:--------:|
| `internal/control/controller.go` | 🔴 极高 | 134-138（5 状态字段）、211-212（常量）、526-537（入口）、612-629（循环）、631-678（状态机）、680-701（标记解析）、703-728（工具函数）、730-736（stop）、778（路由）、918-945（命令处理）、1326-1367（Set/Get） |
| `internal/control/input.go` | 🔴 极高 | 17-18（XML 标签）、22-25（状态常量）、85-92（Compose 注入）、137-148（activeGoalBlock）、181-206（命令解析） |
| `internal/control/auto_plan.go` | 🟡 中 | 46-49（goal 跳过 auto-plan） |
| `internal/control/goal_test.go` | 🔴 极高 | 全文（4 个测试用例） |
| `internal/control/input_test.go` | 🟡 中 | 136-179（2 个测试用例） |

### 2.2 CLI / TUI 层

| 文件 | 引用密度 | 说明 |
|------|:------:|------|
| `internal/cli/loop.go` | 🔴 极高 | 88 行，纯 TUI 实现 |
| `internal/cli/chat_tui.go` | 🟡 中 | modeTagText (3138-3176)、命令分发 (3619-3632)、loop tick (1368-1372) |
| `internal/cli/complete.go` | 🟢 低 | /loop 补全项 (~90) |

### 2.3 Desktop 层

| 文件 | 引用密度 | 关键位置 |
|------|:------:|:---------|
| `desktop/app.go` | 🔴 高 | 10 处（tab.goal、SetGoal/ClearGoal、Meta 字段、控制器恢复） |
| `desktop/tabs.go` | 🟡 中 | 9 处（tab.goal 字段、持久化、currentTabGoal） |
| `desktop/tab_profile_test.go` | 🟢 低 | 4 处测试 |

### 2.4 前端层

| 文件 | 引用密度 | 关键位置 |
|------|:------:|:---------|
| `types.ts` | 🟡 中 | CollaborationMode (296)、TabMeta/ControllerMeta goal 字段 |
| `bridge.ts` | 🟡 中 | SetGoal/ClearGoal API + mock |
| `useController.ts` | 🟢 低 | goal 状态管理 |
| `composerProfile.ts` | 🟡 中 | goalDraftMode、ComposerProfileField |
| `App.tsx` | 🟡 中 | startGoal、handleSend goal 分支、collaborationMode goal |
| `Composer.tsx` | 🔴 高 | 15+ 处（goalModeOn、chip 渲染、验证、placeholder） |
| `StatusBar.tsx` | 🟢 低 | goalMode 检测 + 标签 |
| `TabBar.tsx` | 🟢 低 | goalMode badge |
| `styles.css` | 🟢 低 | 3 处 CSS 类名 |
| locale 文件 (3) | 🟢 低 | 各 3 条 goal 文案 |

### 2.5 其他

| 文件 | 引用密度 | 说明 |
|------|:------:|------|
| `internal/i18n/*` | 🟡 中 | GoalSetFmt/GoalCleared/GoalEmpty/GoalCurrentFmt/CmdGoal (5 条 × 3 语言) |
| `internal/serve/serve.go` | 🟢 低 | 会话元数据 (1 行) |
| `internal/agent/*` | 🟢 无 | **零引用** — agent 层与 Goal 系统完全无关 |

### 2.6 扫描结论

- **24 个源文件** 直接或间接包含 goal 引用
- **~450 行** Goal 相关代码（不含注释/空行）
- **internal/agent/** 不涉及，改造时无需动 agent 层
- 改造影响面：控制层 ~60% / Desktop 后端 ~20% / 前端 ~20%

---

## 3. aloop 设计方案

### 3.1 核心思路

在 Controller 层新增「定时循环」能力，与现有 Goal 智能循环**平行存在**，互不干扰。

| 机制 | 决策者 | 驱动方式 | 适用场景 |
|------|--------|----------|----------|
| **Goal** | 模型自主 (continue/complete/blocked) | 标记驱动 | 多步编码任务 |
| **aloop** | 用户预设 | 定时驱动 | 重复监控/检查 |

### 3.2 Controller 层新增字段与方法

```go
// Controller struct 新增字段
type Controller struct {
    // ... 现有字段 ...
    aloopPrompt  string        // 循环 prompt，空 = 未激活
    aloopInterval time.Duration // 触发间隔
    aloopIter    int           // 已执行次数
    aloopMax     int           // 最大次数（0 = 无限制）
    aloopCancel  context.CancelFunc  // 取消函数
}

// 新增方法
func (c *Controller) StartLoop(prompt string, interval time.Duration, maxIter int) error
func (c *Controller) StopLoop() string    // 返回状态文本
func (c *Controller) LoopStatus() string  // 返回当前状态
```

### 3.3 循环引擎

```
StartLoop(prompt, interval, maxIter)
  ├─ 停止已有 aloop（如果运行中）
  ├─ 设置字段
  ├─ 创建带取消的 context: ctx, c.aloopCancel = context.WithCancel(c.ctx)
  └─ go aloopRunner(ctx, prompt, interval, maxIter)

aloopRunner(ctx, prompt, interval, maxIter)
  ├─ 首次立即执行: runTurnWithRawDisplay(prompt, "/aloop: "+prompt, prompt, "")
  ├─ ticker = time.NewTicker(interval)
  ├─ for {
  │     select {
  │     case <-ctx.Done():   return
  │     case <-ticker.C:
  │         if maxIter > 0 && iter >= maxIter { c.StopLoop(); return }
  │         iter++
  │         runTurnWithRawDisplay(prompt, "/aloop: "+prompt, prompt, "")
  │     }
  │   }
```

### 3.4 与 Compose 的关系

aloop 发送的 prompt 仍然走 `Compose()`（plan mode marker、reasoning language、memory update、background jobs 正常注入），但：

- **不注入** `<active-goal>` 块（aloop 非 goal）
- 不跳过 auto-plan 审批（aloop 不需要模型自主决策）
- 不抑制 plan mode

### 3.5 Slash 命令

```go
// /aloop 语法
/aloop <seconds> <prompt>    // 启动（最小 5 秒）
/aloop stop                  // 停止
/aloop                       // 查看状态
```

### 3.6 前端暴露

| 层 | 实现方式 |
|----|----------|
| **CLI TUI** | 现有 `/loop` 命令改为调用 `Controller.StartLoop()`（迁移） |
| **Desktop** | `bridge.ts` + `app.go` 新增 `StartLoop(secs, prompt) / StopLoop() / LoopStatus()` |
| **HTTP/SSE** | `serve.go` 新增 loop 控制端点 |

---

## 4. Goal 系统的去留策略

### 4.1 选项分析

| 选项 | 说明 | 工作量 | 风险 |
|------|------|:------:|:----:|
| **A. 保留 Goal，新增 aloop** | 两套机制共存 | 小（~200 行） | 低 |
| **B. 删除 Goal，aloop 完全替代** | 彻底重构 | 大（~600 行删除 + ~250 行新增） | 高 |
| **C. 保留 Goal，aloop 作为实验** | 先加 aloop，Goal 标记为 deprecated | 最小（~150 行） | 最低 |

### 4.2 推荐：选项 C → 逐步过渡到选项 B

**第一阶段**：保留 Goal 完整不动，在 Controller 层新增 aloop 机制，Desktop 上通过 `/aloop` 命令暴露。Goal 标记为 deprecated。

**第二阶段**：验证 aloop 稳定后，评估是否可以将部分 Goal 场景迁移到 aloop，逐步缩小 Goal 的使用范围。

**第三阶段**（可选）：如果 Goal 的使用率降至零，整体删除。

### 4.3 可保留 Goal 的场景

Goal 的**模型自主决策**（`[goal:continue] / [goal:complete] / [goal:blocked]`）在以下场景仍有独特价值：

- 「修复所有 lint 错误」— 模型自行判断何时完成
- 「为 API 添加完整 CRUD」— 多步编码无需人工分段
- 「重构 auth 包」— 模型自主决定重构范围

aloop 无法替代这些场景——aloop 只适合**固定 prompt 重复执行**。

### 4.4 结论：Goal 和 aloop 是互补的

```
Goal  — 模型自主推进（智能循环）
aloop — 用户预设定时（机械循环）

两者互补，不是替代关系。
```

---

## 5. 实施路径

### 阶段 1：Controller 层新增 aloop（3 文件，~150 行）

1. `internal/control/controller.go` — 新增 aloop 字段 + `StartLoop`/`StopLoop`/`LoopStatus` + `aloopRunner`
2. `internal/control/input.go` — 新增 `ParseAloopCommand()` / `/aloop` 命令解析
3. `internal/control/controller.go` — `submit()` 中前插 `c.applyAloopCommand()`

**验证**: `go test ./internal/control/...`

### 阶段 2：CLI 迁移（2 文件，~40 行）

4. `internal/cli/loop.go` — 将 `runLoopCommand` 改为调用 `c.StartLoop()`（而非 `m.startTurnWithRaw`）
5. `internal/cli/chat_tui.go` — loop tick 改为检查 Controller 的 aloop 状态

**验证**: TUI 中 `/loop 10 check status` 正常工作

### 阶段 3：Desktop 后端（2 文件，~60 行）

6. `desktop/app.go` — 新增 `StartLoop`/`StopLoop`/`LoopStatus` bridge API
7. `internal/serve/serve.go` — 会话元数据新增 loop 状态

**验证**: Desktop 编译通过

### 阶段 4：Desktop 前端（4 文件，~80 行）

8. `types.ts` — 新增 `LoopStatus` 类型 + bridge 声明
9. `bridge.ts` — 新增 `StartLoop`/`StopLoop`/`LoopStatus` API + mock
10. `App.tsx` — 新增 `/aloop` 命令路由
11. locales — 新增 loop 相关文案（可复用现有 `/loop` i18n 的 `CmdLoop`）

**验证**: Desktop 中输入 `/aloop 10 check status` 触发定时循环

---

## 6. 风险与注意事项

### 6.1 缓存稳定性

Goal 的 `<active-goal>` 块在 non-system-prefix 位置（turn tail）注入，删除不影响 prefix cache。aloop **不注入** <active-goal> 块，所以无影响。

### 6.2 TUI /loop 迁移兼容

现有 `/loop` 命令在 TUI 中使用 `tea.Tick` 定时器（TUI 事件循环的一部分）。迁移到 Controller 后，定时器移到 Controller 的 goroutine 中。

**兼容策略**：保持 `/loop` 命令名不变，内部改为调用 Controller。用户无感知。

### 6.3 Desktop 缺少 mode chip

Goal 在 Desktop Composer 中有独立的 mode chip（选择 `"goal"` 后输入目标）。aloop 也可以有类似的 UI，但第一阶段的 `/aloop` 命令式已经可用。

**建议**：第一阶段只做 `/aloop` 命令，不做 mode chip。视使用反馈决定是否增加 UI。

### 6.4 aloop 与并行 turn 的冲突

如果用户在 aloop 运行时手动发送消息，会发生什么？

- TUI 现有行为：用户输入 → 自动停止 `/loop`
- aloop 建议行为：用户输入 → aloop 暂停 → 用户 turn 完成后继续

**第一阶段**：简单处理——aloop 运行时用户输入 → 自动 `StopLoop()`（与 TUI 现有行为一致）。

### 6.5 Goal 的 `/goal set` / `/goal clear` 过渡

如果未来决定 deprecate Goal，`/goal` 命令可以保留作为 `/aloop` 的别名，但协作模式改为非阻塞的单轮执行。

---

## 7. 结论与建议

### 7.1 评估结论

| 维度 | 结论 |
|------|------|
| 技术可行性 | ✅ **完全可行** |
| 目标 | 新增 aloop，与 Goal 共存互补 |
| 工作量 | 阶段 1-4 共 **~330 行**（远少于 Hope 的 575 行） |
| 影响面 | 7 个文件（远少于 Goal 的 24 个文件） |
| 风险 | 🟢 低（aloop 是新增，不修改 Goal 现有代码） |

### 7.2 与 Hope 方案的关系

```
先前方案（Hope）: 完整复制 Goal 系统 → 575 行，24 文件，风险中等
   ↓
当前方案（aloop）: 新增定时循环 → 330 行，7 文件，风险低
   ↓
核心差异: aloop 放弃了 Goal 的「模型自主决策」特征，
          聚焦于「定时重复执行」这个真正缺失的能力。
```

如果用户需要的是**定时检查/监控**（如「每 30 秒检查一次服务状态」），aloop 是正确方案。如果用户需要的是**完整的模型自主推进**，保留 Goal 不动。

### 7.3 下一步

1. ✅ 确认 aloop 方案（这个文档）
2. 实现阶段 1：Controller 层（~150 行）
3. 实现阶段 2：CLI 迁移（~40 行）
4. 实现阶段 3-4：Desktop（~140 行）
5. 测试验证
